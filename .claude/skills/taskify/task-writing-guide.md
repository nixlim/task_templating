# Task Writing Guide

## Decomposition Principles

1. **Granularity:** Each task = 30 minutes to 4 hours of work
2. **Independence:** Each task should be completable without waiting on external input
3. **Testability:** Every task has acceptance criteria that can be verified mechanically
4. **Narrow scope:** `files_scope` should list only files the task creates or modifies

## Writing Good Goals

Goals describe what the system does when the task is complete, not what the agent does.

| Bad (activity) | Good (outcome) |
|---|---|
| "Try to add caching" | "The GetUser function returns cached results for repeated calls within 5 minutes" |
| "Explore database options" | "The schema migration creates users and sessions tables with foreign key constraints" |
| "Investigate slow queries" | "The ListOrders query executes in under 100ms for tables with 1M rows" |
| "Look into auth options" | "The /login endpoint returns a JWT token for valid credentials and 401 for invalid" |
| "To implement search" | "The Search function returns ranked results using BM25 + vector similarity" |

**Forbidden words:** try, explore, investigate, look into

## Writing Good Acceptance Criteria

Each criterion must be independently verifiable with a concrete expected value.

| Bad (vague) | Good (specific) |
|---|---|
| "It works correctly" | "CalculateTotal(100, Fixed(10)) returns 90.0" |
| "Output is good" | "JSON output is parseable by `jq` without errors" |
| "Functions properly" | "go test ./internal/pricing/... passes with 0 failures" |
| "Should work as expected" | "GET /api/users returns 200 with Content-Type: application/json" |
| "Is correct" | "Given input [3,1,2], output is [1,2,3]" |

## Prohibited Scope-Reduction Language (V11)

Goals and acceptance criteria must describe what the current task delivers, not what it defers. Deferral language hides scope decisions inside the task instead of making them explicit, so reviewers can't tell what is actually in scope.

**Forbidden phrases (whole-word, case-insensitive):**

`v1`, `v2`, `simplified version`, `static for now`, `hardcoded for now`, `future enhancement`, `placeholder`, `basic version`, `minimal implementation`, `will be wired later`, `dynamic in future phase`, `skip for now`

| Bad (deferral) | Good (concrete + explicit non-goal) |
|---|---|
| "Implement search v1 with basic ranking" | Goal: "Search returns up to 10 results ranked by BM25 score." Non-goal: "Vector similarity ranking (deferred to follow-up task)." |
| "Static for now, dynamic in future phase" | Goal: "GetConfig returns values from config.yaml." Non-goal: "Hot-reload of config (deferred)." |
| "Placeholder until real auth lands" | Either implement real auth, or split into a separate task. The placeholder shouldn't appear in goals or acceptance. |
| "Minimal implementation of caching" | Goal: "GetUser caches by user_id with 5-minute TTL using sync.Map." |
| "Will be wired later" | Either wire it (and assert it in acceptance), or move the wiring to a separate task with its own acceptance. |

**Rule:** if you find yourself writing one of these phrases, the scope decision belongs in `non_goals` or in a separate task — not buried in the goal or acceptance.

## Cross-Task Contracts (V12)

When task B's `input.source` reads from task A's output, the input and output types must match. The validator checks this whenever `input.source` mentions a declared dependency's `task_id` and an output name. Compatible types are exact-string matches, or one side wrapping the other in `optional<...>`.

**Example — types align (passes V12):**

```json
{
  "tasks": [
    {
      "task_id": "fetch-records",
      "outputs": [
        {"name": "records", "type": "list<UserRecord>", "constraints": "len > 0", "destination": "Return value"}
      ],
      "depends_on": {"status": "N/A", "reason": "Entry point"}
    },
    {
      "task_id": "rank-records",
      "inputs": [
        {"name": "records", "type": "list<UserRecord>", "constraints": "len > 0", "source": "Output 'records' of fetch-records"}
      ],
      "depends_on": ["fetch-records"]
    }
  ]
}
```

**Example — types mismatch (fails V12):**

```json
{
  "task_id": "rank-records",
  "inputs": [
    {"name": "records", "type": "string", "source": "Output 'records' of fetch-records"}
  ],
  "depends_on": ["fetch-records"]
}
```

`fetch-records.records` is `list<UserRecord>` but `rank-records.records` is declared as `string`. Either align the input type to `list<UserRecord>`, align the producer's output type, or add an explicit transform task between them.

**Practical guidance:**
- Reuse domain types from the graph-level `types` block instead of redeclaring shapes — that makes contracts self-aligning.
- When the consumer only needs a subset of the producer's fields, declare a separate domain type for the projection and add a transform task that produces it.
- `optional<T>` is one-directional: `T` flows into `optional<T>` and vice versa, but unrelated optionals do not match.

## Granularity Signals (V13)

V13 emits INFO-severity hints when the graph drifts away from Nyquist Compliance — task sizes that are too coarse to verify atomically, or graphs/milestones too sprawling to track.

**Heuristics:**

| Signal | Threshold | What it usually means |
|---|---|---|
| `estimate: "large"` on any task | exceeded immediately | The task bundles too many independently verifiable concerns |
| Graph total task count | > 20 | Over-decomposed — many tasks are likely 1-line edits that should be merged |
| Tasks in a single milestone | > 8 | The milestone's outcome is too broad to track as one unit |

**When to split (one task → many):**
- Acceptance criteria number more than ~5 unrelated assertions
- `files_scope` spans unrelated modules (e.g., DB layer + HTTP layer + frontend)
- Inputs come from more than one upstream task
- The goal sentence requires "and" between distinct outcomes
- Fix: extract one outcome per task, declare cross-task contracts via inputs/outputs

**When to merge (many tasks → one):**
- A pair of tasks always run together, share `files_scope`, and the second has only one upstream dependency (the first)
- A task's only acceptance criterion is "the previous task's output is correctly named"
- Tasks were created reflexively per file or per function instead of per outcome
- Fix: collapse into one task, list all files in `files_scope`, combine acceptance criteria

**When to split a milestone:**
- The milestone's tasks fall into two or more cohesive sub-outcomes that ship at different times
- A reviewer cannot describe the milestone's outcome in one sentence
- Fix: split into sub-milestones, optionally chained via `depends_on_milestones`

V13 findings are advisory (INFO), not blocking — but treat them as a prompt to reconsider decomposition before validation passes silently.

## Inputs and Outputs

Specify concrete types and constraints:

```json
{
  "inputs": [
    {"name": "query", "type": "string", "constraints": "len > 0, len <= 2000", "source": "CLI argument"},
    {"name": "limit", "type": "int", "constraints": "1 <= limit <= 100", "source": "CLI flag, default 10"}
  ],
  "outputs": [
    {"name": "results", "type": "list<SearchResult>", "constraints": "len <= limit", "destination": "Return value"}
  ]
}
```

## Priority and Estimate Mapping

### Priority -> bd numeric value
| Priority | bd Value | Use when |
|---|---|---|
| critical | 0 | Blocks all other work, production issue |
| high | 1 | Important feature, significant bug |
| medium | 2 | Standard work (default) |
| low | 3 | Nice to have, cleanup |

### Estimate -> minutes
| Estimate | Minutes | Guideline |
|---|---|---|
| trivial | 15 | Single function, config change |
| small | 60 | One file, straightforward logic |
| medium | 240 | Multiple files, moderate complexity |
| large | 480 | Cross-cutting, significant design |
| unknown | omitted | Research needed before estimating |

## Complete Single Task Example

```json
{
  "task_id": "calculate-discounted-total",
  "task_name": "Implement discount calculation for order totals",
  "goal": "Given a price and a discount (fixed amount or percentage), return the discounted total, guaranteed non-negative.",
  "inputs": [
    {"name": "price", "type": "f64", "constraints": "price > 0", "source": "Order record from database"},
    {"name": "discount", "type": "union(Fixed: f64, Percentage: f64(0..1))", "constraints": "Fixed: value >= 0; Percentage: 0 <= value <= 1", "source": "Promotion rules engine"}
  ],
  "outputs": [
    {"name": "total", "type": "f64", "constraints": "total >= 0", "destination": "Return value"}
  ],
  "acceptance": [
    "CalculateTotal(100.0, Fixed(10.0)) == 90.0",
    "CalculateTotal(100.0, Percentage(0.1)) == 90.0",
    "CalculateTotal(50.0, Fixed(60.0)) == 0.0 (clamped, not negative)",
    "Unit tests pass with 100% branch coverage"
  ],
  "depends_on": {"status": "N/A", "reason": "Pure function, no external dependencies"},
  "constraints": ["Pure function: no side effects, no I/O", "Result must be clamped to 0.0 minimum"],
  "files_scope": ["internal/pricing/discount.go", "internal/pricing/discount_test.go"],
  "error_cases": [
    {"condition": "price is zero or negative", "behavior": "Return error", "output": "invalid price: must be positive"},
    {"condition": "Fixed discount exceeds price", "behavior": "Clamp to 0.0", "output": "N/A (silent clamp)"}
  ],
  "priority": "medium",
  "estimate": "trivial"
}
```

## Complete Task Graph Example

```json
{
  "version": "0.1.0",
  "tasks": [
    {
      "task_id": "add-user-model",
      "task_name": "Create User database model and migration",
      "goal": "The users table exists with id, email, name, created_at columns and the User Go struct maps to it.",
      "inputs": [{"name": "schema", "type": "SQL DDL", "constraints": "valid PostgreSQL", "source": "Design doc"}],
      "outputs": [{"name": "migration", "type": "SQL file", "constraints": "idempotent", "destination": "migrations/"}],
      "acceptance": ["Migration creates users table with 4 columns", "User struct has json and db tags", "go build succeeds"],
      "depends_on": {"status": "N/A", "reason": "First task, no dependencies"},
      "constraints": ["Use sqlc for code generation"],
      "files_scope": ["internal/db/models.go", "migrations/001_users.sql"],
      "priority": "high",
      "estimate": "small"
    },
    {
      "task_id": "add-user-api",
      "task_name": "Implement CRUD API endpoints for users",
      "goal": "POST/GET/PUT/DELETE /api/users endpoints return correct responses with proper status codes.",
      "inputs": [{"name": "request", "type": "HTTP request", "constraints": "valid JSON body", "source": "HTTP client"}],
      "outputs": [{"name": "response", "type": "HTTP response", "constraints": "JSON with status", "destination": "HTTP client"}],
      "acceptance": ["POST /api/users with valid body returns 201", "GET /api/users/1 returns 200 with user JSON", "DELETE /api/users/1 returns 204"],
      "depends_on": ["add-user-model"],
      "constraints": ["Use chi router", "Return 404 for missing users"],
      "files_scope": ["internal/api/users.go", "internal/api/users_test.go"],
      "priority": "high",
      "estimate": "medium"
    }
  ]
}
```
