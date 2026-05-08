# Spec Output Template

Structural template for the spec document `/plan-spec` produces. Fill every
applicable section. Sections marked **(if applicable)** are conditional — drop
them entirely if they do not apply, do not leave empty placeholders. Replace
all `[bracketed placeholders]` with concrete content.

Remove this header block from the final output.

---

## 1. Header

# Feature Specification: [Feature Name]

**Created**: [YYYY-MM-DD]
**Status**: Draft
**Intent**: [One or two sentences stating what this feature does. End with: "Out of scope: [comma-separated list of explicitly excluded items]."]

---

## 2. Implementation Scope

Numbered capabilities the implementer must deliver, followed by guard rails
constraining how. No narrative.

**Capabilities**:

1. [Capability — concrete, testable, one sentence]
2. [Capability]
3. [Capability]

**Guard rails**:

- [Constraint on implementation — e.g., "Must not introduce a new dependency"]
- [Constraint — e.g., "Must preserve existing API response shape for /v1/foo"]
- [Constraint]

---

## 3. Existing Codebase Context *(if modifying existing code)*

| Area | Existing files | Required change |
|------|----------------|-----------------|
| [module/component] | `path/to/file.ext` | [What changes — add, modify, replace] |
| [module/component] | `path/to/file.ext` | [What changes] |

Omit this section if the feature is entirely new with no existing code touched.

---

## 4. Terminology *(if domain terms)*

| Term | Definition |
|------|------------|
| [Term] | [Precise definition — used to disambiguate, not to teach] |
| [Term] | [Definition] |

Omit if no domain-specific terms need disambiguation.

---

## 5. Surface / API Inventory *(if APIs, CLI commands, or tool surfaces)*

### New surfaces

- `[name]` — [purpose, in one line]
- `[name]` — [purpose]

### Modified surfaces

- `[name]` — [what changes]

### Deferred From This Spec

Items deliberately excluded. Each line must give a reason. This section
absorbs what older formats called "Explicit Non-Behaviors" and "Assumptions".

- [Item] — [reason: out of scope, future work, requires upstream change, etc.]
- [Item] — [reason]

Omit the entire section if there is no API/CLI/tool surface and no items are
being deferred.

---

## 6. Data Model Changes *(if schema changes)*

Concrete schema or SQL. Not prose. Each change has a unique ID.

**DM-001**: Add table `[name]`

```sql
CREATE TABLE [name] (
  [column] [type] [constraints],
  ...
);
```

**DM-002**: Modify column `[table.column]`

```sql
ALTER TABLE [table] ALTER COLUMN [column] TYPE [new_type];
```

Migration / backfill notes (one line each):

- [How existing rows are populated, if applicable]
- [Backwards-compat behavior during deploy, if applicable]

Omit the entire section if no schema changes.

---

## 7. Functional Requirements

FRs grouped by **area**, each tagged MUST / SHOULD / MAY. Each FR must pass the
specificity test: a different Claude could execute it without asking
clarifying questions.

### [Area: e.g., Authentication]

- **FR-001** (MUST): The system MUST [precise behavior with measurable criteria].
- **FR-002** (SHOULD): The system SHOULD [behavior, with the deviation condition explicit].
- **FR-003** (MAY): The system MAY [optional behavior].

### [Area: e.g., Storage]

- **FR-004** (MUST): ...

Forbidden language in FRs (and elsewhere): "v1", "v2", "simplified version",
"static for now", "hardcoded for now", "future enhancement", "placeholder",
"basic version", "minimal implementation", "will be wired later", "dynamic in
future phase", "skip for now". If you would write one of these, either
specify the actual current behavior or move the item to **Deferred From
This Spec**.

---

## 8. API / Schema Contracts *(if API)*

For each endpoint or message:

### `[METHOD] /path` — [one-line purpose]

**Request**:

```json
{
  "field": "type — constraint"
}
```

**Response 2xx**:

```json
{
  "field": "type"
}
```

**Auth**: [bearer | session | none] — [scope/role required]
**Content-Type**: [application/json | other]

Omit the entire section if no API surface.

---

## 9. Error Contract *(if error surface)*

Compact table. One row per surfaced error condition.

| Condition | Status | Error code | Notes |
|-----------|--------|------------|-------|
| [Condition — e.g., "Token expired"] | 401 | `auth.token_expired` | [Retryable? User-visible?] |
| [Condition] | 4xx/5xx | `code` | [Notes] |

Error response shape (single declaration covers all rows):

```json
{
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

Omit the entire section if there is no user-visible or programmatic error surface.

---

## 10. Behavioral Scenarios

Given / When / Then. Each scenario carries `Traces to:` (FR-IDs) and
`Category:` (exactly one of Happy Path, Error Path, Edge Case). See
[bdd-template.md](bdd-template.md) for format details.

### Scenario: [Title]

**Traces to**: FR-001, FR-002
**Category**: Happy Path

- **Given** [precondition]
- **When** [single action]
- **Then** [expected outcome]
- **And** [additional assertion]

### Scenario: [Title]

**Traces to**: FR-003
**Category**: Error Path

- **Given** [precondition]
- **When** [action]
- **Then** [error outcome with specific code/message]

### Scenario: [Title]

**Traces to**: FR-001
**Category**: Edge Case

- **Given** [boundary condition]
- **When** [action]
- **Then** [behavior at boundary]

Coverage rule: every MUST FR has at least one Happy Path scenario; every FR
involving external input or dependencies has at least one Error Path scenario.

---

## 11. Testing Requirements

Three bulleted lists. Surfaces, not test cases. The implementer fills in
concrete assertions during TDD.

### Unit

- [Surface to cover — e.g., "Token parsing rejects malformed JWT"]
- [Surface]

### Integration

- [Boundary to cover — e.g., "POST /auth/login against real DB"]
- [Boundary]

### E2E Smoke

- [End-to-end happy path — e.g., "User logs in, accesses protected resource, logs out"]

If any list runs over ~10 items, you are designing tests, not specifying
surfaces — collapse to broader surfaces.

---

## 12. Success Criteria

Measurable outcomes. Each has a numeric threshold or unambiguous pass/fail
condition. No subjective language.

- **SC-001**: [Outcome — e.g., "p95 login latency under 200ms at 1000 concurrent users"]
- **SC-002**: [Outcome — e.g., "All emitted JSON parses with `jq` exit code 0"]
- **SC-003**: [Outcome]

---

## 13. Traceability Matrix

Rolled up. One row per FR range (or per area when ranges are small). NOT one
row per FR.

| FR Range | Area | Scenarios | Test surfaces |
|----------|------|-----------|---------------|
| FR-001..FR-003 | Authentication | Login (Happy), Token expired (Error) | Unit: token parser; Integration: /auth/login |
| FR-004..FR-006 | Storage | Persist (Happy), Disk full (Error), Empty payload (Edge) | Unit: serializer; Integration: storage adapter |

Every FR ID must be covered. Every scenario must appear in at least one row's
`Scenarios` cell.

---

## 14. Task Decomposition Guidance *(optional)*

Ordered build slices for `/taskify`. Use when the feature spans multiple PRs.
Each slice should be independently deployable.

1. **[Slice name]** — covers FR-001..FR-003. [Outcome: what works after this slice ships]
2. **[Slice name]** — covers FR-004..FR-006. [Outcome]
3. **[Slice name]** — covers FR-007..FR-NNN. [Outcome]

Omit if the feature is a single PR.
