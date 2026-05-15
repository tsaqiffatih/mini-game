package game

import (
	"context"
	"errors"
	"time"

	"github.com/tsaqiffatih/mini-game/chess"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

var (
	ErrPlayerNotFound   = errors.New("player not found")
	ErrInvalidGameState = errors.New("invalid game state")
)

type TicTacToeMoveResult struct {
	State     TicTacToeStateSnapshot
	GameEnded bool
}

type ChessMoveResult struct {
	FEN    string
	PGN    []string
	Result *chess.GameResult
}

func (r *Room) HandleTicTacToeMove(
	playerID string,
	row int,
	col int,
) (*TicTacToeMoveResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.handleTicTacToeMoveLocked(playerID, row, col)
	if err != nil {
		return nil, err
	}

	r.scheduleAIMoveAfterHumanMoveLocked(playerID)
	return result, nil
}

func (r *Room) handleTicTacToeMoveLocked(
	playerID string,
	row int,
	col int,
) (*TicTacToeMoveResult, error) {
	player, ok := r.players[playerID]
	if !ok {
		return nil, ErrPlayerNotFound
	}
	if r.ticTacToe == nil {
		return nil, ErrInvalidGameState
	}
	if r.roomState != RoomStatePlaying {
		return nil, errors.New("game is not active")
	}

	if err := r.ticTacToe.ApplyMove(player.Mark, row, col); err != nil {
		return nil, err
	}

	result := &TicTacToeMoveResult{
		State: TicTacToeStateSnapshot{
			Board:  r.ticTacToe.Board,
			Turn:   r.ticTacToe.Turn,
			Winner: r.ticTacToe.Winner,
			Status: r.ticTacToe.Status,
		},
		GameEnded: r.ticTacToe.Status == tictactoe.StatusEnded,
	}

	if result.GameEnded {
		if err := r.transitionLocked(RoomStateFinished); err != nil {
			return nil, err
		}
		r.bumpStateVersionLocked()
		r.scheduleResetLocked()
		return result, nil
	}

	r.bumpStateVersionLocked()
	return result, nil
}

func (r *Room) handleTicTacToeAIMoveLocked() error {
	if !r.isAIEnabled || r.gameType != "tictactoe" || r.ticTacToe == nil {
		return nil
	}
	if r.roomState != RoomStatePlaying || r.ticTacToe.Status != tictactoe.StatusActive {
		return nil
	}

	aiPlayer := r.currentAIPlayerLocked()
	if aiPlayer == nil || r.ticTacToe.Turn != aiPlayer.Mark {
		return nil
	}

	move := tictactoe.ComputeMove(r.ticTacToe, aiPlayer.Mark, r.aiLevel)
	if move.Row < 0 || move.Col < 0 {
		return nil
	}

	_, err := r.handleTicTacToeMoveLocked(aiPlayer.ID, move.Row, move.Col)
	return err
}

func (r *Room) resetTicTacToe() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.ticTacToe == nil {
		return ErrInvalidGameState
	}

	if r.roomState != RoomStateFinished {
		return errors.New("room is not finished")
	}
	r.scheduleResetLocked()
	return nil
}

func (r *Room) HandleChessMove(
	playerID string,
	from string,
	to string,
	promotion string,
) (*ChessMoveResult, error) {
	return r.HandleChessMoveWithContext(context.Background(), playerID, from, to, promotion)
}

func (r *Room) HandleChessMoveWithContext(
	_ context.Context,
	playerID string,
	from string,
	to string,
	promotion string,
) (*ChessMoveResult, error) {
	r.mu.Lock()
	result, aiMove, err := r.handleChessMoveLocked(playerID, from, to, promotion)
	r.mu.Unlock()

	if err != nil {
		return nil, err
	}
	r.scheduleChessAIMove(aiMove)

	return result, nil
}

func (r *Room) HandleChessUndo(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.chess == nil || r.gameType != "chess" {
		return ErrInvalidGameState
	}
	player, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}
	if player.IsAI {
		return errors.New("AI cannot request undo")
	}
	if !r.isAIEnabled {
		return errors.New("undo is only supported for AI rooms")
	}

	r.cancelScheduledAIMoveLocked()
	if err := r.chess.RollbackLastAITurn(); err != nil {
		return err
	}
	r.roomState = RoomStatePlaying
	r.bumpStateVersionLocked()
	return nil
}

type chessAIMoveRequest struct {
	shouldMove bool
	playerID   string
	fen        string
	engine     *StockfishEngine
}

func (r *Room) handleChessMoveLocked(
	playerID string,
	from string,
	to string,
	promotion string,
) (*ChessMoveResult, chessAIMoveRequest, error) {
	player, ok := r.players[playerID]
	if !ok {
		return nil, chessAIMoveRequest{}, ErrPlayerNotFound
	}
	if r.chess == nil {
		return nil, chessAIMoveRequest{}, ErrInvalidGameState
	}
	if r.roomState != RoomStatePlaying {
		return nil, chessAIMoveRequest{}, errors.New("game is not active")
	}

	result, err := r.chess.UpdateState(
		player.ID,
		player.Mark,
		player.IsAI,
		from,
		to,
		promotion,
	)
	if err != nil {
		return nil, chessAIMoveRequest{}, err
	}

	pgn := append([]string(nil), r.chess.PGNMoves()...)
	moveResult := &ChessMoveResult{
		FEN:    r.chess.FEN(),
		PGN:    pgn,
		Result: result.GameResult,
	}

	if result.GameResult.Status != "ongoing" && result.GameResult.Status != "check" {
		if err := r.transitionLocked(RoomStateFinished); err != nil {
			return nil, chessAIMoveRequest{}, err
		}
		r.aiThinking = false
		r.bumpStateVersionLocked()
		r.scheduleResetLocked()
		return moveResult, chessAIMoveRequest{}, nil
	}

	r.bumpStateVersionLocked()
	return moveResult, r.chessAIMoveRequestLocked(player), nil
}

func (r *Room) HandlePlayerDisconnected(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.removePlayerLocked(playerID)
}

func (r *Room) AddChatMessage(playerID string, message string) (ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.players[playerID]
	if !exists {
		return ChatMessage{}, ErrPlayerNotFound
	}

	message, err := normalizeChatMessage(message)
	if err != nil {
		return ChatMessage{}, err
	}

	now := time.Now().UTC()
	chatMessage := ChatMessage{
		ID:         newChatMessageID(now),
		RoomID:     r.RoomID,
		PlayerID:   player.ID,
		PlayerMark: player.Mark,
		Message:    message,
		CreatedAt:  now,
	}

	r.chatMessages = append(r.chatMessages, chatMessage)
	if len(r.chatMessages) > maxChatMessages {
		r.chatMessages = append([]ChatMessage(nil), r.chatMessages[len(r.chatMessages)-maxChatMessages:]...)
	}

	return chatMessage, nil
}

func (r *Room) ChatHistory() []ChatMessage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.chatHistoryLocked()
}

func (r *Room) MarkPlayerConnected(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	player.Session = PlayerSessionConnected
	player.LastActive = time.Now()
	return nil
}

func (r *Room) MarkPlayerDisconnected(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	player.Session = PlayerSessionDisconnected
	player.LastActive = time.Now()
	return nil
}

func (r *Room) RemoveInactivePlayers(now time.Time, duration time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for playerID, player := range r.players {
		if now.Sub(player.LastActive) > duration {
			r.removePlayerLocked(playerID)
			observability.Logger().Info("inactive player removed from room",
				"room_id", r.RoomID,
				"player_id", playerID,
				"event_type", "player_removed",
			)
		}
	}
}

func (r *Room) TouchPlayer(playerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	player, exists := r.players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	player.Session = PlayerSessionConnected
	player.LastActive = time.Now()
	return nil
}

func (r *Room) removePlayerLocked(playerID string) bool {
	player, exists := r.players[playerID]
	if !exists {
		return false
	}

	player.Session = PlayerSessionRemoved
	delete(r.players, playerID)
	r.bumpStateVersionLocked()

	if len(r.players) == 0 {
		r.cancelScheduledAIMoveLocked()
		r.cancelScheduledResetLocked()
		return true
	}

	if len(r.players) < 2 {
		r.cancelScheduledAIMoveLocked()
		r.cancelScheduledResetLocked()
		r.roomState = RoomStateWaiting

		switch r.gameType {
		case "tictactoe":
			r.resetTicTacToeWaitingLocked()
			r.resetRemainingPlayerMark("X")
		case "chess":
			r.resetChessLocked()
			r.resetRemainingPlayerMark("white")
		}
	}

	return false
}

func (r *Room) resetRemainingPlayerMark(mark string) {
	for _, player := range r.players {
		player.Mark = mark
		break
	}
}

func (r *Room) resetTicTacToeWaitingLocked() {
	if r.ticTacToe == nil {
		return
	}

	r.ticTacToe.Reset("X")
	r.ticTacToe.Status = tictactoe.StatusWaiting
}

func (r *Room) resetTicTacToeAfterResettingLocked() {
	r.cancelScheduledAIMoveLocked()
	r.ticTacToe.Reset("X")
	if len(r.players) == 2 {
		r.ticTacToe.Status = tictactoe.StatusActive
		_ = r.transitionLocked(RoomStatePlaying)
		return
	}

	r.ticTacToe.Status = tictactoe.StatusWaiting
	r.roomState = RoomStateWaiting
}

func (r *Room) resetChessLocked() {
	r.chess = chess.NewChessGameState()
	r.aiThinking = false
}

func (r *Room) resetChessAfterResettingLocked() {
	r.cancelScheduledAIMoveLocked()
	r.resetChessLocked()
	if len(r.players) == 2 {
		_ = r.transitionLocked(RoomStatePlaying)
		return
	}

	r.roomState = RoomStateWaiting
}

func (r *Room) AddPlayer(playerSnapshot PlayerSnapshot) (*JoinRoomResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.players) >= 2 {
		return nil, errors.New("room is full")
	}
	if r.roomState != RoomStateWaiting {
		return nil, errors.New("room is not accepting players")
	}

	player := &Player{
		ID:         playerSnapshot.ID,
		Mark:       playerSnapshot.Mark,
		IsAI:       playerSnapshot.IsAI,
		LastActive: playerSnapshot.LastActive,
		Session:    PlayerSessionConnected,
	}

	if _, exists := r.players[player.ID]; exists {
		return nil, errors.New("player already in room")
	}

	player.LastActive = time.Now()
	r.players[player.ID] = player

	switch r.gameType {
	case "tictactoe":
		r.handleTicTacToePlayerJoin(player)
	case "chess":
		r.handleChessPlayerJoin(player)
	}

	if len(r.players) == 2 {
		if err := r.transitionLocked(RoomStatePlaying); err != nil {
			return nil, err
		}
		r.activateGameLocked()
	}
	r.bumpStateVersionLocked()

	return &JoinRoomResponse{
		PlayerID:   player.ID,
		PlayerMark: player.Mark,
		Room:       RoomDTO{RoomID: r.RoomID},
	}, nil
}

func (r *Room) handleTicTacToePlayerJoin(player *Player) {
	if player.IsAI {
		player.Mark = "O"
		return
	}
	if r.hasTicTacToeMarkLocked("X") {
		player.Mark = "O"
		return
	}
	player.Mark = "X"
}

func (r *Room) handleChessPlayerJoin(player *Player) {
	if r.hasChessMarkLocked("white") {
		player.Mark = "black"
		return
	}
	player.Mark = "white"
}

func (r *Room) EnableAI() error {
	return r.EnableAILevel(DefaultAILevel)
}

func (r *Room) EnableAILevel(aiLevel int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.gameType != "tictactoe" && r.gameType != "chess" {
		return errors.New("AI only supported for tictactoe and chess")
	}
	if r.isAIEnabled {
		return errors.New("AI already enabled")
	}
	if len(r.players) >= 2 {
		return errors.New("room is full")
	}
	if r.roomState != RoomStateWaiting {
		return errors.New("room is not accepting players")
	}

	r.aiLevel = normalizeAILevel(aiLevel)
	aiPlayer := &Player{
		ID:         "AI",
		IsAI:       true,
		LastActive: time.Now(),
		Session:    PlayerSessionConnected,
	}
	switch r.gameType {
	case "tictactoe":
		aiPlayer.Mark = "O"
	case "chess":
		aiPlayer.Mark = "black"
		engine, err := NewStockfishEngine(stockfishPathFromEnv(), r.aiLevel)
		if err != nil {
			return err
		}
		r.stockfish = engine
	}

	r.players[aiPlayer.ID] = aiPlayer
	r.isAIEnabled = true
	return nil
}

func (r *Room) GetPlayersSnapshot() []PlayerSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]PlayerSnapshot, 0, len(r.players))

	for _, player := range r.players {
		players = append(players, playerSnapshot(player))
	}

	return players
}

func (r *Room) GetPlayer(playerID string) (PlayerSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	player, exists := r.players[playerID]
	if !exists {
		return PlayerSnapshot{}, ErrPlayerNotFound
	}

	return playerSnapshot(player), nil
}

func (r *Room) activateGameLocked() {
	if r.ticTacToe != nil {
		r.ticTacToe.Status = tictactoe.StatusActive
	}
}

func (r *Room) hasTicTacToeMarkLocked(mark string) bool {
	for _, player := range r.players {
		if player.Mark == mark {
			return true
		}
	}
	return false
}

func (r *Room) hasChessMarkLocked(mark string) bool {
	for _, player := range r.players {
		if player.Mark == mark {
			return true
		}
	}
	return false
}

func (r *Room) chatHistoryLocked() []ChatMessage {
	return append([]ChatMessage(nil), r.chatMessages...)
}

func (r *Room) currentAIPlayerLocked() *Player {
	for _, player := range r.players {
		if player.IsAI {
			return player
		}
	}
	return nil
}

func (r *Room) chessAIMoveRequestLocked(lastPlayer *Player) chessAIMoveRequest {
	if !r.isAIEnabled || r.gameType != "chess" || r.chess == nil || r.stockfish == nil {
		return chessAIMoveRequest{}
	}
	if lastPlayer == nil || lastPlayer.IsAI {
		return chessAIMoveRequest{}
	}
	if r.roomState != RoomStatePlaying || !r.chess.IsActive() {
		return chessAIMoveRequest{}
	}

	aiPlayer := r.currentAIPlayerLocked()
	if aiPlayer == nil || r.chess.CurrentTurn() != aiPlayer.Mark {
		return chessAIMoveRequest{}
	}

	return chessAIMoveRequest{
		shouldMove: true,
		playerID:   aiPlayer.ID,
		fen:        r.chess.FEN(),
		engine:     r.stockfish,
	}
}

func (r *Room) scheduleAIMoveAfterHumanMoveLocked(playerID string) {
	player, exists := r.players[playerID]
	if !exists || player.IsAI {
		return
	}
	if r.aiMoveCancel != nil {
		return
	}

	switch r.gameType {
	case "tictactoe":
		if !r.shouldScheduleTicTacToeAIMoveLocked() {
			return
		}
	case "chess":
		return
	default:
		return
	}

	r.scheduleAIMoveLocked()
}

func (r *Room) shouldScheduleTicTacToeAIMoveLocked() bool {
	if !r.isAIEnabled || r.ticTacToe == nil {
		return false
	}
	if r.roomState != RoomStatePlaying || r.ticTacToe.Status != tictactoe.StatusActive {
		return false
	}

	aiPlayer := r.currentAIPlayerLocked()
	return aiPlayer != nil && r.ticTacToe.Turn == aiPlayer.Mark
}

func (r *Room) scheduleAIMoveLocked() {
	ctx, cancel := context.WithCancel(context.Background())
	r.aiMoveCancel = cancel
	r.aiMoveVersion++
	version := r.aiMoveVersion
	delay := r.aiMoveDelay

	go r.runScheduledAIMove(ctx, version, delay)
}

func (r *Room) runScheduledAIMove(ctx context.Context, version uint64, delay time.Duration) {
	if !waitForDelay(ctx, delay) {
		r.clearScheduledAIMove(version)
		return
	}

	var changed bool
	r.mu.Lock()
	if ctx.Err() == nil && version == r.aiMoveVersion && r.shouldScheduleTicTacToeAIMoveLocked() {
		if err := r.handleTicTacToeAIMoveLocked(); err == nil {
			changed = true
		}
	}
	r.clearScheduledAIMoveLocked(version)
	r.mu.Unlock()

	if changed {
		r.notifyStateChanged()
	}
}

func (r *Room) scheduleChessAIMove(request chessAIMoveRequest) {
	if !request.shouldMove || request.engine == nil {
		return
	}

	r.mu.Lock()
	if r.aiMoveCancel != nil {
		r.mu.Unlock()
		return
	}
	aiCtx, cancel := context.WithCancel(context.Background())
	r.aiMoveCancel = cancel
	r.aiMoveVersion++
	r.aiThinking = true
	r.bumpStateVersionLocked()
	version := r.aiMoveVersion
	delay := r.aiMoveDelay
	r.mu.Unlock()

	go r.runScheduledChessAIMove(aiCtx, version, delay, request)
}

func (r *Room) runScheduledChessAIMove(aiCtx context.Context, version uint64, delay time.Duration, request chessAIMoveRequest) {
	if !waitForDelay(aiCtx, delay) {
		r.finishScheduledAIMove(version, false)
		return
	}

	fen, engine, ok := r.currentChessAIRequest(version, request.playerID)
	if !ok {
		r.finishScheduledAIMove(version, false)
		return
	}

	aiFrom, aiTo, aiPromotion, err := engine.BestMove(aiCtx, fen)
	if err != nil {
		r.finishScheduledAIMove(version, true)
		return
	}

	var changed bool
	r.mu.Lock()
	if aiCtx.Err() == nil && version == r.aiMoveVersion && r.shouldApplyChessAIMoveLocked(request.playerID) {
		if _, _, err := r.handleChessMoveLocked(request.playerID, aiFrom, aiTo, aiPromotion); err == nil {
			changed = true
		}
	}
	if version == r.aiMoveVersion {
		r.aiThinking = false
		r.clearScheduledAIMoveLocked(version)
		r.bumpStateVersionLocked()
	}
	r.mu.Unlock()

	if changed {
		r.notifyStateChanged()
	}
}

func (r *Room) currentChessAIRequest(version uint64, aiPlayerID string) (string, *StockfishEngine, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if version != r.aiMoveVersion || !r.shouldApplyChessAIMoveLocked(aiPlayerID) || r.stockfish == nil {
		return "", nil, false
	}
	return r.chess.FEN(), r.stockfish, true
}

func (r *Room) shouldApplyChessAIMoveLocked(aiPlayerID string) bool {
	if !r.isAIEnabled || r.gameType != "chess" || r.chess == nil {
		return false
	}
	if r.roomState != RoomStatePlaying || !r.chess.IsActive() {
		return false
	}

	aiPlayer, exists := r.players[aiPlayerID]
	return exists && aiPlayer.IsAI && r.chess.CurrentTurn() == aiPlayer.Mark
}

func (r *Room) clearScheduledAIMove(version uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clearScheduledAIMoveLocked(version)
}

func (r *Room) clearScheduledAIMoveLocked(version uint64) {
	if version != r.aiMoveVersion {
		return
	}
	r.aiMoveCancel = nil
}

func (r *Room) finishScheduledAIMove(version uint64, notify bool) {
	var changed bool
	r.mu.Lock()
	if version == r.aiMoveVersion {
		changed = r.aiThinking
		r.aiThinking = false
		r.clearScheduledAIMoveLocked(version)
		if changed {
			r.bumpStateVersionLocked()
		}
	}
	r.mu.Unlock()

	if notify && changed {
		r.notifyStateChanged()
	}
}
