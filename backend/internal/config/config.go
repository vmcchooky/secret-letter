package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServiceName       = "secret-letter-api"
	defaultAppEnv            = "local"
	defaultHost              = "0.0.0.0"
	defaultPort              = "8080"
	defaultAllowedOrigin     = "http://localhost:5173"
	defaultTrustedProxyCIDRs = "127.0.0.1/32,::1/128"
	defaultRedisAddr         = "localhost:6379"
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
	MaxSecretSizeKB   int
}

func Load() (Config, error) {
	appEnv := getEnv("APP_ENV", defaultAppEnv)

	cfg := Config{
		ServiceName:       getEnv("APP_SERVICE_NAME", defaultServiceName),
		AppEnv:            appEnv,
		Host:              getEnv("APP_HOST", defaultHost),
		Port:              getEnv("APP_PORT", defaultPort),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", defaultAllowedOrigin),
		TrustedProxyCIDRs: getEnv("TRUSTED_PROXY_CIDRS", defaultTrustedProxyCIDRs),
		RedisAddr:         getEnv("REDIS_ADDR", defaultRedisAddr),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvInt("REDIS_DB", 0),
		RedisPoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
		RedisMinIdle:      getEnvInt("REDIS_MIN_IDLE", 5),
		RedisMaxRetries:   getEnvInt("REDIS_MAX_RETRIES", 3),
		RateLimitEnabled: getEnvBool(
			"RATE_LIMIT_ENABLED",
			!isLocalEnv(appEnv),
		),
		RateLimitWindow: time.Duration(getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 3600)) * time.Second,
		CreateLimit:     getEnvInt("RATE_LIMIT_CREATE_PER_WINDOW", 120),
		ConsumeLimit:    getEnvInt("RATE_LIMIT_CONSUME_PER_WINDOW", 240),
		StatusLimit:     getEnvInt("RATE_LIMIT_STATUS_PER_WINDOW", 600),
		RevealLimit:     getEnvInt("RATE_LIMIT_REVEAL_SESSION_PER_WINDOW", 240),
	}

	if err := validateLoadedConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
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

func isProductionEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "production", "prod":
		return true
	default:
		return false
	}
}

func validateLoadedConfig(cfg Config) error {
	if !isProductionEnv(cfg.AppEnv) {
		return nil
	}

	if err := validateProductionAllowedOrigin(); err != nil {
		return err
	}

	if err := validateProductionTrustedProxyCIDRs(); err != nil {
		return err
	}

	if !cfg.RateLimitEnabled {
		return fmt.Errorf("RATE_LIMIT_ENABLED must be true when APP_ENV=%q", cfg.AppEnv)
	}

	return nil
}

func validateProductionAllowedOrigin() error {
	rawOrigin, exists := os.LookupEnv("ALLOWED_ORIGIN")
	if !exists || strings.TrimSpace(rawOrigin) == "" {
		return fmt.Errorf("ALLOWED_ORIGIN must be explicitly set when APP_ENV=production")
	}

	origin := strings.TrimSpace(rawOrigin)
	if origin == "*" {
		return fmt.Errorf("ALLOWED_ORIGIN must not be '*' when APP_ENV=production")
	}

	parsed, err := url.Parse(origin)
	if err != nil || !parsed.IsAbs() {
		return fmt.Errorf("ALLOWED_ORIGIN must be a valid absolute origin when APP_ENV=production")
	}

	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("ALLOWED_ORIGIN must use https when APP_ENV=production")
	}

	if parsed.Host == "" || parsed.Hostname() == "" {
		return fmt.Errorf("ALLOWED_ORIGIN must include a host when APP_ENV=production")
	}

	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("ALLOWED_ORIGIN must be a bare origin without path, query, fragment, or credentials")
	}

	hostname := parsed.Hostname()
	if strings.EqualFold(hostname, "localhost") {
		return fmt.Errorf("ALLOWED_ORIGIN must not use localhost when APP_ENV=production")
	}

	if ip := net.ParseIP(hostname); ip != nil && ip.IsLoopback() {
		return fmt.Errorf("ALLOWED_ORIGIN must not use a loopback host when APP_ENV=production")
	}

	return nil
}

func validateProductionTrustedProxyCIDRs() error {
	rawCIDRs, exists := os.LookupEnv("TRUSTED_PROXY_CIDRS")
	if !exists || strings.TrimSpace(rawCIDRs) == "" {
		return fmt.Errorf("TRUSTED_PROXY_CIDRS must be explicitly set when APP_ENV=production")
	}

	for _, part := range strings.Split(rawCIDRs, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("TRUSTED_PROXY_CIDRS must not contain empty entries")
		}

		if ip := net.ParseIP(part); ip != nil {
			continue
		}

		if _, _, err := net.ParseCIDR(part); err != nil {
			return fmt.Errorf("TRUSTED_PROXY_CIDRS entry %q is invalid: %w", part, err)
		}
	}

	return nil
}
