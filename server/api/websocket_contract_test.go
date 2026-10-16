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
	client.RoomID = res.Room.RoomID
	clients := NewClientRegistry()
	clients.clients[newClientKey(res.Room.RoomID, "p1")] = client

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

	conn := dialTestWebSocket(t, server, "room_id="+res.Room.RoomID+"&player_id=missing&session_token=invalid")
	defer conn.Close()

	assertWebSocketCloseCode(t, conn, CloseCodeUnauthorized)
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
	token, err := gameService.SessionTokenForPlayer("p1")
	if err != nil {
		t.Fatalf("SessionTokenForPlayer() error = %v", err)
	}
	query := "room_id=" + res.Room.RoomID + "&player_id=p1&session_token=" + token

	firstConn := dialTestWebSocket(t, server, query)
	defer firstConn.Close()

	// Ensure the first connection has registered before connecting the second one.
	for i := 0; i < 50; i++ {
		if clients.IsConnected(res.Room.RoomID, "p1") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	secondConn := dialTestWebSocket(t, server, query)
	defer secondConn.Close()

	assertWebSocketCloseCode(t, firstConn, CloseCodeDuplicateConnection)
	if !clients.IsConnected(res.Room.RoomID, "p1") {
		t.Fatalf("new duplicate connection should remain active")
	}
}

func TestHandleWebSocket_RequiresMatchingSessionToken(t *testing.T) {
	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	if _, err := gameService.AddPlayer("player-a"); err != nil {
		t.Fatalf("AddPlayer(player-a) error = %v", err)
	}
	if _, err := gameService.AddPlayer("player-b"); err != nil {
		t.Fatalf("AddPlayer(player-b) error = %v", err)
	}
	room, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "player-a")
	if err != nil {
		t.Fatalf("CreateRoomWithContext() error = %v", err)
	}
	if _, err := gameService.JoinRoomWithContext(context.Background(), room.Room.RoomID, "player-b", "tictactoe"); err != nil {
		t.Fatalf("JoinRoomWithContext() error = %v", err)
	}
	tokenA, err := gameService.SessionTokenForPlayer("player-a")
	if err != nil {
		t.Fatalf("SessionTokenForPlayer(player-a) error = %v", err)
	}
	server := newWebSocketTestServer(t, NewClientRegistry(), gameService)

	for _, test := range []struct {
		name  string
		query string
		code  int
	}{
		{name: "missing token", query: "room_id=" + room.Room.RoomID + "&player_id=player-a", code: CloseCodeUnauthorized},
		{name: "invalid token", query: "room_id=" + room.Room.RoomID + "&player_id=player-a&session_token=invalid", code: CloseCodeUnauthorized},
		{name: "cross-player token", query: "room_id=" + room.Room.RoomID + "&player_id=player-b&session_token=" + tokenA, code: CloseCodeUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			conn := dialTestWebSocket(t, server, test.query)
			defer conn.Close()
			assertWebSocketCloseCode(t, conn, test.code)
		})
	}

	conn := dialTestWebSocket(t, server, "room_id="+room.Room.RoomID+"&player_id=player-a&session_token="+tokenA)
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"TICTACTOE_MOVE","payload":{"player_id":"player-b","room_id":"other-room","row":0,"col":0}}`)); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	for i := 0; i < 3; i++ {
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("ReadMessage() error = %v", err)
		}
	}

	deadline := time.Now().Add(time.Second)
	for {
		snapshot, err := gameService.RoomSnapshot(room.Room.RoomID)
		if err != nil {
			t.Fatalf("RoomSnapshot() error = %v", err)
		}
		if snapshot.TicTacToe != nil && snapshot.TicTacToe.Board[0][0] != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("authenticated websocket action was not applied")
		}
		time.Sleep(10 * time.Millisecond)
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
	p1Client.RoomID = res.Room.RoomID
	p2Client.RoomID = res.Room.RoomID
	p2Client.Generation = 1
	clients.clients[newClientKey(res.Room.RoomID, "p1")] = p1Client
	clients.clients[newClientKey(res.Room.RoomID, "p2")] = p2Client
	clients.generations[newClientKey(res.Room.RoomID, "p2")] = 1

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

func TestFastReconnectReplacement_BroadcastsReconnectedToOpponent(t *testing.T) {
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
	p1Client.RoomID = res.Room.RoomID
	p2Client.RoomID = res.Room.RoomID
	clients.clients[newClientKey(res.Room.RoomID, "p1")] = p1Client
	clients.clients[newClientKey(res.Room.RoomID, "p2")] = p2Client

	player, err := gameService.GetPlayerInRoomWithContext(context.Background(), res.Room.RoomID, "p2")
	if err != nil {
		t.Fatalf("GetPlayerInRoomWithContext() error = %v", err)
	}

	notifyRoomOnConnection(
		context.Background(),
		clients,
		gameService,
		res.Room.RoomID,
		player,
		connectionEventType(true, false, true),
	)

	event := readTestEvent(t, p1Client)
	if event.Type != EventPlayerReconnected {
		t.Fatalf("event type = %q, want %q", event.Type, EventPlayerReconnected)
	}
}

func TestHandlePlayerDisconnection_StaleOldClientDoesNotDisconnectNewSession(t *testing.T) {
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
	oldP2Client := newBufferedTestClient("p2")
	newP2Client := newBufferedTestClient("p2")
	p1Client.RoomID = res.Room.RoomID
	oldP2Client.RoomID = res.Room.RoomID
	oldP2Client.Generation = 1
	newP2Client.RoomID = res.Room.RoomID
	newP2Client.Generation = 2
	clients.clients[newClientKey(res.Room.RoomID, "p1")] = p1Client
	clients.clients[newClientKey(res.Room.RoomID, "p2")] = newP2Client
	clients.generations[newClientKey(res.Room.RoomID, "p2")] = 2

	player, err := gameService.GetPlayerInRoomWithContext(context.Background(), res.Room.RoomID, "p2")
	if err != nil {
		t.Fatalf("GetPlayerInRoomWithContext() error = %v", err)
	}

	handlePlayerDisconnection(
		context.Background(),
		clients,
		gameService,
		res.Room.RoomID,
		"p2",
		player,
		oldP2Client,
	)

	assertNoTestEvent(t, p1Client)

	player, err = gameService.GetPlayerInRoomWithContext(context.Background(), res.Room.RoomID, "p2")
	if err != nil {
		t.Fatalf("GetPlayerInRoomWithContext() after stale disconnect error = %v", err)
	}
	if player.Session != game.PlayerSessionConnected {
		t.Fatalf("player session = %q, want %q", player.Session, game.PlayerSessionConnected)
	}
	if !clients.IsConnected(res.Room.RoomID, "p2") {
		t.Fatalf("new p2 client should remain connected")
	}
}

func TestHandleWebSocket_EnforcesAllowedOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.test,http://localhost:3000")

	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	if _, err := gameService.AddPlayer("p1"); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
	room, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoomWithContext() error = %v", err)
	}
	token, err := gameService.SessionTokenForPlayer("p1")
	if err != nil {
		t.Fatalf("SessionTokenForPlayer() error = %v", err)
	}
	server := newWebSocketTestServer(t, NewClientRegistry(), gameService)
	query := "room_id=" + room.Room.RoomID + "&player_id=p1&session_token=" + token

	allowed := dialTestWebSocketWithOrigin(t, server, query, "https://app.example.test")
	allowed.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?" + query
	_, response, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"https://evil.example.test"}})
	if err == nil {
		t.Fatal("disallowed origin connected")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		t.Fatalf("disallowed origin status = %d, want %d", status, http.StatusForbidden)
	}
}

func TestClientRegistry_IsolatesRoomsForSamePlayer(t *testing.T) {
	clients := NewClientRegistry()
	roomAClient := newBufferedTestClient("player-a")
	roomAClient.RoomID = "room-a"
	roomBClient := newBufferedTestClient("player-a")
	roomBClient.RoomID = "room-b"
	clients.clients[newClientKey("room-a", "player-a")] = roomAClient
	clients.clients[newClientKey("room-b", "player-a")] = roomBClient

	if !clients.IsConnected("room-a", "player-a") || !clients.IsConnected("room-b", "player-a") {
		t.Fatal("same player should remain connected in both rooms")
	}
	if !clients.RemoveClient(roomAClient) {
		t.Fatal("room A client should be removed")
	}
	if clients.IsConnected("room-a", "player-a") {
		t.Fatal("room A client remained registered")
	}
	if !clients.IsConnected("room-b", "player-a") {
		t.Fatal("room B client was removed by room A disconnect")
	}
}

func TestNotifySnapshotToClients_IsolatesRoomEvents(t *testing.T) {
	gameService := service.NewGameService(infrastructure.NewMemoryRoomRepository(), game.NewPlayerManager())
	for _, playerID := range []string{"player-a", "player-b", "player-c"} {
		if _, err := gameService.AddPlayer(playerID); err != nil {
			t.Fatalf("AddPlayer(%s) error = %v", playerID, err)
		}
	}
	roomA, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "player-a")
	if err != nil {
		t.Fatalf("CreateRoomWithContext(room A) error = %v", err)
	}
	if _, err := gameService.JoinRoomWithContext(context.Background(), roomA.Room.RoomID, "player-b", "tictactoe"); err != nil {
		t.Fatalf("JoinRoomWithContext(room A) error = %v", err)
	}
	roomB, err := gameService.CreateRoomWithContext(context.Background(), "chess", "player-a")
	if err != nil {
		t.Fatalf("CreateRoomWithContext(room B) error = %v", err)
	}
	if _, err := gameService.JoinRoomWithContext(context.Background(), roomB.Room.RoomID, "player-c", "chess"); err != nil {
		t.Fatalf("JoinRoomWithContext(room B) error = %v", err)
	}

	clients := NewClientRegistry()
	roomAPlayerA := newBufferedTestClient("player-a")
	roomAPlayerA.RoomID = roomA.Room.RoomID
	roomAPlayerB := newBufferedTestClient("player-b")
	roomAPlayerB.RoomID = roomA.Room.RoomID
	roomBPlayerA := newBufferedTestClient("player-a")
	roomBPlayerA.RoomID = roomB.Room.RoomID
	roomBPlayerC := newBufferedTestClient("player-c")
	roomBPlayerC.RoomID = roomB.Room.RoomID
	for _, client := range []*Client{roomAPlayerA, roomAPlayerB, roomBPlayerA, roomBPlayerC} {
		clients.clients[newClientKey(client.RoomID, client.PlayerID)] = client
	}

	snapshotA, err := gameService.RoomSnapshot(roomA.Room.RoomID)
	if err != nil {
		t.Fatalf("RoomSnapshot(room A) error = %v", err)
	}
	NotifySnapshotToClients(clients, snapshotA, Event{Type: EventGameUpdate, Payload: json.RawMessage(`{}`)})
	readTestEvent(t, roomAPlayerA)
	readTestEvent(t, roomAPlayerB)
	assertNoTestEvent(t, roomBPlayerA)
	assertNoTestEvent(t, roomBPlayerC)

	snapshotB, err := gameService.RoomSnapshot(roomB.Room.RoomID)
	if err != nil {
		t.Fatalf("RoomSnapshot(room B) error = %v", err)
	}
	NotifySnapshotToClients(clients, snapshotB, Event{Type: EventGameUpdate, Payload: json.RawMessage(`{}`)})
	readTestEvent(t, roomBPlayerA)
	readTestEvent(t, roomBPlayerC)
	assertNoTestEvent(t, roomAPlayerA)
	assertNoTestEvent(t, roomAPlayerB)
}

func TestConnectionEventType(t *testing.T) {
	tests := []struct {
		name                   string
		wasConnected           bool
		wasDisconnected        bool
		isReconnectReplacement bool
		want                   string
	}{
		{name: "duplicate active connection", wasConnected: true, wasDisconnected: false, want: ""},
		{name: "fast reconnect replacement", wasConnected: true, wasDisconnected: false, isReconnectReplacement: true, want: EventPlayerReconnected},
		{name: "reconnect after temporary disconnect", wasConnected: false, wasDisconnected: true, want: EventPlayerReconnected},
		{name: "new websocket join", wasConnected: false, wasDisconnected: false, want: EventPlayerJoined},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := connectionEventType(tt.wasConnected, tt.wasDisconnected, tt.isReconnectReplacement); got != tt.want {
				t.Fatalf("connectionEventType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func newBufferedTestClient(playerID string) *Client {
	return &Client{
		PlayerID: playerID,
		Send:     make(chan []byte, 8),
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

func dialTestWebSocketWithOrigin(t *testing.T, server *httptest.Server, query string, origin string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	if query != "" {
		wsURL += "?" + query
	}
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{origin}})
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

func assertNoTestEvent(t *testing.T, client *Client) {
	t.Helper()

	select {
	case message := <-client.Send:
		t.Fatalf("client received unexpected event: %s", message)
	default:
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

func TestCrossRoomIsolation_ChatAndGameActions(t *testing.T) {
	repo := infrastructure.NewMemoryRoomRepository()
	gameService := service.NewGameService(repo, game.NewPlayerManager())

	for _, p := range []string{"p1", "p2", "p3", "p4"} {
		if _, err := gameService.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s) error = %v", p, err)
		}
	}

	roomA, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoom roomA error = %v", err)
	}
	if _, err := gameService.JoinRoomWithContext(context.Background(), roomA.Room.RoomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom roomA error = %v", err)
	}

	roomB, err := gameService.CreateRoomWithContext(context.Background(), "tictactoe", "p3")
	if err != nil {
		t.Fatalf("CreateRoom roomB error = %v", err)
	}
	if _, err := gameService.JoinRoomWithContext(context.Background(), roomB.Room.RoomID, "p4", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom roomB error = %v", err)
	}

	clients := NewClientRegistry()
	c1 := newBufferedTestClient("p1")
	c1.RoomID = roomA.Room.RoomID
	c2 := newBufferedTestClient("p2")
	c2.RoomID = roomA.Room.RoomID
	c3 := newBufferedTestClient("p3")
	c3.RoomID = roomB.Room.RoomID
	c4 := newBufferedTestClient("p4")
	c4.RoomID = roomB.Room.RoomID

	clients.clients[newClientKey(c1.RoomID, c1.PlayerID)] = c1
	clients.clients[newClientKey(c2.RoomID, c2.PlayerID)] = c2
	clients.clients[newClientKey(c3.RoomID, c3.PlayerID)] = c3
	clients.clients[newClientKey(c4.RoomID, c4.PlayerID)] = c4

	// 1. Test Chat Isolation
	chatMsg := WebSocketMessage{
		Type:    "CHAT_SEND",
		Payload: json.RawMessage(`{"message":"Secret from Room A"}`),
	}
	p1Snapshot, _ := gameService.GetPlayerInRoom(roomA.Room.RoomID, "p1")
	processChatSend(context.Background(), p1Snapshot, c1, clients, gameService, roomA.Room.RoomID, chatMsg)

	// Room A clients receive chat event
	e1 := readTestEvent(t, c1)
	if e1.Type != EventChatMessage {
		t.Fatalf("c1 event type = %q, want %q", e1.Type, EventChatMessage)
	}
	e2 := readTestEvent(t, c2)
	if e2.Type != EventChatMessage {
		t.Fatalf("c2 event type = %q, want %q", e2.Type, EventChatMessage)
	}
	// Room B clients must receive nothing
	assertNoTestEvent(t, c3)
	assertNoTestEvent(t, c4)

	// 2. Test Game Move Isolation
	moveMsg := WebSocketMessage{
		Type:    "TICTACTOE_MOVE",
		Payload: json.RawMessage(`{"player_id":"p1","room_id":"` + roomA.Room.RoomID + `","row":0,"col":0}`),
	}
	processTicTacToeMove(context.Background(), p1Snapshot, c1, clients, gameService, roomA.Room.RoomID, moveMsg)

	// Room A clients receive game update event
	ge1 := readTestEvent(t, c1)
	if ge1.Type != EventGameUpdate {
		t.Fatalf("c1 event type = %q, want %q", ge1.Type, EventGameUpdate)
	}
	ge2 := readTestEvent(t, c2)
	if ge2.Type != EventGameUpdate {
		t.Fatalf("c2 event type = %q, want %q", ge2.Type, EventGameUpdate)
	}
	// Room B clients must receive nothing
	assertNoTestEvent(t, c3)
	assertNoTestEvent(t, c4)
}

