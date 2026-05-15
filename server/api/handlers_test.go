package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/infrastructure"
	"github.com/tsaqiffatih/mini-game/service"
	"github.com/tsaqiffatih/mini-game/tictactoe"
)

type apiTestServer struct {
	router  *mux.Router
	service *service.GameService
}

type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type joinRoomAPIData struct {
	PlayerID   string       `json:"player_id"`
	PlayerMark string       `json:"player_mark"`
	Room       game.RoomDTO `json:"room"`
}

type moveAPIResponse struct {
	Board  [3][3]string         `json:"board"`
	Turn   string               `json:"turn"`
	Winner string               `json:"winner"`
	Status tictactoe.GameStatus `json:"status"`
	Ended  bool                 `json:"ended"`
}

func newAPITestServer() apiTestServer {
	roomRepository := infrastructure.NewMemoryRoomRepository()
	playerManager := game.NewPlayerManager()
	gameService := service.NewGameService(roomRepository, playerManager)

	router := mux.NewRouter()
	RegisterRouter(router, NewClientRegistry(), gameService)
	registerAPITestAliases(router, gameService)

	return apiTestServer{
		router:  router,
		service: gameService,
	}
}

func registerAPITestAliases(router *mux.Router, gameService *service.GameService) {
	router.HandleFunc("/rooms", func(w http.ResponseWriter, r *http.Request) {
		createRoom(w, r, gameService)
	}).Methods(http.MethodPost)

	router.HandleFunc("/rooms/{room_id}/join", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			PlayerID string `json:"player_id"`
			GameType string `json:"game_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			sendErrorResponse(w, "Invalid request", http.StatusBadRequest)
			return
		}

		res, err := gameService.JoinRoom(mux.Vars(r)["room_id"], request.PlayerID, request.GameType)
		if err != nil {
			sendErrorResponse(w, err.Error(), joinRoomStatus(err))
			return
		}

		response := Response{
			Success: true,
			Message: "Player joined room successfully",
			Data:    res,
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}).Methods(http.MethodPost)

	router.HandleFunc("/rooms/{room_id}/move", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			PlayerID string `json:"player_id"`
			Row      int    `json:"row"`
			Col      int    `json:"col"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			sendErrorResponse(w, "Invalid request", http.StatusBadRequest)
			return
		}

		result, err := gameService.HandleTicTacToeMove(
			mux.Vars(r)["room_id"],
			request.PlayerID,
			request.Row,
			request.Col,
		)
		if err != nil {
			sendErrorResponse(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := Response{
			Success: true,
			Message: "Move applied successfully",
			Data: moveAPIResponse{
				Board:  result.State.Board,
				Turn:   result.State.Turn,
				Winner: result.State.Winner,
				Status: result.State.Status,
				Ended:  result.GameEnded,
			},
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}).Methods(http.MethodPost)
}

func registerPlayerViaAPI(t *testing.T, server apiTestServer, playerID string) {
	t.Helper()

	recorder := doJSONRequest(t, server.router, http.MethodPost, "/create/user", map[string]string{
		"player_id": playerID,
	})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("register player status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response apiResponse
	decodeJSONResponse(t, recorder, &response)
	if !response.Success {
		t.Fatalf("register response success = false, body=%s", recorder.Body.String())
	}
}

func createRoomViaAPI(t *testing.T, server apiTestServer, playerID string) joinRoomAPIData {
	t.Helper()

	recorder := doJSONRequest(t, server.router, http.MethodPost, "/rooms", map[string]string{
		"game_type": "tictactoe",
		"player_id": playerID,
	})
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create room status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response apiResponse
	decodeJSONResponse(t, recorder, &response)
	if !response.Success {
		t.Fatalf("create room response success = false, body=%s", recorder.Body.String())
	}

	var data joinRoomAPIData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("decode create room data: %v", err)
	}
	if data.Room.RoomID == "" {
		t.Fatalf("room_id is empty")
	}

	return data
}

func joinRoomViaAPI(t *testing.T, server apiTestServer, roomID string, playerID string) joinRoomAPIData {
	t.Helper()

	recorder := doJSONRequest(t, server.router, http.MethodPost, "/rooms/"+roomID+"/join", map[string]string{
		"game_type": "tictactoe",
		"player_id": playerID,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("join room status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response apiResponse
	decodeJSONResponse(t, recorder, &response)
	if !response.Success {
		t.Fatalf("join room response success = false, body=%s", recorder.Body.String())
	}

	var data joinRoomAPIData
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("decode join room data: %v", err)
	}

	return data
}

func moveViaAPI(t *testing.T, server apiTestServer, roomID string, playerID string, row int, col int) moveAPIResponse {
	t.Helper()

	recorder := doJSONRequest(t, server.router, http.MethodPost, "/rooms/"+roomID+"/move", map[string]interface{}{
		"player_id": playerID,
		"row":       row,
		"col":       col,
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("move status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response apiResponse
	decodeJSONResponse(t, recorder, &response)
	if !response.Success {
		t.Fatalf("move response success = false, body=%s", recorder.Body.String())
	}

	var data moveAPIResponse
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("decode move data: %v", err)
	}

	return data
}

func doJSONRequest(t *testing.T, handler http.Handler, method string, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	encodedBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}

	request := httptest.NewRequest(method, path, bytes.NewReader(encodedBody))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, target interface{}) {
	t.Helper()

	if err := json.NewDecoder(recorder.Body).Decode(target); err != nil {
		t.Fatalf("decode response JSON: %v; body=%s", err, recorder.Body.String())
	}
}

func TestCreateRoomAPI(t *testing.T) {
	server := newAPITestServer()
	registerPlayerViaAPI(t, server, "p1")

	data := createRoomViaAPI(t, server, "p1")

	if data.PlayerID != "p1" {
		t.Fatalf("player_id = %q, want p1", data.PlayerID)
	}
	if data.PlayerMark == "" {
		t.Fatalf("player_mark is empty")
	}
	if data.Room.RoomID == "" {
		t.Fatalf("room_id is empty")
	}
}

func TestJoinRoomAPI(t *testing.T) {
	server := newAPITestServer()
	registerPlayerViaAPI(t, server, "p1")
	registerPlayerViaAPI(t, server, "p2")
	created := createRoomViaAPI(t, server, "p1")

	joined := joinRoomViaAPI(t, server, created.Room.RoomID, "p2")

	if joined.PlayerID != "p2" {
		t.Fatalf("joined player_id = %q, want p2", joined.PlayerID)
	}

	snapshot, err := server.service.RoomSnapshot(created.Room.RoomID)
	if err != nil {
		t.Fatalf("RoomSnapshot() error = %v", err)
	}
	if len(snapshot.Players) != 2 {
		t.Fatalf("players len = %d, want 2", len(snapshot.Players))
	}
}

func TestTicTacToeMoveAPI(t *testing.T) {
	server := newAPITestServer()
	registerPlayerViaAPI(t, server, "p1")
	registerPlayerViaAPI(t, server, "p2")
	created := createRoomViaAPI(t, server, "p1")
	joinRoomViaAPI(t, server, created.Room.RoomID, "p2")

	data := moveViaAPI(t, server, created.Room.RoomID, "p1", 0, 0)

	if data.Board[0][0] == "" {
		t.Fatalf("board[0][0] is empty, want move applied")
	}
	if data.Turn != "O" {
		t.Fatalf("turn = %q, want O", data.Turn)
	}
}

func TestFullGameFlowAPI(t *testing.T) {
	server := newAPITestServer()
	registerPlayerViaAPI(t, server, "p1")
	registerPlayerViaAPI(t, server, "p2")
	created := createRoomViaAPI(t, server, "p1")
	joinRoomViaAPI(t, server, created.Room.RoomID, "p2")

	moveViaAPI(t, server, created.Room.RoomID, "p1", 0, 0)
	moveViaAPI(t, server, created.Room.RoomID, "p2", 1, 0)
	moveViaAPI(t, server, created.Room.RoomID, "p1", 0, 1)
	moveViaAPI(t, server, created.Room.RoomID, "p2", 1, 1)
	result := moveViaAPI(t, server, created.Room.RoomID, "p1", 0, 2)

	if !result.Ended {
		t.Fatalf("ended = false, want true")
	}
	if result.Winner == "" {
		t.Fatalf("winner is empty")
	}
}
