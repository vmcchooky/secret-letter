//go:build ignore

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/httpapi"
	"secret-letter/backend/internal/secret"
)

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

func runTest(server *httpapi.Server, concurrency int, duration time.Duration) float64 {
	payload, _ := json.Marshal(map[string]interface{}{
		"ciphertext": "dGVzdC1sb2FkLXRlc3Q=",
		"nonce":      "MTIzNDU2Nzg5MDEy",
		"algorithm":  "AES-GCM",
		"ttlSeconds": 3600,
	})

	var counter uint64
	done := make(chan struct{})

	// Start workers
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
					
					if w.Result().StatusCode == http.StatusCreated {
						atomic.AddUint64(&counter, 1)
					}
				}
			}
		}()
	}

	start := time.Now()
	time.Sleep(duration)
	close(done)
	
	elapsed := time.Since(start).Seconds()
	finalCount := atomic.LoadUint64(&counter)
	return float64(finalCount) / elapsed
}

func main() {
	secretService := &MockService{}
	server := httpapi.NewServer(config.Config{
		ServiceName:   "benchmark-api",
		Host:          "127.0.0.1",
		Port:          "0",
		AllowedOrigin: "*",
	}, secretService)

	concurrencies := []int{10, 50, 100, 200, 500, 1000}
	fmt.Println("Concurrency,RPS")
	
	for _, c := range concurrencies {
		// warmup
		runTest(server, c, 1*time.Second)
		
		// actual test
		rps := runTest(server, c, 3*time.Second)
		fmt.Printf("%d,%.2f\n", c, rps)
	}
}
