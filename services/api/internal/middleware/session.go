package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/wick/github-star-manager/services/api/internal/response"
)

type sessionValidator interface {
	ValidateSessionToken(ctx context.Context, token string) (int64, error)
}

type contextKey string

const userIDContextKey contextKey = "userID"

func RequireSession(validator sessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing authorization")
				return
			}
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token == "" {
				response.Error(w, http.StatusUnauthorized, "invalid authorization")
				return
			}

			userID, err := validator.ValidateSessionToken(r.Context(), token)
			if err != nil || userID <= 0 {
				response.Error(w, http.StatusUnauthorized, "invalid session")
				return
			}

			ctx := context.WithValue(r.Context(), userIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	value := ctx.Value(userIDContextKey)
	if value == nil {
		return 0, false
	}
	userID, ok := value.(int64)
	if !ok || userID <= 0 {
		return 0, false
	}
	return userID, true
}
