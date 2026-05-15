package dto

import (
	"time"

	chessdomain "github.com/tsaqiffatih/mini-game/chess"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

type TictactoeMovePayload struct {
	RoomID   string `json:"room_id"`
	PlayerID string `json:"player_id"`
	Row      int    `json:"row"`
	Col      int    `json:"col"`
}

type TicTacToeGameResponse struct {
	Board    [3][3]string `json:"board"`
	Turn     string       `json:"turn"`
	Winner   string       `json:"winner"`
	IsActive bool         `json:"is_active"`
}

type ChessMovePayload struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Promotion string `json:"promotion,omitempty"`
}

type ChessUndoPayload struct {
	Mode string `json:"mode,omitempty"`
}

type ChessMoveRejectedDTO struct {
	RoomID        string           `json:"room_id"`
	PlayerID      string           `json:"player_id"`
	AttemptedMove ChessMovePayload `json:"attempted_move"`
	Code          string           `json:"code"`
	Message       string           `json:"message"`
	Sound         string           `json:"sound"`
}

type ChatSendPayload struct {
	Message string `json:"message"`
}

type ChatMessageDTO struct {
	ID         string    `json:"id"`
	RoomID     string    `json:"room_id,omitempty"`
	PlayerID   string    `json:"player_id"`
	PlayerMark string    `json:"player_mark"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

type ChatHistoryDTO struct {
	Messages []ChatMessageDTO `json:"messages"`
}

type PlayerDTO struct {
	ID         string    `json:"id"`
	PlayerID   string    `json:"player_id"`
	Mark       string    `json:"mark"`
	PlayerMark string    `json:"player_mark"`
	IsAI       bool      `json:"is_ai"`
	LastActive time.Time `json:"last_active"`
	Session    string    `json:"session"`
}

type RoomDTO struct {
	ID     string `json:"id"`
	RoomID string `json:"room_id"`
}

type JoinRoomResponseDTO struct {
	PlayerID   string  `json:"player_id"`
	PlayerMark string  `json:"player_mark"`
	Room       RoomDTO `json:"room"`
}

type RoomSnapshotDTO struct {
	ID           string             `json:"id"`
	RoomID       string             `json:"room_id"`
	StateVersion uint64             `json:"state_version"`
	GameType     string             `json:"game_type"`
	State        string             `json:"state"`
	RoomState    string             `json:"room_state"`
	IsActive     bool               `json:"is_active"`
	IsAIEnabled  bool               `json:"is_ai_enabled"`
	AILevel      int                `json:"ai_level"`
	Players      []PlayerDTO        `json:"players"`
	Game         *GameStateDTO      `json:"game,omitempty"`
	TicTacToe    *TicTacToeStateDTO `json:"tictactoe,omitempty"`
	// Deprecated: chess clients should read the canonical state from
	// game.chess. This top-level alias is kept temporarily for older clients.
	Chess *ChessStateDTO `json:"chess,omitempty"`
}

type GameStateDTO struct {
	Type      string             `json:"type"`
	TicTacToe *TicTacToeStateDTO `json:"tictactoe,omitempty"`
	// Canonical chess state source for websocket snapshots.
	Chess *ChessStateDTO `json:"chess,omitempty"`
}

type TicTacToeStateDTO struct {
	Board    [3][3]string         `json:"board"`
	Turn     string               `json:"turn"`
	Winner   string               `json:"winner"`
	Status   tictactoe.GameStatus `json:"status"`
	IsActive bool                 `json:"is_active"`
}

type ChessStateDTO struct {
	SchemaVersion  int                        `json:"schema_version"`
	FEN            string                     `json:"fen"`
	IsActive       bool                       `json:"is_active"`
	Winner         string                     `json:"winner"`
	PGNMoves       []string                   `json:"pgn_moves"`
	Turn           string                     `json:"turn"`
	Status         string                     `json:"status"`
	Result         string                     `json:"result,omitempty"`
	Ply            int                        `json:"ply"`
	FullMoveNumber int                        `json:"fullmove_number"`
	LastMove       *chessdomain.MoveMetadata  `json:"last_move,omitempty"`
	Check          chessdomain.CheckState     `json:"check"`
	CapturedPieces chessdomain.CapturedPieces `json:"captured_pieces"`
	LegalMoves     map[string][]string        `json:"legal_moves"`
	AI             ChessAIDTO                 `json:"ai"`
	Undo           ChessUndoDTO               `json:"undo"`
}

type ChessAIDTO struct {
	Enabled  bool   `json:"enabled"`
	Thinking bool   `json:"thinking"`
	PlayerID string `json:"player_id,omitempty"`
	Color    string `json:"color,omitempty"`
	Level    int    `json:"level"`
}

type ChessUndoDTO struct {
	CanRequest      bool   `json:"can_request"`
	CanUndoNow      bool   `json:"can_undo_now"`
	LastUndoablePly int    `json:"last_undoable_ply"`
	Pending         string `json:"pending,omitempty"`
}

type MoveResponseDTO struct {
	Room  RoomSnapshotDTO `json:"room"`
	Ended bool            `json:"ended"`
}

func FromJoinRoomResponse(res *game.JoinRoomResponse) JoinRoomResponseDTO {
	if res == nil {
		return JoinRoomResponseDTO{}
	}

	return JoinRoomResponseDTO{
		PlayerID:   res.PlayerID,
		PlayerMark: res.PlayerMark,
		Room:       RoomDTO{ID: res.Room.RoomID, RoomID: res.Room.RoomID},
	}
}

func FromPlayerSnapshot(player game.PlayerSnapshot) PlayerDTO {
	return PlayerDTO{
		ID:         player.ID,
		PlayerID:   player.ID,
		Mark:       player.Mark,
		PlayerMark: player.Mark,
		IsAI:       player.IsAI,
		LastActive: player.LastActive,
		Session:    string(player.Session),
	}
}

func FromChatMessage(message game.ChatMessage) ChatMessageDTO {
	return ChatMessageDTO{
		ID:         message.ID,
		PlayerID:   message.PlayerID,
		PlayerMark: message.PlayerMark,
		Message:    message.Message,
		CreatedAt:  message.CreatedAt,
	}
}

func FromChatMessageEvent(message game.ChatMessage) ChatMessageDTO {
	dto := FromChatMessage(message)
	dto.RoomID = message.RoomID
	return dto
}

func FromChatMessages(messages []game.ChatMessage) []ChatMessageDTO {
	dtos := make([]ChatMessageDTO, 0, len(messages))
	for _, message := range messages {
		dtos = append(dtos, FromChatMessage(message))
	}
	return dtos
}

func FromChatHistory(messages []game.ChatMessage) ChatHistoryDTO {
	return ChatHistoryDTO{Messages: FromChatMessages(messages)}
}

func FromRoomSnapshot(snapshot game.RoomSnapshot) RoomSnapshotDTO {
	players := make([]PlayerDTO, 0, len(snapshot.Players))
	for _, player := range snapshot.Players {
		players = append(players, FromPlayerSnapshot(player))
	}

	dto := RoomSnapshotDTO{
		ID:           snapshot.RoomID,
		RoomID:       snapshot.RoomID,
		StateVersion: snapshot.StateVersion,
		GameType:     snapshot.GameType,
		State:        string(snapshot.RoomState),
		RoomState:    string(snapshot.RoomState),
		IsActive:     snapshot.IsActive,
		IsAIEnabled:  snapshot.IsAIEnabled,
		AILevel:      snapshot.AILevel,
		Players:      players,
	}

	gameState := &GameStateDTO{Type: snapshot.GameType}
	if snapshot.TicTacToe != nil {
		ticTacToe := &TicTacToeStateDTO{
			Board:    snapshot.TicTacToe.Board,
			Turn:     snapshot.TicTacToe.Turn,
			Winner:   snapshot.TicTacToe.Winner,
			Status:   snapshot.TicTacToe.Status,
			IsActive: snapshot.TicTacToe.Status == tictactoe.StatusActive,
		}
		dto.TicTacToe = ticTacToe
		gameState.TicTacToe = ticTacToe
	}

	if snapshot.Chess != nil {
		chess := &ChessStateDTO{
			SchemaVersion:  snapshot.Chess.SchemaVersion,
			FEN:            snapshot.Chess.FEN,
			IsActive:       snapshot.Chess.IsActive,
			Winner:         snapshot.Chess.Winner,
			PGNMoves:       append([]string(nil), snapshot.Chess.PGNMoves...),
			Turn:           snapshot.Chess.Turn,
			Status:         snapshot.Chess.Status,
			Result:         snapshot.Chess.Result,
			Ply:            snapshot.Chess.Ply,
			FullMoveNumber: snapshot.Chess.FullMoveNumber,
			LastMove:       snapshot.Chess.LastMove,
			Check:          snapshot.Chess.Check,
			CapturedPieces: snapshot.Chess.CapturedPieces,
			LegalMoves:     snapshot.Chess.LegalMoves,
			AI: ChessAIDTO{
				Enabled:  snapshot.Chess.AI.Enabled,
				Thinking: snapshot.Chess.AI.Thinking,
				PlayerID: snapshot.Chess.AI.PlayerID,
				Color:    snapshot.Chess.AI.Color,
				Level:    snapshot.Chess.AI.Level,
			},
			Undo: ChessUndoDTO{
				CanRequest:      snapshot.Chess.Undo.CanRequest,
				CanUndoNow:      snapshot.Chess.Undo.CanUndoNow,
				LastUndoablePly: snapshot.Chess.Undo.LastUndoablePly,
				Pending:         snapshot.Chess.Undo.Pending,
			},
		}
		// game.chess is the canonical chess state. The top-level chess field
		// intentionally points at the same DTO as a deprecated compatibility
		// alias so clients do not observe diverging chess representations.
		dto.Chess = chess
		gameState.Chess = chess
	}

	dto.Game = gameState
	return dto
}
