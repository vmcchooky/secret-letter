package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func runTest(server *httpapi.Server, sizeKB int) float64 {
	// Create payload of sizeKB
	cipher := make([]byte, sizeKB*1024)
	for i := range cipher { cipher[i] = 'A' }
	
	payload, _ := json.Marshal(map[string]interface{}{
		"ciphertext": string(cipher),
		"nonce":      "MTIzNDU2Nzg5MDEy",
		"algorithm":  "AES-GCM",
		"ttlSeconds": 3600,
	})

	concurrency := 20
	duration := 2 * time.Second
	
	latencies := make(chan time.Duration, 1000000)
	done := make(chan struct{})

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
					
					startReq := time.Now()
					server.Handler().ServeHTTP(w, req)
					lat := time.Since(startReq)
					
					if w.Result().StatusCode == http.StatusCreated {
						latencies <- lat
					}
				}
			}
		}()
	}

	time.Sleep(duration)
	close(done)
	time.Sleep(100 * time.Millisecond)
	close(latencies)
	
	var total time.Duration
	var count int
	for l := range latencies {
		total += l
		count++
	}
	
	if count == 0 { return 0 }
	return float64(total.Microseconds()) / float64(count) / 1000.0 // in ms
}

func main() {
	server := httpapi.NewServer(config.Config{
		ServiceName:     "test",
		MaxSecretSizeKB: 2000,
	}, &MockService{})
	
	sizes := []int{1, 10, 50, 100, 500, 1000}
	fmt.Println("SizeKB,AvgLatencyMs")
	
	for _, s := range sizes {
		// Warmup
		runTest(server, s)
		
		// Benchmark
		lat := runTest(server, s)
		fmt.Printf("%d,%.3f\n", s, lat)
	}
}
