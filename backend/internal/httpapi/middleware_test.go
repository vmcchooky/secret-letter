package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/ratelimit"
	"secret-letter/backend/internal/secret"

	"github.com/redis/go-redis/v9"
)

// MockService is a mock implementation of secret.Service for testing
type MockService struct{}

func (m *MockService) Health(ctx context.Context) secret.HealthStatus {
	return secret.HealthStatus{
		Store: "mock",
		Mode:  "test",
	}
}

func (m *MockService) CreateSecret(ctx context.Context, req secret.CreateSecretRequest) (*secret.CreateSecretResponse, error) {
	return &secret.CreateSecretResponse{
		SecretID: "mock-secret-id",
	}, nil
}

func (m *MockService) GetSecretStatus(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
	return &secret.SecretStatus{
		SecretID: secretID,
		Status:   "available",
	}, nil
}

func (m *MockService) ConsumeSecret(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
	return &secret.ConsumeSecretResponse{
		SecretID:   secretID,
		Ciphertext: "mock-ciphertext",
		Nonce:      "mock-nonce",
		Algorithm:  "XChaCha20-Poly1305",
	}, nil
}

func TestRateLimitingMiddleware(t *testing.T) {
	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	err := redisClient.Ping(ctx).Err()
	if err != nil {
		t.Skip("Redis not available, skipping rate limiting middleware tests")
		return
	}

	// Create test server with rate limiting
	cfg := config.Config{
		AllowedOrigin:     "*",
		TrustedProxyCIDRs: "192.168.1.0/24",
		CreateLimit:       10,
		ConsumeLimit:      20,
		StatusLimit:       100,
		RevealLimit:       20,
		RateLimitWindow:   time.Hour,
	}
	mockService := &MockService{}
	server := NewServerWithRateLimiting(cfg, mockService, redisClient)
	handler := server.Handler()

	t.Run("allows requests within limit", func(t *testing.T) {
		testIP := "192.168.1.100"
		endpoint := "create_secret"

		// Clean up before test
		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, testIP))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, testIP))

		// Make requests within limit (10 for create_secret)
		for i := 1; i <= 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = testIP + ":12345"
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			// Should not be rate limited
			if w.Code == http.StatusTooManyRequests {
				t.Errorf("request %d should not be rate limited", i)
			}

			// Check rate limit headers
			if w.Header().Get("X-RateLimit-Limit") == "" {
				t.Error("X-RateLimit-Limit header missing")
			}
			if w.Header().Get("X-RateLimit-Remaining") == "" {
				t.Error("X-RateLimit-Remaining header missing")
			}
			if w.Header().Get("X-RateLimit-Reset") == "" {
				t.Error("X-RateLimit-Reset header missing")
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		testIP := "192.168.1.101"
		endpoint := "create_secret"

		// Clean up before test
		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, testIP))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, testIP))

		// Make 10 requests (the limit)
		for i := 1; i <= 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = testIP + ":12345"
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Errorf("request %d should not be rate limited yet", i)
			}
		}

		// 11th request should be blocked
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = testIP + ":12345"
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", w.Code)
		}

		// Check Retry-After header
		if w.Header().Get("Retry-After") == "" {
			t.Error("Retry-After header missing")
		}

		// Check error response
		if !strings.Contains(w.Body.String(), "rate_limit_exceeded") {
			t.Error("expected rate_limit_exceeded error")
		}
	})

	t.Run("different endpoints have different limits", func(t *testing.T) {
		testIP := "192.168.1.102"

		// Clean up before test
		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("create_secret:%s", testIP))
		limiter.Reset(ctx, fmt.Sprintf("consume_secret:%s", testIP))
		defer limiter.Reset(ctx, fmt.Sprintf("create_secret:%s", testIP))
		defer limiter.Reset(ctx, fmt.Sprintf("consume_secret:%s", testIP))

		// Use up create_secret limit (10)
		for i := 1; i <= 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = testIP + ":12345"
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		// create_secret should be blocked
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = testIP + ":12345"
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Error("create_secret should be rate limited")
		}

		// consume_secret should still work (different limit: 20)
		req = httptest.NewRequest(http.MethodPost, "/api/secrets/test123/consume", nil)
		req.RemoteAddr = testIP + ":12345"

		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			t.Error("consume_secret should not be rate limited yet")
		}
	})

	t.Run("different IPs are tracked independently", func(t *testing.T) {
		ip1 := "192.168.1.103"
		ip2 := "192.168.1.104"
		endpoint := "create_secret"

		// Clean up before test
		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, ip1))
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, ip2))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, ip1))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, ip2))

		// Use up limit for ip1
		for i := 1; i <= 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = ip1 + ":12345"
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		// ip1 should be blocked
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = ip1 + ":12345"
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Error("ip1 should be rate limited")
		}

		// ip2 should still work
		req = httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = ip2 + ":12345"
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			t.Error("ip2 should not be rate limited")
		}
	})

	t.Run("health check is not rate limited", func(t *testing.T) {
		testIP := "192.168.1.105"

		// Make many health check requests
		for i := 1; i <= 200; i++ {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.RemoteAddr = testIP + ":12345"

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Errorf("health check should never be rate limited (request %d)", i)
			}

			// Health check should not have rate limit headers
			if w.Header().Get("X-RateLimit-Limit") != "" {
				t.Error("health check should not have rate limit headers")
			}
		}
	})

	t.Run("X-Forwarded-For header is respected from trusted proxy", func(t *testing.T) {
		forwardedIP := "10.0.0.50"
		endpoint := "create_secret"

		// Clean up before test
		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, forwardedIP))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, forwardedIP))

		// Use up limit with X-Forwarded-For
		for i := 1; i <= 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = "192.168.1.1:12345" // Proxy IP
			req.Header.Set("X-Forwarded-For", forwardedIP)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		// Should be blocked based on X-Forwarded-For IP
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = "192.168.1.1:12345"
		req.Header.Set("X-Forwarded-For", forwardedIP)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Error("should be rate limited based on X-Forwarded-For")
		}
	})

	t.Run("X-Forwarded-For header is ignored from untrusted remote", func(t *testing.T) {
		forwardedIP := "10.0.0.51"
		untrustedRemoteIP := "203.0.113.200"
		endpoint := "create_secret"

		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, forwardedIP))
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, untrustedRemoteIP))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, forwardedIP))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, untrustedRemoteIP))

		for i := 1; i <= 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = untrustedRemoteIP + ":12345"
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("%s%d", forwardedIP, i))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = untrustedRemoteIP + ":12345"
		req.Header.Set("X-Forwarded-For", "198.51.100.99")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Error("untrusted remote should be rate limited by RemoteAddr despite spoofed X-Forwarded-For")
		}
	})

	t.Run("rotating spoofed X-Forwarded-For values from untrusted remote cannot bypass limit", func(t *testing.T) {
		untrustedRemoteIP := "203.0.113.210"

		originalLimit := server.config.CreateLimit
		server.config.CreateLimit = 2
		defer func() { server.config.CreateLimit = originalLimit }()

		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("create_secret:%s", untrustedRemoteIP))
		defer limiter.Reset(ctx, fmt.Sprintf("create_secret:%s", untrustedRemoteIP))

		for i := 1; i <= 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = untrustedRemoteIP + ":12345"
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if w.Code == http.StatusTooManyRequests {
				t.Fatalf("request %d should not be rate limited yet", i)
			}
		}

		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = untrustedRemoteIP + ":12345"
		req.Header.Set("X-Forwarded-For", "198.51.100.250")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected spoofed request to be rate limited by remote addr, got %d", w.Code)
		}
	})

	t.Run("rate limit resets after window", func(t *testing.T) {
		testIP := "192.168.1.106"

		// Temporarily override the limit for faster testing.
		originalLimit := server.config.CreateLimit
		originalWindow := server.config.RateLimitWindow
		server.config.CreateLimit = 2
		server.config.RateLimitWindow = 1 * time.Second
		defer func() {
			server.config.CreateLimit = originalLimit
			server.config.RateLimitWindow = originalWindow
		}()

		endpoint := "create_secret"
		limiter := ratelimit.NewLimiter(redisClient)
		limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, testIP))
		defer limiter.Reset(ctx, fmt.Sprintf("%s:%s", endpoint, testIP))

		// Use up the limit
		for i := 1; i <= 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
			req.RemoteAddr = testIP + ":12345"
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}

		// Should be blocked
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = testIP + ":12345"
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Error("should be rate limited")
		}

		// Wait for window to expire
		time.Sleep(1100 * time.Millisecond)

		// Should work again
		req = httptest.NewRequest(http.MethodPost, "/api/secrets", strings.NewReader(`{"ciphertext":"dGVzdA","nonce":"MTIzNDU2Nzg5MDEy","algorithm":"AES-GCM","ttlSeconds":3600}`))
		req.RemoteAddr = testIP + ":12345"
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			t.Error("should not be rate limited after window reset")
		}
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name              string
		remoteAddr        string
		xForwardedFor     string
		xRealIP           string
		trustedProxyCIDRs string
		expectedIP        string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expectedIP: "192.168.1.1",
		},
		{
			name:              "X-Forwarded-For single IP from trusted proxy",
			remoteAddr:        "10.0.0.1:12345",
			xForwardedFor:     "203.0.113.1",
			trustedProxyCIDRs: "10.0.0.0/8",
			expectedIP:        "203.0.113.1",
		},
		{
			name:              "X-Forwarded-For multiple IPs from trusted proxy",
			remoteAddr:        "10.0.0.1:12345",
			xForwardedFor:     "203.0.113.1, 10.0.0.2, 10.0.0.3",
			trustedProxyCIDRs: "10.0.0.0/8",
			expectedIP:        "203.0.113.1",
		},
		{
			name:              "X-Real-IP from trusted proxy",
			remoteAddr:        "10.0.0.1:12345",
			xRealIP:           "203.0.113.2",
			trustedProxyCIDRs: "10.0.0.0/8",
			expectedIP:        "203.0.113.2",
		},
		{
			name:              "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:        "10.0.0.1:12345",
			xForwardedFor:     "203.0.113.1",
			xRealIP:           "203.0.113.2",
			trustedProxyCIDRs: "10.0.0.0/8",
			expectedIP:        "203.0.113.1",
		},
		{
			name:              "X-Forwarded-For ignored from untrusted remote",
			remoteAddr:        "198.51.100.10:12345",
			xForwardedFor:     "203.0.113.1",
			trustedProxyCIDRs: "10.0.0.0/8",
			expectedIP:        "198.51.100.10",
		},
		{
			name:              "malformed X-Forwarded-For falls back to X-Real-IP",
			remoteAddr:        "10.0.0.1:12345",
			xForwardedFor:     "not-an-ip",
			xRealIP:           "203.0.113.2",
			trustedProxyCIDRs: "10.0.0.0/8",
			expectedIP:        "203.0.113.2",
		},
		{
			name:              "single trusted proxy IP is accepted",
			remoteAddr:        "10.0.0.1:12345",
			xForwardedFor:     "203.0.113.3",
			trustedProxyCIDRs: "10.0.0.1",
			expectedIP:        "203.0.113.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req, parseTrustedProxyCIDRs(tt.trustedProxyCIDRs))
			if ip != tt.expectedIP {
				t.Errorf("expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}
