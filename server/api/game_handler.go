package api

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tsaqiffatih/mini-game/actions"
	"github.com/tsaqiffatih/mini-game/chess"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

// Shared utility functions
func parsePayload(conn *websocket.Conn, rawMessage interface{}, payload interface{}) bool {
	payloadBytes, err := json.Marshal(rawMessage)
	if err != nil {
		sendErrorMessage(conn, "Failed to marshal raw message into JSON")
		log.Println("Error marshalling payload:", err)
		return false
	}
	if err := json.Unmarshal(payloadBytes, payload); err != nil {
		sendErrorMessage(conn, "Failed to unmarshal JSON into payload struct")
		log.Println("Error unmarshalling payload:", err)
		return false
	}
	return true
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
		if _, ok := room.GameState.Data.(*chess.ChessGameState); ok {
			for _, player := range room.Players {
				player.Mark = "white"
				log.Printf("Player %s mark reset to white for Chess", player.ID)
				break
			}
		}

	} else {
		log.Println("game state is not recognized")
	}
}

// TicTacToe-related functions
func processTicTacToeMove(conn *websocket.Conn, roomManager *game.RoomManager, message Message) {
	var payload tictactoe.TictactoeMovePayload
	if !parsePayload(conn, message.Message, &payload) {
		return
	}
	handleMakeMoveTictactoe(roomManager, payload, conn)
}

func resetMarkTicTacToeRoom(roomManager *game.RoomManager, room *game.Room) {
	if tictactoeGameState, ok := room.GameState.Data.(*tictactoe.TictactoeGameState); ok {
		resetPlayerMarks(room)

		playerMarks := make(map[string]string)
		for _, player := range room.Players {
			playerMarks[player.ID] = player.Mark
		}
		message := Message{
			Action: actions.MARK_UPDATE,
			Message: map[string]interface{}{
				"marks": playerMarks,
			},
		}

		tictactoeGameState.IsActive = false
		tictactoeGameState.Board = [3][3]string{}
		tictactoeGameState.Turn = "X"

		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
		NotifyTicTacToeClients(roomManager, room.RoomID)
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
		return
	}

	// Trigger AI move if the opponent is AIbled
	if room.IsAIEnabled {
		tictactoeGameState.MakeAIMove()
		NotifyTicTacToeClients(roomManager, payload.RoomID)
	}
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
				}
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
		Action:  actions.TICTACTOE_GAME_STATE,
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

// Chess-related functions
func processChessMove(conn *websocket.Conn, roomManager *game.RoomManager, roomID string, playerId string, message Message) {

	var payload chess.ChessMovePayload
	if !parsePayload(conn, message.Message, &payload) {
		return
	}
	handleChessMove(roomManager, roomID, playerId, payload, conn)
}

func resetMarkChessRoom(roomManager *game.RoomManager, room *game.Room) {
	resetPlayerMarks(room)

	playerMarks := make(map[string]string)
	for _, player := range room.Players {
		playerMarks[player.ID] = player.Mark
	}
	message := Message{
		Action: actions.MARK_UPDATE,
		Message: map[string]interface{}{
			"marks":  playerMarks,
			"active": room.IsActive,
		},
	}
	log.Println("room isActive =>", room.IsActive)
	NotifyToClientsInRoom(roomManager, room.RoomID, &message)

	// if gameState, ok := room.GameState.Data.(string); ok && gameState != "" {
	// 	room.GameState.Data = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	// 	message := Message{
	// 		Action:  "CHESS_GAME_STATE",
	// 		Message: room.GameState.Data,
	// 	}
	// 	NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	// }
	// If game state exists and is chess, reset to starting position
	if chessGameState, ok := room.GameState.Data.(*chess.ChessGameState); ok && chessGameState != nil {
		// replace with new fresh game
		room.GameState.Data = chess.NewChessGameState()
		message := Message{
			Action:  actions.CHESS_GAME_STATE,
			Message: room.GameState.Data.(*chess.ChessGameState).FEN(),
		}
		NotifyToClientsInRoom(roomManager, room.RoomID, &message)
	}
}

func handleChessMove(roomManager *game.RoomManager, roomID, playerID string, payload chess.ChessMovePayload, conn *websocket.Conn) {

	room, err := roomManager.GetRoomByID(roomID)
	if err != nil {
		log.Println("Room not found:", err)
		sendErrorMessage(conn, "Room not found")
		return
	}

	player, exist := room.Players[playerID]
	if !exist {
		sendErrorMessage(conn, "Player not found in room")
		log.Println("Player not found in room:", playerID)
		return
	}

	// if room.GameState.GameType != "chess" {
	// 	log.Println("Invalid game type for chess move")
	// 	sendErrorMessage(conn, "Invalid game type for chess move")
	// 	return
	// }

	// Expect the backend to hold a ChessGameState
	chessGameState, ok := room.GameState.Data.(*chess.ChessGameState)
	if !ok || chessGameState == nil {
		log.Println("Invalid chess game state stored in room")
		sendErrorMessage(conn, "Invalid chess game state")
		return
	}

	// apply move using backend chess engine (UCI: from+to)
	result, err := chessGameState.UpdateState(player.Mark, payload.From, payload.To, payload.Promotion)
	if err != nil {
		// illegal move — notify sender (and optionally broadcast error)
		log.Println("Illegal chess move attempted:", err)
		sendErrorMessage(conn, "Illegal chess move: "+err.Error())
		return
	}

	// broadcast the authoritative FEN from backend
	newFen := chessGameState.FEN()
	message := Message{
		Action: actions.CHESS_MOVE,
		Message: map[string]interface{}{
			"fen":      newFen,
			"lastMove": payload,
			"result":   result,
		},
		Sender: &game.Player{ID: playerID},
	}

	log.Printf("DEBUG CHESS_MOVE (server-validated): %+v\n", message)

	NotifyToClientsInRoom(roomManager, roomID, &message)
}
