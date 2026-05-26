package secret

import (
	"encoding/base64"
	"errors"
	"fmt"
)

// Validation errors with detailed messages
var (
	ErrInvalidAlgorithm      = errors.New("algorithm must be 'AES-GCM'")
	ErrInvalidTTL            = errors.New("ttlSeconds must be one of: 3600 (1 hour), 86400 (24 hours), or 604800 (7 days)")
	ErrInvalidNonceLength    = errors.New("nonce must be exactly 12 bytes when base64-decoded (16 base64url characters)")
	ErrCiphertextTooLarge    = errors.New("ciphertext exceeds maximum size of 15KB")
	ErrEmptyCiphertext       = errors.New("ciphertext is required and cannot be empty")
	ErrEmptyNonce            = errors.New("nonce is required and cannot be empty")
	ErrInvalidNonceFormat    = errors.New("nonce must be valid base64url-encoded string")
	ErrPlaintextNotAllowed   = errors.New("plaintext content is not accepted by the public API")
	ErrBurnAfterReadRequired = errors.New("one-time links always burn after read")
)

const (
	MaxCiphertextSize = 15 * 1024 // 15KB (includes base64 overhead)
	NonceLength       = 12        // 12 bytes for AES-GCM
)

var allowedTTLs = map[int]bool{
	3600:   true, // 1 hour
	86400:  true, // 24 hours
	604800: true, // 7 days
}

var allowedExpirationMinutes = map[int]bool{
	60:    true, // 1 hour
	1440:  true, // 24 hours
	10080: true, // 7 days
}

// ValidationError represents a field-specific validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// ValidateCreateSecretRequest validates the incoming request
// Returns a slice of validation errors for better error reporting
func ValidateCreateSecretRequest(req CreateSecretRequest) error {
	var validationErrors []ValidationError

	// Validate TTL
	if req.TTLSeconds == 0 && req.ExpiresInMinutes != 0 {
		if !allowedExpirationMinutes[req.ExpiresInMinutes] {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "expiresInMinutes",
				Message: "Expiration must be one of: 60, 1440, or 10080 minutes",
				Code:    "invalid_expiration",
			})
		}
	} else if !allowedTTLs[req.TTLSeconds] {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "ttlSeconds",
			Message: "TTL must be one of: 3600 (1 hour), 86400 (24 hours), or 604800 (7 days)",
			Code:    "invalid_ttl",
		})
	}

	hasContent := req.Content != ""
	hasCiphertext := req.Ciphertext != ""

	if hasContent {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "content",
			Message: "Plaintext content is not accepted by the public API. Encrypt on the client and send ciphertext.",
			Code:    "plaintext_not_allowed",
		})
	}

	if !hasCiphertext {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "ciphertext",
			Message: "Ciphertext is required and cannot be empty",
			Code:    "empty_ciphertext",
		})
	}

	if req.BurnAfterRead != nil && !*req.BurnAfterRead {
		validationErrors = append(validationErrors, ValidationError{
			Field:   "burnAfterRead",
			Message: "One-time links always burn after read",
			Code:    "burn_after_read_required",
		})
	}

	if hasCiphertext {
		// Validate algorithm
		if req.Algorithm != "AES-GCM" {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "algorithm",
				Message: "Algorithm must be 'AES-GCM'",
				Code:    "invalid_algorithm",
			})
		}

		if len(req.Ciphertext) > MaxCiphertextSize {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "ciphertext",
				Message: fmt.Sprintf("Ciphertext exceeds maximum size of %d bytes", MaxCiphertextSize),
				Code:    "ciphertext_too_large",
			})
		}

		// Validate nonce
		if req.Nonce == "" {
			validationErrors = append(validationErrors, ValidationError{
				Field:   "nonce",
				Message: "Nonce is required and cannot be empty",
				Code:    "empty_nonce",
			})
		} else {
			// Decode and validate nonce length
			nonceBytes, err := decodeBase64Url(req.Nonce)
			if err != nil {
				validationErrors = append(validationErrors, ValidationError{
					Field:   "nonce",
					Message: "Nonce must be a valid base64url-encoded string",
					Code:    "invalid_nonce_format",
				})
			} else if len(nonceBytes) != NonceLength {
				validationErrors = append(validationErrors, ValidationError{
					Field:   "nonce",
					Message: fmt.Sprintf("Nonce must be exactly %d bytes when decoded (16 base64url characters)", NonceLength),
					Code:    "invalid_nonce_length",
				})
			}
		}
	}

	// Return errors if any
	if len(validationErrors) > 0 {
		return &MultiValidationError{Errors: validationErrors}
	}

	return nil
}

// EffectiveTTLSeconds returns the canonical TTL value from either supported request shape.
func EffectiveTTLSeconds(req CreateSecretRequest) int {
	if req.TTLSeconds != 0 {
		return req.TTLSeconds
	}
	return req.ExpiresInMinutes * 60
}

// EffectiveBurnAfterRead is always true because public links are one-time-only.
func EffectiveBurnAfterRead(req CreateSecretRequest) bool {
	_ = req
	return true
}

// EffectiveTheme returns the default Secret Letter theme.
func EffectiveTheme(req CreateSecretRequest) string {
	if req.Theme == "" {
		return "classic-letter"
	}
	return req.Theme
}

// MultiValidationError represents multiple validation errors
type MultiValidationError struct {
	Errors []ValidationError
}

func (e *MultiValidationError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Message
	}
	return fmt.Sprintf("%d validation errors occurred", len(e.Errors))
}

// decodeBase64Url decodes base64url string to bytes
func decodeBase64Url(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	// Use URL encoding (base64url uses - and _ instead of + and /)
	return base64.URLEncoding.DecodeString(s)
}
