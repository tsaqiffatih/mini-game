package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		wantIP     string
	}{
		{
			name:       "remote addr with port",
			remoteAddr: "192.168.1.100:12345",
			wantIP:     "192.168.1.100",
		},
		{
			name:       "remote addr without port fallback",
			remoteAddr: "192.168.1.100",
			wantIP:     "192.168.1.100",
		},
		{
			name:       "x-forwarded-for single",
			remoteAddr: "127.0.0.1:8080",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195",
			},
			wantIP: "203.0.113.195",
		},
		{
			name:       "x-forwarded-for multiple takes first",
			remoteAddr: "127.0.0.1:8080",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178",
			},
			wantIP: "203.0.113.195",
		},
		{
			name:       "x-real-ip header",
			remoteAddr: "127.0.0.1:8080",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.17",
			},
			wantIP: "198.51.100.17",
		},
		{
			name:       "invalid header falls back to remote addr",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "invalid-ip",
			},
			wantIP: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := ClientIP(req)
			if got != tt.wantIP {
				t.Fatalf("ClientIP() = %q, want %q", got, tt.wantIP)
			}
		})
	}
}

func TestRateLimiter_BypassesHealthEndpoints(t *testing.T) {
	handler := RateLimiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/healthz"} {
		// Send multiple requests that would exceed burst
		for i := 0; i < 40; i++ {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.RemoteAddr = "1.2.3.4:1234"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("request %d to %s returned code %d, want %d", i, path, rec.Code, http.StatusOK)
			}
		}
	}
}
