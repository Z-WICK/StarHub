package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wick/github-star-manager/services/api/internal/response"
)

func CORS(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func Auth(appSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing authorization")
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != appSecret {
				response.Error(w, http.StatusUnauthorized, "invalid authorization token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RateLimit(perMinute int) func(http.Handler) http.Handler {
	type clientState struct {
		count       int
		windowStart time.Time
	}

	states := make(map[string]clientState)
	var mu sync.Mutex

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for now := range ticker.C {
			mu.Lock()
			for clientID, state := range states {
				if now.Sub(state.windowStart) > 10*time.Minute {
					delete(states, clientID)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID := clientIP(r)
			now := time.Now()

			mu.Lock()
			state := states[clientID]
			if state.windowStart.IsZero() || now.Sub(state.windowStart) >= time.Minute {
				state = clientState{count: 0, windowStart: now}
			}
			state.count++
			states[clientID] = state
			exceeded := state.count > perMinute
			mu.Unlock()

			if exceeded {
				response.Error(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			candidate := strings.TrimSpace(parts[0])
			if candidate != "" {
				return candidate
			}
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && ip != "" {
		return ip
	}
	if r.RemoteAddr == "" {
		return "unknown"
	}
	return r.RemoteAddr
}
