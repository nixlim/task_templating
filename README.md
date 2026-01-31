# Task Templating

A machine-readable task specification format for AI coding agents, with a Go CLI validator (`taskval`) that ensures tasks are structurally complete and semantically consistent before execution.

## Problem

Natural language task descriptions are ambiguous. "Implement search" leaves scope, constraints, acceptance criteria, and dependencies implicit. AI coding agents interpret these gaps with assumptions that diverge from intent, causing rework, scope creep, and defects.

## Solution

A structured task format (defined in [STRUCTURED_TEMPLATE_SPEC.md](STRUCTURED_TEMPLATE_SPEC.md)) with a two-tier validation pipeline:

```
TASK (natural language)
  -> LLM compiles to JSON
  -> Tier 1: JSON Schema (structural)
  -> Tier 2: Semantic checks (cross-node)
  -> PASS: ready for execution
  -> FAIL: LLM reads errors, refines, re-validates
```

## Project Structure

```
task_templating/
├── STRUCTURED_TEMPLATE_SPEC.md          # The specification (authoritative)
├── CLI_COMMAND_REFERENCE.md             # Full CLI reference with examples
├── schemas/
│   ├── task_node.schema.json            # JSON Schema for a single task
│   └── task_graph.schema.json           # JSON Schema for a task graph
├── cmd/taskval/
│   └── main.go                          # CLI entry point
├── internal/validator/
│   ├── types.go                         # ValidationError, ValidationResult
│   ├── models.go                        # TaskGraph, TaskNode, InputSpec, etc.
│   ├── schema.go                        # Tier 1: JSON Schema validation
│   ├── semantic.go                      # Tier 2: DAG, references, goal quality
│   ├── validate.go                      # Orchestrator (Tier 1 then Tier 2)
│   ├── validate_test.go                 # Unit tests
│   └── schemas/                         # Embedded copies (go:embed)
│       ├── task_node.schema.json
│       └── task_graph.schema.json
└── examples/
    ├── valid_single_task.json           # Passes both tiers
    ├── valid_task_graph.json            # Multi-task graph, passes both tiers
    ├── invalid_task.json                # Fails Tier 1 (schema errors)
    └── invalid_semantic.json            # Passes Tier 1, fails Tier 2
```

## Quick Start

### Install

```bash
go install github.com/foundry-zero/task-templating/cmd/taskval@latest
```

Or build from source:

```bash
git clone git@github.com:nixlim/task_templating.git
cd task_templating
go build -o taskval ./cmd/taskval/
```

### Validate a single task

```bash
$ taskval --mode=task examples/valid_single_task.json
VALIDATION PASSED
  Tasks validated: 1
  No errors or warnings.
```

### Validate a task graph

```bash
$ taskval examples/valid_task_graph.json
VALIDATION PASSED
  Tasks validated: 3
  No errors or warnings.
```

### See validation errors (structural)

```bash
$ taskval --mode=task examples/invalid_task.json
VALIDATION FAILED

Summary: 9 error(s), 0 warning(s), 0 info(s) across 0 task(s)

--- ERRORS (must fix) ---

  1. [ERROR] Rule SCHEMA
     Path:    /task_id/pattern
     Problem: Value does not match the required pattern ^[a-z0-9]+(-[a-z0-9]+)*$
     Fix:     Add the missing required field at '/task_id/pattern'. Check the
              spec's Quick Reference (Appendix A) for the expected format.

  2. [ERROR] Rule SCHEMA
     Path:    /task_name/maxLength
     Problem: Value should be at most 80 characters

  3. [ERROR] Rule SCHEMA
     Path:    /priority/enum
     Problem: Value urgent should be one of the allowed values: critical, high,
              medium, low

  4. [ERROR] Rule SCHEMA
     Path:    /inputs/minItems
     Problem: Value should have at least 1 items
  ...
```

### See validation errors (semantic)

```bash
$ taskval examples/invalid_semantic.json
VALIDATION FAILED

Summary: 5 error(s), 7 warning(s), 0 info(s) across 3 task(s)

--- ERRORS (must fix) ---

  1. [ERROR] Rule V4
     Path:    tasks[2].depends_on
     Problem: Task 'task-c' depends on 'nonexistent-task', but no task with that
              task_id exists in the graph.
     Fix:     Either add a task with task_id 'nonexistent-task' to the graph, or
              remove 'nonexistent-task' from the depends_on list of task
              'task-c'.

  2. [ERROR] Rule V5
     Path:    tasks
     Problem: Dependency graph contains a cycle. 2 task(s) are involved:
              [task-a, task-b]. A valid task graph must be a DAG (Directed
              Acyclic Graph).
     Fix:     Review the depends_on fields of the listed tasks. Break the cycle
              by removing one dependency or decomposing a task into sub-tasks.

  3. [ERROR] Rule V6
     Path:    tasks[0].goal
     Problem: Goal contains the forbidden word/phrase 'try'. Goals must describe
              testable outcomes, not activities or explorations.
     Fix:     Rewrite the goal as a concrete, testable outcome. Instead of 'try
              ...', describe what the system does when the task is complete.
  ...

--- WARNINGS (should fix) ---

  6. [WARNING] Rule V7
     Path:    tasks[0].acceptance[0]
     Problem: Acceptance criterion contains the vague phrase 'works correctly'.
              Criteria must be independently verifiable with concrete expected
              values.
     Fix:     Replace with a specific assertion. Example: Instead of 'it works
              correctly', write 'Given input "test", the function returns
              ["result1", "result2"] with status 200.'
  ...
```

### JSON output for programmatic consumption

```bash
$ taskval --mode=task --output=json examples/valid_single_task.json
{
  "valid": true,
  "stats": {
    "total_tasks": 1,
    "error_count": 0,
    "warning_count": 0,
    "info_count": 0
  }
}
```

### Read from stdin

```bash
cat my_task.json | taskval --mode=task -
```

## Two-Tier Validation

### Tier 1: Structural (JSON Schema)

Deterministic checks enforced by JSON Schema Draft 2020-12 via [kaptinlin/jsonschema](https://github.com/kaptinlin/jsonschema):

- Required fields present (`task_id`, `task_name`, `goal`, `inputs`, `outputs`, `acceptance`)
- Field types correct (string, array, object)
- `task_id` matches kebab-case pattern `^[a-z0-9]+(-[a-z0-9]+)*$`
- `task_name` length between 5-80 characters
- `priority` is one of: `critical`, `high`, `medium`, `low`
- `estimate` is one of: `trivial`, `small`, `medium`, `large`, `unknown`
- `inputs` and `outputs` arrays are non-empty
- Each `InputSpec` has `name`, `type`, `constraints`, `source`
- Each `OutputSpec` has `name`, `type`, `constraints`, `destination`
- Contextual fields accept either an array or `{"status": "N/A", "reason": "..."}`
- `effects` accepts either an array of `EffectSpec` or the string `"None"`

### Tier 2: Semantic (Programmatic)

Cross-node and content-quality checks that JSON Schema cannot express. Tier 2 runs only if Tier 1 passes.

| Rule | Check | Severity |
|---|---|---|
| V2 | Duplicate `task_id` detection | ERROR |
| V4 | Dangling `depends_on` references | ERROR |
| V5 | Self-dependencies | ERROR |
| V5 | Dependency graph cycle detection (Kahn's algorithm) | ERROR |
| V6 | Goal contains forbidden words: "try", "explore", "investigate", "look into" | ERROR |
| V6 | Goal starts with "To ..." (activity phrasing) | WARNING |
| V7 | Acceptance criteria contain vague phrases: "works correctly", "is correct", "is good", "looks right", "properly", "as expected", "should work", "is fine" | WARNING |
| V9 | Contextual fields (`depends_on`, `constraints`, `files_scope`) missing without N/A | WARNING |
| V10 | Implementation tasks missing `files_scope` | WARNING |
| MILESTONE | Duplicate milestone names, dangling task/milestone references | ERROR |

## Task JSON Format

A single task node in JSON:

```json
{
  "task_id": "calculate-discounted-total",
  "task_name": "Implement discount calculation for order totals",
  "goal": "Given a price and a discount, return the discounted total, guaranteed non-negative.",
  "inputs": [
    {
      "name": "price",
      "type": "f64",
      "constraints": "price > 0",
      "source": "Order record from database"
    }
  ],
  "outputs": [
    {
      "name": "total",
      "type": "f64",
      "constraints": "total >= 0",
      "destination": "Return value"
    }
  ],
  "acceptance": [
    "CalculateTotal(100.0, Fixed(10.0)) == 90.0",
    "CalculateTotal(50.0, Fixed(60.0)) == 0.0 (clamped, not negative)"
  ],
  "depends_on": {"status": "N/A", "reason": "Pure function, no external dependencies"},
  "constraints": ["Pure function: no side effects, no I/O"],
  "files_scope": ["internal/pricing/discount.go", "internal/pricing/discount_test.go"],
  "priority": "medium",
  "estimate": "trivial"
}
```

A task graph wraps multiple tasks with optional metadata:

```json
{
  "version": "0.1.0",
  "types": {
    "ChunkResult": {"chunk_id": "int", "text": "string", "score": "f64"}
  },
  "defaults": {
    "constraints": ["All code must pass go vet and go fmt"],
    "acceptance": ["go test ./... passes"]
  },
  "milestones": [
    {
      "name": "M1 - Core",
      "task_ids": ["task-a", "task-b"]
    },
    {
      "name": "M2 - Features",
      "depends_on_milestones": ["M1 - Core"],
      "task_ids": ["task-c"]
    }
  ],
  "tasks": [ ... ]
}
```

See [STRUCTURED_TEMPLATE_SPEC.md](STRUCTURED_TEMPLATE_SPEC.md) for the full specification with field definitions, type vocabulary, constraint language, and complete examples.

## LLM Agent Feedback Loop

The intended workflow for LLM-compiled tasks:

1. Agent receives a natural language task description.
2. Agent compiles it to the JSON format defined by the spec.
3. Agent runs `taskval --mode=task --output=json task.json`.
4. If valid: the task is ready for execution.
5. If invalid: the agent reads the structured error output (each error includes `rule`, `path`, `message`, `suggestion`) and revises the identified fields.
6. Agent re-runs `taskval`. Repeat until validation passes.
7. Maximum 3 refinement attempts. If the task still fails, escalate to the user.

## Running Tests

```bash
go test ./... -v
```

```
=== RUN   TestValidSingleTask
--- PASS: TestValidSingleTask (0.04s)
=== RUN   TestInvalidTaskID
--- PASS: TestInvalidTaskID (0.01s)
=== RUN   TestCycleDetection
--- PASS: TestCycleDetection (0.00s)
=== RUN   TestGoalForbiddenWords
--- PASS: TestGoalForbiddenWords (0.03s)
=== RUN   TestDanglingDependencyReference
--- PASS: TestDanglingDependencyReference (0.00s)
=== RUN   TestAcceptanceVagueness
--- PASS: TestAcceptanceVagueness (0.00s)
PASS
ok  	github.com/foundry-zero/task-templating/internal/validator	1.096s
```

## Dependencies

| Dependency | Purpose |
|---|---|
| [kaptinlin/jsonschema](https://github.com/kaptinlin/jsonschema) v0.6.9 | JSON Schema Draft 2020-12 validation |

No other direct dependencies. The Go standard library provides everything else (JSON parsing, regex, embedded filesystem).

## License

See repository for license details.
