package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/secret"
)

// mockSecretService implements secret.Service for testing
type mockSecretService struct {
	createSecretFunc          func(ctx context.Context, req secret.CreateSecretRequest) (*secret.CreateSecretResponse, error)
	healthFunc                func(ctx context.Context) secret.HealthStatus
	getSecretStatusFunc       func(ctx context.Context, secretID string) (*secret.SecretStatus, error)
	consumeSecretFunc         func(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error)
	createRevealSessionFunc   func(ctx context.Context, secretID string) (*secret.RevealSessionResponse, error)
	validateRevealSessionFunc func(ctx context.Context, secretID, sessionID string) error
}

func (m *mockSecretService) CreateSecret(ctx context.Context, req secret.CreateSecretRequest) (*secret.CreateSecretResponse, error) {
	if m.createSecretFunc != nil {
		return m.createSecretFunc(ctx, req)
	}
	return &secret.CreateSecretResponse{
		SecretID:  "test-secret-id-123",
		ExpiresAt: "2026-04-15T13:00:00Z",
	}, nil
}

func (m *mockSecretService) Health(ctx context.Context) secret.HealthStatus {
	if m.healthFunc != nil {
		return m.healthFunc(ctx)
	}
	return secret.HealthStatus{Store: "mock", Mode: "test"}
}

func (m *mockSecretService) GetSecretStatus(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
	if m.getSecretStatusFunc != nil {
		return m.getSecretStatusFunc(ctx, secretID)
	}
	return &secret.SecretStatus{
		SecretID:  secretID,
		Status:    "pending",
		CreatedAt: "2026-04-15T12:00:00Z",
		ExpiresAt: "2026-04-15T13:00:00Z",
	}, nil
}

func (m *mockSecretService) ConsumeSecret(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
	if m.consumeSecretFunc != nil {
		return m.consumeSecretFunc(ctx, secretID)
	}
	return &secret.ConsumeSecretResponse{
		SecretID:   secretID,
		Ciphertext: "dGVzdC1jaXBoZXJ0ZXh0",
		Nonce:      "MTIzNDU2Nzg5MDEy",
		Algorithm:  "AES-GCM",
		ConsumedAt: "2026-04-15T12:30:00Z",
	}, nil
}

func (m *mockSecretService) CreateRevealSession(ctx context.Context, secretID string) (*secret.RevealSessionResponse, error) {
	if m.createRevealSessionFunc != nil {
		return m.createRevealSessionFunc(ctx, secretID)
	}
	return &secret.RevealSessionResponse{
		SessionID: "test-session-id-123",
		SecretID:  secretID,
		Status:    "active",
		ExpiresAt: "2026-04-15T12:35:00Z",
	}, nil
}

func (m *mockSecretService) ValidateRevealSession(ctx context.Context, secretID, sessionID string) error {
	if m.validateRevealSessionFunc != nil {
		return m.validateRevealSessionFunc(ctx, secretID, sessionID)
	}
	return nil
}

func TestCreateSecretEndpoint(t *testing.T) {
	cfg := config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "8080",
		AllowedOrigin: "http://localhost:5173",
	}

	t.Run("creates secret successfully with valid request", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"ciphertext": "dGVzdC1jaXBoZXJ0ZXh0LWJhc2U2NHVybA",
			"nonce":      "MTIzNDU2Nzg5MDEy",
			"algorithm":  "AES-GCM",
			"ttlSeconds": 3600,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", rec.Code)
		}

		var resp secret.CreateSecretResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.SecretID == "" {
			t.Error("secretId should not be empty")
		}

		if resp.ExpiresAt == "" {
			t.Error("expiresAt should not be empty")
		}
	})

	t.Run("rejects request with invalid algorithm", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"ciphertext": "dGVzdA",
			"nonce":      "MTIzNDU2Nzg5MDEy",
			"algorithm":  "AES-CBC",
			"ttlSeconds": 3600,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects request with invalid TTL", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"ciphertext": "dGVzdA",
			"nonce":      "MTIzNDU2Nzg5MDEy",
			"algorithm":  "AES-GCM",
			"ttlSeconds": 7200, // invalid
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("rejects plaintext content request", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"content":          "server-side plaintext is not accepted",
			"expiresInMinutes": 60,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		if !bytes.Contains(rec.Body.Bytes(), []byte("plaintext_not_allowed")) {
			t.Errorf("expected plaintext_not_allowed validation error, got %s", rec.Body.String())
		}
	})

	t.Run("rejects false burnAfterRead", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"ciphertext":    "dGVzdA",
			"nonce":         "MTIzNDU2Nzg5MDEy",
			"algorithm":     "AES-GCM",
			"ttlSeconds":    3600,
			"burnAfterRead": false,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		if !bytes.Contains(rec.Body.Bytes(), []byte("burn_after_read_required")) {
			t.Errorf("expected burn_after_read_required validation error, got %s", rec.Body.String())
		}
	})

	t.Run("rejects request with invalid JSON", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("returns request ID in response header", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"ciphertext": "dGVzdA",
			"nonce":      "MTIzNDU2Nzg5MDEy",
			"algorithm":  "AES-GCM",
			"ttlSeconds": 3600,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", "test-request-123")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		requestID := rec.Header().Get("X-Request-ID")
		if requestID != "test-request-123" {
			t.Errorf("expected request ID 'test-request-123', got '%s'", requestID)
		}
	})

	t.Run("includes CORS headers", func(t *testing.T) {
		mockService := &mockSecretService{}
		server := NewServer(cfg, mockService)

		reqBody := map[string]interface{}{
			"ciphertext": "dGVzdA",
			"nonce":      "MTIzNDU2Nzg5MDEy",
			"algorithm":  "AES-GCM",
			"ttlSeconds": 3600,
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.Handler().ServeHTTP(rec, req)

		origin := rec.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:5173" {
			t.Errorf("expected CORS origin 'http://localhost:5173', got '%s'", origin)
		}

		exposeHeaders := rec.Header().Get("Access-Control-Expose-Headers")
		wantExposeHeaders := "X-Request-ID, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, Retry-After"
		if exposeHeaders != wantExposeHeaders {
			t.Errorf("expected exposed headers %q, got %q", wantExposeHeaders, exposeHeaders)
		}
	})
}
