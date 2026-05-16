package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/service"
)

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventPayload struct {
	Message   string         `json:"message,omitempty"`
	Player    *dto.PlayerDTO `json:"player,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

type RoomEventPayload struct {
	Room dto.RoomSnapshotDTO `json:"room"`
	Data json.RawMessage     `json:"data,omitempty"`
}

type ErrorMessage struct {
	Message string `json:"message"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

const (
	EventRoomUpdate   = "room_update"
	EventGameUpdate   = "game_update"
	EventPlayerJoined = "player_joined"
	EventPlayerLeft   = "player_left"
	EventChatMessage  = "chat_message"
	EventChatHistory  = "chat_history"

	writeWait  = 10 * time.Second
	pingPeriod = 30 * time.Second
	pongWait   = 60 * time.Second
)

const (
	CloseCodeRoomExpired         = 4001
	CloseCodeRoomFull            = 4002
	CloseCodeInvalidRoom         = 4003
	CloseCodePlayerNotFound      = 4004
	CloseCodeDuplicateConnection = 4005
)

const closeFrameWait = time.Second

func HandleWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	clients *ClientRegistry,
	gameService *service.GameService) {

	roomID := r.URL.Query().Get("room_id")
	playerID := r.URL.Query().Get("player_id")
	ctx, endSpan := observability.StartSpan(r.Context(), "websocket.connect")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	// for upgrade http connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		spanErr = err
		observability.Logger().WarnContext(ctx, "websocket upgrade failed",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_upgrade_failed",
			"error", err,
		)
		return
	}
	defer conn.Close()

	if code, reason, ok := validateWebSocketIDs(roomID, playerID); !ok {
		observability.Logger().WarnContext(ctx, "websocket validation failed",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_validation_failed",
			"close_code", code,
			"close_reason", reason,
		)
		closeWebsocketWithCode(conn, code, reason)
		return
	}

	player, err := gameService.GetPlayerInRoomWithContext(ctx, roomID, playerID)
	if err != nil {
		spanErr = err
		code, reason := websocketCloseForValidationError(err)
		observability.Logger().WarnContext(ctx, "websocket session rejected",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_session_rejected",
			"close_code", code,
			"close_reason", reason,
			"error", err,
		)
		closeWebsocketWithCode(conn, code, reason)
		return
	}

	generation := clients.NextGeneration(playerID)
	if err := gameService.MarkPlayerConnectedWithContext(ctx, roomID, playerID); err != nil {
		spanErr = err
		code, reason := websocketCloseForValidationError(err)
		observability.Logger().WarnContext(ctx, "websocket session rejected",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_session_rejected",
			"close_code", code,
			"close_reason", reason,
			"error", err,
		)
		closeWebsocketWithCode(conn, code, reason)
		return
	}

	client := clients.AttachWithGeneration(playerID, generation, conn, pongWait)

	go client.WritePump(writeWait, pingPeriod)

	observability.Logger().InfoContext(ctx, "websocket connected",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "websocket_connected",
		"generation", client.Generation,
	)

	notifyRoomOnConnection(ctx, clients, gameService, roomID, player)

	done := make(chan websocketReadResult, 1)

	// goroutine to read messages
	go readMessages(ctx, conn, done, clients, gameService, roomID, player, client)

	readResult := <-done
	observability.Logger().InfoContext(ctx, "websocket disconnected",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "websocket_disconnected",
		"close_code", readResult.CloseCode,
		"close_reason", readResult.CloseReason,
		"generation", client.Generation,
	)

	// goroutine for remove player after a delay
	handlePlayerDisconnection(ctx, clients, gameService, roomID, playerID, player, client)
}

func closeWebsocketWithCode(conn *websocket.Conn, code int, reason string) {
	if conn == nil {
		return
	}

	message := websocket.FormatCloseMessage(code, reason)
	deadline := time.Now().Add(closeFrameWait)
	_ = conn.WriteControl(websocket.CloseMessage, message, deadline)
	_ = conn.Close()
}

func validateWebSocketIDs(roomID string, playerID string) (int, string, bool) {
	if roomID == "" {
		return CloseCodeInvalidRoom, "invalid room", false
	}
	if playerID == "" {
		return CloseCodePlayerNotFound, "player not found", false
	}
	return websocket.CloseNormalClosure, "", true
}

func websocketCloseForValidationError(err error) (int, string) {
	switch {
	case errors.Is(err, service.ErrRoomNotFound):
		return CloseCodeRoomExpired, "room expired"
	case errors.Is(err, service.ErrPlayerNotFound):
		return CloseCodePlayerNotFound, "player not found"
	default:
		return websocket.CloseInternalServerErr, "internal error"
	}
}

func handlePlayerDisconnection(ctx context.Context, clients *ClientRegistry, gameService *service.GameService, roomID string, playerID string, player game.PlayerSnapshot, client *Client) {
	if !clients.RemoveClient(client) {
		return
	}
	_ = gameService.MarkPlayerDisconnectedWithContext(ctx, roomID, playerID)

	NotifyToClientsInRoom(clients, gameService, roomID, EventPlayerLeft, EventPayload{
		Message:   fmt.Sprintf("Player %s left the room", playerID),
		Player:    playerEventDTO(player),
		Timestamp: time.Now(),
	})

	observability.Logger().InfoContext(ctx, "delayed player removal scheduled",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "player_removal_scheduled",
		"delay", 30*time.Second,
		"generation", client.Generation,
	)
	gameService.RemovePlayerAfterDelayForGeneration(roomID, playerID, client.Generation, 30*time.Second, clients.IsCurrentDisconnectedGeneration)
}

func playerEventDTO(player game.PlayerSnapshot) *dto.PlayerDTO {
	playerDTO := dto.FromPlayerSnapshot(player)
	return &playerDTO
}
