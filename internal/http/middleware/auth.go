package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/princekumarofficial/stories-service/internal/utils/jwt"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware creates a middleware that validates JWT tokens and extracts user ID
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(
					errors.New("Authorization header required")))
				return
			}

			// Check if the header starts with "Bearer "
			if !strings.HasPrefix(authHeader, "Bearer ") {
				response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(
					errors.New("Invalid authorization header format")))
				return
			}

			// Extract the token
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(
					errors.New("Token not provided")))
				return
			}

			// Extract user ID from token
			userID, err := jwt.ExtractUserIDFromToken(token, jwtSecret)
			if err != nil {
				response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(
					errors.New("Invalid token")))
				return
			}

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}
