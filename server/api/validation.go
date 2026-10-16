package api

import (
	"net/http"
	"regexp"

	"github.com/tsaqiffatih/mini-game/internal/observability"
)

// validation.go
func validateRoomAndPlayerIDs(w http.ResponseWriter, roomID, playerID string) bool {
	// IDs must be 6-30 characters, alphanumeric, underscores or hyphens
	var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,30}$`)

	if !idPattern.MatchString(roomID) {
		observability.Logger().Warn("invalid room ID format",
			"room_id", roomID,
			"event_type", "validation_error",
		)
		sendErrorResponse(w, "invalid room ID", http.StatusBadRequest)
		return false
	}
	if !idPattern.MatchString(playerID) {
		observability.Logger().Warn("invalid player ID format",
			"player_id", playerID,
			"event_type", "validation_error",
		)
		sendErrorResponse(w, "invalid player ID", http.StatusBadRequest)
		return false
	}
	return true
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	writeErrorResponse(w, statusCode, message)
}
