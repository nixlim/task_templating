## 1. Header

# Feature Specification: Password Reset

**Created**: 2026-05-08
**Status**: Draft
**Intent**: Provide self-service password reset via emailed time-limited token, plus an admin force-reset action with audit. Out of scope: SMS-based reset, security questions, multi-factor authentication, password rotation policy, account lockout.

---

## 2. Implementation Scope

**Capabilities**:

1. Authenticated email-bound reset request that emits a single-use, time-limited token via email.
2. Token-based completion endpoint that updates the password and invalidates all existing sessions.
3. Admin force-reset that invalidates the current password and triggers the same reset email path.
4. Audit log entries for every issued, used, and expired token, plus every admin force-reset.

**Guard rails**:

- Must not leak whether an email is registered (anti-enumeration).
- Must not introduce a new email-delivery dependency — use the existing `mailer` service at `internal/mail/sender.go`.
- Must not change the existing `users` table schema; tokens live in a separate table.
- Must not log token strings, password values, or full email addresses in any environment.

---

## 3. Existing Codebase Context

| Area | Existing files | Required change |
|------|----------------|-----------------|
| HTTP routing | `internal/http/router.go` | Add three routes: `POST /v1/auth/reset-request`, `POST /v1/auth/reset-complete`, `POST /v1/admin/users/{id}/force-reset` |
| User store | `internal/store/users.go` | Add `UpdatePasswordHash(userID, hash)` and `InvalidateAllSessions(userID)` methods |
| Mailer | `internal/mail/sender.go` | Add `SendPasswordResetEmail(to, link)` method using existing `Transactional` template flow |
| Auth middleware | `internal/auth/middleware.go` | Reuse existing `RequireRole("admin")` for force-reset endpoint |
| Audit log | `internal/audit/log.go` | Add four new event types: `reset.token.issued`, `reset.token.used`, `reset.token.expired`, `reset.admin.forced` |

---

## 4. Terminology

| Term | Definition |
|------|------------|
| Reset token | Cryptographically random 32-byte value (base64url encoded), single-use, with a TTL. |
| Token TTL | Time from issuance until automatic expiry. Set to 60 minutes. |
| Active session | Any session row in `sessions` with `revoked_at IS NULL` and `expires_at > now()`. |
| Force-reset | Admin action that nullifies a user's password hash and triggers a reset email path. |

---

## 5. Surface / API Inventory

### New surfaces

- `POST /v1/auth/reset-request` — request a reset email by email address.
- `POST /v1/auth/reset-complete` — submit token plus new password to complete reset.
- `POST /v1/admin/users/{id}/force-reset` — admin-triggered reset for a target user.

### Modified surfaces

- None.

### Deferred From This Spec

- SMS-based reset delivery — out of scope; product has not chosen SMS provider.
- Security questions as an alternative recovery factor — explicitly disallowed by security review.
- Differentiated rate-limit thresholds for trusted IP ranges — future iteration once IP reputation service ships.

---

## 6. Data Model Changes

**DM-001**: Create `password_reset_tokens` table.

```sql
CREATE TABLE password_reset_tokens (
  token_hash      BYTEA       PRIMARY KEY,
  user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  issued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at      TIMESTAMPTZ NOT NULL,
  used_at         TIMESTAMPTZ,
  issued_by       TEXT        NOT NULL CHECK (issued_by IN ('user', 'admin')),
  admin_actor_id  UUID        REFERENCES users(id)
);

CREATE INDEX idx_prt_user_active
  ON password_reset_tokens(user_id)
  WHERE used_at IS NULL;
```

Migration / backfill:

- New table; no backfill required.
- `token_hash` stores SHA-256 of the raw token; raw token is never persisted.

---

## 7. Functional Requirements

### Reset Request

- **FR-001** (MUST): The system MUST issue a reset token with TTL of 60 minutes when `POST /v1/auth/reset-request` receives a registered email and rate limit is not exceeded, and email the user a link of the form `https://{host}/reset?token={raw_token}`.
- **FR-002** (MUST): The system MUST return HTTP 202 with body `{"status":"ok"}` for every well-formed reset-request, regardless of whether the email is registered, and MUST send no email when the email is not registered.
- **FR-003** (MUST): The system MUST reject reset-request calls exceeding 5 requests per email address per rolling 60-minute window with HTTP 429 and error code `rate.exceeded`.
- **FR-004** (MUST): When a new token is issued for a user, the system MUST mark all prior unexpired tokens for the same user as `used_at = now()` before inserting the new token.

### Reset Completion

- **FR-005** (MUST): The system MUST update the user's password hash and set `used_at = now()` on the token when `POST /v1/auth/reset-complete` receives a valid unused unexpired token plus a password meeting FR-008.
- **FR-006** (MUST): The system MUST invalidate all active sessions for the user (set `revoked_at = now()`) atomically with the password update from FR-005.
- **FR-007** (MUST): The system MUST reject expired tokens with HTTP 400 and error code `token.expired`, and reject already-used tokens with HTTP 400 and error code `token.used`.
- **FR-008** (MUST): The system MUST require submitted passwords to satisfy: minimum length 12 characters, at least one lowercase letter, at least one uppercase letter, at least one digit. Failures return HTTP 422 with error code `password.weak`.
- **FR-009** (SHOULD): The system SHOULD rate-limit reset-complete attempts to 5 per token before treating the token as compromised and marking it used.

### Admin Force-Reset

- **FR-010** (MUST): The system MUST, on `POST /v1/admin/users/{id}/force-reset`, set the target user's password hash to NULL, issue a new reset token with `issued_by = 'admin'` and `admin_actor_id = caller.id`, and send the reset email.
- **FR-011** (MUST): The system MUST reject force-reset calls from callers without role `admin` with HTTP 403 and error code `auth.forbidden`.

### Audit and Observability

- **FR-012** (MUST): The system MUST emit one audit log entry per token state transition (`reset.token.issued`, `reset.token.used`, `reset.token.expired`) and per admin action (`reset.admin.forced`), with user_id, actor (where applicable), and event timestamp. Audit entries MUST NOT contain raw or hashed token values.
- **FR-013** (SHOULD): The system SHOULD emit Prometheus counters `password_reset_requested_total`, `password_reset_completed_total`, `password_reset_expired_total`, labeled by `issued_by`.

---

## 8. API / Schema Contracts

### `POST /v1/auth/reset-request` — request a reset email

**Request**:

```json
{
  "email": "string — RFC 5322 address, max 254 chars"
}
```

**Response 202**:

```json
{ "status": "ok" }
```

**Auth**: none. **Content-Type**: `application/json`.

### `POST /v1/auth/reset-complete` — complete a reset

**Request**:

```json
{
  "token": "string — base64url, 43 chars",
  "new_password": "string — 12-128 chars"
}
```

**Response 200**:

```json
{ "status": "ok" }
```

**Auth**: none (token is the credential). **Content-Type**: `application/json`.

### `POST /v1/admin/users/{id}/force-reset` — admin force-reset

**Request**: empty body. Path param `id` is a UUID.

**Response 202**:

```json
{ "status": "ok" }
```

**Auth**: bearer token with role `admin`. **Content-Type**: `application/json`.

---

## 9. Error Contract

| Condition | Status | Error code | Notes |
|-----------|--------|------------|-------|
| Token TTL elapsed | 400 | `token.expired` | User-visible. Not retryable with same token. |
| Token already used | 400 | `token.used` | User-visible. Not retryable. |
| Password fails strength check | 422 | `password.weak` | User-visible. Retryable with stronger password. |
| Rate limit exceeded | 429 | `rate.exceeded` | Retry after `Retry-After` header value. |
| Caller lacks admin role | 403 | `auth.forbidden` | Logged at warn level. |
| Malformed request body | 400 | `request.invalid` | User-visible. |

Error response shape:

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

---

## 10. Behavioral Scenarios

### Scenario: Registered user requests a reset

**Traces to**: FR-001
**Category**: Happy Path

- **Given** a registered user with email `alice@example.com` and no active reset token
- **When** the client submits `POST /v1/auth/reset-request` with `{"email":"alice@example.com"}`
- **Then** the response is HTTP 202 with body `{"status":"ok"}`
- **And** an email is queued to `alice@example.com` containing a link with a 43-character base64url token
- **And** a `password_reset_tokens` row exists with `expires_at = issued_at + 60 minutes`
- **And** an audit entry of type `reset.token.issued` is emitted

### Scenario: Reset request for unknown email returns generic success

**Traces to**: FR-002
**Category**: Happy Path

- **Given** no user exists with email `nobody@example.com`
- **When** the client submits `POST /v1/auth/reset-request` with `{"email":"nobody@example.com"}`
- **Then** the response is HTTP 202 with body `{"status":"ok"}`
- **And** no email is queued
- **And** no `password_reset_tokens` row is inserted

### Scenario: Reset request exceeds rate limit

**Traces to**: FR-003
**Category**: Error Path

- **Given** the email `alice@example.com` has triggered 5 reset-requests within the past 60 minutes
- **When** the client submits a sixth `POST /v1/auth/reset-request` for that email
- **Then** the response is HTTP 429 with body `{"error":{"code":"rate.exceeded","message":"..."}}`
- **And** no token is issued and no email is queued

### Scenario: New reset request invalidates prior unused token

**Traces to**: FR-004
**Category**: Edge Case

- **Given** user Alice holds an unused, unexpired token T1
- **When** Alice triggers a new reset request and a new token T2 is issued
- **Then** T1 has `used_at` set to the time of the new request
- **And** T2 is the only active token for Alice

### Scenario: User completes reset with valid token and strong password

**Traces to**: FR-005, FR-006, FR-008
**Category**: Happy Path

- **Given** user Alice holds a valid, unused, unexpired token and has two active sessions
- **When** Alice submits `POST /v1/auth/reset-complete` with the token and password `Str0ngPassw0rd!`
- **Then** the response is HTTP 200 with body `{"status":"ok"}`
- **And** Alice's password hash is updated
- **And** both of Alice's prior sessions have `revoked_at` set
- **And** the token row has `used_at` set

### Scenario: Reset completion rejects expired token

**Traces to**: FR-007
**Category**: Error Path

- **Given** Alice holds a token whose `expires_at` is in the past
- **When** Alice submits `POST /v1/auth/reset-complete` with that token
- **Then** the response is HTTP 400 with `error.code = "token.expired"`
- **And** Alice's password hash is unchanged

### Scenario: Reset completion rejects weak password

**Traces to**: FR-008
**Category**: Error Path

- **Given** Alice holds a valid unexpired token
- **When** Alice submits `POST /v1/auth/reset-complete` with new password `short`
- **Then** the response is HTTP 422 with `error.code = "password.weak"`
- **And** the token is not consumed

### Scenario: Token reuse rejected after first successful completion

**Traces to**: FR-007
**Category**: Edge Case

- **Given** Alice already used token T to reset her password
- **When** any caller submits `POST /v1/auth/reset-complete` with token T
- **Then** the response is HTTP 400 with `error.code = "token.used"`

### Scenario: Admin forces reset for a target user

**Traces to**: FR-010, FR-012
**Category**: Happy Path

- **Given** an admin caller and a target user Bob
- **When** the admin submits `POST /v1/admin/users/{bob_id}/force-reset`
- **Then** the response is HTTP 202
- **And** Bob's password hash is NULL
- **And** a token row exists with `issued_by = 'admin'` and `admin_actor_id = admin_id`
- **And** an audit entry of type `reset.admin.forced` is emitted with `actor = admin_id`

### Scenario: Non-admin cannot call force-reset

**Traces to**: FR-011
**Category**: Error Path

- **Given** a caller with role `user` (not `admin`)
- **When** the caller submits `POST /v1/admin/users/{any_id}/force-reset`
- **Then** the response is HTTP 403 with `error.code = "auth.forbidden"`
- **And** the target user's password hash is unchanged

---

## 11. Testing Requirements

### Unit

- Token generator: produces 32 bytes of randomness, encodes to 43-char base64url.
- Password strength validator: rejects below 12 chars, missing class, accepts compliant passwords.
- Rate limiter: counts within rolling 60-minute window, resets after window slides.
- Token hashing: SHA-256 of raw token; raw token never logged.

### Integration

- `POST /v1/auth/reset-request` against real DB: registered email path issues token + email; unknown email path issues neither.
- `POST /v1/auth/reset-complete` against real DB: success path updates hash, revokes sessions, marks token used — all in a single transaction.
- `POST /v1/admin/users/{id}/force-reset` against real DB + real RBAC middleware: admin succeeds, non-admin gets 403.
- Audit log writer: each lifecycle event produces exactly one row with no token material.

### E2E Smoke

- End-to-end happy path: user submits email → receives email (captured by mailer test fixture) → extracts token → submits new password → logs in successfully with new password and old session is rejected.

---

## 12. Success Criteria

- **SC-001**: p95 latency for `POST /v1/auth/reset-request` under 250ms at 200 RPS sustained for 5 minutes.
- **SC-002**: Zero occurrences of raw token strings or password values in any log stream during a 24-hour soak test (verified by automated grep over collected logs).
- **SC-003**: Audit log row count equals issued+used+expired+forced event count (no dropped events) over the same 24-hour soak.
- **SC-004**: Email enumeration probe (1000 unknown + 1000 known emails) shows identical response status, body, and timing distribution within ±15ms p95.

---

## 13. Traceability Matrix

| FR Range | Area | Scenarios | Test surfaces |
|----------|------|-----------|---------------|
| FR-001..FR-004 | Reset Request | Registered request (Happy), Unknown email (Happy), Rate limit (Error), Prior token invalidation (Edge) | Unit: token generator, rate limiter; Integration: `POST /reset-request`; E2E smoke leg 1 |
| FR-005..FR-009 | Reset Completion | Valid completion (Happy), Expired token (Error), Weak password (Error), Token reuse (Edge) | Unit: password validator, token hashing; Integration: `POST /reset-complete`; E2E smoke leg 2 |
| FR-010..FR-011 | Admin Force-Reset | Admin force (Happy), Non-admin rejected (Error) | Integration: `POST /admin/users/{id}/force-reset` with real RBAC |
| FR-012..FR-013 | Audit & Observability | (covered transitively by Reset Request, Reset Completion, and Admin scenarios via audit assertions) | Integration: audit log writer; metrics scrape assertions |

---

## 14. Task Decomposition Guidance

1. **Reset Request slice** — covers FR-001..FR-004, FR-012 (issuance audit), DM-001. Outcome: registered users can request a reset and receive an email; unknown emails appear identical.
2. **Reset Completion slice** — covers FR-005..FR-009, FR-012 (used/expired audit). Outcome: users with a valid token can set a new password and lose all prior sessions.
3. **Admin Force-Reset slice** — covers FR-010, FR-011, FR-012 (admin audit), FR-013 (metrics). Outcome: admins can force a reset; non-admins are blocked; metrics and audit reflect both paths.
