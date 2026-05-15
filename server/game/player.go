package game

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/tsaqiffatih/mini-game/internal/observability"
)

type PlayerSessionStatus string

const (
	PlayerSessionConnected    PlayerSessionStatus = "connected"
	PlayerSessionDisconnected PlayerSessionStatus = "disconnected"
	PlayerSessionRemoved      PlayerSessionStatus = "removed"
)

type Player struct {
	ID         string `json:"player_id"`
	Mark       string `json:"player_mark"`
	IsAI       bool   `json:"is_ai"`
	LastActive time.Time
	Session    PlayerSessionStatus
}

type PlayerManager struct {
	players map[string]*Player
	mu      sync.RWMutex
}

// NewPlayerManager initializes a new PlayerManager.
// It manages the creation and tracking of players.
func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players: make(map[string]*Player),
	}
}

// RemoveInactivePlayers removes players who have been inactive for a specified duration.
// It runs periodically based on the provided interval.
func (pm *PlayerManager) RemoveInactivePlayers(ctx context.Context, duration, tickerInterval time.Duration) {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		now := time.Now()
		pm.mu.Lock()
		for playerID, player := range pm.players {
			if now.Sub(player.LastActive) > duration {
				delete(pm.players, playerID)
				observability.Logger().InfoContext(ctx, "inactive player removed",
					"room_id", "",
					"player_id", playerID,
					"event_type", "player_removed",
				)
			}
		}
		pm.mu.Unlock()
	}
}

// AddPlayer adds a new player to the manager.
// Returns an error if the player already exists.
func (pm *PlayerManager) AddPlayer(playerID string) (PlayerSnapshot, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, exists := pm.players[playerID]
	if exists {
		return PlayerSnapshot{}, errors.New("player already exists, choose another name")
	}

	player := &Player{
		ID:         playerID,
		LastActive: time.Now(),
		Session:    PlayerSessionDisconnected,
	}
	pm.players[playerID] = player
	return playerSnapshot(player), nil
}

// GetPlayer retrieves a player by their ID.
// Returns an error if the player is not found.
func (pm *PlayerManager) GetPlayer(playerID string) (PlayerSnapshot, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	player, exists := pm.players[playerID]
	if !exists {
		return PlayerSnapshot{}, errors.New("player not found")
	}
	return playerSnapshot(player), nil
}

// RemovePlayer removes a player by their ID.
// It deletes the player from the manager's tracking.
func (pm *PlayerManager) RemovePlayer(playerID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.players, playerID)
}

// GetAllPlayers retrieves all players managed by the PlayerManager.
// Returns a slice of all players.
func (pm *PlayerManager) GetAllPlayers() []PlayerSnapshot {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	allPlayers := []PlayerSnapshot{}
	for _, player := range pm.players {
		allPlayers = append(allPlayers, playerSnapshot(player))
	}
	return allPlayers
}

func playerSnapshot(player *Player) PlayerSnapshot {
	return PlayerSnapshot{
		ID:         player.ID,
		Mark:       player.Mark,
		IsAI:       player.IsAI,
		LastActive: player.LastActive,
		Session:    player.Session,
	}
}
