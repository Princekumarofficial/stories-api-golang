package cache

import (
	"context"
	"net/http"
	"strings"

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

// cachePatterns maps the ?type= of /cache/clear to the key patterns it deletes.
// "all" means all CACHE keys: it deliberately leaves rate_limit:* buckets alone,
// so clearing the cache cannot reset everyone's rate limits.
var cachePatterns = map[string][]string{
	"feed":      {"feed:*"},
	"followees": {"user:followees:*"},
	"stats":     {"user:stats:*"},
	"stories":   {"story:*"},
	"all":       {"feed:*", "user:followees:*", "user:stats:*", "story:*"},
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

		// Sample a few cache keys with one SCAN step. KEYS walks the entire
		// keyspace in a single blocking call and stalls every other Redis client.
		sample, _, err := redisClient.Scan(ctx, 0, "user:*", 100).Result()
		if err == nil {
			if len(sample) > 10 {
				sample = sample[:10] // Show only first 10
			}
			stats.CacheKeys = sample
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
		patterns, ok := cachePatterns[r.URL.Query().Get("type")]
		if !ok {
			patterns = cachePatterns["feed"] // Default to feed cache
		}
		pattern := strings.Join(patterns, ",")

		deleted, sample, err := deleteByPatterns(ctx, redisClient, patterns)
		if err != nil {
			response.WriteJSON(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		if deleted > 0 {
			result := map[string]interface{}{
				"pattern":      pattern,
				"deleted_keys": deleted,
				"keys_sample":  sample, // First 5 deleted keys
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

// deleteByPatterns removes keys matching the patterns. It finds them with SCAN,
// in small steps, instead of KEYS (which blocks Redis for the whole walk), and
// frees them with UNLINK in batches. Matching keys are collected before any are
// deleted, so deleting cannot shift the scan cursor and skip keys. It returns
// the number of deleted keys and up to five of their names.
func deleteByPatterns(ctx context.Context, rdb *redis.Client, patterns []string) (int64, []string, error) {
	const batchSize = 500

	var keys []string
	for _, pattern := range patterns {
		iter := rdb.Scan(ctx, 0, pattern, batchSize).Iterator()
		for iter.Next(ctx) {
			keys = append(keys, iter.Val())
		}
		if err := iter.Err(); err != nil {
			return 0, nil, err
		}
	}

	var deleted int64
	for start := 0; start < len(keys); start += batchSize {
		end := start + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		n, err := rdb.Unlink(ctx, keys[start:end]...).Result()
		if err != nil {
			return deleted, nil, err
		}
		deleted += n
	}

	sample := keys
	if len(sample) > 5 {
		sample = sample[:5]
	}
	return deleted, sample, nil
}
