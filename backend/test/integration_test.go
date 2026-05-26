package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const apiBaseURL = "http://localhost:8080"

type CreateSecretRequest struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	TTLSeconds int    `json:"ttlSeconds"`
}

type CreateSecretResponse struct {
	SecretID  string `json:"secretId"`
	ExpiresAt string `json:"expiresAt"`
}

type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}

func TestMilestone2Comprehensive(t *testing.T) {
	// Check if backend is running
	resp, err := http.Get(apiBaseURL + "/healthz")
	if err != nil {
		t.Skip("Backend not running, skipping integration tests")
		return
	}
	resp.Body.Close()

	t.Run("Size Limits", func(t *testing.T) {
		t.Run("small plaintext should pass", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1zbWFsbC1wbGFpbnRleHQ",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 201)
		})

		t.Run("medium plaintext ~5KB should pass", func(t *testing.T) {
			mediumText := strings.Repeat("a", 5000)
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: mediumText,
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 201)
		})

		t.Run("large plaintext ~10KB should pass", func(t *testing.T) {
			largeText := strings.Repeat("a", 10000)
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: largeText,
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 201)
		})

		t.Run("too large plaintext >15KB should fail", func(t *testing.T) {
			tooLargeText := strings.Repeat("a", 16000)
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: tooLargeText,
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 413)
		})
	})

	t.Run("TTL Validation", func(t *testing.T) {
		t.Run("TTL 3600 should pass", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC10dGwtMzYwMA",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 201)
		})

		t.Run("TTL 86400 should pass", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC10dGwtODY0MDA",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 86400,
			}, 201)
		})

		t.Run("TTL 604800 should pass", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC10dGwtNjA0ODAw",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 604800,
			}, 201)
		})

		t.Run("invalid TTL 7200 should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1pbnZhbGlkLXR0bA",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 7200,
			}, 400)
		})

		t.Run("zero TTL should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC16ZXJvLXR0bA",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 0,
			}, 400)
		})

		t.Run("negative TTL should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1uZWdhdGl2ZS10dGw",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: -1,
			}, 400)
		})
	})

	t.Run("Algorithm Validation", func(t *testing.T) {
		t.Run("AES-GCM should pass", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1hZXMtZ2Nt",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 201)
		})

		t.Run("AES-CBC should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1hZXMtY2Jj",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-CBC",
				TTLSeconds: 3600,
			}, 400)
		})

		t.Run("ChaCha20 should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1jaGFjaGEyMA",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "ChaCha20",
				TTLSeconds: 3600,
			}, 400)
		})

		t.Run("empty algorithm should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1lbXB0eS1hbGdv",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "",
				TTLSeconds: 3600,
			}, 400)
		})
	})

	t.Run("Nonce Validation", func(t *testing.T) {
		t.Run("12-byte nonce should pass", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC12YWxpZC1ub25jZQ",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 201)
		})

		t.Run("8-byte nonce should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1zaG9ydC1ub25jZQ",
				Nonce:      "MTIzNDU2Nzg",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 400)
		})

		t.Run("16-byte nonce should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1sb25nLW5vbmNl",
				Nonce:      "MTIzNDU2Nzg5MDEyMzQ1Ng",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 400)
		})

		t.Run("empty nonce should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "dGVzdC1lbXB0eS1ub25jZQ",
				Nonce:      "",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 400)
		})
	})

	t.Run("Ciphertext Validation", func(t *testing.T) {
		t.Run("empty ciphertext should fail", func(t *testing.T) {
			testCreateSecret(t, CreateSecretRequest{
				Ciphertext: "",
				Nonce:      "MTIzNDU2Nzg5MDEy",
				Algorithm:  "AES-GCM",
				TTLSeconds: 3600,
			}, 400)
		})
	})
}

func testCreateSecret(t *testing.T, req CreateSecretRequest, expectedStatus int) {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", apiBaseURL+"/api/secrets", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Request-ID", fmt.Sprintf("test-%d", time.Now().Unix()))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	if resp.StatusCode == 201 {
		var result CreateSecretResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if result.SecretID == "" {
			t.Error("secretId should not be empty")
		}

		if result.ExpiresAt == "" {
			t.Error("expiresAt should not be empty")
		}

		t.Logf("Created secret: %s, expires: %s", result.SecretID, result.ExpiresAt)
	}
}

type SecretStatus struct {
	SecretID  string `json:"secretId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ConsumeSecretResponse struct {
	SecretID   string `json:"secretId"`
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
	ConsumedAt string `json:"consumedAt"`
}

func TestMilestone3RevealFlow(t *testing.T) {
	// Check if backend is running
	resp, err := http.Get(apiBaseURL + "/healthz")
	if err != nil {
		t.Skip("Backend not running, skipping integration tests")
		return
	}
	resp.Body.Close()

	t.Run("Complete Reveal Flow", func(t *testing.T) {
		// Step 1: Create a secret
		createReq := CreateSecretRequest{
			Ciphertext: "dGVzdC1yZXZlYWwtZmxvdw",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}

		secretID := createSecretForTest(t, createReq)
		if secretID == "" {
			t.Fatal("failed to create secret")
		}

		// Step 2: Check status (should be pending)
		status := getSecretStatus(t, secretID)
		if status.Status != "pending" {
			t.Errorf("expected status 'pending', got '%s'", status.Status)
		}
		if status.SecretID != secretID {
			t.Errorf("expected secretId '%s', got '%s'", secretID, status.SecretID)
		}

		// Step 3: Consume the secret (first time should succeed)
		consumeResp := consumeSecret(t, secretID, 200)
		if consumeResp.SecretID != secretID {
			t.Errorf("expected secretId '%s', got '%s'", secretID, consumeResp.SecretID)
		}
		if consumeResp.Ciphertext != createReq.Ciphertext {
			t.Errorf("ciphertext mismatch")
		}
		if consumeResp.Nonce != createReq.Nonce {
			t.Errorf("nonce mismatch")
		}
		if consumeResp.Algorithm != createReq.Algorithm {
			t.Errorf("algorithm mismatch")
		}

		// Step 4: Try to consume again (should fail with 410)
		consumeSecret(t, secretID, 410)

		// Step 5: Check status again (should be not_found)
		status = getSecretStatus(t, secretID)
		if status.Status != "not_found" {
			t.Errorf("expected status 'not_found', got '%s'", status.Status)
		}
	})

	t.Run("Status Check for Non-Existent Secret", func(t *testing.T) {
		status := getSecretStatus(t, "non-existent-secret-id")
		if status.Status != "not_found" {
			t.Errorf("expected status 'not_found', got '%s'", status.Status)
		}
	})

	t.Run("Consume Non-Existent Secret", func(t *testing.T) {
		consumeSecret(t, "non-existent-secret-id", 410)
	})

	t.Run("Concurrent Consume Attempts", func(t *testing.T) {
		// Create a secret
		createReq := CreateSecretRequest{
			Ciphertext: "dGVzdC1jb25jdXJyZW50",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		}

		secretID := createSecretForTest(t, createReq)
		if secretID == "" {
			t.Fatal("failed to create secret")
		}

		// Try to consume concurrently
		results := make(chan int, 3)
		for i := 0; i < 3; i++ {
			go func() {
				resp := consumeSecretRaw(t, secretID)
				results <- resp.StatusCode
				resp.Body.Close()
			}()
		}

		// Collect results
		statusCodes := make([]int, 3)
		for i := 0; i < 3; i++ {
			statusCodes[i] = <-results
		}

		// Exactly one should succeed (200), others should fail (410)
		successCount := 0
		failCount := 0
		for _, code := range statusCodes {
			if code == 200 {
				successCount++
			} else if code == 410 {
				failCount++
			}
		}

		if successCount != 1 {
			t.Errorf("expected exactly 1 success, got %d", successCount)
		}
		if failCount != 2 {
			t.Errorf("expected exactly 2 failures, got %d", failCount)
		}
	})

	t.Run("Multiple Secrets Independent", func(t *testing.T) {
		// Create two secrets
		secret1ID := createSecretForTest(t, CreateSecretRequest{
			Ciphertext: "dGVzdC1zZWNyZXQtMQ",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		})

		secret2ID := createSecretForTest(t, CreateSecretRequest{
			Ciphertext: "dGVzdC1zZWNyZXQtMg",
			Nonce:      "MTIzNDU2Nzg5MDEy",
			Algorithm:  "AES-GCM",
			TTLSeconds: 3600,
		})

		// Consume first secret
		consumeSecret(t, secret1ID, 200)

		// Second secret should still be available
		status := getSecretStatus(t, secret2ID)
		if status.Status != "pending" {
			t.Errorf("secret2 should still be pending, got '%s'", status.Status)
		}

		// Consume second secret
		consumeSecret(t, secret2ID, 200)

		// Both should now be consumed
		status1 := getSecretStatus(t, secret1ID)
		status2 := getSecretStatus(t, secret2ID)

		if status1.Status != "not_found" {
			t.Errorf("secret1 should be not_found, got '%s'", status1.Status)
		}
		if status2.Status != "not_found" {
			t.Errorf("secret2 should be not_found, got '%s'", status2.Status)
		}
	})
}

func createSecretForTest(t *testing.T, req CreateSecretRequest) string {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", apiBaseURL+"/api/secrets", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var result CreateSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return result.SecretID
}

func getSecretStatus(t *testing.T, secretID string) SecretStatus {
	t.Helper()

	url := fmt.Sprintf("%s/api/secrets/%s/status", apiBaseURL, secretID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var status SecretStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	return status
}

func consumeSecret(t *testing.T, secretID string, expectedStatus int) ConsumeSecretResponse {
	t.Helper()

	resp := consumeSecretRaw(t, secretID)
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	if resp.StatusCode == 200 {
		var result ConsumeSecretResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		return result
	}

	return ConsumeSecretResponse{}
}

func consumeSecretRaw(t *testing.T, secretID string) *http.Response {
	t.Helper()

	url := fmt.Sprintf("%s/api/secrets/%s/consume", apiBaseURL, secretID)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return resp
}
