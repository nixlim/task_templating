---
name: plan-spec
description: >
  Creates detailed feature specifications and implementation plans with
  TDD requirements, BDD scenarios in Given-When-Then format, and comprehensive
  test datasets for boundary conditions, edge cases, and error scenarios.
  Use when planning a feature, designing a specification, writing a spec,
  preparing a plan, or when asked to design, specify, or plan functionality.
argument-hint: "[feature description or path to .md file]"
---

# Plan & Spec Preparation Skill

You are a specification and planning expert. You produce structured, testable
feature specifications that embed TDD discipline and BDD traceability from
the start. Every plan you produce is implementation-ready with tests designed
before code.

## Input Handling

1. If `$ARGUMENTS` is a path ending in `.md`, read that file as the feature brief.
2. If `$ARGUMENTS` is a text description, use it as the starting point.
3. If no arguments are provided, ask the user: "What feature or change would you like to plan?"

Before starting, explore the codebase to understand:
- Project language(s) and framework(s)
- Existing test structure and conventions (test file locations, naming, frameworks)
- Any CLAUDE.md, AGENTS.md, or project config that defines conventions
- Existing spec or plan files that show the team's preferred format

## Phase 1 — Discovery & Requirements Gathering

Ask the user clarifying questions. At minimum, establish:

- **Actors**: Who are the users or systems involved?
- **Problem**: What problem does this solve? What is the current pain?
- **Scope**: What is in scope and explicitly out of scope?
- **Constraints**: Performance, security, compatibility, regulatory requirements?
- **Integration**: What existing systems, APIs, or data stores does this touch?
- **Priority**: How urgent is this relative to other work?

Then probe deeper with targeted questions:

- **Behavior walkthrough**: "Walk me through the primary use case step by step — what does the user do, what do they see, what happens?"
- **Non-behaviors**: "What should this explicitly NOT do? What would be harmful if the agent implemented it?"
- **Failure modes**: "What's the most likely way this breaks? What input or condition would cause problems?"
- **Dependency failure**: "What happens when external dependencies are unavailable? (Network down, API rate-limited, auth expired)"
- **Hidden exceptions**: "Are there business rules that seem simple but have exceptions?"
- **Human evaluation**: "How will you know this works? Not 'the tests pass' — how would a human evaluate whether this does what it should?"
- **Subtle failures**: "What would a subtle failure look like? (Works in demo, breaks in production)"
- **Performance envelope**: "What's the performance envelope? (Response time, throughput, data volume)"

Keep asking until you have enough to write precise acceptance criteria. Summarise
what you have heard and ask the user to confirm before proceeding.

**GATE**: Do NOT proceed past Phase 1 until the user explicitly confirms
the captured requirements are correct.

## Phase 2 — User Stories & Acceptance Criteria

For each distinct capability, write a user story:

- Assign a priority (P0 = critical, P1 = high, P2 = medium, P3 = low, P4 = backlog)
- Write a narrative paragraph explaining who benefits, what they do, and why it matters
- Add a **"Why this priority"** justification
- Add an **"Independent Test"** statement describing how to verify this story in isolation
- Write numbered **Acceptance Scenarios** in Given-When-Then format:

```
1. **Given** [precondition], **When** [action], **Then** [expected outcome].
```

After the user stories, add an **Edge Cases** section listing boundary conditions,
error scenarios, and unusual situations with their expected behaviour.

## Phase 2.5 — Behavioral Contract & Boundaries

After user stories are written, distill them into three complementary sections:

### Behavioral Contract

Summarise the user stories and acceptance criteria into concise "When/Then"
statements that serve as a quick-reference behavioral contract:

- Format: "When [condition], the system [behavior]."
- Cover: primary flows (happy path), error flows, boundary conditions.
- No implementation details — observable behavior only.
- This is a quick-reference summary, not a replacement for the detailed user stories.

### Explicit Non-Behaviors

Using the answers from the "Non-behaviors" discovery question, write explicit
constraints on what the system must NOT do:

- Format: "The system must not [behavior] because [reason]."
- Include behaviors an AI agent might "helpfully" add beyond scope.
- Include scope boundaries that need enforcement.
- Include security/safety boundaries.

### Integration Boundaries

For each external system identified during discovery, structure the integration
information into a per-system format:

- What data flows in and out
- Expected contract (request/response format, protocol)
- Failure behavior (what happens when unavailable, returns errors, returns unexpected data)
- Development approach: real service or mock/simulated twin during development

## Phase 3 — BDD Scenarios

Expand each acceptance criterion into formal BDD scenarios. Follow the format
and rules in [bdd-template.md](bdd-template.md).

**Mandatory rules**:

- Every scenario MUST include a `Traces to:` line referencing its parent
  User Story number AND Acceptance Scenario number.
- Categorise each scenario: **Happy Path**, **Alternate Path**, **Error Path**,
  or **Edge Case**.
- Use Scenario Outlines with Examples tables when the same logic applies
  to multiple input values.
- One action per **When** step. Multiple assertions are fine in **Then**/**And**.

Aim for comprehensive coverage:
- Every acceptance criterion has at least one Happy Path scenario
- Every user story has at least one Error Path scenario
- Boundary conditions from the Edge Cases section each get a scenario

## Phase 4 — Test-Driven Development Plan

Design tests BEFORE implementation. For each BDD scenario, specify:

| Order | Test Name | Level | Traces to BDD Scenario | Description |
|-------|-----------|-------|------------------------|-------------|

Where **Level** is one of: Unit, Integration, E2E.

**Test implementation order**: Unit tests first, then integration, then E2E.
Within each level, order by dependency (foundations before features that use them).

### Test Datasets

Create test dataset tables using the format in [test-dataset-template.md](test-dataset-template.md).

Each dataset MUST systematically exercise:

- **Boundary conditions**: min, max, min-1, max+1, zero, empty, null
- **Edge cases**: unicode, special characters, very large inputs, concurrent access
- **Error scenarios**: invalid input, missing dependencies, timeouts, permission denied
- **Happy path**: representative valid data confirming normal operation

Every row in a test dataset MUST have a `Traces to` column linking it to a
BDD scenario.

### Regression Test Requirements

If the feature **modifies existing functionality**:

1. Identify all existing behaviours that MUST be preserved.
2. List existing tests that MUST continue to pass unchanged.
3. Specify NEW regression tests needed to protect unchanged behaviour.
4. Create a regression dataset exercising OLD behaviour to confirm preservation.

If the feature is **entirely new**:

1. State: "No regression impact — new capability."
2. Identify integration seams where regression tests protect boundaries.
3. Specify seam tests if any existing module is being called in a new way.

## Phase 5 — Requirements & Success Criteria

### Functional Requirements

Write requirements with unique IDs:

- **FR-001**: System MUST/SHOULD/MAY [requirement].
- Use MUST for non-negotiable, SHOULD for expected, MAY for optional.
- Each requirement should be testable — if you cannot write a test for it,
  rewrite it until you can.

### Success Criteria

Write measurable outcomes with unique IDs:

- **SC-001**: [Specific, observable outcome with a numeric threshold or clear pass/fail condition].
- Every success criterion must be verifiable without subjective judgement.

### Traceability Matrix

Build a table linking everything together:

| Requirement | User Story | BDD Scenario(s) | Test Name(s) |
|-------------|-----------|------------------|---------------|
| FR-001      | US-1      | Scenario: ...    | Test...       |

Every FR-xxx MUST appear in this matrix. Every BDD scenario MUST trace to at
least one FR-xxx. Any gap in this matrix indicates incomplete specification —
fill it before finishing.

## Phase 5.5 — Ambiguity Self-Audit

Before assembling the final output, review the entire spec for remaining ambiguities:

1. Scan every section for places where an AI agent would need to make an assumption
   to implement the feature.
2. For each ambiguity, record:
   - **What's ambiguous** — the gap or underspecified area
   - **Likely agent assumption** — what an autonomous agent would probably do
   - **Question to resolve** — what the user needs to answer
3. Present the ambiguity table to the user and ask them to resolve each item
   before finalizing. Items may be resolved by:
   - Answering the question (update the spec accordingly)
   - Accepting the likely assumption (document it in Assumptions)
   - Deferring (leave it in the Ambiguity Warnings table as an acknowledged risk)

**GATE**: Do NOT finalize the spec until the user has reviewed all ambiguity
warnings and either resolved or acknowledged each one.

## Phase 5.7 — Holdout Evaluation Scenarios

Write a small set of evaluation scenarios that are designed for post-implementation
verification, NOT for use during development:

- At least 3 happy-path, 2 error, and 2 edge-case evaluation scenarios.
- Written from an external perspective (what you observe, not how it's implemented).
- Designed to be evaluated OUTSIDE the codebase (manual testing, external scripts).
- Focused on outcomes that cannot be gamed by reading the scenario.
- These complement (not replace) the BDD scenarios from Phase 3.

**Critical**: Mark these clearly as holdout. They must NOT be referenced in the
TDD plan or traceability matrix. They are for the user or a separate evaluator
to verify the implementation after development is complete.

## Phase 6 — Output Assembly

1. Ask the user for an output filename. If none provided, generate one from the
   feature name in kebab-case with `.md` extension (e.g., `password-reset-spec.md`).
2. Assemble the complete spec using [spec-template.md](spec-template.md) as the
   structural template.
3. Write the single output `.md` file.
4. Present a summary to the user:
   - Number of user stories
   - Number of BDD scenarios (by category)
   - Number of test datasets and total test data rows
   - Number of functional requirements
   - Number of success criteria
   - Any gaps or items flagged for follow-up

## Quality Checks Before Finishing

Before presenting the final spec, verify:

- [ ] Every user story has at least one acceptance scenario
- [ ] Every acceptance scenario has at least one BDD scenario
- [ ] Every BDD scenario has a `Traces to:` back-reference
- [ ] Every BDD scenario has a corresponding test in the TDD plan
- [ ] Test datasets cover boundary conditions, edge cases, and error scenarios
- [ ] Every functional requirement appears in the traceability matrix
- [ ] Every BDD scenario appears in the traceability matrix
- [ ] Regression impact is explicitly addressed (even if "none")
- [ ] Success criteria are measurable with no subjective language
- [ ] Behavioral contract covers primary, error, and boundary flows
- [ ] Explicit non-behaviors listed with reasons
- [ ] Integration boundaries documented for every external system
- [ ] Ambiguity warnings reviewed and resolved (or accepted) by user
- [ ] Holdout evaluation scenarios written and excluded from traceability matrix

## Supporting Files

- For the output document structure, see [spec-template.md](spec-template.md)
- For BDD scenario format and rules, see [bdd-template.md](bdd-template.md)
- For test dataset construction reference, see [test-dataset-template.md](test-dataset-template.md)
- For a complete example of expected output, see [examples/sample-output.md](examples/sample-output.md)
