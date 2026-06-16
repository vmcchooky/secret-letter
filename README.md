# secret-letter

`secret-letter` là ứng dụng chia sẻ secret một lần, được thiết kế ưu tiên cho mục đích chia sẻ các dữ liệu nhạy cảm như password, token,... một cách an toàn.

## Trọng Tâm Hiện Tại

**Milestone 4 đã hoàn thành! ✅**

Repository hiện đang ở giai đoạn:

- ✅ **Milestone 1:** Foundation and Local Development - COMPLETE
- ✅ **Milestone 2:** Client-Side Encryption and Secret Creation - COMPLETE
- ✅ **Milestone 3:** Reveal Gate and Status Checking - COMPLETE
- ✅ **Milestone 4:** Rate Limiting and Production Readiness - COMPLETE
- ⏳ **Milestone 5:** Production Deployment - NEXT

Truoc khi bat dau sua code cho production go-live, xem spec hardening:
- [docs/deployment/production-hardening-upgrade-spec.md](docs/deployment/production-hardening-upgrade-spec.md)

### Milestone 4 Highlights

- ✅ Rate limiting (120/hr create, 240/hr consume, 600/hr status, 240/hr reveal session)
- ✅ Enhanced error handling với structured errors
- ✅ Performance optimization (caching, metrics)
- ✅ Security headers (HSTS, CSP, X-Frame-Options)
- ✅ Production build scripts
- ✅ Deployment checklist và troubleshooting guide
- ✅ Load testing scripts

Xem chi tiết: `docs/MILESTONE_4_COMPLETION.md`

## Cấu Trúc Repository

```text
backend/
  cmd/api/
  internal/
frontend/
  web-app/
deploy/
  local/
  prod/
docs/
scripts/
```

## Phát Triển Local

### 1. Khởi Động Redis

```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

### 2. Chạy Go API

```bash
go run ./backend/cmd/api
```

API sẽ lắng nghe tại `http://localhost:8080` theo mặc định.

### 3. Chạy Frontend

```bash
cd frontend/web-app
npm install
npm run dev
```

Frontend sẽ chạy tại `http://localhost:5173` theo mặc định.

### Dừng Local Dev

```powershell
.\scripts\stop-dev.ps1
```

```bash
./scripts/stop-dev.sh
```

## Tài Liệu Chính

### Product Documentation
- [docs/README.md](docs/README.md) - Tổng quan tài liệu
- [docs/product-spec/secret-letter-requirements.md](docs/product-spec/secret-letter-requirements.md) - Yêu cầu sản phẩm
- [docs/product-spec/secret-letter-architecture.md](docs/product-spec/secret-letter-architecture.md) - Kiến trúc hệ thống
- [docs/product-spec/secret-letter-milestones.md](docs/product-spec/secret-letter-milestones.md) - Lộ trình phát triển

### Development Documentation
- [DEVELOPMENT.md](DEVELOPMENT.md) - Hướng dẫn phát triển chi tiết
- [docs/contracts/public-http-api.md](docs/contracts/public-http-api.md) - API contract
- [docs/contracts/crypto-and-api-decisions.md](docs/contracts/crypto-and-api-decisions.md) - Crypto specifications

### Deployment Documentation
- [docs/deployment/deployment-guide.md](docs/deployment/deployment-guide.md) - Hướng dẫn deployment
- [docs/deployment/production-checklist.md](docs/deployment/production-checklist.md) - Production checklist
- [docs/deployment/production-hardening-upgrade-spec.md](docs/deployment/production-hardening-upgrade-spec.md) - Spec nâng cấp hardening trước go-live

### Milestone Documentation
- [docs/MILESTONE_1_COMPLETION.md](docs/MILESTONE_1_COMPLETION.md) - Milestone 1 completion report
- [docs/MILESTONE_2_COMPLETION.md](docs/MILESTONE_2_COMPLETION.md) - Milestone 2 completion report (includes quality assessment, metrics, and checklist)
- [docs/MILESTONE_3_COMPLETION.md](docs/MILESTONE_3_COMPLETION.md) - Milestone 3 completion report
- [docs/MILESTONE_3_QUICK_REFERENCE.md](docs/MILESTONE_3_QUICK_REFERENCE.md) - Milestone 3 quick reference guide
- [docs/MILESTONE_4_COMPLETION.md](docs/MILESTONE_4_COMPLETION.md) - Milestone 4 completion report
- [docs/MILESTONE_4_QUICK_REFERENCE.md](docs/MILESTONE_4_QUICK_REFERENCE.md) - Milestone 4 quick reference guide
- [docs/PRODUCTION_CHECKLIST.md](docs/PRODUCTION_CHECKLIST.md) - Production deployment checklist
- [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - Troubleshooting guide

## Milestone Progress

### ✅ Milestone 1: Foundation and Local Development (COMPLETE)

**Completed Features:**
- ✅ Monorepo structure với frontend/backend separation
- ✅ React + TypeScript frontend shell với Vite
- ✅ Go backend với internal service boundaries
- ✅ Docker Compose cho local Redis
- ✅ Health endpoint và basic HTTP routing
- ✅ Structured logging và CORS middleware
- ✅ Request ID tracking
- ✅ Request size limiting (15KB)
- ✅ IP và User-Agent hashing trong logs
- ✅ API contract documentation

**Documentation:** `docs/MILESTONE_1_COMPLETION.md`

### ✅ Milestone 2: Client-Side Encryption and Secret Creation (COMPLETE)

**Completed Features:**

**Backend:**
- ✅ POST /api/secrets endpoint
- ✅ Redis service với TTL auto-expiration
- ✅ Comprehensive validation (algorithm, TTL, nonce, ciphertext)
- ✅ Error handling (400, 413, 500, 201)
- ✅ Cryptographically random base64url tokens cho secret IDs
- ✅ JSON serialization
- ✅ 81.5% test coverage cho httpapi

**Frontend:**
- ✅ AES-GCM 256-bit encryption với Web Crypto API
- ✅ 12-byte nonce generation
- ✅ Base64url encoding (RFC 4648)
- ✅ Create secret form với TTL selection (1h, 24h, 7d)
- ✅ Plaintext size validation (10KB limit)
- ✅ Byte counter
- ✅ Secret link generation với key trong URL fragment
- ✅ Copy to clipboard functionality
- ✅ Vietnamese UI

**Testing:**
- ✅ 20+ test cases
- ✅ ~70% overall coverage
- ✅ Integration tests
- ✅ Manual test scripts

**Quality:**
- ✅ Code quality: 9.0/10
- ✅ Security: 10/10
- ✅ Specs compliance: 100%

**Documentation:** 
- `docs/MILESTONE_2_COMPLETION.md` - Full completion report
- `docs/MILESTONE_2_REVIEW.md` - Code review and quality assessment
- `docs/MILESTONE_2_QUICK_REFERENCE.md` - Quick reference guide

### ✅ Milestone 3: Reveal Gate and Status Checking (COMPLETE)

**Completed Features:**
- ✅ GET /api/secrets/{id}/status endpoint
- ✅ POST /api/secrets/{id}/consume endpoint
- ✅ Reveal page với preview bot protection
- ✅ React Router integration
- ✅ Client-side decryption
- ✅ Atomic Redis claim/finalize operations
- ✅ 16 test cases (11 unit + 5 integration)

**Documentation:** 
- `docs/MILESTONE_3_COMPLETION.md` - Full completion report
- `docs/MILESTONE_3_QUICK_REFERENCE.md` - Quick reference guide

### ✅ Milestone 4: Rate Limiting and Production Readiness (COMPLETE)

**Completed Features:**

**Rate Limiting:**
- ✅ Redis-based rate limiter (120/hr create, 240/hr consume, 600/hr status, 240/hr reveal session)
- ✅ Per-IP tracking với X-Forwarded-For support
- ✅ Rate limit headers (X-RateLimit-*)
- ✅ 429 responses với Retry-After
- ✅ Graceful degradation

**Error Handling:**
- ✅ Structured error system với AppError
- ✅ Field-specific validation errors
- ✅ Multiple validation errors returned together
- ✅ Error logging với context (no sensitive data)

**Performance:**
- ✅ No-store dependency-aware health/readiness checks
- ✅ Request metrics và slow request logging
- ✅ Load testing scripts (PowerShell + Bash)
- ✅ P95 < 100ms, 100+ req/s

**Production:**
- ✅ Security headers (HSTS, CSP, X-Frame-Options, etc.)
- ✅ Production build scripts với security audit
- ✅ Production config template
- ✅ Deployment checklist
- ✅ Troubleshooting guide

**Documentation:**
- `docs/MILESTONE_4_COMPLETION.md` - Full completion report
- `docs/MILESTONE_4_QUICK_REFERENCE.md` - Quick reference guide
- `docs/PRODUCTION_CHECKLIST.md` - Deployment checklist
- `docs/TROUBLESHOOTING.md` - Common issues guide

### ⏳ Milestone 5: Production Deployment (NEXT)

**Planned Features:**
- VPS setup và configuration
- HTTPS/TLS configuration
- Frontend deployment (Vercel)
- DNS configuration
- Monitoring setup

## Testing

### Run Backend Tests

```bash
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Integration Tests

```bash
# Requires backend running
cd backend
go test -v ./test
```

### Manual API Testing

```powershell
# PowerShell
.\scripts\test-create-secret.ps1
.\scripts\test-milestone2-comprehensive.ps1
```

```bash
# Bash
./scripts/test-create-secret.sh
```

### Fresh Clone / CI Verification

```bash
go test ./...
gosec ./...
govulncheck ./...

cd frontend/web-app
npm ci
npm test
npm run build
npm audit --omit=dev
npm run test:e2e
```

The frontend intentionally has no local `file:` dependency such as `@quorix/ui`, so `npm ci` works on a clean checkout.

GitHub Actions CI also validates the shared-edge deployment artifacts by rendering the Compose topology, checking shell/PowerShell helper scripts, dry-running the staging deploy helper with synthetic release bundles, validating the Caddyfile, and building the API container image.

For tags, releases, and manual gated staging deploys, `.github/workflows/cd.yml` builds release assets, emits a release manifest plus SHA-256 checksum sidecars, uploads workflow artifacts, attaches assets to published GitHub Releases, and can run a `staging` environment deployment after approval. The staging deploy helper verifies those artifacts locally and remotely, keeps `current`/`previous` staged release pointers for rollback, and the manual trigger still accepts frontend URL overrides so staging builds do not have to reuse production origins.

## Current Features

### Working Features ✅

1. **Create Secret Flow**
   - Enter plaintext (max 10KB)
   - Select TTL (1 hour, 24 hours, 7 days)
   - Client-side AES-GCM encryption
   - Submit ciphertext only; plaintext never leaves the browser
   - Generate shareable link
   - Copy link to clipboard

2. **Reveal Secret Flow**
   - Status check (non-destructive)
   - Reveal gate với preview bot protection
   - Reveal session handshake before open
   - One-time consumption (atomic Redis claim/finalize)
   - Client-side decryption
   - Error handling cho all states

3. **Backend API**
   - POST /api/secrets - Create encrypted secret
   - GET /api/secrets/{id}/status - Check secret status
   - POST /api/reveal-sessions - Create reveal session handshake
   - POST /api/secrets/{id}/consume - Consume secret (one-time)
   - POST /api/secrets/{id}/open - Alias for consume secret
   - GET /healthz - Dependency-aware health check
   - GET /readyz - Readiness check
   - Request validation
   - Error handling
   - Redis storage với TTL
   - Rate limiting per IP

4. **Security**
   - Client-side encryption (plaintext never sent to server)
   - Encryption key in URL fragment (never sent to server)
   - Input validation
   - Size limits (10KB plaintext, 15KB request)
   - CORS protection
   - IP/UA hashing in logs
   - Rate limiting (120/hr create, 240/hr consume, 600/hr status, 240/hr reveal session)
   - Security headers (HSTS, CSP, X-Frame-Options, etc.)
   - Atomic operations (race condition prevention)

5. **Production Readiness**
   - Structured error handling
   - Performance optimization (caching, metrics)
   - Load testing scripts
   - Production build scripts
   - Deployment checklist
   - Troubleshooting guide

### Pending Features ⏳

1. **Production Deployment** (Milestone 5)
   - VPS setup
   - HTTPS/TLS configuration
   - Frontend deployment
   - DNS configuration
   - Monitoring setup

2. **Advanced Features** (Future Milestones)
   - Custom expiration times
   - Email delivery
   - Admin dashboard
   - Analytics

## Technology Stack

### Backend
- **Language:** Go 1.26
- **Framework:** Standard library (net/http)
- **Database:** Redis 7
- **Dependencies:**
  - github.com/google/uuid - request ID generation
  - github.com/redis/go-redis/v9 - Redis client

### Frontend
- **Framework:** React 19
- **Language:** TypeScript 5.8
- **Build Tool:** Vite 7
- **Crypto:** Web Crypto API (native)
- **UI Styling:** Local CSS committed in `frontend/web-app/src`; no external sibling package required

### Infrastructure
- **Development:** Docker Compose
- **Production Target:** VPS + Vercel
- **Reverse Proxy:** Caddy

## Project Structure

```
secret-letter/
├── backend/
│   ├── cmd/api/              # Application entry point
│   ├── internal/
│   │   ├── config/           # Configuration
│   │   ├── httpapi/          # HTTP handlers and middleware
│   │   ├── secret/           # Secret service and types
│   │   └── store/            # Redis client
│   └── test/                 # Integration tests
├── frontend/web-app/
│   └── src/
│       ├── components/       # React components
│       ├── lib/              # Utilities (crypto, api, types)
│       └── styles.css        # Styling
├── deploy/
│   ├── local/                # Docker Compose for local dev
│   └── prod/                 # Production configs
├── docs/
│   ├── contracts/            # API and crypto specs
│   ├── deployment/           # Deployment guides
│   ├── product-spec/         # Product requirements
│   └── MILESTONE_*.md        # Milestone reports
└── scripts/                  # Test and utility scripts
```

## Environment Variables

### Backend (.env)

```bash
APP_SERVICE_NAME=secret-letter-api
APP_ENV=development
APP_HOST=0.0.0.0
APP_PORT=8080
ALLOWED_ORIGIN=http://localhost:5173
TRUSTED_PROXY_CIDRS=127.0.0.1/32,::1/128
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
SECRET_ENCRYPTION_KEY=
RATE_LIMIT_ENABLED=false
RATE_LIMIT_WINDOW_SECONDS=3600
RATE_LIMIT_CREATE_PER_WINDOW=120
RATE_LIMIT_CONSUME_PER_WINDOW=240
RATE_LIMIT_STATUS_PER_WINDOW=600
RATE_LIMIT_REVEAL_SESSION_PER_WINDOW=240
```

`SECRET_ENCRYPTION_KEY` is required when `APP_ENV=production`. It must decode to exactly 32 bytes and is used for server-side AES-GCM encryption of Redis payloads. For production, generate it once and keep it stable across restarts:

```powershell
$key = [byte[]]::new(32); [Security.Cryptography.RandomNumberGenerator]::Fill($key); [Convert]::ToBase64String($key).TrimEnd('=').Replace('+','-').Replace('/','_')
```

`TRUSTED_PROXY_CIDRS` controls when the API trusts `X-Forwarded-For` and `X-Real-IP` for rate limiting. In production behind Docker/Caddy, set it to the reverse-proxy network only, for example `172.16.0.0/12`.

Rate limiting is disabled automatically for `APP_ENV=local`, `dev`, `development`, and `test`. Production should use `APP_ENV=production` and `RATE_LIMIT_ENABLED=true`.

### Frontend (.env)

```bash
VITE_API_BASE_URL=http://localhost:8080
```

## Đóng Góp

Chúng tôi hoan nghênh mọi đóng góp! Xem [CONTRIBUTING.md](CONTRIBUTING.md) để biết chi tiết về:
- Cách báo cáo bugs
- Cách đề xuất features
- Quy trình code contribution
- Code style guidelines
- Testing requirements

Hoặc xem [DEVELOPMENT.md](DEVELOPMENT.md) để biết chi tiết về:
- Development workflow
- Local setup
- Testing procedures

## Project Status

**Current Status:** Production deployment planning and hardening in progress
**Completed Milestones:** 4/7 (57%)  
**Next Milestone:** Production Deployment

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Quorix Việt Nam

## Contact

**Developed by:** Quorix Việt Nam

- **Website:** [quorix.io.vn](https://quorix.io.vn)
- **Email:** contact@quorix.io.vn
- **Facebook:** [facebook.com/quorixvietnam](https://facebook.com/quorixvietnam)

For bug reports and feature requests, please open an issue on GitHub.
