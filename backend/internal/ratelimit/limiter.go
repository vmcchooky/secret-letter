package ratelimit

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter implements rate limiting using Redis
type Limiter struct {
	client *redis.Client
}

// Config holds rate limit configuration for an endpoint
type Config struct {
	Limit  int           // Maximum requests allowed
	Window time.Duration // Time window for the limit
}

// Result represents the result of a rate limit check
type Result struct {
	Allowed    bool      // Whether the request is allowed
	Limit      int       // Maximum requests allowed
	Remaining  int       // Remaining requests in current window
	ResetAt    time.Time // When the limit resets
	RetryAfter int       // Seconds to wait before retrying (if not allowed)
}

var allowScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
end

local ttl = redis.call("PTTL", KEYS[1])
if ttl < 0 then
	redis.call("PEXPIRE", KEYS[1], ARGV[1])
	ttl = tonumber(ARGV[1])
end

return {count, ttl}
`)

// NewLimiter creates a new rate limiter
func NewLimiter(client *redis.Client) *Limiter {
	return &Limiter{
		client: client,
	}
}

// Allow checks if a request is allowed under the rate limit
func (l *Limiter) Allow(ctx context.Context, key string, config Config) (*Result, error) {
	// Create Redis key
	redisKey := fmt.Sprintf("ratelimit:%s", key)

	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	windowMillis := config.Window.Milliseconds()
	if windowMillis < 1 {
		windowMillis = 1
	}

	resultRaw, err := allowScript.Run(ctx, l.client, []string{redisKey}, windowMillis).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to run rate limit script: %w", err)
	}

	values, ok := resultRaw.([]interface{})
	if !ok || len(values) < 2 {
		return nil, fmt.Errorf("unexpected rate limit script response")
	}

	count, ok := redisInt64(values[0])
	if !ok {
		return nil, fmt.Errorf("unexpected rate limit count type")
	}

	ttlMillis, ok := redisInt64(values[1])
	if !ok {
		return nil, fmt.Errorf("unexpected rate limit TTL type")
	}

	ttl := time.Duration(ttlMillis) * time.Millisecond

	// Calculate reset time
	resetAt := time.Now().Add(ttl)

	// Check if limit exceeded
	allowed := count <= int64(config.Limit)
	remaining := config.Limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	result := &Result{
		Allowed:    allowed,
		Limit:      config.Limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: int(math.Ceil(ttl.Seconds())),
	}

	return result, nil
}

func redisInt64(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	default:
		return 0, false
	}
}

// Reset clears the rate limit for a key (useful for testing)
func (l *Limiter) Reset(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("ratelimit:%s", key)
	return l.client.Del(ctx, redisKey).Err()
}
