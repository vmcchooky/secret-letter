package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/httpapi"
	"secret-letter/backend/internal/secret"
)

type MockService struct{}
func (s *MockService) Health(ctx context.Context) secret.HealthStatus { return secret.HealthStatus{} }
func (s *MockService) CreateSecret(ctx context.Context, req secret.CreateSecretRequest) (*secret.CreateSecretResponse, error) {
	return &secret.CreateSecretResponse{SecretID: "mock", ExpiresAt: ""}, nil
}
func (s *MockService) GetSecretStatus(ctx context.Context, secretID string) (*secret.SecretStatus, error) { return nil, nil }
func (s *MockService) ConsumeSecret(ctx context.Context, secretID string) (*secret.ConsumeSecretResponse, error) { return nil, nil }

func main() {
	server := httpapi.NewServer(config.Config{
		ServiceName:     "test",
		MaxSecretSizeKB: 2000,
	}, &MockService{})

	// Create payload (10KB typical size)
	cipher := make([]byte, 10*1024)
	for i := range cipher { cipher[i] = 'X' }
	payload, _ := json.Marshal(map[string]interface{}{
		"ciphertext": string(cipher),
		"nonce":      "MTIzNDU2Nzg5MDEy",
		"algorithm":  "AES-GCM",
		"ttlSeconds": 3600,
	})

	concurrency := 100
	done := make(chan struct{})

	// Blast requests
	for i := 0; i < concurrency; i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
					req := httptest.NewRequest(http.MethodPost, "/api/secrets", bytes.NewBuffer(payload))
					req.Header.Set("Content-Type", "application/json")
					w := httptest.NewRecorder()
					server.Handler().ServeHTTP(w, req)
				}
			}
		}()
	}

	fmt.Println("TimeSeconds,MemoryMB")
	
	// Record memory every 1 second for 15 seconds
	var memStats runtime.MemStats
	for i := 0; i <= 15; i++ {
		runtime.ReadMemStats(&memStats)
		// Convert bytes to MB
		fmt.Printf("%d,%.2f\n", i, float64(memStats.Alloc)/1024/1024)
		if i < 15 {
			time.Sleep(1 * time.Second)
		}
	}
	close(done)
}
