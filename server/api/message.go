package api

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/actions"
	"github.com/tsaqiffatih/mini-game/chess"
	"github.com/tsaqiffatih/mini-game/game"
)

// message.go
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
	case actions.TICTACTOE_MOVE:
		processTicTacToeMove(conn, roomManager, message)
	case actions.CHESS_MOVE:
		processChessMove(conn, roomManager, room.RoomID, player.ID, message)
	case actions.CREATE_ROOM_WITH_AI:
		room, err := roomManager.CreateRoomWithAI(message.Message.(string), "tictactoe")
		if err != nil {
			sendErrorMessage(conn, err.Error())
			return
		}
		NotifyToClientsInRoom(roomManager, room.RoomID, &Message{
			Action:  actions.ROOM_CREATED,
			Message: room,
		})
	default:
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}
}

func notifyRoomOnConnection(roomManager *game.RoomManager, room *game.Room, player *game.Player) {
	message := Message{
		Action:  actions.CONNECTED_ON_SERVER,
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
	if chessState, ok := room.GameState.Data.(*chess.ChessGameState); ok && chessState != nil {
		message := Message{
			Action:  actions.CHESS_GAME_STATE,
			Message: chessState.FEN(),
			Sender:  player,
		}
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}

	if room.IsActive {
		message := Message{
			Action:  actions.START_GAME,
			Message: fmt.Sprintf("Player %s left the room", player.ID),
			Sender:  player,
		}
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}
}

func handleRoomAfterPlayerLeft(roomManager *game.RoomManager, room *game.Room) {

	if len(room.Players) < 2 {
		room.IsActive = false
		switch room.GameState.GameType {
		case "chess":
			resetMarkChessRoom(roomManager, room)
		case "tictactoe":
			resetMarkTicTacToeRoom(roomManager, room)
		default:
			resetPlayerMarks(room)
		}
	}

	if len(room.Players) == 0 {
		roomManager.RemoveRoom(room.RoomID)
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
