package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/httpapi"
	"secret-letter/backend/internal/secret"
)

// MockService simulates a fast backend store
type MockService struct{}

func (s *MockService) Health(ctx context.Context) secret.HealthStatus {
	return secret.HealthStatus{Store: "mock", Mode: "ok"}
}

func (s *MockService) CreateSecret(ctx context.Context, req secret.CreateSecretRequest) (*secret.CreateSecretResponse, error) {
	return &secret.CreateSecretResponse{
		SecretID:  "mock-secret-id",
		ExpiresAt: time.Now().Add(time.Hour).Format(time.RFC3339),
	}, nil
}

func (s *MockService) GetSecretStatus(ctx context.Context, secretID string) (*secret.SecretStatus, error) {
	return nil, nil
}

func (s *MockService) ConsumeSecret(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) {
	return nil, nil
}

func BenchmarkCreateSecretAPI(b *testing.B) {
	// Setup fast mock service
	secretService := &MockService{}
	server := httpapi.NewServer(config.Config{
		ServiceName:   "benchmark-api",
		Host:          "127.0.0.1",
		Port:          "0",
		AllowedOrigin: "*",
	}, secretService)

	payload, _ := json.Marshal(map[string]interface{}{
		"ciphertext": "dGVzdC1sb2FkLXRlc3Q=",
		"nonce":      "MTIzNDU2Nzg5MDEy",
		"algorithm":  "AES-GCM",
		"ttlSeconds": 3600,
	})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.Handler().ServeHTTP(w, req)
			if w.Result().StatusCode != http.StatusCreated {
				b.Fatalf("expected 201, got %d", w.Result().StatusCode)
			}
		}
	})
}
