package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/api"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/middleware"
)

func main() {

	playerManager := game.NewPlayerManager()
	roomManager := game.NewRoomManager()

	r := mux.NewRouter()

	api.RegisterRouter(r, roomManager, playerManager)

	corsHandler := handlers.CORS(
		middleware.CORSAllowedHeaders(),
		middleware.CORSAllowedMethods(),
		middleware.CORSAllowedOrigins(),
	)

	tickerInterval := 30 * time.Minute
	duration := 24 * time.Hour

	go playerManager.RemoveInactivePlayers(duration, tickerInterval)
	go roomManager.RemoveInactivePlayersFromRoom(duration, tickerInterval)

	r.Use(middleware.RateLimiter)

	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", corsHandler(r)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
