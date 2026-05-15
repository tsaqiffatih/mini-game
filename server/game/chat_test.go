package game

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRoom_AddChatMessage_TrimsAndStoresMessage(t *testing.T) {
	room := newChatTestRoom(t)

	message, err := room.AddChatMessage("p1", "  hello world  ")
	if err != nil {
		t.Fatalf("AddChatMessage() error = %v", err)
	}

	if message.RoomID != room.RoomID {
		t.Fatalf("RoomID = %q, want %q", message.RoomID, room.RoomID)
	}
	if message.PlayerID != "p1" {
		t.Fatalf("PlayerID = %q, want p1", message.PlayerID)
	}
	if message.PlayerMark != "X" {
		t.Fatalf("PlayerMark = %q, want X", message.PlayerMark)
	}
	if message.Message != "hello world" {
		t.Fatalf("Message = %q, want trimmed message", message.Message)
	}
	if message.ID == "" {
		t.Fatalf("ID is empty")
	}
	if message.CreatedAt.IsZero() {
		t.Fatalf("CreatedAt is zero")
	}
}

func TestRoom_AddChatMessage_ValidatesMessage(t *testing.T) {
	room := newChatTestRoom(t)

	if _, err := room.AddChatMessage("p1", "   "); !errors.Is(err, ErrChatMessageEmpty) {
		t.Fatalf("empty message error = %v, want %v", err, ErrChatMessageEmpty)
	}

	tooLong := strings.Repeat("a", maxChatMessageChars+1)
	if _, err := room.AddChatMessage("p1", tooLong); !errors.Is(err, ErrChatMessageTooLong) {
		t.Fatalf("too long message error = %v, want %v", err, ErrChatMessageTooLong)
	}

	if _, err := room.AddChatMessage("missing", "hello"); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("missing player error = %v, want %v", err, ErrPlayerNotFound)
	}
}

func TestRoom_ChatHistory_IsBoundedAndCopied(t *testing.T) {
	room := newChatTestRoom(t)

	for i := 0; i < maxChatMessages+5; i++ {
		if _, err := room.AddChatMessage("p1", fmt.Sprintf("message-%02d", i)); err != nil {
			t.Fatalf("AddChatMessage(%d) error = %v", i, err)
		}
	}

	history := room.ChatHistory()
	if len(history) != maxChatMessages {
		t.Fatalf("history len = %d, want %d", len(history), maxChatMessages)
	}
	if history[0].Message != "message-05" {
		t.Fatalf("oldest retained message = %q, want message-05", history[0].Message)
	}

	history[0].Message = "mutated"
	nextHistory := room.ChatHistory()
	if nextHistory[0].Message == "mutated" {
		t.Fatalf("ChatHistory exposed mutable internal slice")
	}
}

func newChatTestRoom(t *testing.T) *Room {
	t.Helper()

	room, err := NewRoom("ROOM123", "tictactoe")
	if err != nil {
		t.Fatalf("NewRoom() error = %v", err)
	}
	if _, err := room.AddPlayer(PlayerSnapshot{ID: "p1"}); err != nil {
		t.Fatalf("AddPlayer() error = %v", err)
	}
	return room
}
