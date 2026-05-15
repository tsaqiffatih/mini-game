package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tsaqiffatih/mini-game/api"
	"github.com/tsaqiffatih/mini-game/game"
	"github.com/tsaqiffatih/mini-game/infrastructure"
	"github.com/tsaqiffatih/mini-game/internal/observability"
	"github.com/tsaqiffatih/mini-game/middleware"
	"github.com/tsaqiffatih/mini-game/service"
)

func main() {
	observability.Init("mini-game")
	logger := observability.Logger()

	if _, err := os.Stat(".env"); err == nil {
		err := godotenv.Load()
		if err != nil {
			logger.Error("failed to load env file", "event_type", "startup", "error", err)
			os.Exit(1)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	playerManager := game.NewPlayerManager()
	roomRepository := infrastructure.NewMemoryRoomRepository()
	gameService := service.NewGameService(roomRepository, playerManager)
	clients := api.NewClientRegistry()
	gameService.SetRoomNotifier(func(snapshot game.RoomSnapshot) {
		api.NotifyGameUpdateToClients(clients, snapshot)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	gameService.SetContext(ctx)

	middleware.StartRateLimiterCleanup(ctx)

	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response := api.Response{
			Success: true,
			Message: "Hello World from Mini Game",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	api.RegisterRouter(
		r,
		clients,
		gameService,
	)

	corsHandler := handlers.CORS(
		middleware.CORSAllowedHeaders(),
		middleware.CORSAllowedMethods(),
		middleware.CORSAllowedOrigins(),
	)

	tickerInterval := 30 * time.Minute
	duration := 24 * time.Hour

	go playerManager.RemoveInactivePlayers(ctx, duration, tickerInterval)
	go gameService.StartRoomCleanup(ctx, duration, tickerInterval)

	r.Use(observability.RequestMiddleware)
	r.Use(middleware.RateLimiter)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           corsHandler(r),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server starting", "event_type", "startup", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "event_type", "startup", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received", "event_type", "shutdown")

	clients.CloseAll()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", "event_type", "shutdown", "error", err)
	}
	if err := gameService.CleanupRooms(shutdownCtx, 0); err != nil {
		logger.Warn("room cleanup failed during shutdown", "event_type", "shutdown", "error", err)
	}
	logger.Info("shutdown complete", "event_type", "shutdown")
}
