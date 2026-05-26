package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestLimiter(t *testing.T) {
	// Skip if Redis not available
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Skip("Redis not available, skipping rate limiter tests")
		return
	}

	limiter := NewLimiter(client)

	t.Run("allows requests within limit", func(t *testing.T) {
		key := "test:allow:1"
		defer limiter.Reset(ctx, key)

		config := Config{
			Limit:  5,
			Window: 10 * time.Second,
		}

		// First 5 requests should be allowed
		for i := 1; i <= 5; i++ {
			result, err := limiter.Allow(ctx, key, config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Allowed {
				t.Errorf("request %d should be allowed", i)
			}

			if result.Limit != 5 {
				t.Errorf("expected limit 5, got %d", result.Limit)
			}

			expectedRemaining := 5 - i
			if result.Remaining != expectedRemaining {
				t.Errorf("expected remaining %d, got %d", expectedRemaining, result.Remaining)
			}
		}
	})

	t.Run("sets TTL atomically on first request", func(t *testing.T) {
		key := "test:ttl:1"
		defer limiter.Reset(ctx, key)

		config := Config{
			Limit:  5,
			Window: 10 * time.Second,
		}

		_, err := limiter.Allow(ctx, key, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ttl, err := client.TTL(ctx, "ratelimit:"+key).Result()
		if err != nil {
			t.Fatalf("failed to read TTL: %v", err)
		}

		if ttl <= 0 {
			t.Fatalf("expected positive TTL, got %v", ttl)
		}
		if ttl > config.Window {
			t.Fatalf("expected TTL not to exceed window %v, got %v", config.Window, ttl)
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		key := "test:block:1"
		defer limiter.Reset(ctx, key)

		config := Config{
			Limit:  3,
			Window: 10 * time.Second,
		}

		// First 3 requests allowed
		for i := 1; i <= 3; i++ {
			result, err := limiter.Allow(ctx, key, config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !result.Allowed {
				t.Errorf("request %d should be allowed", i)
			}
		}

		// 4th request should be blocked
		result, err := limiter.Allow(ctx, key, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Allowed {
			t.Error("request should be blocked")
		}

		if result.Remaining != 0 {
			t.Errorf("expected remaining 0, got %d", result.Remaining)
		}

		if result.RetryAfter <= 0 {
			t.Error("retry after should be positive")
		}
	})

	t.Run("resets after window expires", func(t *testing.T) {
		key := "test:reset:1"
		defer limiter.Reset(ctx, key)

		config := Config{
			Limit:  2,
			Window: 1 * time.Second,
		}

		// Use up the limit
		for i := 1; i <= 2; i++ {
			_, err := limiter.Allow(ctx, key, config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		// Should be blocked
		result, err := limiter.Allow(ctx, key, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("request should be blocked")
		}

		// Wait for window to expire
		time.Sleep(1100 * time.Millisecond)

		// Should be allowed again
		result, err = limiter.Allow(ctx, key, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("request should be allowed after reset")
		}
		if result.Remaining != 1 {
			t.Errorf("expected remaining 1, got %d", result.Remaining)
		}
	})

	t.Run("different keys are independent", func(t *testing.T) {
		key1 := "test:independent:1"
		key2 := "test:independent:2"
		defer limiter.Reset(ctx, key1)
		defer limiter.Reset(ctx, key2)

		config := Config{
			Limit:  2,
			Window: 10 * time.Second,
		}

		// Use up limit for key1
		for i := 1; i <= 2; i++ {
			_, err := limiter.Allow(ctx, key1, config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		// key1 should be blocked
		result, err := limiter.Allow(ctx, key1, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("key1 should be blocked")
		}

		// key2 should still be allowed
		result, err = limiter.Allow(ctx, key2, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("key2 should be allowed")
		}
	})

	t.Run("reset clears the limit", func(t *testing.T) {
		key := "test:manual-reset:1"
		defer limiter.Reset(ctx, key)

		config := Config{
			Limit:  2,
			Window: 10 * time.Second,
		}

		// Use up the limit
		for i := 1; i <= 2; i++ {
			_, err := limiter.Allow(ctx, key, config)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}

		// Should be blocked
		result, err := limiter.Allow(ctx, key, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Allowed {
			t.Error("request should be blocked")
		}

		// Reset manually
		err = limiter.Reset(ctx, key)
		if err != nil {
			t.Fatalf("failed to reset: %v", err)
		}

		// Should be allowed again
		result, err = limiter.Allow(ctx, key, config)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Error("request should be allowed after reset")
		}
	})
}
