# ADR-001: Server-Side Sessions

## Status

Accepted (Commit 3)

## Context

The application initially used cookie-based sessions with HMAC signing
to store user identity. This approach had several limitations:

- Session data was limited to what fits in a cookie (4 KB).
- Flash messages had to be encoded in the cookie.
- Logout required invalidating a client-side value; there was
  no server-side revocation mechanism.
- The HMAC secret was a single point of trust.

## Decision

Replace cookie-based sessions with server-side sessions stored in SQLite.

- The cookie contains only a session ID (64 hex characters, generated
  from 32 bytes of `crypto/rand`).
- Session data (UserID, Flash, Values) lives in the `sessions` table.
- Flash messages are a domain model on the Session struct, not a middleware.
- Sessions are created only after successful authentication.
- PRG (Post-Redirect-Get) for data-changing forms; login errors use
  direct render with NoCache.

## Consequences

### Positive

- Session data is not limited by cookie size.
- Sessions can be revoked server-side (logout deletes from DB).
- No HMAC secret needed for session integrity.
- Flash messages are not exposed to the client.

### Negative

- Every request that loads a session requires a DB query.
- Logout requires a DB write (session deletion).
- Session cleanup requires periodic maintenance (lazy cleanup every 100 ops).

## Technical Notes

- Session ID: 32 bytes from `crypto/rand`, hex-encoded (64 chars).
- Cookie: `HttpOnly`, `SameSite=Lax`, `Path=/`.
- Store uses a repository pattern: `Create`, `Save`, `FindByID`, `Delete`.
- `Touch` updates `last_seen_at` if >5 min since last update, refreshes
  `expires_at` if >50% of idle timeout elapsed.
- Lazy cleanup every 100 writes; deletes all sessions with `expires_at < now`.
- Schema version bumped from 1 to 2 (existing DBs are recreated).
