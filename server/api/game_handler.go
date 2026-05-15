package api

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/tsaqiffatih/mini-game/actions"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/service"
)

// Shared utility functions
func parsePayload(ctx context.Context, client *Client, roomID, eventType string, rawMessage json.RawMessage, payload interface{}) bool {
	if err := json.Unmarshal(rawMessage, payload); err != nil {
		sendErrorMessage(client, "Failed to unmarshal JSON into payload struct")
		observability.Logger().WarnContext(ctx, "websocket payload unmarshal failed",
			"room_id", roomID,
			"player_id", client.PlayerID,
			"event_type", eventType,
			"error", err,
		)
		return false
	}
	return true
}

// TicTacToe-related functions
func processTicTacToeMove(
	ctx context.Context,
	player game.PlayerSnapshot,
	client *Client,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	message WebSocketMessage,
) {
	var payload dto.TictactoeMovePayload
	if !parsePayload(ctx, client, roomID, message.Type, message.Payload, &payload) {
		return
	}
	handleMakeMoveTictactoe(
		ctx,
		client,
		clients,
		gameService,
		roomID,
		player.ID,
		payload,
	)
}

func handleMakeMoveTictactoe(
	ctx context.Context,
	client *Client,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	playerID string,
	payload dto.TictactoeMovePayload,
) {
	_, err := gameService.HandleTicTacToeMoveWithContext(
		ctx,
		roomID,
		playerID,
		payload.Row,
		payload.Col,
	)
	if err != nil {
		sendErrorMessage(client, err.Error())
		return
	}

	NotifyTicTacToeClients(clients, gameService, roomID)
}

func NotifyTicTacToeClients(clients *ClientRegistry, gameService *service.GameService, roomID string) {
	snapshot, err := gameService.RoomSnapshot(roomID)
	if err != nil {
		observability.Logger().Warn("room snapshot failed",
			"room_id", roomID,
			"player_id", "",
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	if snapshot.GameType != "tictactoe" || snapshot.TicTacToe == nil {
		return
	}

	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventGameUpdate,
		Payload: marshalPayload(dto.FromRoomSnapshot(snapshot)),
	})
}

// Chess-related functions
func processChessMove(
	ctx context.Context,
	player game.PlayerSnapshot,
	client *Client,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	playerID string,
	message WebSocketMessage) {

	var payload dto.ChessMovePayload
	if err := json.Unmarshal(message.Payload, &payload); err != nil {
		sendChessMoveRejectedWithCode(client, roomID, playerID, payload, "invalid_payload", "Invalid chess move payload")
		observability.Logger().WarnContext(ctx, "websocket chess move payload unmarshal failed",
			"room_id", roomID,
			"player_id", client.PlayerID,
			"event_type", message.Type,
			"error", err,
		)
		return
	}
	handleChessMove(
		ctx,
		client,
		clients,
		gameService,
		roomID,
		playerID,
		payload,
	)
}

func handleChessMove(
	ctx context.Context,
	client *Client,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
	playerID string,
	payload dto.ChessMovePayload,
) {
	if _, err := gameService.HandleChessMoveWithContext(
		ctx,
		roomID,
		playerID,
		payload.From,
		payload.To,
		payload.Promotion,
	); err != nil {
		sendChessMoveRejected(client, roomID, playerID, payload, err)
		return
	}

	snapshot, err := gameService.RoomSnapshotWithContext(ctx, roomID)
	if err != nil {
		observability.Logger().WarnContext(ctx, "room snapshot failed after chess move",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventGameUpdate,
		Payload: marshalPayload(dto.FromRoomSnapshot(snapshot)),
	})
}

func processChessUndo(
	ctx context.Context,
	player game.PlayerSnapshot,
	client *Client,
	clients *ClientRegistry,
	gameService *service.GameService,
	roomID string,
) {
	if err := gameService.HandleChessUndoWithContext(ctx, roomID, player.ID); err != nil {
		sendErrorMessage(client, err.Error())
		return
	}

	snapshot, err := gameService.RoomSnapshotWithContext(ctx, roomID)
	if err != nil {
		observability.Logger().WarnContext(ctx, "room snapshot failed after chess undo",
			"room_id", roomID,
			"player_id", player.ID,
			"event_type", "room_snapshot_error",
			"error", err,
		)
		return
	}

	NotifySnapshotToClients(clients, snapshot, Event{
		Type:    EventGameUpdate,
		Payload: marshalPayload(dto.FromRoomSnapshot(snapshot)),
	})
}

func sendChessMoveRejected(client *Client, roomID string, playerID string, payload dto.ChessMovePayload, err error) {
	sendChessMoveRejectedWithCode(client, roomID, playerID, payload, chessMoveRejectionCode(err), err.Error())
}

func sendChessMoveRejectedWithCode(client *Client, roomID string, playerID string, payload dto.ChessMovePayload, code string, message string) {
	sendEvent(client, actions.CHESS_MOVE_REJECTED, dto.ChessMoveRejectedDTO{
		RoomID:        roomID,
		PlayerID:      playerID,
		AttemptedMove: payload,
		Code:          code,
		Message:       message,
		Sound:         "illegal",
	})
}

func chessMoveRejectionCode(err error) string {
	if err == nil {
		return "unknown"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not your turn"):
		return "not_your_turn"
	case strings.Contains(message, "promotion required"):
		return "promotion_required"
	case strings.Contains(message, "game is not active"), strings.Contains(message, "game not active"):
		return "game_not_active"
	case strings.Contains(message, "player not found"):
		return "player_not_in_room"
	case strings.Contains(message, "illegal move"), strings.Contains(message, "invalid move"):
		return "illegal_move"
	default:
		return "invalid_move"
	}
}
