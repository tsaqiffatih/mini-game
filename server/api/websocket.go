package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/middleware"
	"github.com/tsaqiffatih/mini-game/service"
)

func closeAndLog(conn *websocket.Conn, code int, reason string, ctx context.Context, roomID, playerID string) {
	closeWebsocketWithCode(conn, code, reason)
	observability.Logger().InfoContext(ctx, "websocket closed with custom code",
		"room_id", roomID,
		"player_id", playerID,
		"close_code", code,
		"close_reason", reason,
	)
}

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
	// Use custom origin check to enforce allowed origins.
	CheckOrigin: checkWebSocketOrigin,
}

func checkWebSocketOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	originURL, ok := parseOrigin(origin)
	if !ok {
		return false
	}

	allowedEnv := os.Getenv("ALLOWED_ORIGINS")
	if allowedEnv == "" {
		// No allowed origins configured, deny by default
		return false
	}

	for _, configuredOrigin := range strings.Split(allowedEnv, ",") {
		trimmed := strings.TrimSpace(configuredOrigin)
		if trimmed == "" {
			continue
		}
		allowedURL, ok := parseOrigin(trimmed)
		if !ok {
			continue
		}
		// Allow exact match or if the origin starts with the allowed URL (handles possible trailing slash)
		if originURL == allowedURL || strings.HasPrefix(originURL, allowedURL) {
			return true
		}
	}

	return false
}

func parseOrigin(value string) (string, bool) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}

	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), true
}

const (
	EventRoomUpdate         = "room_update"
	EventGameUpdate         = "game_update"
	EventPlayerJoined       = "player_joined"
	EventPlayerReconnecting = "player_reconnecting"
	EventPlayerReconnected  = "player_reconnected"
	EventPlayerLeft         = "player_left"
	EventChatMessage        = "chat_message"
	EventChatHistory        = "chat_history"

	writeWait       = 10 * time.Second
	pongWait        = 15 * time.Second
	pingPeriod      = 5 * time.Second
	maxMessageBytes = 64 * 1024
)

const (
	CloseCodeRoomExpired         = 4001
	CloseCodeRoomFull            = 4002
	CloseCodeInvalidRoom         = 4003
	CloseCodePlayerNotFound      = 4004
	CloseCodeDuplicateConnection = 4005
	CloseCodeUnauthorized        = 4006
	CloseCodeInternalServerError = 1011 // Standard internal server error
)

const (
	closeFrameWait       = time.Second
	reconnectGracePeriod = 30 * time.Second
)

func HandleWebSocket(
	w http.ResponseWriter,
	r *http.Request,
	clients *ClientRegistry,
	gameService *service.GameService) {

	roomID := r.URL.Query().Get("room_id")
	playerID := r.URL.Query().Get("player_id")
	sessionToken := r.URL.Query().Get("session_token")
	ctx, endSpan := observability.StartSpan(r.Context(), "websocket.connect")
	var spanErr error
	defer func() { endSpan(spanErr) }()

	// for upgrade http connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		spanErr = err
		closeAndLog(conn, CloseCodeInternalServerError, "upgrade failed", ctx, roomID, playerID)
		observability.Logger().WarnContext(ctx, "websocket upgrade failed",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_upgrade_failed",
			"error", err,
		)

		return
	}
	defer conn.Close()

	if roomID == "" || !validRoomID(roomID) {
		closeWebsocketWithCode(conn, CloseCodeInvalidRoom, "invalid room id")
		return
	}

	// Strict validation of room and player IDs
	if !validateRoomAndPlayerIDs(w, roomID, playerID) {
		return
	}

	if !middleware.VerifySessionToken(gameService, playerID, sessionToken) {
		spanErr = errors.New("unauthorized")
		observability.Logger().WarnContext(ctx, "websocket authentication rejected",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_authentication_rejected",
			"close_code", CloseCodeUnauthorized,
		)
		closeWebsocketWithCode(conn, CloseCodeUnauthorized, "unauthorized")
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

	generation := clients.NextGeneration(roomID, playerID)
	wasConnected := clients.IsConnected(roomID, playerID)
	isReconnectReplacement := clients.HasClientInRoom(playerID, roomID)
	wasDisconnected := player.Session == game.PlayerSessionDisconnected
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

	client := clients.AttachWithGenerationInRoom(roomID, playerID, generation, conn, pongWait)

	go client.WritePump(writeWait, pingPeriod)

	observability.Logger().InfoContext(ctx, "websocket connected",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "websocket_connected",
		"generation", client.Generation,
		"connection_id", client.ConnectionID,
	)

	notifyRoomOnConnection(ctx, clients, gameService, roomID, player, connectionEventType(wasConnected, wasDisconnected, isReconnectReplacement))

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
		"connection_id", client.ConnectionID,
	)

	// goroutine for remove player after a delay
	handlePlayerDisconnection(ctx, clients, gameService, roomID, playerID, player, client)
}

func closeWebsocketWithCode(conn *websocket.Conn, code int, reason string) {
	if conn == nil {
		return
	}

	// Send a close frame with the provided code and reason.
	message := websocket.FormatCloseMessage(code, reason)
	_ = conn.SetWriteDeadline(time.Now().Add(closeFrameWait))
	_ = conn.WriteMessage(websocket.CloseMessage, message)
	// Give a brief delay so close frame can be transmitted before socket closes
	time.Sleep(20 * time.Millisecond)
}

func validateWebSocketIDs(roomID string, playerID string) (int, string, bool) {
	if !validRoomID(roomID) {
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
		observability.Logger().InfoContext(ctx, "websocket disconnect ignored for stale client",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "websocket_stale_disconnect_ignored",
			"generation", client.Generation,
			"connection_id", client.ConnectionID,
		)
		return
	}
	if err := gameService.MarkPlayerDisconnectedWithContext(ctx, roomID, playerID); err != nil {
		observability.Logger().WarnContext(ctx, "mark player disconnected failed",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "player_disconnect_mark_failed",
			"generation", client.Generation,
			"connection_id", client.ConnectionID,
			"error", err,
		)
	}

	NotifyToClientsInRoom(clients, gameService, roomID, EventPlayerReconnecting, EventPayload{
		Message:   fmt.Sprintf("Player %s disconnected temporarily", playerID),
		Player:    playerEventDTO(player),
		Timestamp: time.Now(),
	})

	observability.Logger().InfoContext(ctx, "delayed player removal scheduled",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "player_removal_scheduled",
		"delay", 30*time.Second,
		"generation", client.Generation,
		"connection_id", client.ConnectionID,
	)
	gameService.RemovePlayerAfterDelayForGenerationWithCallback(roomID, playerID, client.Generation, reconnectGracePeriod, clients.IsCurrentDisconnectedGenerationInRoom(roomID), func() {
		NotifyToClientsInRoom(clients, gameService, roomID, EventPlayerLeft, EventPayload{
			Message:   fmt.Sprintf("Player %s left the room", playerID),
			Player:    playerEventDTO(player),
			Timestamp: time.Now(),
		})
	})
}

func connectionEventType(wasConnected bool, wasDisconnected bool, isReconnectReplacement bool) string {
	if isReconnectReplacement {
		return EventPlayerReconnected
	}
	if wasConnected {
		return ""
	}
	if wasDisconnected {
		return EventPlayerReconnected
	}
	return EventPlayerJoined
}

func playerEventDTO(player game.PlayerSnapshot) *dto.PlayerDTO {
	playerDTO := dto.FromPlayerSnapshot(player)
	return &playerDTO
}
