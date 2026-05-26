package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"secret-letter/backend/internal/config"
	"secret-letter/backend/internal/ratelimit"
	"secret-letter/backend/internal/secret"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	config           config.Config
	secretService    secret.Service
	rateLimiter      *ratelimit.Limiter
	trustedProxyNets []*net.IPNet
}

type contextKey string

const (
	requestIDKey contextKey = "request_id"

	corsAllowedHeaders = "Content-Type, Accept, X-Request-ID, X-Reveal-Session"
	corsAllowedMethods = "GET, POST, OPTIONS"
	corsExposedHeaders = "X-Request-ID, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, Retry-After"
	corsMaxAge         = "86400"
)

func NewServer(cfg config.Config, secretService secret.Service) *Server {
	return &Server{
		config:           cfg,
		secretService:    secretService,
		rateLimiter:      nil, // Will be set if Redis client provided
		trustedProxyNets: parseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs),
	}
}

func NewServerWithRateLimiting(cfg config.Config, secretService secret.Service, redisClient *redis.Client) *Server {
	return &Server{
		config:           cfg,
		secretService:    secretService,
		rateLimiter:      ratelimit.NewLimiter(redisClient),
		trustedProxyNets: parseTrustedProxyCIDRs(cfg.TrustedProxyCIDRs),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleHealth)
	mux.HandleFunc("/api/secrets", s.handleCreateSecret)
	mux.HandleFunc("/api/secrets/", s.handleSecretRoutes)
	mux.HandleFunc("/api/reveal-sessions", s.handleCreateRevealSession)

	return withRequestID(
		withSecurityHeaders(
			withCORS(s.config.AllowedOrigin,
				withJSONHeaders(
					withCaching(
						s.withMetrics(
							s.withRateLimiting(
								withRequestSizeLimit(15*1024, // 15KB limit
									withRequestLogging(mux)))))))))
}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withCORS(allowedOrigin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if allowedOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
		w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
		w.Header().Set("Access-Control-Expose-Headers", corsExposedHeaders)
		w.Header().Set("Access-Control-Max-Age", corsMaxAge)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func withJSONHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (only set if request is HTTPS)
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

		// Permissions Policy (formerly Feature Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

func withRequestSizeLimit(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		requestID := r.Context().Value(requestIDKey)
		if requestID == nil {
			requestID = ""
		}

		// Hash IP and User-Agent for privacy
		ipHash := hashString(r.RemoteAddr)
		uaHash := hashString(r.UserAgent())

		logEntry := map[string]interface{}{
			"timestamp":       time.Now().UTC().Format(time.RFC3339),
			"level":           "info",
			"event":           "http_request",
			"request_id":      requestID,
			"method":          r.Method,
			"path":            sanitizeLogPath(r.URL.Path),
			"status":          rw.statusCode,
			"duration_ms":     duration.Milliseconds(),
			"ip_hash":         ipHash,
			"user_agent_hash": uaHash,
		}

		logJSON, _ := json.Marshal(logEntry)
		log.Println(string(logJSON))
	})
}

func hashString(s string) string {
	if s == "" {
		return ""
	}
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8]) // Use first 8 bytes for brevity
}

func getRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func sanitizeLogPath(path string) string {
	if strings.HasPrefix(path, "/api/secrets/") {
		switch {
		case strings.HasSuffix(path, "/status"):
			return "/api/secrets/:token/status"
		case strings.HasSuffix(path, "/open"):
			return "/api/secrets/:token/open"
		case strings.HasSuffix(path, "/consume"):
			return "/api/secrets/:token/consume"
		default:
			return "/api/secrets/:token"
		}
	}

	if strings.HasPrefix(path, "/s/") {
		return "/s/:token"
	}

	if strings.HasPrefix(path, "/reveal/") {
		return "/reveal/:token"
	}

	return path
}
