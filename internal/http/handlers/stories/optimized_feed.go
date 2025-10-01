package stories

import (
	"errors"
	"net/http"

	"github.com/princekumarofficial/stories-service/internal/cache"
	"github.com/princekumarofficial/stories-service/internal/http/middleware"
	"github.com/princekumarofficial/stories-service/internal/storage"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

// OptimizedFeed handles the optimized stories feed endpoint with caching and N+1 avoidance
// @Summary Get optimized stories feed
// @Description Get stories feed with caching and preloaded metadata to avoid N+1 queries
// @Tags stories
// @Security BearerAuth
// @Success 200 {object} response.Response "Optimized feed retrieved successfully"
// @Failure 401 {object} response.Response "Unauthorized"
// @Failure 500 {object} response.Response "Internal server error"
// @Router /feed/optimized [get]
func OptimizedFeed(cacheService *cache.CacheService, optimizedQuery *cache.OptimizedFeedQuery) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		// First try to get cached feed
		cachedStories, err := cacheService.GetCachedFeed(r.Context(), userID)
		if err == nil && len(cachedStories) > 0 {
			response.WriteJSON(w, http.StatusOK, response.RequestOK("Cached feed retrieved successfully", cachedStories))
			return
		}

		// Cache miss or empty - fetch optimized feed with all metadata
		optimizedStories, err := optimizedQuery.GetOptimizedFeedForUser(r.Context(), userID)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Optimized feed retrieved successfully", optimizedStories))
	}
}

func CachedFeed(cacheService storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from context
		userID, ok := middleware.GetUserIDFromContext(r.Context())
		if !ok {
			response.WriteJSON(w, http.StatusUnauthorized, response.GeneralError(errors.New("user not authenticated")))
			return
		}

		// This will use the cache service which automatically handles caching
		stories, err := cacheService.GetStoriesForUser(userID)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Cached feed retrieved successfully", stories))
	}
}
