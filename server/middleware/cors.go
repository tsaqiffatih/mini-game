package middleware

import (
	"log"
	"os"

	"github.com/gorilla/handlers"
)

func CORSAllowedOrigins() handlers.CORSOption {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	log.Println("Allowed Origins: ", allowedOrigins)
	// if allowedOrigins == "" {
	// 	allowedOrigins = "http://localhost:5173"
	// }
	return handlers.AllowedOrigins([]string{allowedOrigins})
}

func CORSAllowedHeaders() handlers.CORSOption {
	return handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
}

func CORSAllowedMethods() handlers.CORSOption {
	return handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
}
