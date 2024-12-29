package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/game"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var mu sync.Mutex

func RegisterRouter(r *mux.Router, roomManager *game.RoomManager, playerManager *game.PlayerManager) {
	r.HandleFunc("/create/user", func(w http.ResponseWriter, r *http.Request) {
		addPlayer(w, r, playerManager)
	}).Methods("POST")

	r.HandleFunc("/room/join", func(w http.ResponseWriter, r *http.Request) {
		joinRoom(w, r, roomManager, playerManager)
	}).Methods("POST")

	r.HandleFunc("/room/create", func(w http.ResponseWriter, r *http.Request) {
		createRoom(w, r, roomManager, playerManager)
	}).Methods("POST")

	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		HandleWebSocket(w, r, roomManager, playerManager)
	})

}

func createRoom(w http.ResponseWriter, r *http.Request, roomManager *game.RoomManager, playerManager *game.PlayerManager) {
	mu.Lock()
	defer mu.Unlock()

	var request struct {
		GameType string `json:"game_type"`
		PlayerID string `json:"player_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := Response{
			Success: false,
			Message: "Invalid request",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validasi input request
	if request.GameType == "" {
		response := Response{
			Success: false,
			Message: "RoomID and GameType are required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	player, err := playerManager.GetPlayer(request.PlayerID)
	if err != nil {
		response := Response{
			Success: false,
			Message: "Player not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	roomId := roomManager.GenerateRandomRoomCode()

	fmt.Println("Room created handlers:", roomId)

	room, err := roomManager.CreateRoom(roomId, request.GameType)
	if err != nil {
		response := Response{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	res, err := roomManager.JoinRoom(room.RoomID, player)
	if err != nil {
		response := Response{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Success: true,
		Message: "Room created successfully",
		Data:    res,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func joinRoom(w http.ResponseWriter, r *http.Request, roomManager *game.RoomManager, playerManager *game.PlayerManager) {
	mu.Lock()
	defer mu.Unlock()

	var request struct {
		RoomID   string `json:"room_id"`
		PlayerID string `json:"player_id"`
		GameType string `json:"game_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := Response{
			Success: false,
			Message: "Invalid request",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	player, err := playerManager.GetPlayer(request.PlayerID)
	if err != nil {
		response := Response{
			Success: false,
			Message: "Player not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	player.LastActive = time.Now()

	room, err := roomManager.GetRoomByID(request.RoomID)
	if err != nil {
		response := Response{
			Success: false,
			Message: "Room not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	if room.GameState.GameType != request.GameType {
		response := Response{
			Success: false,
			Message: "Game type not match",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	res, err := roomManager.JoinRoom(room.RoomID, player)
	if err != nil {
		response := Response{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Success: true,
		Message: "Player joined room successfully",
		Data:    res,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func addPlayer(w http.ResponseWriter, r *http.Request, playerManager *game.PlayerManager) {
	mu.Lock()
	defer mu.Unlock()

	var request struct {
		PlayerID string `json:"player_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		response := Response{
			Success: false,
			Message: "Invalid request",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	player, err := playerManager.AddPlayer(request.PlayerID)
	if err != nil {
		response := Response{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := Response{
		Success: true,
		Message: "Success registering player",
		Data:    player,
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
