# Gemini Handoff - VPS Setup & Initial Deployment

## Context

Mình đang triển khai dự án `one-time-link` trên một VPS đã có sẵn.

Thông tin hiện tại:

- Domain đã thuê: `quorix.io.vn` qua PA Việt Nam
- Hiện tại chưa tạo subdomain nào cho dự án này
- VPS đã có sẵn, nhưng chưa chốt toàn bộ cấu hình ban đầu
- Mục tiêu là nhờ Gemini hướng dẫn tiếp các bước setup VPS, DNS, SSL, reverse proxy và chạy production

## Mục tiêu cần Gemini tiếp tục hướng dẫn

1. Đề xuất cấu trúc subdomain phù hợp cho dự án.
2. Hướng dẫn tạo DNS records trên PA Việt Nam.
3. Hướng dẫn hardening VPS ban đầu.
4. Hướng dẫn cài và cấu hình reverse proxy / HTTPS.
5. Hướng dẫn deploy backend, frontend và Redis cho production.
6. Hướng dẫn kiểm tra sau deploy và checklist an toàn.

## Repo hiện có

Các file liên quan đã có sẵn trong repo:

- `deploy/prod/init-vps.sh`
- `deploy/prod/docker-compose.yml`
- `deploy/prod/Caddyfile`
- `deploy/prod/.env.example`
- `docs/PRODUCTION_CHECKLIST.md`
- `docs/deployment/deployment-guide.md`
- `docs/deployment/production-checklist.md`
- `docs/deployment/quorix-cheap-deployment-plan.md`
- `docs/deployment/provider-selection.md`

## Ghi chú quan trọng

- Nếu Gemini thấy cần đổi cấu trúc subdomain, hãy nói rõ lý do và đề xuất naming cụ thể.
- Nếu có nhiều phương án, ưu tiên phương án đơn giản, dễ vận hành, ít chi phí.
- Hướng dẫn nên theo kiểu từng bước, để mình làm theo trên VPS thật.
- Nếu có phần nào chưa đủ thông tin, hãy hỏi lại ngắn gọn thay vì giả định quá nhiều.

## Prompt gợi ý để đưa cho Gemini

> Mình đang có một VPS và domain `quorix.io.vn` đã mua tại PA Việt Nam. Hiện chưa tạo subdomain nào cho dự án `one-time-link`. Trong repo đã có các file deployment như `deploy/prod/init-vps.sh`, `deploy/prod/docker-compose.yml`, `deploy/prod/Caddyfile`, và `deploy/prod/.env.example`.  
>  
> Hãy đóng vai DevOps guide và hướng dẫn mình từng bước để hoàn tất setup production ban đầu: đề xuất subdomain phù hợp, hướng dẫn tạo DNS records ở PA Việt Nam, hardening VPS, cài Docker/Caddy/Redis nếu cần, cấu hình environment variables, reverse proxy, SSL, và kiểm tra deploy.  
>  
> Mình muốn hướng dẫn thực tế, từng bước, ưu tiên phương án đơn giản và chi phí thấp.

## Trạng thái hiện tại

- VPS: đã có
- Domain: đã có `quorix.io.vn`
- Subdomain: chưa tạo
- Production deploy: chưa bắt đầu

