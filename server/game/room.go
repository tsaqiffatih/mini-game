package game

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/tsaqiffatih/mini-game/chess"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

type RoomState string

const (
	RoomStateWaiting   RoomState = "WAITING"
	RoomStatePlaying   RoomState = "PLAYING"
	RoomStateFinished  RoomState = "FINISHED"
	RoomStateResetting RoomState = "RESETTING"
)

const (
	DefaultFinishedResetDelay = 5 * time.Second
	DefaultResettingDelay     = 3 * time.Second
	DefaultAILevel            = 10
	DefaultAIMoveDelay        = 1000 * time.Millisecond
)

type Room struct {
	RoomID        string `json:"room_id"`
	players       map[string]*Player
	gameType      string
	roomState     RoomState
	ticTacToe     *tictactoe.TictactoeGameState
	chess         *chess.ChessGameState
	isAIEnabled   bool
	aiLevel       int
	aiMoveDelay   time.Duration
	aiMoveCancel  context.CancelFunc
	aiMoveVersion uint64
	aiThinking    bool
	// stateVersion is the authoritative monotonic version for room gameplay
	// snapshots. Increment it only while holding mu and only after mutations
	// that change the authoritative state a client should reconcile: players,
	// room lifecycle, game board/chess state, AI thinking, resets, and undo.
	// Do not increment it for rejected/no-op actions or activity timestamps.
	stateVersion       uint64
	stockfish          *StockfishEngine
	finishedResetDelay time.Duration
	resettingDelay     time.Duration
	resetCancel        context.CancelFunc
	resetVersion       uint64
	stateNotifier      func(RoomSnapshot)
	chatMessages       []ChatMessage
	mu                 sync.RWMutex
}

type JoinRoomResponse struct {
	PlayerID   string  `json:"player_id"`
	PlayerMark string  `json:"player_mark"`
	Room       RoomDTO `json:"room"`
}

type RoomDTO struct {
	RoomID string `json:"room_id"`
}

type PlayerSnapshot struct {
	ID         string              `json:"player_id"`
	Mark       string              `json:"player_mark"`
	IsAI       bool                `json:"is_ai"`
	LastActive time.Time           `json:"LastActive"`
	Session    PlayerSessionStatus `json:"session"`
}

type TicTacToeStateSnapshot struct {
	Board  [3][3]string
	Turn   string
	Winner string
	Status tictactoe.GameStatus
}

type ChessStateSnapshot struct {
	SchemaVersion  int
	FEN            string
	IsActive       bool
	Winner         string
	PGNMoves       []string
	Turn           string
	Status         string
	Result         string
	Ply            int
	FullMoveNumber int
	LastMove       *chess.MoveMetadata
	Check          chess.CheckState
	CapturedPieces chess.CapturedPieces
	LegalMoves     map[string][]string
	AI             ChessAISnapshot
	Undo           ChessUndoSnapshot
}

type ChessAISnapshot struct {
	Enabled  bool
	Thinking bool
	PlayerID string
	Color    string
	Level    int
}

type ChessUndoSnapshot struct {
	CanRequest      bool
	CanUndoNow      bool
	LastUndoablePly int
	Pending         string
}

type RoomSnapshot struct {
	RoomID       string
	StateVersion uint64
	GameType     string
	RoomState    RoomState
	IsActive     bool
	IsAIEnabled  bool
	AILevel      int
	Players      []PlayerSnapshot
	TicTacToe    *TicTacToeStateSnapshot
	Chess        *ChessStateSnapshot
}

func NewRoom(roomID string, gameType string) (*Room, error) {
	room := &Room{
		RoomID:             roomID,
		players:            make(map[string]*Player),
		gameType:           gameType,
		roomState:          RoomStateWaiting,
		aiLevel:            DefaultAILevel,
		aiMoveDelay:        DefaultAIMoveDelay,
		finishedResetDelay: DefaultFinishedResetDelay,
		resettingDelay:     DefaultResettingDelay,
	}

	switch gameType {
	case "tictactoe":
		room.ticTacToe = tictactoe.NewGameState("X")
	case "chess":
		room.chess = chess.NewChessGameState()
	default:
		return nil, errors.New("unknown game type")
	}

	return room, nil
}

func (r *Room) SetResetDelays(finishedDelay time.Duration, resettingDelay time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if finishedDelay > 0 {
		r.finishedResetDelay = finishedDelay
	}
	if resettingDelay > 0 {
		r.resettingDelay = resettingDelay
	}
}

func (r *Room) SetAIMoveDelay(delay time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if delay >= 0 {
		r.aiMoveDelay = delay
	}
}

func (r *Room) SetStateNotifier(notifier func(RoomSnapshot)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stateNotifier = notifier
}

func NewRoomWithAI(roomID string, gameType string) (*Room, error) {
	return NewRoomWithAILevel(roomID, gameType, DefaultAILevel)
}

func NewRoomWithAILevel(roomID string, gameType string, aiLevel int) (*Room, error) {
	room, err := NewRoom(roomID, gameType)
	if err != nil {
		return nil, err
	}

	if err := room.EnableAILevel(aiLevel); err != nil {
		return nil, err
	}

	return room, nil
}

func (r *Room) Players() map[string]PlayerSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make(map[string]PlayerSnapshot, len(r.players))
	for playerID, player := range r.players {
		players[playerID] = playerSnapshot(player)
	}

	return players
}

func (r *Room) IsActive() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.isActiveLocked()
}

func (r *Room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.players) == 0
}

func (r *Room) LastActive() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lastActive time.Time
	for _, player := range r.players {
		if player.LastActive.After(lastActive) {
			lastActive = player.LastActive
		}
	}
	return lastActive
}

func (r *Room) IsAIEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.isAIEnabled
}

func (r *Room) AILevel() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.aiLevel
}

func (r *Room) IsTicTacToe() bool {
	return r.GameType() == "tictactoe"
}

func (r *Room) IsChess() bool {
	return r.GameType() == "chess"
}

func (r *Room) Snapshot() RoomSnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	snapshot := RoomSnapshot{
		RoomID:       r.RoomID,
		StateVersion: r.stateVersion,
		GameType:     r.gameType,
		RoomState:    r.roomState,
		IsActive:     r.isActiveLocked(),
		IsAIEnabled:  r.isAIEnabled,
		AILevel:      r.aiLevel,
		Players:      make([]PlayerSnapshot, 0, len(r.players)),
	}

	for _, player := range r.players {
		snapshot.Players = append(snapshot.Players, playerSnapshot(player))
	}

	if r.ticTacToe != nil {
		snapshot.TicTacToe = &TicTacToeStateSnapshot{
			Board:  r.ticTacToe.Board,
			Turn:   r.ticTacToe.Turn,
			Winner: r.ticTacToe.Winner,
			Status: r.ticTacToe.Status,
		}
	}

	if r.chess != nil {
		ai := r.chessAISnapshotLocked()
		snapshot.Chess = &ChessStateSnapshot{
			SchemaVersion:  chess.SchemaVersion,
			FEN:            r.chess.FEN(),
			IsActive:       r.chess.IsActive(),
			Winner:         r.chess.Winner(),
			PGNMoves:       append([]string(nil), r.chess.PGNMoves()...),
			Turn:           r.chess.CurrentTurn(),
			Status:         r.chess.Status(),
			Result:         r.chess.Result(),
			Ply:            r.chess.Ply(),
			FullMoveNumber: r.chess.FullMoveNumber(),
			LastMove:       r.chess.LastMove(),
			Check:          r.chess.CheckState(),
			CapturedPieces: r.chess.CapturedPieces(),
			LegalMoves:     r.chess.LegalMoves(),
			AI:             ai,
			Undo: ChessUndoSnapshot{
				CanRequest:      r.isAIEnabled && r.chess.CanUndoAI(),
				CanUndoNow:      r.isAIEnabled && r.chess.CanUndoAI(),
				LastUndoablePly: r.chess.LastUndoablePly(),
			},
		}
	}

	return snapshot
}

func (r *Room) GameType() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.gameType
}

func (r *Room) Close() {
	r.mu.Lock()
	r.cancelScheduledResetLocked()
	r.cancelScheduledAIMoveLocked()
	stockfish := r.stockfish
	r.stockfish = nil
	r.mu.Unlock()

	if stockfish != nil {
		stockfish.Close()
	}
}

func (r *Room) transitionLocked(next RoomState) error {
	switch r.roomState {
	case RoomStateWaiting:
		if next == RoomStatePlaying {
			r.roomState = next
			return nil
		}
	case RoomStatePlaying:
		if next == RoomStateFinished {
			r.roomState = next
			return nil
		}
	case RoomStateFinished:
		if next == RoomStateResetting {
			r.roomState = next
			return nil
		}
	case RoomStateResetting:
		if next == RoomStatePlaying {
			r.roomState = next
			return nil
		}
	}

	return errors.New("invalid room state transition")
}

func (r *Room) isActiveLocked() bool {
	return r.roomState == RoomStatePlaying
}

func (r *Room) scheduleResetLocked() {
	if r.resetCancel != nil {
		r.resetCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	r.resetCancel = cancel
	r.resetVersion++
	version := r.resetVersion
	finishedDelay := r.finishedResetDelay
	resettingDelay := r.resettingDelay

	go r.runScheduledReset(ctx, version, finishedDelay, resettingDelay)
}

func (r *Room) runScheduledReset(ctx context.Context, version uint64, finishedDelay time.Duration, resettingDelay time.Duration) {
	if !waitForDelay(ctx, finishedDelay) {
		return
	}

	r.mu.Lock()
	if ctx.Err() != nil || version != r.resetVersion || r.roomState != RoomStateFinished {
		r.mu.Unlock()
		return
	}
	if err := r.transitionLocked(RoomStateResetting); err != nil {
		r.mu.Unlock()
		return
	}
	r.bumpStateVersionLocked()
	r.mu.Unlock()
	r.notifyStateChanged()

	if !waitForDelay(ctx, resettingDelay) {
		return
	}

	r.mu.Lock()
	if ctx.Err() != nil || version != r.resetVersion || r.roomState != RoomStateResetting {
		r.mu.Unlock()
		return
	}

	switch r.gameType {
	case "tictactoe":
		r.resetTicTacToeAfterResettingLocked()
	case "chess":
		r.resetChessAfterResettingLocked()
	}
	r.bumpStateVersionLocked()

	if r.resetCancel != nil {
		r.resetCancel()
		r.resetCancel = nil
	}
	r.mu.Unlock()
	r.notifyStateChanged()
}

func waitForDelay(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *Room) cancelScheduledResetLocked() {
	if r.resetCancel != nil {
		r.resetCancel()
		r.resetCancel = nil
	}
	r.resetVersion++
}

func (r *Room) cancelScheduledAIMoveLocked() {
	if r.aiMoveCancel != nil {
		r.aiMoveCancel()
		r.aiMoveCancel = nil
	}
	r.aiThinking = false
	r.aiMoveVersion++
}

func (r *Room) bumpStateVersionLocked() {
	// Clients should treat lower or equal state_version snapshots as stale for
	// the same room. This protects websocket consumers from delayed broadcasts
	// without changing the existing snapshot synchronization architecture.
	r.stateVersion++
}

func (r *Room) chessAISnapshotLocked() ChessAISnapshot {
	ai := ChessAISnapshot{
		Enabled:  r.isAIEnabled,
		Thinking: r.aiThinking,
		Level:    r.aiLevel,
	}
	for _, player := range r.players {
		if player.IsAI {
			ai.PlayerID = player.ID
			ai.Color = player.Mark
			break
		}
	}
	return ai
}

func (r *Room) notifyStateChanged() {
	r.mu.RLock()
	notifier := r.stateNotifier
	r.mu.RUnlock()

	if notifier == nil {
		return
	}

	notifier(r.Snapshot())
}

func normalizeAILevel(level int) int {
	if level == 0 {
		return DefaultAILevel
	}
	if level < 1 {
		return 1
	}
	if level > 10 {
		return 10
	}
	return level
}

func stockfishPathFromEnv() string {
	path := os.Getenv("STOCKFISH_PATH")
	if path == "" {
		return defaultStockfishPath
	}
	return path
}
