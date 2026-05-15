package service

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
)

var (
	ErrGameTypeRequired = errors.New("RoomID and GameType are required")
	ErrPlayerNotFound   = errors.New("Player not found")
	ErrRoomNotFound     = errors.New("Room not found")
	ErrGameTypeMismatch = errors.New("Game type not match")
)

var (
	roomCodeRandMu sync.Mutex
	roomCodeRand   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type RoomRepository interface {
	GetByID(ctx context.Context, roomID string) (*game.Room, error)
	Save(ctx context.Context, room *game.Room) error
	Delete(ctx context.Context, roomID string) error
	List(ctx context.Context) ([]*game.Room, error)
}

type GameService struct {
	rooms         RoomRepository
	playerManager *game.PlayerManager
	ctx           context.Context
	roomNotifier  func(game.RoomSnapshot)
}

func NewGameService(
	rooms RoomRepository,
	playerManager *game.PlayerManager,
) *GameService {
	return &GameService{
		rooms:         rooms,
		playerManager: playerManager,
		ctx:           context.Background(),
	}
}

func (s *GameService) SetContext(ctx context.Context) {
	if ctx == nil {
		return
	}
	s.ctx = ctx
}

func (s *GameService) SetRoomNotifier(notifier func(game.RoomSnapshot)) {
	s.roomNotifier = notifier
}

func (s *GameService) context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

func (s *GameService) attachRoomNotifier(room *game.Room) {
	if room == nil {
		return
	}
	room.SetStateNotifier(s.roomNotifier)
}

type RoomCreatedEvent struct {
	RoomID  string
	Payload game.RoomDTO
}

func (s *GameService) AddPlayer(playerID string) (game.PlayerSnapshot, error) {
	return s.playerManager.AddPlayer(playerID)
}

func (s *GameService) CreateRoom(gameType string, playerID string) (*game.JoinRoomResponse, error) {
	return s.CreateRoomWithContext(s.context(), gameType, playerID)
}

func (s *GameService) CreateRoomWithContext(ctx context.Context, gameType string, playerID string) (*game.JoinRoomResponse, error) {
	ctx, endSpan := observability.StartSpan(ctx, "game.create_room")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	if gameType == "" {
		spanErr = ErrGameTypeRequired
		return nil, ErrGameTypeRequired
	}

	player, err := s.playerManager.GetPlayer(playerID)
	if err != nil {
		spanErr = ErrPlayerNotFound
		return nil, ErrPlayerNotFound
	}

	roomID := generateRandomRoomCode()

	room, err := game.NewRoom(roomID, gameType)
	if err != nil {
		spanErr = err
		return nil, err
	}
	s.attachRoomNotifier(room)

	if err := s.rooms.Save(ctx, room); err != nil {
		spanErr = err
		return nil, err
	}

	res, err := room.AddPlayer(player)
	if err != nil {
		spanErr = err
		return nil, err
	}
	observability.Logger().InfoContext(ctx, "room created",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "room_created",
		"game_type", gameType,
	)
	return res, nil
}

func (s *GameService) CreateRoomWithAI(gameType string, playerID string) (*game.JoinRoomResponse, error) {
	return s.CreateRoomWithAIWithContext(s.context(), gameType, playerID)
}

func (s *GameService) CreateRoomWithAIWithContext(ctx context.Context, gameType string, playerID string) (*game.JoinRoomResponse, error) {
	return s.CreateRoomWithAILevelWithContext(ctx, gameType, playerID, game.DefaultAILevel)
}

func (s *GameService) CreateRoomWithAILevelWithContext(ctx context.Context, gameType string, playerID string, aiLevel int) (*game.JoinRoomResponse, error) {
	ctx, endSpan := observability.StartSpan(ctx, "game.create_room_with_ai")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	if gameType == "" {
		spanErr = ErrGameTypeRequired
		return nil, ErrGameTypeRequired
	}

	player, err := s.playerManager.GetPlayer(playerID)
	if err != nil {
		spanErr = ErrPlayerNotFound
		return nil, ErrPlayerNotFound
	}

	roomID := generateRandomRoomCode()

	room, err := game.NewRoomWithAILevel(roomID, gameType, aiLevel)
	if err != nil {
		spanErr = err
		return nil, err
	}
	s.attachRoomNotifier(room)

	if err := s.rooms.Save(ctx, room); err != nil {
		room.Close()
		spanErr = err
		return nil, err
	}

	res, err := room.AddPlayer(player)
	if err != nil {
		room.Close()
		spanErr = err
		return nil, err
	}
	observability.Logger().InfoContext(ctx, "room created with ai",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "room_created",
		"game_type", gameType,
		"ai_enabled", true,
		"ai_level", room.AILevel(),
	)
	return res, nil
}

func (s *GameService) JoinRoom(roomID string, playerID string, gameType string) (*game.JoinRoomResponse, error) {
	return s.JoinRoomWithContext(s.context(), roomID, playerID, gameType)
}

func (s *GameService) JoinRoomWithContext(ctx context.Context, roomID string, playerID string, gameType string) (*game.JoinRoomResponse, error) {
	ctx, endSpan := observability.StartSpan(ctx, "game.join_room")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	player, err := s.playerManager.GetPlayer(playerID)
	if err != nil {
		spanErr = ErrPlayerNotFound
		return nil, ErrPlayerNotFound
	}

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		spanErr = ErrRoomNotFound
		return nil, ErrRoomNotFound
	}

	if room.GameType() != gameType {
		spanErr = ErrGameTypeMismatch
		return nil, ErrGameTypeMismatch
	}

	res, err := room.AddPlayer(player)
	if err != nil {
		spanErr = err
		return nil, err
	}
	observability.Logger().InfoContext(ctx, "player joined room",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "player_joined",
		"game_type", gameType,
	)
	return res, nil
}

func (s *GameService) GetPlayerInRoom(roomID string, playerID string) (game.PlayerSnapshot, error) {
	return s.GetPlayerInRoomWithContext(s.context(), roomID, playerID)
}

func (s *GameService) GetPlayerInRoomWithContext(ctx context.Context, roomID string, playerID string) (game.PlayerSnapshot, error) {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return game.PlayerSnapshot{}, ErrRoomNotFound
	}

	player, err := room.GetPlayer(playerID)
	if err != nil {
		return game.PlayerSnapshot{}, ErrPlayerNotFound
	}

	return player, nil
}

func (s *GameService) UpdatePlayerActivity(roomID string, playerID string) {
	s.UpdatePlayerActivityWithContext(s.context(), roomID, playerID)
}

func (s *GameService) UpdatePlayerActivityWithContext(ctx context.Context, roomID string, playerID string) {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return
	}

	_ = room.TouchPlayer(playerID)
}

func (s *GameService) CreateRoomWithAIByID(roomID string, gameType string) (*RoomCreatedEvent, error) {
	return s.CreateRoomWithAIByIDWithContext(s.context(), roomID, gameType)
}

func (s *GameService) CreateRoomWithAIByIDWithContext(ctx context.Context, roomID string, gameType string) (*RoomCreatedEvent, error) {
	return s.CreateRoomWithAIByIDLevelWithContext(ctx, roomID, gameType, game.DefaultAILevel)
}

func (s *GameService) CreateRoomWithAIByIDLevelWithContext(ctx context.Context, roomID string, gameType string, aiLevel int) (*RoomCreatedEvent, error) {
	room, err := game.NewRoomWithAILevel(roomID, gameType, aiLevel)
	if err != nil {
		return nil, err
	}
	s.attachRoomNotifier(room)

	if err := s.rooms.Save(ctx, room); err != nil {
		room.Close()
		return nil, err
	}

	return &RoomCreatedEvent{
		RoomID:  room.RoomID,
		Payload: game.RoomDTO{RoomID: room.RoomID},
	}, nil
}

func (s *GameService) RoomSnapshot(roomID string) (game.RoomSnapshot, error) {
	return s.RoomSnapshotWithContext(s.context(), roomID)
}

func (s *GameService) RoomSnapshotWithContext(ctx context.Context, roomID string) (game.RoomSnapshot, error) {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return game.RoomSnapshot{}, err
	}

	return room.Snapshot(), nil
}

func (s *GameService) RoomPlayersSnapshot(roomID string) ([]game.PlayerSnapshot, error) {
	room, err := s.rooms.GetByID(s.context(), roomID)
	if err != nil {
		return nil, err
	}

	return room.Snapshot().Players, nil
}

func (s *GameService) HandleChatMessageWithContext(ctx context.Context, roomID string, playerID string, message string) (game.ChatMessage, error) {
	ctx, endSpan := observability.StartSpan(ctx, "game.chat_message")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		spanErr = err
		return game.ChatMessage{}, err
	}

	chatMessage, err := room.AddChatMessage(playerID, message)
	if err != nil {
		spanErr = err
		return game.ChatMessage{}, err
	}

	observability.Logger().InfoContext(ctx, "chat message handled",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "chat_message",
		"message_id", chatMessage.ID,
	)
	return chatMessage, nil
}

func (s *GameService) ChatHistoryWithContext(ctx context.Context, roomID string) ([]game.ChatMessage, error) {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	return room.ChatHistory(), nil
}

func (s *GameService) RemovePlayerAfterDelay(roomID string, playerID string, delay time.Duration, isConnected func(string) bool) {
	s.RemovePlayerAfterDelayWithContext(s.context(), roomID, playerID, delay, isConnected)
}

func (s *GameService) RemovePlayerAfterDelayWithContext(ctx context.Context, roomID string, playerID string, delay time.Duration, isConnected func(string) bool) {
	go func() {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}

		if isConnected(playerID) {
			return
		}

		room, err := s.rooms.GetByID(ctx, roomID)
		if err != nil {
			return
		}

		shouldRemoveRoom := room.HandlePlayerDisconnected(playerID)
		if shouldRemoveRoom {
			_ = s.rooms.Delete(ctx, roomID)
		}
	}()
}

func (s *GameService) MarkPlayerConnected(roomID string, playerID string) error {
	return s.MarkPlayerConnectedWithContext(s.context(), roomID, playerID)
}

func (s *GameService) MarkPlayerConnectedWithContext(ctx context.Context, roomID string, playerID string) error {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return ErrRoomNotFound
	}

	if err := room.MarkPlayerConnected(playerID); err != nil {
		return ErrPlayerNotFound
	}
	return nil
}

func (s *GameService) MarkPlayerDisconnected(roomID string, playerID string) error {
	return s.MarkPlayerDisconnectedWithContext(s.context(), roomID, playerID)
}

func (s *GameService) MarkPlayerDisconnectedWithContext(ctx context.Context, roomID string, playerID string) error {
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return ErrRoomNotFound
	}

	if err := room.MarkPlayerDisconnected(playerID); err != nil {
		return ErrPlayerNotFound
	}
	return nil
}

func (s *GameService) CleanupRooms(ctx context.Context, inactiveFor time.Duration) error {
	rooms, err := s.rooms.List(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, room := range rooms {
		if err := ctx.Err(); err != nil {
			return err
		}

		room.RemoveInactivePlayers(now, inactiveFor)
		if room.IsEmpty() || roomInactive(room, now, inactiveFor) {
			room.Close()
			if err := s.rooms.Delete(ctx, room.RoomID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *GameService) StartRoomCleanup(ctx context.Context, inactiveFor time.Duration, tickerInterval time.Duration) {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.CleanupRooms(ctx, inactiveFor)
		}
	}
}

func roomInactive(room *game.Room, now time.Time, inactiveFor time.Duration) bool {
	lastActive := room.LastActive()
	return !lastActive.IsZero() && now.Sub(lastActive) > inactiveFor
}

//
// ===============================
// TIC TAC TOE
// ===============================
//

func (s *GameService) HandleTicTacToeMove(
	roomID string,
	playerID string,
	row int,
	col int,
) (*game.TicTacToeMoveResult, error) {
	return s.HandleTicTacToeMoveWithContext(s.context(), roomID, playerID, row, col)
}

func (s *GameService) HandleTicTacToeMoveWithContext(
	ctx context.Context,
	roomID string,
	playerID string,
	row int,
	col int,
) (*game.TicTacToeMoveResult, error) {
	ctx, endSpan := observability.StartSpan(ctx, "game.tictactoe_move")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		spanErr = err
		return nil, err
	}

	result, err := room.HandleTicTacToeMove(
		playerID,
		row,
		col,
	)
	if err != nil {
		spanErr = err
		return nil, err
	}

	observability.Logger().InfoContext(ctx, "tictactoe move handled",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "tictactoe_move",
		"row", row,
		"col", col,
		"game_ended", result.GameEnded,
	)
	return result, nil
}

//
// ===============================
// CHESS
// ===============================
//

func (s *GameService) HandleChessMove(
	roomID string,
	playerID string,
	from string,
	to string,
	promotion string,
) (*game.ChessMoveResult, error) {
	return s.HandleChessMoveWithContext(s.context(), roomID, playerID, from, to, promotion)
}

func (s *GameService) HandleChessMoveWithContext(
	ctx context.Context,
	roomID string,
	playerID string,
	from string,
	to string,
	promotion string,
) (*game.ChessMoveResult, error) {
	ctx, endSpan := observability.StartSpan(ctx, "game.chess_move")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		spanErr = err
		return nil, err
	}

	result, err := room.HandleChessMoveWithContext(
		ctx,
		playerID,
		from,
		to,
		promotion,
	)
	if err != nil {
		spanErr = err
		return nil, err
	}
	observability.Logger().InfoContext(ctx, "chess move handled",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "chess_move",
		"from", from,
		"to", to,
	)
	return result, nil
}

func (s *GameService) HandleChessUndoWithContext(ctx context.Context, roomID string, playerID string) error {
	ctx, endSpan := observability.StartSpan(ctx, "game.chess_undo")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		spanErr = err
		return err
	}

	if err := room.HandleChessUndo(playerID); err != nil {
		spanErr = err
		return err
	}

	observability.Logger().InfoContext(ctx, "chess undo handled",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "chess_undo",
	)
	return nil
}

func generateRandomRoomCode() string {
	const possibleCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	gameCode := make([]byte, 7)

	roomCodeRandMu.Lock()
	defer roomCodeRandMu.Unlock()
	for i := 0; i < 7; i++ {
		randomIndex := roomCodeRand.Intn(len(possibleCharacters))
		gameCode[i] = possibleCharacters[randomIndex]
	}

	return string(gameCode)
}
