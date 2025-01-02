package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/tsaqiffatih/mini-game/game"
)

// validation.go
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

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := Response{
		Success: false,
		Message: message,
	}
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
