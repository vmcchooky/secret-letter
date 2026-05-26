package secret

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	revealSessionTTL    = 15 * time.Minute
	revealSessionPrefix = "reveal:session:"
	revealSessionStatus = "active"
)

type revealSessionRecord struct {
	SessionID     string `json:"sessionId"`
	TokenHash     string `json:"tokenHash"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	ExpiresAt     string `json:"expiresAt"`
	CreatedAtUnix int64  `json:"createdAtUnix"`
	ExpiresAtUnix int64  `json:"expiresAtUnix"`
}

func (s *RedisService) CreateRevealSession(ctx context.Context, secretID string) (*RevealSessionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, RedisOperationTimeout)
	defer cancel()

	status, err := s.GetSecretStatus(ctx, secretID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect secret status before creating reveal session: %w", err)
	}

	switch status.Status {
	case StatusActive:
		// continue
	case StatusConsumed:
		return nil, ErrSecretConsumed
	case StatusExpired:
		return nil, ErrSecretExpired
	case StatusDeleted, "not_found":
		return nil, ErrSecretNotFound
	default:
		return nil, fmt.Errorf("secret is not available for reveal sessions")
	}

	tokenHash, err := hashToken(secretID)
	if err != nil {
		return nil, err
	}

	sessionID, err := generateOpenClaimID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reveal session id: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(revealSessionTTL)
	record := revealSessionRecord{
		SessionID:     sessionID,
		TokenHash:     tokenHash,
		Status:        revealSessionStatus,
		CreatedAt:     now.Format(time.RFC3339),
		ExpiresAt:     expiresAt.Format(time.RFC3339),
		CreatedAtUnix: now.Unix(),
		ExpiresAtUnix: expiresAt.Unix(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal reveal session: %w", err)
	}

	if err := s.client.Set(ctx, revealSessionKey(sessionID), data, revealSessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("failed to store reveal session in Redis: %w", err)
	}

	return &RevealSessionResponse{
		SessionID: sessionID,
		SecretID:  secretID,
		Status:    revealSessionStatus,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}, nil
}

func (s *RedisService) ValidateRevealSession(ctx context.Context, secretID, sessionID string) error {
	ctx, cancel := context.WithTimeout(ctx, RedisOperationTimeout)
	defer cancel()

	if sessionID == "" {
		return ErrInvalidToken
	}

	tokenHash, err := hashToken(secretID)
	if err != nil {
		return err
	}

	record, err := s.getRevealSession(ctx, sessionID)
	if err != nil {
		if err == redis.Nil {
			return ErrSecretNotFound
		}
		return fmt.Errorf("failed to read reveal session from Redis: %w", err)
	}

	if record.TokenHash != tokenHash {
		return ErrInvalidToken
	}

	now := time.Now().UTC().Unix()
	if record.Status != revealSessionStatus || record.ExpiresAtUnix <= now {
		_ = s.client.Del(ctx, revealSessionKey(sessionID)).Err()
		return ErrSecretExpired
	}

	return nil
}

func (s *RedisService) getRevealSession(ctx context.Context, sessionID string) (*revealSessionRecord, error) {
	data, err := s.client.Get(ctx, revealSessionKey(sessionID)).Bytes()
	if err != nil {
		return nil, err
	}

	var record revealSessionRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reveal session: %w", err)
	}

	return &record, nil
}

func revealSessionKey(sessionID string) string {
	return revealSessionPrefix + sessionID
}
