package middleware

import (
    "net/http"
    "github.com/tsaqiffatih/mini-game/service"
)

// VerifySessionToken validates the provided session token for a given player ID using the GameService.
// Returns true if the token is valid and matches the player's current session token.
func VerifySessionToken(gameService *service.GameService, playerID, token string) bool {
    if token == "" {
        return false
    }
    // GameService.AuthenticatePlayer returns an error if authentication fails.
    if _, err := gameService.AuthenticatePlayer(playerID, token); err != nil {
        return false
    }
    return true
}

// SessionAuthMiddleware returns an HTTP middleware that extracts the Authorization header and a player_id
// (expected to be present as a query parameter or form value) and validates the session token.
// If authentication fails, it responds with 401 Unauthorized and aborts the request chain.
func SessionAuthMiddleware(gameService *service.GameService) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Retrieve player ID from request (query, form, or JSON body is not parsed here).
            playerID := r.URL.Query().Get("player_id")
            if playerID == "" {
                // No player ID, reject.
                w.WriteHeader(http.StatusUnauthorized)
                w.Write([]byte(`{"error":"player_id missing"}`))
                return
            }
            // Extract Bearer token.
            tokenHeader := r.Header.Get("Authorization")
            const prefix = "Bearer "
            if len(tokenHeader) < len(prefix) || tokenHeader[:len(prefix)] != prefix {
                w.WriteHeader(http.StatusUnauthorized)
                w.Write([]byte(`{"error":"missing or invalid Authorization header"}`))
                return
            }
            token := tokenHeader[len(prefix):]
            if !VerifySessionToken(gameService, playerID, token) {
                w.WriteHeader(http.StatusUnauthorized)
                w.Write([]byte(`{"error":"unauthorized"}`))
                return
            }
            // Token valid, proceed.
            next.ServeHTTP(w, r)
        })
    }
}
