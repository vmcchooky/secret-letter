# Deployment Decision Summary

## 1. Muc tieu

Tai lieu nay la ban tom tat cuoi cung de de theo doi.

No gom lai cac quyet dinh chinh ve:

- domain
- ha tang
- provider
- chi phi
- failover
- thu tu trien khai

## 2. Boi canh hien tai

- `quorix.io.vn` la website ca nhan cua ban
- website nay dang build bang Hugo + PaperMod
- website nay dang deploy tren Vercel
- domain dang quan ly qua PA Viet Nam
- `one-time-link` duoc lam de:
  - dua vao portfolio
  - xay dung brand ca nhan
  - co san pham that cho recruiter xem
  - giu chi phi rat thap

## 3. Quyet dinh tong quan

Phuong an chot hien tai la:

- giu `quorix.io.vn` tren Vercel nhu hien tai
- deploy frontend `one-time-link` tren Vercel
- dung `VPS Viet Nam` lam production primary
- dung `Oracle Cloud VPS` lam standby va noi hoc them cloud
- chua deploy microservices thanh nhieu may rieng
- chua dung active-active

## 4. Domain va vai tro

- `quorix.io.vn`: website ca nhan
- `secret.quorix.io.vn`: frontend one-time-link
- `api.secret.quorix.io.vn`: backend API

## 5. Ha tang chot

### Frontend

- host tren Vercel
- dung React + TypeScript

### Backend

- chay 1 binary Go trong giai doan dau
- reverse proxy bang Caddy hoac Nginx

### Data store

- Redis self-host tren cung VPS voi backend

## 6. Vai tro tung node

## Primary

- provider: VPS nha cung cap o Viet Nam
- chay:
  - Go API
  - Redis
  - reverse proxy

### Ly do

- latency dep hon cho nguoi dung Viet Nam
- de thanh toan va support
- phu hop de lam production chinh cho portfolio

## Standby

- provider: Oracle Cloud VPS
- chay:
  - Go API standby
  - Redis standby hoac node recovery
  - reverse proxy standby

### Ly do

- hoc them cloud
- co phuong an failover
- khong phu thuoc hoan toan vao 1 provider

## 7. Chien luoc availability

### Khong chon

- active-active
- multi-region phuc tap
- dong bo Redis hai chieu som

### Chon

- `primary + standby`
- failover bang tay qua DNS

### Ly do

Voi bai toan one-time-link, active-active som se lam tang do phuc tap o:

- consume 1 lan
- TTL
- race condition
- dong bo trang thai

Trong khi do, `primary + standby` da du de:

- giam downtime
- hoc van hanh
- co cau chuyen hay cho portfolio

## 8. Chi phi da chot

### Gan nhu khong tang them

- website ca nhan: giu nguyen
- domain: giu nguyen
- frontend one-time-link tren Vercel: co the `0 USD`

### Chi phi tang them chinh

- `1 VPS primary`

### Chi phi co the tang them nhung chua bat buoc

- snapshot
- backup
- monitoring
- managed Redis

### Ket luan chi phi

Trong giai doan dau, chi phi tang them chu yeu la:

- `VPS`

Oracle Cloud standby co the dung theo huong free tier neu kha thi, nhung khong nen la noi production duy nhat.

## 9. DNS va deploy flow

### DNS

Tai PA Viet Nam:

- giu record cua `quorix.io.vn`
- tao `CNAME` cho `secret.quorix.io.vn` tro toi Vercel
- tao `A` record cho `api.secret.quorix.io.vn` tro toi IP VPS primary

### Deploy flow

1. Deploy frontend len Vercel.
2. Deploy backend Go + Redis len VPS Viet Nam.
3. Cau hinh `api.secret.quorix.io.vn`.
4. Test full flow create -> reveal -> already used.
5. Provision Oracle standby.
6. Test failover gia lap.

## 10. Security minimum can co

- client-side encryption
- key giai ma nam trong URL fragment
- khong log plaintext
- Redis chi bind localhost
- CORS chi mo cho frontend cua ban
- rate limiting co ban
- atomic consume voi Redis
- HTTPS cho frontend va API

## 11. Tieu chi thanh cong cua ban deploy dau tien

Ban deploy duoc xem la thanh cong khi:

- `secret.quorix.io.vn` mo duoc
- tao duoc secret moi
- mo link khong reveal ngay
- bam reveal thi doc duoc secret
- bam lai lan 2 thi nhan `already used`
- secret het han tra ve `expired`
- backend khong log plaintext

## 12. Thu tu tai lieu nen doc

Neu muon theo doi de dang, doc theo thu tu nay:

1. `docs/product-spec/one-time-link-requirements.md`
2. `docs/product-spec/one-time-link-architecture.md`
3. `docs/product-spec/one-time-link-milestones.md`
4. `docs/deployment/deployment-decision-summary.md`
5. `docs/deployment/quorix-cheap-deployment-plan.md`
6. `docs/deployment/production-checklist.md`
7. `docs/deployment/provider-selection.md`
8. `docs/deployment/failover-runbook.md`

## 13. Quyet dinh cuoi cung

Neu chi can mot cau de nho:

`one-time-link` se duoc deploy voi frontend tren Vercel, backend Go + Redis tren 1 VPS Viet Nam lam primary, Oracle Cloud lam standby, va failover bang tay qua DNS.
