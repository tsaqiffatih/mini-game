package game

import (
	"errors"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID         string `json:"player_id"`
	Mark       string `json:"player_mark"`
	Conn       *websocket.Conn
	LastActive time.Time
	Send       chan []byte `json:"-"`
}

type PlayerManager struct {
	players map[string]*Player
}

// jalanin goroutine lagi untuk menghapus user yang udah gak aktif selama 1x24 jam

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players: make(map[string]*Player),
	}
}

// func (pm *PlayerManager) SwitchIsActive(gs *GameState) {
// 	if len(pm.players) == 2 {
// 		gs.IsActive = true
// 	} else {
// 		gs.IsActive = false
// 	}
// }

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

func (p *Player) WritePump() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in WritePump:", r)
		}
	}()
	for {
		select {
		case message, ok := <-p.Send:
			if !ok {
				// channel closed, close the WebSocket connection
				if p.Conn != nil {
					p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				}
				return
			}

			// for write message to WebSocket connection
			if err := p.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println("Error writing message to WebSocket:", err)
				return
			}
		}
	}
}

func (pm *PlayerManager) GetPlayer(playerID string) (*Player, error) {
	player, exists := pm.players[playerID]
	if !exists {
		return nil, errors.New("player not found")
	}
	return player, nil
}

func (pm *PlayerManager) RemovePlayer(playerID string) {
	delete(pm.players, playerID)
}

func (pm *PlayerManager) GetAllPlayers() []*Player {
	allPlayers := []*Player{}
	for _, player := range pm.players {
		allPlayers = append(allPlayers, player)
	}
	return allPlayers
}
