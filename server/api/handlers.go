package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/service"

)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *string     `json:"error"`
	Message string      `json:"-"`
}

func RegisterRouter(r *mux.Router, clients *ClientRegistry, gameService *service.GameService) {
	r.HandleFunc("/health", health).Methods(http.MethodGet)
	r.HandleFunc("/healthz", health).Methods(http.MethodGet)
	r.HandleFunc("/create/user", func(w http.ResponseWriter, r *http.Request) {
		addPlayer(w, r, gameService)
	}).Methods("POST")

	r.HandleFunc("/room/join", func(w http.ResponseWriter, r *http.Request) {
		joinRoom(w, r, gameService)
	}).Methods("POST")

	r.HandleFunc("/room/create", func(w http.ResponseWriter, r *http.Request) {
		createRoom(w, r, gameService)
	}).Methods("POST")

	r.HandleFunc("/room/create/ai", func(w http.ResponseWriter, r *http.Request) {
		createRoomWithAi(w, r, gameService)
	}).Methods("POST")

	r.HandleFunc("/room/recover", func(w http.ResponseWriter, r *http.Request) {
		recoverRoom(w, r, gameService)
	}).Methods("POST")

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(w, r, clients, gameService)
	})

}

func createRoom(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {

	var request struct {
		GameType string `json:"game_type"`
		PlayerID string `json:"player_id"`
	}

	if err := decodeRequestBody(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !authenticateHTTPPlayer(w, r, gameService, request.PlayerID) {
		return
	}

	res, err := gameService.CreateRoomWithContext(r.Context(), request.GameType, request.PlayerID)
	if err != nil {
		writeErrorResponse(w, createRoomStatus(err), err.Error())
		return
	}

	writeSuccessResponse(w, http.StatusCreated, dto.FromJoinRoomResponse(res))
}

func createRoomWithAi(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {

	var request struct {
		GameType string `json:"game_type"`
		PlayerID string `json:"player_id"`
		AILevel  *int   `json:"ai_level,omitempty"`
	}

	if err := decodeRequestBody(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !authenticateHTTPPlayer(w, r, gameService, request.PlayerID) {
		return
	}
	aiLevel := game.DefaultAILevel
	if request.AILevel != nil {
		aiLevel = *request.AILevel
	}
	if !game.ValidAILevel(aiLevel) {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	res, err := gameService.CreateRoomWithAILevelWithContext(r.Context(), request.GameType, request.PlayerID, aiLevel)
	if err != nil {
		writeErrorResponse(w, createRoomStatus(err), err.Error())
		return
	}

	writeSuccessResponse(w, http.StatusCreated, dto.FromJoinRoomResponse(res))
}

func joinRoom(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {

	var request struct {
		RoomID   string `json:"room_id"`
		PlayerID string `json:"player_id"`
		GameType string `json:"game_type"`
	}

	if err := decodeRequestBody(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !validRoomID(request.RoomID) {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !authenticateHTTPPlayer(w, r, gameService, request.PlayerID) {
		return
	}

	res, err := gameService.JoinRoomWithContext(r.Context(), request.RoomID, request.PlayerID, request.GameType)
	if err != nil {
		writeErrorResponse(w, joinRoomStatus(err), err.Error())
		return
	}

	writeSuccessResponse(w, http.StatusOK, dto.FromJoinRoomResponse(res))
}

func recoverRoom(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {
	var request struct {
		RoomID   string `json:"room_id"`
		PlayerID string `json:"player_id"`
		GameType string `json:"game_type"`
	}

	if err := decodeRequestBody(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !validRoomID(request.RoomID) {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if !authenticateHTTPPlayer(w, r, gameService, request.PlayerID) {
		return
	}

	snapshot, player, err := gameService.RecoverRoomSessionWithContext(r.Context(), request.RoomID, request.PlayerID, request.GameType)
	if err != nil {
		reason := recoverRoomFailureReason(err)
		observability.Logger().WarnContext(r.Context(), "room recovery failed",
			"room_id", request.RoomID,
			"player_id", request.PlayerID,
			"event_type", "room_recovery_failed",
			"game_type", request.GameType,
			"reason", reason,
			"error", err,
		)
		writeErrorResponse(w, recoverRoomStatus(err), err.Error())
		return
	}

	writeSuccessResponse(w, http.StatusOK, struct {
		PlayerID   string              `json:"player_id"`
		PlayerMark string              `json:"player_mark"`
		Room       dto.RoomSnapshotDTO `json:"room"`
	}{
		PlayerID:   player.ID,
		PlayerMark: player.Mark,
		Room:       dto.FromRoomSnapshot(snapshot),
	})

	observability.Logger().InfoContext(r.Context(), "room recovery succeeded",
		"room_id", request.RoomID,
		"player_id", request.PlayerID,
		"event_type", "room_recovery_succeeded",
		"game_type", request.GameType,
		"room_state", snapshot.RoomState,
		"is_ai_enabled", snapshot.IsAIEnabled,
		"player_session", player.Session,
	)
}

func addPlayer(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {

	var request struct {
		PlayerID string `json:"player_id"`
	}

	if err := decodeRequestBody(w, r, &request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	player, err := gameService.AddPlayer(request.PlayerID)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	sessionToken, err := gameService.SessionTokenForPlayer(player.ID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Unable to create player")
		return
	}

	writeSuccessResponse(w, http.StatusCreated, struct {
		dto.PlayerDTO
		SessionToken string `json:"session_token"`
	}{
		PlayerDTO:    dto.FromPlayerSnapshot(player),
		SessionToken: sessionToken,
	})
}

const maxHTTPRequestBodyBytes int64 = 1 << 20

var roomIDPattern = regexp.MustCompile(`^[A-Z0-9]{7}$`)

func decodeRequestBody(w http.ResponseWriter, r *http.Request, destination interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxHTTPRequestBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeSuccessResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func validRoomID(roomID string) bool {
	return roomIDPattern.MatchString(roomID)
}

func authenticateHTTPPlayer(w http.ResponseWriter, r *http.Request, gameService *service.GameService, playerID string) bool {
	token, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}

	if _, err := gameService.AuthenticatePlayer(playerID, token); err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}

	return true
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}

func writeSuccessResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	writeJSONResponse(w, statusCode, Response{
		Success: true,
		Data:    data,
		Error:   nil,
	})
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	writeJSONResponse(w, statusCode, Response{
		Success: false,
		Data:    map[string]interface{}{},
		Error:   &message,
	})
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, response Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func createRoomStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrPlayerNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrGameTypeRequired):
		return http.StatusBadRequest
	case errors.Is(err, game.ErrChessAIUnavailable), errors.Is(err, service.ErrMaxAIRoomsReached):
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadRequest
	}
}

func joinRoomStatus(err error) int {
	switch err {
	case service.ErrGameTypeMismatch:
		return http.StatusBadRequest
	case service.ErrPlayerNotFound:
		return http.StatusNotFound
	case game.ErrInvalidGameState:
		return http.StatusBadRequest
	default:
		return http.StatusNotFound
	}
}

func recoverRoomStatus(err error) int {
	switch err {
	case service.ErrGameTypeRequired, service.ErrGameTypeMismatch:
		return http.StatusBadRequest
	case service.ErrPlayerNotFound, service.ErrRoomNotFound:
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func recoverRoomFailureReason(err error) string {
	switch {
	case errors.Is(err, service.ErrRoomNotFound):
		return "room_not_found"
	case errors.Is(err, service.ErrPlayerNotFound):
		return "player_not_found"
	case errors.Is(err, service.ErrGameTypeMismatch), errors.Is(err, service.ErrGameTypeRequired):
		return "invalid_session"
	default:
		return "recovery_failed"
	}
}
