package api

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func newBufferedTestClient(playerID string) *Client {
	return &Client{
		PlayerID: playerID,
		Send:     make(chan []byte, 1),
		done:     make(chan struct{}),
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
