package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/princekumarofficial/stories-service/internal/ratelimit"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

type RateLimitConfig struct {
	redisClient *redis.Client
	limiters    map[string]*ratelimit.TokenBucket
}

func NewRateLimitConfig(redisClient *redis.Client) *RateLimitConfig {
	config := &RateLimitConfig{
		redisClient: redisClient,
		limiters:    make(map[string]*ratelimit.TokenBucket),
	}

	// Configure rate limits for different actions
	// POST /stories: 20/min per user
	config.limiters["stories"] = ratelimit.NewTokenBucket(redisClient, 20, 20)

	// POST /reactions: 60/min per user
	config.limiters["reactions"] = ratelimit.NewTokenBucket(redisClient, 60, 60)

	return config
}

func (rlc *RateLimitConfig) RateLimitMiddleware(action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (assumes auth middleware ran first)
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(
					errors.New("user not authenticated")))
				return
			}

			// Get the appropriate rate limiter
			limiter, exists := rlc.limiters[action]
			if !exists {
				// If no rate limiter configured for this action, allow the request
				next.ServeHTTP(w, r)
				return
			}

			// Check if user is allowed to perform this action
			allowed, err := limiter.Allow(r.Context(), userID, action)
			if err != nil {
				response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(
					fmt.Errorf("rate limit check failed: %w", err)))
				return
			}

			if !allowed {
				// Get remaining tokens for rate limit headers
				remaining, _ := limiter.GetRemaining(r.Context(), userID, action)

				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit", getLimitForAction(action))
				w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
				w.Header().Set("X-RateLimit-Reset", "60") // Reset in 60 seconds (1 minute window)

				response.WriteJSON(w, http.StatusTooManyRequests, response.GeneralError(
					errors.New("rate limit exceeded")))
				return
			}

			// Get remaining tokens for response headers
			remaining, _ := limiter.GetRemaining(r.Context(), userID, action)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", getLimitForAction(action))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			w.Header().Set("X-RateLimit-Reset", "60")

			// Allow the request to proceed
			next.ServeHTTP(w, r)
		})
	}
}

// Helper function to get the limit for display in headers
func getLimitForAction(action string) string {
	switch action {
	case "stories":
		return "20"
	case "reactions":
		return "60"
	default:
		return "100" // default fallback
	}
}

// RateLimitedHandler wraps a handler with rate limiting for a specific action
func (rlc *RateLimitConfig) RateLimitedHandler(action string, handler http.HandlerFunc) http.Handler {
	return rlc.RateLimitMiddleware(action)(http.HandlerFunc(handler))
}
