package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/httpapi"
	"secret-letter/backend/internal/secret"

	"github.com/redis/go-redis/v9"
)

const (
	redisDialTimeout  = 2 * time.Second
	redisReadTimeout  = 3 * time.Second
	redisWriteTimeout = 3 * time.Second
	redisPoolTimeout  = 3 * time.Second
	serverShutdownTTL = 15 * time.Second
)

func main() {
	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize Redis client with connection pooling
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: cfg.RedisMinIdle,
		MaxRetries:   cfg.RedisMaxRetries,
		DialTimeout:  redisDialTimeout,
		ReadTimeout:  redisReadTimeout,
		WriteTimeout: redisWriteTimeout,
		PoolTimeout:  redisPoolTimeout,
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("warning: failed to close Redis client cleanly: %v", err)
		}
	}()

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
	log.Printf("Redis: %s (pool: %d, min idle: %d, max retries: %d, dial/read/write/pool timeouts: %s/%s/%s/%s)",
		cfg.RedisAddr, cfg.RedisPoolSize, cfg.RedisMinIdle, cfg.RedisMaxRetries,
		redisDialTimeout, redisReadTimeout, redisWriteTimeout, redisPoolTimeout)
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

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.ListenAndServe()
	}()

	shutdownSignalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	select {
	case err := <-serverErrCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	case <-shutdownSignalCtx.Done():
		log.Printf("shutdown signal received, draining %s", cfg.ServiceName)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTTL)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("graceful shutdown failed: %v", err)
		}

		if err := <-serverErrCh; err != nil && err != http.ErrServerClosed {
			log.Fatalf("server exited with error during shutdown: %v", err)
		}

		log.Printf("%s shut down cleanly", cfg.ServiceName)
	}
}
