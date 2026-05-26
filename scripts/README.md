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
2. Security audit (govulncheck)
3. Build for Linux (amd64)
4. Build for Windows (amd64)
5. Create deployment package

**Output:**
- `build/secret-letter-api-linux-amd64`
- `build/secret-letter-api-windows-amd64.exe`
- `build/secret-letter-api-{version}.tar.gz`

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
