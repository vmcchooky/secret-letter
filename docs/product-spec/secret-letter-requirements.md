# Secret Letter Requirements

## 1. Product Summary

`secret-letter` is a web application for sharing secrets through links that can be revealed only once.

The MVP allows a sender to:
- Enter a text secret (up to 10KB)
- Choose an expiration time (1 hour, 24 hours, 7 days)
- Generate a shareable link with client-side encryption

The receiver can:
- Open the link to see a reveal gate page
- Click "Reveal Secret" to consume the secret exactly once
- See clear states for used, expired, or invalid links

## 2. Problem Statement

People need to send temporary secrets like passwords, API keys, recovery codes, and private notes. Sending them through chat or email leaves permanent copies in multiple systems. A one-time link reduces exposure by making the secret short-lived and consumable only once.

The product appears simple but has three critical technical challenges:

1. **Preview bots** can consume links before the intended recipient
2. **Race conditions** can allow multiple reveals of the same secret
3. **Server-side plaintext storage** breaks the trust model

This specification treats these three issues as first-class requirements, not optional hardening.

## 3. Product goals

- Deliver a usable MVP for one-time secret sharing.
- Keep server-side secret exposure as low as possible through client-side encryption.
- Prevent accidental consumption by link preview bots.
- Guarantee atomic one-time reveal semantics.
- Keep the system small enough for a Junior or Mid-level developer to understand and grow.
- Design service boundaries so the system can evolve into microservices without rewriting the product model.

## 4. Non-goals for MVP

- Real-time collaboration
- Multi-user organizations and RBAC
- Large file sharing
- End-to-end audit reporting for enterprises
- Full anonymous anti-abuse perfection
- Long-term secret history after reveal

## 5. Primary users

### Sender

Someone who wants to share a sensitive text secret quickly and safely.

### Receiver

Someone who receives a link and needs one clear path to reveal the message.

### Operator

The developer or maintainer who needs logs, health checks, rate limiting, and predictable runtime behavior.

## 6. Core User Journeys

### 6.1 Create a Secret

1. Sender opens the create page at `https://secret.quorix.io.vn`
2. Sender enters secret text (max 10KB) and selects TTL (1h/24h/7d)
3. Browser generates 256-bit AES-GCM key and random 96-bit nonce
4. Browser encrypts secret using Web Crypto API
5. Frontend sends only ciphertext, nonce, algorithm, and TTL to backend
6. Backend stores encrypted payload with Redis TTL
7. Frontend returns link: `https://secret.quorix.io.vn/s/<secret_id>#<base64_key>`

**Critical**: The decryption key in the URL fragment (`#`) is never sent to the server.

### 6.2 Open a Secret Link

1. Receiver opens the link
2. Server returns reveal gate page (does NOT consume the secret)
3. Page shows "Click to reveal secret" button
4. User clicks button, triggering reveal request
5. Backend atomically claims and finalizes the ciphertext reveal in Redis
6. Frontend decrypts using fragment key and displays secret
7. Subsequent attempts return "already used"

**Critical**: The initial page load must not consume the secret to prevent preview bot consumption.

### 6.3 Error States

The system provides clear feedback for:
- `consumed`: Secret was previously revealed
- `expired`: Secret TTL has passed
- `not_found`: Invalid secret ID
- `missing_key`: URL fragment key is absent
- `invalid_key_format`: URL fragment key is malformed and should not consume the secret
- `decrypt_failed`: Fragment key is well-formed but cannot decrypt the payload
- `network_or_server_error`: API/network failure, shown separately from key errors

## 7. Functional requirements

### FR-1 Client-Side Encryption

The system shall encrypt secrets in the browser before transmission.

**Implementation Requirements:**
- Use AES-GCM with 256-bit keys
- Generate cryptographically secure 96-bit nonces
- Use Web Crypto API: `crypto.subtle.encrypt()`
- Encode ciphertext and nonce as Base64 for transmission
- Store decryption key in URL fragment only

### FR-2 Shareable Secret Letter Generation

The frontend shall generate URLs with server identifier and client-only key.

**Implementation Requirements:**
- URL format: `https://secret.quorix.io.vn/s/{secret_id}#{base64_key}`
- `secret_id`: Server-generated opaque base64url token
- `base64_key`: Base64-encoded 256-bit AES key
- Fragment key must never be logged or transmitted to server

### FR-3 Preview Bot Protection

The system shall require explicit user interaction before consuming secrets.

**Implementation Requirements:**
- Initial page load returns reveal gate UI
- Reveal requires POST request triggered by button click
- No auto-reveal on page load or metadata requests
- User-Agent logging for bot detection (without storing plaintext)

### FR-4 Atomic One-Time Consumption

The reveal operation shall guarantee exactly-once semantics.

**Implementation Requirements:**
- Use Redis scripting to claim, finalize, or abort one-time reveal attempts atomically
- Handle concurrent requests: only first succeeds
- Return deterministic error for subsequent attempts
- No partial reveals or race conditions

### FR-5 TTL-Based Expiration

The system shall support automatic secret expiration.

**Implementation Requirements:**
- TTL options: 1 hour, 24 hours, 7 days
- Use Redis `EXPIRE` for automatic cleanup
- Return `expired` status for TTL-exceeded secrets
- No manual cleanup required

### FR-6 Clear State Communication

The system shall provide unambiguous status information.

**Implementation Requirements:**
- States: `active`, `consumed`, `expired`, `deleted`, `not_found`
- User-friendly error messages
- Structured error responses for API clients
- No generic "error" states for normal conditions

### FR-7 Input Validation and Rate Limiting

The system shall enforce security boundaries.

**Implementation Requirements:**
- Secret size limit: 10KB plaintext
- Rate limiting: 120 creates/hour per IP, 240 open/consume requests/hour per IP, 600 status checks/hour per IP, 240 reveal-session requests/hour per IP
- Input sanitization for all user data
- Request timeout: fail fast; current implementation uses 5s Redis operations and 10s frontend API calls

## 8. Security Requirements

### SR-1 Client-Side Encryption Implementation

**Requirement**: Plaintext secrets must never be transmitted to or stored on the server.

**Implementation**:
- Use Web Crypto API `AES-GCM` algorithm
- Generate keys with `crypto.getRandomValues(new Uint8Array(32))`
- Generate nonces with `crypto.getRandomValues(new Uint8Array(12))`
- Encrypt: `crypto.subtle.encrypt({name: "AES-GCM", iv: nonce}, key, plaintext)`
- Store key in URL fragment, never in request body or headers

### SR-2 Atomic Consumption

**Requirement**: Secret reveal must be atomic and exactly-once.

**Implementation**:
- Atomically claim the payload, finalize metadata, and delete the payload after a successful decrypt
- Handle consumed, expired, and not-found states explicitly through metadata
- No separate GET followed by DEL operations
- Log consumption events without secret content

### SR-3 Preview Bot Resistance

**Requirement**: Initial page load must not consume secrets.

**Implementation**:
- Serve static reveal gate on GET `/s/{id}`
- Require POST `/api/secrets/{id}/consume` for actual reveal
- Validate User-Agent patterns for known bots
- Rate limit by IP address and User-Agent combination

### SR-4 Transport and Storage Security

**Requirement**: All production traffic uses HTTPS with secure headers.

**Implementation**:
- HTTPS-only cookies and headers
- HSTS headers: `max-age=31536000; includeSubDomains`
- CSP headers restricting script sources
- Redis AUTH password in production
- Redis bind to localhost only

### SR-5 Input Validation

**Requirement**: All inputs must be validated and sanitized.

**Implementation**:
- Secret size: max 10KB plaintext before encryption
- TTL values: only 3600, 86400, 604800 seconds allowed
- Secret ID format: opaque base64url token validation
- Request body size limit: 15KB total
- Timeout all requests conservatively; current implementation uses 5s Redis operations and 10s frontend API calls

## 9. Non-functional requirements

### NFR-1 Latency

- create and reveal APIs should feel near-instant for text payloads
- encryption and decryption should happen fast enough on typical mobile devices

### NFR-2 Reliability

- reveal must remain correct under concurrent access
- service restarts must not break valid but unread links

### NFR-3 Simplicity

- the MVP should stay small enough to run locally with `frontend + go services + redis`
- each service should have one clear responsibility

### NFR-4 Operability

- local development should work with Docker Compose
- production deployment should support horizontal scaling of stateless services

## 10. MVP Scope Definition

### Included in MVP

**Core Features:**
- Text secret creation (max 10KB)
- Three TTL options: 1 hour, 24 hours, 7 days
- AES-GCM client-side encryption with Web Crypto API
- One-time reveal with explicit click gate
- Redis-backed atomic claim/finalize consumption
- Clear status messages for used/expired/invalid states

**Security Features:**
- Client-side encryption (server never sees plaintext)
- Preview bot protection via interaction gate
- Basic rate limiting (120 creates/hour, 240 open/reveal requests/hour per IP)
- Input validation and size limits
- HTTPS-only in production

**Operational Features:**
- Health endpoint for monitoring
- Structured JSON logging
- Automatic TTL-based cleanup
- Single-binary deployment

### Explicitly Deferred

**Advanced Features:**
- Custom passphrases or additional encryption layers
- File attachments or binary data
- Email/SMS delivery integration
- User accounts or authentication
- Admin dashboard or analytics

**Enterprise Features:**
- Multi-tenant organizations
- Audit trails and compliance reporting
- Advanced abuse detection beyond rate limiting
- Multi-region deployment
- Database backup/restore (secrets are ephemeral by design)

**Operational Complexity:**
- Kubernetes deployment
- Service mesh or advanced networking
- Automated failover (manual DNS failover is acceptable for MVP)
- Managed Redis (self-hosted Redis is acceptable for MVP)

## 11. Product decisions and rationale

### Why TypeScript in the frontend

- The browser owns encryption and decryption, so frontend correctness matters a lot.
- TypeScript helps make crypto payload shapes, API contracts, and UI states explicit.
- Web Crypto flows are easier to maintain when data types are clear.

### Why Go in the backend

- Go is a strong fit for small, fast API services.
- Concurrency primitives and HTTP tooling are simple enough for a learning project, but production-worthy.
- Shipping a single binary per service keeps deployment simple.

### Why Redis as the primary database

- TTL is built in.
- Atomic commands are extremely useful for one-time reveal semantics.
- The data model for the MVP is short-lived and key-value shaped.

## 12. Success criteria for the MVP

The MVP is successful when:

- a sender can create an encrypted note and copy a link
- a preview bot opening the page does not consume the note
- the first real reveal succeeds
- every later reveal attempt fails safely as `already used`
- expired secrets disappear automatically
- the backend never stores plaintext in normal operation
