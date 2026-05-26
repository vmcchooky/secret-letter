package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"secret-letter/backend/internal/secret"
	"strings"
	"time"
)

type healthResponse struct {
	Service      string            `json:"service"`
	Status       string            `json:"status"`
	Timestamp    string            `json:"timestamp"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

type createRevealSessionRequest struct {
	SecretID string `json:"secretId"`
}

type revealSessionCreator interface {
	CreateRevealSession(ctx context.Context, secretID string) (*secret.RevealSessionResponse, error)
}

type revealSessionValidator interface {
	ValidateRevealSession(ctx context.Context, secretID, sessionID string) error
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, r, ErrMethodNotAllowed("Method not allowed"))
		return
	}

	dependency := s.secretService.Health(r.Context())
	status := "healthy"
	statusCode := http.StatusOK
	if !isHealthyDependency(dependency) {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	writeJSON(w, statusCode, healthResponse{
		Service:      s.config.ServiceName,
		Status:       status,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Version:      "1.0.0",
		Dependencies: dependencyMap(dependency),
	})
}

func isHealthyDependency(dependency secret.HealthStatus) bool {
	switch strings.ToLower(strings.TrimSpace(dependency.Mode)) {
	case "healthy", "test", "scaffold":
		return true
	default:
		return false
	}
}

func dependencyMap(dependency secret.HealthStatus) map[string]string {
	store := strings.TrimSpace(dependency.Store)
	if store == "" {
		store = "store"
	}

	mode := strings.TrimSpace(dependency.Mode)
	if mode == "" {
		mode = "unknown"
	}

	return map[string]string{store: mode}
}

func (s *Server) handleCreateSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, r, ErrMethodNotAllowed("Method not allowed"))
		return
	}

	// Read body to trigger MaxBytesReader if size exceeded
	body, err := io.ReadAll(r.Body)
	if err != nil {
		// MaxBytesReader will return error if limit exceeded
		RespondError(w, r, ErrPayloadTooLarge("Request body exceeds 15KB limit").
			WithDetail("max_size_bytes", 15*1024))
		return
	}

	// Parse JSON
	var req secret.CreateSecretRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		RespondError(w, r, ErrInvalidRequest("Invalid JSON request body").
			WithError(err))
		return
	}

	// Validate request
	if err := secret.ValidateCreateSecretRequest(req); err != nil {
		// Check if it's a multi-validation error
		if multiErr, ok := err.(*secret.MultiValidationError); ok {
			appErr := ErrValidationFailed("Request validation failed")

			// Add each validation error to details
			for _, valErr := range multiErr.Errors {
				AddValidationError(appErr, valErr.Field, valErr.Message, valErr.Code)
			}

			RespondError(w, r, appErr)
			return
		}

		// Fallback for other validation errors
		RespondError(w, r, ErrInvalidRequest(err.Error()).WithError(err))
		return
	}

	// Create secret using service
	resp, err := s.secretService.CreateSecret(r.Context(), req)
	if err != nil {
		RespondError(w, r, ErrInternal("Failed to create secret").WithError(err))
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleCreateRevealSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, r, ErrMethodNotAllowed("Method not allowed"))
		return
	}

	setNoStoreHeaders(w)

	var req createRevealSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, ErrInvalidRequest("Invalid JSON request body").WithError(err))
		return
	}

	if strings.TrimSpace(req.SecretID) == "" {
		RespondError(w, r, ErrInvalidSecretID("Secret ID is required"))
		return
	}

	creator, ok := s.secretService.(revealSessionCreator)
	if !ok {
		RespondError(w, r, ErrNotImplemented("Reveal sessions require a Redis-backed secret service"))
		return
	}

	response, err := creator.CreateRevealSession(r.Context(), req.SecretID)
	if err != nil {
		switch {
		case errors.Is(err, secret.ErrInvalidToken):
			RespondError(w, r, ErrInvalidSecretID("Secret token is invalid"))
			return
		case errors.Is(err, secret.ErrSecretConsumed):
			RespondError(w, r, ErrSecretConsumed("This secret has already vanished."))
			return
		case errors.Is(err, secret.ErrSecretExpired):
			RespondError(w, r, ErrSecretExpired("This secret has expired."))
			return
		case errors.Is(err, secret.ErrSecretNotFound):
			RespondError(w, r, ErrSecretNotFound("No secret letter was found here."))
			return
		}

		RespondError(w, r, ErrInternal("Failed to create reveal session").WithError(err))
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleSecretRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/secrets/")

	switch {
	case strings.HasSuffix(path, "/status") && r.Method == http.MethodGet:
		s.handleGetSecretStatus(w, r, path)
	case strings.HasSuffix(path, "/open") && r.Method == http.MethodPost:
		s.handleOpenSecret(w, r, path, "/open")
	case strings.HasSuffix(path, "/consume") && r.Method == http.MethodPost:
		s.handleOpenSecret(w, r, path, "/consume")
	default:
		RespondError(w, r, ErrNotFound("Route not found"))
	}
}

func (s *Server) handleGetSecretStatus(w http.ResponseWriter, r *http.Request, path string) {
	setNoStoreHeaders(w)

	// Extract secret ID from path (remove "/status" suffix)
	secretID := strings.TrimSuffix(path, "/status")
	if secretID == "" {
		RespondError(w, r, ErrInvalidSecretID("Secret ID is required"))
		return
	}

	// Get secret status from service
	status, err := s.secretService.GetSecretStatus(r.Context(), secretID)
	if err != nil {
		if errors.Is(err, secret.ErrInvalidToken) {
			RespondError(w, r, ErrInvalidSecretID("Secret token is invalid"))
			return
		}
		RespondError(w, r, ErrInternal("Failed to get secret status").WithError(err))
		return
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleOpenSecret(w http.ResponseWriter, r *http.Request, path string, suffix string) {
	setNoStoreHeaders(w)

	token := strings.TrimSuffix(path, suffix)
	if token == "" {
		RespondError(w, r, ErrInvalidSecretID("Secret ID is required"))
		return
	}

	if sessionID := strings.TrimSpace(r.Header.Get("X-Reveal-Session")); sessionID != "" {
		if validator, ok := s.secretService.(revealSessionValidator); ok {
			if err := validator.ValidateRevealSession(r.Context(), token, sessionID); err != nil {
				switch {
				case errors.Is(err, secret.ErrInvalidToken):
					RespondError(w, r, ErrInvalidRequest("Reveal session is invalid."))
					return
				case errors.Is(err, secret.ErrSecretExpired):
					RespondError(w, r, ErrSecretExpired("Reveal session has expired."))
					return
				case errors.Is(err, secret.ErrSecretNotFound):
					RespondError(w, r, ErrSecretNotFound("Reveal session not found."))
					return
				}

				RespondError(w, r, ErrInternal("Failed to validate reveal session").WithError(err))
				return
			}
		}
	}

	response, err := s.secretService.ConsumeSecret(r.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, secret.ErrInvalidToken):
			RespondError(w, r, ErrInvalidSecretID("Secret token is invalid"))
			return
		case errors.Is(err, secret.ErrSecretConsumed):
			RespondError(w, r, ErrSecretConsumed("This secret has already vanished."))
			return
		case errors.Is(err, secret.ErrSecretExpired):
			RespondError(w, r, ErrSecretExpired("This secret has expired."))
			return
		case errors.Is(err, secret.ErrSecretNotFound):
			RespondError(w, r, ErrSecretNotFound("No secret letter was found here."))
			return
		case strings.Contains(err.Error(), "not found or already consumed"):
			RespondError(w, r, ErrSecretConsumed("This secret has already vanished."))
			return
		}

		RespondError(w, r, ErrInternal("Failed to consume secret").WithError(err))
		return
	}

	// Return consumed secret data
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func setNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
