package api

import (
	"context"
	"encoding/json"
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

	if !validateRoomAndPlayerIDs(w, roomID, playerID) {
		return
	}

	player, err := gameService.GetPlayerInRoomWithContext(ctx, roomID, playerID)
	if err != nil {
		spanErr = err
		if err == service.ErrPlayerNotFound {
			sendErrorResponse(w, "player not found in room", http.StatusNotFound)
			return
		}
		sendErrorResponse(w, "could not find room", http.StatusNotFound)
		return
	}

	// for upgrade http connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		spanErr = err
		sendErrorResponse(w, "could not upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	client := clients.Attach(playerID, conn, pongWait)
	_ = gameService.MarkPlayerConnectedWithContext(ctx, roomID, playerID)

	go client.WritePump(writeWait, pingPeriod)

	observability.Logger().InfoContext(ctx, "websocket connected",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "websocket_connected",
	)

	notifyRoomOnConnection(ctx, clients, gameService, roomID, player)

	done := make(chan struct{})

	// goroutine to read messages
	go readMessages(ctx, conn, done, clients, gameService, roomID, player, client)

	<-done
	observability.Logger().InfoContext(ctx, "websocket disconnected",
		"room_id", roomID,
		"player_id", playerID,
		"event_type", "websocket_disconnected",
	)

	// goroutine for remove player after a delay
	handlePlayerDisconnection(ctx, clients, gameService, roomID, playerID, player, client)
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

	gameService.RemovePlayerAfterDelay(roomID, playerID, 30*time.Second, clients.IsConnected)
}

func playerEventDTO(player game.PlayerSnapshot) *dto.PlayerDTO {
	playerDTO := dto.FromPlayerSnapshot(player)
	return &playerDTO
}
