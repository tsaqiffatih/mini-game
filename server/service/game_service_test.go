package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/tsaqiffatih/mini-game/game"
)

type fakeRoomRepository struct {
	mu      sync.RWMutex
	rooms   map[string]*game.Room
	deleted map[string]bool
}

func newFakeRoomRepository() *fakeRoomRepository {
	return &fakeRoomRepository{
		rooms:   make(map[string]*game.Room),
		deleted: make(map[string]bool),
	}
}

func (r *fakeRoomRepository) GetByID(ctx context.Context, roomID string) (*game.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	room, ok := r.rooms[roomID]
	if !ok {
		return nil, errors.New("room not found")
	}

	return room, nil
}

func (r *fakeRoomRepository) Save(ctx context.Context, room *game.Room) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rooms[room.RoomID]; exists {
		return errors.New("room already exists")
	}

	r.rooms[room.RoomID] = room
	return nil
}

func (r *fakeRoomRepository) Delete(ctx context.Context, roomID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if room, exists := r.rooms[roomID]; exists {
		room.Close()
	}
	delete(r.rooms, roomID)
	r.deleted[roomID] = true
	return nil
}

func (r *fakeRoomRepository) List(ctx context.Context) ([]*game.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	rooms := make([]*game.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}
	return rooms, nil
}

func (r *fakeRoomRepository) wasDeleted(roomID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.deleted[roomID]
}

func newGameServiceForTest() (*GameService, *fakeRoomRepository) {
	repo := newFakeRoomRepository()
	playerManager := game.NewPlayerManager()
	return NewGameService(repo, playerManager), repo
}

func addServicePlayerForTest(t *testing.T, service *GameService, playerID string) {
	t.Helper()

	if _, err := service.AddPlayer(playerID); err != nil {
		t.Fatalf("AddPlayer(%q) error = %v", playerID, err)
	}
}

func createTicTacToeRoomForServiceTest(t *testing.T, service *GameService, playerID string) string {
	t.Helper()

	res, err := service.CreateRoom("tictactoe", playerID)
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}
	if res.Room.RoomID == "" {
		t.Fatalf("created room id is empty")
	}

	return res.Room.RoomID
}

func TestGameService_CreateRoom_TicTacToe_SavesRoomAndJoinsCreator(t *testing.T) {
	service, repo := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")

	res, err := service.CreateRoom("tictactoe", "p1")
	if err != nil {
		t.Fatalf("CreateRoom() error = %v", err)
	}

	if res.PlayerID != "p1" {
		t.Fatalf("PlayerID = %q, want p1", res.PlayerID)
	}
	if res.PlayerMark != "X" {
		t.Fatalf("PlayerMark = %q, want X", res.PlayerMark)
	}
	if res.Room.RoomID == "" {
		t.Fatalf("RoomID is empty")
	}

	room, err := repo.GetByID(context.Background(), res.Room.RoomID)
	if err != nil {
		t.Fatalf("repo.GetByID() error = %v", err)
	}

	snapshot := room.Snapshot()
	if snapshot.GameType != "tictactoe" {
		t.Fatalf("game type = %q, want tictactoe", snapshot.GameType)
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("players len = %d, want 1", len(snapshot.Players))
	}
	if snapshot.Players[0].ID != "p1" || snapshot.Players[0].Mark != "X" {
		t.Fatalf("creator snapshot = %+v, want p1/X", snapshot.Players[0])
	}
}

func TestGameService_JoinRoom_ValidSecondPlayer_JoinsRoom(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	addServicePlayerForTest(t, service, "p2")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")

	res, err := service.JoinRoom(roomID, "p2", "tictactoe")
	if err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	if res.PlayerID != "p2" {
		t.Fatalf("PlayerID = %q, want p2", res.PlayerID)
	}
	if res.PlayerMark != "O" {
		t.Fatalf("PlayerMark = %q, want O", res.PlayerMark)
	}

	snapshot, err := service.RoomSnapshot(roomID)
	if err != nil {
		t.Fatalf("RoomSnapshot() error = %v", err)
	}
	if !snapshot.IsActive {
		t.Fatalf("room active = false, want true")
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(snapshot.Players))
	}
}

func TestGameService_JoinRoom_GameTypeMismatch_ReturnsErrGameTypeMismatch(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	addServicePlayerForTest(t, service, "p2")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")

	_, err := service.JoinRoom(roomID, "p2", "chess")
	if !errors.Is(err, ErrGameTypeMismatch) {
		t.Fatalf("JoinRoom() error = %v, want %v", err, ErrGameTypeMismatch)
	}

	snapshot, err := service.RoomSnapshot(roomID)
	if err != nil {
		t.Fatalf("RoomSnapshot() error = %v", err)
	}
	if len(snapshot.Players) != 1 {
		t.Fatalf("players len = %d, want 1", len(snapshot.Players))
	}
}

func TestGameService_JoinRoom_RoomNotFound_ReturnsError(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")

	_, err := service.JoinRoom("missing-room", "p1", "tictactoe")
	if !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("JoinRoom() error = %v, want %v", err, ErrRoomNotFound)
	}
}

func TestGameService_GetPlayerInRoom_Existing_ReturnsSnapshot(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")

	player, err := service.GetPlayerInRoom(roomID, "p1")
	if err != nil {
		t.Fatalf("GetPlayerInRoom() error = %v", err)
	}

	if player.ID != "p1" {
		t.Fatalf("player ID = %q, want p1", player.ID)
	}
	if player.Mark != "X" {
		t.Fatalf("player mark = %q, want X", player.Mark)
	}
}

func TestGameService_GetPlayerInRoom_PlayerNotInRoom_ReturnsError(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	addServicePlayerForTest(t, service, "p2")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")

	_, err := service.GetPlayerInRoom(roomID, "p2")
	if !errors.Is(err, ErrPlayerNotFound) {
		t.Fatalf("GetPlayerInRoom() error = %v, want %v", err, ErrPlayerNotFound)
	}
}

func TestGameService_HandleTicTacToeMove_Valid_DelegatesToRoom(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	addServicePlayerForTest(t, service, "p2")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")
	if _, err := service.JoinRoom(roomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	result, err := service.HandleTicTacToeMove(roomID, "p1", 0, 0)
	if err != nil {
		t.Fatalf("HandleTicTacToeMove() error = %v", err)
	}

	if result.State.Board[0][0] != "X" {
		t.Fatalf("board[0][0] = %q, want X", result.State.Board[0][0])
	}
	if result.State.Turn != "O" {
		t.Fatalf("turn = %q, want O", result.State.Turn)
	}
}

func TestGameService_RemovePlayerAfterDelay_PlayerDisconnected_RemovesPlayer(t *testing.T) {
	service, repo := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	addServicePlayerForTest(t, service, "p2")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")
	if _, err := service.JoinRoom(roomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	service.RemovePlayerAfterDelay(roomID, "p2", time.Millisecond, func(string) bool {
		return false
	})

	deadline := time.After(200 * time.Millisecond)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for player removal")
		case <-ticker.C:
			snapshot, err := service.RoomSnapshot(roomID)
			if err != nil {
				if repo.wasDeleted(roomID) {
					t.Fatalf("room was deleted, want room retained with one player")
				}
				continue
			}

			if len(snapshot.Players) == 1 && snapshot.Players[0].ID == "p1" {
				return
			}
		}
	}
}

func TestGameService_RemovePlayerAfterDelayForGeneration_StaleGenerationSkipsRemoval(t *testing.T) {
	service, _ := newGameServiceForTest()
	addServicePlayerForTest(t, service, "p1")
	addServicePlayerForTest(t, service, "p2")
	roomID := createTicTacToeRoomForServiceTest(t, service, "p1")
	if _, err := service.JoinRoom(roomID, "p2", "tictactoe"); err != nil {
		t.Fatalf("JoinRoom() error = %v", err)
	}

	service.RemovePlayerAfterDelayForGeneration(roomID, "p2", 1, time.Millisecond, func(_ string, generation uint64) bool {
		return generation == 2
	})

	time.Sleep(20 * time.Millisecond)

	snapshot, err := service.RoomSnapshot(roomID)
	if err != nil {
		t.Fatalf("RoomSnapshot() error = %v", err)
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("players = %d, want 2 after stale delayed removal", len(snapshot.Players))
	}
}

func TestGameService_Stress_MultipleRooms_ConcurrentJoin(t *testing.T) {
	service, _ := newGameServiceForTest()

	const rooms = 50
	const joiners = 200

	roomIDs := make([]string, 0, rooms)
	for i := 0; i < rooms; i++ {
		creatorID := fmt.Sprintf("creator-%03d", i)
		addServicePlayerForTest(t, service, creatorID)

		res, err := service.CreateRoom("tictactoe", creatorID)
		if err != nil {
			t.Fatalf("CreateRoom(%q) error = %v", creatorID, err)
		}
		roomIDs = append(roomIDs, res.Room.RoomID)
	}

	for i := 0; i < joiners; i++ {
		addServicePlayerForTest(t, service, fmt.Sprintf("joiner-%03d", i))
	}

	start := make(chan struct{})
	errs := make(chan error, joiners)

	var wg sync.WaitGroup
	for i := 0; i < joiners; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					errs <- fmt.Errorf("panic: %v", recovered)
				}
			}()

			<-start

			playerID := fmt.Sprintf("joiner-%03d", i)
			roomID := roomIDs[i%len(roomIDs)]
			_, _ = service.JoinRoom(roomID, playerID, "tictactoe")
			errs <- nil
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, roomID := range roomIDs {
		snapshot, err := service.RoomSnapshot(roomID)
		if err != nil {
			t.Fatalf("RoomSnapshot(%q) error = %v", roomID, err)
		}
		if len(snapshot.Players) > 2 {
			t.Fatalf("room %q players len = %d, want at most 2", roomID, len(snapshot.Players))
		}

		seenPlayers := map[string]bool{}
		for _, player := range snapshot.Players {
			if player.ID == "" {
				t.Fatalf("room %q has empty player ID", roomID)
			}
			if seenPlayers[player.ID] {
				t.Fatalf("room %q has duplicate player ID %q", roomID, player.ID)
			}
			seenPlayers[player.ID] = true
		}
	}
}
