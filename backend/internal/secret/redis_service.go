package secret

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// RedisOperationTimeout is the default timeout for Redis operations.
	RedisOperationTimeout = 5 * time.Second

	atRestAlgorithm        = "AES-GCM"
	atRestKeyLength        = 32
	tokenLengthBytes       = 16
	minTokenLength         = 16
	metadataRetention      = 7 * 24 * time.Hour
	openClaimTTL           = 30 * time.Second
	appEnvEnv              = "APP_ENV"
	encryptionKeyEnv       = "SECRET_ENCRYPTION_KEY"
	legacyEncryptionKeyEnv = "OTL_ENCRYPTION_KEY"
)

var claimOpenSecretScript = redis.NewScript(`
local metaRaw = redis.call("GET", KEYS[1])
if not metaRaw then
	return {"not_found"}
end

local meta = cjson.decode(metaRaw)
local now = tonumber(ARGV[1])
local retention = tonumber(ARGV[2])
local claimID = ARGV[3]
local claimUntil = tonumber(ARGV[4])

if meta.status == "consumed" then
	return {"consumed"}
end

if meta.status == "expired" then
	return {"expired"}
end

if meta.status == "deleted" then
	return {"not_found"}
end

if meta.status == "opening" then
	local openingUntil = tonumber(meta.openingUntilUnix) or 0
	if openingUntil > now then
		return {"consumed"}
	end

	meta.status = "active"
	meta.openingID = nil
	meta.openingUntilUnix = nil
end

if tonumber(meta.expiresAtUnix) <= now then
	meta.status = "expired"
	meta.openingID = nil
	meta.openingUntilUnix = nil
	redis.call("DEL", KEYS[2])
	redis.call("SET", KEYS[1], cjson.encode(meta), "EX", retention)
	return {"expired"}
end

local payloadRaw = redis.call("GET", KEYS[2])
if not payloadRaw then
	meta.status = "expired"
	meta.openingID = nil
	meta.openingUntilUnix = nil
	redis.call("SET", KEYS[1], cjson.encode(meta), "EX", retention)
	return {"expired"}
end

meta.status = "opening"
meta.openingID = claimID
meta.openingUntilUnix = claimUntil
redis.call("SET", KEYS[1], cjson.encode(meta), "KEEPTTL")

return {"ok", payloadRaw}
`)

var finalizeOpenSecretScript = redis.NewScript(`
local metaRaw = redis.call("GET", KEYS[1])
if not metaRaw then
	return {"not_found"}
end

local meta = cjson.decode(metaRaw)
local retention = tonumber(ARGV[1])
local consumedAt = ARGV[2]
local consumedAtUnix = tonumber(ARGV[3])
local claimID = ARGV[4]

if meta.status == "consumed" then
	return {"consumed"}
end

if meta.status == "expired" then
	return {"expired"}
end

if meta.status == "deleted" then
	return {"not_found"}
end

if meta.status ~= "opening" or meta.openingID ~= claimID then
	return {"conflict"}
end

if redis.call("EXISTS", KEYS[2]) == 0 then
	meta.status = "expired"
	meta.openingID = nil
	meta.openingUntilUnix = nil
	redis.call("SET", KEYS[1], cjson.encode(meta), "EX", retention)
	return {"expired"}
end

meta.status = "consumed"
meta.openingID = nil
meta.openingUntilUnix = nil
meta.consumedAtUnix = consumedAtUnix
meta.consumedAt = consumedAt
meta.viewCount = (tonumber(meta.viewCount) or 0) + 1

local updatedMeta = cjson.encode(meta)
redis.call("DEL", KEYS[2])
redis.call("SET", KEYS[1], updatedMeta, "EX", retention)

return {"ok", updatedMeta}
`)

var abortOpenSecretScript = redis.NewScript(`
local metaRaw = redis.call("GET", KEYS[1])
if not metaRaw then
	return {"not_found"}
end

local meta = cjson.decode(metaRaw)
local now = tonumber(ARGV[1])
local retention = tonumber(ARGV[2])
local claimID = ARGV[3]

if meta.status ~= "opening" or meta.openingID ~= claimID then
	return {"ignored"}
end

meta.openingID = nil
meta.openingUntilUnix = nil

if tonumber(meta.expiresAtUnix) <= now then
	meta.status = "expired"
	redis.call("DEL", KEYS[2])
	redis.call("SET", KEYS[1], cjson.encode(meta), "EX", retention)
	return {"expired"}
end

meta.status = "active"
redis.call("SET", KEYS[1], cjson.encode(meta), "KEEPTTL")
return {"ok"}
`)

// RedisService implements secret storage using Redis.
type RedisService struct {
	client    *redis.Client
	atRestKey []byte
}

// NewRedisService creates a new Redis-backed secret service.
func NewRedisService(client *redis.Client) (*RedisService, error) {
	key, err := loadAtRestKey()
	if err != nil {
		return nil, err
	}

	return NewRedisServiceWithEncryptionKey(client, key)
}

// NewRedisServiceWithEncryptionKey creates a Redis service with an explicit 256-bit AES-GCM key.
func NewRedisServiceWithEncryptionKey(client *redis.Client, key []byte) (*RedisService, error) {
	if len(key) != atRestKeyLength {
		return nil, fmt.Errorf("secret at-rest encryption key must be %d bytes", atRestKeyLength)
	}

	keyCopy := make([]byte, len(key))
	copy(keyCopy, key)

	return &RedisService{
		client:    client,
		atRestKey: keyCopy,
	}, nil
}

// CreateSecret stores a secret in Redis with a random token, hashed token keying, and encrypted payload.
func (s *RedisService) CreateSecret(ctx context.Context, req CreateSecretRequest) (*CreateSecretResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, RedisOperationTimeout)
	defer cancel()

	token, tokenHash, err := generateTokenAndHash()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret token: %w", err)
	}

	ttlSeconds := EffectiveTTLSeconds(req)
	if ttlSeconds <= 0 {
		return nil, fmt.Errorf("ttl must be positive")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(ttlSeconds) * time.Second)

	payload := SecretPayload{
		Content:       req.Content,
		Ciphertext:    req.Ciphertext,
		Nonce:         req.Nonce,
		Algorithm:     req.Algorithm,
		CreatedAt:     now,
		ExpiresAt:     expiresAt,
		BurnAfterRead: EffectiveBurnAfterRead(req),
		Theme:         EffectiveTheme(req),
	}

	envelope, err := s.encryptPayload(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret payload: %w", err)
	}

	payloadData, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal encrypted secret payload: %w", err)
	}

	metadata := SecretMetadata{
		TokenHash:     tokenHash,
		Status:        StatusActive,
		CreatedAt:     now.Format(time.RFC3339),
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		CreatedAtUnix: now.Unix(),
		ExpiresAtUnix: expiresAt.Unix(),
		TTLSeconds:    ttlSeconds,
		ViewCount:     0,
		BurnAfterRead: payload.BurnAfterRead,
		Theme:         payload.Theme,
	}

	metadataData, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret metadata: %w", err)
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, secretPayloadKey(tokenHash), payloadData, ttl)
	pipe.Set(ctx, secretMetadataKey(tokenHash), metadataData, ttl+metadataRetention)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to store secret in Redis: %w", err)
	}

	return &CreateSecretResponse{
		SecretID:  token,
		Token:     token,
		URL:       "/" + token,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

// Health checks Redis connectivity.
func (s *RedisService) Health(ctx context.Context) HealthStatus {
	ctx, cancel := context.WithTimeout(ctx, RedisOperationTimeout)
	defer cancel()

	err := s.client.Ping(ctx).Err()
	if err != nil {
		return HealthStatus{
			Store: "redis",
			Mode:  "unhealthy",
		}
	}

	return HealthStatus{
		Store: "redis",
		Mode:  "healthy",
	}
}

// GetSecretStatus checks if a secret exists and returns its lifecycle status.
func (s *RedisService) GetSecretStatus(ctx context.Context, token string) (*SecretStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, RedisOperationTimeout)
	defer cancel()

	tokenHash, err := hashToken(token)
	if err != nil {
		return nil, err
	}

	metadata, err := s.getMetadata(ctx, tokenHash)
	if err == redis.Nil {
		return &SecretStatus{
			SecretID: token,
			Status:   "not_found",
			Message:  "No secret letter was found here.",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret metadata from Redis: %w", err)
	}

	if metadata.Status == StatusActive && metadata.ExpiresAtUnix <= time.Now().UTC().Unix() {
		metadata.Status = StatusExpired
		if err := s.persistMetadata(ctx, tokenHash, metadata, metadataRetention); err != nil {
			return nil, fmt.Errorf("failed to mark secret expired: %w", err)
		}
		_ = s.client.Del(ctx, secretPayloadKey(tokenHash)).Err()
	}

	return &SecretStatus{
		SecretID:  token,
		Status:    metadata.Status,
		CreatedAt: metadata.CreatedAt,
		ExpiresAt: metadata.ExpiresAt,
		Message:   statusMessage(metadata.Status),
	}, nil
}

// ConsumeSecret atomically opens a secret exactly once.
func (s *RedisService) ConsumeSecret(ctx context.Context, token string) (*ConsumeSecretResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, RedisOperationTimeout)
	defer cancel()

	tokenHash, err := hashToken(token)
	if err != nil {
		return nil, err
	}

	claimID, err := generateOpenClaimID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate open claim: %w", err)
	}

	now := time.Now().UTC()
	result, err := claimOpenSecretScript.Run(ctx, s.client, []string{
		secretMetadataKey(tokenHash),
		secretPayloadKey(tokenHash),
	}, now.Unix(), int(metadataRetention.Seconds()), claimID, now.Add(openClaimTTL).Unix()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to open secret atomically: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("unexpected Redis script response")
	}

	code, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected Redis script status")
	}

	switch code {
	case "ok":
		if len(values) < 2 {
			return nil, fmt.Errorf("missing Redis script payload")
		}

		payloadRaw, ok := values[1].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected Redis script payload type")
		}

		payload, err := s.decryptPayload([]byte(payloadRaw))
		if err != nil {
			_ = s.abortOpenClaim(ctx, tokenHash, claimID)
			return nil, fmt.Errorf("failed to decrypt secret payload: %w", err)
		}

		metadata, err := s.finalizeOpenClaim(ctx, tokenHash, claimID, time.Now().UTC())
		if err != nil {
			return nil, err
		}

		response := &ConsumeSecretResponse{
			SecretID:   token,
			Ciphertext: payload.Ciphertext,
			Nonce:      payload.Nonce,
			Algorithm:  payload.Algorithm,
			Content:    payload.Content,
			ConsumedAt: metadata.ConsumedAt,
		}

		if payload.Content != "" {
			response.Secret = &OpenedSecret{
				Content:       payload.Content,
				CreatedAt:     payload.CreatedAt.Format(time.RFC3339),
				ExpiresAt:     payload.ExpiresAt.Format(time.RFC3339),
				Theme:         payload.Theme,
				BurnAfterRead: payload.BurnAfterRead,
			}
		}

		return response, nil
	case "consumed":
		return nil, ErrSecretConsumed
	case "expired":
		return nil, ErrSecretExpired
	case "not_found":
		return nil, ErrSecretNotFound
	default:
		return nil, fmt.Errorf("unexpected Redis script status %q", code)
	}
}

func (s *RedisService) finalizeOpenClaim(ctx context.Context, tokenHash string, claimID string, consumedAt time.Time) (*SecretMetadata, error) {
	result, err := finalizeOpenSecretScript.Run(ctx, s.client, []string{
		secretMetadataKey(tokenHash),
		secretPayloadKey(tokenHash),
	}, int(metadataRetention.Seconds()), consumedAt.Format(time.RFC3339), consumedAt.Unix(), claimID).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to finalize opened secret: %w", err)
	}

	values, ok := result.([]interface{})
	if !ok || len(values) == 0 {
		return nil, fmt.Errorf("unexpected Redis finalize script response")
	}

	code, ok := values[0].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected Redis finalize script status")
	}

	switch code {
	case "ok":
		if len(values) < 2 {
			return nil, fmt.Errorf("missing Redis finalize metadata")
		}

		metadataRaw, ok := values[1].(string)
		if !ok {
			return nil, fmt.Errorf("unexpected Redis finalize metadata type")
		}

		var metadata SecretMetadata
		if err := json.Unmarshal([]byte(metadataRaw), &metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal opened secret metadata: %w", err)
		}

		return &metadata, nil
	case "consumed":
		return nil, ErrSecretConsumed
	case "expired":
		return nil, ErrSecretExpired
	case "not_found":
		return nil, ErrSecretNotFound
	case "conflict":
		return nil, fmt.Errorf("open claim was superseded before finalize")
	default:
		return nil, fmt.Errorf("unexpected Redis finalize script status %q", code)
	}
}

func (s *RedisService) abortOpenClaim(ctx context.Context, tokenHash string, claimID string) error {
	now := time.Now().UTC()
	_, err := abortOpenSecretScript.Run(ctx, s.client, []string{
		secretMetadataKey(tokenHash),
		secretPayloadKey(tokenHash),
	}, now.Unix(), int(metadataRetention.Seconds()), claimID).Result()
	if err != nil {
		return fmt.Errorf("failed to abort opened secret claim: %w", err)
	}

	return nil
}

func (s *RedisService) getMetadata(ctx context.Context, tokenHash string) (*SecretMetadata, error) {
	data, err := s.client.Get(ctx, secretMetadataKey(tokenHash)).Bytes()
	if err != nil {
		return nil, err
	}

	var metadata SecretMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret metadata: %w", err)
	}

	return &metadata, nil
}

func (s *RedisService) persistMetadata(ctx context.Context, tokenHash string, metadata *SecretMetadata, ttl time.Duration) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, secretMetadataKey(tokenHash), data, ttl).Err()
}

func (s *RedisService) encryptPayload(payload SecretPayload) (*AtRestEnvelope, error) {
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.atRestKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// To prevent AES-GCM nonce collision (birthday paradox), we use 8 bytes of time + 4 bytes of randomness
	nonce := make([]byte, gcm.NonceSize())
	binary.BigEndian.PutUint64(nonce[:8], uint64(time.Now().UnixNano()))
	if _, err := io.ReadFull(rand.Reader, nonce[8:]); err != nil {
		return nil, err
	}
	// #nosec G407 -- False positive: Nonce is dynamically generated using time and randomness
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &AtRestEnvelope{
		Ciphertext: encodeBase64URL(ciphertext),
		Nonce:      encodeBase64URL(nonce),
		Algorithm:  atRestAlgorithm,
	}, nil
}

func (s *RedisService) decryptPayload(data []byte) (*SecretPayload, error) {
	var envelope AtRestEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}

	if envelope.Algorithm != atRestAlgorithm {
		return nil, fmt.Errorf("unsupported at-rest encryption algorithm")
	}

	nonce, err := decodeBase64URL(envelope.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := decodeBase64URL(envelope.Ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(s.atRestKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var payload SecretPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func loadAtRestKey() ([]byte, error) {
	for _, keyName := range []string{encryptionKeyEnv, legacyEncryptionKeyEnv} {
		value := os.Getenv(keyName)
		if value == "" {
			continue
		}

		key, err := parseAtRestKey(value)
		if err != nil {
			return nil, fmt.Errorf("%s is invalid: %w", keyName, err)
		}

		return key, nil
	}

	if isProductionEnv(os.Getenv(appEnvEnv)) {
		return nil, fmt.Errorf("%s must be set when %s=production", encryptionKeyEnv, appEnvEnv)
	}

	key, err := secureRandomBytes(atRestKeyLength)
	if err != nil {
		return nil, err
	}

	log.Printf("warning: %s is not set; generated an ephemeral development key, existing secrets will not survive process restart", encryptionKeyEnv)
	return key, nil
}

func isProductionEnv(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "production" || value == "prod"
}

func parseAtRestKey(value string) ([]byte, error) {
	// Strict checking against Base64 RawURL and Hex to prevent timing side channels
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil && len(decoded) == atRestKeyLength {
		return decoded, nil
	}

	if decoded, err := hex.DecodeString(value); err == nil && len(decoded) == atRestKeyLength {
		return decoded, nil
	}

	if len([]byte(value)) == atRestKeyLength {
		return []byte(value), nil
	}

	return nil, fmt.Errorf("expected 32 raw bytes, hex, or base64-encoded 32 bytes")
}

func generateTokenAndHash() (string, string, error) {
	tokenBytes, err := secureRandomBytes(tokenLengthBytes)
	if err != nil {
		return "", "", err
	}

	token := encodeBase64URL(tokenBytes)
	tokenHash, err := hashToken(token)
	if err != nil {
		return "", "", err
	}

	return token, tokenHash, nil
}

func generateOpenClaimID() (string, error) {
	claimBytes, err := secureRandomBytes(16)
	if err != nil {
		return "", err
	}

	return encodeBase64URL(claimBytes), nil
}

func secureRandomBytes(length int) ([]byte, error) {
	randomBytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
		return nil, err
	}

	return randomBytes, nil
}

var sha256Pool = sync.Pool{
	New: func() interface{} {
		return sha256.New()
	},
}

func hashToken(token string) (string, error) {
	if !isValidToken(token) {
		return "", ErrInvalidToken
	}

	h := sha256Pool.Get().(hash.Hash)
	defer sha256Pool.Put(h)
	
	h.Reset()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil)), nil
}

func isValidToken(token string) bool {
	if len(token) < minTokenLength || len(token) > 200 {
		return false
	}

	for _, ch := range token {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-' || ch == '_':
		default:
			return false
		}
	}

	return true
}

func secretPayloadKey(tokenHash string) string {
	return "secret:payload:" + tokenHash
}

func secretMetadataKey(tokenHash string) string {
	return "secret:meta:" + tokenHash
}

func encodeBase64URL(bytes []byte) string {
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func decodeBase64URL(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

func statusMessage(status string) string {
	switch status {
	case StatusActive:
		return "A secret letter is waiting for you."
	case StatusConsumed:
		return "This secret has already vanished."
	case StatusExpired:
		return "This secret has expired."
	default:
		return "No secret letter was found here."
	}
}
