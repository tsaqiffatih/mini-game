package game

import (
	"errors"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/tsaqiffatih/mini-game/tictactoe"
)

type Room struct {
	RoomID      string             `json:"room_id"`
	Players     map[string]*Player `json:"players"`
	GameState   *GameRoomState     `json:"game_state"`
	IsActive    bool               `json:"is_active"`
	IsAIEnabled bool               `json:"is_ai_enabled"`
	Mu          sync.Mutex
}

type GameRoomState struct {
	GameType string      `json:"game_type"` // Type of the game, e.g., "chess" or "tictactoe"
	Data     interface{} `json:"data"`      // Game-specific data (e.g., FEN for chess)
}

type JoinRoomResponse struct {
	PlayerID   string `json:"player_id"`
	PlayerMark string `json:"player_mark"`
	Room       *Room  `json:"room"`
}

type RoomManager struct {
	rooms map[string]*Room
	Mu    sync.Mutex
}

// NewRoomManager initializes a new RoomManager.
// It manages the creation and tracking of game rooms.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

// RemoveInactivePlayersFromRoom removes inactive players from all rooms.
// It runs periodically based on the provided duration and interval.
func (rm *RoomManager) RemoveInactivePlayersFromRoom(duration, tickerInterval time.Duration) {
	log.Println("Removing inactive players from room")
	ticker := time.NewTicker(tickerInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()
		rm.Mu.Lock()
		for _, room := range rm.rooms {
			rm.removeInactivePlayersFromRoom(room, now, duration)
		}
		rm.Mu.Unlock()
	}
}

func (rm *RoomManager) removeInactivePlayersFromRoom(room *Room, now time.Time, duration time.Duration) {
	room.Mu.Lock()
	defer room.Mu.Unlock()
	for playerID, player := range room.Players {
		if now.Sub(player.LastActive) > duration {
			delete(room.Players, playerID)
			log.Printf("Player %s removed from room %s due to inactivity", playerID, room.RoomID)
		}
	}
}

// func updatePlayerActivity(player *Player) {
// 	player.LastActive = time.Now()
// }

// CreateRoom creates a new game room with the specified game type.
// It initializes the game state and adds the room to the manager.
func (rm *RoomManager) CreateRoom(roomID string, gameType string) (*Room, error) {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	if _, exists := rm.rooms[roomID]; exists {
		return nil, errors.New("room already exists")
	}

	log.Println("gameType:", gameType)

	gameState := rm.createGameState(gameType)

	log.Println("gameState type:", reflect.TypeOf(gameState))

	room := &Room{
		RoomID:  roomID,
		Players: make(map[string]*Player),
		GameState: &GameRoomState{
			GameType: gameType,
			Data:     gameState,
		},
		IsActive: false,
	}
	log.Println("room.GameState.Data: =>", room.GameState.Data)

	rm.rooms[roomID] = room

	return room, nil
}

func (rm *RoomManager) CreateRoomWithAI(roomID string, gameType string) (*Room, error) {
	room, err := rm.CreateRoom(roomID, gameType)
	if err != nil {
		return nil, err
	}

	// Add AI player
	aiPlayer := &Player{
		ID:   "AI",
		Mark: "O",
	}
	room.Players[aiPlayer.ID] = aiPlayer
	room.IsAIEnabled = true // Mark the room as AI-enabled

	if gameType == "tictactoe" {
		if tictactoeGameState, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
			tictactoeGameState.IsActive = true
		}
	}

	room.IsActive = true
	return room, nil
}

func (rm *RoomManager) createGameState(gameType string) interface{} {
	switch gameType {
	case "tictactoe":
		log.Println("Creating TicTacToe game state")
		return tictactoe.NewGameState()
	case "chess":
		log.Println("Creating Chess game state")
		return "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	default:
		log.Println("Unknown game type:", gameType)
		return nil
	}
}

// JoinRoom allows a player to join an existing room.
// It assigns marks to players and activates the room if ready.
func (rm *RoomManager) JoinRoom(roomID string, player *Player) (*JoinRoomResponse, error) {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	room, exists := rm.rooms[roomID]
	if !exists {
		return nil, errors.New("room not found")
	}

	if len(room.Players) >= 2 {
		return nil, errors.New("room is full")
	}

	if _, exists := room.Players[player.ID]; exists {
		return nil, errors.New("player already in room")
	}

	room.Mu.Lock()
	defer room.Mu.Unlock()

	room.Players[player.ID] = player
	log.Println("gameState type Join Room:", reflect.TypeOf(room.GameState))

	// mengatur mark player untuk game tictactoe
	if room.GameState.GameType == "tictactoe" {
		if tictactoeGameState, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
			if len(room.Players) < 2 {
				player.Mark = "X"
			} else {
				player.Mark = "O"
			}

			if len(room.Players) == 2 || room.IsAIEnabled {
				log.Println("game state is *tictactoe.TictactoeGameState")
				tictactoeGameState.IsActive = true
			}
		} else {
			log.Println("game state is not *tictactoe.TictactoeGameState")
		}
	}

	// mengatur mark player untuk game chess
	if room.GameState.GameType == "chess" {
		if len(room.Players) < 2 {
			log.Println("player mark white")
			player.Mark = "white"
		} else {
			log.Println("player mark black")
			player.Mark = "black"
		}
	}

	if len(room.Players) == 2 {
		room.IsActive = true
	}

	return &JoinRoomResponse{
		PlayerID:   player.ID,
		PlayerMark: player.Mark,
		Room:       room,
	}, nil
}

// GetRoomByID retrieves a room by its ID.
// Returns an error if the room is not found.
func (rm *RoomManager) GetRoomByID(roomID string) (*Room, error) {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	room, exists := rm.rooms[roomID]
	if exists {
		return room, nil
	}
	return nil, errors.New("room not found")
}

// RemoveRoom deletes a room by its ID.
// It removes the room from the manager's tracking.
func (rm *RoomManager) RemoveRoom(roomID string) {
	rm.Mu.Lock()
	defer rm.Mu.Unlock()

	log.Println("Removing room:", roomID)
	delete(rm.rooms, roomID)
}

// GenerateRandomRoomCode generates a random 7-character room code.
// It uses a mix of letters and numbers.
func (rm *RoomManager) GenerateRandomRoomCode() string {
	const possibleCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	gameCode := make([]byte, 7)

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 7; i++ {
		randomIndex := rand.Intn(len(possibleCharacters))
		gameCode[i] = possibleCharacters[randomIndex]
	}

	return string(gameCode)
}
