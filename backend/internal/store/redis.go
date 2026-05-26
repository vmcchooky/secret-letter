package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// RedisConnectionTimeout is the timeout for initial connection
	RedisConnectionTimeout = 5 * time.Second
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr, password string, db int) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Use timeout for connection check
	ctx, cancel := context.WithTimeout(context.Background(), RedisConnectionTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{client: client}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}
