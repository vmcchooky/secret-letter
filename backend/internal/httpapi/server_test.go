package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/secret"

	"github.com/redis/go-redis/v9"
)

func TestHealthEndpoint(t *testing.T) {
	t.Run("returns healthy with dependency status", func(t *testing.T) {
		server := NewServer(config.Config{
			ServiceName:   "test-api",
			Host:          "127.0.0.1",
			Port:          "8080",
			AllowedOrigin: "http://localhost:5173",
		}, secret.NewInMemoryService())

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Fatalf("expected CORS origin header to be set, got %q", got)
		}

		if got := rec.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("expected health response to be no-store, got %q", got)
		}

		var response healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Service != "test-api" {
			t.Errorf("expected service name 'test-api', got %q", response.Service)
		}

		if response.Status != "healthy" {
			t.Errorf("expected status 'healthy', got %q", response.Status)
		}

		if response.Dependencies["in-memory placeholder"] != "scaffold" {
			t.Errorf("expected dependency status to be exposed, got %#v", response.Dependencies)
		}
	})

	t.Run("returns 503 when dependency is unhealthy", func(t *testing.T) {
		server := NewServer(config.Config{
			ServiceName:   "test-api",
			Host:          "127.0.0.1",
			Port:          "8080",
			AllowedOrigin: "http://localhost:5173",
		}, &mockSecretService{
			healthFunc: func(ctx context.Context) secret.HealthStatus {
				return secret.HealthStatus{Store: "redis", Mode: "unhealthy"}
			},
		})

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", rec.Code)
		}

		var response healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Status != "unhealthy" {
			t.Errorf("expected status 'unhealthy', got %q", response.Status)
		}

		if response.Dependencies["redis"] != "unhealthy" {
			t.Errorf("expected redis dependency to be unhealthy, got %#v", response.Dependencies)
		}
	})

	t.Run("returns 503 when Redis dependency is unreachable", func(t *testing.T) {
		redisClient := redis.NewClient(&redis.Options{
			Addr:         "127.0.0.1:1",
			DialTimeout:  25 * time.Millisecond,
			ReadTimeout:  25 * time.Millisecond,
			WriteTimeout: 25 * time.Millisecond,
			MaxRetries:   0,
		})
		defer redisClient.Close()

		secretService, err := secret.NewRedisServiceWithEncryptionKey(redisClient, []byte("0123456789abcdefghijklmnopqrstuv"))
		if err != nil {
			t.Fatalf("NewRedisServiceWithEncryptionKey failed: %v", err)
		}

		server := NewServer(config.Config{
			ServiceName:   "test-api",
			Host:          "127.0.0.1",
			Port:          "8080",
			AllowedOrigin: "http://localhost:5173",
		}, secretService)

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", rec.Code)
		}

		var response healthResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Dependencies["redis"] != "unhealthy" {
			t.Errorf("expected redis dependency to be unhealthy, got %#v", response.Dependencies)
		}
	})

	t.Run("readyz uses the same dependency-aware health response", func(t *testing.T) {
		server := NewServer(config.Config{
			ServiceName:   "test-api",
			Host:          "127.0.0.1",
			Port:          "8080",
			AllowedOrigin: "http://localhost:5173",
		}, &mockSecretService{
			healthFunc: func(ctx context.Context) secret.HealthStatus {
				return secret.HealthStatus{Store: "redis", Mode: "healthy"}
			},
		})

		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestOptionsRequestReturnsNoContent(t *testing.T) {
	server := NewServer(config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}, secret.NewInMemoryService())

	req := httptest.NewRequest(http.MethodOptions, "/api/secrets", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Fatalf("expected Access-Control-Max-Age to be 86400, got %q", got)
	}
}

func TestRequestIDHeader(t *testing.T) {
	server := NewServer(config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}, secret.NewInMemoryService())

	t.Run("generates request ID when not provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		requestID := rec.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("expected X-Request-ID header to be set")
		}
	})

	t.Run("echoes provided request ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("X-Request-ID", "test-request-id-123")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		requestID := rec.Header().Get("X-Request-ID")
		if requestID != "test-request-id-123" {
			t.Errorf("expected X-Request-ID to be 'test-request-id-123', got %q", requestID)
		}
	})
}

func TestRequestSizeLimit(t *testing.T) {
	server := NewServer(config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}, secret.NewInMemoryService())

	t.Run("accepts request within size limit", func(t *testing.T) {
		body := bytes.NewBufferString(`{"test": "data"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		// Should return 501 Not Implemented, not 413
		if rec.Code == http.StatusRequestEntityTooLarge {
			t.Error("small request should not be rejected for size")
		}
	})

	t.Run("rejects request exceeding size limit", func(t *testing.T) {
		// Create a payload larger than 15KB
		largePayload := strings.Repeat("a", 16*1024)
		body := bytes.NewBufferString(largePayload)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("expected status 413, got %d", rec.Code)
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	server := NewServer(config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}, secret.NewInMemoryService())

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	wantHeaders := map[string]string{
		"Access-Control-Allow-Origin":   "http://localhost:5173",
		"Access-Control-Allow-Headers":  "Content-Type, Accept, X-Request-ID, X-Reveal-Session",
		"Access-Control-Allow-Methods":  "GET, POST, OPTIONS",
		"Access-Control-Expose-Headers": "X-Request-ID, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, Retry-After",
		"Access-Control-Max-Age":        "86400",
	}

	for header, want := range wantHeaders {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("expected %s to be %q, got %q", header, want, got)
		}
	}
}
