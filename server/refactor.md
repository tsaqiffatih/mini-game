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

func HandleWebSocket(w http.ResponseWriter, r *http.Request, roomManager *game.RoomManager, playerManager *game.PlayerManager) {
	roomID := r.URL.Query().Get("room_id")
	playerID := r.URL.Query().Get("player_id")

	if !validateRoomAndPlayerIDs(w, roomID, playerID) {
		return
	}

	room, player, err := getRoomAndPlayer(w, roomManager, roomID, playerID)
	if err != nil {
		return
	}

	// for upgrade http connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sendErrorResponse(w, "could not upgrade connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	// for update player's connection
	player.Conn = conn
	player.Send = make(chan []byte, 256)

	go player.WritePump()

	log.Printf("Player %s connected to room %s", playerID, roomID)

	notifyRoomOnConnection(roomManager, room, player)

	done := make(chan struct{})

	// goroutine to read messages
	go readMessages(conn, done, roomManager, room, player, playerManager)

	<-done
	fmt.Println("Socket connection closed")

	// goroutine for remove player after a delay
	handlePlayerDisconnection(roomManager, room, playerID, player)
}

func validateRoomAndPlayerIDs(w http.ResponseWriter, roomID, playerID string) bool {
	if roomID == "" || playerID == "" {
		log.Println("roomID and playerID are required")
		sendErrorResponse(w, "roomID and playerID are required", http.StatusBadRequest)
		return false
	}
	return true
}

func getRoomAndPlayer(w http.ResponseWriter, roomManager *game.RoomManager, roomID, playerID string) (*game.Room, *game.Player, error) {
	room, err := roomManager.GetRoomByID(roomID)
	if err != nil {
		sendErrorResponse(w, "could not find room", http.StatusInternalServerError)
		return nil, nil, err
	}

	player, exists := room.Players[playerID]
	if !exists {
		sendErrorResponse(w, "player not found in room", http.StatusInternalServerError)
		return nil, nil, fmt.Errorf("player not found")
	}

	return room, player, nil
}

func notifyRoomOnConnection(roomManager *game.RoomManager, room *game.Room, player *game.Player) {
	message := Message{
		Action:  "CONNECTED_ON_SERVER",
		Message: fmt.Sprintf("Player %s connected to room %s", player.ID, room.RoomID),
		Sender:  player,
	}
	NotifyToClientsInRoom(roomManager, room.RoomID, &message)

	switch room.GameState.GameType {
	case "tictactoe":
		NotifyTicTacToeClients(roomManager, room.RoomID)
	case "chess":
		notifyChessClients(roomManager, room, player)
	}
}

func notifyChessClients(roomManager *game.RoomManager, room *game.Room, player *game.Player) {
	if gameState, ok := room.GameState.Data.(string); ok && gameState != "" {
		message := Message{
			Action:  "CHESS_GAME_STATE",
			Message: gameState,
			Sender:  player,
		}
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}

	if room.IsActive {
		message := Message{
			Action:  "START_GAME",
			Message: fmt.Sprintf("Player %s left the room", player.ID),
			Sender:  player,
		}
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}
}

func readMessages(conn *websocket.Conn, done chan struct{}, roomManager *game.RoomManager, room *game.Room, player *game.Player, playerManager *game.PlayerManager) {
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

		playerManager.UpdatePlayerActivity(player.ID)
		handleMessageAction(conn, roomManager, room, player, message)
	}
}

func handleMessageAction(conn *websocket.Conn, roomManager *game.RoomManager, room *game.Room, player *game.Player, message Message) {
	switch message.Action {
	case "TICTACTOE_MOVE":
		processTicTacToeMove(conn, roomManager, message)
	case "CHESS_MOVE":
		processChessMove(conn, roomManager, room.RoomID, player.ID, message)
	default:
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}
}

func processTicTacToeMove(conn *websocket.Conn, roomManager *game.RoomManager, message Message) {
	var payload tictactoe.TictactoeMovePayload
	if !parsePayload(conn, message.Message, &payload) {
		return
	}
	handleMakeMoveTictactoe(roomManager, payload, conn)
}

func processChessMove(conn *websocket.Conn, roomManager *game.RoomManager, roomID string, playerId string, message Message) {
	var payload struct {
		FEN string `json:"fen"`
	}
	if !parsePayload(conn, message.Message, &payload) {
		return
	}
	handleChessMove(roomManager, roomID, playerId, payload.FEN, conn)
}

func parsePayload(conn *websocket.Conn, rawMessage interface{}, payload interface{}) bool {
	payloadBytes, err := json.Marshal(rawMessage)
	if err != nil {
		sendErrorMessage(conn, "Invalid payload")
		log.Println("Error marshalling payload:", err)
		return false
	}
	if err := json.Unmarshal(payloadBytes, payload); err != nil {
		sendErrorMessage(conn, "Invalid payload")
		log.Println("Error unmarshalling payload:", err)
		return false
	}
	return true
}

func handlePlayerDisconnection(roomManager *game.RoomManager, room *game.Room, playerID string, player *game.Player) {
	player.Conn = nil
	message := Message{
		Action:  "USER_LEFT_ROOM",
		Message: fmt.Sprintf("Player %s left the room", playerID),
		Sender:  player,
	}
	NotifyToClientsInRoom(roomManager, room.RoomID, &message)

	go removePlayerAfterDelay(roomManager, room, playerID)
}

func removePlayerAfterDelay(roomManager *game.RoomManager, room *game.Room, playerID string) {
	time.Sleep(30 * time.Second)
	room.Mu.Lock()
	defer room.Mu.Unlock()

	if player, exists := room.Players[playerID]; exists && player.Conn == nil {
		delete(room.Players, playerID)
		handleRoomAfterPlayerLeft(roomManager, room)
	}
}

func handleRoomAfterPlayerLeft(roomManager *game.RoomManager, room *game.Room) {
	if len(room.Players) == 0 {
		roomManager.RemoveRoom(room.RoomID)
	}
	
	if len(room.Players) < 2 {
		room.IsActive = false
		switch room.GameState.GameType {
		case "chess":
			resetMarkChessRoom(room)
		case "tictactoe":
			resetMarkTicTacToeRoom(roomManager, room)
		default:
			resetPlayerMarks(room)
		}
	}
}

func resetMarkChessRoom(room *game.Room) {
	resetPlayerMarks(room)
	if gameState, ok := room.GameState.Data.(string); ok && gameState != "" {
		room.GameState.Data = ""
		message := Message{
			Action:  "CHESS_GAME_STATE",
			Message: "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		}
		NotifyToClientsInRoom(nil, room.RoomID, &message)
	}
}

func resetMarkTicTacToeRoom(roomManager *game.RoomManager, room *game.Room) {
	if tictactoeGameState, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
		tictactoeGameState.IsActive = false
		tictactoeGameState.ResetGame()
		NotifyTicTacToeClients(roomManager, room.RoomID)
		log.Println("TicTacToe gameState =>", tictactoeGameState.IsActive)
	}
	resetPlayerMarks(room)
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := Response{
		Success: false,
		Message: message,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
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
					log.Println("NotifyTicTacToeClients TicTacToe gameState =>", tictactoeGameState.IsActive)
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

	log.Println("isActive =>", isActive)

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
