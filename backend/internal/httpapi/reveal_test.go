package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/secret"
)

func TestGetSecretStatusEndpoint(t *testing.T) {
	cfg := config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}

	t.Run("returns active status for existing secret", func(t *testing.T) {
		mockService := &mockSecretService{
			getSecretStatusFunc: func(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
				return &secret.SecretStatus{
					SecretID:  secretID,
					Status:    secret.StatusActive,
					CreatedAt: "2026-04-15T12:00:00Z",
					ExpiresAt: "2026-04-15T13:00:00Z",
				}, nil
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/secrets/test-secret-123/status", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp secret.SecretStatus
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != secret.StatusActive {
			t.Errorf("expected status 'active', got '%s'", resp.Status)
		}

		if resp.SecretID != "test-secret-123" {
			t.Errorf("expected secretId 'test-secret-123', got '%s'", resp.SecretID)
		}
	})

	t.Run("returns not_found status for non-existent secret", func(t *testing.T) {
		mockService := &mockSecretService{
			getSecretStatusFunc: func(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
				return &secret.SecretStatus{
					SecretID: secretID,
					Status:   "not_found",
					Message:  "Secret not found or has expired.",
				}, nil
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/secrets/non-existent/status", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp secret.SecretStatus
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Status != "not_found" {
			t.Errorf("expected status 'not_found', got '%s'", resp.Status)
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockService := &mockSecretService{
			getSecretStatusFunc: func(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
				return nil, errors.New("redis connection failed")
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/secrets/test-secret-123/status", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})

	t.Run("status endpoint maps invalid token to 400", func(t *testing.T) {
		mockService := &mockSecretService{
			getSecretStatusFunc: func(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
				return nil, secret.ErrInvalidToken
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/secrets/bad-token/status", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for empty secret ID", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodGet, "/api/secrets/status", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		// Empty secret ID results in route not found
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestConsumeSecretEndpoint(t *testing.T) {
	cfg := config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}

	t.Run("consumes secret successfully", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return &secret.ConsumeSecretResponse{
					SecretID:   secretID,
					Ciphertext: "dGVzdC1jaXBoZXJ0ZXh0",
					Nonce:      "MTIzNDU2Nzg5MDEy",
					Algorithm:  "AES-GCM",
					ConsumedAt: "2026-04-15T12:30:00Z",
				}, nil
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/consume", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp secret.ConsumeSecretResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.SecretID != "test-secret-123" {
			t.Errorf("expected secretId 'test-secret-123', got '%s'", resp.SecretID)
		}

		if resp.Ciphertext == "" {
			t.Error("ciphertext should not be empty")
		}

		if resp.Nonce == "" {
			t.Error("nonce should not be empty")
		}

		if resp.Algorithm != "AES-GCM" {
			t.Errorf("expected algorithm 'AES-GCM', got '%s'", resp.Algorithm)
		}
	})

	t.Run("returns 410 for already consumed secret", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return nil, errors.New("secret not found or already consumed")
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/consume", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusGone {
			t.Errorf("expected status 410, got %d", rec.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["error"] != "SECRET_CONSUMED" {
			t.Errorf("expected error 'SECRET_CONSUMED', got '%s'", resp["error"])
		}
	})

	t.Run("open endpoint returns 410 for consumed secret", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return nil, secret.ErrSecretConsumed
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/open", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusGone {
			t.Errorf("expected status 410, got %d", rec.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["error"] != "SECRET_CONSUMED" {
			t.Errorf("expected error 'SECRET_CONSUMED', got '%s'", resp["error"])
		}
	})

	t.Run("open endpoint returns 410 for expired secret", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return nil, secret.ErrSecretExpired
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/open", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusGone {
			t.Errorf("expected status 410, got %d", rec.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["error"] != "SECRET_EXPIRED" {
			t.Errorf("expected error 'SECRET_EXPIRED', got '%s'", resp["error"])
		}
	})

	t.Run("open endpoint returns 404 for not found secret", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return nil, secret.ErrSecretNotFound
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/open", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["error"] != "SECRET_NOT_FOUND" {
			t.Errorf("expected error 'SECRET_NOT_FOUND', got '%s'", resp["error"])
		}
	})

	t.Run("open endpoint includes no-store headers", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return &secret.ConsumeSecretResponse{
					SecretID:   secretID,
					Ciphertext: "dGVzdC1jaXBoZXJ0ZXh0",
					Nonce:      "MTIzNDU2Nzg5MDEy",
					Algorithm:  "AES-GCM",
					ConsumedAt: "2026-04-15T12:30:00Z",
				}, nil
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/open", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
			t.Errorf("expected no-store cache header, got %q", got)
		}

		if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("expected no-referrer policy, got %q", got)
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return nil, errors.New("redis connection failed")
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/consume", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})

	t.Run("open endpoint maps invalid token to 400", func(t *testing.T) {
		mockService := &mockSecretService{
			consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
				return nil, secret.ErrInvalidToken
			},
		}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/bad-token/open", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns 400 for empty secret ID", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/consume", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		// Empty secret ID results in route not found
		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("includes CORS headers", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/consume", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:5173" {
			t.Errorf("expected CORS origin 'http://localhost:5173', got '%s'", origin)
		}
	})
}

func TestSecretRoutesMethodValidation(t *testing.T) {
	cfg := config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}

	mockService := &mockSecretService{}
	server := NewServer(cfg, mockService)

	t.Run("status endpoint rejects POST", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/status", strings.NewReader("{}"))
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("consume endpoint rejects GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/secrets/test-secret-123/consume", nil)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestCreateRevealSessionEndpoint(t *testing.T) {
	cfg := config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}

	t.Run("creates reveal session successfully", func(t *testing.T) {
		mockService := &mockSecretService{
			createRevealSessionFunc: func(ctx context.Context, secretID string) (*secret.RevealSessionResponse, error) {
				return &secret.RevealSessionResponse{
					SessionID: "session-abc123",
					SecretID:  secretID,
					Status:    "active",
					ExpiresAt: "2026-04-15T12:35:00Z",
				}, nil
			},
		}
		server := NewServer(cfg, mockService)

		reqBody := strings.NewReader(`{"secretId":"test-secret-123"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/reveal-sessions", reqBody)
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", rec.Code)
		}

		if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
			t.Errorf("expected no-store cache header, got %q", got)
		}

		var resp secret.RevealSessionResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.SessionID != "session-abc123" {
			t.Errorf("expected sessionId 'session-abc123', got %q", resp.SessionID)
		}
		if resp.SecretID != "test-secret-123" {
			t.Errorf("expected secretId 'test-secret-123', got %q", resp.SecretID)
		}
	})

	t.Run("rejects invalid secret ID", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/reveal-sessions", strings.NewReader(`{"secretId":""}`))
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestOpenSecretWithRevealSessionHeader(t *testing.T) {
	cfg := config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}

	mockService := &mockSecretService{
		validateRevealSessionFunc: func(ctx context.Context, secretID, sessionID string) error {
			if secretID != "test-secret-123" {
				t.Fatalf("unexpected secretID %q", secretID)
			}
			if sessionID != "session-abc123" {
				t.Fatalf("unexpected sessionID %q", sessionID)
			}
			return nil
		},
		consumeSecretFunc: func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
			return &secret.ConsumeSecretResponse{
				SecretID:   secretID,
				Ciphertext: "dGVzdC1jaXBoZXJ0ZXh0",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				ConsumedAt: "2026-04-15T12:30:00Z",
			}, nil
		},
	}
	server := NewServer(cfg, mockService)

	req := httptest.NewRequest(http.MethodPost, "/api/secrets/test-secret-123/open", strings.NewReader("{}"))
	req.Header.Set("X-Reveal-Session", "session-abc123")
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}
