package main

import (
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/api"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/utils"
)

func main() {

	playerManager := game.NewPlayerManager()
	roomManager := game.NewRoomManager()

	r := mux.NewRouter()

	api.RegisterRouter(r, roomManager, playerManager)

	corsHandler := handlers.CORS(
		utils.CORSAllowedHeaders(),
		utils.CORSAllowedMethods(),
		utils.CORSAllowedOrigins(),
	)

	go roomManager.RemoveInactivePlayers()
	// {"type": "makeMove", "payload": {"room_id": "SOVPUAD", "player_id": "fatih", "row": 1, "col": 1}}

	//   {"action": "MAKE_MOVE", "message": "{"room_id":"8TFTXLJ","player_id":"player1","row":1,"col":1}}
	// {"action": "MAKE_MOVE","message": {"room_id": "4B9L392","player_id": "player1","row": 1,"col": 1}}

	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", corsHandler(r)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
