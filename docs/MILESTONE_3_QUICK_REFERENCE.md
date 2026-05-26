# Milestone 3 Quick Reference

**Milestone:** Reveal Gate and Status Checking  
**Status:** ✅ Complete

## Quick Commands

### Start Services
```bash
# Redis
docker run -d -p 6379:6379 redis:7-alpine

# Backend
cd backend && go run ./cmd/api

# Frontend
cd frontend/web-app && npm run dev
```

### Run Tests
```bash
# Backend unit tests
cd backend && go test ./...

# Backend integration tests
cd backend && go test ./test -v

# Manual test script (PowerShell)
.\scripts\test-milestone3-reveal.ps1

# Manual test script (Bash)
./scripts/test-milestone3-reveal.sh
```

---

## API Endpoints

### Status Endpoint
```http
GET /api/secrets/{secretId}/status
```

**Response (200 - Active):**
```json
{
  "secretId": "uuid",
  "status": "active",
  "createdAt": "2026-04-16T12:00:00Z",
  "expiresAt": "2026-04-16T13:00:00Z"
}
```

**Response (200 - Not Found):**
```json
{
  "secretId": "uuid",
  "status": "not_found",
  "message": "Secret not found or has expired."
}
```

### Open Endpoint
```http
POST /api/secrets/{secretId}/open
Content-Type: application/json

{}
```

`POST /api/secrets/{secretId}/consume` is still available as a compatibility alias.

**Response (200 - Success):**
```json
{
  "secretId": "uuid",
  "ciphertext": "base64url-encoded",
  "nonce": "base64url-encoded",
  "algorithm": "AES-GCM",
  "consumedAt": "2026-04-16T12:30:00Z"
}
```

**Response (410 - Already Consumed):**
```json
{
  "error": "SECRET_CONSUMED",
  "message": "This secret has already vanished.",
  "timestamp": "2026-04-16T12:30:00Z",
  "request_id": "uuid"
}
```

---

## Frontend Routes

### Home Page
```
http://localhost:5173/
```
- Create secret form
- Health status display

### Reveal Page
```
http://localhost:5173/reveal/{secretId}#{key}
```
- Reveal gate UI
- Status check
- Consume + decrypt
- Error handling

---

## User Flow

### 1. Create Secret
1. Open http://localhost:5173
2. Enter plaintext
3. Select TTL (1h, 24h, 7d)
4. Click "Tạo liên kết"
5. Copy generated link

### 2. Reveal Secret
1. Open link in browser
2. See reveal gate (secret NOT consumed)
3. Read warning message
4. Click "Nhấn để xem bí mật"
5. See decrypted plaintext
6. Click "Sao chép" to copy

### 3. Verify One-Time
1. Refresh page
2. See error: "đã được xem trước đó"
3. Cannot view again

---

## Error Messages

| Scenario | Vietnamese Message |
|----------|-------------------|
| Already consumed | Bí mật này đã được xem trước đó. |
| Not found/expired | Bí mật không tồn tại hoặc đã hết hạn. |
| Invalid key | Liên kết không hợp lệ hoặc dữ liệu đã bị hỏng. |
| Missing secret ID | Liên kết không hợp lệ. Thiếu ID bí mật. |
| Missing key | Liên kết không hợp lệ. Thiếu khóa giải mã. |

---

## Testing Checklist

### Manual Testing
- [ ] Create secret → receive link
- [ ] Open link → see reveal gate
- [ ] Status shows "active"
- [ ] Click reveal → see plaintext
- [ ] Refresh → see "already consumed"
- [ ] Invalid key → see error
- [ ] Expired secret → see error

### Integration Testing
- [ ] Complete reveal flow
- [ ] Concurrent consumption
- [ ] Multiple secrets independence
- [ ] Non-existent secret handling

---

## Key Files

### Backend
- `backend/internal/secret/redis_service.go` - Status and consume logic
- `backend/internal/httpapi/handlers.go` - HTTP handlers
- `backend/internal/httpapi/reveal_test.go` - Unit tests
- `backend/test/integration_test.go` - Integration tests

### Frontend
- `frontend/web-app/src/pages/RevealPage.tsx` - Reveal page
- `frontend/web-app/src/lib/api.ts` - API client
- `frontend/web-app/src/lib/crypto.ts` - Decryption

### Scripts
- `scripts/test-milestone3-reveal.ps1` - PowerShell test
- `scripts/test-milestone3-reveal.sh` - Bash test

### Documentation
- `docs/MILESTONE_3_COMPLETION.md` - Full report
- `docs/MILESTONE_3_QUICK_REFERENCE.md` - This file

---

## Troubleshooting

### Backend Issues

**Redis connection failed:**
```bash
# Check Redis is running
docker ps | grep redis

# Start Redis if not running
docker run -d -p 6379:6379 redis:7-alpine
```

**Port 8080 already in use:**
```bash
# Change port in .env
APP_PORT=8081
```

### Frontend Issues

**React Router not found:**
```bash
cd frontend/web-app
npm install react-router-dom
```

**CORS errors:**
```bash
# Check ALLOWED_ORIGIN in backend .env
ALLOWED_ORIGIN=http://localhost:5173
```

### Test Issues

**Integration tests skipped:**
- Backend must be running on localhost:8080
- Redis must be running on localhost:6379

**Test script fails:**
- Check backend is running
- Check Redis is running
- Check no firewall blocking

---

## Performance Notes

- **Status check:** ~5ms (Redis GET)
- **Open:** ~10ms (Redis claim/finalize + JSON)
- **Decrypt:** ~5ms (Web Crypto API)
- **Total reveal time:** ~20ms

---

## Security Notes

- ✅ Fragment key never sent to server
- ✅ Status check is non-destructive
- ✅ Atomic Redis claim/finalize prevents race conditions
- ✅ 5-second timeout on all Redis operations
- ✅ Explicit user action required to reveal

---

## Next Milestone

**Milestone 4:** Atomic Consumption and Race Condition Prevention
- Rate limiting
- Advanced error handling
- Performance optimization
- Production readiness

---

**For detailed information, see:** [MILESTONE_3_COMPLETION.md](MILESTONE_3_COMPLETION.md)
