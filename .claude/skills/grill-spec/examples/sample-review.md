# Adversarial Review: Password Reset

**Spec reviewed**: docs/plan/password-reset/password-reset-spec.md
**Review date**: 2026-05-08
**Cycle**: 1
**Verdict**: BLOCK

## Executive Summary

The password-reset spec covers the core happy path but ships with a security
hole, a non-falsifiable acceptance criterion, and one piece of out-of-scope
work that has crept in from a separate ticket — three BLOCKERs total. Eleven
WARNINGs and one INFO surface gaps in concurrency, observability, codebase
fit, and use of "v2"/"will be wired later" deferral phrases. Spec must be
revised before any task decomposition.

| Severity | Count |
|----------|-------|
| BLOCKER | 3 |
| WARNING | 11 |
| INFO | 1 |
| **Total** | **15** |

---

## Findings

### BLOCKER Findings

#### [G-001] Reset-token brute-force not mitigated

- **Dimension**: insecurity
- **Affected**: FR-004
- **Description**: FR-004 requires "System MUST generate a unique,
  time-limited reset token" but specifies no rate limiting on the token
  verification endpoint. STRIDE — elevation of privilege via brute force: a
  6-digit numeric token has 10^6 possibilities, exhaustible in minutes
  against an unrate-limited endpoint.
- **Impact**: Account takeover for any user whose reset token has been
  generated within the TTL window.
- **Fix hint**: Add FR-011: "Token verification MUST be rate-limited to 5
  attempts per token per 15-minute window. After 5 failed attempts, the
  token MUST be invalidated; the user MUST request a new one." Add a matching
  behavioural scenario under `## Behavioral Scenarios`.

---

#### [G-002] Acceptance criterion AC-03 is non-falsifiable

- **Dimension**: quality_gate
- **Affected**: FR-006 → AC-03 ("users find the reset flow intuitive")
- **Description**: Quality-gate QG-FALSIFY fires. "Users find the flow
  intuitive" is subjective with no measurable threshold. Two implementers
  cannot agree on whether the criterion is met.
- **Fix hint**: Replace with a measurable success criterion such as: "≥95%
  of users complete the reset flow without abandoning, measured over the
  first 1000 production sessions." Or remove the AC if it is aspirational
  rather than load-bearing.

---

#### [G-003] Spec smuggles in "account recovery via security questions"

- **Dimension**: scope_alignment
- **Affected**: FR-009, behavioural scenario "Recover account when email is
  inaccessible"
- **Description**: SCP-01 violation. The originating ticket
  (`AUTH-412: password reset via email`) requested email-based reset only.
  FR-009 introduces a security-question recovery path that has no entry in
  the Implementation Scope section and is not mentioned in AUTH-412.
- **Fix hint**: Either (a) delete FR-009 and its scenario, leaving the spec
  to AUTH-412's actual scope; or (b) open a separate ticket for security-
  question recovery and reference it here as out-of-scope. Do not silently
  expand scope.

---

### WARNING Findings

#### [G-004] No specification for concurrent reset requests

- **Dimension**: incompleteness
- **Affected**: User Story 1 — no concurrency scenario exists
- **Description**: The spec does not address what happens when a user
  requests a second reset before the first token expires or is consumed.
  Two valid tokens? First invalidated by second? Both valid? Each behaviour
  has different security and UX implications.
- **Fix hint**: Add FR-012 specifying that issuing a new token MUST
  invalidate any prior outstanding token for the same user, and add a
  behavioural scenario "Re-request reset within TTL window" tracing to it.

---

#### [G-005] FR uses GSD-prohibited deferral phrases

- **Dimension**: quality_gate
- **Affected**: FR-008 ("SMS-based reset will be wired later in v2")
- **Description**: QG-WEASEL fires twice in one FR — `v2` and `will be wired
  later`. Procrastination markers in functional requirements indicate the
  decision hasn't been made; FRs should be either in-scope and specified,
  or out-of-scope and not present as FRs.
- **Fix hint**: Move SMS-based reset out of FR-008. Reference it once in an
  explicit `## Out of Scope` section as: "SMS-based reset is deferred —
  tracked under AUTH-501." Delete FR-008 entirely.

---

#### [G-006] Token TTL inconsistency between FR and scenario

- **Dimension**: inconsistency
- **Affected**: FR-004 (states "15 minutes") vs. behavioural scenario
  "Reset token expires" (states "30 minutes")
- **Description**: CON-04 violation. The same value is specified two
  different ways in two adjacent sections. Implementers will pick one and
  tests will diverge.
- **Fix hint**: Fix the scenario to use 15 minutes to match FR-004, or
  change FR-004 to 30 if 30 was the intent. Pick one source of truth.

---

#### [G-007] "Robust password reset flow" — specificity failure

- **Dimension**: quality_gate
- **Affected**: Spec overview, paragraph 1 ("implement a robust password
  reset flow")
- **Description**: QG-SPEC-FAIL fires. A different Claude executing this
  spec without our conversation would need to ask: robust against what?
  Brute force? Network failures? UI errors? The word `robust` carries no
  decision.
- **Fix hint**: Replace with the specific properties the flow must have —
  "MUST tolerate SMTP delivery failure with a retry-after-30s policy", "MUST
  rate-limit token verification per FR-011", etc. Then drop "robust".

---

#### [G-008] Token uniqueness mechanism unspecified

- **Dimension**: infeasibility
- **Affected**: FR-004 ("tokens MUST be globally unique")
- **Description**: FEA-04 violation. "Globally unique" is a property
  achievable several ways (UUIDv4, secure-random + length, HMAC of
  user_id+timestamp+nonce). Each has different security and testability
  characteristics. The spec does not pick one, so the requirement is not
  reproducibly testable.
- **Fix hint**: Specify: "Tokens MUST be 32-byte secure-random values, base64-
  url encoded. Collision probability MUST be ≤ 2^-128 over the token
  lifetime." Then a test can falsify generator quality.

---

#### [G-009] No observability for reset attempts

- **Dimension**: inoperability
- **Affected**: Spec is silent on logging / alerting
- **Description**: OPS-02, OPS-04, OPS-05 violations. Reset attempts (issued,
  verified, expired, rate-limited, abused) have no specified observable
  indicator. On-call cannot tell if reset is broken or under attack.
- **Fix hint**: Add an Observability section: emit structured log entries
  with `event=reset_token_{issued|verified|expired|rate_limited}`, `user_id`
  (hashed), `request_id`, `outcome`. Alert if `rate_limited` events exceed
  N per minute.

---

#### [G-010] Spec assumes `user.email` column exists; codebase uses
`user.contact_email`

- **Dimension**: incorrectness
- **Affected**: FR-002 ("Look up user by `user.email`")
- **Description**: COR-05 violation. Reading `internal/users/model.go` shows
  the column is named `contact_email`. FR-002 will not compile against the
  actual schema; implementer will either rename the column (breaks every
  other consumer) or rewrite the FR.
- **Fix hint**: Replace `user.email` with `user.contact_email` throughout
  FR-002 and any tracing scenarios.

---

#### [G-011] Proposed `internal/auth/reset` directory contradicts codebase
layout

- **Dimension**: codebase_fit
- **Affected**: Implementation Scope, "New module: `internal/auth/reset`"
- **Description**: FIT-01 violation. The codebase nests authentication code
  under `internal/users/auth/` (see `internal/users/auth/login.go`,
  `internal/users/auth/session.go`). Introducing `internal/auth/reset`
  fragments authentication across two parallel hierarchies.
- **Fix hint**: Place new code in `internal/users/auth/reset.go`. Update
  the Implementation Scope section accordingly.

---

#### [G-012] `TokenStrategy` interface for a single implementation

- **Dimension**: overcomplexity
- **Affected**: Implementation Scope, "Define a `TokenStrategy` interface so
  alternative algorithms can be plugged in"
- **Description**: CPX-01 violation. The spec specifies exactly one token
  algorithm (SHA-256 of secure random, per FR-004). A pluggable strategy
  interface for one implementation is speculative generality. The codebase
  has no other token-strategy users.
- **Fix hint**: Replace with a concrete `GenerateResetToken(userID UUID)
  (string, error)` function in `internal/users/auth/reset.go`. Drop the
  interface and any "for testability" justification — a fake function is
  trivially substituted in tests.

---

#### [G-013] Pronoun ambiguity in FR-005

- **Dimension**: ambiguity
- **Affected**: FR-005 ("when the user submits it, the system validates it")
- **Description**: AMB-04 violation. Two `it` pronouns with unclear
  antecedents — could be the email, the token, or the new password. AC and
  scenario authors will resolve differently.
- **Fix hint**: Rewrite as "When the user submits the reset token plus a
  new password, the system MUST validate the token (per FR-004) and the
  password (per the existing password-policy module) before persisting."

---

#### [G-014] Resurrected "Behavioral Contract" heading

- **Dimension**: quality_gate
- **Affected**: Spec contains a `## Behavioral Contract` section
- **Description**: QG-RESURRECTED fires. "Behavioral Contract" was removed
  from the lean spec template in favour of `## Behavioral Scenarios`.
  Re-introducing it duplicates the scenario list and creates two sources of
  truth for the same content.
- **Fix hint**: Delete the `## Behavioral Contract` section. If any content
  there isn't covered by `## Behavioral Scenarios`, fold it in as
  scenarios; otherwise discard.

---

### INFO Findings

#### [G-015] Deferred item lacks reason

- **Dimension**: quality_gate
- **Affected**: `## Out of Scope` → "Account locking after N failed resets"
- **Description**: QG-DEFER-NO-REASON fires. The item is listed as deferred
  but no reason or follow-up ticket is given. Deferral without reason tends
  to become "lost" work.
- **Fix hint**: Add either a reason ("blocked on lockout policy decision —
  see SEC-218") or a tracking ticket reference. One sentence suffices.

---

## Structural / Narrative Integrity

| Check | Applies when | Result | Notes |
|-------|--------------|--------|-------|
| Every FR-NNN has at least one behavioural scenario | Spec contains FR-NNN IDs | FAIL | FR-008 (SMS reset, see G-005) and FR-009 (security-questions recovery, see G-003) have no scenarios; both should be deleted rather than scenarized |
| Every behavioural scenario traces back to an FR-NNN | Spec contains `## Behavioral Scenarios` | PASS | All 6 scenarios have `Traces to:` lines |
| Acceptance criteria are falsifiable and measurable | Spec contains acceptance criteria | FAIL | AC-03 is non-falsifiable (G-002) |
| Cross-references resolve (no dangling IDs) | Spec uses any ID scheme | PASS | All FR / AC / SC IDs resolve |
| Scope boundaries are explicit (in / out / deferred) | Always | FAIL | `## Out of Scope` exists but is incomplete (missing FR-008/FR-009 — see G-003, G-005) |
| Error and failure modes addressed | Always | FAIL | No SMTP failure handling, no concurrent-request handling (G-004) |
| Dependencies between requirements identified | Spec has multiple FRs | PASS | Dependency graph at `## Implementation Dependencies` is complete |
| Actors named | Always | PASS | User, MailService, AuthDB clearly identified |
| Implementation detail sufficient to begin work | Always | FAIL | Token generation mechanism unspecified (G-008) |
| Assumptions and constraints stated explicitly | Always | PASS | `## Assumptions` section is present and accurate |

---

## Alignment Summary

### Scope Alignment (Check 9)

The spec drifts beyond AUTH-412's stated scope. Five of nine FRs trace to
the originating ticket; FR-008 (SMS reset) and FR-009 (security-question
recovery) do not. FR-008's drift is masked by `v2`/`will be wired later`
language (G-005); FR-009 is unmasked scope creep (G-003). After removing
both, the spec maps cleanly to AUTH-412's stated capability list.

### Codebase Fit (Check 10)

Two fit problems. FR-002 references a column (`user.email`) that does not
exist in the actual schema (G-010). The Implementation Scope proposes a new
top-level directory (`internal/auth/reset`) that contradicts the existing
authentication layout (G-011). Token storage and email-template handling
otherwise reuse existing modules correctly.

---

## Quality Gate Summary

| Check | Triggered? | Locations |
|-------|-----------|-----------|
| QG-USER-STORY (user-story narrative) | No | — |
| QG-RESURRECTED (deleted section reappeared) | Yes | `## Behavioral Contract` (G-014) |
| QG-TEST-CATALOG (numbered test catalog >10) | No | — |
| QG-WEASEL (weasel words in FRs) | Yes | FR-008: `v2`, `will be wired later` (G-005) |
| QG-FALSIFY (non-falsifiable AC) | Yes | AC-03 "users find the flow intuitive" (G-002) |
| QG-SPEC-FAIL (specificity failure) | Yes | Overview paragraph 1: `robust` (G-007) |
| QG-DEFER-NO-REASON (deferred without reason) | Yes | "Account locking after N failed resets" (G-015) |

---

## Test Coverage Assessment

| Missing category | Affected FRs | Why it matters |
|------------------|--------------|----------------|
| Concurrency tests | FR-004, FR-012 (proposed) | Without a test, the "issuing a new token invalidates prior token" guarantee is unverified and easy to regress |
| Negative-path tests for SMTP failure | FR-003 | If MailService is down, the spec doesn't say what the user sees; no test forces the implementer to decide |
| Rate-limit boundary tests | FR-011 (proposed) | The 5-attempt threshold needs tests at 4, 5, and 6 attempts to confirm correct off-by-one behaviour |

---

## Unasked Questions

1. What does the user see if the SMTP relay rejects the reset email? Retry?
   Generic success message anyway? (relates to G-009)
2. Are reset tokens stored hashed at rest, or plaintext? FR-004 is silent.
3. Is the "from" address for the reset email a fixed transactional address,
   or per-tenant? (codebase has multi-tenant email config — see
   `internal/users/email/sender.go`)
4. Does the reset flow invalidate active sessions for that user, or only
   issue a new password? Either is defensible; the spec doesn't choose.

---

## Verdict Rationale

Three BLOCKERs make this BLOCK rather than REVISE: a security hole (G-001),
a non-falsifiable AC (G-002), and out-of-scope work that fundamentally
changes what the spec is for (G-003). The eleven WARNINGs are concentrated
in incompleteness, codebase fit, and quality-gate territory and should all
be addressed in the next revision cycle. This is cycle 1 of at most 3.

### Recommended Next Actions

- [ ] Address G-001 (rate-limit FR), G-002 (replace AC-03), G-003 (delete
      FR-009) before resubmitting
- [ ] Resolve scope by deleting FR-008 / FR-009 and adjusting `## Out of
      Scope` (G-003, G-005)
- [ ] Run `/plan-spec --revise docs/plan/password-reset/password-reset-spec.md
      docs/plan/password-reset/password-reset-spec-review.md` to consume the
      YAML issues block below

---

## Issues

```yaml
issues:
  - id: G-001
    dimension: insecurity
    severity: blocker
    affected: "FR-004"
    description: |
      FR-004 requires unique time-limited tokens but specifies no rate
      limiting on the verification endpoint. STRIDE elevation-of-privilege
      via brute force: a 6-digit numeric token is exhaustible in minutes.
    fix_hint: |
      Add FR-011: "Token verification MUST be rate-limited to 5 attempts
      per token per 15-minute window. After 5 failed attempts, the token
      MUST be invalidated." Add a matching behavioural scenario.

  - id: G-002
    dimension: quality_gate
    severity: blocker
    affected: "AC-03"
    description: |
      QG-FALSIFY fires. AC-03 ("users find the reset flow intuitive") is
      subjective with no measurable threshold; two implementers cannot
      agree on whether it is met.
    fix_hint: |
      Replace with a measurable criterion such as "≥95% of users complete
      the flow without abandoning over the first 1000 production sessions."

  - id: G-003
    dimension: scope_alignment
    severity: blocker
    affected: "FR-009"
    description: |
      SCP-01 violation. AUTH-412 requested email-based reset only. FR-009
      adds a security-question recovery path that is not in AUTH-412 and
      not listed in Implementation Scope.
    fix_hint: |
      Delete FR-009 and its scenario. If the capability is wanted, open a
      separate ticket and reference it under `## Out of Scope`.

  - id: G-004
    dimension: incompleteness
    severity: warning
    affected: "User Story 1"
    description: |
      No scenario covers a second reset request issued before the first
      token expires. Behaviour is undefined: two valid tokens? first
      invalidated? both valid?
    fix_hint: |
      Add FR-012: "Issuing a new token MUST invalidate any prior
      outstanding token for the same user." Add scenario "Re-request reset
      within TTL window" tracing to FR-012.

  - id: G-005
    dimension: quality_gate
    severity: warning
    affected: "FR-008"
    description: |
      QG-WEASEL fires twice in one FR — `v2` and `will be wired later`.
      Procrastination markers in functional requirements indicate the
      decision hasn't been made.
    fix_hint: |
      Delete FR-008. Reference SMS-based reset once under `## Out of
      Scope`: "SMS-based reset deferred — tracked under AUTH-501."

  - id: G-006
    dimension: inconsistency
    severity: warning
    affected: "FR-004 vs. scenario 'Reset token expires'"
    description: |
      CON-04 violation. FR-004 specifies 15-minute TTL; scenario says 30
      minutes. Implementers will pick one and tests will diverge.
    fix_hint: |
      Pick one source of truth. Use 15 minutes in both places, or change
      FR-004 to 30 if that was the intent.

  - id: G-007
    dimension: quality_gate
    severity: warning
    affected: "Spec overview, paragraph 1"
    description: |
      QG-SPEC-FAIL fires on `robust`. A different Claude would have to
      ask: robust against what? The word carries no decision.
    fix_hint: |
      Replace with the specific properties the flow must have — SMTP
      retry policy, rate limiting per FR-011, etc. — then drop "robust".

  - id: G-008
    dimension: infeasibility
    severity: warning
    affected: "FR-004"
    description: |
      FEA-04 violation. "Globally unique" is achievable several ways with
      different security characteristics. The spec does not pick one, so
      the requirement is not reproducibly testable.
    fix_hint: |
      Specify: "Tokens MUST be 32-byte secure-random values, base64-url
      encoded. Collision probability MUST be ≤ 2^-128 over token lifetime."

  - id: G-009
    dimension: inoperability
    severity: warning
    affected: "Spec is silent on observability"
    description: |
      OPS-02 / OPS-04 / OPS-05 violations. Reset attempts have no
      specified log events, alerts, or correlation IDs. On-call cannot
      tell if reset is broken or under attack.
    fix_hint: |
      Add Observability section: emit structured logs with
      `event=reset_token_{issued|verified|expired|rate_limited}`,
      `user_id` (hashed), `request_id`. Alert if `rate_limited` exceeds
      N/min.

  - id: G-010
    dimension: incorrectness
    severity: warning
    affected: "FR-002"
    description: |
      COR-05 violation. FR-002 references `user.email` but the actual
      column in `internal/users/model.go` is `user.contact_email`.
    fix_hint: |
      Replace `user.email` with `user.contact_email` in FR-002 and tracing
      scenarios.

  - id: G-011
    dimension: codebase_fit
    severity: warning
    affected: "Implementation Scope: 'internal/auth/reset'"
    description: |
      FIT-01 violation. The codebase nests authentication under
      `internal/users/auth/`. A new top-level `internal/auth/reset`
      fragments authentication across two hierarchies.
    fix_hint: |
      Place new code at `internal/users/auth/reset.go`. Update
      Implementation Scope accordingly.

  - id: G-012
    dimension: overcomplexity
    severity: warning
    affected: "Implementation Scope: TokenStrategy interface"
    description: |
      CPX-01 violation. The spec specifies exactly one token algorithm.
      A pluggable strategy interface for one implementation is
      speculative generality.
    fix_hint: |
      Replace with a concrete `GenerateResetToken(userID UUID) (string,
      error)` function. Drop the interface.

  - id: G-013
    dimension: ambiguity
    severity: warning
    affected: "FR-005"
    description: |
      AMB-04 violation. Two `it` pronouns with unclear antecedents — could
      be the email, the token, or the new password.
    fix_hint: |
      Rewrite as "When the user submits the reset token plus a new
      password, the system MUST validate the token (per FR-004) and the
      password (per the password-policy module) before persisting."

  - id: G-014
    dimension: quality_gate
    severity: warning
    affected: "## Behavioral Contract"
    description: |
      QG-RESURRECTED fires. "Behavioral Contract" was removed from the
      lean template in favour of `## Behavioral Scenarios`. Re-introducing
      it creates two sources of truth.
    fix_hint: |
      Delete the section. Fold any unique content into
      `## Behavioral Scenarios` as scenarios; discard the rest.

  - id: G-015
    dimension: quality_gate
    severity: info
    affected: "## Out of Scope: 'Account locking after N failed resets'"
    description: |
      QG-DEFER-NO-REASON fires. The item is deferred without a reason or
      follow-up reference; deferred-without-reason work tends to get lost.
    fix_hint: |
      Add a reason ("blocked on lockout policy — SEC-218") or a tracking
      ticket reference. One sentence is enough.
```
