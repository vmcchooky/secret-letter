# Milestone 2 Quick Reference

**Quick links và commands cho Milestone 2**

## 📋 Status

✅ **COMPLETE** - All acceptance criteria met

## 📚 Documentation

| Document | Purpose |
|----------|---------|
| [MILESTONE_2_COMPLETION.md](MILESTONE_2_COMPLETION.md) | Implementation details và test results |
| [MILESTONE_2_REVIEW.md](MILESTONE_2_REVIEW.md) | Code quality assessment và recommendations |
| [MILESTONE_2_CHECKLIST.md](MILESTONE_2_CHECKLIST.md) | Complete checklist của tất cả tasks |
| [MILESTONE_2_FINAL_SUMMARY.md](MILESTONE_2_FINAL_SUMMARY.md) | Executive summary và metrics |

## 🚀 Quick Start

### Start Development Environment

```bash
# 1. Start Redis
docker compose -f deploy/local/docker-compose.yml up -d

# 2. Start Backend
cd backend
go run ./cmd/api

# 3. Start Frontend (new terminal)
cd frontend/web-app
npm run dev
```

### Run Tests

```bash
# Backend tests
cd backend
go test ./...

# Integration tests (requires running backend)
go test -v ./test

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 🔑 Key Endpoints

### Backend API

```
POST /api/secrets
- Creates encrypted secret
- Returns: { secretId, expiresAt }
- Status: 201 (success), 400 (invalid), 413 (too large)

GET /healthz
- Health check
- Returns: { service, status, timestamp, version }
```

### Frontend

```
http://localhost:5173
- Create secret form
- Generates link: /reveal/{secretId}#{key}
```

## 📝 Test Scripts

### Manual API Testing

```powershell
# PowerShell
.\scripts\test-create-secret.ps1
.\scripts\test-milestone2-comprehensive.ps1

# Bash
./scripts/test-create-secret.sh
```

### Redis Verification

```bash
# Connect to Redis
docker exec -it <container-id> redis-cli

# Check secret exists
EXISTS secret:{secretId}

# Check TTL
TTL secret:{secretId}

# View data
GET secret:{secretId}
```

## 🔧 Configuration

### Backend Environment Variables

```bash
APP_SERVICE_NAME=one-time-link-api
APP_HOST=0.0.0.0
APP_PORT=8080
ALLOWED_ORIGIN=http://localhost:5173
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Frontend Environment Variables

```bash
VITE_API_BASE_URL=http://localhost:8080
```

## 📊 Validation Rules

### Algorithm
- ✅ `AES-GCM` only
- ❌ Other algorithms rejected

### TTL (seconds)
- ✅ `3600` (1 hour)
- ✅ `86400` (24 hours)
- ✅ `604800` (7 days)
- ❌ Other values rejected

### Nonce
- ✅ Exactly 12 bytes (base64url encoded)
- ❌ Other lengths rejected

### Ciphertext
- ✅ Not empty
- ✅ Max 15KB
- ❌ Empty or too large rejected

## 🧪 Test Coverage

| Package | Coverage | Tests |
|---------|----------|-------|
| httpapi | 81.5% | 6 cases |
| secret | 44.4% | 14 cases |
| Overall | ~70% | 20+ cases |

## 🎯 Key Features

### Backend
- ✅ POST /api/secrets endpoint
- ✅ Redis storage với TTL
- ✅ Comprehensive validation
- ✅ Error handling
- ✅ Request ID tracking
- ✅ Structured logging

### Frontend
- ✅ AES-GCM 256-bit encryption
- ✅ Create secret form
- ✅ TTL selection
- ✅ Size validation
- ✅ Link generation
- ✅ Copy to clipboard

## 🐛 Troubleshooting

### Backend won't start
```bash
# Check Redis is running
docker ps | grep redis

# Check port 8080 is free
netstat -an | grep 8080

# Check environment variables
cat backend/.env.example
```

### Frontend won't connect
```bash
# Check API URL
echo $VITE_API_BASE_URL

# Check CORS
curl -I http://localhost:8080/healthz

# Check browser console for errors
```

### Tests failing
```bash
# Ensure Redis is running for integration tests
docker compose -f deploy/local/docker-compose.yml up -d

# Run specific test
go test -v ./internal/httpapi -run TestCreateSecretEndpoint

# Check test output
go test ./... -v
```

## 📈 Metrics

- **Files Created:** 16
- **Files Modified:** 8
- **Total LOC:** ~1,800 lines
- **Test Cases:** 20+
- **Quality Score:** 9.0/10
- **Compliance:** 100%

## 🔗 Related Files

### Implementation
- `backend/internal/secret/redis_service.go` - Redis service
- `backend/internal/secret/validation.go` - Validation logic
- `backend/internal/httpapi/handlers.go` - HTTP handlers
- `frontend/web-app/src/lib/crypto.ts` - Crypto helpers
- `frontend/web-app/src/components/CreateSecretForm.tsx` - Form component

### Tests
- `backend/internal/secret/validation_test.go` - Validation tests
- `backend/internal/httpapi/create_secret_test.go` - API tests
- `backend/test/integration_test.go` - Integration tests

### Configuration
- `backend/.env.example` - Backend config
- `frontend/web-app/.env.example` - Frontend config
- `deploy/local/docker-compose.yml` - Redis config

## ⏭️ Next Steps

**Milestone 3: Secret Reveal and Consumption**

Will implement:
- GET /api/secrets/{id}/status
- POST /api/secrets/{id}/consume
- Reveal page component
- Client-side decryption
- Already-used state tracking

## 📞 Support

For issues or questions:
1. Check documentation in `docs/`
2. Review test scripts in `scripts/`
3. Check code comments
4. Review specs in `docs/contracts/`

---

**Last Updated:** 2026-04-15  
**Milestone:** 2 of 7  
**Status:** ✅ COMPLETE
