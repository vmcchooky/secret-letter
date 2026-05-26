# Backend - One-Time Link API

Go backend cho one-time-link application. Production-ready với rate limiting, error handling, và performance optimization.

## Current Status

**Milestone:** 4/7 complete (57%)  
**Status:** Production-ready  
**Next:** Production deployment

## Architecture

Single Go binary với clean service boundaries:

- `cmd/api`: Main application entrypoint
- `internal/config`: Environment-based configuration
- `internal/httpapi`: HTTP routing, handlers, và middleware
- `internal/secret`: Secret lifecycle logic (create, status, consume)
- `internal/ratelimit`: Rate limiting implementation
- `internal/store`: Redis client management

## Features

### Core Functionality
- ✅ Create encrypted secrets với TTL
- ✅ Check secret status (non-destructive)
- ✅ Consume secrets (one-time, atomic)
- ✅ Health check endpoint

### Security
- ✅ Rate limiting per IP (120/hr create, 240/hr consume, 600/hr status, 240/hr reveal session)
- ✅ Security headers (HSTS, CSP, X-Frame-Options, etc.)
- ✅ Input validation với detailed errors
- ✅ CORS protection
- ✅ Request size limits (15KB)
- ✅ Ciphertext-only public create API
- ✅ No sensitive data in logs

### Performance
- ✅ No-store dependency-aware health/readiness checks
- ✅ Redis connection pooling
- ✅ Request metrics tracking
- ✅ Slow request logging (>100ms)
- ✅ P95 < 100ms, 100+ req/s

### Operations
- ✅ Structured JSON logging
- ✅ Request ID tracking
- ✅ Error logging với context
- ✅ Graceful degradation (Redis failures)

## Run Locally

### Prerequisites
- Go 1.21+
- Redis running (port 6379)

### Start Backend
```bash
# From repository root
go run ./backend/cmd/api

# Or from backend directory
cd backend
go run ./cmd/api
```

Backend sẽ lắng nghe tại `http://localhost:8080`

### Configuration

Create `.env` file trong backend directory:

```bash
SERVICE_NAME=one-time-link-api
HOST=0.0.0.0
PORT=8080
ALLOWED_ORIGIN=http://localhost:5173
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE=5
REDIS_MAX_RETRIES=3
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Run With Coverage
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Integration Tests
```bash
# Requires backend running
go test -v ./test
```

### Load Testing
```bash
# From repository root
./scripts/load-test.sh --concurrent 10 --requests 100
```

## API Endpoints

### Health and Readiness Checks
```
GET /healthz
GET /readyz
Response: 200 OK
{
  "service": "one-time-link-api",
  "status": "healthy",
  "timestamp": "2026-04-16T12:00:00Z",
  "version": "1.0.0",
  "dependencies": {
    "redis": "healthy"
  }
}
```

Both endpoints return `503 Service Unavailable` when Redis is unhealthy.

### Create Secret
```
POST /api/secrets
Content-Type: application/json

{
  "ciphertext": "base64url-encoded",
  "nonce": "base64url-encoded",
  "algorithm": "AES-GCM",
  "ttlSeconds": 3600
}

Response: 201 Created
{
  "secretId": "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "token": "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "url": "/s/AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "expiresAt": "2026-04-16T13:00:00Z"
}
```

The public API rejects plaintext `content`; clients must encrypt in the browser and submit ciphertext only.

### Check Status
```
GET /api/secrets/{id}/status

Response: 200 OK
{
  "secretId": "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "status": "active",
  "createdAt": "2026-04-16T12:00:00Z",
  "expiresAt": "2026-04-16T13:00:00Z"
}
```

### Open Secret
```
POST /api/secrets/{id}/open

Legacy alias: POST /api/secrets/{id}/consume

Response: 200 OK
{
  "secretId": "AbCdEfGhIjKlMnOpQrStUvWxYz0123456789abcd",
  "ciphertext": "base64url-encoded",
  "nonce": "base64url-encoded",
  "algorithm": "AES-GCM",
  "consumedAt": "2026-04-16T12:30:00Z"
}
```

## Rate Limiting

All endpoints (except health check) are rate limited per endpoint and client IP. Local mode disables rate limiting by default; production mode enables it.

| Endpoint | Limit | Window |
|----------|-------|--------|
| Create secret | 120 requests | 1 hour |
| Consume secret | 240 requests | 1 hour |
| Status check | 600 requests | 1 hour |
| Reveal session | 240 requests | 1 hour |

Production limits are configurable:

```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_WINDOW_SECONDS=3600
RATE_LIMIT_CREATE_PER_WINDOW=120
RATE_LIMIT_CONSUME_PER_WINDOW=240
RATE_LIMIT_STATUS_PER_WINDOW=600
RATE_LIMIT_REVEAL_SESSION_PER_WINDOW=240
```

Rate limit headers on rate-limited endpoint responses:
- `X-RateLimit-Limit`: Maximum requests allowed
- `X-RateLimit-Remaining`: Requests remaining
- `X-RateLimit-Reset`: Unix timestamp when limit resets

429 response when exceeded:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please try again later.",
  "details": {
    "retry_after": 3600
  },
  "timestamp": "2026-04-16T12:00:00Z",
  "request_id": "uuid"
}
```

## Error Handling

All errors follow consistent format:

```json
{
  "error": "error_code",
  "message": "Human-readable message",
  "details": {
    "validation_errors": [
      {
        "field": "algorithm",
        "message": "Algorithm must be 'AES-GCM'",
        "code": "invalid_algorithm"
      }
    ]
  },
  "timestamp": "2026-04-16T12:00:00Z",
  "request_id": "uuid"
}
```

## Production Build

```bash
# From repository root
./scripts/build-production.sh

# Output: build/one-time-link-api-linux-amd64
```

Build script includes:
- All tests
- Security audit (govulncheck)
- Linux + Windows builds
- Deployment package

## Documentation

- [API Contract](../docs/contracts/public-http-api.md)
- [Crypto Specs](../docs/contracts/crypto-and-api-decisions.md)
- [Production Checklist](../docs/PRODUCTION_CHECKLIST.md)
- [Troubleshooting](../docs/TROUBLESHOOTING.md)

## Why This Architecture

Code is structured với clean domain boundaries để:
- Keep deployment simple (single binary)
- Enable future service splitting without rewrite
- Maintain clear separation of concerns
- Support testing và maintenance

Later, boundaries có thể split thành dedicated services nếu cần scale.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

Copyright (c) 2026 Quorix Việt Nam

## Contact

**Developed by:** Quorix Việt Nam

- **Website:** [quorix.io.vn](https://quorix.io.vn)
- **Email:** contact@quorix.io.vn
- **Facebook:** [facebook.com/quorixvietnam](https://facebook.com/quorixvietnam)
