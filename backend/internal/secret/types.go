package secret

import "time"

const (
	StatusActive   = "active"
	StatusConsumed = "consumed"
	StatusExpired  = "expired"
	StatusDeleted  = "deleted"
)

// CreateSecretRequest represents the incoming request to create a secret
type CreateSecretRequest struct {
	Content          string `json:"content,omitempty"`
	Ciphertext       string `json:"ciphertext,omitempty"`
	Nonce            string `json:"nonce,omitempty"`
	Algorithm        string `json:"algorithm,omitempty"`
	TTLSeconds       int    `json:"ttlSeconds,omitempty"`
	ExpiresInMinutes int    `json:"expiresInMinutes,omitempty"`
	BurnAfterRead    *bool  `json:"burnAfterRead,omitempty"`
	Theme            string `json:"theme,omitempty"`
}

// CreateSecretResponse represents the response after creating a secret
type CreateSecretResponse struct {
	SecretID  string `json:"secretId"`
	Token     string `json:"token,omitempty"`
	URL       string `json:"url,omitempty"`
	ExpiresAt string `json:"expiresAt"`
}

// SecretStatus represents the status of a secret
type SecretStatus struct {
	SecretID  string `json:"secretId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	Message   string `json:"message,omitempty"`
}

// SecretPayload is encrypted before storage in Redis.
type SecretPayload struct {
	Content       string    `json:"content,omitempty"`
	Ciphertext    string    `json:"ciphertext,omitempty"`
	Nonce         string    `json:"nonce,omitempty"`
	Algorithm     string    `json:"algorithm,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	ExpiresAt     time.Time `json:"expiresAt"`
	BurnAfterRead bool      `json:"burnAfterRead"`
	Theme         string    `json:"theme"`
}

// SecretMetadata is lifecycle-only data stored separately from the encrypted payload.
type SecretMetadata struct {
	TokenHash      string `json:"tokenHash"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
	ExpiresAt      string `json:"expiresAt"`
	CreatedAtUnix  int64  `json:"createdAtUnix"`
	ExpiresAtUnix  int64  `json:"expiresAtUnix"`
	ConsumedAt     string `json:"consumedAt,omitempty"`
	ConsumedAtUnix int64  `json:"consumedAtUnix,omitempty"`
	TTLSeconds     int    `json:"ttlSeconds"`
	ViewCount      int    `json:"viewCount"`
	BurnAfterRead  bool   `json:"burnAfterRead"`
	Theme          string `json:"theme"`
}

// AtRestEnvelope contains server-side encrypted payload bytes.
type AtRestEnvelope struct {
	Ciphertext string `json:"ciphertext"`
	Nonce      string `json:"nonce"`
	Algorithm  string `json:"algorithm"`
}

// OpenedSecret represents the SRS-style secret data returned on first open.
type OpenedSecret struct {
	Content       string `json:"content,omitempty"`
	CreatedAt     string `json:"createdAt"`
	ExpiresAt     string `json:"expiresAt"`
	Theme         string `json:"theme"`
	BurnAfterRead bool   `json:"burnAfterRead"`
}

// ConsumeSecretResponse represents the response when consuming a secret
type ConsumeSecretResponse struct {
	SecretID   string        `json:"secretId"`
	Ciphertext string        `json:"ciphertext,omitempty"`
	Nonce      string        `json:"nonce,omitempty"`
	Algorithm  string        `json:"algorithm,omitempty"`
	Content    string        `json:"content,omitempty"`
	Secret     *OpenedSecret `json:"secret,omitempty"`
	ConsumedAt string        `json:"consumedAt"`
}

// RevealSessionResponse represents the response when creating a reveal session.
type RevealSessionResponse struct {
	SessionID string `json:"sessionId"`
	SecretID  string `json:"secretId"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expiresAt"`
}

// HealthStatus represents the health status of a dependency
type HealthStatus struct {
	Store string `json:"store"`
	Mode  string `json:"mode"`
}
