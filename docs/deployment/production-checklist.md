# Production Checklist Cho Phuong An Sieu Re

Truoc khi sua code hardening hoac bat dau go-live, doc [production-hardening-upgrade-spec.md](production-hardening-upgrade-spec.md).

## 1. Muc tieu

Checklist nay dung cho phuong an:

- website ca nhan `quorix.io.vn` giu nguyen tren Vercel
- frontend `secret-letter` deploy tren Vercel
- backend Go + Redis chay tren 1 VPS
- domain van do PA Viet Nam quan ly

## 2. Cau truc domain de xuat

- `quorix.io.vn`: website ca nhan Hugo/PaperMod
- `secret.quorix.io.vn`: frontend secret-letter
- `api.secret.quorix.io.vn`: backend API

## 3. Viec can chuan bi truoc khi deploy

### Tai khoan

- tai khoan Vercel
- tai khoan VPS provider
- quyen quan ly DNS cua domain tai PA Viet Nam

### May chu

- 1 VPS Linux nho
- SSH key
- IP public

### Du an

- frontend build duoc o local
- backend Go chay duoc o local
- Redis config tuong thich local va production
- env vars duoc tach rieng cho frontend va backend
- `SECRET_ENCRYPTION_KEY` da duoc generate va luu trong secret manager hoac `.env` production
- file cau hinh deploy da duoc backup truoc khi deploy
- staging da duoc deploy va smoke test truoc production

### Quality gate truoc deploy

Chay tu root repository:

```powershell
go test ./...
gosec ./...
govulncheck ./...
cd frontend/web-app
npm ci
npm test
npm run build
npm audit --omit=dev
```

Neu thieu tool Go security:

```powershell
go install github.com/securego/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
```

## 4. DNS checklist

### Tai PA Viet Nam

- giu record cho `quorix.io.vn` nhu hien tai
- tao `CNAME` cho `secret.quorix.io.vn` theo huong dan Vercel
- tao `A` record cho `api.secret.quorix.io.vn` tro toi IP VPS

### Kiem tra

- `secret.quorix.io.vn` resolve dung ve Vercel
- `api.secret.quorix.io.vn` resolve dung ve VPS
- SSL tren ca hai subdomain hoat dong binh thuong

## 5. Vercel checklist cho frontend

### Project setup

- tao project rieng cho frontend secret-letter
- root directory tro dung vao folder frontend
- build command va output directory dung voi framework dang dung

### Environment variables

- `VITE_API_BASE_URL=https://api.secret.quorix.io.vn`

### Domain

- add custom domain `secret.quorix.io.vn`
- verify domain theo huong dan cua Vercel

### Kiem tra

- frontend load duoc
- frontend goi duoc API production
- khong co mixed content

## 6. VPS checklist

### He dieu hanh

- Ubuntu LTS hoac Debian stable

### Bao mat co ban

- tao user rieng, khong lam viec bang root
- tat dang nhap password qua SSH
- chi dung SSH key
- doi SSH port neu ban muon giam bot scan, nhung day la toi uu nho, khong phai lop bao mat chinh
- bat firewall

### Goi can cai

- Go runtime neu can build tren server, hoac copy binary build san
- Redis
- Caddy hoac Nginx
- systemd service files

### Kiem tra

- chi mo cong `80` va `443`, va `22` neu can
- khong public Redis tren `6379`
- Redis chi bind localhost hoac private network
- backend chay o cong noi bo, vi du `127.0.0.1:8080`

## 7. Reverse proxy checklist

### De xuat

- dung `Caddy` cho nhanh va gon

### Cac viec can co

- HTTPS tu dong
- reverse proxy tu `api.secret.quorix.io.vn` vao app Go
- bat gzip hoac compression co ban
- log request o muc toi thieu
- request body limit tai proxy
- reverse proxy la noi set hoac sanitize `X-Forwarded-For`
- chay `./scripts/test-trusted-proxy.sh` hoac `./scripts/test-trusted-proxy.ps1` tu may deploy de xac nhan spoofed header khong doi `ip_hash`

### Kiem tra

- `https://api.secret.quorix.io.vn/healthz` tra ve thanh cong
- `https://api.secret.quorix.io.vn/readyz` tra ve thanh cong
- chung chi TLS hop le

## 8. Backend checklist

### Configuration

- `APP_SERVICE_NAME=secret-letter-api`
- `APP_ENV=production`
- `APP_HOST=0.0.0.0`
- `APP_PORT=8080`
- `ALLOWED_ORIGIN=https://secret.quorix.io.vn`
- `TRUSTED_PROXY_CIDRS` chi gom CIDR cua reverse proxy/load balancer, vi du `172.16.0.0/12` cho Docker bridge
- `REDIS_ADDR`
- `REDIS_PASSWORD` neu co
- `REDIS_DB`
- `REDIS_POOL_SIZE`
- `REDIS_MIN_IDLE`
- `REDIS_MAX_RETRIES`
- `SECRET_ENCRYPTION_KEY` bat buoc, 32 bytes base64url/hex/raw va khong duoc thay doi sau deploy
- `RATE_LIMIT_ENABLED=true`
- `RATE_LIMIT_WINDOW_SECONDS=3600`
- `RATE_LIMIT_CREATE_PER_WINDOW=120`
- `RATE_LIMIT_CONSUME_PER_WINDOW=240`
- `RATE_LIMIT_STATUS_PER_WINDOW=600`
- `RATE_LIMIT_REVEAL_SESSION_PER_WINDOW=240`
- khong duoc rely vao default localhost cho `ALLOWED_ORIGIN` hoac `TRUSTED_PROXY_CIDRS`

### API can co toi thieu

- `POST /api/secrets`
- `GET /api/secrets/{id}/status`
- `POST /api/secrets/{id}/open`
- `POST /api/secrets/{id}/consume` alias tuong thich nguoc
- `GET /healthz`
- `GET /readyz`

### Bao mat va on dinh

- validate request body
- gioi han kich thuoc payload tai API va tai proxy
- CORS chi mo cho frontend cua ban
- rate limiting co ban
- graceful shutdown da duoc implement va verify
- Redis client co timeout ro rang cho ket noi/doc/ghi
- neu deploy bang container thi process API chay non-root
- neu deploy bang container thi root filesystem cua API la read-only va chi cap tmpfs toi thieu cho `/tmp`
- neu deploy bang container thi API drop toan bo Linux capabilities va giu `no-new-privileges`
- khong log plaintext, raw token, fragment key, hoac ciphertext dai bat thuong
- atomic consume voi Redis claim/finalize

## 9. Redis checklist

### Cau hinh

- bind `127.0.0.1`
- giu Redis private trong localhost hoac private compose network
- dat password neu can, du la chi local
- bat append-only neu ban muon giam rui ro mat du lieu do restart
- neu muon dung remote managed Redis thi can mot dot nang cap rieng cho TLS/config

### Nghiep vu

- luu secret voi TTL
- luu lifecycle metadata rieng voi TTL retention ngan hon payload
- consume secret bang claim/finalize atomic

### Kiem tra

- secret het han thi tu xoa
- hai request dong thoi chi co 1 request lay duoc secret

## 10. Frontend checklist ve UX va security

### UX

- trang tao secret ro rang
- trang reveal co nut bam ro rang
- state `already used`, `expired`, `invalid` hien thi than thien

### Security

- ma hoa o client
- key giai ma nam trong fragment `#`
- khong gui fragment len server
- thieu fragment phai hien thi loi ro rang va khong consume secret
- fragment sai format phai hien thi loi key format va khong consume secret
- khong luu plaintext trong localStorage neu khong can

## 11. Monitoring toi thieu

### Logs

- log co timestamp
- log co request id neu co
- log ket qua `created`, `revealed`, `expired`, `blocked`

### Health

- `/healthz` cho app
- kiem tra Redis connectivity trong health check hoac readiness

### Canh bao toi thieu bang tay

- neu frontend khong goi duoc API
- neu Redis khong ket noi duoc
- neu ty le `consume` loi tang bat thuong

## 12. Smoke test bat buoc sau deploy

- chay `./scripts/test-production-smoke.sh` hoac `./scripts/test-production-smoke.ps1`
- tao secret moi va luu lai full link
- kiem tra `status` truoc khi reveal
- restart API service/container truoc lan reveal dau va xac nhan secret cu van mo duoc
- doi `/readyz` ve `200` truoc khi tiep tuc traffic sau restart
- mo link lan dau thanh cong
- mo lai link lan hai nhan `consumed`
- mo link khong co fragment va xac nhan khong bi consume
- mo link voi fragment sai format va xac nhan loi format
- gui request qua lon va xac nhan bi chan dung boundary

## 13. Backup toi thieu

### Can backup gi

- file cau hinh server
- Caddyfile hoac Nginx config
- systemd unit files
- script deploy
- source code dang luu tren Git remote

### Can backup du lieu secret khong?

Khong nhat thiet cho MVP.

Ly do:

- day la du lieu ngan han
- backup secret da ma hoa van tao them do phuc tap van hanh
- voi san pham secret-letter, mat secret sau su co co the chap nhan hon so voi lo secret

## 14. Thu tu deploy de xuat

1. Chuan bi frontend build on o local.
2. Chuan bi backend Go + Redis chay on o local.
3. Thue VPS va harden co ban.
4. Cai Redis, Caddy, binary Go.
5. Cau hinh systemd cho backend.
6. Add DNS `api.secret.quorix.io.vn`.
7. Test API production bang `healthz` va `readyz`.
8. Tao project frontend tren Vercel.
9. Add DNS `secret.quorix.io.vn`.
10. Cau hinh env var frontend tro toi API production.
11. Test full flow create -> reveal -> consumed.
12. Tao mot secret, restart API container/service, roi mo secret do thanh cong de xac nhan `SECRET_ENCRYPTION_KEY` on dinh.
13. Test expired flow.
14. Test preview-bot-safe flow bang cach mo trang ma khong bam reveal.
15. Kiem tra log khong co plaintext, raw token, fragment key, hoac ciphertext dai bat thuong.

## 15. Tieu chi xem nhu deploy thanh cong

- `secret.quorix.io.vn` mo duoc
- tao duoc link moi
- mo link chi hien trang gate, chua reveal ngay
- bam reveal thi doc duoc secret
- bam lan hai nhan `consumed`
- secret het han thi tra ve `expired`
- backend khong log plaintext
- `/healthz` va `/readyz` tra `healthy` khi Redis san sang, va 503 khi Redis mat ket noi
- Redis khong public tren internet
- trusted proxy behavior dung voi topology da chon

## 16. Khuyen nghi cuoi

Khong can lam day du tat ca ky thuat production nang ngay tu dau.

Thu quan trong nhat cho portfolio la:

- dung bai toan
- dung flow bao mat co ban
- deploy that
- domain that
- giai thich duoc trade-off

Neu 4 dieu nay tot, recruiter da co the nhin thay chat luong cua du an.
