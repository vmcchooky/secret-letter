package secret

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestLoadAtRestKeyRequiresProductionKey(t *testing.T) {
	t.Setenv(appEnvEnv, "production")
	t.Setenv(encryptionKeyEnv, "")
	t.Setenv(legacyEncryptionKeyEnv, "")

	_, err := loadAtRestKey()
	if err == nil {
		t.Fatal("expected production key requirement error")
	}

	if !strings.Contains(err.Error(), encryptionKeyEnv) {
		t.Fatalf("expected error to mention %s, got %v", encryptionKeyEnv, err)
	}
}

func TestLoadAtRestKeyAcceptsConfiguredProductionKey(t *testing.T) {
	key := bytes.Repeat([]byte{7}, atRestKeyLength)

	t.Setenv(appEnvEnv, "production")
	t.Setenv(encryptionKeyEnv, encodeBase64URL(key))
	t.Setenv(legacyEncryptionKeyEnv, "")

	got, err := loadAtRestKey()
	if err != nil {
		t.Fatalf("loadAtRestKey failed: %v", err)
	}

	if !bytes.Equal(got, key) {
		t.Fatal("loaded key does not match configured key")
	}
}

func TestLoadAtRestKeyAcceptsConfiguredProductionKeyEncodings(t *testing.T) {
	key := []byte("0123456789abcdefghijklmnopqrstuv")
	if len(key) != atRestKeyLength {
		t.Fatalf("test key must be %d bytes, got %d", atRestKeyLength, len(key))
	}

	tests := []struct {
		name  string
		value string
		want  []byte
	}{
		{
			name:  "base64url",
			value: encodeBase64URL(key),
			want:  key,
		},
		{
			name:  "hex",
			value: hex.EncodeToString(key),
			want:  key,
		},
		{
			name:  "raw 32-byte string",
			value: string(key),
			want:  key,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(appEnvEnv, "production")
			t.Setenv(encryptionKeyEnv, tt.value)
			t.Setenv(legacyEncryptionKeyEnv, "")

			got, err := loadAtRestKey()
			if err != nil {
				t.Fatalf("loadAtRestKey failed: %v", err)
			}

			if !bytes.Equal(got, tt.want) {
				t.Fatalf("loaded key does not match configured key")
			}
		})
	}
}

func TestNewRedisServiceWithEncryptionKeyRejectsInvalidLength(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer client.Close()

	_, err := NewRedisServiceWithEncryptionKey(client, []byte("too-short"))
	if err == nil {
		t.Fatal("expected invalid key length error")
	}
}

func TestHashTokenRejectsMalformedTokens(t *testing.T) {
	tests := []string{
		"",
		"too-short",
		strings.Repeat("a", 201),
		"contains/slash_____________________",
		"contains.dot_______________________",
		"contains space_____________________",
		"contains%percent___________________",
	}

	for _, token := range tests {
		t.Run(token, func(t *testing.T) {
			_, err := hashToken(token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("expected ErrInvalidToken, got %v", err)
			}
		})
	}
}

// TestRedisServiceIntegration tests the Redis service with a real Redis instance
// This test requires Redis to be running on localhost:6379
// Skip this test if Redis is not available
func TestRedisServiceIntegration(t *testing.T) {
	// Try to connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()

	// Check if Redis is available
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration test")
		return
	}
	defer client.Close()

	service, err := NewRedisService(client)
	if err != nil {
		t.Fatalf("NewRedisService failed: %v", err)
	}

	t.Run("creates secret successfully", func(t *testing.T) {
		req := CreateSecretRequest{
			Content:    "top secret plaintext",
			TTLSeconds: 3600,
		}

		resp, err := service.CreateSecret(ctx, req)
		if err != nil {
			t.Fatalf("CreateSecret failed: %v", err)
		}

		if resp.SecretID == "" {
			t.Error("SecretID should not be empty")
		}

		if resp.ExpiresAt == "" {
			t.Error("ExpiresAt should not be empty")
		}

		tokenHash, err := hashToken(resp.SecretID)
		if err != nil {
			t.Fatalf("failed to hash token: %v", err)
		}

		payloadKey := secretPayloadKey(tokenHash)
		metadataKey := secretMetadataKey(tokenHash)

		// Verify only hashed-token keys exist in Redis.
		exists, err := client.Exists(ctx, payloadKey, metadataKey).Result()
		if err != nil {
			t.Fatalf("failed to check Redis keys: %v", err)
		}

		if exists != 2 {
			t.Errorf("payload and metadata keys should exist in Redis, got %d keys", exists)
		}

		rawKeyExists, err := client.Exists(ctx, "secret:"+resp.SecretID).Result()
		if err != nil {
			t.Fatalf("failed to check raw token key: %v", err)
		}

		if rawKeyExists != 0 {
			t.Error("raw token key should not exist in Redis")
		}

		payloadRaw, err := client.Get(ctx, payloadKey).Result()
		if err != nil {
			t.Fatalf("failed to read encrypted payload: %v", err)
		}

		if strings.Contains(payloadRaw, req.Content) {
			t.Error("encrypted payload should not contain plaintext content")
		}

		// Verify TTL is set
		ttl, err := client.TTL(ctx, payloadKey).Result()
		if err != nil {
			t.Fatalf("Failed to get TTL: %v", err)
		}

		if ttl <= 0 || ttl > time.Hour {
			t.Errorf("TTL should be between 0 and 1 hour, got %v", ttl)
		}

		// Cleanup
		client.Del(ctx, payloadKey, metadataKey)
	})

	t.Run("health check returns healthy when Redis is available", func(t *testing.T) {
		status := service.Health(ctx)

		if status.Store != "redis" {
			t.Errorf("expected store 'redis', got '%s'", status.Store)
		}

		if status.Mode != "healthy" {
			t.Errorf("expected mode 'healthy', got '%s'", status.Mode)
		}
	})

	t.Run("secret expires after TTL", func(t *testing.T) {
		req := CreateSecretRequest{
			Content:    "test expiring secret",
			TTLSeconds: 1, // 1 second for quick test
		}

		resp, err := service.CreateSecret(ctx, req)
		if err != nil {
			t.Fatalf("CreateSecret failed: %v", err)
		}

		tokenHash, err := hashToken(resp.SecretID)
		if err != nil {
			t.Fatalf("failed to hash token: %v", err)
		}

		payloadKey := secretPayloadKey(tokenHash)
		metadataKey := secretMetadataKey(tokenHash)

		// Verify secret exists
		exists, _ := client.Exists(ctx, payloadKey, metadataKey).Result()
		if exists != 2 {
			t.Error("payload and metadata should exist immediately after creation")
		}

		status, err := service.GetSecretStatus(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("GetSecretStatus failed: %v", err)
		}
		if status.Status != StatusActive {
			t.Fatalf("secret should be active immediately after creation, got %s", status.Status)
		}

		// Wait for expiration
		time.Sleep(2 * time.Second)

		status, err = service.GetSecretStatus(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("GetSecretStatus failed after expiration: %v", err)
		}

		if status.Status != StatusExpired {
			t.Errorf("secret should be expired, got %s", status.Status)
		}

		exists, _ = client.Exists(ctx, payloadKey).Result()
		if exists != 0 {
			t.Error("payload should be removed after expiration check")
		}

		exists, _ = client.Exists(ctx, metadataKey).Result()
		if exists != 1 {
			t.Error("metadata should remain after expiration")
		}

		client.Del(ctx, payloadKey, metadataKey)
	})

	t.Run("consumes secret exactly once", func(t *testing.T) {
		resp, err := service.CreateSecret(ctx, CreateSecretRequest{
			Content:    "open once",
			TTLSeconds: 3600,
		})
		if err != nil {
			t.Fatalf("CreateSecret failed: %v", err)
		}

		opened, err := service.ConsumeSecret(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("first ConsumeSecret failed: %v", err)
		}

		if opened.Content != "open once" {
			t.Errorf("expected content to round trip, got %q", opened.Content)
		}

		_, err = service.ConsumeSecret(ctx, resp.SecretID)
		if !errors.Is(err, ErrSecretConsumed) {
			t.Fatalf("second ConsumeSecret should return ErrSecretConsumed, got %v", err)
		}

		status, err := service.GetSecretStatus(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("GetSecretStatus failed: %v", err)
		}

		if status.Status != StatusConsumed {
			t.Errorf("expected consumed status, got %s", status.Status)
		}

		tokenHash, _ := hashToken(resp.SecretID)
		client.Del(ctx, secretPayloadKey(tokenHash), secretMetadataKey(tokenHash))
	})

	t.Run("decrypt failure does not consume or delete payload", func(t *testing.T) {
		keyA := bytes.Repeat([]byte{1}, atRestKeyLength)
		keyB := bytes.Repeat([]byte{2}, atRestKeyLength)

		serviceA, err := NewRedisServiceWithEncryptionKey(client, keyA)
		if err != nil {
			t.Fatalf("NewRedisServiceWithEncryptionKey A failed: %v", err)
		}

		serviceB, err := NewRedisServiceWithEncryptionKey(client, keyB)
		if err != nil {
			t.Fatalf("NewRedisServiceWithEncryptionKey B failed: %v", err)
		}

		resp, err := serviceA.CreateSecret(ctx, CreateSecretRequest{
			Content:    "survives wrong key",
			TTLSeconds: 3600,
		})
		if err != nil {
			t.Fatalf("CreateSecret failed: %v", err)
		}

		tokenHash, err := hashToken(resp.SecretID)
		if err != nil {
			t.Fatalf("failed to hash token: %v", err)
		}

		payloadKey := secretPayloadKey(tokenHash)
		metadataKey := secretMetadataKey(tokenHash)
		defer client.Del(ctx, payloadKey, metadataKey)

		_, err = serviceB.ConsumeSecret(ctx, resp.SecretID)
		if err == nil {
			t.Fatal("expected decrypt failure with wrong key")
		}

		if !strings.Contains(err.Error(), "failed to decrypt secret payload") {
			t.Fatalf("expected decrypt error, got %v", err)
		}

		exists, err := client.Exists(ctx, payloadKey).Result()
		if err != nil {
			t.Fatalf("failed to check payload key: %v", err)
		}
		if exists != 1 {
			t.Fatal("payload should remain after decrypt failure")
		}

		status, err := serviceA.GetSecretStatus(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("GetSecretStatus failed: %v", err)
		}
		if status.Status != StatusActive {
			t.Fatalf("secret should remain active after decrypt failure, got %s", status.Status)
		}

		opened, err := serviceA.ConsumeSecret(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("ConsumeSecret with original key failed: %v", err)
		}
		if opened.Content != "survives wrong key" {
			t.Fatalf("expected original content, got %q", opened.Content)
		}
	})

	t.Run("restart with same key decrypts existing payload", func(t *testing.T) {
		key := bytes.Repeat([]byte{3}, atRestKeyLength)

		serviceBeforeRestart, err := NewRedisServiceWithEncryptionKey(client, key)
		if err != nil {
			t.Fatalf("NewRedisServiceWithEncryptionKey before restart failed: %v", err)
		}

		resp, err := serviceBeforeRestart.CreateSecret(ctx, CreateSecretRequest{
			Content:    "stable key survives restart",
			TTLSeconds: 3600,
		})
		if err != nil {
			t.Fatalf("CreateSecret failed: %v", err)
		}

		tokenHash, err := hashToken(resp.SecretID)
		if err != nil {
			t.Fatalf("failed to hash token: %v", err)
		}
		defer client.Del(ctx, secretPayloadKey(tokenHash), secretMetadataKey(tokenHash))

		serviceAfterRestart, err := NewRedisServiceWithEncryptionKey(client, key)
		if err != nil {
			t.Fatalf("NewRedisServiceWithEncryptionKey after restart failed: %v", err)
		}

		opened, err := serviceAfterRestart.ConsumeSecret(ctx, resp.SecretID)
		if err != nil {
			t.Fatalf("ConsumeSecret after restart failed: %v", err)
		}

		if opened.Content != "stable key survives restart" {
			t.Fatalf("expected content to survive restart, got %q", opened.Content)
		}
	})

	t.Run("returns not found for unknown valid token", func(t *testing.T) {
		unknownToken := strings.Repeat("a", 32)

		_, err := service.ConsumeSecret(ctx, unknownToken)
		if !errors.Is(err, ErrSecretNotFound) {
			t.Fatalf("expected ErrSecretNotFound, got %v", err)
		}
	})

	t.Run("simultaneous open race has exactly one winner", func(t *testing.T) {
		resp, err := service.CreateSecret(ctx, CreateSecretRequest{
			Content:    "race secret",
			TTLSeconds: 3600,
		})
		if err != nil {
			t.Fatalf("CreateSecret failed: %v", err)
		}

		const attempts = 12
		var wg sync.WaitGroup
		errs := make(chan error, attempts)

		for i := 0; i < attempts; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := service.ConsumeSecret(ctx, resp.SecretID)
				errs <- err
			}()
		}

		wg.Wait()
		close(errs)

		successes := 0
		consumed := 0
		for err := range errs {
			switch {
			case err == nil:
				successes++
			case errors.Is(err, ErrSecretConsumed):
				consumed++
			default:
				t.Fatalf("unexpected open result: %v", err)
			}
		}

		if successes != 1 {
			t.Errorf("expected exactly one successful open, got %d", successes)
		}
		if consumed != attempts-1 {
			t.Errorf("expected %d consumed failures, got %d", attempts-1, consumed)
		}

		tokenHash, _ := hashToken(resp.SecretID)
		client.Del(ctx, secretPayloadKey(tokenHash), secretMetadataKey(tokenHash))
	})
}
