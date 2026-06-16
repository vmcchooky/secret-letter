# Frontend - Secret Letter Web App

React + TypeScript frontend cho secret-letter application. Client-side encryption với Web Crypto API.

## Current Status

**Milestone:** 4/7 complete (57%)  
**Status:** Production-ready  
**Next:** Production deployment (Vercel)

## Features

### Implemented ✅

**Create Secret Flow:**
- ✅ Create secret form với TTL selection (1h, 24h, 7d)
- ✅ Client-side AES-GCM 256-bit encryption
- ✅ 12-byte nonce generation
- ✅ Base64url encoding (RFC 4648)
- ✅ Plaintext size validation (10KB limit)
- ✅ Byte counter
- ✅ Secret link generation với key trong URL fragment
- ✅ Copy to clipboard functionality

**Reveal Secret Flow:**
- ✅ Reveal gate UI (preview bot protection)
- ✅ Status check (non-destructive)
- ✅ One-time consumption
- ✅ Client-side decryption
- ✅ Error handling cho all states
- ✅ Expiry time display

**UI/UX:**
- ✅ Vietnamese interface
- ✅ Responsive design
- ✅ Clear error messages
- ✅ Loading states
- ✅ Success/error feedback

### Security

- ✅ **Plaintext never sent to server** - All encryption happens in browser
- ✅ **Key in URL fragment** - Fragment never sent to server
- ✅ **Preview bot protection** - Reveal gate prevents accidental consumption
- ✅ **Input validation** - Size limits và format checking
- ✅ **No sensitive data in logs** - Client-side only

## Tech Stack

- **Framework:** React 19
- **Language:** TypeScript 5.8
- **Build Tool:** Vite 7
- **Crypto:** Web Crypto API (native browser API)
- **Routing:** React Router 7

## Local Development

### Prerequisites
- Node.js 18+
- Backend running on `http://localhost:8080`

### Setup

```bash
# Install dependencies
npm install

# Start dev server
npm run dev
```

Frontend sẽ chạy tại `http://localhost:5173`

### Configuration

Create `.env` file:

```bash
VITE_API_BASE_URL=http://localhost:8080
```

## Project Structure

```
src/
├── components/
│   └── CreateSecretForm.tsx    # Create secret form component
├── pages/
│   ├── HomePage.tsx             # Home page với create form
│   └── RevealPage.tsx           # Reveal page với gate UI
├── lib/
│   ├── crypto.ts                # Web Crypto API helpers
│   ├── api.ts                   # API client functions
│   └── types.ts                 # TypeScript types
├── App.tsx                      # Main app với routing
├── main.tsx                     # Entry point
└── styles.css                   # Global styles
```

## Crypto Implementation

### Encryption Flow

```typescript
// 1. Generate 256-bit key
const key = await generateKey();

// 2. Generate 12-byte nonce
const nonce = generateNonce();

// 3. Encrypt plaintext
const ciphertext = await encryptSecret(plaintext, key, nonce);

// 4. Export key for URL fragment
const keyBase64 = await exportKeyToBase64Url(key);

// 5. Create link
const link = `${baseUrl}/reveal/${secretId}#${keyBase64}`;
```

### Decryption Flow

```typescript
// 1. Extract key from URL fragment
const keyBase64 = window.location.hash.slice(1);

// 2. Import key
const key = await importKeyFromBase64Url(keyBase64);

// 3. Decrypt ciphertext
const plaintext = await decryptSecret(ciphertext, key, nonce);
```

## API Integration

### Create Secret

```typescript
const response = await createSecret({
  ciphertext: "base64url-encoded",
  nonce: "base64url-encoded",
  algorithm: "AES-GCM",
  ttlSeconds: 3600
});
// Returns: { secretId, expiresAt }
```

### Check Status

```typescript
const status = await getSecretStatus(secretId);
// Returns: { secretId, status, createdAt, expiresAt }
```

### Consume Secret

```typescript
const response = await consumeSecret(secretId);
// Returns: { secretId, ciphertext, nonce, algorithm, consumedAt }
```

### Create Reveal Session

```typescript
const session = await createRevealSession(secretId);
// Returns: { sessionId, secretId, status, expiresAt }
```

## User Flows

### Create Secret

1. User enters plaintext (max 10KB)
2. User selects TTL (1h, 24h, 7d)
3. Click "Tạo liên kết"
4. Frontend encrypts plaintext client-side
5. Frontend sends ciphertext to backend
6. Backend returns secret ID
7. Frontend generates link với key trong fragment
8. User copies link

### Reveal Secret

1. User opens link
2. Frontend extracts secret ID from URL path
3. Frontend extracts key from URL fragment
4. Frontend checks status (non-destructive)
5. User sees reveal gate với warning
6. User clicks "Nhấn để xem bí mật"
7. Frontend creates a short-lived reveal session and consumes secret (one-time)
8. Frontend decrypts ciphertext client-side
9. User sees plaintext
10. User can copy plaintext

### Error Handling

- **Already consumed:** "Bí mật này đã được xem trước đó"
- **Not found/expired:** "Bí mật không tồn tại hoặc đã hết hạn"
- **Invalid key:** "Liên kết không hợp lệ hoặc dữ liệu đã bị hỏng"
- **Missing secret ID:** "Liên kết không hợp lệ. Thiếu ID bí mật"
- **Missing key:** "Liên kết không hợp lệ. Thiếu khóa giải mã"

## Build for Production

```bash
# Build
npm run build

# Preview build
npm run preview
```

Build output trong `dist/` directory.

## Deployment

Recommended: Deploy to Vercel

```bash
# Install Vercel CLI
npm i -g vercel

# Deploy
vercel
```

Set environment variable:
- `VITE_API_BASE_URL`: Production API URL

## Testing

Automated tests are available:

1. Unit/component tests: `npm test`
2. End-to-end browser flow: `npm run test:e2e`
3. Deployed-site browser flow: `E2E_TARGET_BASE_URL=https://secret.quorix.io.vn npm run test:e2e`

Manual testing workflow is still available if you want to poke the app by hand:

1. Start backend: `go run ./backend/cmd/api`
2. Start frontend: `npm run dev`
3. Test create flow
4. Test reveal flow
5. Test error scenarios

## Documentation

- [API Contract](../../docs/contracts/public-http-api.md)
- [Crypto Specs](../../docs/contracts/crypto-and-api-decisions.md)
- [Milestone 3 Completion](../../docs/MILESTONE_3_COMPLETION.md)
- [Milestone 4 Completion](../../docs/MILESTONE_4_COMPLETION.md)

## Future Enhancements

- [ ] Custom expiration times
- [ ] File upload support
- [ ] Password protection
- [ ] Email delivery

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

Copyright (c) 2026 Quorix Việt Nam

## Contact

**Developed by:** Quorix Việt Nam

- **Website:** [quorix.io.vn](https://quorix.io.vn)
- **Email:** contact@quorix.io.vn
- **Facebook:** [facebook.com/quorixvietnam](https://facebook.com/quorixvietnam)
