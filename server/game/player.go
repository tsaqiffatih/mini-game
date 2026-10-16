package game

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/tsaqiffatih/mini-game/internal/observability"
)

const maxPlayerIDRunes = 32

var ErrInvalidPlayerID = errors.New("invalid player ID")

type PlayerSessionStatus string

const (
	PlayerSessionConnected    PlayerSessionStatus = "connected"
	PlayerSessionDisconnected PlayerSessionStatus = "disconnected"
	PlayerSessionRemoved      PlayerSessionStatus = "removed"
)

type Player struct {
	ID           string `json:"player_id"`
	Mark         string `json:"player_mark"`
	IsAI         bool   `json:"is_ai"`
	LastActive   time.Time
	Session      PlayerSessionStatus
	SessionToken string
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
	if !validPlayerID(playerID) {
		return PlayerSnapshot{}, ErrInvalidPlayerID
	}

	sessionToken, err := generateSessionToken()
	if err != nil {
		return PlayerSnapshot{}, err
	}

	return pm.addPlayer(playerID, sessionToken)
}

func (pm *PlayerManager) addPlayer(playerID string, sessionToken string) (PlayerSnapshot, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	_, exists := pm.players[playerID]
	if exists {
		return PlayerSnapshot{}, errors.New("player already exists, choose another name")
	}

	player := &Player{
		ID:           playerID,
		LastActive:   time.Now(),
		Session:      PlayerSessionDisconnected,
		SessionToken: sessionToken,
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

// AuthenticatePlayer verifies that a session token belongs to the requested player.
func (pm *PlayerManager) AuthenticatePlayer(playerID string, sessionToken string) (PlayerSnapshot, bool) {
	if sessionToken == "" {
		return PlayerSnapshot{}, false
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	player, exists := pm.players[playerID]
	if !exists || subtle.ConstantTimeCompare([]byte(player.SessionToken), []byte(sessionToken)) != 1 {
		return PlayerSnapshot{}, false
	}

	return playerSnapshot(player), true
}

func (pm *PlayerManager) SessionTokenForPlayer(playerID string) (string, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	player, exists := pm.players[playerID]
	if !exists {
		return "", false
	}
	return player.SessionToken, true
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

func validPlayerID(playerID string) bool {
	trimmed := strings.TrimSpace(playerID)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxPlayerIDRunes {
		return false
	}
	for _, r := range trimmed {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func generateSessionToken() (string, error) {
	const tokenBytes = 32

	bytes := make([]byte, tokenBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
