# Adversarial Review Constitution

These principles govern the adversarial review process. The reviewer MUST
evaluate the spec against every applicable principle. When a principle is
violated, it becomes a finding.

## Core Axioms

1. **The spec is wrong until proven right.** Do not extend the benefit of
   the doubt. If something is unclear, it is a defect.
2. **Silence is a bug.** If the spec does not address a concern, that concern
   is unaddressed — not implicitly handled.
3. **Every requirement must be testable.** If you cannot write a test for a
   requirement, the requirement is defective.
4. **Every test must trace to a requirement.** Orphan tests indicate scope
   creep or missing requirements.
5. **Failure is the default.** Assume every external call fails, every input
   is malformed, every user is confused, and every attacker is motivated.

## Lens 1 Principles: Ambiguity

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| AMB-01 | Every domain term must be defined exactly once | Using "workload", "service", and "application" interchangeably |
| AMB-02 | Requirements must use RFC 2119 language (MUST/SHOULD/MAY) | "The system will try to..." or "The system handles..." |
| AMB-03 | Numeric thresholds must have explicit units and bounds | "Response time should be fast" without ms/s and percentile |
| AMB-04 | Conditional logic must cover all branches | "If the user is authenticated, show the dashboard" (what about unauthenticated?) |
| AMB-05 | Error messages must specify exact content or format | "Display an appropriate error message" |
| AMB-06 | Time references must be absolute or relative with a defined anchor | "Recently created", "old records", "stale data" |
| AMB-07 | Quantities must be explicit | "Multiple retries", "a few seconds", "several items" |

## Lens 2 Principles: Incompleteness

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| INC-01 | Every external dependency must have a failure mode scenario | Assuming the database/API/queue is always available |
| INC-02 | Every user input must have validation rules specified | Accepting user input without defining valid range/format |
| INC-03 | Every state machine must show all transitions, including error states | Happy path only state diagrams |
| INC-04 | Data lifecycle must be complete: create, read, update, delete, archive | Specifying creation but not cleanup/deletion |
| INC-05 | Concurrency model must be specified for shared resources | Assuming single-threaded access to shared state |
| INC-06 | Idempotency requirements must be stated for retryable operations | POST/PUT without duplicate detection |
| INC-07 | Timeout values must be specified for every blocking operation | "Wait for response" without timeout or fallback |
| INC-08 | Pagination must be specified for any list/query operation | Returning unbounded result sets |
| INC-09 | Rate limiting must be specified for any public-facing endpoint | No throttling on API endpoints |
| INC-10 | Migration strategy must be specified for schema/data changes | Adding new fields without specifying existing data handling |

## Lens 3 Principles: Inconsistency

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| CON-01 | The same concept must use the same name everywhere | "user ID" in stories, "userId" in BDD, "user_id" in datasets |
| CON-02 | Traceability must be bidirectional with no orphans | Requirements without scenarios, scenarios without tests |
| CON-03 | Priority ordering must be consistent across dependencies | P0 feature depending on P3 prerequisite |
| CON-04 | Data types must be consistent across all references | String in one place, integer in another for the same field |
| CON-05 | Error codes/messages must be consistent across scenarios | Different error messages for the same failure condition |
| CON-06 | Acceptance criteria must not contradict each other | "MUST allow special characters" and "input MUST be alphanumeric" |

## Lens 4 Principles: Infeasibility

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| FEA-01 | Requirements must be achievable with the stated tech stack | Requiring real-time guarantees on eventually-consistent systems |
| FEA-02 | Performance targets must be realistic for the architecture | Sub-millisecond response with multiple service hops |
| FEA-03 | Test scenarios must be reproducible in CI/CD | Tests requiring manual setup, specific network conditions, or time-of-day |
| FEA-04 | Success criteria must be measurable with available tooling | Metrics requiring instrumentation that doesn't exist |
| FEA-05 | Ordering guarantees must be achievable in distributed systems | Assuming global ordering without coordination mechanism |

## Lens 5 Principles: Insecurity (STRIDE)

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| SEC-01 | Every entry point must specify authentication mechanism | Endpoints without auth requirements |
| SEC-02 | Every operation must specify authorization rules | "Authenticated users can..." without role/permission checks |
| SEC-03 | Every sensitive operation must produce an audit log entry | State changes without audit trail |
| SEC-04 | Error responses must not leak internal details | Stack traces, internal IPs, or database schemas in error messages |
| SEC-05 | All inputs must be validated at the system boundary | Trusting data from external sources |
| SEC-06 | Secrets must never appear in logs, URLs, or error messages | API keys in query parameters, tokens in log output |
| SEC-07 | Data at rest and in transit must specify encryption requirements | Storing PII without encryption specification |
| SEC-08 | Session/token management must specify expiry and revocation | Tokens without TTL or invalidation mechanism |
| SEC-09 | Resource limits must be specified to prevent exhaustion | Unbounded file uploads, unbounded query results, no connection limits |

## Lens 6 Principles: Inoperability

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| OPS-01 | Every new component must specify health check endpoints | Services without liveness/readiness probes |
| OPS-02 | Every failure mode must specify an observable indicator | Failures that are silent or only visible in logs |
| OPS-03 | Rollback procedure must be specified or feature-flagged | Big-bang deployments with no rollback plan |
| OPS-04 | Structured logging must include correlation IDs | Log messages without traceId, requestId, or workload identifiers |
| OPS-05 | Alerting thresholds must be specified for key metrics | Monitoring without actionable alerts |
| OPS-06 | Graceful degradation behaviour must be specified | Feature fails completely instead of degrading |
| OPS-07 | Configuration must be externalized, not hardcoded | Magic numbers, embedded URLs, inline credentials |
| OPS-08 | Startup and shutdown behaviour must be specified | No graceful shutdown, no dependency readiness checks |

## Lens 7 Principles: Incorrectness

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| COR-01 | Business rules must match source-of-truth documentation | Spec contradicts Jira ticket, Confluence doc, or existing code |
| COR-02 | Boundary values in test data must be mathematically correct | Off-by-one errors in min/max calculations |
| COR-03 | Given preconditions must be achievable from a clean state | BDD scenarios assuming state that no scenario creates |
| COR-04 | Time zone and locale assumptions must be explicit | Assuming UTC, assuming English, assuming Gregorian calendar |
| COR-05 | Existing code behaviour assumed by the spec must be verified | Spec assumes an API returns X but it actually returns Y |
| COR-06 | Race conditions between concurrent operations must be identified | Two users modifying the same resource simultaneously |

## Lens 8 Principles: Overcomplexity

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| CPX-01 | Every abstraction must have at least two concrete implementations or a stated reason to exist | Interface with one implementation "for testability" when a concrete type and simple test double would suffice |
| CPX-02 | Configuration options must correspond to values that will realistically change | Externalizing a retry count that has been 3 for five years and nobody has ever changed |
| CPX-03 | The number of architectural layers must be justified by the problem's complexity | Request → Controller → Service → Repository → DAO → Database for a single-table CRUD operation |
| CPX-04 | Requirements must solve the current problem, not hypothetical future ones | "MAY support pluggable storage backends" when the only backend is Azure Blob Storage |
| CPX-05 | Error handling complexity must match error likelihood and impact | Circuit breakers and exponential backoff for an internal synchronous call that fails once a year |
| CPX-06 | Test infrastructure must not exceed the complexity of the code under test | Test factories, builders, and fixtures more complex than the production code they test |
| CPX-07 | The simplest solution that satisfies all stated requirements is the correct one | Introducing an event-driven architecture when a direct function call achieves the same result |
| CPX-08 | Feature flags, toggles, and gradual rollout mechanisms must justify their maintenance cost | Feature flag for a feature that will never be toggled off after initial release |
| CPX-09 | New concepts (types, services, tables, queues) must each solve a distinct stated problem | Creating a dedicated microservice for logic that belongs in an existing module |
| CPX-10 | Performance optimizations must target measured bottlenecks, not theoretical ones | Adding caching, connection pooling, or async processing without evidence of a performance problem |

## Tier 2 — Alignment Principles

The eight lenses above evaluate the spec on its own terms. Tier 2 asks whether
the spec is the *right* spec for this user and this codebase. Two checks.

### Check 9 Principles: Scope Alignment

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| SCP-01 | Every FR must trace to a stated capability in the spec's Implementation Scope section | Functional requirements introduced for capabilities that are not listed under in-scope deliverables |
| SCP-02 | Deferred items must not re-enter the spec through back doors | An item marked "deferred to v2" reappears as an FR, an acceptance criterion, or a behavioural scenario in the current spec |
| SCP-03 | Guard rails must constrain scope, not expand it | "Out of scope" / "non-goals" / "constraint" sections that introduce new requirements instead of bounding existing ones |

**Test for scope drift**: For every FR, name the line in Implementation Scope
that the FR realises. If you cannot, the FR is a scope-drift finding. For
every "deferred"/"v2"/"future" item, confirm it is referenced ONLY as
out-of-scope and not load-bearing in any acceptance criterion.

### Check 10 Principles: Codebase Fit

| ID | Principle | Anti-Pattern |
|----|-----------|-------------|
| FIT-01 | The spec must reference actual existing files, modules, and symbols — not hypothetical ones | "We'll use the AuthService" when no AuthService exists; "in `internal/auth/middleware.go`" when that path doesn't exist |
| FIT-02 | Proposed patterns must align with the project's existing conventions | Spec proposes a global singleton when the codebase uses dependency injection everywhere else |
| FIT-03 | Data types and naming in the spec must match existing codebase style | Spec uses `userID string` when the codebase consistently uses `UserID uuid.UUID`; spec uses `camelCase` when the codebase is `snake_case` |

**Test for codebase fit**: For every concrete reference (file path, type
name, function name) and every architectural decision, locate the
corresponding artefact in the codebase. If it does not exist, or contradicts
established convention, that is a codebase-fit finding.

## Tier 3 — Anti-Bloat Definitions

Tier 3 is a set of mechanical checks that produce findings independent of the
lenses. Each check has a precise definition so the reviewer applies it
consistently across runs.

### Weasel-Word Prohibited List

Functional requirements MUST NOT contain weasel words. Two categories — they
catch different real defects, and both produce WARNING findings
(`dimension: quality_gate`, classified as `weasel_word`). Each occurrence is
one finding.

#### Category A: Quality-attribute weasels (untestable thresholds)

Words that imply a quality without specifying a measurable threshold. The
defect is *un­testability* — two engineers cannot agree whether the
requirement is met.

- `fast`, `quick`, `responsive`, `snappy`, `performant`
- `secure`, `safe`, `hardened`
- `robust`, `resilient`, `reliable`
- `scalable`, `efficient`, `optimized`
- `user-friendly`, `intuitive`, `seamless`, `clean`
- `appropriate`, `reasonable`, `sensible`, `proper`
- `flexible`, `extensible`, `modular`
- `simple`, `easy`, `straightforward`
- `etc.`, `and so on`, `as appropriate`, `as needed`, `where applicable`

A Category A word is acceptable ONLY if the same requirement contains a
quantitative threshold defining what the word means. Example: "responses
MUST be fast (P95 < 200ms at 100 RPS)" is acceptable; "responses MUST be
fast" is a finding.

#### Category B: Procrastination markers (deferred decisions)

Phrases that defer real specification decisions to "later" without naming
when, where, or by whom. The defect is *incompleteness disguised as scope
management* — the spec author hasn't decided.

- `v1`, `v2`, `v3`, `vN` — version-deferred work
- `simplified version`, `basic version`, `minimal implementation`
- `static for now`, `hardcoded for now`
- `placeholder` — an admission that the real value isn't decided
- `future enhancement`, `dynamic in future phase`
- `will be wired later`, `skip for now`
- `to be determined`, `TBD` (in functional requirement bodies)

A Category B phrase is acceptable ONLY if it appears in an explicit
out-of-scope / non-goals section AND the in-scope FR does not depend on the
deferred behaviour to be testable. Example: "FR-007: User can request a
password reset via email link." with a non-goals line "SMS-based reset is a
future enhancement" is acceptable. A line in FR-007 itself saying "SMS reset
will be wired later" is a finding.

### Specificity Test Definition

Apply the following thought experiment to every requirement and acceptance
criterion:

> Could a different Claude instance — with the same codebase access and the
> same spec but none of this conversation — execute this requirement without
> asking clarifying questions?

If the answer is "no" or "would need to ask", the requirement fails
specificity and becomes a WARNING finding (`dimension: quality_gate`,
classified as `specificity_failure`).

Common specificity failures:

- Requirements that name a goal without naming the mechanism ("the system
  handles invalid input gracefully")
- Requirements with multiple plausible architectures and no constraint
  selecting between them ("provide a way to expire sessions")
- Acceptance criteria using the spec's own undefined terms
- Requirements that lean on prior conversation context the executor will not
  have

### Anti-Bloat Check Definitions

Each check below corresponds to one row in the Phase 4 quality-gate table in
SKILL.md. Definitions are precise so detection is deterministic.

| Check ID | Detection | Severity | Dimension classifier |
|----------|-----------|----------|----------------------|
| QG-USER-STORY | Spec contains the substring `As a` followed (within 80 chars) by `I want` or `so that`, OR a heading named "User Story / User Stories" | WARNING | `quality_gate` / `user_story_narrative` |
| QG-RESURRECTED | Spec contains a heading named exactly "Behavioral Contract", "Holdout", "Why this priority", or "Independent Test" (these were intentionally removed from the lean template) | WARNING | `quality_gate` / `resurrected_section` |
| QG-TEST-CATALOG | Any numbered list contains more than 10 items where each item names a test (e.g. `1. Test that …`, `2. Verify …`) | WARNING | `quality_gate` / `numbered_test_catalog` |
| QG-WEASEL | Any FR body contains a word from the Weasel-Word Prohibited List without a co-located quantitative threshold | WARNING per occurrence | `quality_gate` / `weasel_word` |
| QG-FALSIFY | An acceptance criterion is subjective and lacks a measurable threshold (e.g. "users find the flow intuitive") | BLOCKER | `quality_gate` / `non_falsifiable_ac` |
| QG-SPEC-FAIL | Specificity Test (above) returns "no" for a requirement | WARNING | `quality_gate` / `specificity_failure` |
| QG-DEFER-NO-REASON | An item marked deferred / out-of-scope has no stated reason | INFO | `quality_gate` / `deferred_no_reason` |

Each triggered check becomes one finding in the YAML issues block.

## Severity Mapping

The constitution uses three severities only. Older drafts using
CRITICAL / MAJOR / MINOR / OBSERVATION are deprecated. Map them as follows:

| Old | New |
|-----|-----|
| CRITICAL | BLOCKER |
| MAJOR | WARNING |
| MINOR | WARNING (if actionable) or INFO (if cosmetic) |
| OBSERVATION | INFO |

Verdict mapping:

- Any BLOCKER → **BLOCK**
- Any WARNING (no BLOCKER) → **REVISE**
- Only INFO (or none) → **PASS**

## Review Completeness Check

Before finalising the review, verify:

- [ ] Every Tier 1 lens has been applied (or explicitly marked as not applicable with justification)
- [ ] Tier 2 alignment checks (Scope Alignment, Codebase Fit) have each produced an explicit assessment
- [ ] Every Tier 3 anti-bloat check has been run against the spec
- [ ] Every finding has a specific section reference, FR ID, or quoted phrase from the spec
- [ ] Every finding has a concrete, actionable `fix_hint`
- [ ] Findings are classified by severity (BLOCKER, WARNING, INFO) — old severities (CRITICAL/MAJOR/MINOR/OBSERVATION) MUST NOT appear
- [ ] STRIDE thinking has been applied to every component / data flow, but no STRIDE matrix table appears in the output
- [ ] No "Dataset Gaps" table appears in the output
- [ ] No false reassurance language appears in the report
- [ ] The YAML issues block uses sequential `G-NNN` IDs starting at `G-001`
- [ ] The unasked questions section identifies genuine gaps, not rhetorical questions
