package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// TokenBucket represents a token bucket rate limiter
type TokenBucket struct {
	redis    *redis.Client
	capacity int64         // Maximum number of tokens
	refill   int64         // Number of tokens to refill per minute
	window   time.Duration // Time window for refilling (1 minute)
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(redisClient *redis.Client, capacity, refillRate int64) *TokenBucket {
	return &TokenBucket{
		redis:    redisClient,
		capacity: capacity,
		refill:   refillRate,
		window:   time.Minute,
	}
}

// Allow checks if the user can perform an action based on rate limiting
// Returns true if action is allowed, false otherwise
func (tb *TokenBucket) Allow(ctx context.Context, userID, action string) (bool, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, action)

	// Lua script for atomic token bucket operations
	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local window = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		-- Get current bucket state
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate tokens to add based on time elapsed
		local time_passed = now - last_refill
		local tokens_to_add = math.floor((time_passed / window) * refill_rate)
		
		if tokens_to_add > 0 then
			tokens = math.min(capacity, tokens + tokens_to_add)
			last_refill = now
		end
		
		-- Check if we can consume a token
		if tokens > 0 then
			tokens = tokens - 1
			-- Update bucket state
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
			redis.call('EXPIRE', key, window * 2) -- Set expiration to 2x window
			return 1
		else
			-- Update last_refill even if no tokens available
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', last_refill)
			redis.call('EXPIRE', key, window * 2)
			return 0
		end
	`

	now := time.Now().Unix()
	result, err := tb.redis.Eval(ctx, luaScript, []string{key},
		tb.capacity, tb.refill, int64(tb.window.Seconds()), now).Result()

	if err != nil {
		return false, fmt.Errorf("rate limit check failed: %w", err)
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from rate limit script")
	}

	return allowed == 1, nil
}

// GetRemaining returns the number of remaining tokens for a user action
func (tb *TokenBucket) GetRemaining(ctx context.Context, userID, action string) (int64, error) {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, action)

	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local window = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])
		
		-- Get current bucket state
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1]) or capacity
		local last_refill = tonumber(bucket[2]) or now
		
		-- Calculate tokens to add based on time elapsed
		local time_passed = now - last_refill
		local tokens_to_add = math.floor((time_passed / window) * refill_rate)
		
		if tokens_to_add > 0 then
			tokens = math.min(capacity, tokens + tokens_to_add)
		end
		
		return tokens
	`

	now := time.Now().Unix()
	result, err := tb.redis.Eval(ctx, luaScript, []string{key},
		tb.capacity, tb.refill, int64(tb.window.Seconds()), now).Result()

	if err != nil {
		return 0, fmt.Errorf("failed to get remaining tokens: %w", err)
	}

	remaining, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected result type from remaining tokens script")
	}

	return remaining, nil
}

// Reset clears the rate limit for a specific user action
func (tb *TokenBucket) Reset(ctx context.Context, userID, action string) error {
	key := fmt.Sprintf("rate_limit:%s:%s", userID, action)
	return tb.redis.Del(ctx, key).Err()
}
