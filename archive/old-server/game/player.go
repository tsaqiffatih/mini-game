package game

import (
	"errors"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID         string `json:"player_id"`
	Mark       string `json:"player_mark"`
	IsAI       bool   `json:"is_ai"`
	Conn       *websocket.Conn
	LastActive time.Time
	Send       chan []byte `json:"-"`
}

type PlayerManager struct {
	players map[string]*Player
	mu      sync.Mutex
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
func (pm *PlayerManager) RemoveInactivePlayers(duration, tickerInterval time.Duration) {
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()
		pm.mu.Lock()
		for playerID, player := range pm.players {
			if now.Sub(player.LastActive) > duration {
				delete(pm.players, playerID)
				log.Printf("Player %s removed from system due to inactivity", playerID)
			}
		}
		pm.mu.Unlock()
	}
}

// UpdatePlayerActivity updates the last active time of a player.
// It ensures the player is marked as active.
func (pm *PlayerManager) UpdatePlayerActivity(playerID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if player, exists := pm.players[playerID]; exists {
		player.LastActive = time.Now()
	}
}

// AddPlayer adds a new player to the manager.
// Returns an error if the player already exists.
func (pm *PlayerManager) AddPlayer(playerID string) (*Player, error) {
	_, exists := pm.players[playerID]
	if exists {
		return nil, errors.New("player already exists, choose another name")
	}

	player := &Player{
		ID:         playerID,
		LastActive: time.Now(),
	}
	pm.players[playerID] = player
	return player, nil
}

// WritePump handles sending messages to the player's WebSocket connection.
// It ensures messages are delivered or the connection is closed.
func (p *Player) WritePump() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in WritePump:", r)
		}
	}()
	for message := range p.Send {
		// for write message to WebSocket connection
		if err := p.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("Error writing message to WebSocket:", err)
			return
		}
	}

	// channel closed, close the WebSocket connection
	if p.Conn != nil {
		p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
	}
}

// GetPlayer retrieves a player by their ID.
// Returns an error if the player is not found.
func (pm *PlayerManager) GetPlayer(playerID string) (*Player, error) {
	player, exists := pm.players[playerID]
	if !exists {
		return nil, errors.New("player not found")
	}
	return player, nil
}

// RemovePlayer removes a player by their ID.
// It deletes the player from the manager's tracking.
func (pm *PlayerManager) RemovePlayer(playerID string) {
	delete(pm.players, playerID)
}

// GetAllPlayers retrieves all players managed by the PlayerManager.
// Returns a slice of all players.
func (pm *PlayerManager) GetAllPlayers() []*Player {
	allPlayers := []*Player{}
	for _, player := range pm.players {
		allPlayers = append(allPlayers, player)
	}
	return allPlayers
}
