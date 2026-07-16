package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"ticket-system/internal/auth"
)

type contextKey string

const (
	UserIDKey   contextKey = "user_id"
	UsernameKey contextKey = "username"
)

// UserIDFromContext retrieves the authenticated user's ID from the request context.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(UserIDKey).(int64)
	return id, ok
}

// UsernameFromContext retrieves the authenticated user's username from the request context.
func UsernameFromContext(ctx context.Context) (string, bool) {
	name, ok := ctx.Value(UsernameKey).(string)
	return name, ok
}

// AuthMiddleware creates a middleware that protects endpoints using JWT authentication.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondWithError(w, http.StatusUnauthorized, "Missing Authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				respondWithError(w, http.StatusUnauthorized, "Invalid Authorization header format. Must be Bearer <token>")
				return
			}

			tokenStr := parts[1]
			claims, err := auth.ValidateToken(tokenStr, jwtSecret)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UsernameKey, claims.Username)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
