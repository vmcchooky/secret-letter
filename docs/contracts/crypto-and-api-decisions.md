# Crypto and API Decisions

This document records the current implementation decisions for cryptography, payload limits, API naming, and related Milestone 2 boundaries.

It should be treated as the source of truth for low-level protocol and contract decisions that both frontend and backend must follow.

## 1. Decision Scope

This file defines:

- client-side plaintext size limits
- request body size limits
- allowed TTL values
- one-time link semantics
- cryptographic defaults for MVP
- encoding rules
- API field naming conventions
- status value conventions
- Milestone 2 scope boundaries

## 2. Plaintext and Payload Limits

### Client-side plaintext limit

- Maximum plaintext secret size: `10 KB`

Why:

- large enough for passwords, API keys, and short secure notes
- small enough to reduce abuse risk and memory overhead
- avoids turning the MVP into a file-sharing product too early

The public HTTP API does not accept plaintext `content`. The browser validates this limit before encryption, then submits only `ciphertext`, `nonce`, and `algorithm` to the backend.

### Request body limit

- Maximum HTTP request body size: `15 KB`

Why:

- allows JSON overhead and encoded ciphertext metadata
- still keeps request handling tight and predictable
- supports defensive request validation at the API layer

## 3. Allowed TTL Values

The MVP supports only these expiration options:

- `3600` seconds (`1 hour`)
- `86400` seconds (`24 hours`)
- `604800` seconds (`7 days`)

Why:

- fixed choices keep validation and UX simple
- these values cover the most common real-world secret sharing cases
- free-form TTL input can be added later when the product or library mode becomes more configurable

## 4. One-Time Link Semantics

Public links always burn after read. `burnAfterRead` may be omitted or set to `true` in create requests; `false` is rejected by the API.

Why:

- keeps the product contract unambiguous
- prevents clients from accidentally creating reusable links
- matches the security expectation set by the product name

## 5. Cryptographic Defaults

### Primary algorithm for MVP

- `AES-GCM`

### Key size

- `256-bit`

### Nonce length

- `12 bytes`

Why `AES-GCM`:

- strong and practical authenticated encryption for browser-based applications
- well-supported through the Web Crypto API
- easier to implement correctly in the browser than other attractive alternatives

### Deferred algorithms

These may be added later, but are not part of the Milestone 2 implementation scope:

- `AES-GCM-SIV`
- `ChaCha20-Poly1305`
- `XChaCha20-Poly1305`

Reason for deferring:

- the MVP should prioritize the cleanest browser-native path
- additional algorithm support increases testing and interoperability complexity
- future library-oriented design can expose configurable algorithm selection

## 6. Encoding Rules

The project uses:

- `base64url`

Apply this consistently to:

- ciphertext
- nonce
- serialized key material when represented in the frontend

Why:

- safer for URLs and fragments than standard base64
- reduces escaping and copy-paste issues
- keeps the frontend and future link formats cleaner

## 7. API Field Naming Convention

The public API uses:

- `camelCase`

Examples:

- `secretId`
- `expiresAt`
- `ttlSeconds`
- `sessionId`

Why:

- aligns naturally with TypeScript and frontend usage
- still easy to represent from Go using explicit JSON tags
- avoids mixed naming styles across the stack

### Go naming guidance

- Go struct fields should continue to use `PascalCase`
- JSON serialization should use explicit `camelCase` tags

Example:

```go
type CreateSecretResponse struct {
    SecretID  string `json:"secretId"`
    ExpiresAt string `json:"expiresAt"`
}
```

## 8. Status Value Convention

The canonical status values are:

- `active`
- `consumed`
- `expired`
- `deleted`
- `not_found`

Why:

- they are explicit enough for UX and debugging
- they avoid ambiguity between missing data and consumed data

## 9. Metadata Strategy

The system should not rely on a single destructive Redis read result for all state interpretation.

Recommended approach:

- one key for encrypted secret payload
- one metadata record for lifecycle state

Metadata should be capable of representing:

- `createdAt`
- `expiresAt`
- `consumedAt`
- current status

Why:

- lets the system distinguish `expired`, `consumed`, and `not_found`
- improves UX accuracy
- supports later operational visibility without storing plaintext

## 10. Reveal Session Scope

Reveal session support remains part of the architecture. The current codebase implements a lightweight reveal-session handshake in the HTTP API, but the broader anti-bot and orchestration layer is still an architectural boundary:

- Milestone 2 did **not** require the reveal-session handshake

Why:

- Milestone 2 should stay focused on secret creation and storage
- reveal sessions belong more naturally to the anti-bot and reveal-control work in later milestones

## 11. Frontend Decryption Failure UX

The frontend should distinguish:

- missing fragment key: `Liên kết không hợp lệ. Thiếu khóa giải mã.`
- malformed fragment key: `Khóa giải mã trong liên kết không đúng định dạng. Hãy kiểm tra lại phần sau dấu #.`
- decrypt failure after a successful open: `Khóa giải mã không khớp hoặc dữ liệu đã bị hỏng.`
- API/network/server errors: show the API-specific message instead of blaming the key

Why:

- it is understandable for users
- it does not leak unnecessary implementation detail
- it avoids telling users the key is wrong when the API or network is the real problem

## 12. Milestone 2 Boundary

Milestone 2 should implement:

- create secret form
- browser-side plaintext validation
- client-side encryption with `AES-GCM`
- ciphertext-only public API submission
- API request using `camelCase`
- backend validation and secret creation path
- Redis-backed storage with TTL

Milestone 2 should not yet implement:

- broader reveal session orchestration and advanced anti-bot heuristics
- multiple algorithm support
- production-grade failover logic in code

## 13. Change Policy

If any of these decisions change later:

- update this file first
- then update `public-http-api.md`
- then update affected frontend and backend types

This keeps protocol decisions explicit and prevents silent divergence between docs and code.
