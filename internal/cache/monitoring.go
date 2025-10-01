package cache

import (
	"context"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/princekumarofficial/stories-service/internal/utils/response"
)

// CacheStats represents cache performance statistics
type CacheStats struct {
	RedisConnected bool              `json:"redis_connected"`
	RedisInfo      map[string]string `json:"redis_info"`
	CacheKeys      []string          `json:"cache_keys_sample"`
	KeyCount       int               `json:"total_keys"`
}

// GetCacheStats returns cache performance statistics
func GetCacheStats(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		stats := CacheStats{
			RedisConnected: true,
			RedisInfo:      make(map[string]string),
		}

		// Test Redis connection
		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			stats.RedisConnected = false
			response.WriteJSON(w, http.StatusOK, response.RequestOK("Cache stats retrieved", stats))
			return
		}

		// Get Redis INFO
		infoResult := redisClient.Info(ctx, "memory", "stats")
		if infoResult.Err() == nil {
			// Parse basic info
			stats.RedisInfo["info"] = "available"
		}

		// Get cache keys (sample)
		keys := redisClient.Keys(ctx, "user:*")
		if keys.Err() == nil {
			stats.CacheKeys = keys.Val()
			if len(stats.CacheKeys) > 10 {
				stats.CacheKeys = stats.CacheKeys[:10] // Show only first 10
			}
		}

		// Get total key count
		dbSize := redisClient.DBSize(ctx)
		if dbSize.Err() == nil {
			stats.KeyCount = int(dbSize.Val())
		}

		response.WriteJSON(w, http.StatusOK, response.RequestOK("Cache stats retrieved", stats))
	}
}

// ClearCache endpoint for administrative purposes
func ClearCache(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		// Get cache type from query parameter
		cacheType := r.URL.Query().Get("type")

		var pattern string
		switch cacheType {
		case "feed":
			pattern = "feed:*"
		case "followees":
			pattern = "user:followees:*"
		case "stats":
			pattern = "user:stats:*"
		case "stories":
			pattern = "story:*"
		case "all":
			pattern = "*"
		default:
			pattern = "feed:*" // Default to feed cache
		}

		// Delete matching keys
		keys := redisClient.Keys(ctx, pattern)
		if keys.Err() != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(keys.Err()))
			return
		}

		if len(keys.Val()) > 0 {
			deleted := redisClient.Del(ctx, keys.Val()...)
			if deleted.Err() != nil {
				response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(deleted.Err()))
				return
			}

			result := map[string]interface{}{
				"pattern":      pattern,
				"deleted_keys": deleted.Val(),
				"keys_sample":  keys.Val()[:min(len(keys.Val()), 5)], // Show first 5 deleted keys
			}
			response.WriteJSON(w, http.StatusOK, response.RequestOK("Cache cleared successfully", result))
		} else {
			result := map[string]interface{}{
				"pattern":      pattern,
				"deleted_keys": 0,
				"message":      "No keys found matching pattern",
			}
			response.WriteJSON(w, http.StatusOK, response.RequestOK("No cache keys to clear", result))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
