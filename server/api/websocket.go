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

	if room.GameState.GameType == "tictactoe" {
		if _, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
			NotifyTicTacToeClients(roomManager, room.RoomID)
		}
	}

	if room.GameState.GameType == "chess" {
		if gameState, ok := room.GameState.Data.(string); ok && gameState != "" {
			chessMessage := Message{
				Action:  "CHESS_GAME_STATE",
				Message: gameState,
				Sender:  player,
			}
			NotifyToClientsInRoom(roomManager, roomID, &chessMessage)
		}

		if room.IsActive == true {
			// Notify other players in the room about the new connection
			message := Message{
				Action:  "START_GAME",
				Message: fmt.Sprintf("Player %s left the room", playerID),
				Sender:  player,
			}
			NotifyToClientsInRoom(roomManager, roomID, &message)
		}
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
			case "CHESS_MOVE":
				var payload struct {
					FEN string `json:"fen"`
				}
				payloadBytes, err := json.Marshal(message.Message)
				if err != nil {
					sendErrorMessage(conn, "Invalid move payload")
					log.Println("Error marshalling move payload:", err)
					continue
				}
				if err := json.Unmarshal(payloadBytes, &payload); err != nil {
					sendErrorMessage(conn, "Invalid move payload")
					log.Println("Error unmarshalling move payload:", err)
					continue
				}
				handleChessMove(roomManager, roomID, playerID, payload.FEN, conn)
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
		time.Sleep(30 * time.Second)
		room.Mu.Lock()
		defer room.Mu.Unlock()

		// check if player has reconnected
		if player.Conn == nil {
			log.Printf("Removing player %s from room %s after delay", playerID, roomID)

			// remove player from room
			delete(room.Players, playerID)
			log.Printf("Player %s removed from room %s", playerID, roomID)

			if len(room.Players) < 2 {
				room.IsActive = false
				if room.GameState.GameType == "chess" {
					resetPlayerMarks(room)

					playerMarks := make(map[string]string)
					for _, player := range room.Players {
						playerMarks[player.ID] = player.Mark
					}
					message := Message{
						Action: "MARK_UPDATE",
						Message: map[string]interface{}{
							"marks": playerMarks,
						},
					}
					NotifyToClientsInRoom(roomManager, room.RoomID, &message)
					if gameState, ok := room.GameState.Data.(string); ok && gameState != "" {
						room.GameState.Data = ""
						gamesteMessage := Message{
							Action:  "CHESS_GAME_STATE",
							Message: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
							Sender:  player,
						}
						NotifyToClientsInRoom(roomManager, roomID, &gamesteMessage)
					}
				} else if room.GameState.GameType == "tictactoe" {
					if tictactoeGameState, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
						resetPlayerMarks(room)
						playerMarks := make(map[string]string)
						for _, player := range room.Players {
							playerMarks[player.ID] = player.Mark
						}
						message := Message{
							Action: "MARK_UPDATE",
							Message: map[string]interface{}{
								"marks": playerMarks,
							},
						}
						tictactoeGameState.IsActive = false
						NotifyToClientsInRoom(roomManager, room.RoomID, &message)
						NotifyTicTacToeClients(roomManager, room.RoomID)
					} else {
						log.Println("Error asserting gameState to *tictactoe.TictactoeGameState")
					}
				} else {
					resetPlayerMarks(room)
				}
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

func handleChessMove(roomManager *game.RoomManager, roomID, playerID, fen string, conn *websocket.Conn) {
	room, err := roomManager.GetRoomByID(roomID)
	if err != nil {
		log.Println("Room not found:", err)
		sendErrorMessage(conn, "Room not found")
		return
	}

	if room.GameState.GameType != "chess" {
		log.Println("Invalid game type for chess move")
		sendErrorMessage(conn, "Invalid game type for chess move")
		return
	}

	room.GameState.Data = fen
	log.Println("chessGameState:", fen)

	// Notify other players in the room about the move
	message := Message{
		Action:  "CHESS_MOVE",
		Message: fen,
		Sender:  &game.Player{ID: playerID},
	}

	NotifyToClientsInRoom(roomManager, roomID, &message)
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
			if room.GameState.GameType == "tictactoe" {
				if tictactoeGameState, ok := gameState.Data.(*tictactoe.TictactoeGameState); ok {
					sendTictactoeGameState(player, tictactoeGameState)
				} else {
					log.Println("Error asserting gameState to *tictactoe.TictactoeGameState")
				}
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

	tictactoeGameState, ok := room.GameState.Data.(*tictactoe.TictactoeGameState)
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

func sendTictactoeGameState(player *game.Player, gameState *tictactoe.TictactoeGameState) {
	board, turn, winner, isActive := gameState.GetState()
	log.Println("Sending game state to client:", player.ID, board, turn, winner, isActive)
	response := tictactoe.TicTacToeGameResponse{
		Board:    board,
		Turn:     turn,
		Winner:   winner,
		IsActive: isActive,
	}

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

func resetPlayerMarks(room *game.Room) {
	if room.GameState.GameType == "tictactoe" {
		if _, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
			// ticTacToe: reset ke "X" jika tinggal satu pemain
			for _, player := range room.Players {
				player.Mark = "X"
				log.Printf("Player %s mark reset to X for TicTacToe", player.ID)
				break
			}
		}
	} else if room.GameState.GameType == "chess" {
		// chess: reset ke "white" jika tinggal satu pemain
		for _, player := range room.Players {
			player.Mark = "white"
			log.Printf("Player %s mark reset to white for Chess", player.ID)
			break
		}

	} else {
		log.Println("game state is not recognized")
	}
}
