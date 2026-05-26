# Hướng Dẫn Phát Triển

Tài liệu này cung cấp hướng dẫn nhanh để thiết lập môi trường phát triển local cho dự án `secret-letter`.

## Trạng Thái Dự Án

**Milestone hiện tại:** 4/7 hoàn thành (57%)  
**Trạng thái:** Production-ready, sẵn sàng deploy  
**Milestone tiếp theo:** Production Deployment

## Yêu Cầu Hệ Thống

- **Go**: 1.21 trở lên
- **Node.js**: 18 trở lên
- **Docker**: để chạy Redis local
- **Git**: để quản lý source code

## Thiết Lập Môi Trường Local

### 1. Clone Repository

```bash
git clone https://github.com/vmcchooky/secret-letter.git
cd secret-letter
```

### 2. Khởi Động Redis

```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

Kiểm tra Redis đang chạy:

```bash
docker compose -f deploy/local/docker-compose.yml ps
```

### 3. Chạy Backend API

```bash
# Từ thư mục gốc của repository
go run ./backend/cmd/api
```

Backend sẽ lắng nghe tại `http://localhost:8080`

Kiểm tra health endpoint:

```bash
curl http://localhost:8080/healthz
```

### 4. Chạy Frontend

```bash
cd frontend/web-app
npm install
npm run dev
```

Frontend sẽ chạy tại `http://localhost:5173`

## Cấu Hình Môi Trường

### Backend

Tạo file `.env` trong thư mục `backend/` (tùy chọn):

```bash
cp backend/.env.example backend/.env
```

Các biến môi trường có sẵn:
- `APP_SERVICE_NAME`: Tên service (mặc định: `secret-letter-api`)
- `APP_ENV`: Môi trường chạy (`development` hoặc `production`)
- `APP_HOST`: Host để bind (mặc định: `0.0.0.0`)
- `APP_PORT`: Port để lắng nghe (mặc định: `8080`)
- `ALLOWED_ORIGIN`: CORS origin cho phép (mặc định: `http://localhost:5173`)
- `TRUSTED_PROXY_CIDRS`: Danh sách CIDR proxy được tin khi đọc `X-Forwarded-For`/`X-Real-IP` cho rate limit (mặc định: `127.0.0.1/32,::1/128`)
- `SECRET_ENCRYPTION_KEY`: Khóa AES-GCM 32 bytes cho payload trong Redis. Ở development có thể bỏ trống, nhưng secret cũ sẽ không giải mã được sau khi restart backend. Ở production bắt buộc phải cấu hình.

### Frontend

Tạo file `.env` trong thư mục `frontend/web-app/` (tùy chọn):

```bash
cp frontend/web-app/.env.example frontend/web-app/.env
```

## Chạy Tests

### Backend Tests

```bash
# Chạy tất cả tests
go test ./backend/...

# Chạy tests với coverage
go test -cover ./backend/...

# Chạy tests với output chi tiết
go test -v ./backend/...

# Chạy integration tests
go test -v ./backend/test
```

### Load Testing

```bash
# PowerShell
.\scripts\load-test.ps1 -Concurrent 10 -Requests 100

# Bash
./scripts/load-test.sh --concurrent 10 --requests 100
```

### Rate Limiting Tests

```powershell
# PowerShell
.\scripts\test-rate-limiting.ps1
```

### Frontend Verification

Frontend test runner has not been added yet. The web app must build from a fresh clone without any local sibling packages.

Use build verification instead:

```bash
cd frontend/web-app
npm ci
npm run build
```

The frontend has no `file:` dependency on local packages such as `@quorix/ui`; all required styling is committed in this repository.

### Security Verification

Run these before opening a PR or deploying:

```bash
go test ./...
gosec ./...
govulncheck ./...
cd frontend/web-app
npm ci
npm run build
npm audit --omit=dev
```

Install Go security tools if your machine does not have them yet:

```bash
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

### Manual Reveal Smoke Tests

- Create a secret in the frontend and open the generated link once.
- Refresh or open the same link again; it should show the consumed state.
- Open the same link without the `#fragment`; it should show the missing key error.
- Replace the fragment with malformed text such as `#not-valid!!!`; it should show the invalid key format error and must not call the open API.
- Test hold-to-open on desktop and a mobile viewport.

## Kiểm Tra Nhanh

Sau khi khởi động tất cả services, kiểm tra các endpoint sau:

1. **Backend Health**: `http://localhost:8080/healthz`
2. **Backend Readiness**: `http://localhost:8080/readyz`
3. **Frontend**: `http://localhost:5173`
4. **Redis**: `docker compose -f deploy/local/docker-compose.yml exec redis redis-cli ping`

## Dừng Services

### Dừng toàn bộ local dev

```powershell
.\scripts\stop-dev.ps1
```

```bash
./scripts/stop-dev.sh
```

Script sẽ dừng frontend (`5173`), backend (`8080`) và Redis local.

### Dừng thủ công

Backend và frontend có thể dừng bằng `Ctrl+C` trong terminal đang chạy từng service.

```bash
docker compose -f deploy/local/docker-compose.yml down
```

## Cấu Trúc Thư Mục

```
secret-letter/
├── backend/           # Go backend API
│   ├── cmd/api/      # Main application entry point
│   └── internal/     # Internal packages
├── frontend/         # React frontend
│   └── web-app/     # Main web application
├── deploy/          # Deployment configurations
│   ├── local/       # Local development (Docker Compose)
│   └── prod/        # Production configurations
└── docs/            # Documentation
```

## Troubleshooting

### Backend không khởi động được

- Kiểm tra port 8080 có bị chiếm không: `lsof -i :8080` (macOS/Linux) hoặc `netstat -ano | findstr :8080` (Windows)
- Kiểm tra Go version: `go version`

### Frontend không khởi động được

- Xóa `node_modules` và cài lại: `rm -rf node_modules && npm install`
- Kiểm tra Node version: `node --version`

### Redis không kết nối được

- Kiểm tra Docker đang chạy: `docker ps`
- Kiểm tra logs: `docker compose -f deploy/local/docker-compose.yml logs redis`
- Restart Redis: `docker compose -f deploy/local/docker-compose.yml restart redis`

## Tài Liệu Bổ Sung

- [README.md](README.md) - Tổng quan dự án
- [docs/README.md](docs/README.md) - Tài liệu chi tiết
- [docs/contracts/public-http-api.md](docs/contracts/public-http-api.md) - API contract
- [docs/product-spec/secret-letter-milestones.md](docs/product-spec/secret-letter-milestones.md) - Lộ trình phát triển

## Đóng Góp

Hiện tại dự án đã hoàn thành Milestone 4 và sẵn sàng cho production deployment. Vui lòng tham khảo:
- [docs/product-spec/secret-letter-milestones.md](docs/product-spec/secret-letter-milestones.md) - Lộ trình phát triển
- [docs/MILESTONE_4_COMPLETION.md](docs/MILESTONE_4_COMPLETION.md) - Milestone 4 completion report
- [docs/PRODUCTION_CHECKLIST.md](docs/PRODUCTION_CHECKLIST.md) - Production deployment checklist

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Quorix Việt Nam

## Contact

**Developed by:** Quorix Việt Nam

- **Website:** [quorix.io.vn](https://quorix.io.vn)
- **Email:** contact@quorix.io.vn
- **Facebook:** [facebook.com/quorixvietnam](https://facebook.com/quorixvietnam)
