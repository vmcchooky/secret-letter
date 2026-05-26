package secret

import "errors"

var (
	ErrSecretConsumed = errors.New("secret already consumed")
	ErrSecretExpired  = errors.New("secret expired")
	ErrSecretNotFound = errors.New("secret not found")
	ErrInvalidToken   = errors.New("invalid secret token")
)
