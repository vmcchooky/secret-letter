package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"secret-letter/backend/internal/ratelimit"
)

// EndpointConfig defines rate limit configuration for specific endpoints
type EndpointConfig struct {
	Limit  int
	Window time.Duration
}

var (
	CreateSecretLimit = EndpointConfig{
		Limit:  120,
		Window: time.Hour,
	}
	ConsumeSecretLimit = EndpointConfig{
		Limit:  240,
		Window: time.Hour,
	}
	StatusCheckLimit = EndpointConfig{
		Limit:  600,
		Window: time.Hour,
	}
	RevealSessionLimit = EndpointConfig{
		Limit:  240,
		Window: time.Hour,
	}
)

// Metrics holds request metrics
type Metrics struct {
	TotalRequests  int64
	TotalErrors    int64
	TotalDuration  time.Duration
	RequestsByPath map[string]int64
	ErrorsByPath   map[string]int64
	DurationByPath map[string]time.Duration
}

// withMetrics adds metrics collection middleware
func (s *Server) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		rw := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// Log performance metrics for slow requests (>100ms)
		if duration > 100*time.Millisecond {
			requestID := getRequestID(r.Context())
			logSlowRequest(r, rw.statusCode, duration, requestID)
		}
	})
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// logSlowRequest logs requests that take longer than expected
func logSlowRequest(r *http.Request, statusCode int, duration time.Duration, requestID string) {
	// Only log if duration is significant
	if duration < 100*time.Millisecond {
		return
	}

	logEntry := map[string]interface{}{
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"level":       "warn",
		"event":       "slow_request",
		"request_id":  requestID,
		"method":      r.Method,
		"path":        r.URL.Path,
		"status":      statusCode,
		"duration_ms": duration.Milliseconds(),
	}

	// Add warning level based on duration
	if duration > 500*time.Millisecond {
		logEntry["level"] = "error"
		logEntry["severity"] = "critical"
	} else if duration > 200*time.Millisecond {
		logEntry["severity"] = "high"
	}

	logJSON, _ := json.Marshal(logEntry)
	log.Println(string(logJSON))
}

// withCaching adds response caching middleware
func withCaching(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			w.Header().Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}

// withRateLimiting adds rate limiting middleware to the handler
func (s *Server) withRateLimiting(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if limiter not configured
		if s.rateLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Skip rate limiting for health and readiness checks
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}

		// Determine endpoint and config
		endpoint, config := s.getEndpointConfig(r)
		if config == nil {
			// No rate limit for this endpoint
			next.ServeHTTP(w, r)
			return
		}

		// Get client IP
		clientIP := s.getClientIP(r)

		// Create rate limit key: endpoint:ip
		key := fmt.Sprintf("%s:%s", endpoint, clientIP)

		// Check rate limit
		result, err := s.rateLimiter.Allow(r.Context(), key, ratelimit.Config{
			Limit:  config.Limit,
			Window: config.Window,
		})

		if err != nil {
			// Log error but don't block request on rate limiter failure
			// This ensures service availability even if Redis has issues
			next.ServeHTTP(w, r)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.ResetAt.Unix()))

		// Check if rate limit exceeded
		if !result.Allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", result.RetryAfter))
			RespondError(w, r, ErrRateLimitExceeded("Too many requests. Please try again later.").
				WithDetail("retry_after", result.RetryAfter))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getEndpointConfig returns the endpoint name and rate limit config for a request
func (s *Server) getEndpointConfig(r *http.Request) (string, *EndpointConfig) {
	path := r.URL.Path
	method := r.Method

	// POST /api/secrets - Create secret
	if path == "/api/secrets" && method == http.MethodPost {
		return "create_secret", s.rateLimitConfig(s.config.CreateLimit, CreateSecretLimit)
	}

	// POST /api/secrets/{id}/consume - Consume secret
	if strings.HasPrefix(path, "/api/secrets/") && strings.HasSuffix(path, "/consume") && method == http.MethodPost {
		return "consume_secret", s.rateLimitConfig(s.config.ConsumeLimit, ConsumeSecretLimit)
	}

	// GET /api/secrets/{id}/status - Status check
	if strings.HasPrefix(path, "/api/secrets/") && strings.HasSuffix(path, "/status") && method == http.MethodGet {
		return "status_check", s.rateLimitConfig(s.config.StatusLimit, StatusCheckLimit)
	}

	// POST /api/reveal-sessions - Create reveal session
	if path == "/api/reveal-sessions" && method == http.MethodPost {
		return "reveal_session", s.rateLimitConfig(s.config.RevealLimit, RevealSessionLimit)
	}

	// No rate limit for other endpoints
	return "", nil
}

func (s *Server) rateLimitConfig(limit int, fallback EndpointConfig) *EndpointConfig {
	window := s.config.RateLimitWindow
	if window <= 0 {
		window = fallback.Window
	}

	if limit <= 0 {
		limit = fallback.Limit
	}

	return &EndpointConfig{
		Limit:  limit,
		Window: window,
	}
}

// getClientIP extracts the client IP address from the request.
func (s *Server) getClientIP(r *http.Request) string {
	return getClientIP(r, s.trustedProxyNets)
}

func getClientIP(r *http.Request, trustedProxyNets []*net.IPNet) string {
	remoteIP, remoteIPString := remoteIPFromRequest(r)

	if isTrustedProxy(remoteIP, trustedProxyNets) {
		if forwardedIP := firstValidForwardedIP(r.Header.Get("X-Forwarded-For")); forwardedIP != "" {
			return forwardedIP
		}

		if realIP := validHeaderIP(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}

	return remoteIPString
}

func parseTrustedProxyCIDRs(raw string) []*net.IPNet {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	var networks []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if ip := net.ParseIP(part); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			networks = append(networks, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
			continue
		}

		_, network, err := net.ParseCIDR(part)
		if err != nil {
			log.Printf("warning: ignoring invalid TRUSTED_PROXY_CIDRS entry %q: %v", part, err)
			continue
		}

		networks = append(networks, network)
	}

	return networks
}

func isTrustedProxy(ip net.IP, trustedProxyNets []*net.IPNet) bool {
	if ip == nil {
		return false
	}

	for _, network := range trustedProxyNets {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func firstValidForwardedIP(header string) string {
	for _, part := range strings.Split(header, ",") {
		if ip := net.ParseIP(strings.TrimSpace(part)); ip != nil {
			return ip.String()
		}
	}

	return ""
}

func validHeaderIP(header string) string {
	if ip := net.ParseIP(strings.TrimSpace(header)); ip != nil {
		return ip.String()
	}

	return ""
}

func remoteIPFromRequest(r *http.Request) (net.IP, string) {
	host := r.RemoteAddr
	if splitHost, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = splitHost
	}

	if ip := net.ParseIP(strings.TrimSpace(host)); ip != nil {
		return ip, ip.String()
	}

	return nil, strings.TrimSpace(host)
}
