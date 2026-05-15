package game

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tsaqiffatih/mini-game/tictactoe"
)

func newTicTacToeRoomForTest(t *testing.T) *Room {
	t.Helper()

	room, err := NewRoom("room-1", "tictactoe")
	if err != nil {
		t.Fatalf("NewRoom() error = %v", err)
	}

	return room
}

func addPlayerToRoomForTest(t *testing.T, room *Room, playerID string) *JoinRoomResponse {
	t.Helper()

	res, err := room.AddPlayer(PlayerSnapshot{ID: playerID})
	if err != nil {
		t.Fatalf("AddPlayer(%q) error = %v", playerID, err)
	}

	return res
}

func TestRoom_AddPlayer_FirstTicTacToePlayer_AssignsX(t *testing.T) {
	room := newTicTacToeRoomForTest(t)

	res := addPlayerToRoomForTest(t, room, "p1")
	snapshot := room.Snapshot()

	if res.PlayerID != "p1" {
		t.Fatalf("PlayerID = %q, want %q", res.PlayerID, "p1")
	}
	if res.PlayerMark != "X" {
		t.Fatalf("PlayerMark = %q, want %q", res.PlayerMark, "X")
	}
	if snapshot.IsActive {
		t.Fatalf("room active = true, want false")
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("players len = %d, want 1", len(snapshot.Players))
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Status != tictactoe.StatusWaiting {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusWaiting)
	}
}

func TestRoom_AddPlayer_SecondTicTacToePlayer_AssignsOAndActivates(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")

	res := addPlayerToRoomForTest(t, room, "p2")
	snapshot := room.Snapshot()

	if res.PlayerMark != "O" {
		t.Fatalf("PlayerMark = %q, want %q", res.PlayerMark, "O")
	}
	if !snapshot.IsActive {
		t.Fatalf("room active = false, want true")
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Status != tictactoe.StatusActive {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusActive)
	}
}

func TestRoom_AddPlayer_ThirdPlayer_ReturnsRoomFull(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	_, err := room.AddPlayer(PlayerSnapshot{ID: "p3"})
	if err == nil {
		t.Fatalf("AddPlayer third player error = nil, want room full error")
	}
	if !strings.Contains(err.Error(), "room is full") {
		t.Fatalf("error = %q, want room full", err)
	}

	snapshot := room.Snapshot()
	if len(snapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(snapshot.Players))
	}
}

func TestRoom_HandleTicTacToeMove_ValidMove_UpdatesSnapshot(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	result, err := room.HandleTicTacToeMove("p1", 0, 0)
	if err != nil {
		t.Fatalf("HandleTicTacToeMove() error = %v", err)
	}

	if result.State.Board[0][0] != "X" {
		t.Fatalf("board[0][0] = %q, want X", result.State.Board[0][0])
	}
	if result.State.Turn != "O" {
		t.Fatalf("turn = %q, want O", result.State.Turn)
	}
	if result.GameEnded {
		t.Fatalf("GameEnded = true, want false")
	}
}

func TestRoom_HandleTicTacToeMove_AIEnabled_AppliesAIMove(t *testing.T) {
	room, err := NewRoomWithAILevel("room-ai", "tictactoe", 10)
	if err != nil {
		t.Fatalf("NewRoomWithAILevel() error = %v", err)
	}
	room.SetAIMoveDelay(10 * time.Millisecond)

	addPlayerToRoomForTest(t, room, "p1")

	result, err := room.HandleTicTacToeMove("p1", 0, 0)
	if err != nil {
		t.Fatalf("HandleTicTacToeMove() error = %v", err)
	}

	if result.State.Board[0][0] != "X" {
		t.Fatalf("human move board[0][0] = %q, want X", result.State.Board[0][0])
	}
	if filled := countFilledCells(result.State.Board); filled != 1 {
		t.Fatalf("immediate filled cells = %d, want human move only", filled)
	}
	if result.State.Turn != "O" {
		t.Fatalf("immediate turn = %q, want AI turn", result.State.Turn)
	}

	waitForFilledCells(t, room, 2, 250*time.Millisecond)
	snapshot := room.Snapshot()
	if snapshot.TicTacToe.Turn != "X" && snapshot.TicTacToe.Status == tictactoe.StatusActive {
		t.Fatalf("turn = %q, want X after AI response", snapshot.TicTacToe.Turn)
	}
}

func TestRoom_HandleChessMove_AIEnabled_AppliesStockfishMove(t *testing.T) {
	if _, err := os.Stat(defaultStockfishPath); err != nil {
		t.Skipf("stockfish binary unavailable: %v", err)
	}

	room, err := NewRoomWithAILevel("chess-ai", "chess", 1)
	if err != nil {
		t.Skipf("stockfish unavailable: %v", err)
	}
	defer room.Close()
	room.SetAIMoveDelay(10 * time.Millisecond)

	res := addPlayerToRoomForTest(t, room, "p1")
	if res.PlayerMark != "white" {
		t.Fatalf("human player mark = %q, want white", res.PlayerMark)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := room.HandleChessMoveWithContext(ctx, "p1", "e2", "e4", ""); err != nil {
		t.Fatalf("HandleChessMoveWithContext() error = %v", err)
	}

	snapshot := room.Snapshot()
	if snapshot.Chess == nil {
		t.Fatalf("Chess snapshot is nil")
	}
	if moves := len(snapshot.Chess.PGNMoves); moves != 1 {
		t.Fatalf("immediate PGN moves len = %d, want human move only", moves)
	}

	waitForChessMoves(t, room, 2, time.Second)
	snapshot = room.Snapshot()
	if moves := len(snapshot.Chess.PGNMoves); moves != 2 {
		t.Fatalf("PGN moves len = %d, want human + AI moves", moves)
	}
}

func TestRoom_HandleChessUndo_AIEnabled_RollsBackHumanAndAIMove(t *testing.T) {
	if _, err := os.Stat(defaultStockfishPath); err != nil {
		t.Skipf("stockfish binary unavailable: %v", err)
	}

	room, err := NewRoomWithAILevel("chess-ai-undo", "chess", 1)
	if err != nil {
		t.Skipf("stockfish unavailable: %v", err)
	}
	defer room.Close()
	room.SetAIMoveDelay(10 * time.Millisecond)

	addPlayerToRoomForTest(t, room, "p1")

	before := room.Snapshot()
	if _, err := room.HandleChessMoveWithContext(context.Background(), "p1", "e2", "e4", ""); err != nil {
		t.Fatalf("HandleChessMoveWithContext() error = %v", err)
	}
	waitForChessMoves(t, room, 2, time.Second)

	if err := room.HandleChessUndo("p1"); err != nil {
		t.Fatalf("HandleChessUndo() error = %v", err)
	}

	after := room.Snapshot()
	if after.Chess == nil {
		t.Fatalf("Chess snapshot is nil")
	}
	if after.Chess.FEN != before.Chess.FEN {
		t.Fatalf("FEN after undo = %q, want initial %q", after.Chess.FEN, before.Chess.FEN)
	}
	if moves := len(after.Chess.PGNMoves); moves != 0 {
		t.Fatalf("PGN moves len after undo = %d, want 0", moves)
	}
	if after.StateVersion <= before.StateVersion {
		t.Fatalf("state version after undo = %d, want > %d", after.StateVersion, before.StateVersion)
	}
}

func TestRoom_HandleTicTacToeMove_NotPlayersTurn_ReturnsError(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	_, err := room.HandleTicTacToeMove("p2", 0, 0)
	if err == nil {
		t.Fatalf("HandleTicTacToeMove() error = nil, want not your turn")
	}
	if !strings.Contains(err.Error(), "not your turn") {
		t.Fatalf("error = %q, want not your turn", err)
	}

	snapshot := room.Snapshot()
	if snapshot.TicTacToe.Board[0][0] != "" {
		t.Fatalf("board[0][0] = %q, want empty", snapshot.TicTacToe.Board[0][0])
	}
}

func TestRoom_HandleChessMove_RejectedMoveDoesNotIncrementStateVersion(t *testing.T) {
	room, err := NewRoom("chess-invalid", "chess")
	if err != nil {
		t.Fatalf("NewRoom() error = %v", err)
	}
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	before := room.Snapshot().StateVersion
	_, err = room.HandleChessMove("p2", "e7", "e5", "")
	if err == nil {
		t.Fatalf("HandleChessMove() error = nil, want not your turn")
	}

	after := room.Snapshot().StateVersion
	if after != before {
		t.Fatalf("state version after rejected move = %d, want %d", after, before)
	}
}

func TestRoom_HandleTicTacToeMove_OutOfBounds_ReturnsError(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	_, err := room.HandleTicTacToeMove("p1", -1, 0)
	if err == nil {
		t.Fatalf("HandleTicTacToeMove() error = nil, want invalid position")
	}
	if !strings.Contains(err.Error(), "invalid position") {
		t.Fatalf("error = %q, want invalid position", err)
	}

	snapshot := room.Snapshot()
	if snapshot.TicTacToe.Board != ([3][3]string{}) {
		t.Fatalf("board = %+v, want empty board", snapshot.TicTacToe.Board)
	}
}

func TestRoom_HandleTicTacToeMove_OccupiedCell_ReturnsError(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	if _, err := room.HandleTicTacToeMove("p1", 0, 0); err != nil {
		t.Fatalf("HandleTicTacToeMove() error = %v", err)
	}

	_, err := room.HandleTicTacToeMove("p2", 0, 0)
	if err == nil {
		t.Fatalf("HandleTicTacToeMove() error = nil, want cell already occupied")
	}
	if !strings.Contains(err.Error(), "cell already occupied") {
		t.Fatalf("error = %q, want cell already occupied", err)
	}

	snapshot := room.Snapshot()
	if snapshot.TicTacToe.Board[0][0] != "X" {
		t.Fatalf("board[0][0] = %q, want X", snapshot.TicTacToe.Board[0][0])
	}
	if snapshot.TicTacToe.Turn != "O" {
		t.Fatalf("turn = %q, want O", snapshot.TicTacToe.Turn)
	}
}

func TestRoom_HandleTicTacToeMove_WinningMove_EndsGame(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

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
		if _, err := room.HandleTicTacToeMove(move.playerID, move.row, move.col); err != nil {
			t.Fatalf("HandleTicTacToeMove(%q, %d, %d) error = %v", move.playerID, move.row, move.col, err)
		}
	}

	result, err := room.HandleTicTacToeMove("p1", 0, 2)
	if err != nil {
		t.Fatalf("winning move error = %v", err)
	}

	if !result.GameEnded {
		t.Fatalf("GameEnded = false, want true")
	}
	if result.State.Winner != "X" {
		t.Fatalf("winner = %q, want X", result.State.Winner)
	}
	if result.State.Status != tictactoe.StatusEnded {
		t.Fatalf("status = %q, want %q", result.State.Status, tictactoe.StatusEnded)
	}
}

func TestRoom_HandleTicTacToeMove_DelayedResetStatesAreObservable(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	room.SetResetDelays(20*time.Millisecond, 20*time.Millisecond)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	moves := []struct {
		playerID string
		row      int
		col      int
	}{
		{"p1", 0, 0},
		{"p2", 1, 0},
		{"p1", 0, 1},
		{"p2", 1, 1},
		{"p1", 0, 2},
	}

	for _, move := range moves {
		if _, err := room.HandleTicTacToeMove(move.playerID, move.row, move.col); err != nil {
			t.Fatalf("HandleTicTacToeMove(%q, %d, %d) error = %v", move.playerID, move.row, move.col, err)
		}
	}

	snapshot := room.Snapshot()
	if snapshot.RoomState != RoomStateFinished {
		t.Fatalf("room state = %q, want %q", snapshot.RoomState, RoomStateFinished)
	}
	if snapshot.TicTacToe.Status != tictactoe.StatusEnded {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusEnded)
	}
	if _, err := room.HandleTicTacToeMove("p2", 2, 2); err == nil {
		t.Fatalf("HandleTicTacToeMove during FINISHED error = nil, want error")
	}

	waitForRoomState(t, room, RoomStateResetting, 250*time.Millisecond)
	waitForRoomState(t, room, RoomStatePlaying, 250*time.Millisecond)

	snapshot = room.Snapshot()
	if snapshot.TicTacToe.Status != tictactoe.StatusActive {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusActive)
	}
	if filled := countFilledCells(snapshot.TicTacToe.Board); filled != 0 {
		t.Fatalf("filled cells = %d, want 0", filled)
	}
}

func TestRoom_HandleTicTacToeMove_Draw_EndsGame(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	moves := []struct {
		playerID string
		row      int
		col      int
	}{
		{"p1", 0, 0},
		{"p2", 0, 1},
		{"p1", 0, 2},
		{"p2", 1, 1},
		{"p1", 1, 0},
		{"p2", 1, 2},
		{"p1", 2, 1},
		{"p2", 2, 0},
	}

	for _, move := range moves {
		if _, err := room.HandleTicTacToeMove(move.playerID, move.row, move.col); err != nil {
			t.Fatalf("HandleTicTacToeMove(%q, %d, %d) error = %v", move.playerID, move.row, move.col, err)
		}
	}

	result, err := room.HandleTicTacToeMove("p1", 2, 2)
	if err != nil {
		t.Fatalf("draw move error = %v", err)
	}

	if !result.GameEnded {
		t.Fatalf("GameEnded = false, want true")
	}
	if result.State.Winner != "Draw" {
		t.Fatalf("winner = %q, want Draw", result.State.Winner)
	}
	if result.State.Status != tictactoe.StatusEnded {
		t.Fatalf("status = %q, want %q", result.State.Status, tictactoe.StatusEnded)
	}
}

func TestRoom_HandlePlayerDisconnected_TwoToOneTicTacToe_ResetsRoom(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	if _, err := room.HandleTicTacToeMove("p1", 0, 0); err != nil {
		t.Fatalf("HandleTicTacToeMove() error = %v", err)
	}

	shouldRemoveRoom := room.HandlePlayerDisconnected("p2")
	if shouldRemoveRoom {
		t.Fatalf("HandlePlayerDisconnected() = true, want false")
	}

	snapshot := room.Snapshot()
	if snapshot.IsActive {
		t.Fatalf("room active = true, want false")
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("players len = %d, want 1", len(snapshot.Players))
	}
	if snapshot.Players[0].ID != "p1" {
		t.Fatalf("remaining player = %q, want p1", snapshot.Players[0].ID)
	}
	if snapshot.Players[0].Mark != "X" {
		t.Fatalf("remaining player mark = %q, want X", snapshot.Players[0].Mark)
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Status != tictactoe.StatusWaiting {
		t.Fatalf("status = %q, want %q", snapshot.TicTacToe.Status, tictactoe.StatusWaiting)
	}
	if snapshot.TicTacToe.Board[0][0] != "" {
		t.Fatalf("board[0][0] = %q, want empty", snapshot.TicTacToe.Board[0][0])
	}
}

func TestRoom_Snapshot_ReturnsCopiedPlayerData(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")

	snapshot := room.Snapshot()
	if len(snapshot.Players) != 1 {
		t.Fatalf("players len = %d, want 1", len(snapshot.Players))
	}

	snapshot.Players[0].Mark = "mutated"

	nextSnapshot := room.Snapshot()
	if nextSnapshot.Players[0].Mark != "X" {
		t.Fatalf("stored player mark = %q, want X", nextSnapshot.Players[0].Mark)
	}
}

func TestRoom_Concurrent_AddPlayer_SameRoom(t *testing.T) {
	room := newTicTacToeRoomForTest(t)

	const goroutines = 20
	start := make(chan struct{})
	results := make(chan error, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start

			_, err := room.AddPlayer(PlayerSnapshot{ID: "p" + string(rune('a'+i))})
			results <- err
		}(i)
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}

	snapshot := room.Snapshot()
	if successes != 2 {
		t.Fatalf("successful joins = %d, want 2", successes)
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(snapshot.Players))
	}
	if !snapshot.IsActive {
		t.Fatalf("room active = false, want true")
	}

	seenPlayers := map[string]bool{}
	marks := map[string]bool{}
	for _, player := range snapshot.Players {
		if player.ID == "" {
			t.Fatalf("player ID is empty")
		}
		if seenPlayers[player.ID] {
			t.Fatalf("duplicate player ID %q in snapshot %+v", player.ID, snapshot.Players)
		}
		seenPlayers[player.ID] = true
		if player.Mark != "X" && player.Mark != "O" {
			t.Fatalf("player mark = %q, want X or O", player.Mark)
		}
		marks[player.Mark] = true
	}
	if !marks["X"] || !marks["O"] {
		t.Fatalf("player marks = %+v, want X and O", marks)
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Turn != "X" && snapshot.TicTacToe.Turn != "O" {
		t.Fatalf("turn = %q, want X or O", snapshot.TicTacToe.Turn)
	}
	if !isAllowedTicTacToeStatus(snapshot.TicTacToe.Status) {
		t.Fatalf("status = %q, want allowed TicTacToe status", snapshot.TicTacToe.Status)
	}
	assertValidTicTacToeBoard(t, snapshot.TicTacToe.Board)
}

func TestRoom_Concurrent_HandleTicTacToeMove_SameCell(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	const goroutines = 2
	start := make(chan struct{})
	results := make(chan error, goroutines)

	var wg sync.WaitGroup
	for _, playerID := range []string{"p1", "p2"} {
		wg.Add(1)
		go func(playerID string) {
			defer wg.Done()
			<-start

			_, err := room.HandleTicTacToeMove(playerID, 0, 0)
			results <- err
		}(playerID)
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}

	snapshot := room.Snapshot()
	if successes != 1 {
		t.Fatalf("successful moves = %d, want 1", successes)
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(snapshot.Players))
	}
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if snapshot.TicTacToe.Board[0][0] != "X" && snapshot.TicTacToe.Board[0][0] != "O" {
		t.Fatalf("board[0][0] = %q, want filled by X or O", snapshot.TicTacToe.Board[0][0])
	}
	if snapshot.TicTacToe.Turn != "X" && snapshot.TicTacToe.Turn != "O" {
		t.Fatalf("turn = %q, want X or O", snapshot.TicTacToe.Turn)
	}
	if !isAllowedTicTacToeStatus(snapshot.TicTacToe.Status) {
		t.Fatalf("status = %q, want allowed TicTacToe status", snapshot.TicTacToe.Status)
	}
	assertUniquePlayers(t, snapshot.Players)
	assertValidTicTacToeBoard(t, snapshot.TicTacToe.Board)
}

func TestRoom_Concurrent_HandleTicTacToeMove_AndSnapshot(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	start := make(chan struct{})
	moveErrs := make(chan error, 4)
	snapshots := make(chan RoomSnapshot, 100)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start

		moves := []struct {
			playerID string
			row      int
			col      int
		}{
			{"p1", 0, 0},
			{"p2", 1, 1},
			{"p1", 0, 1},
			{"p2", 2, 2},
		}

		for _, move := range moves {
			_, err := room.HandleTicTacToeMove(move.playerID, move.row, move.col)
			moveErrs <- err
		}
	}()

	const snapshotWorkers = 10
	const snapshotsPerWorker = 10
	for i := 0; i < snapshotWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			for j := 0; j < snapshotsPerWorker; j++ {
				snapshots <- room.Snapshot()
			}
		}()
	}

	close(start)
	wg.Wait()
	close(moveErrs)
	close(snapshots)

	for err := range moveErrs {
		if err != nil {
			t.Fatalf("HandleTicTacToeMove() error = %v", err)
		}
	}

	for snapshot := range snapshots {
		if snapshot.TicTacToe == nil {
			t.Fatalf("TicTacToe snapshot is nil")
		}
		if len(snapshot.Players) != 2 {
			t.Fatalf("players len = %d, want 2", len(snapshot.Players))
		}
		if snapshot.TicTacToe.Turn != "X" && snapshot.TicTacToe.Turn != "O" {
			t.Fatalf("turn = %q, want X or O", snapshot.TicTacToe.Turn)
		}
		if !isAllowedTicTacToeStatus(snapshot.TicTacToe.Status) {
			t.Fatalf("status = %q, want allowed TicTacToe status", snapshot.TicTacToe.Status)
		}
		assertUniquePlayers(t, snapshot.Players)
		assertValidTicTacToeBoard(t, snapshot.TicTacToe.Board)
	}

	finalSnapshot := room.Snapshot()
	if finalSnapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	if len(finalSnapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(finalSnapshot.Players))
	}
	if finalSnapshot.TicTacToe.Turn != "X" && finalSnapshot.TicTacToe.Turn != "O" {
		t.Fatalf("final turn = %q, want X or O", finalSnapshot.TicTacToe.Turn)
	}
	if !isAllowedTicTacToeStatus(finalSnapshot.TicTacToe.Status) {
		t.Fatalf("final status = %q, want allowed TicTacToe status", finalSnapshot.TicTacToe.Status)
	}
	if countFilledCells(finalSnapshot.TicTacToe.Board) != 4 {
		t.Fatalf("filled cells = %d, want 4", countFilledCells(finalSnapshot.TicTacToe.Board))
	}
	assertUniquePlayers(t, finalSnapshot.Players)
	assertValidTicTacToeBoard(t, finalSnapshot.TicTacToe.Board)
}

func TestRoom_Stress_SingleRoom_HighContention(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	positions := [][2]int{
		{0, 0}, {0, 1}, {0, 2},
		{1, 0}, {1, 1}, {1, 2},
		{2, 0}, {2, 1}, {2, 2},
	}

	const goroutines = 100
	start := make(chan struct{})
	errs := make(chan error, goroutines)
	successfulMoves := make(chan struct{}, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					errs <- fmt.Errorf("panic: %v", recovered)
				}
			}()

			<-start

			if i%3 == 0 {
				errs <- validateRoomSnapshotInvariant(room.Snapshot(), 2)
				return
			}

			position := positions[i%len(positions)]
			playerID := "p1"
			if i%2 == 1 {
				playerID = "p2"
			}

			_, err := room.HandleTicTacToeMove(playerID, position[0], position[1])
			if err == nil {
				successfulMoves <- struct{}{}
			}
			errs <- nil
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)
	close(successfulMoves)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if successes := len(successfulMoves); successes > 9 {
		t.Fatalf("successful moves = %d, want at most 9", successes)
	}

	finalSnapshot := room.Snapshot()
	if err := validateRoomSnapshotInvariant(finalSnapshot, 2); err != nil {
		t.Fatal(err)
	}
	if filled := countFilledCells(finalSnapshot.TicTacToe.Board); filled > 9 {
		t.Fatalf("filled cells = %d, want at most 9", filled)
	}
}

func TestRoom_Stress_SnapshotConsistency(t *testing.T) {
	room := newTicTacToeRoomForTest(t)
	addPlayerToRoomForTest(t, room, "p1")
	addPlayerToRoomForTest(t, room, "p2")

	moves := []struct {
		playerID string
		row      int
		col      int
	}{
		{"p1", 0, 0},
		{"p2", 1, 1},
		{"p1", 0, 1},
		{"p2", 2, 2},
		{"p1", 0, 2},
	}

	const snapshotWorkers = 25
	const snapshotsPerWorker = 20
	start := make(chan struct{})
	errs := make(chan error, snapshotWorkers*snapshotsPerWorker+len(moves))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if recovered := recover(); recovered != nil {
				errs <- fmt.Errorf("panic: %v", recovered)
			}
		}()

		<-start
		for _, move := range moves {
			_, err := room.HandleTicTacToeMove(move.playerID, move.row, move.col)
			if err != nil {
				errs <- fmt.Errorf("HandleTicTacToeMove(%q, %d, %d): %w", move.playerID, move.row, move.col, err)
				return
			}
			errs <- validateRoomSnapshotInvariant(room.Snapshot(), 2)
		}
	}()

	for i := 0; i < snapshotWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					errs <- fmt.Errorf("panic: %v", recovered)
				}
			}()

			<-start
			for j := 0; j < snapshotsPerWorker; j++ {
				errs <- validateRoomSnapshotInvariant(room.Snapshot(), 2)
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	if err := validateRoomSnapshotInvariant(room.Snapshot(), 2); err != nil {
		t.Fatal(err)
	}
}

func assertUniquePlayers(t *testing.T, players []PlayerSnapshot) {
	t.Helper()

	seen := map[string]bool{}
	for _, player := range players {
		if player.ID == "" {
			t.Fatalf("player ID is empty")
		}
		if seen[player.ID] {
			t.Fatalf("duplicate player ID %q in snapshot %+v", player.ID, players)
		}
		seen[player.ID] = true
		if player.Mark != "X" && player.Mark != "O" {
			t.Fatalf("player mark = %q, want X or O", player.Mark)
		}
	}
}

func assertValidTicTacToeBoard(t *testing.T, board [3][3]string) {
	t.Helper()

	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := board[row][col]
			if cell != "" && cell != "X" && cell != "O" {
				t.Fatalf("board[%d][%d] = %q, want empty, X, or O", row, col, cell)
			}
		}
	}
}

func isAllowedTicTacToeStatus(status tictactoe.GameStatus) bool {
	return status == tictactoe.StatusWaiting ||
		status == tictactoe.StatusActive ||
		status == tictactoe.StatusEnded
}

func countFilledCells(board [3][3]string) int {
	count := 0
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if board[row][col] != "" {
				count++
			}
		}
	}
	return count
}

func waitForRoomState(t *testing.T, room *Room, want RoomState, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if snapshot := room.Snapshot(); snapshot.RoomState == want {
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Fatalf("room state = %q, want %q", room.Snapshot().RoomState, want)
}

func waitForFilledCells(t *testing.T, room *Room, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snapshot := room.Snapshot()
		if snapshot.TicTacToe != nil && countFilledCells(snapshot.TicTacToe.Board) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}

	snapshot := room.Snapshot()
	if snapshot.TicTacToe == nil {
		t.Fatalf("TicTacToe snapshot is nil")
	}
	t.Fatalf("filled cells = %d, want %d", countFilledCells(snapshot.TicTacToe.Board), want)
}

func waitForChessMoves(t *testing.T, room *Room, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		snapshot := room.Snapshot()
		if snapshot.Chess != nil && len(snapshot.Chess.PGNMoves) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}

	snapshot := room.Snapshot()
	if snapshot.Chess == nil {
		t.Fatalf("Chess snapshot is nil")
	}
	t.Fatalf("PGN moves len = %d, want %d", len(snapshot.Chess.PGNMoves), want)
}

func validateRoomSnapshotInvariant(snapshot RoomSnapshot, expectedPlayers int) error {
	if snapshot.TicTacToe == nil {
		return fmt.Errorf("TicTacToe snapshot is nil")
	}
	if len(snapshot.Players) != expectedPlayers {
		return fmt.Errorf("players len = %d, want %d", len(snapshot.Players), expectedPlayers)
	}
	if snapshot.TicTacToe.Turn != "X" && snapshot.TicTacToe.Turn != "O" {
		return fmt.Errorf("turn = %q, want X or O", snapshot.TicTacToe.Turn)
	}
	if !isAllowedTicTacToeStatus(snapshot.TicTacToe.Status) {
		return fmt.Errorf("status = %q, want allowed TicTacToe status", snapshot.TicTacToe.Status)
	}

	seenPlayers := map[string]bool{}
	for _, player := range snapshot.Players {
		if player.ID == "" {
			return fmt.Errorf("player ID is empty")
		}
		if seenPlayers[player.ID] {
			return fmt.Errorf("duplicate player ID %q in snapshot %+v", player.ID, snapshot.Players)
		}
		seenPlayers[player.ID] = true
		if player.Mark != "X" && player.Mark != "O" {
			return fmt.Errorf("player mark = %q, want X or O", player.Mark)
		}
	}

	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			cell := snapshot.TicTacToe.Board[row][col]
			if cell != "" && cell != "X" && cell != "O" {
				return fmt.Errorf("board[%d][%d] = %q, want empty, X, or O", row, col, cell)
			}
		}
	}

	return nil
}
