# Structured Task Template Spec - Quick Reference

## Field Definitions

### Required Fields (must be present on every task)

| Field | Type | Constraints |
|---|---|---|
| `task_id` | string | kebab-case, `^[a-z0-9]+(-[a-z0-9]+)*$`, max 60 chars, globally unique |
| `task_name` | string | Imperative phrase starting with a verb, max 80 chars |
| `goal` | string | Single testable sentence describing the end state. Forbidden words: "try", "explore", "investigate", "look into" |
| `inputs` | array of InputSpec | Each: `{name: string, type: string, constraints: string, source: string}` |
| `outputs` | array of OutputSpec | Each: `{name: string, type: string, constraints: string, destination: string}` |
| `acceptance` | array of string | Each criterion must be independently verifiable with concrete expected values |

### Contextual Fields (must provide value OR explicit N/A)

| Field | Type | N/A Pattern |
|---|---|---|
| `depends_on` | array of task_id strings | `{"status": "N/A", "reason": "..."}` |
| `constraints` | array of strings | `{"status": "N/A", "reason": "..."}` |
| `files_scope` | array of file paths/globs | `{"status": "N/A", "reason": "..."}` |

### Optional Fields

| Field | Type | Notes |
|---|---|---|
| `non_goals` | array of string | Explicit exclusions |
| `effects` | string or array of EffectSpec | `"None"` or `[{type: string, target: string}]` |
| `error_cases` | array of ErrorSpec | Each: `{condition: string, behavior: string, output: string}` |
| `priority` | string | One of: "critical", "high", "medium", "low" |
| `estimate` | string | One of: "trivial", "small", "medium", "large", "unknown" |
| `notes` | string | Free-form human-readable context |

## Type Vocabulary

### Primitives
`string`, `int`, `f64`, `bool`, `bytes`

### Compounds
`list<T>`, `set<T>`, `map<K,V>`, `optional<T>`, `tuple(T1, T2, ...)`

### Refined types (with constraints)
`string(pattern)`, `int(min..max)`, `f64(min..max)`

### Domain types
Define in the graph-level `types` block:
```json
{"types": {"ChunkResult": {"chunk_id": "int", "text": "string", "score": "f64"}}}
```

### Union types
`union(Variant1: T1, Variant2: T2)`

## JSON Schema Structure

### Single Task (task_node)
```json
{
  "task_id": "kebab-case-id",
  "task_name": "Imperative verb phrase",
  "goal": "Testable outcome sentence.",
  "inputs": [{"name": "...", "type": "...", "constraints": "...", "source": "..."}],
  "outputs": [{"name": "...", "type": "...", "constraints": "...", "destination": "..."}],
  "acceptance": ["Specific verifiable assertion"],
  "depends_on": ["other-task-id"] | {"status": "N/A", "reason": "..."},
  "constraints": ["constraint"] | {"status": "N/A", "reason": "..."},
  "files_scope": ["path/to/file.go"] | {"status": "N/A", "reason": "..."}
}
```

### Task Graph (task_graph)
```json
{
  "version": "0.1.0",
  "types": {},
  "defaults": {"constraints": [], "acceptance": [], "non_goals": []},
  "milestones": [{"name": "...", "task_ids": [...], "depends_on_milestones": [...]}],
  "tasks": [/* array of task_node objects */]
}
```

Only `version` and `tasks` are required at the graph level.

## Validation Rules Reference

| Rule | Severity | What it checks | Pass example | Fail example |
|---|---|---|---|---|
| SCHEMA | ERROR | JSON structure matches schema | All required fields present with correct types | Missing `goal` field |
| V2 | ERROR | Unique task_ids | Each task has a distinct ID | Two tasks with `task_id: "setup"` |
| V4 | ERROR | depends_on references exist | `depends_on: ["task-a"]` where task-a exists | `depends_on: ["nonexistent"]` |
| V5 | ERROR | No dependency cycles | A->B->C (linear) | A->B->A (cycle) |
| V6 | ERROR | Goal quality (no forbidden words) | "The function returns sorted results" | "Try to implement sorting" |
| V6 | WARNING | Goal not activity-phrased | "Search returns ranked results" | "To add search functionality" |
| V7 | WARNING | Acceptance not vague | "Given input 5, returns 25" | "it works correctly" |
| V9 | WARNING | Contextual fields present or N/A | `files_scope: ["a.go"]` or `{"status":"N/A","reason":"..."}` | Field entirely missing |
| V10 | WARNING | Implementation tasks have files_scope | Task named "Implement X" has files_scope | Task named "Implement X" missing files_scope |

## N/A Pattern for Contextual Fields

When a contextual field genuinely doesn't apply, mark it explicitly:
```json
{
  "depends_on": {"status": "N/A", "reason": "Standalone pure function, no external dependencies"},
  "constraints": {"status": "N/A", "reason": "No behavioral constraints beyond the acceptance criteria"},
  "files_scope": {"status": "N/A", "reason": "Documentation-only task, no code changes"}
}
```

The reason must explain **why** the field doesn't apply, not just state that it doesn't.
