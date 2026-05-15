package middleware

import (
	"log"
	"os"

	"github.com/gorilla/handlers"
)

// CORSAllowedOrigins sets the allowed origins for CORS requests.
// It reads the allowed origins from environment variables.
func CORSAllowedOrigins() handlers.CORSOption {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	log.Println("Allowed Origins: ", allowedOrigins)
	// if allowedOrigins == "" {
	// 	allowedOrigins = "http://localhost:5173"
	// }
	return handlers.AllowedOrigins([]string{allowedOrigins})
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
