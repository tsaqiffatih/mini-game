package middleware

import (
	"net/http"
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
)

func getClient(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	client, exists := clients[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 5)
		clients[ip] = &Client{limiter, time.Now()}
		return limiter
	}

	client.lastSeen = time.Now()
	return client.limiter
}

func cleanupClients() {
	for {
		time.Sleep(time.Minute)

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

func RateLimiter(next http.Handler) http.Handler {
	go cleanupClients()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		limiter := getClient(ip)

		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
