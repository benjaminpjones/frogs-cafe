package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"frogs_cafe/auth"
)

type contextKey string

const PlayerIDKey contextKey = "playerID"

// RequireAuth middleware validates session and adds player ID to context
func RequireAuth(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header or query parameter
			authHeader := r.Header.Get("Authorization")
			token := r.URL.Query().Get("token")

			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			if token == "" {
				http.Error(w, "Unauthorized: No token provided", http.StatusUnauthorized)
				return
			}

			// Validate session
			playerID, _, err := auth.ValidateSession(db, token)
			if err != nil {
				http.Error(w, "Unauthorized: Invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Add player ID to context
			ctx := context.WithValue(r.Context(), PlayerIDKey, playerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetPlayerID retrieves player ID from request context
func GetPlayerID(r *http.Request) (int, bool) {
	playerID, ok := r.Context().Value(PlayerIDKey).(int)
	return playerID, ok
}
