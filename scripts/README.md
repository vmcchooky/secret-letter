# Scripts

Thư mục này chứa các script hỗ trợ development, testing, và deployment.

## Development Scripts

### stop-dev.sh / stop-dev.ps1

Script để dừng toàn bộ local development stack: frontend Vite (`5173`), backend API (`8080`), và Redis Docker local.

**Bash (Linux/Mac):**
```bash
./scripts/stop-dev.sh
```

**PowerShell (Windows):**
```powershell
.\scripts\stop-dev.ps1
```

**PowerShell options:**
```powershell
# Chỉ dừng frontend/backend, giữ Redis chạy
.\scripts\stop-dev.ps1 -SkipRedis

# Dừng thêm port custom
.\scripts\stop-dev.ps1 -Ports 5173,8080,4173
```

## Test Scripts

### test-create-secret.sh / test-create-secret.ps1

Script để test endpoint POST /api/secrets với nhiều test cases khác nhau.

**Bash (Linux/Mac):**
```bash
# Chạy với backend mặc định (localhost:8080)
./scripts/test-create-secret.sh

# Chạy với custom API URL
API_BASE_URL=http://localhost:3000 ./scripts/test-create-secret.sh
```

**PowerShell (Windows):**
```powershell
# Chạy với backend mặc định (localhost:8080)
.\scripts\test-create-secret.ps1

# Chạy với custom API URL
$env:API_BASE_URL="http://localhost:3000"
.\scripts\test-create-secret.ps1
```

**Test Cases:**
1. ✅ Valid request với TTL 1 giờ
2. ✅ Valid request với TTL 24 giờ
3. ❌ Invalid algorithm (expect 400)
4. ❌ Invalid TTL (expect 400)
5. ❌ Empty ciphertext (expect 400)
6. ❌ Invalid nonce length (expect 400)
7. ❌ Payload too large (expect 413)

### test-milestone2-comprehensive.ps1

Comprehensive test script cho Milestone 2 features.

**PowerShell:**
```powershell
.\scripts\test-milestone2-comprehensive.ps1
```

### test-milestone3-reveal.sh / test-milestone3-reveal.ps1

Script để test reveal flow (status check và consume).

**Bash (Linux/Mac):**
```bash
./scripts/test-milestone3-reveal.sh
```

**PowerShell (Windows):**
```powershell
.\scripts\test-milestone3-reveal.ps1
```

**Test Cases:**
1. ✅ Create secret
2. ✅ Check status (active)
3. ✅ Consume secret
4. ❌ Second consume (expect 410)
5. ✅ Status after consume (not_found)

### test-rate-limiting.ps1

Script để test rate limiting functionality.

**PowerShell:**
```powershell
.\scripts\test-rate-limiting.ps1
```

**Test Cases:**
1. ✅ Normal requests under limit
2. ❌ Requests exceeding limit (expect 429)
3. ✅ Rate limit headers present
4. ✅ Retry-After header in 429 response

### test-trusted-proxy.sh / test-trusted-proxy.ps1

Script để verify trusted proxy boundary trên production/shared edge bằng cách:
- gửi request bình thường qua public edge
- gửi lại request với `X-Forwarded-For` giả mạo
- đối chiếu `ip_hash` trong API logs theo `X-Request-ID`

**Bash (Linux/Mac):**
```bash
./scripts/test-trusted-proxy.sh
```

**PowerShell (Windows):**
```powershell
.\scripts\test-trusted-proxy.ps1
```

**Useful environment variables:**
- `EDGE_API_URL` - public API URL, default `https://api.secret.quorix.io.vn`
- `LOG_SOURCE_COMMAND` - command lấy API logs, default dùng `docker compose ... logs api`

### test-production-smoke.sh / test-production-smoke.ps1

Production smoke test sau deploy/restart. Script này:
1. check `healthz` + `readyz`
2. create secret
3. restart API
4. wait `readyz` recover
5. reveal secret cũ để verify `SECRET_ENCRYPTION_KEY` vẫn ổn
6. gửi oversized request qua edge và expect `413`

**Bash (Linux/Mac):**
```bash
./scripts/test-production-smoke.sh
```

**PowerShell (Windows):**
```powershell
.\scripts\test-production-smoke.ps1
```

**Useful environment variables:**
- `API_BASE_URL` - local/private API URL, default `http://127.0.0.1:18080`
- `EDGE_API_URL` - public API URL for proxy boundary test
- `RESTART_COMMAND` - command restart API, default dùng `docker compose ... restart api`
- `READY_TIMEOUT_SECONDS` - thời gian chờ `readyz` sau restart

### test-redis-expiration.ps1

Script để test Redis TTL expiration.

**PowerShell:**
```powershell
.\scripts\test-redis-expiration.ps1
```

## Load Testing Scripts

### load-test.sh / load-test.ps1

Script để test performance và load handling.

**Bash (Linux/Mac):**
```bash
# Default: 10 concurrent, 100 requests
./scripts/load-test.sh

# Custom load
./scripts/load-test.sh --concurrent 50 --requests 500
```

**PowerShell (Windows):**
```powershell
# Default: 10 concurrent, 100 requests
.\scripts\load-test.ps1

# Custom load
.\scripts\load-test.ps1 -Concurrent 50 -Requests 500
```

**Metrics:**
- Response times (P50, P95, P99)
- Success rate
- Throughput (req/s)
- Error rate

## Build Scripts

### build-production.sh / build-production.ps1

Script để build production-ready binary với security audit.

**Bash (Linux/Mac):**
```bash
./scripts/build-production.sh
```

**PowerShell (Windows):**
```powershell
.\scripts\build-production.ps1
```

**Steps:**
1. Run all tests
2. Security audit (`gosec` + `govulncheck` when installed)
3. Build for Linux (amd64)
4. Build for Windows (amd64)
5. Package the current `deploy/prod` templates and deployment assets

**Output:**
- `build/secret-letter-api-linux-amd64`
- `build/secret-letter-api-windows-amd64.exe`
- `build/secret-letter-api-{version}.tar.gz`

### deploy-staging.sh

CI helper script for the gated staging deployment path used by `.github/workflows/cd.yml`.

This script uploads the release source/backend/frontend archives to the staging host over SSH, unpacks them into a versioned release directory, then executes a remote deploy command that you provide through GitHub environment variables.

When `cd.yml` is started manually, you can also override `vite_api_base_url` and `vite_public_secret_origin` so the staged frontend artifact points at staging endpoints instead of production ones.

Before your remote command runs, the script exports:
- `RELEASE_VERSION`
- `RELEASE_DIR`
- `SOURCE_DIR`
- `BACKEND_ARTIFACT_DIR`
- `FRONTEND_ARTIFACT_DIR`

**Required environment variables:**
- `STAGING_SSH_HOST`
- `STAGING_SSH_USER`
- `STAGING_SSH_PRIVATE_KEY_FILE`
- `STAGING_DEPLOY_PATH`
- `STAGING_REMOTE_DEPLOY_COMMAND`
- `STAGING_RELEASE_VERSION`
- `STAGING_SOURCE_ARCHIVE`
- `STAGING_BACKEND_ARCHIVE`

**Optional environment variables:**
- `STAGING_FRONTEND_ARCHIVE`
- `STAGING_REMOTE_POST_DEPLOY_COMMAND`
- `STAGING_SSH_PORT`
- `STAGING_KNOWN_HOSTS_FILE`
- `DRY_RUN=1`

**Shared-edge example:**
- `STAGING_REMOTE_DEPLOY_COMMAND=docker compose -f deploy/prod/docker-compose.yml -f deploy/prod/docker-compose.vps-edge.yml up -d --build redis api`
- `STAGING_REMOTE_POST_DEPLOY_COMMAND=./scripts/test-production-smoke.sh`

## Development Workflow

Xem `DEVELOPMENT.md` ở root directory để biết chi tiết về:
- Cách chạy backend
- Cách chạy frontend
- Cách chạy Redis local
- Cách chạy tests

## Future Scripts

Các script có thể được thêm trong tương lai:
- Smoke tests cho production
- Failover drills
- Database backup scripts
- Monitoring setup scripts
