# Deployment Documentation

Tài liệu deployment cho secret-letter application.

## Current Status

**Milestone:** 4/7 complete (57%)  
**Status:** Production deployment planning and hardening in progress
**Next:** Production Deployment (Milestone 5)

## Documents

- **`production-hardening-upgrade-spec.md`** - Spec trung tam cho dot hardening truoc khi sua code production
- **`shared-vps-edge.md`** - Reference deployment path duoc uu tien cho dot hardening/go-live dau tien
- **`deployment-decision-summary.md`** - Tóm tắt các quyết định deployment quan trọng nhất
- **`quorix-cheap-deployment-plan.md`** - Phân tích phương án deploy siêu rẻ cho `quorix.io.vn` và `secret-letter`
- **`production-checklist.md`** - Checklist từng bước để đưa phương án deploy siêu rẻ lên production
- **`provider-selection.md`** - Hướng dẫn chọn provider, gồm cả mô hình `VPS Việt Nam + Oracle Cloud standby`
- **`failover-runbook.md`** - Runbook failover bằng tay từ `VPS Việt Nam` sang `Oracle Cloud`
- **`deployment-guide.md`** - Complete deployment guide (English)

## Quick Links

- [Production Hardening Upgrade Spec](production-hardening-upgrade-spec.md) - Scope and open questions before implementation
- [Shared VPS Edge](shared-vps-edge.md) - Primary production reference path
- [Production Checklist](../PRODUCTION_CHECKLIST.md) - Main production deployment checklist
- [Troubleshooting Guide](../TROUBLESHOOTING.md) - Common issues and solutions
- [Milestone 4 Completion](../MILESTONE_4_COMPLETION.md) - Production readiness report

## Notes

- Các tài liệu trong thư mục này là legacy documentation bằng tiếng Việt
- Tài liệu chính hiện tại là `docs/PRODUCTION_CHECKLIST.md` (English)
- `shared-vps-edge.md` la production runbook uu tien hien tai
- `deployment-guide.md` (English) la tai lieu secondary/reference only
- Trước khi code hardening, đọc `production-hardening-upgrade-spec.md`
- Reference production path hien tai: `shared-vps-edge.md`

---

**Last Updated:** 2026-04-16  
**License:** MIT License - Copyright (c) 2026 Quorix Việt Nam  
**Contact:** contact@quorix.io.vn | [quorix.io.vn](https://quorix.io.vn)
