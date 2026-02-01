# Task Creation Instructions for AI Agents

## 1. Overview

This document describes the **taskval + bd** workflow for quality-gated task tracking. AI coding agents use this workflow to:

1. Decompose work into structured, machine-verifiable task JSON
2. Validate tasks against the Structured Task Template Spec (catching errors before execution)
3. Record validated tasks as Beads issues with full traceability
4. Execute work using `bd ready` to pick unblocked tasks

**When to use this workflow:**
- New features requiring multiple implementation steps
- Bug fixes with clear acceptance criteria
- Refactoring plans touching multiple files
- Any multi-step work benefiting from structured tracking

## 2. Prerequisites

| Requirement | How to verify | How to install |
|---|---|---|
| `taskval` binary | `taskval --help` | `go build -o taskval ./cmd/taskval/` |
| `bd` (beads) CLI | `bd --help` | `go install github.com/steveyegge/beads/cmd/bd@latest` |
| Beads initialized | `bd list --limit 0` | `bd init` |

Task JSON files must conform to the [Structured Task Template Spec](STRUCTURED_TEMPLATE_SPEC.md).

## 3. Workflow A: User-Provided Tasks

When a user describes what they want built:

```
1. User describes feature/bug/refactor
2. Agent decomposes into task graph JSON
3. Agent writes JSON to .tasks/<name>.task.json
4. Agent validates:
   taskval --mode=graph .tasks/<name>.task.json
5. On failure: read errors, fix JSON, retry (up to 3x)
6. On success: create beads:
   taskval --mode=graph --create-beads .tasks/<name>.task.json
7. Agent runs: bd ready
8. Agent picks first available task and begins work
```

**Preview before creating:**
```bash
taskval --mode=graph --create-beads --dry-run .tasks/<name>.task.json
```

## 4. Workflow B: Existing Task Files in Repository

When task files already exist in the repository:

```bash
# 1. Find task files
find . -name "*.task.json"

# 2. Validate each
taskval --mode=graph <file>

# 3. If valid and not yet in beads
taskval --mode=graph --create-beads <file>

# 4. Find available work
bd ready
```

## 5. Writing Valid Task JSON

### Required Fields (6)

| Field | Type | Rules |
|---|---|---|
| `task_id` | string | kebab-case, unique, max 60 chars |
| `task_name` | string | Imperative phrase starting with verb, max 80 chars |
| `goal` | string | Single testable sentence. **Forbidden:** "try", "explore", "investigate", "look into" |
| `inputs` | array | Each: `{name, type, constraints, source}` |
| `outputs` | array | Each: `{name, type, constraints, destination}` |
| `acceptance` | array | Each must be independently verifiable. **Avoid:** "works correctly", "is good", "properly" |

### Contextual Fields (3) -- provide value or `{"status": "N/A", "reason": "..."}`

| Field | Type | Notes |
|---|---|---|
| `depends_on` | array of task_ids or N/A | References must exist in graph |
| `constraints` | array of strings or N/A | Behavioral/performance constraints |
| `files_scope` | array of paths/globs or N/A | Required for implementation tasks |

### Optional Fields (6)

| Field | Type | Notes |
|---|---|---|
| `non_goals` | array | Explicit exclusions |
| `effects` | string or array | Side effects (e.g., "None", or `[{type, target}]`) |
| `error_cases` | array | Each: `{condition, behavior, output}` |
| `priority` | string | "critical", "high", "medium", "low" |
| `estimate` | string | "trivial", "small", "medium", "large", "unknown" |
| `notes` | string | Human-readable context |

### Common Validation Failures

| Code | Problem | Fix |
|---|---|---|
| SCHEMA | Missing required fields, wrong types | Add the missing field with correct type |
| V2 | Duplicate task_id | Make each task_id globally unique |
| V4 | Dangling depends_on reference | Add the referenced task or remove the dependency |
| V5 | Dependency cycle | Break the cycle by removing one edge |
| V6 | Forbidden word in goal | Rewrite as testable outcome: "The function returns X" |
| V7 | Vague acceptance criteria | Replace "works correctly" with specific assertions |
| V9 | Missing contextual field | Add the field or mark as `{"status": "N/A", "reason": "..."}` |
| V10 | Implementation task missing files_scope | Add files_scope with paths the task will modify |

### Single Task Example

```json
{
  "task_id": "calculate-discounted-total",
  "task_name": "Implement discount calculation for order totals",
  "goal": "Given a price and a discount, return the discounted total, guaranteed non-negative.",
  "inputs": [
    {"name": "price", "type": "f64", "constraints": "price > 0", "source": "Order record"}
  ],
  "outputs": [
    {"name": "total", "type": "f64", "constraints": "total >= 0", "destination": "Return value"}
  ],
  "acceptance": [
    "CalculateTotal(100.0, Fixed(10.0)) == 90.0",
    "CalculateTotal(50.0, Fixed(60.0)) == 0.0 (clamped)"
  ],
  "depends_on": {"status": "N/A", "reason": "Pure function, no dependencies"},
  "constraints": ["Pure function: no side effects"],
  "files_scope": ["internal/pricing/discount.go", "internal/pricing/discount_test.go"],
  "priority": "medium",
  "estimate": "trivial"
}
```

### Task Graph Example

```json
{
  "version": "0.1.0",
  "tasks": [
    {
      "task_id": "task-a",
      "task_name": "Implement feature A",
      "goal": "Feature A returns correct results for all inputs.",
      "inputs": [{"name": "x", "type": "int", "constraints": "x > 0", "source": "arg"}],
      "outputs": [{"name": "y", "type": "int", "constraints": "y >= 0", "destination": "return"}],
      "acceptance": ["Given x=5, returns 25"],
      "depends_on": {"status": "N/A", "reason": "No dependencies"},
      "constraints": ["No external calls"],
      "files_scope": ["pkg/a.go"]
    },
    {
      "task_id": "task-b",
      "task_name": "Implement feature B using A",
      "goal": "Feature B composes A's output into a formatted report.",
      "inputs": [{"name": "data", "type": "list<int>", "constraints": "len > 0", "source": "API"}],
      "outputs": [{"name": "report", "type": "string", "constraints": "len > 0", "destination": "stdout"}],
      "acceptance": ["Given [1,2,3], report contains all values"],
      "depends_on": ["task-a"],
      "constraints": ["Use A's output directly"],
      "files_scope": ["pkg/b.go"]
    }
  ]
}
```

## 6. Task Decomposition Guidelines

- **Granularity:** One task = one coherent unit of work (30 minutes to 4 hours)
- **Goal format:** Single testable sentence starting with a verb. Describes what the system does when complete, not what the agent does.
- **Acceptance criteria:** Each must be independently verifiable with concrete expected values. No vague terms.
- **Dependencies:** Must form a DAG (Directed Acyclic Graph). No cycles, no self-references.
- **N/A pattern:** Use `{"status": "N/A", "reason": "your justification"}` for non-applicable contextual fields.
- **files_scope:** As narrow as possible. Lists only files the task creates or modifies (blast radius control).

## 7. The Validation -> Beads Pipeline

### Command Reference

```bash
# Validate only (no beads)
taskval --mode=graph plan.task.json
taskval --mode=task single.task.json

# Validate + preview beads commands
taskval --mode=graph --create-beads --dry-run plan.task.json

# Validate + create beads
taskval --mode=graph --create-beads plan.task.json

# With custom epic title
taskval --mode=graph --create-beads --epic-title "OAuth Feature" plan.task.json

# JSON output
taskval --mode=graph --create-beads --output=json plan.task.json
```

### On Success

- **Graph mode:** Creates 1 epic + N tasks + dependency links
- **Single task mode:** Creates 1 task issue
- All issues labeled `taskval-managed`
- Template metadata stored in the `design` field

### On Validation Failure

1. Read the error output carefully
2. Fix each reported issue in the JSON
3. Re-run validation
4. Repeat up to 3 times

### On bd Failure

- Partial results are printed (what was created before failure)
- Exit code 2 (not validation failure)
- No automatic rollback. Clean up manually with `bd close` or `bd delete`

## 8. After Tasks Are Created in Beads

```bash
# Find unblocked work
bd ready

# Claim a task
bd update <id> --status in_progress

# Do the work (respect files_scope from template metadata)

# Mark complete
bd close <id> --reason "Completed: <brief summary>"

# Sync at end of session
bd sync
```

**If new tasks emerge during work:**
1. Create a new `.task.json` file
2. Validate: `taskval --mode=graph <file>`
3. Create beads: `taskval --mode=graph --create-beads <file>`

## 9. Quick Reference

### Commands

| Action | Command |
|---|---|
| Validate | `taskval --mode=graph <file>` |
| Preview beads | `taskval --mode=graph --create-beads --dry-run <file>` |
| Create beads | `taskval --mode=graph --create-beads <file>` |
| Find work | `bd ready` |
| Claim task | `bd update <id> --status in_progress` |
| Complete task | `bd close <id> --reason "..."` |
| Sync | `bd sync` |

### Priority Mapping

| Template | bd Priority |
|---|---|
| critical | 0 |
| high | 1 |
| medium | 2 (default) |
| low | 3 |

### Estimate Mapping

| Template | Minutes |
|---|---|
| trivial | 15 |
| small | 60 |
| medium | 240 |
| large | 480 |
| unknown | omitted |

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Validation passed (and beads created if --create-beads) |
| 1 | Validation failed |
| 2 | Usage error, internal error, or bd failure |
