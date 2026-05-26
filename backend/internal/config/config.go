package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceName       string
	AppEnv            string
	Host              string
	Port              string
	AllowedOrigin     string
	TrustedProxyCIDRs string
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
	RedisPoolSize     int
	RedisMinIdle      int
	RedisMaxRetries   int
	RateLimitEnabled  bool
	RateLimitWindow   time.Duration
	CreateLimit       int
	ConsumeLimit      int
	StatusLimit       int
	RevealLimit       int
}

func Load() Config {
	return Config{
		ServiceName:       getEnv("APP_SERVICE_NAME", "secret-letter-api"),
		AppEnv:            getEnv("APP_ENV", "local"),
		Host:              getEnv("APP_HOST", "0.0.0.0"),
		Port:              getEnv("APP_PORT", "8080"),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		TrustedProxyCIDRs: getEnv("TRUSTED_PROXY_CIDRS", "127.0.0.1/32,::1/128"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		RedisPoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
		RedisMinIdle:      getEnvInt("REDIS_MIN_IDLE", 5),
		RedisMaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
		RateLimitEnabled: getEnvBool(
			"RATE_LIMIT_ENABLED",
			!isLocalEnv(getEnv("APP_ENV", "local")),
		),
		RateLimitWindow: time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 3600)) * time.Second,
		CreateLimit:     getEnvInt("RATE_LIMIT_CREATE_PER_WINDOW", 120),
		ConsumeLimit:    getEnvInt("RATE_LIMIT_CONSUME_PER_WINDOW", 240),
		StatusLimit:     getEnvInt("RATE_LIMIT_STATUS_PER_WINDOW", 600),
		RevealLimit:     getEnvInt("RATE_LIMIT_REVEAL_SESSION_PER_WINDOW", 240),
	}
}

func (c Config) ListenAddress() string {
	return c.Host + ":" + c.Port
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}

	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func isLocalEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}
