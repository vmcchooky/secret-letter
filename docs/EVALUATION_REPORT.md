# Đánh giá Toàn diện Dự án Secret Letter

Báo cáo này trình bày đánh giá chuyên sâu từ A đến Z đối với dự án Secret Letter, tập trung vào Backend (Go), Redis, DevOps/Docker và Bảo mật, đối chiếu với các tiêu chuẩn quốc tế (NIST, OWASP) cũng như thực tiễn kỹ thuật phần mềm tốt nhất.

## 1. Kiến trúc Backend (Golang)

### 1.1. Điểm mạnh
- **Cấu trúc thư mục:** Sử dụng `internal/` (tránh import ngoài ý muốn) và `cmd/api/` theo chuẩn "Standard Go Project Layout", giúp codebase module hóa tốt.
- **Middleware Chain:** Pattern middleware cho Request ID, CORS, Security Headers, Logging, Rate Limiting, Caching là rất rành mạch.
- **Graceful Error Handling:** Sử dụng `AppError` trả về JSON theo chuẩn, kèm detail cụ thể giúp Frontend dễ dàng bắt lỗi.
- **Dependency Injection:** Inject `config` và `redisClient` vào `Server` và `RedisService` dễ dàng cho Unit Test và mock.

### 1.2. Điểm chưa tối ưu & Khuyến nghị
- **Graceful Shutdown:** `main.go` gọi `srv.ListenAndServe()` trực tiếp mà không có lắng nghe signal (SIGINT/SIGTERM).
  - **Khuyến nghị:** Cần bắt `os.Signal` và gọi `srv.Shutdown(ctx)` để xử lý triệt để các kết nối/request đang dang dở (nhất là các request mở secret).
- **Concurrency & Context Propagation:** `withRequestLogging` tính thời gian duration, nhưng khi hash IP/UserAgent có thể dùng pooling (ví dụ `sync.Pool` cho sha256) nếu load lớn để giảm thiểu heap allocation. Việc ghi log trực tiếp qua `log.Println` trong middleware có thể tạo bottleneck về I/O; nên cân nhắc sử dụng thư viện log bất đồng bộ có cấu trúc (vd: `zerolog` hoặc `zap`).
- **Memory Allocation:** Ở endpoint `HandleCreateSecret`, request size giới hạn là 15KB (ổn định). Tuy nhiên, hàm xử lý `json.Unmarshal` trực tiếp load vào memory, nên dùng `json.NewDecoder(r.Body).Decode()` sẽ tối ưu memory hơn cho high load.

## 2. Bảo mật (Security)

Đánh giá dựa trên OWASP Top 10 và NIST Cryptographic Standards.

### 2.1. Điểm mạnh
- **Mã hóa E2E & Tại Nghỉ (At-Rest):** Frontend mã hóa (AES-GCM 256-bit) trước khi gửi. Tuyệt vời hơn, Backend mã hóa lần 2 "at-rest" (AES-GCM 256) trên Redis. Plaintext không bao giờ tồn tại trên server. Key mã hóa URL (fragment `#`) không gửi qua mạng.
- **Rate Limiting:** IP-based tracking bảo vệ mọi API endpoints. Ngăn chặn brute-force và DDoS (OWASP API4:2023).
- **Hashing Security:** Hashing `token` (bằng SHA-256) trước khi query Redis (tránh leak key trong logs và timing attacks).
- **Security Headers:** HSTS, CSP `default-src 'none'`, `X-Frame-Options DENY`, `X-Content-Type-Options nosniff` (OWASP API8:2023).

### 2.2. Điểm chưa tối ưu & Khuyến nghị
- **AES-GCM Nonce Reuse Risk (Server-side):** Tại `encryptPayload` (`redis_service.go`), nonce (12 bytes) được generate random (`crypto/rand`). Với số lượng lớn secret, nguy cơ đụng độ nonce random (birthday paradox) trên cùng một key có thể xảy ra, mặc dù xác suất cực kỳ nhỏ.
  - **Khuyến nghị:** Cân nhắc việc xoay vòng key mã hóa (Key Rotation) định kỳ trên server hoặc thêm context vào nonce.
- **CORS Configuration:** `AllowedOrigin` đang được map linh hoạt 1 origin thông qua Env (mặc định localhost). Cần chắc chắn môi trường Prod chỉ allowed đúng domain chính chủ. `corsMaxAge` 86400 (24h) là rất tốt, giảm thiểu số lượng preflight request (OPTIONS).
- **Rate Limit IP Spoofing:** Rate limit lấy IP thông qua `X-Forwarded-For` với sự tin cậy từ `TRUSTED_PROXY_CIDRS`. Tuy nhiên, nếu Caddy không cấu hình cẩn thận xóa đi header từ user gửi lên, hacker có thể spoof `X-Forwarded-For`.
  - **Khuyến nghị:** Caddyfile hiện tại *chưa* có directive `trusted_proxies` để quản lý `X-Forwarded-For` chặt chẽ, cần đảm bảo Caddy override các header client giả mạo thay vì append thêm.
- **Base64URL Decode Tùy Chỉnh:** Ở backend, `parseAtRestKey` đang thử `RawURLEncoding`, `URLEncoding`, `StdEncoding` lần lượt. Điều này tạo timing side-channel rò rỉ nhẹ thông tin định dạng và có thể parse sai một key không mong đợi. Nên ép buộc một chuẩn chung trị số tĩnh cho key production (ví dụ Hex).

## 3. Redis & Data Lifecycle

### 3.1. Điểm mạnh
- **Atomic Operations:** Consume secret sử dụng Redis Lua Script. Đảm bảo tính ACID, loại bỏ triệt để *Race Conditions* (khi 2 user đồng thời ấn mở). Đây là điểm thiết kế cực kỳ xuất sắc.
- **TTL Lifecycle:** Secret payload được gán TTL chính xác theo thời gian expire, tự dọn rác mà không cần background cronjob.

### 3.2. Điểm chưa tối ưu & Khuyến nghị
- **Lua Script Caching:** Các biến Lua script hiện tại đang sử dụng `redis.NewScript`, module `go-redis` có tính năng `SCRIPT LOAD` và `EVALSHA`. Hãy chắc chắn rằng hệ thống đang gọi bằng `EVALSHA` thay vì truyền toàn bộ mã nguồn script (gần 40 lines) trong mỗi request `ConsumeSecret` để tiết kiệm băng thông network.
- **Redis Connection Pool:** `MaxRetries` set là 3, nhưng không có chiến lược Circuit Breaker. Nếu Redis chết đột ngột, toàn bộ request sẽ nghẽn chờ Timeout rồi mới báo lỗi. Cần Timeout thấp hơn cho Redis connect (`DialTimeout`, `ReadTimeout`, `WriteTimeout`) thay vì chỉ dùng default config của `go-redis`.

## 4. DevOps, Docker & Hiệu năng (Performance)

### 4.1. Điểm mạnh
- **Multi-stage Dockerfile:** Build image cực nhỏ (chỉ copy file binary và CA certs từ alpine), giảm attack surface. `CGO_ENABLED=0` hỗ trợ static binary hoàn hảo.
- **Load Testing & Metrics:** System có slow request logging (p95 < 100ms) là điểm tốt cho operation.
- **Health Checks:** Dependency-aware. docker-compose có healthcheck tốt với `readyz`.

### 4.2. Điểm chưa tối ưu & Khuyến nghị
- **Docker Image:** `alpine:3.20` được dùng làm runner. Để đạt bảo mật tối đa (hardened), cân nhắc chuyển sang Google `distroless/static-debian12` hoặc `scratch`. Các image này không có shell (loại trừ RCE vulnerability).
- **Caddy Reverse Proxy:**
  - `Caddyfile` hiện sử dụng `encode zstd gzip`. Nên cân nhắc cấu hình giới hạn kích thước request max trên Caddy (`request_body` directive) để chặn sớm các cuộc tấn công payload bự ngay tại proxy layer trước khi đến backend Go.
  - Cần thêm `tls { protocols tls1.3 }` để rào lại các giao thức TLS cũ, giúp nâng hạng chứng chỉ SSL Labs lên điểm A+.
- **User Privilege:** Trong Dockerfile `alpine:3.20`, app chạy quyền `root` (vì không tạo User mới).
  - **Khuyến nghị:** Tạo một non-root user `appuser` và `USER appuser` trước lệnh `CMD` (tuân thủ nguyên tắc Least Privilege).

## 5. Tổng kết

Dự án có nền tảng cực kỳ vững chắc, tư duy bảo mật *Security-by-Design* (Zero-knowledge từ phía client, Lua scripts chống Race Condition) và kiến trúc codebase sạch sẽ. Mức độ sẵn sàng cho Production là khoảng 90%.

**Top 3 hành động khắc phục ngay trước khi go-live:**
1. Áp dụng Graceful Shutdown trong `main.go`.
2. Tạo Non-root User trong Dockerfile và dùng distroless image nếu có thể.
3. Review kỹ cấu hình Caddyfile để chặn IP spoofing qua `X-Forwarded-For` và ép chuẩn TLS 1.3.
