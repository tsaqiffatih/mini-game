package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/api/dto"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/service"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *string     `json:"error"`
	Message string      `json:"-"`
}

func RegisterRouter(r *mux.Router, clients *ClientRegistry, gameService *service.GameService) {
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

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(w, r, clients, gameService)
	})

}

func createRoom(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {

	var request struct {
		GameType string `json:"game_type"`
		PlayerID string `json:"player_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
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
		AILevel  int    `json:"ai_level,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	res, err := gameService.CreateRoomWithAILevelWithContext(r.Context(), request.GameType, request.PlayerID, request.AILevel)
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

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	res, err := gameService.JoinRoomWithContext(r.Context(), request.RoomID, request.PlayerID, request.GameType)
	if err != nil {
		writeErrorResponse(w, joinRoomStatus(err), err.Error())
		return
	}

	writeSuccessResponse(w, http.StatusOK, dto.FromJoinRoomResponse(res))
}

func addPlayer(w http.ResponseWriter, r *http.Request, gameService *service.GameService) {

	var request struct {
		PlayerID string `json:"player_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request")
		return
	}

	player, err := gameService.AddPlayer(request.PlayerID)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	writeSuccessResponse(w, http.StatusCreated, dto.FromPlayerSnapshot(player))
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
	switch err {
	case service.ErrPlayerNotFound:
		return http.StatusNotFound
	case service.ErrGameTypeRequired:
		return http.StatusBadRequest
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
