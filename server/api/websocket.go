package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Message struct {
	Action    string       `json:"action"`
	Message   interface{}  `json:"message"`
	Sender    *game.Player `json:"sender"`
	TimeStamp time.Time    `json:"timestamp"`
}

type ErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request, roomManager *game.RoomManager) {

	roomID := r.URL.Query().Get("room_id")
	playerID := r.URL.Query().Get("player_id")
	log.Println(roomID)
	log.Println(playerID)
	if roomID == "" || playerID == "" {
		response := Response{
			Success: false,
			Message: "roomID and playerID are required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// find player in room
	room, err := roomManager.GetRoomByID(roomID)
	if err != nil {
		log.Println("Error getting room HandleWebSocket:", err)
		response := Response{
			Success: false,
			Message: "could not find room",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	player, exists := room.Players[playerID]
	if !exists {
		log.Println("Player not found in room:", playerID)
		response := Response{
			Success: false,
			Message: "player not found in room",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// for upgrade http connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		response := Response{
			Success: false,
			Message: "could not upgrade connection",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer conn.Close()

	// for update player's connection
	player.Conn = conn
	player.Send = make(chan []byte, 256)

	go player.WritePump()

	log.Printf("Player %s connected to room %s", playerID, roomID)

	message := Message{
		Action:  "CONNECTED_ON_SERVER",
		Message: fmt.Sprintf("Player %s connected to room %s", playerID, room.RoomID),
		Sender:  player,
	}

	// notify other players in the room about the new connection
	NotifyToClientsInRoom(roomManager, roomID, &message)

	if _, ok := room.GameState.(*tictactoe.TictactoeGameState); ok {
		NotifyTicTacToeClients(roomManager, room.RoomID)
	}

	done := make(chan struct{})

	// goroutine to read messages
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message from conn:", err)
				break
			}

			var message Message
			if err := json.Unmarshal(msg, &message); err != nil {
				sendErrorMessage(conn, "Invalid message format")
				log.Println("Error unmarshalling message:", err)
				continue
			}

			switch message.Action {
			case "TICTACTOE_MOVE":
				var payload tictactoe.TictactoeMovePayload
				payloadBytes, err := json.Marshal(message.Message)
				if err != nil {
					sendErrorMessage(conn, "Invalid makeMove payload")
					log.Println("Error marshalling makeMove payload:", err)
					continue
				}
				if err := json.Unmarshal(payloadBytes, &payload); err != nil {
					sendErrorMessage(conn, "Invalid makeMove payload")
					log.Println("Error unmarshalling makeMove payload:", err)
					continue
				}
				handleMakeMoveTictactoe(roomManager, payload, conn)
			default:
				log.Println("Invalid action:", message)
				NotifyToClientsInRoom(roomManager, room.RoomID, &message)
			}
		}
	}()

	<-done
	fmt.Println("Socket connection closed")

	// remove player's connection when websocket is closed
	player.Conn = nil
	close(player.Send)
	log.Printf("Player %s disconnected from room %s", playerID, roomID)

	// notify other players in the room about the disconnection
	disconnectMessage := Message{
		Action:  "USER_LEFT_ROOM",
		Message: fmt.Sprintf("Player %s left the room", playerID),
		Sender:  player,
	}
	NotifyToClientsInRoom(roomManager, roomID, &disconnectMessage)

	// goroutine for remove player after a delay
	go func() {
		time.Sleep(2 * time.Minute)
		room.Mu.Lock()
		defer room.Mu.Unlock()

		// check if player has reconnected
		if player.Conn == nil {
			log.Printf("Removing player %s from room %s after delay", playerID, roomID)

			// remove player from room
			delete(room.Players, playerID)
			log.Printf("Player %s removed from room %s", playerID, roomID)

			if tictactoeGameState, ok := room.GameState.(*tictactoe.TictactoeGameState); ok {
				if len(room.Players) < 2 {
					tictactoeGameState.IsActive = false
					NotifyTicTacToeClients(roomManager, room.RoomID)
				}
			} else {
				log.Println("game state is not *tictactoe.TictactoeGameState")
			}

			if len(room.Players) < 2 {
				room.IsActive = false
				NotifyTicTacToeClients(roomManager, room.RoomID)
			}

			// remove Room if no player is connected
			if len(room.Players) == 0 {
				log.Println("Removing room:", roomID)
				roomManager.RemoveRoom(roomID)
			}
		} else {
			log.Printf("Player %s reconnected to room %s", playerID, roomID)
		}
	}()
}

func NotifyTicTacToeClients(roomManager *game.RoomManager, roomID string) {
	room, err := roomManager.GetRoomByID(roomID)
	if err != nil {
		log.Println("Error getting room NotifyClients:", err)
		return
	}

	gameState := room.GameState
	for _, player := range room.Players {
		if player.Conn != nil {
			if tictactoeGameState, ok := gameState.(*tictactoe.TictactoeGameState); ok {
				sendGameState(player, tictactoeGameState)
			} else {
				log.Println("Error asserting gameState to *tictactoe.TictactoeGameState")
			}
		}
	}
}

func handleMakeMoveTictactoe(roomManager *game.RoomManager, payload tictactoe.TictactoeMovePayload, conn *websocket.Conn) {
	room, err := roomManager.GetRoomByID(payload.RoomID)
	if err != nil {
		sendErrorMessage(conn, "Room not found")
		log.Println("Room not found:", payload.RoomID)
		return
	}

	player, exists := room.Players[payload.PlayerID]
	if !exists {
		sendErrorMessage(conn, "Player not found in room")
		log.Println("Player not found in room:", payload.PlayerID)
		return
	}

	player.LastActive = time.Now()

	tictactoeGameState, ok := room.GameState.(*tictactoe.TictactoeGameState)
	if !ok {
		sendErrorMessage(conn, "Invalid game state")
		log.Println("Invalid game state for room:", payload.RoomID)
		return
	}
	err = tictactoeGameState.UpdateState(player.Mark, payload.Row, payload.Col)
	if err != nil {
		sendErrorMessage(conn, err.Error())
		log.Println("Error making move:", err)
		return
	}

	log.Println("Move made by player:", payload.PlayerID)
	NotifyTicTacToeClients(roomManager, payload.RoomID)

	if !tictactoeGameState.IsActive {
		time.Sleep(5 * time.Second)
		tictactoeGameState.ResetGame()
		NotifyTicTacToeClients(roomManager, payload.RoomID)
	}
}

// func NotifyToClientsInRoom(roomManager *game.RoomManager, roomID string, message *Message) {
// 	room, err := roomManager.GetRoomByID(roomID)
// 	if err != nil {
// 		log.Println("Error getting room NotifyClients:", err)
// 		return
// 	}
// 	for _, player := range room.Players {
// 		if player.Conn != nil {
// 			if err := player.Conn.WriteJSON(message); err != nil {
// 				log.Println("Error sending message to client:", err)
// 			}
// 		}
// 	}
// }

func NotifyToClientsInRoom(roomManager *game.RoomManager, roomID string, message *Message) {
	room, err := roomManager.GetRoomByID(roomID)
	if err != nil {
		log.Println("Error getting room NotifyClients:", err)
		return
	}

	message.TimeStamp = time.Now()

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling message:", err)
		return
	}

	for _, player := range room.Players {
		if player.Conn != nil {
			select {
			case player.Send <- messageBytes:
			default:
				close(player.Send)
				delete(room.Players, player.ID)
			}
		}
	}
}

func sendGameState(player *game.Player, gameState *tictactoe.TictactoeGameState) {
	board, turn, winner, isActive := gameState.GetState()
	log.Println("Sending game state to client:", player.ID, board, turn, winner, isActive)
	response := tictactoe.TicTacToeGameResponse{
		Board:    board,
		Turn:     turn,
		Winner:   winner,
		IsActive: isActive,
	}
	// responseBytes, err := json.Marshal(response)
	// if err != nil {
	// 	log.Println("Error marshalling game state response:", err)
	// 	return
	// }
	message := Message{
		Action:  "TICTACTOE_GAME_STATE",
		Message: response,
		Sender: &game.Player{
			ID:         player.ID,
			Mark:       player.Mark,
			Conn:       nil,
			LastActive: player.LastActive,
			Send:       nil,
		},
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Println("Error marshalling game state message:", err)
		return
	}

	select {
	case player.Send <- messageBytes:
	default:
		close(player.Send)
	}
}

func sendErrorMessage(conn *websocket.Conn, message string) {
	errorMessage := ErrorMessage{
		Type:    "error",
		Message: message,
	}
	if err := conn.WriteJSON(errorMessage); err != nil {
		log.Println("Error sending error message:", err)
	}
}
