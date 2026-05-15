package api

import (
	"net/http"

	"github.com/tsaqiffatih/mini-game/internal/observability"
)

// validation.go
func validateRoomAndPlayerIDs(w http.ResponseWriter, roomID, playerID string) bool {
	if roomID == "" || playerID == "" {
		observability.Logger().Warn("room and player ids are required",
			"room_id", roomID,
			"player_id", playerID,
			"event_type", "validation_error",
		)
		sendErrorResponse(w, "roomID and playerID are required", http.StatusBadRequest)
		return false
	}
	return true
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	writeErrorResponse(w, statusCode, message)
}
