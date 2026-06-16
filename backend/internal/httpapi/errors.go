package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
}

// ValidationError represents a field-specific validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ErrorCode represents standard error codes
type ErrorCode string

const (
	ErrCodeInvalidRequest    ErrorCode = "invalid_request"
	ErrCodeValidationFailed  ErrorCode = "validation_failed"
	ErrCodeNotFound          ErrorCode = "not_found"
	ErrCodeAlreadyConsumed   ErrorCode = "already_consumed"
	ErrCodeSecretConsumed    ErrorCode = "SECRET_CONSUMED"
	ErrCodeSecretExpired     ErrorCode = "SECRET_EXPIRED"
	ErrCodeSecretNotFound    ErrorCode = "SECRET_NOT_FOUND"
	ErrCodeRateLimitExceeded ErrorCode = "rate_limit_exceeded"
	ErrCodePayloadTooLarge   ErrorCode = "payload_too_large"
	ErrCodeMethodNotAllowed  ErrorCode = "method_not_allowed"
	ErrCodeInternalError     ErrorCode = "internal_error"
	ErrCodeNotImplemented    ErrorCode = "not_implemented"
	ErrCodeInvalidSecretID   ErrorCode = "invalid_secret_id"
)

// AppError represents an application error with context
type AppError struct {
	Code       ErrorCode
	Message    string
	StatusCode int
	Details    map[string]interface{}
	Err        error // underlying error (not exposed to client)
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Details:    make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	e.Details[key] = value
	return e
}

// WithError adds an underlying error
func (e *AppError) WithError(err error) *AppError {
	e.Err = err
	return e
}

// Common error constructors

// ErrInvalidRequest creates an invalid request error
func ErrInvalidRequest(message string) *AppError {
	return NewAppError(ErrCodeInvalidRequest, message, http.StatusBadRequest)
}

// ErrValidationFailed creates a validation failed error
func ErrValidationFailed(message string) *AppError {
	return NewAppError(ErrCodeValidationFailed, message, http.StatusBadRequest)
}

// ErrNotFound creates a not found error
func ErrNotFound(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message, http.StatusNotFound)
}

// ErrAlreadyConsumed creates an already consumed error
func ErrAlreadyConsumed(message string) *AppError {
	return NewAppError(ErrCodeAlreadyConsumed, message, http.StatusGone)
}

// ErrSecretConsumed creates an SRS-style consumed secret error.
func ErrSecretConsumed(message string) *AppError {
	return NewAppError(ErrCodeSecretConsumed, message, http.StatusGone)
}

// ErrSecretExpired creates an SRS-style expired secret error.
func ErrSecretExpired(message string) *AppError {
	return NewAppError(ErrCodeSecretExpired, message, http.StatusGone)
}

// ErrSecretNotFound creates an SRS-style not-found secret error.
func ErrSecretNotFound(message string) *AppError {
	return NewAppError(ErrCodeSecretNotFound, message, http.StatusNotFound)
}

// ErrRateLimitExceeded creates a rate limit exceeded error
func ErrRateLimitExceeded(message string) *AppError {
	return NewAppError(ErrCodeRateLimitExceeded, message, http.StatusTooManyRequests)
}

// ErrPayloadTooLarge creates a payload too large error
func ErrPayloadTooLarge(message string) *AppError {
	return NewAppError(ErrCodePayloadTooLarge, message, http.StatusRequestEntityTooLarge)
}

// ErrMethodNotAllowed creates a method not allowed error
func ErrMethodNotAllowed(message string) *AppError {
	return NewAppError(ErrCodeMethodNotAllowed, message, http.StatusMethodNotAllowed)
}

// ErrInternal creates an internal error
func ErrInternal(message string) *AppError {
	return NewAppError(ErrCodeInternalError, message, http.StatusInternalServerError)
}

// ErrNotImplemented creates a not implemented error
func ErrNotImplemented(message string) *AppError {
	return NewAppError(ErrCodeNotImplemented, message, http.StatusNotImplemented)
}

// ErrInvalidSecretID creates an invalid secret ID error
func ErrInvalidSecretID(message string) *AppError {
	return NewAppError(ErrCodeInvalidSecretID, message, http.StatusBadRequest)
}

// RespondError writes an error response to the client
func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := getRequestID(r.Context())

	// Check if it's an AppError
	if appErr, ok := err.(*AppError); ok {
		// Log error with context (without sensitive data)
		logError(r, appErr, requestID)

		// Write error response
		w.WriteHeader(appErr.StatusCode)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Error:     string(appErr.Code),
			Message:   appErr.Message,
			Details:   appErr.Details,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: requestID,
		})
		return
	}

	// Fallback for unknown errors
	logUnknownError(r, err, requestID)
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error:     string(ErrCodeInternalError),
		Message:   "An unexpected error occurred",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
	})
}

// logError logs an application error with context
func logError(r *http.Request, err *AppError, requestID string) {
	// Only log errors that are not client errors (4xx)
	if err.StatusCode >= 500 {
		args := []any{
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", sanitizeLogPath(r.URL.Path)),
			slog.String("error_code", string(err.Code)),
			slog.Int("status", err.StatusCode),
		}

		if err.Err != nil {
			args = append(args, slog.String("underlying_error", err.Err.Error()))
		}

		slog.ErrorContext(r.Context(), err.Message, args...)
	} else {
		// For client errors (4xx), log at info level with less detail
		slog.InfoContext(r.Context(), "client_error",
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", sanitizeLogPath(r.URL.Path)),
			slog.String("error_code", string(err.Code)),
			slog.Int("status", err.StatusCode),
		)
	}
}

// logUnknownError logs an unexpected error
func logUnknownError(r *http.Request, err error, requestID string) {
	slog.ErrorContext(r.Context(), "unknown_error",
		slog.String("request_id", requestID),
		slog.String("method", r.Method),
		slog.String("path", sanitizeLogPath(r.URL.Path)),
		slog.String("error", err.Error()),
		slog.Int("status", http.StatusInternalServerError),
	)
}

// AddValidationError adds a validation error to the error details
func AddValidationError(err *AppError, field, message, code string) *AppError {
	if err.Details == nil {
		err.Details = make(map[string]interface{})
	}

	// Get existing validation errors or create new slice
	var validationErrors []ValidationError
	if existing, ok := err.Details["validation_errors"].([]ValidationError); ok {
		validationErrors = existing
	}

	// Add new validation error
	validationErrors = append(validationErrors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})

	err.Details["validation_errors"] = validationErrors
	return err
}
