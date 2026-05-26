package main

import (
	"log"
	"net/http"
	"time"

	"one-time-link/backend/internal/config"
	"one-time-link/backend/internal/httpapi"
	"one-time-link/backend/internal/secret"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	// Initialize Redis client with connection pooling
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdle,
		MaxRetries:   cfg.RedisMaxRetries,
	})

	// Create Redis-backed secret service
	secretService, err := secret.NewRedisService(redisClient)
	if err != nil {
		log.Fatalf("failed to initialize secret service: %v", err)
	}

	// Create server. Local mode disables rate limiting so manual testing is not
	// blocked by old Redis buckets or repeated form submissions.
	server := httpapi.NewServer(cfg, secretService)
	if cfg.RateLimitEnabled {
		server = httpapi.NewServerWithRateLimiting(cfg, secretService, redisClient)
	}

	log.Printf("starting %s on %s", cfg.ServiceName, cfg.ListenAddress())
	log.Printf("Redis: %s (pool: %d, min idle: %d, max retries: %d)",
		cfg.RedisAddr, cfg.RedisPoolSize, cfg.RedisMinIdle, cfg.RedisMaxRetries)
	if cfg.RateLimitEnabled {
		log.Printf("Rate limiting enabled: create=%d/%s, consume=%d/%s, status=%d/%s, reveal_session=%d/%s",
			cfg.CreateLimit, cfg.RateLimitWindow,
			cfg.ConsumeLimit, cfg.RateLimitWindow,
			cfg.StatusLimit, cfg.RateLimitWindow,
			cfg.RevealLimit, cfg.RateLimitWindow)
	} else {
		log.Printf("Rate limiting disabled for APP_ENV=%q", cfg.AppEnv)
	}

	srv := &http.Server{
		Addr:         cfg.ListenAddress(),
		Handler:      server.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
