package middleware

import (
	"context"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type Client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*Client)
	defaultRPS   = 20
	defaultBurst = 40
)

// getClient retrieves or creates a rate limiter for a client identified by a generic identifier (e.g., IP, token, or user ID).
func getClient(identifier string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	client, exists := clients[identifier]
	if !exists {
		// Read limits from env (fallback to defaults)
		rps := defaultRPS
		if v, ok := os.LookupEnv("RATE_LIMIT_RPS"); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				rps = n
			}
		}
		burst := defaultBurst
		if v, ok := os.LookupEnv("RATE_LIMIT_BURST"); ok {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				burst = n
			}
		}
		limiter := rate.NewLimiter(rate.Limit(rps), burst)
		clients[identifier] = &Client{limiter, time.Now()}
		return limiter
	}


	client.lastSeen = time.Now()
	return client.limiter
}

// cleanupClients removes stale clients that haven't been seen for a while.
func cleanupClients(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		var staleClients []string

		mu.Lock()
		for ip, client := range clients {
			if time.Since(client.lastSeen) > 3*time.Minute {
				staleClients = append(staleClients, ip)
			}
		}
		mu.Unlock()

		if len(staleClients) > 0 {
			mu.Lock()
			for _, ip := range staleClients {
				delete(clients, ip)
			}
			mu.Unlock()
		}
	}
}

func StartRateLimiterCleanup(ctx context.Context) {
	go cleanupClients(ctx)
}

// ClientIP extracts the real client IP, considering proxy headers like X-Forwarded-For and X-Real-IP.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for _, part := range parts {
			ip := strings.TrimSpace(part)
			if parsed := net.ParseIP(ip); parsed != nil {
				return ip
			}
		}
	}

	if xRealIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); xRealIP != "" {
		if parsed := net.ParseIP(xRealIP); parsed != nil {
			return xRealIP
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not rate limit health checks
		if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		// Determine client identifier: prefer Bearer token, fallback to IP
	identifier := func() string {
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, prefix) {
			token := strings.TrimSpace(strings.TrimPrefix(auth, prefix))
			if token != "" {
				return token
			}
		}
		return ClientIP(r)
	}()
	limiter := getClient(identifier)

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
