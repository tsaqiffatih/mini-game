package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/tsaqiffatih/mini-game/api"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/middleware"
)

func main() {

	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatalf("Error loading .env file")
	// }

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	playerManager := game.NewPlayerManager()
	roomManager := game.NewRoomManager()

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := api.Response{
			Success: true,
			Message: "Hello World from Mini Game",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

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
	if err := http.ListenAndServe(":"+port, corsHandler(r)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
