# taskval CLI Command Reference

`taskval` validates task definitions against the Structured Task Template Specification. It performs two-tier validation: structural checks via JSON Schema, then semantic checks for cross-node consistency and content quality.

## Synopsis

```
taskval [flags] <file.json>
taskval [flags] -
```

## Flags

| Flag | Type | Default | Values | Description |
|---|---|---|---|---|
| `--mode` | string | `graph` | `task`, `graph` | `task`: validate a single task node. `graph`: validate a full task graph with milestones and dependencies. |
| `--output` | string | `text` | `text`, `json` | `text`: human/LLM-readable formatted output. `json`: machine-readable structured JSON. |
| `--help` | | | | Print usage information. |

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Validation passed. No ERROR-severity findings. Warnings may be present. |
| `1` | Validation failed. One or more ERROR-severity findings. |
| `2` | Usage error (bad flag, missing file, too many files) or internal error (schema compilation failure). |

## Input

`taskval` accepts exactly one positional argument: a file path or `-` for stdin.

### From a file

```bash
taskval --mode=task my_task.json
taskval --mode=graph my_graph.json
```

### From stdin

```bash
cat my_task.json | taskval --mode=task -
echo '{"version":"0.1.0","tasks":[...]}' | taskval -
```

---

## Commands by Example

Every example below shows the exact command, the complete output, and the exit code as verified against the `examples/` directory in this repository.

### 1. Help

```bash
$ taskval --help
```

```
taskval — Structured Task Template Spec validator

Usage:
  taskval [flags] <file.json>
  taskval [flags] -          (read from stdin)

Flags:
  -mode string
    	Validation mode: 'task' for a single task node, 'graph' for a full task graph (default "graph")
  -output string
    	Output format: 'text' for human/LLM-readable, 'json' for machine-readable (default "text")

Exit codes:
  0  Validation passed (no errors)
  1  Validation failed (errors found)
  2  Usage or internal error
```

Exit code: `0`

---

### 2. No Arguments

```bash
$ taskval
```

```
Error: no input file specified. Use 'taskval <file.json>' or 'taskval -' for stdin
```

Exit code: `2`

---

### 3. Too Many Arguments

```bash
$ taskval file1.json file2.json
```

```
Error: expected exactly one input file, got 2
```

Exit code: `2`

---

### 4. Nonexistent File

```bash
$ taskval nonexistent.json
```

```
Error: reading file 'nonexistent.json': open nonexistent.json: no such file or directory
```

Exit code: `2`

---

### 5. Invalid Mode Flag

```bash
$ taskval --mode=invalid examples/valid_single_task.json
```

```
Error: invalid mode 'invalid'. Must be 'task' or 'graph'.
```

Exit code: `2`

---

### 6. Invalid Output Flag

```bash
$ taskval --output=xml examples/valid_single_task.json
```

```
Error: invalid output format 'xml'. Must be 'text' or 'json'.
```

Exit code: `2`

---

### 7. Invalid JSON Input

```bash
$ echo "not json at all" | taskval --mode=task -
```

```
VALIDATION FAILED

Summary: 1 error(s), 0 warning(s), 0 info(s) across 0 task(s)

--- ERRORS (must fix) ---

  1. [ERROR] Rule SCHEMA
     Path:    format
     Problem: Invalid JSON format
```

Exit code: `1`

---

### 8. Valid Single Task (text output)

```bash
$ taskval --mode=task examples/valid_single_task.json
```

```
VALIDATION PASSED
  Tasks validated: 1
  No errors or warnings.
```

Exit code: `0`

---

### 9. Valid Single Task (JSON output)

```bash
$ taskval --mode=task --output=json examples/valid_single_task.json
```

```json
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

Exit code: `0`

---

### 10. Valid Task Graph (text output)

```bash
$ taskval examples/valid_task_graph.json
```

```
VALIDATION PASSED
  Tasks validated: 3
  No errors or warnings.
```

Exit code: `0`

Note: `--mode=graph` is the default and can be omitted.

---

### 11. Valid Task Graph (JSON output)

```bash
$ taskval --output=json examples/valid_task_graph.json
```

```json
{
  "valid": true,
  "stats": {
    "total_tasks": 3,
    "error_count": 0,
    "warning_count": 0,
    "info_count": 0
  }
}
```

Exit code: `0`

---

### 12. Invalid Task -- Tier 1 Schema Errors (text output)

Input file `examples/invalid_task.json` contains: uppercase `task_id`, `task_name` over 80 chars, empty `inputs`/`outputs` arrays, invalid `priority` and `estimate` enum values.

```bash
$ taskval --mode=task examples/invalid_task.json
```

```
VALIDATION FAILED

Summary: 9 error(s), 0 warning(s), 0 info(s) across 0 task(s)

--- ERRORS (must fix) ---

  1. [ERROR] Rule SCHEMA
     Path:    properties
     Problem: Properties 'estimate', 'inputs', 'outputs', 'priority', 'task_id',
              'task_name' do not match their schemas

  2. [ERROR] Rule SCHEMA
     Path:    /estimate/enum
     Problem: Value huge should be one of the allowed values: trivial, small,
              medium, large, unknown

  3. [ERROR] Rule SCHEMA
     Path:    /task_name/maxLength
     Problem: Value should be at most 80 characters

  4. [ERROR] Rule SCHEMA
     Path:    /outputs/minItems
     Problem: Value should have at least 1 items

  5. [ERROR] Rule SCHEMA
     Path:    /task_id/pattern
     Problem: Value does not match the required pattern ^[a-z0-9]+(-[a-z0-9]+)*$
     Fix:     Add the missing required field at '/task_id/pattern'. Check the
              spec's Quick Reference (Appendix A) for the expected format.

  6. [ERROR] Rule SCHEMA
     Path:    /inputs/minItems
     Problem: Value should have at least 1 items

  7. [ERROR] Rule SCHEMA
     Path:    /depends_on/$ref
     Problem: Value does not match the reference schema

  8. [ERROR] Rule SCHEMA
     Path:    /depends_on/type
     Problem: Value is array but should be object

  9. [ERROR] Rule SCHEMA
     Path:    /priority/enum
     Problem: Value urgent should be one of the allowed values: critical, high,
              medium, low
```

Exit code: `1`

Tier 2 semantic checks are skipped because Tier 1 failed.

---

### 13. Invalid Task -- Tier 1 Schema Errors (JSON output)

```bash
$ taskval --mode=task --output=json examples/invalid_task.json
```

```json
{
  "valid": false,
  "errors": [
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "properties",
      "message": "Properties 'estimate', 'inputs', 'outputs', 'priority', 'task_id', 'task_name' do not match their schemas"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/priority/enum",
      "message": "Value urgent should be one of the allowed values: critical, high, medium, low"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/task_name/maxLength",
      "message": "Value should be at most 80 characters"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/inputs/minItems",
      "message": "Value should have at least 1 items"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/depends_on/$ref",
      "message": "Value does not match the reference schema"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/depends_on/type",
      "message": "Value is array but should be object"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/estimate/enum",
      "message": "Value huge should be one of the allowed values: trivial, small, medium, large, unknown"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/outputs/minItems",
      "message": "Value should have at least 1 items"
    },
    {
      "rule": "SCHEMA",
      "severity": "ERROR",
      "path": "/task_id/pattern",
      "message": "Value does not match the required pattern ^[a-z0-9]+(-[a-z0-9]+)*$",
      "suggestion": "Add the missing required field at '/task_id/pattern'. Check the spec's Quick Reference (Appendix A) for the expected format."
    }
  ],
  "stats": {
    "total_tasks": 0,
    "error_count": 9,
    "warning_count": 0,
    "info_count": 0
  }
}
```

Exit code: `1`

---

### 14. Invalid Graph -- Tier 2 Semantic Errors (text output)

Input file `examples/invalid_semantic.json` passes Tier 1 but contains: a dependency cycle between `task-a` and `task-b`, a dangling reference to `nonexistent-task`, forbidden words in a goal, vague acceptance criteria, and missing contextual fields.

```bash
$ taskval examples/invalid_semantic.json
```

```
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
     Value:   "nonexistent-task"

  2. [ERROR] Rule V5
     Path:    tasks
     Problem: Dependency graph contains a cycle. 2 task(s) are involved:
              [task-a, task-b]. A valid task graph must be a DAG (Directed
              Acyclic Graph).
     Fix:     Review the depends_on fields of the listed tasks. Break the cycle
              by removing one dependency or decomposing a task into sub-tasks.
     Value:   "task-a, task-b"

  3. [ERROR] Rule V6
     Path:    tasks[0].goal
     Problem: Goal contains the forbidden word/phrase 'try'. Goals must describe
              testable outcomes, not activities or explorations.
     Fix:     Rewrite the goal as a concrete, testable outcome. Instead of 'try
              ...', describe what the system does when the task is complete.
              Example: 'The function returns X when given Y.'
     Value:   "Try to explore adding feature A and investigate options for it"

  4. [ERROR] Rule V6
     Path:    tasks[0].goal
     Problem: Goal contains the forbidden word/phrase 'explore'. Goals must
              describe testable outcomes, not activities or explorations.
     Fix:     Rewrite the goal as a concrete, testable outcome. Instead of
              'explore ...', describe what the system does when the task is
              complete. Example: 'The function returns X when given Y.'
     Value:   "Try to explore adding feature A and investigate options for it"

  5. [ERROR] Rule V6
     Path:    tasks[0].goal
     Problem: Goal contains the forbidden word/phrase 'investigate'. Goals must
              describe testable outcomes, not activities or explorations.
     Fix:     Rewrite the goal as a concrete, testable outcome. Instead of
              'investigate ...', describe what the system does when the task is
              complete. Example: 'The function returns X when given Y.'
     Value:   "Try to explore adding feature A and investigate options for it"

--- WARNINGS (should fix) ---

  6. [WARNING] Rule V6
     Path:    tasks[1].goal
     Problem: Goal starts with 'To ...' which suggests an activity rather than a
              testable outcome.
     Fix:     Rewrite as a state-of-the-world assertion. Example: Instead of 'To
              add search functionality', write 'The Search() function returns
              ranked results from Weaviate hybrid search.'
     Value:   "To add feature B that does something useful"

  7. [WARNING] Rule V7
     Path:    tasks[0].acceptance[0]
     Problem: Acceptance criterion contains the vague phrase 'works correctly'.
              Criteria must be independently verifiable with concrete expected
              values.
     Fix:     Replace with a specific assertion. Example: Instead of 'it works
              correctly', write 'Given input "test", the function returns
              ["result1", "result2"] with status 200.'
     Value:   "it works correctly"

  8. [WARNING] Rule V7
     Path:    tasks[0].acceptance[1]
     Problem: Acceptance criterion contains the vague phrase 'looks right'.
              Criteria must be independently verifiable with concrete expected
              values.
     Fix:     Replace with a specific assertion. Example: Instead of 'it works
              correctly', write 'Given input "test", the function returns
              ["result1", "result2"] with status 200.'
     Value:   "output looks right and is fine"

  9. [WARNING] Rule V7
     Path:    tasks[0].acceptance[1]
     Problem: Acceptance criterion contains the vague phrase 'is fine'. Criteria
              must be independently verifiable with concrete expected values.
     Fix:     Replace with a specific assertion. Example: Instead of 'it works
              correctly', write 'Given input "test", the function returns
              ["result1", "result2"] with status 200.'
     Value:   "output looks right and is fine"

  10. [WARNING] Rule V9
     Path:    tasks[2].constraints
     Problem: Contextual field 'constraints' is missing from task 'task-c'.
              Contextual fields should be explicitly present or set to
              {"status": "N/A", "reason": "..."}.
     Fix:     Either provide a value for 'constraints' or explicitly mark it as
              not applicable: {"status": "N/A", "reason": "your justification
              here"}.

  11. [WARNING] Rule V9
     Path:    tasks[2].files_scope
     Problem: Contextual field 'files_scope' is missing from task 'task-c'.
              Contextual fields should be explicitly present or set to
              {"status": "N/A", "reason": "..."}.
     Fix:     Either provide a value for 'files_scope' or explicitly mark it as
              not applicable: {"status": "N/A", "reason": "your justification
              here"}.

  12. [WARNING] Rule V10
     Path:    tasks[2].files_scope
     Problem: Task 'task-c' appears to be an implementation task (name starts
              with an implementation verb) but has no files_scope defined.
     Fix:     Add a files_scope listing the files the agent should create or
              modify. This prevents unintended changes to other parts of the
              codebase.
```

Exit code: `1`

---

### 15. Invalid Graph -- Tier 2 Semantic Errors (JSON output)

```bash
$ taskval --output=json examples/invalid_semantic.json
```

```json
{
  "valid": false,
  "errors": [
    {
      "rule": "V4",
      "severity": "ERROR",
      "path": "tasks[2].depends_on",
      "message": "Task 'task-c' depends on 'nonexistent-task', but no task with that task_id exists in the graph.",
      "suggestion": "Either add a task with task_id 'nonexistent-task' to the graph, or remove 'nonexistent-task' from the depends_on list of task 'task-c'.",
      "context": "nonexistent-task"
    },
    {
      "rule": "V5",
      "severity": "ERROR",
      "path": "tasks",
      "message": "Dependency graph contains a cycle. 2 task(s) are involved: [task-a, task-b]. A valid task graph must be a DAG (Directed Acyclic Graph).",
      "suggestion": "Review the depends_on fields of the listed tasks. Break the cycle by removing one dependency or decomposing a task into sub-tasks.",
      "context": "task-a, task-b"
    },
    {
      "rule": "V6",
      "severity": "ERROR",
      "path": "tasks[0].goal",
      "message": "Goal contains the forbidden word/phrase 'try'. Goals must describe testable outcomes, not activities or explorations.",
      "suggestion": "Rewrite the goal as a concrete, testable outcome. Instead of 'try ...', describe what the system does when the task is complete. Example: 'The function returns X when given Y.'",
      "context": "Try to explore adding feature A and investigate options for it"
    },
    ...
  ],
  "stats": {
    "total_tasks": 3,
    "error_count": 5,
    "warning_count": 7,
    "info_count": 0
  }
}
```

Exit code: `1`

---

### 16. Stdin Pipe

```bash
$ cat examples/valid_single_task.json | taskval --mode=task -
```

```
VALIDATION PASSED
  Tasks validated: 1
  No errors or warnings.
```

Exit code: `0`

---

### 17. Wrong Mode (graph schema applied to a single task)

Using `--mode=graph` on a single task file produces schema errors because the document lacks the required `version` and `tasks` envelope:

```bash
$ taskval --mode=graph examples/valid_single_task.json
```

```
VALIDATION FAILED

Summary: 21 error(s), 0 warning(s), 0 info(s) across 0 task(s)

--- ERRORS (must fix) ---

  1. [ERROR] Rule SCHEMA
     Path:    required
     Problem: Required properties 'version', 'tasks' are missing

  2. [ERROR] Rule SCHEMA
     Path:    additionalProperties
     Problem: Additional properties 'outputs', 'notes', 'non_goals',
              'error_cases', 'priority', 'estimate', 'inputs', 'constraints',
              'effects', 'task_id', 'task_name', 'goal', 'acceptance',
              'depends_on', 'files_scope' do not match the schema
  ...
```

Exit code: `1`

---

## Validation Rules Reference

### Tier 1 Rules (JSON Schema)

All reported with rule ID `SCHEMA` and severity `ERROR`.

| What | Schema keyword | Example error message |
|---|---|---|
| Missing required field | `required` | `Required properties 'task_id', 'goal' are missing` |
| Wrong type | `type` | `Value is array but should be object` |
| Pattern mismatch | `pattern` | `Value does not match the required pattern ^[a-z0-9]+(-[a-z0-9]+)*$` |
| Enum mismatch | `enum` | `Value urgent should be one of the allowed values: critical, high, medium, low` |
| String too long | `maxLength` | `Value should be at most 80 characters` |
| Array too short | `minItems` | `Value should have at least 1 items` |
| Unknown field | `additionalProperties` | `Additional properties 'foo' do not match the schema` |
| oneOf mismatch | `oneOf` / `$ref` | `Value does not match the reference schema` |

### Tier 2 Rules (Semantic)

| Rule ID | Severity | What it checks |
|---|---|---|
| V2 | ERROR | Every `task_id` is unique across all tasks in the graph |
| V4 | ERROR | Every ID in `depends_on` references an existing task |
| V5 | ERROR | No task depends on itself |
| V5 | ERROR | The dependency graph is acyclic (DAG). Uses Kahn's algorithm. |
| V6 | ERROR | `goal` does not contain: "try", "explore", "investigate", "look into" |
| V6 | WARNING | `goal` does not start with "To ..." |
| V7 | WARNING | `acceptance` criteria do not contain: "works correctly", "is correct", "is good", "looks right", "properly", "as expected", "should work", "is fine" |
| V9 | WARNING | Contextual fields (`depends_on`, `constraints`, `files_scope`) are present or explicitly N/A |
| V10 | WARNING | Implementation tasks (name starts with implement/add/fix/create/build/write) have `files_scope` |
| MILESTONE | ERROR | No duplicate milestone names; all `task_ids` and `depends_on_milestones` references resolve |

---

## Output Format Details

### Text Output Structure

**Validation passed (clean):**

```
VALIDATION PASSED
  Tasks validated: <N>
  No errors or warnings.
```

**Validation passed (with warnings):**

```
VALIDATION PASSED (with warnings)

Summary: 0 error(s), <W> warning(s), <I> info(s) across <N> task(s)

--- WARNINGS (should fix) ---
  1. [WARNING] Rule <ID>
     Path:    <json.path>
     Problem: <description>
     Fix:     <suggestion>
     Value:   "<offending value>"
```

**Validation failed:**

```
VALIDATION FAILED

Summary: <E> error(s), <W> warning(s), <I> info(s) across <N> task(s)

--- ERRORS (must fix) ---
  1. [ERROR] Rule <ID>
     Path:    <json.path>
     Problem: <description>
     Fix:     <suggestion>
     Value:   "<offending value>"

--- WARNINGS (should fix) ---
  ...
```

### JSON Output Structure

```json
{
  "valid": true|false,
  "errors": [
    {
      "rule": "V5",
      "severity": "ERROR",
      "path": "tasks",
      "message": "Dependency graph contains a cycle...",
      "suggestion": "Review the depends_on fields...",
      "context": "task-a, task-b"
    }
  ],
  "stats": {
    "total_tasks": 3,
    "error_count": 1,
    "warning_count": 2,
    "info_count": 0
  }
}
```

**JSON error object fields:**

| Field | Type | Always present | Description |
|---|---|---|---|
| `rule` | string | yes | Rule ID: `SCHEMA`, `V2`, `V4`, `V5`, `V6`, `V7`, `V9`, `V10`, `MILESTONE` |
| `severity` | string | yes | `ERROR`, `WARNING`, or `INFO` |
| `path` | string | yes | JSON path to the problematic field, e.g. `tasks[0].goal` |
| `message` | string | yes | Human/LLM-readable problem description |
| `suggestion` | string | no | Actionable fix recommendation (omitted if empty) |
| `context` | string | no | The offending value, truncated to 120 chars (omitted if empty) |
