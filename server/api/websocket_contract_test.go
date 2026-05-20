package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/infrastructure"
	"github.com/tsaqiffatih/mini-game/service"
)

func TestNotifyToClientsInRoom_RoomUpdateUsesDirectSnapshotPayload(t *testing.T) {
	repo := infrastructure.NewMemoryRoomRepository()
	gameService := service.NewGameService(repo, game.NewPlayerManager())
	if _, err := gameService.AddPlayer("p1"); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
	res, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoomWithContext() error = %v", err)
	}

	client := newBufferedTestClient("p1")
	clients := NewClientRegistry()
	clients.clients["p1"] = client

	NotifyToClientsInRoom(clients, gameService, res.Room.RoomID, EventRoomUpdate, dto.RoomDTO{
		ID:     res.Room.RoomID,
		RoomID: res.Room.RoomID,
	})

	event := readTestEvent(t, client)
	if event.Type != EventRoomUpdate {
		t.Fatalf("event type = %q, want %q", event.Type, EventRoomUpdate)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if _, wrapped := payload["room"]; wrapped {
		t.Fatalf("room_update payload was wrapped with room field: %s", event.Payload)
	}
	if _, ok := payload["room_id"]; !ok {
		t.Fatalf("room_update payload missing room_id: %s", event.Payload)
	}
}

func TestFromRoomSnapshot_ChessCanonicalAndDeprecatedAliasShareState(t *testing.T) {
	room, err := game.NewRoom("chess-room", "chess")
	if err != nil {
		t.Fatalf("NewRoom() error = %v", err)
	}
	addPlayerToContractRoom(t, room, "p1")

	snapshotDTO := dto.FromRoomSnapshot(room.Snapshot())
	if snapshotDTO.Game == nil || snapshotDTO.Game.Chess == nil {
		t.Fatalf("canonical game.chess is nil")
	}
	if snapshotDTO.Chess == nil {
		t.Fatalf("deprecated top-level chess alias is nil")
	}
	if snapshotDTO.Game.Chess != snapshotDTO.Chess {
		t.Fatalf("game.chess and top-level chess should share the same DTO instance")
	}
}

func TestProcessChessMove_InvalidPayloadSendsMoveRejected(t *testing.T) {
	client := newBufferedTestClient("p1")

	processChessMove(
		context.Background(),
		game.PlayerSnapshot{ID: "p1", Mark: "white"},
		client,
		NewClientRegistry(),
		nil,
		"room-1",
		"p1",
		WebSocketMessage{Type: "CHESS_MOVE", Payload: json.RawMessage(`{"from":`)},
	)

	event := readTestEvent(t, client)
	if event.Type != "chess_move_rejected" {
		t.Fatalf("event type = %q, want chess_move_rejected", event.Type)
	}

	var rejected dto.ChessMoveRejectedDTO
	if err := json.Unmarshal(event.Payload, &rejected); err != nil {
		t.Fatalf("decode rejected payload: %v", err)
	}
	if rejected.Code != "invalid_payload" {
		t.Fatalf("rejection code = %q, want invalid_payload", rejected.Code)
	}
}

func TestHandleWebSocket_MissingRoomID_UpgradesThenClosesInvalidRoom(t *testing.T) {
	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	server := newWebSocketTestServer(t, NewClientRegistry(), gameService)

	conn := dialTestWebSocket(t, server, "player_id=p1")
	defer conn.Close()

	assertWebSocketCloseCode(t, conn, CloseCodeInvalidRoom)
}

func TestHandleWebSocket_PlayerNotFound_UpgradesThenClosesPlayerNotFound(t *testing.T) {
	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	if _, err := gameService.AddPlayer("p1"); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
	res, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoomWithContext() error = %v", err)
	}
	server := newWebSocketTestServer(t, NewClientRegistry(), gameService)

	conn := dialTestWebSocket(t, server, "room_id="+res.Room.RoomID+"&player_id=missing")
	defer conn.Close()

	assertWebSocketCloseCode(t, conn, CloseCodePlayerNotFound)
}

func TestHandleWebSocket_DuplicateConnection_ReplacesExistingConnection(t *testing.T) {
	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	if _, err := gameService.AddPlayer("p1"); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
	res, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoomWithContext() error = %v", err)
	}
	clients := NewClientRegistry()
	server := newWebSocketTestServer(t, clients, gameService)
	query := "room_id=" + res.Room.RoomID + "&player_id=p1"

	firstConn := dialTestWebSocket(t, server, query)
	defer firstConn.Close()

	secondConn := dialTestWebSocket(t, server, query)
	defer secondConn.Close()

	assertWebSocketCloseCode(t, firstConn, CloseCodeDuplicateConnection)
	if !clients.IsConnected("p1") {
		t.Fatalf("new duplicate connection should remain active")
	}
}

func TestHandlePlayerDisconnection_BroadcastsReconnectingNotLeft(t *testing.T) {
	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	if _, err := gameService.AddPlayer("p1"); err != nil {
		t.Fatalf("AddPlayer(p1) error = %v", err)
	}
	if _, err := gameService.AddPlayer("p2"); err != nil {
		t.Fatalf("AddPlayer(p2) error = %v", err)
	}
	res, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoomWithContext() error = %v", err)
	}
	if _, err := gameService.JoinRoomWithContext(context.Background(), res.Room.RoomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoomWithContext() error = %v", err)
	}

	clients := NewClientRegistry()
	p1Client := newBufferedTestClient("p1")
	p2Client := newBufferedTestClient("p2")
	p2Client.Generation = 1
	clients.clients["p1"] = p1Client
	clients.clients["p2"] = p2Client
	clients.generations["p2"] = 1

	handlePlayerDisconnection(
		context.Background(),
		clients,
		gameService,
		res.Room.RoomID,
		"p2",
		game.PlayerSnapshot{ID: "p2"},
		p2Client,
	)

	event := readTestEvent(t, p1Client)
	if event.Type != EventPlayerReconnecting {
		t.Fatalf("event type = %q, want %q", event.Type, EventPlayerReconnecting)
	}
}

func TestConnectionEventType(t *testing.T) {
	tests := []struct {
		name            string
		wasConnected    bool
		wasDisconnected bool
		want            string
	}{
		{name: "duplicate active connection", wasConnected: true, wasDisconnected: false, want: ""},
		{name: "reconnect after temporary disconnect", wasConnected: false, wasDisconnected: true, want: EventPlayerReconnected},
		{name: "new websocket join", wasConnected: false, wasDisconnected: false, want: EventPlayerJoined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connectionEventType(tt.wasConnected, tt.wasDisconnected); got != tt.want {
				t.Fatalf("connectionEventType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func newBufferedTestClient(playerID string) *Client {
	return &Client{
		PlayerID: playerID,
		Send:     make(chan []byte, 1),
		done:     make(chan struct{}),
	}
}

func newWebSocketTestServer(t *testing.T, clients *ClientRegistry, gameService *service.GameService) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(w, r, clients, gameService)
	}))
	t.Cleanup(server.Close)
	return server
}

func dialTestWebSocket(t *testing.T, server *httptest.Server, query string) *websocket.Conn {
	t.Helper()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	if query != "" {
		wsURL += "?" + query
	}

	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("websocket dial error = %v, status = %d", err, status)
	}
	return conn
}

func assertWebSocketCloseCode(t *testing.T, conn *websocket.Conn, expectedCode int) {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	for {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}

		var closeErr *websocket.CloseError
		if errors.As(err, &closeErr) {
			if closeErr.Code != expectedCode {
				t.Fatalf("close code = %d, want %d; reason=%q", closeErr.Code, expectedCode, closeErr.Text)
			}
			return
		}

		t.Fatalf("ReadMessage() error = %v, want websocket close code %d", err, expectedCode)
	}
}

func readTestEvent(t *testing.T, client *Client) Event {
	t.Helper()

	select {
	case message := <-client.Send:
		var event Event
		if err := json.Unmarshal(message, &event); err != nil {
			t.Fatalf("decode event: %v; message=%s", err, message)
		}
		return event
	default:
		t.Fatalf("client did not receive an event")
		return Event{}
	}
}

func addPlayerToContractRoom(t *testing.T, room *game.Room, playerID string) {
	t.Helper()

	_, err := room.AddPlayer(game.PlayerSnapshot{
		ID:         playerID,
		LastActive: roomTestTime(),
	})
	if err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
}

func roomTestTime() time.Time {
	return time.Now().UTC()
}
