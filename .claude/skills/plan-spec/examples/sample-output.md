# Feature Specification: Password Reset

**Created**: 2026-02-03
**Status**: Draft
**Input**: Users need to reset their password when they forget it. The current system has no self-service reset — users must contact support.

---

## User Stories & Acceptance Criteria

### User Story 1 — Request Password Reset (Priority: P1)

A registered user who has forgotten their password visits the login page and requests a reset link. The system sends a time-limited reset email to their registered address. This eliminates the most common support ticket category (40% of all tickets are password resets) and unblocks users within minutes instead of hours.

**Why this priority**: Blocking issue — users cannot access the system at all without this. Highest support cost driver.

**Independent Test**: Can be verified by requesting a reset for a known email and confirming the email arrives with a valid token. Delivers value even before the "complete reset" flow is built (user sees the email, confirming the system recognised them).

**Acceptance Scenarios**:

1. **Given** a registered user with email "alice@example.com", **When** they submit the reset form with "alice@example.com", **Then** the system sends a reset email containing a unique link that expires in 1 hour.
2. **Given** a non-existent email "unknown@example.com", **When** they submit the reset form, **Then** the system displays the same "check your email" message (to prevent email enumeration) and sends no email.
3. **Given** a registered user, **When** they request a reset and a previous unexpired token exists, **Then** the previous token is invalidated and a new one is issued.
4. **Given** a registered user, **When** they request more than 5 resets in 1 hour, **Then** the system returns a rate limit error and sends no email.

---

### User Story 2 — Complete Password Reset (Priority: P1)

A user who received a reset email clicks the link and sets a new password. The system validates the token, enforces password strength rules, updates the password, invalidates all existing sessions, and confirms success. This completes the self-service reset flow.

**Why this priority**: Without this, the reset email from US-1 is useless. These two stories form the minimum viable reset flow.

**Independent Test**: Can be verified by using a valid token to set a new password and then logging in with the new password. Requires US-1 to generate the token.

**Acceptance Scenarios**:

1. **Given** a valid, unexpired reset token, **When** the user submits a new password meeting strength requirements, **Then** the password is updated, all existing sessions are invalidated, and a confirmation page is shown.
2. **Given** an expired reset token (older than 1 hour), **When** the user submits a new password, **Then** the system displays "This link has expired. Please request a new reset."
3. **Given** a valid token, **When** the user submits a password shorter than 8 characters, **Then** the system displays a validation error and does not change the password.
4. **Given** a valid token that has already been used, **When** someone attempts to use it again, **Then** the system rejects it with "This link has already been used."

---

### User Story 3 — Admin Force-Reset (Priority: P3)

An administrator forces a password reset for a specific user account. The system invalidates the user's current password, sends them a reset email, and logs the admin action. This supports security incident response (compromised accounts) and compliance requirements.

**Why this priority**: Important for security but not a launch blocker. Manual account suspension is the interim workaround.

**Independent Test**: Can be verified by an admin triggering force-reset for a test account and confirming the user can no longer log in with their old password and receives a reset email.

**Acceptance Scenarios**:

1. **Given** an admin user, **When** they trigger force-reset for user "bob@example.com", **Then** Bob's password is invalidated, a reset email is sent, all Bob's sessions are terminated, and an audit log entry is created.
2. **Given** a non-admin user, **When** they attempt to trigger force-reset, **Then** the system returns a 403 Forbidden error.
3. **Given** an admin user, **When** they trigger force-reset for a non-existent account, **Then** the system returns a 404 error.

---

## Edge Cases

- What happens when the email service is unavailable during reset request? System queues the email for retry and still shows "check your email" to the user. If retry fails after 3 attempts, an alert is raised to ops.
- What happens when a user changes their email address while a reset token is outstanding? The token remains tied to the original email. Reset completes normally since the token references the user ID, not the email.
- What happens when the reset link is opened on a different device/browser? It works — the token is not bound to a session or device.
- What happens when the password hash algorithm is upgraded between request and completion? The new password is hashed with the current algorithm. No issue.
- What happens when two reset requests arrive simultaneously? Both requests generate tokens. The second invalidates the first (last-write-wins). Only the most recent token is valid.

---

## BDD Scenarios

### Feature: Password Reset

#### Scenario: Registered user requests password reset

**Traces to**: User Story 1, Acceptance Scenario 1
**Category**: Happy Path

- **Given** a registered user with email "alice@example.com"
- **When** they submit the password reset form with "alice@example.com"
- **Then** the system sends an email to "alice@example.com"
- **And** the email contains a reset link with a unique token
- **And** the token expires 1 hour from the time of request
- **And** the page displays "If an account exists for that email, we have sent a reset link."

---

#### Scenario: Unregistered email receives no email but same message

**Traces to**: User Story 1, Acceptance Scenario 2
**Category**: Error Path

- **Given** no account exists for "unknown@example.com"
- **When** someone submits the password reset form with "unknown@example.com"
- **Then** no email is sent
- **And** the page displays "If an account exists for that email, we have sent a reset link."

---

#### Scenario: New reset request invalidates previous token

**Traces to**: User Story 1, Acceptance Scenario 3
**Category**: Alternate Path

- **Given** a registered user with an existing unexpired reset token "token-old"
- **When** they request a new password reset
- **Then** "token-old" is invalidated
- **And** a new token "token-new" is generated and emailed

---

#### Scenario: Rate limiting after excessive reset requests

**Traces to**: User Story 1, Acceptance Scenario 4
**Category**: Error Path

- **Given** a registered user who has requested 5 resets in the last hour
- **When** they request a 6th reset
- **Then** the system returns a rate limit error
- **And** no email is sent
- **And** the previous valid token remains active

---

#### Scenario: User completes password reset with valid token

**Traces to**: User Story 2, Acceptance Scenario 1
**Category**: Happy Path

- **Given** a valid, unexpired reset token for user "alice@example.com"
- **When** the user submits a new password "N3wSecur3P@ss!" via the reset form
- **Then** the password for "alice@example.com" is updated
- **And** all existing sessions for "alice@example.com" are invalidated
- **And** a confirmation page is displayed

---

#### Scenario: Expired token is rejected

**Traces to**: User Story 2, Acceptance Scenario 2
**Category**: Error Path

- **Given** a reset token that was generated 61 minutes ago
- **When** the user submits a new password
- **Then** the system displays "This link has expired. Please request a new reset."
- **And** the password is not changed

---

#### Scenario Outline: Password strength validation on reset

**Traces to**: User Story 2, Acceptance Scenario 3
**Category**: Error Path

- **Given** a valid, unexpired reset token
- **When** the user submits password `<password>`
- **Then** the system displays error `<error_message>`
- **And** the password is not changed

**Examples**:

| password | error_message |
|----------|--------------|
| `"short"` | "Password must be at least 8 characters." |
| `"abcdefgh"` | "Password must contain at least one number." |
| `"12345678"` | "Password must contain at least one letter." |

---

#### Scenario: Already-used token is rejected

**Traces to**: User Story 2, Acceptance Scenario 4
**Category**: Error Path

- **Given** a reset token that has already been used to change a password
- **When** someone submits a new password with that token
- **Then** the system displays "This link has already been used."
- **And** no password change occurs

---

#### Scenario: Admin force-resets a user account

**Traces to**: User Story 3, Acceptance Scenario 1
**Category**: Happy Path

- **Given** an authenticated admin user
- **And** a registered user "bob@example.com"
- **When** the admin triggers force-reset for "bob@example.com"
- **Then** Bob's current password is invalidated
- **And** all of Bob's active sessions are terminated
- **And** a reset email is sent to "bob@example.com"
- **And** an audit log entry records the admin's ID, Bob's ID, and a timestamp

---

#### Scenario: Non-admin cannot force-reset

**Traces to**: User Story 3, Acceptance Scenario 2
**Category**: Error Path

- **Given** an authenticated non-admin user
- **When** they attempt to trigger force-reset for "bob@example.com"
- **Then** the system returns a 403 Forbidden error
- **And** no changes are made to Bob's account

---

#### Scenario: Simultaneous reset requests — last write wins

**Traces to**: User Story 1, Edge Case (concurrent requests)
**Category**: Edge Case

- **Given** a registered user "alice@example.com"
- **When** two reset requests arrive simultaneously
- **Then** only one token is valid (the last one written)
- **And** the earlier token is invalidated

---

## Test-Driven Development Plan

### Test Hierarchy

| Level       | Scope                                    | Purpose                                            |
|-------------|------------------------------------------|----------------------------------------------------|
| Unit        | Token generation, validation, password strength | Validates core logic in isolation               |
| Integration | Email service, database, session store    | Validates components interact correctly             |
| E2E         | Full reset flow: request -> email -> complete | Validates the user-facing workflow end to end   |

### Test Implementation Order

| Order | Test Name | Level | Traces to BDD Scenario | Description |
|-------|-----------|-------|------------------------|-------------|
| 1 | test_generate_reset_token_format | Unit | Registered user requests reset | Token is a valid UUID, not guessable |
| 2 | test_token_expiry_set_to_one_hour | Unit | Registered user requests reset | Token expiry is exactly 60 minutes from creation |
| 3 | test_password_strength_minimum_length | Unit | Password strength validation | Rejects passwords < 8 chars |
| 4 | test_password_strength_requires_number | Unit | Password strength validation | Rejects all-letter passwords |
| 5 | test_password_strength_requires_letter | Unit | Password strength validation | Rejects all-number passwords |
| 6 | test_previous_token_invalidated_on_new_request | Unit | New request invalidates previous token | Old token marked invalid when new one created |
| 7 | test_rate_limit_blocks_after_five_requests | Unit | Rate limiting after excessive requests | 6th request in 1 hour is rejected |
| 8 | test_used_token_cannot_be_reused | Unit | Already-used token is rejected | Token marked used after password change |
| 9 | test_expired_token_rejected | Unit | Expired token is rejected | Token older than 60 min returns error |
| 10 | test_reset_email_sent_for_registered_user | Integration | Registered user requests reset | Email service called with correct recipient and token link |
| 11 | test_no_email_sent_for_unregistered_address | Integration | Unregistered email receives no email | Email service NOT called |
| 12 | test_password_updated_in_database | Integration | User completes reset with valid token | Password hash changes in user record |
| 13 | test_sessions_invalidated_on_reset | Integration | User completes reset with valid token | Session store clears all sessions for user |
| 14 | test_admin_force_reset_creates_audit_log | Integration | Admin force-resets a user account | Audit log entry written with correct fields |
| 15 | test_non_admin_cannot_force_reset | Integration | Non-admin cannot force-reset | 403 returned, no side effects |
| 16 | test_full_reset_flow_request_to_completion | E2E | Registered user requests + completes reset | Request reset, receive email, click link, set password, login with new password |
| 17 | test_full_reset_flow_expired_token | E2E | Expired token is rejected | Request reset, wait > 1 hour, attempt completion, see expiry message |

### Test Datasets

#### Dataset: Email Address Inputs (Reset Request Form)

| # | Input | Boundary Type | Expected Output | Traces to | Notes |
|---|-------|---------------|-----------------|-----------|-------|
| 1 | `""` | Empty | Validation error: email required | BDD: Registered user requests reset | Empty form submission |
| 2 | `"alice@example.com"` | Valid (registered) | Success message, email sent | BDD: Registered user requests reset | Happy path |
| 3 | `"unknown@example.com"` | Valid (not registered) | Success message, no email | BDD: Unregistered email | Prevents enumeration |
| 4 | `"not-an-email"` | Invalid format | Validation error: invalid email | BDD: Registered user requests reset | Client-side catch |
| 5 | `"alice@example.com "` | Trailing whitespace | Trimmed, treated as valid | BDD: Registered user requests reset | Whitespace handling |
| 6 | `"ALICE@EXAMPLE.COM"` | Case variation | Matches alice@example.com | BDD: Registered user requests reset | Case-insensitive lookup |
| 7 | `"a" * 255 + "@example.com"` | Max length | Validation error: too long | BDD: Registered user requests reset | 254 char RFC limit |
| 8 | `"alice+tag@example.com"` | Plus addressing | Valid, processed normally | BDD: Registered user requests reset | Subaddressing |
| 9 | `"<script>alert(1)</script>"` | XSS attempt | Validation error: invalid email | BDD: Registered user requests reset | Injection prevention |

#### Dataset: Password Inputs (Reset Completion Form)

| # | Input | Boundary Type | Expected Output | Traces to | Notes |
|---|-------|---------------|-----------------|-----------|-------|
| 1 | `""` | Empty | Error: password required | BDD: Password strength validation | Empty submission |
| 2 | `"Ab1"` | Min - 1 (3 chars) | Error: at least 8 characters | BDD: Password strength validation | Below minimum |
| 3 | `"Abcdef1!"` | Min (8 chars) | Accepted | BDD: User completes reset | Exact minimum |
| 4 | `"abcdefgh"` | No number | Error: must contain number | BDD: Password strength validation | Letters only |
| 5 | `"12345678"` | No letter | Error: must contain letter | BDD: Password strength validation | Numbers only |
| 6 | `"Ab1" * 43` | Max (128 chars) | Accepted | BDD: User completes reset | At upper limit |
| 7 | `"Ab1" * 44` | Max + 1 (129 chars) | Error: too long | BDD: Password strength validation | Over limit |
| 8 | `"P@ssw0rd"` | Common password | Warning or rejection (policy dependent) | BDD: Password strength validation | Dictionary check |
| 9 | `"Contraseña1"` | Unicode | Accepted | BDD: User completes reset | Non-ASCII letters |
| 10 | `"Pass word1"` | Contains spaces | Accepted | BDD: User completes reset | Spaces are valid characters |

#### Dataset: Reset Token Inputs

| # | Input | Boundary Type | Expected Output | Traces to | Notes |
|---|-------|---------------|-----------------|-----------|-------|
| 1 | Valid token, 0 minutes old | Fresh | Accepted | BDD: User completes reset | Immediate use |
| 2 | Valid token, 59 minutes old | Near expiry | Accepted | BDD: User completes reset | Just under limit |
| 3 | Valid token, 60 minutes old | At expiry | Rejected: expired | BDD: Expired token rejected | Exact boundary |
| 4 | Valid token, 61 minutes old | Past expiry | Rejected: expired | BDD: Expired token rejected | Just over limit |
| 5 | Already-used token | Used | Rejected: already used | BDD: Already-used token rejected | Replay prevention |
| 6 | `""` | Empty | Rejected: invalid token | BDD: Expired token rejected | No token provided |
| 7 | `"not-a-real-token"` | Non-existent | Rejected: invalid token | BDD: Expired token rejected | Random string |
| 8 | Token with tampered characters | Corrupted | Rejected: invalid token | BDD: Expired token rejected | Modified UUID |

### Regression Test Requirements

> No regression impact — new capability. The password reset feature is entirely new. Integration seams protected by:
> - Existing login tests (confirm login still works with unchanged passwords)
> - Existing session management tests (confirm normal session lifecycle unchanged)
> - Existing email service tests (confirm other email types still send correctly)

---

## Functional Requirements

- **FR-001**: System MUST send a password reset email when a registered user submits the reset form.
- **FR-002**: System MUST display an identical response for registered and unregistered emails to prevent email enumeration.
- **FR-003**: System MUST generate reset tokens that expire after 1 hour.
- **FR-004**: System MUST invalidate all previous reset tokens when a new one is requested for the same user.
- **FR-005**: System MUST enforce rate limiting of no more than 5 reset requests per user per hour.
- **FR-006**: System MUST enforce password strength rules: minimum 8 characters, at least one letter, at least one number.
- **FR-007**: System MUST invalidate all active sessions for a user when their password is changed via reset.
- **FR-008**: System MUST prevent reuse of a reset token after it has been used.
- **FR-009**: System MUST restrict force-reset to admin users and return 403 for non-admin attempts.
- **FR-010**: System MUST create an audit log entry for every admin force-reset action.

---

## Success Criteria

- **SC-001**: Reset email is delivered within 30 seconds of form submission for 99% of requests.
- **SC-002**: 100% of expired tokens are rejected — zero successful resets with tokens older than 60 minutes.
- **SC-003**: 100% of used tokens are rejected on second use — zero token replay.
- **SC-004**: Response to the reset form is identical (content and timing) for registered and unregistered emails.
- **SC-005**: Admin force-reset produces an audit log entry in 100% of cases.
- **SC-006**: All existing login and session tests pass without modification after the feature is deployed.

---

## Traceability Matrix

| Requirement | User Story | BDD Scenario(s) | Test Name(s) |
|-------------|-----------|------------------|---------------|
| FR-001 | US-1 | Registered user requests reset | test_reset_email_sent_for_registered_user, test_full_reset_flow |
| FR-002 | US-1 | Unregistered email receives no email | test_no_email_sent_for_unregistered_address |
| FR-003 | US-1, US-2 | Registered user requests reset, Expired token rejected | test_token_expiry_set_to_one_hour, test_expired_token_rejected |
| FR-004 | US-1 | New request invalidates previous token | test_previous_token_invalidated_on_new_request |
| FR-005 | US-1 | Rate limiting after excessive requests | test_rate_limit_blocks_after_five_requests |
| FR-006 | US-2 | Password strength validation (outline) | test_password_strength_minimum_length, _requires_number, _requires_letter |
| FR-007 | US-2 | User completes reset with valid token | test_sessions_invalidated_on_reset |
| FR-008 | US-2 | Already-used token is rejected | test_used_token_cannot_be_reused |
| FR-009 | US-3 | Non-admin cannot force-reset | test_non_admin_cannot_force_reset |
| FR-010 | US-3 | Admin force-resets a user account | test_admin_force_reset_creates_audit_log |

---

## Assumptions

- The application already has a user registration and login system.
- An email delivery service (e.g., SES, SendGrid) is configured and operational.
- Session management supports selective invalidation by user ID.
- An audit log table or service exists for recording admin actions.
- Password hashing uses bcrypt or argon2 (not specified here — existing convention applies).

## Clarifications

### 2026-02-03

- Q: Should the reset token be a UUID or a signed JWT? -> A: UUID stored in the database. Simpler, no secret rotation issues, easy to invalidate.
- Q: Should we support "magic link" login (passwordless) as part of this? -> A: Out of scope. This spec covers password reset only. Magic links can be a separate feature.
- Q: What happens if the email service is down? -> A: Queue for retry (up to 3 attempts). Show the same "check your email" message to the user. Raise an ops alert if all retries fail.
- Q: Should password history be checked (prevent reusing last N passwords)? -> A: Not in this version. Can be added as a future enhancement.
