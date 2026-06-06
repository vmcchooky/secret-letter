package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/httpapi"
	"secret-letter/backend/internal/secret"

	"github.com/redis/go-redis/v9"
)

const (
	integrationRedisDB = 15
	requestTimeout     = 10 * time.Second
)

var integrationAtRestKey = []byte("0123456789abcdefghijklmnopqrstuv")

type integrationAPI struct {
	baseURL string
	client  *http.Client
}

type createSecretRequest struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	TTLSeconds int    `json:"ttlSeconds"`
}

type createSecretResponse struct {
	SecretID  string `json:"secretId"`
	ExpiresAt string `json:"expiresAt"`
}

type revealSessionResponse struct {
	SessionID string `json:"sessionId"`
	SecretID  string `json:"secretId"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
}

type secretStatus struct {
	SecretID  string `json:"secretId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Message   string `json:"message,omitempty"`
}

type consumeSecretResponse struct {
	SecretID   string `json:"secretId"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	ConsumedAt string `json:"consumedAt"`
}

type errorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

func newIntegrationAPI(t *testing.T) *integrationAPI {
	t.Helper()

	redisAddr := os.Getenv("REDIS_ADDR")
	if strings.TrimSpace(redisAddr) == "" {
		redisAddr = "127.0.0.1:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   integrationRedisDB,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available at %s, skipping integration tests: %v", redisAddr, err)
	}

	if err := redisClient.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("failed to clear Redis test database: %v", err)
	}

	secretService, err := secret.NewRedisServiceWithEncryptionKey(redisClient, integrationAtRestKey)
	if err != nil {
		t.Fatalf("failed to create Redis-backed secret service: %v", err)
	}

	server := httpapi.NewServer(config.Config{
		ServiceName:   "test-api",
		Host:          "127.0.0.1",
		Port:          "0",
		AllowedOrigin: "http://localhost:5173",
	}, secretService)

	testServer := httptest.NewServer(server.Handler())
	client := testServer.Client()
	client.Timeout = requestTimeout

	t.Cleanup(func() {
		testServer.Close()
		_ = redisClient.FlushDB(ctx).Err()
		_ = redisClient.Close()
	})

	return &integrationAPI{
		baseURL: testServer.URL,
		client:  client,
	}
}

func TestCreateSecretContract(t *testing.T) {
	api := newIntegrationAPI(t)

	t.Run("accepts a valid encrypted secret", func(t *testing.T) {
		response := createSecret(t, api, createSecretRequest{
			Ciphertext: "dGVzdC1jaXBoZXJ0ZXh0",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}, http.StatusCreated)

		if response.SecretID == "" {
			t.Fatal("expected secretId to be returned")
		}

		if response.ExpiresAt == "" {
			t.Fatal("expected expiresAt to be returned")
		}
	})

	t.Run("rejects invalid ttl", func(t *testing.T) {
		response := createSecretError(t, api, createSecretRequest{
			Ciphertext: "dGVzdC1pbnZhbGlkLXR0bA",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 7200,
		}, http.StatusBadRequest)

		if response.Error != "validation_failed" {
			t.Fatalf("expected validation_failed, got %q", response.Error)
		}
	})

	t.Run("rejects invalid algorithm", func(t *testing.T) {
		response := createSecretError(t, api, createSecretRequest{
			Ciphertext: "dGVzdC1pbnZhbGlkLWFsZ28",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-CBC",
			TTLSeconds: 3600,
		}, http.StatusBadRequest)

		if response.Error != "validation_failed" {
			t.Fatalf("expected validation_failed, got %q", response.Error)
		}
	})

	t.Run("rejects invalid nonce length", func(t *testing.T) {
		response := createSecretError(t, api, createSecretRequest{
			Ciphertext: "dGVzdC1pbnZhbGlkLW5vbmNl",
			Nonce:      "MTIzNDU2Nzg",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}, http.StatusBadRequest)

		if response.Error != "validation_failed" {
			t.Fatalf("expected validation_failed, got %q", response.Error)
		}
	})

	t.Run("rejects empty ciphertext", func(t *testing.T) {
		response := createSecretError(t, api, createSecretRequest{
			Ciphertext: "",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}, http.StatusBadRequest)

		if response.Error != "validation_failed" {
			t.Fatalf("expected validation_failed, got %q", response.Error)
		}
	})

	t.Run("rejects oversized request bodies", func(t *testing.T) {
		response := createSecretError(t, api, createSecretRequest{
			Ciphertext: strings.Repeat("a", 16*1024),
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}, http.StatusRequestEntityTooLarge)

		if response.Error != "payload_too_large" {
			t.Fatalf("expected payload_too_large, got %q", response.Error)
		}
	})
}

func TestRevealFlowIntegration(t *testing.T) {
	api := newIntegrationAPI(t)

	t.Run("create -> reveal-session -> open -> consumed", func(t *testing.T) {
		createRequest := createSecretRequest{
			Ciphertext: "dGVzdC1yZXZlYWwtZmxvdw",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}

		created := createSecret(t, api, createRequest, http.StatusCreated)
		status := getSecretStatus(t, api, created.SecretID, http.StatusOK)
		if status.Status != secret.StatusActive {
			t.Fatalf("expected status %q before open, got %q", secret.StatusActive, status.Status)
		}

		revealSession := createRevealSession(t, api, created.SecretID, http.StatusCreated)
		if revealSession.SessionID == "" {
			t.Fatal("expected reveal session id to be returned")
		}

		opened := openSecret(t, api, created.SecretID, revealSession.SessionID, "/open", http.StatusOK)
		if opened.SecretID != created.SecretID {
			t.Fatalf("expected secretId %q, got %q", created.SecretID, opened.SecretID)
		}
		if opened.Ciphertext != createRequest.Ciphertext {
			t.Fatalf("expected ciphertext to round-trip, got %q", opened.Ciphertext)
		}
		if opened.Nonce != createRequest.Nonce {
			t.Fatalf("expected nonce to round-trip, got %q", opened.Nonce)
		}
		if opened.Algorithm != createRequest.Algorithm {
			t.Fatalf("expected algorithm %q, got %q", createRequest.Algorithm, opened.Algorithm)
		}

		consumed := openSecretError(t, api, created.SecretID, revealSession.SessionID, "/open", http.StatusGone)
		if consumed.Error != "SECRET_CONSUMED" {
			t.Fatalf("expected SECRET_CONSUMED, got %q", consumed.Error)
		}

		status = getSecretStatus(t, api, created.SecretID, http.StatusOK)
		if status.Status != secret.StatusConsumed {
			t.Fatalf("expected status %q after open, got %q", secret.StatusConsumed, status.Status)
		}
	})

	t.Run("consume alias still opens a secret exactly once", func(t *testing.T) {
		createRequest := createSecretRequest{
			Ciphertext: "dGVzdC1jb25zdW1lLWFsaWFz",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}

		created := createSecret(t, api, createRequest, http.StatusCreated)
		opened := openSecret(t, api, created.SecretID, "", "/consume", http.StatusOK)
		if opened.SecretID != created.SecretID {
			t.Fatalf("expected secretId %q, got %q", created.SecretID, opened.SecretID)
		}

		consumed := openSecretError(t, api, created.SecretID, "", "/consume", http.StatusGone)
		if consumed.Error != "SECRET_CONSUMED" {
			t.Fatalf("expected SECRET_CONSUMED, got %q", consumed.Error)
		}
	})

	t.Run("unknown valid token reports not_found for status and 404 for open", func(t *testing.T) {
		unknownToken := strings.Repeat("a", 32)

		status := getSecretStatus(t, api, unknownToken, http.StatusOK)
		if status.Status != "not_found" {
			t.Fatalf("expected not_found, got %q", status.Status)
		}

		notFound := openSecretError(t, api, unknownToken, "", "/open", http.StatusNotFound)
		if notFound.Error != "SECRET_NOT_FOUND" {
			t.Fatalf("expected SECRET_NOT_FOUND, got %q", notFound.Error)
		}
	})

	t.Run("malformed token is rejected before reveal", func(t *testing.T) {
		invalidToken := "bad-token"

		statusError := getSecretStatusError(t, api, invalidToken, http.StatusBadRequest)
		if statusError.Error != "invalid_secret_id" {
			t.Fatalf("expected invalid_secret_id, got %q", statusError.Error)
		}

		openError := openSecretError(t, api, invalidToken, "", "/open", http.StatusBadRequest)
		if openError.Error != "invalid_secret_id" {
			t.Fatalf("expected invalid_secret_id, got %q", openError.Error)
		}
	})
}

func createSecret(t *testing.T, api *integrationAPI, request createSecretRequest, expectedStatus int) createSecretResponse {
	t.Helper()

	response := postJSON[createSecretResponse](t, api, "/api/secrets", request, expectedStatus, nil)
	if response.SecretID == "" {
		t.Fatal("expected secretId to be set")
	}
	return response
}

func createSecretError(t *testing.T, api *integrationAPI, request createSecretRequest, expectedStatus int) errorResponse {
	t.Helper()
	return postJSON[errorResponse](t, api, "/api/secrets", request, expectedStatus, nil)
}

func createRevealSession(t *testing.T, api *integrationAPI, secretID string, expectedStatus int) revealSessionResponse {
	t.Helper()
	return postJSON[revealSessionResponse](t, api, "/api/reveal-sessions", map[string]string{
		"secretId": secretID,
	}, expectedStatus, nil)
}

func getSecretStatus(t *testing.T, api *integrationAPI, secretID string, expectedStatus int) secretStatus {
	t.Helper()
	return getJSON[secretStatus](t, api, fmt.Sprintf("/api/secrets/%s/status", secretID), expectedStatus)
}

func getSecretStatusError(t *testing.T, api *integrationAPI, secretID string, expectedStatus int) errorResponse {
	t.Helper()
	return getJSON[errorResponse](t, api, fmt.Sprintf("/api/secrets/%s/status", secretID), expectedStatus)
}

func openSecret(t *testing.T, api *integrationAPI, secretID string, revealSessionID string, suffix string, expectedStatus int) consumeSecretResponse {
	t.Helper()
	return postJSON[consumeSecretResponse](t, api, fmt.Sprintf("/api/secrets/%s%s", secretID, suffix), map[string]string{}, expectedStatus, map[string]string{
		"X-Reveal-Session": revealSessionID,
	})
}

func openSecretError(t *testing.T, api *integrationAPI, secretID string, revealSessionID string, suffix string, expectedStatus int) errorResponse {
	t.Helper()
	return postJSON[errorResponse](t, api, fmt.Sprintf("/api/secrets/%s%s", secretID, suffix), map[string]string{}, expectedStatus, map[string]string{
		"X-Reveal-Session": revealSessionID,
	})
}

func getJSON[T any](t *testing.T, api *integrationAPI, requestPath string, expectedStatus int) T {
	t.Helper()

	request, err := http.NewRequest(http.MethodGet, api.baseURL+requestPath, nil)
	if err != nil {
		t.Fatalf("failed to create GET request: %v", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Request-ID", fmt.Sprintf("integration-%d", time.Now().UnixNano()))

	response, err := api.client.Do(request)
	if err != nil {
		t.Fatalf("GET %s failed: %v", requestPath, err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("GET %s expected status %d, got %d", requestPath, expectedStatus, response.StatusCode)
	}

	return decodeJSON[T](t, response)
}

func postJSON[T any](t *testing.T, api *integrationAPI, requestPath string, payload any, expectedStatus int, headers map[string]string) T {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal request payload: %v", err)
	}

	request, err := http.NewRequest(http.MethodPost, api.baseURL+requestPath, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create POST request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Request-ID", fmt.Sprintf("integration-%d", time.Now().UnixNano()))
	for key, value := range headers {
		if strings.TrimSpace(value) != "" {
			request.Header.Set(key, value)
		}
	}

	response, err := api.client.Do(request)
	if err != nil {
		t.Fatalf("POST %s failed: %v", requestPath, err)
	}
	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		t.Fatalf("POST %s expected status %d, got %d", requestPath, expectedStatus, response.StatusCode)
	}

	return decodeJSON[T](t, response)
}

func decodeJSON[T any](t *testing.T, response *http.Response) T {
	t.Helper()

	var decoded T
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	return decoded
}
