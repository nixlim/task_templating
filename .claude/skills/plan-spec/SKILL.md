---
name: plan-spec
description: >
  Produces lean, implementation-ready feature specifications through a focused
  4-phase workflow: discovery interview, codebase-aware FR drafting, behavioral
  scenarios with FR traceability, and a quality gate. Use when planning a feature,
  designing a specification, writing a spec, preparing a plan, or when asked to
  design, specify, or plan functionality. Supports --revise mode to systematically
  address findings from /grill-spec reviews. Use /plan-spec --revise <spec.md>
  <review.md> to revise a spec based on its adversarial review.
argument-hint: "[feature description or path to .md file] or [--revise path/to/spec.md path/to/spec-review.md]"
---

# Plan & Spec Preparation Skill

You produce structured, testable feature specifications. Specs you produce are
lean by default: every section earns its place, every FR is testable, every
scenario traces to an FR. You do not pad. You do not hedge with weasel words.

## Input Handling

1. If `$ARGUMENTS` starts with `--revise`, parse the two paths that follow:
   - First path: the spec file to revise
   - Second path: the review file from `/grill-spec`
   - **Jump directly to [Revision Mode](#revision-mode--revise-workflow).** Do NOT run Phases 1-4.
2. If `$ARGUMENTS` is a path ending in `.md`, read that file as the feature brief.
3. If `$ARGUMENTS` is a text description, use it as the starting point.
4. If no arguments are provided, ask: "What feature or change would you like to plan?"

For cases 2-4, proceed to Phase 1.

---

## Phase 1 — Discovery (Interview)

Interview the user. Do not run a fixed questionnaire — figure out what blocks
you from producing a precise spec, and ask about *that*. By the end of Phase 1
you must have enough signal on each of these dimensions:

- **Capability** — what the system will do, in one or two sentences
- **Scope boundaries** — what is in scope, what is explicitly out of scope
- **Systems touched** — which existing modules, services, data stores, or APIs
- **Failure modes** — how this realistically breaks (input, dependencies, concurrency)
- **Non-functional constraints** — performance, security, compatibility, regulatory

Ask whatever questions get you there. Probe until vague answers become precise.
If the user says "fast", ask for a number. If they say "secure", ask against
what threat. If they say "integrate with the API", ask which endpoints and
which failure responses matter.

Summarise what you have heard and ask the user to confirm.

**GATE — Scope confirmation:** Do NOT proceed to Phase 2 until the user
explicitly confirms the captured scope and boundaries. This is the single
checkpoint before FR drafting; get it right.

---

## Phase 2 — Codebase Scouting + FR Drafting

### 2.1 Codebase scouting

Explore the codebase to ground the spec in reality. Read the files, modules,
or services that the feature will touch or depend on. Note conventions
(test framework, error handling style, naming, validation patterns).

Fill in the **Existing Codebase Context** table for the spec:

| Area | Location | Relevance |
|------|----------|-----------|
| [module/service] | `path/to/file.ext` | [why this matters to the feature] |

Keep it tight — list what the implementer needs to know, not everything you read.

### 2.2 Functional Requirements

Draft FRs grouped by **area** (e.g., "Authentication", "Storage", "API"),
not chronology. Each FR has a unique ID and one of three modal verbs:

- `FR-001` (MUST) — non-negotiable; failure to implement breaks the feature
- `FR-002` (SHOULD) — expected; deviation requires justification
- `FR-003` (MAY) — optional; included if reasonable

Format:

```
**FR-001** (MUST): The system MUST [precise behavior with measurable criteria].
```

### 2.3 Specificity test

Apply this test to **every FR** before moving on:

> "Could a different Claude instance, with no access to this conversation,
> execute this FR without asking clarifying questions?"

If the answer is no, rewrite. Add the specific values, branches, formats, or
thresholds that are missing. Untestable FRs are not FRs — they are wishes.

### 2.4 Weasel-word check

The following phrases are **prohibited** in FRs, success criteria, and scope
statements. They signal deferred decisions masquerading as requirements:

- "v1", "v2"
- "simplified version", "basic version", "minimal implementation"
- "static for now", "hardcoded for now"
- "future enhancement", "placeholder"
- "will be wired later", "dynamic in future phase"
- "skip for now"

If you catch yourself writing one of these, stop. Either:
- Specify the actual current behavior precisely (e.g., "returns the literal string `default`" rather than "hardcoded for now"), OR
- Move the item to **Deferred From This Spec** with a one-line reason

### 2.5 Data Model / API Schemas / Error Contract

Draft these inline in the spec (do not defer them to a future doc):

- **Data Model** — entities, fields, types, relationships, persistence
- **API Schemas** — request/response shapes, status codes, content types, auth
- **Error Contract** — error categories, error shape, which errors are retryable, which surface to the user vs. log only

Use the format from [spec-template.md](spec-template.md). Be concrete — types
and example payloads, not prose descriptions.

---

## Phase 3 — Behavioral Scenarios + Tests + Traceability

### 3.1 Behavioral scenarios

Write Given/When/Then scenarios that trace to FR-IDs (not to user stories —
user stories are not part of this spec format). Use the format in
[bdd-template.md](bdd-template.md).

Each scenario has:

- **Traces to:** one or more FR-IDs (e.g., `FR-001, FR-003`)
- **Category:** exactly one of `Happy Path`, `Error Path`, `Edge Case`

Three categories — no others. If a scenario does not fit one of these, it does
not belong in the spec.

Coverage rule:

- Every MUST FR has at least one Happy Path scenario
- Every FR involving external input, dependencies, or failure modes has at
  least one Error Path scenario
- Boundary conditions identified during discovery get an Edge Case scenario

### 3.2 Testing Requirements

Three bulleted lists. **Not** a numbered test catalog. The implementing agent
fills in concrete test cases during TDD; the spec defines the surfaces.

```
### Unit
- [unit-level surface to cover]
- [unit-level surface to cover]

### Integration
- [integration boundary to cover]
- [integration boundary to cover]

### E2E Smoke
- [end-to-end happy path to verify]
```

Keep it short. If the list runs over ~10 items per level, you are designing
tests, not specifying surfaces.

### 3.3 Rolled-up traceability matrix

One row per FR range (e.g., `FR-001..FR-003`), grouped by area. Not one row per
FR — that is busywork.

| FR Range | Area | Scenarios | Test Surfaces |
|----------|------|-----------|---------------|
| FR-001..FR-003 | Authentication | S1 (Happy), S5 (Error) | Unit: token parsing; Integration: /auth/login |

Every FR ID must be covered by exactly one row. Every scenario must appear in
at least one row's `Scenarios` cell.

### 3.4 Task Decomposition Guidance (optional)

If the feature is large enough to span multiple PRs, add a short section
suggesting how it breaks down by FR groups or area. Skip if the feature is
small.

---

## Phase 4 — Quality Gate + Output

### 4.1 In-conversation self-audit

Before writing the spec file, run these audits **in the conversation** (not in
the spec output):

1. **Ambiguity self-audit** — re-read every FR. For each, ask: "what would a
   different agent assume here that I did not intend?" If something would be
   assumed, make it explicit.
2. **Specificity sweep** — apply the Phase 2.3 specificity test once more, in
   sequence, to every FR.
3. **Traceability check** — every FR appears in the traceability matrix; every
   scenario traces to an FR; every traceability row's scenarios actually exist.
4. **Bloat check** — does any section exist only because it "feels expected"?
   If yes, cut it.

These audits are workflow steps. They do not appear as sections in the output.

### 4.2 Seven-check gate

Before writing the file, verify ALL of these:

- [ ] **(1)** Every FR has MUST / SHOULD / MAY
- [ ] **(2)** Every scenario has both `Traces to:` and `Category:`
- [ ] **(3)** Every FR appears in the traceability matrix
- [ ] **(4)** Every success criterion is measurable (numeric threshold or clear pass/fail)
- [ ] **(5)** No weasel words from the prohibited list (Phase 2.4)
- [ ] **(6)** Testing Requirements is three bulleted lists, not a numbered catalog
- [ ] **(7)** Every item in Deferred From This Spec has a one-line reason

Any unchecked box: fix it before writing the file.

### 4.3 Output

1. Ask the user for an output filename. If none provided, generate one from
   the feature name in kebab-case with `.md` extension.
2. Assemble using [spec-template.md](spec-template.md) as the structure.
3. Write the file.
4. Present a one-paragraph summary: number of FRs (by modality), number of
   scenarios (by category), and any items moved to Deferred.
5. Recommend the next step:

   > Next: run `/grill-spec <path>` to adversarially review this spec before implementation.

---

## Sections this spec format does NOT contain

The following sections from older formats are deliberately omitted. Do not add
them. If the user asks for them, explain that the corresponding signal is
captured elsewhere:

- **User Stories & Acceptance Criteria** (narrative, "Why this priority", "Independent Test") — capability is captured in the Capability summary and FRs
- **Behavioral Contract (When/Then summary)** — duplicates the BDD scenarios
- **Edge Cases section** — folded into scenarios with `Category: Edge Case`
- **Explicit Non-Behaviors** — recast as `Deferred From This Spec` with reasons
- **Integration Boundaries** — captured by Existing Codebase Context + API Schemas
- **TDD Plan: Test Hierarchy / Implementation Order / Test Datasets** — replaced by Testing Requirements bullets
- **Regression Test Requirements** — implementer's responsibility, not spec scope
- **Ambiguity Warnings** — kept in-conversation (Phase 4.1), not in the spec
- **Evaluation Scenarios / Holdout** — out of scope
- **Assumptions** — promote to FRs or move to Deferred
- **Clarifications Q&A** — answers fold into FRs

---

## Revision Mode (`--revise` Workflow)

Triggered when `$ARGUMENTS` starts with `--revise`. Single linear pass — no
R0/R1/R2/R3 ceremony. The goal is to drive BLOCKER and WARNING counts to zero
in as few rounds as possible.

### Inputs

- Spec file (path 1)
- Review file from `/grill-spec` (path 2). Must contain a YAML `## Issues`
  block with entries shaped like:

  ```yaml
  ## Issues
  - id: I-001
    severity: BLOCKER     # or WARNING or NIT
    dimension: Ambiguity  # see strategy table below
    section: "Phase 2 — FR-007"
    finding: "FR-007 says 'fast response' without a numeric threshold"
    recommendation: "Specify p95 latency target in milliseconds"
  ```

### Workflow

1. **Read both files** (spec and review). Parse the YAML `## Issues` block.
2. **Sort issues:** BLOCKER first, then WARNING, then NIT (NITs only addressed
   if no BLOCKER/WARNING remains).
3. **For each issue, apply the strategy from the dimension-to-strategy table**
   (below). The strategy is deterministic — pick it from the table, then apply
   the review's `recommendation` as the concrete edit.
4. **Re-run the Phase 4.2 seven-check gate** on the revised spec.
5. **Stall detection:** count BLOCKER+WARNING issues you addressed this round.
   If a subsequent round is requested and the new review's BLOCKER+WARNING
   count has not decreased, **stop and surface this to the user.** Stalling
   means either the issues are interdependent or the user needs to make a
   judgement call you cannot make from the review alone.
6. **Hard cap:** maximum 3 revision rounds. After round 3, surface remaining
   issues to the user with a recommendation: re-scope, accept the issue as
   known risk, or restart the spec from scratch.
7. **Write the revised spec** in place. Recommend re-running `/grill-spec`.

### Dimension-to-strategy table

Each review issue carries a `dimension`. Map it to an action:

| Dimension | Strategy |
|-----------|----------|
| **Ambiguity** | Replace vague language with precise terms. Add the missing units, thresholds, or branch values directly into the FR or scenario. |
| **Incompleteness** | Add the missing FR, scenario, or schema field. Add corresponding row to traceability matrix. |
| **Inconsistency** | Pick one canonical version. Update all other locations to match. |
| **Infeasibility** | Rewrite the FR to be testable. If a numeric target is unrealistic, lower it (and note the change in the revision summary so the user can object). |
| **Insecurity** | Add the missing auth, validation, rate limit, or secret-handling FR. Add an Error Path scenario for the misuse case. |
| **Inoperability** | Add monitoring/observability/rollback FR. Add an Error Path scenario for the failure that operability would catch. |
| **Incorrectness** | Fix the logic. Update the affected scenario's `Given` or `Then` to reflect correct behavior. |
| **Overcomplexity** | Cut the offending FR or section. Move to `Deferred From This Spec` with reason. |

### Revision summary

After writing the revised spec, present:

```
## Revision Summary

**Spec:** <path>
**Review:** <path>
**Round:** N of 3

| Severity | Addressed | Deferred |
|----------|-----------|----------|
| BLOCKER  | N | 0 |
| WARNING  | N | M |
| NIT      | N | M |

### Changes made
- [section]: [what changed]
- ...

### Stall check
[Round 1 baseline OR "BLOCKER+WARNING reduced from X to Y"]

### Next step
Run `/grill-spec <spec-path>` to verify the revision.
```

---

## Supporting Files

- For the output document structure, see [spec-template.md](spec-template.md)
- For BDD scenario format and rules, see [bdd-template.md](bdd-template.md)
- For a complete example of expected output, see [examples/sample-output.md](examples/sample-output.md)
