package cache

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func clearCache(t *testing.T, rdb *redis.Client, query string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	ClearCache(rdb)(rec, httptest.NewRequest(http.MethodDelete, "/cache/clear"+query, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			DeletedKeys int `json:"deleted_keys"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	return body.Data.DeletedKeys
}

func TestClearAllKeepsRateLimitBuckets(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	keep := []string{"rate_limit:1:stories", "rate_limit:2:reactions"}
	drop := []string{"user:followees:1", "user:stats:1", "story:42"}
	for _, k := range append(keep, drop...) {
		mr.Set(k, "x")
	}
	// Enough feed keys to need several SCAN/UNLINK batches.
	for i := 0; i < 1200; i++ {
		mr.Set(fmt.Sprintf("feed:user:%d", i), "x")
	}

	if got, want := clearCache(t, rdb, "?type=all"), 1200+len(drop); got != want {
		t.Errorf("deleted_keys = %d, want %d", got, want)
	}
	for _, k := range keep {
		if !mr.Exists(k) {
			t.Errorf("%s was deleted; rate-limit buckets must survive type=all", k)
		}
	}
	if left := mr.Keys(); len(left) != len(keep) {
		t.Errorf("keys left after clear = %v, want only %v", left, keep)
	}
}

func TestClearDefaultsToFeedOnly(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mr.Set("feed:user:1", "x")
	mr.Set("story:1", "x")

	if got := clearCache(t, rdb, ""); got != 1 {
		t.Errorf("deleted_keys = %d, want 1", got)
	}
	if !mr.Exists("story:1") {
		t.Error("story cache was deleted by a feed-only clear")
	}
}
