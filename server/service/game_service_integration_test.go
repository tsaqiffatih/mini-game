package service

import (
	"context"
	"testing"

	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/infrastructure"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

func newIntegrationGameService() (*GameService, *infrastructure.MemoryRoomRepository) {
	repo := infrastructure.NewMemoryRoomRepository()
	playerManager := game.NewPlayerManager()
	return NewGameService(repo, playerManager), repo
}

func addIntegrationPlayer(t *testing.T, service *GameService, playerID string) {
	t.Helper()

	if _, err := service.AddPlayer(playerID); err != nil {
		t.Fatalf("AddPlayer(%q) error = %v", playerID, err)
	}
}

func createIntegrationTicTacToeRoom(t *testing.T, service *GameService, creatorID string) string {
	t.Helper()

	res, err := service.CreateRoom("tictactoe", creatorID)
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if res.Room.RoomID == "" {
		t.Fatalf("room ID is empty")
	}

	return res.Room.RoomID
}

func TestGameServiceIntegration_FullTicTacToeFlow(t *testing.T) {
	service, _ := newIntegrationGameService()
	addIntegrationPlayer(t, service, "p1")
	addIntegrationPlayer(t, service, "p2")

	roomID := createIntegrationTicTacToeRoom(t, service, "p1")
	if _, err := service.JoinRoom(roomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	moves := []struct {
		playerID string
		row      int
		col      int
	}{
		{"p1", 0, 0},
		{"p2", 1, 0},
		{"p1", 0, 1},
		{"p2", 1, 1},
	}

	for _, move := range moves {
		if _, err := service.HandleTicTacToeMove(roomID, move.playerID, move.row, move.col); err != nil {
			t.Fatalf("HandleTicTacToeMove(%q, %d, %d) error = %v", move.playerID, move.row, move.col, err)
		}
	}

	result, err := service.HandleTicTacToeMove(roomID, "p1", 0, 2)
	if err != nil {
		t.Fatalf("winning move error = %v", err)
	}

	if !result.GameEnded {
		t.Fatalf("GameEnded = false, want true")
	}
	if result.State.Winner != "X" && result.State.Winner != "O" {
		t.Fatalf("winner = %q, want valid player mark", result.State.Winner)
	}
	if result.State.Status != tictactoe.StatusEnded {
		t.Fatalf("status = %q, want %q", result.State.Status, tictactoe.StatusEnded)
	}
}

func TestGameServiceIntegration_DisconnectFlow(t *testing.T) {
	service, repo := newIntegrationGameService()
	addIntegrationPlayer(t, service, "p1")
	addIntegrationPlayer(t, service, "p2")

	roomID := createIntegrationTicTacToeRoom(t, service, "p1")
	if _, err := service.JoinRoom(roomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}
	if _, err := service.HandleTicTacToeMove(roomID, "p1", 0, 0); err != nil {
		t.Fatalf("HandleTicTacToeMove() error = %v", err)
	}

	room, err := repo.GetByID(context.Background(), roomID)
	if err != nil {
		t.Fatalf("repo.GetByID() error = %v", err)
	}

	shouldRemoveRoom := room.HandlePlayerDisconnected("p2")
	if shouldRemoveRoom {
		t.Fatalf("HandlePlayerDisconnected() = true, want false")
	}

	snapshot, err := service.RoomSnapshot(roomID)
	if err != nil {
		t.Fatalf("RoomSnapshot() error = %v", err)
	}
	if snapshot.IsActive {
		t.Fatalf("room active = true, want false")
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("players len = %d, want 1", len(snapshot.Players))
	}
	for _, player := range snapshot.Players {
		if player.ID == "" {
			t.Fatalf("remaining player ID is empty")
		}
		if player.IsAI {
			t.Fatalf("remaining player IsAI = true, want human player")
		}
		if player.Mark != "X" && player.Mark != "O" {
			t.Fatalf("remaining player mark = %q, want valid player mark", player.Mark)
		}
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Status != tictactoe.StatusWaiting {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusWaiting)
	}
	if snapshot.TicTacToe.Board != ([3][3]string{}) {
		t.Fatalf("board = %+v, want empty board", snapshot.TicTacToe.Board)
	}
}

func TestGameServiceIntegration_AIFlow(t *testing.T) {
	service, _ := newIntegrationGameService()
	addIntegrationPlayer(t, service, "p1")

	res, err := service.CreateRoomWithAI("tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoomWithAI() error = %v", err)
	}

	snapshot, err := service.RoomSnapshot(res.Room.RoomID)
	if err != nil {
		t.Fatalf("RoomSnapshot() error = %v", err)
	}

	if !snapshot.IsActive {
		t.Fatalf("room active = false, want true")
	}
	if !snapshot.IsAIEnabled {
		t.Fatalf("AI enabled = false, want true")
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Status != tictactoe.StatusActive {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusActive)
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(snapshot.Players))
	}

	aiPlayers := 0
	humanPlayers := 0
	for _, player := range snapshot.Players {
		if player.IsAI {
			aiPlayers++
			continue
		}
		humanPlayers++
		if player.ID == "" {
			t.Fatalf("human player ID is empty")
		}
		if player.Mark != "X" && player.Mark != "O" {
			t.Fatalf("human player mark = %q, want valid player mark", player.Mark)
		}
	}
	if aiPlayers != 1 {
		t.Fatalf("AI player count = %d, want 1 in snapshot %+v", aiPlayers, snapshot.Players)
	}
	if humanPlayers != 1 {
		t.Fatalf("human player count = %d, want 1 in snapshot %+v", humanPlayers, snapshot.Players)
	}
}
