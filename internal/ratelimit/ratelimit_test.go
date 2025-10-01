package ratelimit

import (
	"context"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/alicebob/miniredis/v2"
)

// setupTestRedis creates an in-memory Redis server for testing
func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
		DB:   0,
	})

	// Test connection
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		t.Fatalf("Failed to connect to test Redis: %v", err)
	}

	cleanup := func() {
		redisClient.Close()
		mr.Close()
	}

	return redisClient, cleanup
}

func TestTokenBucket_Allow(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	// Create token bucket with 5 tokens, refill 5 per minute
	bucket := NewTokenBucket(redisClient, 5, 5)

	ctx := context.Background()
	userID := "test_user"
	action := "test_action"

	// Test that we can consume tokens up to the limit
	for i := 0; i < 5; i++ {
		allowed, err := bucket.Allow(ctx, userID, action)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !allowed {
			t.Fatalf("Expected request %d to be allowed", i+1)
		}
	}

	// Test that the 6th request is denied
	allowed, err := bucket.Allow(ctx, userID, action)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if allowed {
		t.Fatal("Expected request to be denied after limit reached")
	}

	// Test remaining tokens
	remaining, err := bucket.GetRemaining(ctx, userID, action)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("Expected 0 remaining tokens, got %d", remaining)
	}
}

func TestTokenBucket_GetRemaining(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	bucket := NewTokenBucket(redisClient, 10, 10)

	ctx := context.Background()
	userID := "test_user_2"
	action := "test_action_2"

	// Initially should have full capacity
	remaining, err := bucket.GetRemaining(ctx, userID, action)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if remaining != 10 {
		t.Fatalf("Expected 10 remaining tokens, got %d", remaining)
	}

	// Consume 3 tokens
	for i := 0; i < 3; i++ {
		bucket.Allow(ctx, userID, action)
	}

	// Should have 7 remaining
	remaining, err = bucket.GetRemaining(ctx, userID, action)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if remaining != 7 {
		t.Fatalf("Expected 7 remaining tokens, got %d", remaining)
	}
}

func TestTokenBucket_Reset(t *testing.T) {
	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	bucket := NewTokenBucket(redisClient, 5, 5)

	ctx := context.Background()
	userID := "test_user_3"
	action := "test_action_3"

	// Consume all tokens
	for i := 0; i < 5; i++ {
		bucket.Allow(ctx, userID, action)
	}

	// Reset the bucket
	err := bucket.Reset(ctx, userID, action)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should be able to consume tokens again
	remaining, err := bucket.GetRemaining(ctx, userID, action)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if remaining != 5 {
		t.Fatalf("Expected 5 remaining tokens after reset, got %d", remaining)
	}
}
