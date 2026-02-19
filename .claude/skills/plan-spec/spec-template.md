# Spec Output Template

This is the structural template for the output document. Fill in every section.
Replace all `[bracketed placeholders]` with actual content. Remove this header
block from the final output.

---

# Feature Specification: [Feature Name]

**Created**: [YYYY-MM-DD]
**Status**: Draft
**Input**: [Brief description of what triggered this spec, or link to source document]

---

## User Stories & Acceptance Criteria

### User Story 1 — [Title] (Priority: P[0-4])

[Narrative paragraph: A [role/actor] wants to [action] so that [benefit].
Describe the current pain point and how this story addresses it.]

**Why this priority**: [Justify why this story has this priority relative to others.]

**Independent Test**: [Describe how this story can be verified in isolation,
delivering value even if other stories are not yet implemented.]

**Acceptance Scenarios**:

1. **Given** [precondition], **When** [action], **Then** [expected outcome].
2. **Given** [precondition], **When** [action], **Then** [expected outcome].

---

### User Story 2 — [Title] (Priority: P[0-4])

[Repeat the same structure for each user story.]

---

## Behavioral Contract

Primary flows:
- When [condition], the system [behavior].

Error flows:
- When [error condition], the system [behavior].

Boundary conditions:
- When [boundary condition], the system [behavior].

---

## Edge Cases

- [What happens when [unusual condition]? Expected: [behaviour].]
- [What happens when [boundary condition]? Expected: [behaviour].]
- [What happens when [error condition]? Expected: [behaviour].]

---

## Explicit Non-Behaviors

- The system must not [behavior] because [reason].
- The system must not [behavior] because [reason].

---

## Integration Boundaries

### [External System Name]

- **Data in**: [what this system receives]
- **Data out**: [what this system returns]
- **Contract**: [request/response format, protocol, auth]
- **On failure**: [behavior when unavailable or returning errors]
- **Development**: [real service | mock/simulated twin] — [reason]

---

## BDD Scenarios

### Feature: [Feature Name]

#### Scenario: [Descriptive Scenario Title]

**Traces to**: User Story [N], Acceptance Scenario [M]
**Category**: [Happy Path | Alternate Path | Error Path | Edge Case]

- **Given** [precondition]
- **And** [additional precondition, if needed]
- **When** [action]
- **And** [additional action, if needed]
- **Then** [expected outcome]
- **And** [additional assertion, if needed]

---

#### Scenario Outline: [Descriptive Title for Parameterised Scenario]

**Traces to**: User Story [N], Acceptance Scenario [M]
**Category**: [Happy Path | Alternate Path | Error Path | Edge Case]

- **Given** [precondition with `<placeholder>`]
- **When** [action with `<placeholder>`]
- **Then** [expected outcome with `<placeholder>`]

**Examples**:

| placeholder_1 | placeholder_2 | expected |
|---------------|---------------|----------|
| value_a       | value_b       | result_a |
| value_c       | value_d       | result_b |

---

[Repeat for all scenarios. Group by User Story for readability.]

---

## Test-Driven Development Plan

### Test Hierarchy

| Level       | Scope                        | Purpose                                    |
|-------------|------------------------------|--------------------------------------------|
| Unit        | [Individual functions/methods]| [Validates logic in isolation]              |
| Integration | [Module interactions]        | [Validates components work together]        |
| E2E         | [Full user workflows]        | [Validates complete feature from user view] |

### Test Implementation Order

Write these tests BEFORE implementing the feature code. Order: unit first,
then integration, then E2E. Within each level, order by dependency.

| Order | Test Name | Level | Traces to BDD Scenario | Description |
|-------|-----------|-------|------------------------|-------------|
| 1     | [test_name] | Unit | Scenario: [title] | [What this test verifies] |
| 2     | [test_name] | Unit | Scenario: [title] | [What this test verifies] |
| ...   | ...       | Integration | Scenario: [title] | ... |
| ...   | ...       | E2E  | Scenario: [title] | ... |

### Test Datasets

#### Dataset: [Context — e.g., "Email Input Validation"]

| # | Input | Boundary Type | Expected Output | Traces to | Notes |
|---|-------|---------------|-----------------|-----------|-------|
| 1 | [value] | [type] | [expected] | BDD Scenario: [title] | [note] |
| 2 | [value] | [type] | [expected] | BDD Scenario: [title] | [note] |

[Repeat dataset tables for each distinct input domain or validation context.]

### Regression Test Requirements

**If modifying existing functionality:**

| Existing Behaviour | Existing Test | New Regression Test Needed | Notes |
|--------------------|---------------|---------------------------|-------|
| [behaviour]        | [test name]   | [Yes/No — if yes, name]   | [why] |

**If new functionality:**

> No regression impact — new capability. Integration seams protected by: [list of existing tests covering the boundary].

---

## Functional Requirements

- **FR-001**: System MUST [requirement].
- **FR-002**: System SHOULD [requirement].
- **FR-003**: System MAY [requirement].

---

## Success Criteria

- **SC-001**: [Measurable outcome with specific threshold — e.g., "Response time under 200ms at p95 for 1000 concurrent users."]
- **SC-002**: [Observable pass/fail condition — e.g., "All exported files are valid JSON parseable by `jq`."]

---

## Traceability Matrix

| Requirement | User Story | BDD Scenario(s)          | Test Name(s)            |
|-------------|-----------|--------------------------|-------------------------|
| FR-001      | US-1      | Scenario: [title]        | [test_name]             |
| FR-002      | US-1, US-2| Scenario: [t1], [t2]     | [test_a], [test_b]      |

**Completeness check**: Every FR-xxx row must have at least one BDD scenario
and one test. Every BDD scenario must appear in at least one row.

---

## Ambiguity Warnings

| # | What's Ambiguous | Likely Agent Assumption | Question to Resolve |
|---|------------------|------------------------|---------------------|
| 1 | [gap in spec]    | [what agent would do]  | [question for user] |

---

## Evaluation Scenarios (Holdout)

> **Note**: These scenarios are for post-implementation evaluation only.
> They must NOT be visible to the implementing agent during development.
> Do not reference these in the TDD plan or traceability matrix.

### Scenario: [Title]
- **Setup**: [initial conditions]
- **Action**: [what is done]
- **Expected outcome**: [observable result]
- **Category**: [Happy Path | Error | Edge Case]

### Scenario: [Title]
- **Setup**: [initial conditions]
- **Action**: [what is done]
- **Expected outcome**: [observable result]
- **Category**: [Happy Path | Error | Edge Case]

---

## Assumptions

- [Assumption about environment, dependencies, user behaviour, or infrastructure.]
- [Assumption about what is NOT in scope.]

## Clarifications

### [YYYY-MM-DD]

- Q: [Question raised during discovery] -> A: [Answer or decision made.]
- Q: [Another question] -> A: [Answer.]
