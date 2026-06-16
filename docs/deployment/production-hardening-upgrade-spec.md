# Production Hardening Upgrade Spec

## 1. Muc tieu cua tai lieu

Tai lieu nay mo ta pham vi nang cap van hanh va hardening can duoc chot truoc khi bat dau sua code cho production go-live.

No ton tai vi codebase hien tai da co nen tang tot:

- backend, frontend, va e2e flow da co test va build xanh
- secret lifecycle va one-time reveal flow da duoc bao ve kha tot
- tai lieu deployment da phong phu

Nhung van con khoang cach giua "production-ready foundation" va "co the go-live cong khai voi boundary van hanh ro rang".

## 2. Thuc trang hien tai

### 2.1 Nhung gi da tot

- Redis claim/finalize scripts da bao ve one-time consume rat chac.
- `SECRET_ENCRYPTION_KEY` da la yeu cau bat buoc khi `APP_ENV=production`.
- Rate limiting, request size limit, structured error handling, va reveal-session flow da ton tai trong code.
- `go test ./...`, `npm test`, va `npm run build` deu qua tren codebase hien tai.

### 2.2 Khoang cach con lai truoc go-live

- API chua co graceful shutdown cho `SIGINT`/`SIGTERM`.
- Redis client moi co `MaxRetries`, chua co `DialTimeout`, `ReadTimeout`, `WriteTimeout`, `PoolTimeout` ro rang.
- Production config van co the roi vao fallback localhost cho `ALLOWED_ORIGIN` va `TRUSTED_PROXY_CIDRS`.
- Docker image API van chay voi root user.
- Caddy config mau chua the hien ro request body limit va trusted-proxy verification path.
- Tai lieu dang overstate o mot so cho la "production-ready" trong khi hardening sprint chua xong.

## 3. Muc tieu nang cap

- Lam ro contract production config va startup behavior.
- Hardening runtime lifecycle de restart/deploy an toan hon.
- Khoa ro trust boundary giua reverse proxy, API, va Redis.
- Giam rui ro van hanh ma van giu duoc kha nang debug hop ly cho MVP.
- Dong bo lai docs, checklist, va smoke tests voi hien trang codebase.

## 4. Ngoai pham vi dot nay

- Tach microservices hoac doi kien truc lon.
- Bat buoc dung distroless ngay lap tuc.
- Ep TLS 1.3 only.
- Key rotation AES-GCM, manual EVALSHA tuning, hoac logging library migration.
- Redis TLS cho moi mo hinh deployment. MVP hien uu tien Redis private trong localhost hoac private compose network.

## 5. Workstreams de thuc hien sau khi chot spec

### 5.1 Server lifecycle va Redis resilience

- Them signal handling trong `backend/cmd/api/main.go`.
- Goi `srv.Shutdown(ctx)` voi timeout hop ly khi nhan `SIGINT`/`SIGTERM`.
- Dong Redis client trong shutdown path.
- Cau hinh Redis client voi `DialTimeout`, `ReadTimeout`, `WriteTimeout`, va `PoolTimeout`.
- Xac minh `readyz` fail fast khi Redis unavailable thay vi treo lau.

### 5.2 Production config contract

- Production phai khai bao explicit:
  - `APP_SERVICE_NAME`
  - `APP_ENV=production`
  - `APP_HOST`
  - `APP_PORT`
  - `ALLOWED_ORIGIN`
  - `TRUSTED_PROXY_CIDRS`
  - `SECRET_ENCRYPTION_KEY`
  - `RATE_LIMIT_ENABLED=true`
- Khong duoc rely vao fallback localhost trong production.
- Tai lieu phai dung dung ten env var thuc te, khong dung alias cu nhu `SERVICE_NAME`, `HOST`, hoac `PORT`.
- Can quyet dinh ro startup behavior: production co nen hard-fail neu `ALLOWED_ORIGIN` hoac `TRUSTED_PROXY_CIDRS` dang de fallback hay khong.

### 5.3 Reverse proxy va request boundary

- Tai lieu hoa ro hai deployment mode duoc ho tro:
  - single-binary/systemd + host-level Caddy
  - Docker Compose + private network + optional bundled Caddy
- Them yeu cau request body limit tai proxy layer, khong chi tai Go API.
- Xac minh reverse proxy la noi dat/sanitize `X-Forwarded-For` truoc khi API trust header.
- Giu baseline TLS o muc `TLS 1.2+` va strong ciphers, khong ep `TLS 1.3 only` cho MVP.

### 5.4 Container va runtime hardening

- API container phai chay non-root.
- API runtime root filesystem nen read-only, kem tmpfs toi thieu cho `/tmp`.
- API runtime nen drop toan bo Linux capabilities va giu `no-new-privileges`.
- Healthcheck va deploy path phai van debug duoc tren MVP.
- Distroless la tuy chon P1/P2, khong phai gate cho dot hardening dau tien.
- Sau P0, co the siet them cac runtime defaults tuong tu cho cac service tuy chon khac neu can.

### 5.5 Verification va tai lieu

- Tao bo smoke test production nho gon nhung bat buoc:
  - create secret thanh cong
  - open secret lan dau thanh cong
  - open lai tra ve consumed
  - mo link khong co fragment khong duoc consume
  - restart service/container va van mo duoc secret tao truoc do
  - rate-limit headers xuat hien dung
  - request qua lon bi chan dung boundary
- Cap nhat checklist, guide, va README de phan biet:
  - implemented baseline
  - hardening pending
  - deployment mode dang duoc khuyen nghi

## 6. Quyet dinh da chot

### 6.1 Deployment reference path

**Quyet dinh:** lay `shared host-level Caddy edge` lam deployment path tham chieu chinh cho dot hardening va go-live dau tien.

**Cu the:**

- Host-level Caddy phuc vu `secret.quorix.io.vn` va reverse proxy `api.secret.quorix.io.vn`.
- `secret-letter` van dung Docker Compose cho `api` va `redis`.
- Compose `caddy` service bi vo hieu hoa tren shared VPS edge.
- API publish noi bo qua `127.0.0.1:18080`.

**Ly do:**

- Khep nhat voi tai lieu va artifact da co san trong repo (`shared-vps-edge.md`, `docker-compose.vps-edge.yml`).
- Giu boundary ro rang: public TLS/proxy o host edge, private app/data o Compose network.
- Van giu duoc loi ich cua container hardening ma khong phai dua them 1 public Caddy nua vao app stack.

**He qua:**

- `systemd + single binary` tro thanh secondary path de tham khao, khong phai duong chinh cua dot hardening nay.
- `Docker Compose + bundled Caddy` chi dung cho local hoac dedicated edge, khong phai reference production path.

### 6.2 Redis deployment model

**Quyet dinh:** MVP tiep tuc gioi han Redis o mode `private/local only`.

**Cu the:**

- Uu tien Redis trong localhost hoac private Compose network.
- Khong them support remote managed Redis voi TLS trong dot nay.
- Neu muon ho tro managed Redis sau nay, xem nhu mot mini-project rieng voi config, testing, va security review rieng.

**Ly do:**

- Giam scope hardening va tranh mo rong bai toan qua som.
- Khep voi deployment path shared VPS edge.
- Khong buoc codebase phai them TLS/options matrix cho Redis khi chua can.

### 6.3 Production startup safety policy

**Quyet dinh:** production startup phai `hard-fail` neu phat hien config an toan co ban khong dat.

**Can hard-fail khi `APP_ENV=production` va mot trong cac dieu sau xay ra:**

- `SECRET_ENCRYPTION_KEY` khong hop le
- `ALLOWED_ORIGIN` rong, fallback localhost, wildcard, hoac khong phai HTTPS production origin mong muon
- `TRUSTED_PROXY_CIDRS` rong hoac dang de loopback default khong khop topology production
- `RATE_LIMIT_ENABLED=false`

**Canh gioi cua quyet dinh nay:**

- Khong hard-fail vi `REDIS_ADDR=127.0.0.1:6379` neu deployment mode thuc su dung Redis local/private.
- Khong hard-fail vi chua dung distroless hay chua bat them runtime flags nang cao.

### 6.4 Trusted proxy verification mode

**Quyet dinh bo sung:** trusted proxy verification se duoc dong goi thanh:

- 1 bo smoke test production trong checklist
- 1 script/probe nho de test spoofed forwarded headers

Khong de o muc "kiem tra bang tay neu nho".

## 7. Implementation tasks cu the

### Phase 0: Doc va config baseline

1. Chot va document `shared-vps-edge` la reference path chinh trong docs deployment.
   - Files: `docs/deployment/README.md`, `docs/deployment/shared-vps-edge.md`, `docs/deployment/deployment-guide.md`
   - Acceptance: nguoi doc biet ngay duong nao la duong production tham chieu.

2. Dong bo env var contract cho production.
   - Files: `backend/internal/config/config.go`, `README.md`, deployment docs
   - Acceptance: khong con tai lieu hoac mau env nao dung nham `SERVICE_NAME`, `HOST`, `PORT`.

### Phase 1: Backend runtime safety

3. Them production config validation fail-fast.
   - Files: `backend/internal/config/config.go`, co the them test moi trong `backend/internal/config/`
   - Task:
     - them ham validate production config
     - reject wildcard/localhost `ALLOWED_ORIGIN`
     - reject empty/default `TRUSTED_PROXY_CIDRS` cho production
     - reject `RATE_LIMIT_ENABLED=false` trong production
   - Acceptance: app thoat som voi error ro rang truoc khi bind port.

4. Implement graceful shutdown cho API.
   - Files: `backend/cmd/api/main.go`
   - Task:
     - bat `SIGINT`/`SIGTERM`
     - chay `srv.ListenAndServe()` trong goroutine
     - `srv.Shutdown(ctx)` voi timeout
     - `redisClient.Close()`
   - Acceptance: restart/deploy khong cat request dang xu ly mot cach tho bao.

5. Cau hinh Redis client timeouts ro rang.
   - Files: `backend/cmd/api/main.go`, co the mo rong `backend/internal/config/config.go`
   - Task:
     - them `DialTimeout`, `ReadTimeout`, `WriteTimeout`, `PoolTimeout`
     - quyet dinh dung constant sane defaults hay env vars explicit
   - Acceptance: readiness va request path fail fast hon khi Redis gap su co.

6. Dong bo nguon IP cho logging va rate limiting.
   - Files: `backend/internal/httpapi/server.go`, `backend/internal/httpapi/middleware.go`, tests lien quan
   - Task:
     - logging su dung cung logic trusted-proxy voi rate limiter
   - Acceptance: log va limiter khong nhin hai "client IP" khac nhau sau reverse proxy.

### Phase 2: Container va proxy hardening

7. Chay API container voi non-root user.
   - Files: `Dockerfile`, co the can update `deploy/prod/docker-compose.yml`
   - Task:
     - tao non-root runtime user
     - dam bao binary va working dir doc/chay duoc
   - Acceptance: `docker inspect`/runtime behavior cho thay app khong chay root.

8. Hardening request boundary tai proxy.
   - Files: `deploy/prod/Caddyfile`, tai lieu `docs/deployment/shared-vps-edge.md`
   - Task:
     - them request body limit tai proxy
     - ghi ro trusted proxy/sanitized forwarded header requirement
     - giu baseline TLS 1.2+
   - Acceptance: oversized request bi chan tu proxy layer; docs mo ta ro trust boundary.

9. Chinh compose cho reference path shared edge.
   - Files: `deploy/prod/docker-compose.yml`, `deploy/prod/docker-compose.vps-edge.yml`
   - Task:
     - dam bao chi bind API vao `127.0.0.1:18080`
     - nhac ro khong start bundled Caddy tren shared edge
     - kiem tra env template phu hop
   - Acceptance: topology production tham chieu co the bring-up bang 1 huong dan duy nhat.

### Phase 3: Verification va smoke tests

10. Them trusted proxy verification script.
   - Files: `scripts/` (de xuat `test-trusted-proxy.ps1` va/hoac `.sh`)
   - Task:
     - test request binh thuong qua edge
     - test spoofed `X-Forwarded-For` tu untrusted source
   - Acceptance: co cach kiem tra lap lai duoc thay vi test tay mo ho.

11. Them smoke test production sau restart.
   - Files: `docs/PRODUCTION_CHECKLIST.md`, `docs/deployment/production-checklist.md`, co the them script sau
   - Task:
     - tao secret
     - restart API
     - reveal secret cu
   - Acceptance: xac nhan `SECRET_ENCRYPTION_KEY` on dinh qua restart.

12. Bo sung test oversized request va readiness behavior.
   - Files: backend tests hoac deploy scripts
   - Acceptance: boundary `413`/proxy reject va `readyz` behavior duoc verify co he thong.

### Phase 4: Cleanup va secondary paths

13. Ha muc do khuyen nghi cua `systemd` guide xuong "secondary/reference only".
   - Files: `docs/deployment/deployment-guide.md`
   - Acceptance: khong ai doc guide chinh roi tuong nham day la production path duoc uu tien.

14. Xem xet backlog sau hardening P0.
   - Items:
     - distroless hoac image minimization sau khi debug path on dinh
     - mo rong runtime hardening tuong tu cho bundled edge services neu can
     - managed Redis/TLS support
     - stronger deployment automation

## 8. Thu tu thuc hien de xuat

1. Phase 1 task 3, 4, 5
2. Phase 2 task 7, 8, 9
3. Phase 1 task 6
4. Phase 3 task 10, 11, 12
5. Phase 4 task 13, 14

Ly do:

- can fail-fast config va graceful shutdown truoc
- sau do moi hardening container/proxy theo reference topology
- roi moi dong goi verification thanh cac buoc lap lai duoc

## 9. Phat hien them trong qua trinh viet spec

### 7.1 Mau thuan tai lieu

- `docs/PRODUCTION_CHECKLIST.md` dang dung ten env var khong khop code (`SERVICE_NAME`, `HOST`, `PORT`).
- Checklist English dang de firewall mo `6379`, mau thuan voi yeu cau Redis private.
- Mot so tai lieu dang gia dinh Redis production co TLS, trong khi code hien tai chua co config Redis TLS.
- Chinh sach backup Redis data dang khong dong nhat giua cac tai lieu.
- `docs/README.md` bi lap section Milestone 4.

### 7.2 Van de wording/trang thai

- Cum "Production-ready, awaiting deployment" dang qua manh so voi hien trang hardening.
- Can chuyen sang wording thuc te hon: codebase da co production foundation, nhung production hardening sprint van dang pending.

## 10. Dieu kien hoan tat dot hardening

- Co spec nay duoc chot va cac cau hoi mo da co huong tra loi.
- Checklist deployment chinh va checklist tieng Viet da dong bo.
- README va deployment index da tro dung vao tai lieu moi.
- Danh sach thay doi code sau nay co the duoc thuc hien ma khong can doan lai boundary van hanh.
