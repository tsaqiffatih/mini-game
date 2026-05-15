package middleware

import (
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/tsaqiffatih/mini-game/internal/observability"
)

// CORSAllowedOrigins sets the allowed origins for CORS requests.
// It reads the allowed origins from environment variables.
func CORSAllowedOrigins() handlers.CORSOption {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	origins := strings.Split(allowedOrigins, ",")
	observability.Logger().Info("cors configured",
		"room_id", "",
		"player_id", "",
		"event_type", "cors_configured",
		"allowed_origins", allowedOrigins,
	)
	// if allowedOrigins == "" {
	// 	allowedOrigins = "http://localhost:5173"
	// }
	return handlers.AllowedOrigins(origins)
}

// CORSAllowedHeaders sets the allowed headers for CORS requests.
// It specifies headers like Content-Type and Authorization.
func CORSAllowedHeaders() handlers.CORSOption {
	return handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
}

// CORSAllowedMethods sets the allowed HTTP methods for CORS requests.
// It specifies methods like GET, POST, PUT, and DELETE.
func CORSAllowedMethods() handlers.CORSOption {
	return handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
}
