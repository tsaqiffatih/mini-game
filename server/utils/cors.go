package utils

import "github.com/gorilla/handlers"

func CORSAllowedOrigins() handlers.CORSOption {
	return handlers.AllowedOrigins([]string{"http://localhost:5173"})
}

func CORSAllowedHeaders() handlers.CORSOption {
	return handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
}

func CORSAllowedMethods() handlers.CORSOption {
	return handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"})
}
