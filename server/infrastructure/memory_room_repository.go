package infrastructure

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
)

type MemoryRoomRepository struct {
	rooms map[string]*game.Room
	mu    sync.RWMutex
}

func NewMemoryRoomRepository() *MemoryRoomRepository {
	return &MemoryRoomRepository{
		rooms: make(map[string]*game.Room),
	}
}

func (r *MemoryRoomRepository) GetByID(ctx context.Context, roomID string) (*game.Room, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	room, exists := r.rooms[roomID]
	if exists {
		return room, nil
	}

	return nil, errors.New("room not found")
}

func (r *MemoryRoomRepository) Save(ctx context.Context, room *game.Room) error {
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

func (r *MemoryRoomRepository) Delete(ctx context.Context, roomID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if room, exists := r.rooms[roomID]; exists {
		room.Close()
	}
	delete(r.rooms, roomID)
	observability.Logger().InfoContext(ctx, "room removed",
		"room_id", roomID,
		"player_id", "",
		"event_type", "room_removed",
	)
	return nil
}

func (r *MemoryRoomRepository) List(ctx context.Context) ([]*game.Room, error) {
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

func (r *MemoryRoomRepository) RemoveInactivePlayersFromRoom(duration, tickerInterval time.Duration) {
	ctx := context.Background()
	observability.Logger().InfoContext(ctx, "room inactive player cleanup started",
		"room_id", "",
		"player_id", "",
		"event_type", "cleanup_started",
	)
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()
		rooms, err := r.List(ctx)
		if err != nil {
			continue
		}
		for _, room := range rooms {
			room.RemoveInactivePlayers(now, duration)
			if room.IsEmpty() {
				_ = r.Delete(ctx, room.RoomID)
			}
		}
	}
}
