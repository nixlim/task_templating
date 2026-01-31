# Structured Task Template Specification

**Version:** 0.1.0
**Status:** Draft
**Purpose:** Machine-readable task specification format for AI coding agents

---

## 1. Preamble

### 1.1 Problem Statement

Natural language task descriptions are ambiguous. Phrases like "implement search" leave scope, constraints, acceptance criteria, and dependencies implicit. AI coding agents interpret these gaps with assumptions that frequently diverge from the author's intent, resulting in rework, scope creep, and defects.

### 1.2 Design Goals

1. **Eliminate ambiguity** — every task field has defined semantics; agents never guess.
2. **Enforce completeness** — required fields ensure critical information is always present.
3. **Enable validation** — the template itself is machine-checkable before execution begins.
4. **Remain human-writable** — no compiler or tooling required to author a task; any text editor suffices.
5. **Stay language-agnostic** — the format describes *what* to build, not *how* in a specific language.

### 1.3 Scope

This specification defines the schema for individual **Task Nodes** and the rules for composing them into a **Task Graph**. It does not define a runtime, a compiler, or an execution engine. It is a documentation format that AI agents are instructed to parse and follow.

---

## 2. Schema Overview

A Task Node is a block of structured text. Fields are written as `FIELD_NAME: value`. Multi-value fields use YAML-style lists. The ordering of fields within a node is fixed (as defined below) to aid both human scanning and machine parsing.

### 2.1 Notation Conventions

| Convention | Meaning |
|---|---|
| **REQUIRED** | Field must be present. Omission is a validation error. |
| **CONTEXTUAL** | Required when applicable; omission requires explicit `N/A` with justification. |
| **OPTIONAL** | May be omitted without justification. |
| `<angle brackets>` | Placeholder for a value. |
| `[square brackets]` | List of values. |
| `{curly braces}` | Structured sub-object. |
| `\|` | Logical OR (one of the listed values). |

---

## 3. Task Node Schema

### 3.1 Required Fields

#### `TASK_ID`

- **Type:** `string`
- **Format:** Kebab-case, globally unique within the project. Pattern: `^[a-z0-9]+(-[a-z0-9]+)*$`
- **Max length:** 60 characters
- **Semantics:** Immutable identifier. Once assigned, never reused even if the task is deleted.
- **Example:** `weaviate-hybrid-search`, `cli-export-markdown`

#### `TASK_NAME`

- **Type:** `string`
- **Format:** Short imperative phrase. Max 80 characters.
- **Semantics:** Human-readable label. Begins with a verb (Implement, Add, Fix, Refactor, Remove, Extract, Migrate).
- **Example:** `Implement hybrid BM25 + vector search via Weaviate`

#### `GOAL`

- **Type:** `string`
- **Format:** Single sentence. Must describe a **testable outcome**, not an activity.
- **Semantics:** Defines *what success looks like*. An agent that achieves the GOAL has completed the task, even if the approach differs from what the author envisioned.
- **Validation rule:** Must not contain the words "try", "explore", "investigate", or "look into" — these indicate the task is underspecified and should be decomposed further.
- **Good example:** `The CLI accepts a --format flag that outputs extraction results as valid Markdown or JSON to stdout.`
- **Bad example:** `Look into adding export functionality.`

#### `INPUTS`

- **Type:** `list<InputSpec>`
- **Format:** Each entry is a structured object:
  ```
  - name: <identifier>
    type: <Type>
    constraints: <Constraint | "none">
    source: <description of where this value comes from>
  ```
- **Semantics:** The data or preconditions the task requires to begin. For tasks with no programmatic inputs (e.g., refactoring), describe the preconditions instead.
- **Example:**
  ```
  INPUTS:
    - name: query
      type: string
      constraints: len > 0, len <= 2000
      source: User-provided CLI argument
    - name: limit
      type: int
      constraints: 1 <= limit <= 100
      source: CLI flag --limit, default 10
  ```

#### `OUTPUTS`

- **Type:** `list<OutputSpec>`
- **Format:** Each entry is a structured object:
  ```
  - name: <identifier>
    type: <Type>
    constraints: <Constraint | "none">
    destination: <description of where this value goes>
  ```
- **Semantics:** The artifacts or state changes the task produces. Every output must be observable (file on disk, return value, database row, CLI output).
- **Example:**
  ```
  OUTPUTS:
    - name: results
      type: list<ChunkResult>
      constraints: len <= limit, each item has score > 0.0
      destination: Return value from Search() function
  ```

#### `ACCEPTANCE`

- **Type:** `list<string>`
- **Format:** Each entry is a testable assertion. Must be phrased as a verifiable statement, not a subjective judgment.
- **Semantics:** The agent must satisfy **all** acceptance criteria to consider the task complete. These map directly to test cases or manual verification steps.
- **Validation rule:** Each criterion must be independently verifiable. "It works correctly" is not acceptable. "Given input X, output equals Y" is acceptable.
- **Example:**
  ```
  ACCEPTANCE:
    - "aqe extract 'machine learning' --limit 5" returns at most 5 results, each with a Harvard citation
    - Empty query string returns exit code 1 and a usage error on stderr
    - Results are sorted by descending relevance score
    - Each result contains: quote text, page number, in-text citation, full reference
    - Integration test passes: tests/integration/search_test.go
  ```

### 3.2 Contextual Fields

These fields are required when applicable. If genuinely not applicable, write `N/A` with a brief justification.

#### `DEPENDS_ON`

- **Type:** `list<TASK_ID>`
- **Format:** References to other Task Nodes by their `TASK_ID`.
- **Semantics:** This task cannot begin until all listed dependencies are completed. Defines a DAG (directed acyclic graph). Cycles are a validation error.
- **Validation rule:** Every referenced TASK_ID must exist in the task graph.
- **Example:**
  ```
  DEPENDS_ON:
    - sqlite-schema-init
    - weaviate-client-setup
  ```
- **N/A example:**
  ```
  DEPENDS_ON: N/A (standalone utility function with no external dependencies)
  ```

#### `CONSTRAINTS`

- **Type:** `list<string>`
- **Format:** Each entry is a hard rule or architectural boundary.
- **Semantics:** Non-negotiable requirements that restrict *how* the task is implemented. Violating a constraint means the task is not complete, even if ACCEPTANCE criteria pass.
- **Example:**
  ```
  CONSTRAINTS:
    - Must use the official weaviate-go-client/v4; no third-party wrappers
    - No abstraction layer over the Weaviate client (use directly)
    - Quote text must come from SQLite, never from LLM generation
    - Function must be safe for concurrent use (no shared mutable state)
  ```

#### `FILES_SCOPE`

- **Type:** `list<string>`
- **Format:** File paths or glob patterns relative to project root.
- **Semantics:** The set of files the agent is expected to create or modify. Files outside this scope should not be touched without explicit justification. New files not listed here are acceptable only if they are test files or directly required by a listed file.
- **Example:**
  ```
  FILES_SCOPE:
    - internal/search/weaviate.go
    - internal/search/weaviate_test.go
    - internal/models/chunk.go
  ```

### 3.3 Optional Fields

#### `NON_GOALS`

- **Type:** `list<string>`
- **Semantics:** Explicit exclusions. Things the agent might reasonably attempt but should not. Prevents scope creep.
- **Example:**
  ```
  NON_GOALS:
    - Do not implement caching of search results
    - Do not add pagination (will be a separate task)
    - Do not modify the CLI command structure
  ```

#### `EFFECTS`

- **Type:** `list<EffectSpec>`
- **Format:** Each entry declares a side effect:
  ```
  - type: <DB.Read | DB.Write | Network.Out | Filesystem.Write | Subprocess | None>
    target: <description>
  ```
- **Semantics:** Declares what external state the implementation will touch. Enables reviewers and agents to assess blast radius.
- **Example:**
  ```
  EFFECTS:
    - type: Network.Out
      target: Weaviate at localhost:8080
    - type: DB.Read
      target: SQLite chunks table
  ```

#### `ERROR_CASES`

- **Type:** `list<ErrorSpec>`
- **Format:**
  ```
  - condition: <when this happens>
    behavior: <what the code should do>
    output: <what the user sees>
  ```
- **Semantics:** Expected failure modes. Each error case should result in a deterministic, user-appropriate response.
- **Example:**
  ```
  ERROR_CASES:
    - condition: Weaviate service unreachable
      behavior: Return wrapped error with timeout context
      output: "Error: search service unavailable. Is Weaviate running? (docker-compose up -d)"
    - condition: Query returns zero results
      behavior: Return empty list (not an error)
      output: "No matching quotes found for '<query>'"
  ```

#### `PRIORITY`

- **Type:** `enum(critical, high, medium, low)`
- **Semantics:** Execution priority when multiple tasks are unblocked simultaneously.

#### `ESTIMATE`

- **Type:** `enum(trivial, small, medium, large, unknown)`
- **Semantics:**
  - `trivial` — Single function, < 20 lines, no new dependencies
  - `small` — Single file, < 100 lines, straightforward logic
  - `medium` — Multiple files, new types or interfaces, moderate logic
  - `large` — Cross-cutting change, new subsystem, significant testing
  - `unknown` — Cannot estimate; task may need decomposition

#### `NOTES`

- **Type:** `string` (free-text)
- **Semantics:** Context, rationale, references to specs, or edge case discussion that doesn't fit other fields. This is the only field where unstructured prose is acceptable.

---

## 4. Type Vocabulary

All `type` annotations in INPUTS, OUTPUTS, and elsewhere use this vocabulary.

### 4.1 Primitive Types

| Type | Description |
|---|---|
| `string` | UTF-8 text |
| `int` | Signed integer (platform word size) |
| `i32`, `i64` | Explicit-width signed integers |
| `float`, `f64` | IEEE 754 floating point |
| `bool` | `true` or `false` |
| `bytes` | Raw byte sequence |

### 4.2 Compound Types

| Type | Syntax | Description |
|---|---|---|
| List | `list<T>` | Ordered sequence of T |
| Map | `map<K, V>` | Key-value mapping |
| Option | `option<T>` | Value may be absent (null/nil/None) |
| Union | `union(A, B, C)` | Exactly one of the listed types |
| Tuple | `tuple(T1, T2)` | Fixed-size heterogeneous sequence |

### 4.3 Refined Types

Refinements narrow a type to a subset of its values.

| Syntax | Meaning |
|---|---|
| `int(1..100)` | Integer between 1 and 100 inclusive |
| `float(> 0)` | Positive float |
| `string(len: 1..2000)` | String with length between 1 and 2000 |
| `string(pattern: "^[a-z]+$")` | String matching regex |
| `list<T>(len: 1..50)` | Non-empty list with at most 50 elements |

### 4.4 Domain Types

These are semantic aliases for common patterns. Projects may extend this list.

| Type | Underlying | Description |
|---|---|---|
| `filepath` | `string` | Path to a file (relative to project root) |
| `url` | `string(pattern: "^https?://...")` | Valid URL |
| `uuid` | `string(pattern: UUID v4)` | Universally unique identifier |
| `datetime` | `string(ISO 8601)` | Timestamp |
| `exit_code` | `int(0..255)` | Process exit code |

---

## 5. Constraint Language

Constraints in INPUTS, OUTPUTS, and CONSTRAINTS fields use a minimal expression syntax.

### 5.1 Comparison Operators

```
value > 0
value >= 1
value < 100
value <= limit
value == "expected"
value != ""
```

### 5.2 Logical Operators

```
len > 0 AND len <= 2000
status == "active" OR status == "pending"
NOT (value == 0)
```

### 5.3 Set Operators

```
format IN ["markdown", "json", "csv"]
status NOT_IN ["deleted", "archived"]
```

### 5.4 Field References

Constraints may reference other fields within the same Task Node:

```
output.count <= input.limit
output.total == input.price - input.discount
```

### 5.5 Quantifiers (for lists)

```
EACH item IN results: item.score > 0.0
SOME item IN results: item.is_primary == true
```

---

## 6. Dependency Graph Rules

### 6.1 DAG Enforcement

The set of all Task Nodes and their `DEPENDS_ON` relationships must form a Directed Acyclic Graph. Cycles are a validation error and indicate circular dependencies that must be resolved by decomposing tasks.

### 6.2 Parallel Safety

Tasks with no dependency relationship (neither directly nor transitively) are **parallel-safe** — an agent may execute them in any order or simultaneously.

### 6.3 Milestone Grouping

Tasks may be grouped into milestones using a `MILESTONE` header:

```
MILESTONE: M1 - Core Infrastructure
  TASKS: [sqlite-schema-init, weaviate-client-setup, docling-client-setup]

MILESTONE: M2 - Ingestion Pipeline
  DEPENDS_ON_MILESTONE: M1
  TASKS: [document-parser, chunker-integration, embedding-pipeline]
```

Milestone dependencies are syntactic sugar: they imply that every task in the dependent milestone depends on every task in the prerequisite milestone.

### 6.4 Critical Path

The **critical path** is the longest chain of sequential dependencies through the graph. Agents should prioritize tasks on the critical path when multiple unblocked tasks are available, unless PRIORITY fields override this.

---

## 7. Agent Consumption Protocol

This section defines how an AI coding agent should interpret and execute tasks written in this format.

### 7.1 Parsing

1. Read the task file (or task block within a larger file).
2. Validate all REQUIRED fields are present.
3. Validate all TASK_IDs in DEPENDS_ON exist in the task graph.
4. Validate the dependency graph is acyclic.
5. If validation fails: **halt and report the specific validation error to the user.** Do not attempt to execute a malformed task.

### 7.2 Execution Order

1. Build the dependency graph from all DEPENDS_ON fields.
2. Identify all tasks with no unmet dependencies (the "ready set").
3. From the ready set, select the highest-priority task (PRIORITY field, then critical path position, then document order).
4. Execute the task.
5. On completion, mark the task done and recompute the ready set.
6. Repeat until all tasks are complete or a task fails.

### 7.3 Executing a Single Task

1. Read GOAL — this is the success criterion. Keep it in working memory throughout.
2. Read CONSTRAINTS — these are hard boundaries. Violating any constraint is a failure.
3. Read INPUTS and OUTPUTS — understand the data contract.
4. Read FILES_SCOPE — limit changes to these files unless strictly necessary.
5. Read NON_GOALS — do not implement these.
6. Implement the solution.
7. Verify each ACCEPTANCE criterion. If any fails, iterate.
8. Report completion with a summary mapping each ACCEPTANCE criterion to its verification result.

### 7.4 Ambiguity Protocol

If any field is ambiguous, contradictory, or underspecified:

1. **Do not guess.** Do not make assumptions about the author's intent.
2. **Halt the current task.**
3. **Report the specific ambiguity** to the user, quoting the problematic field.
4. **Propose 2-3 concrete interpretations** for the user to choose from.
5. **Resume only after receiving clarification.**

### 7.5 Deviation Protocol

If during implementation the agent discovers that the task as specified is impossible, incorrect, or suboptimal:

1. **Stop implementing.**
2. **Document the issue** — what was discovered, why the spec is problematic.
3. **Propose a revision** to the relevant fields (INPUTS, OUTPUTS, CONSTRAINTS, etc.).
4. **Wait for approval** before continuing.

---

## 8. Validation Checklist

A task graph passes validation if and only if all of the following hold:

| # | Rule | Severity |
|---|---|---|
| V1 | Every Task Node has all REQUIRED fields | Error |
| V2 | Every `TASK_ID` is unique within the project | Error |
| V3 | Every `TASK_ID` matches pattern `^[a-z0-9]+(-[a-z0-9]+)*$` | Error |
| V4 | Every `DEPENDS_ON` reference resolves to an existing `TASK_ID` | Error |
| V5 | The dependency graph contains no cycles | Error |
| V6 | Every `GOAL` is phrased as a testable outcome (no "try", "explore", etc.) | Error |
| V7 | Every `ACCEPTANCE` criterion is independently verifiable | Error |
| V8 | Every `type` annotation uses vocabulary from Section 4 | Warning |
| V9 | Every `CONTEXTUAL` field is either populated or explicitly `N/A` with justification | Warning |
| V10 | `FILES_SCOPE` is non-empty for implementation tasks | Warning |

---

## 9. Complete Examples

### Example A: Pure Computation Task

```yaml
TASK_ID: calculate-discounted-total
TASK_NAME: Implement discount calculation for order totals
GOAL: Given a price and a discount (fixed amount or percentage), return the discounted total, guaranteed non-negative.

INPUTS:
  - name: price
    type: f64
    constraints: price > 0
    source: Order record from database
  - name: discount
    type: union(Fixed: f64, Percentage: f64(0..1))
    constraints: "Fixed: value >= 0; Percentage: 0 <= value <= 1"
    source: Promotion rules engine

OUTPUTS:
  - name: total
    type: f64
    constraints: total >= 0
    destination: Return value, stored to order.total_amount

ACCEPTANCE:
  - CalculateTotal(100.0, Fixed(10.0)) == 90.0
  - CalculateTotal(100.0, Percentage(0.1)) == 90.0
  - CalculateTotal(50.0, Fixed(60.0)) == 0.0 (clamped, not negative)
  - CalculateTotal(0.01, Percentage(0.99)) > 0 (no floating point underflow to negative)
  - Unit tests pass with 100% branch coverage for this function

DEPENDS_ON: N/A (pure function, no external dependencies)

CONSTRAINTS:
  - Pure function: no side effects, no I/O
  - Result must be clamped to 0.0 minimum (never return negative)
  - Use decimal-safe arithmetic if available in the language; document precision limits otherwise

FILES_SCOPE:
  - internal/pricing/discount.go
  - internal/pricing/discount_test.go

NON_GOALS:
  - Do not implement tax calculation
  - Do not handle currency conversion
  - Do not persist the result (caller's responsibility)

EFFECTS: None (pure function)

ERROR_CASES:
  - condition: price is zero or negative
    behavior: Return error
    output: "invalid price: must be positive"
  - condition: Fixed discount exceeds price
    behavior: Clamp to 0.0 (not an error)
    output: N/A (silent clamp)

PRIORITY: medium
ESTIMATE: trivial
NOTES: This function is on the critical path for the order pipeline. Keep it simple and fast.
```

### Example B: CLI Feature Task

```yaml
TASK_ID: cli-export-format-flag
TASK_NAME: Add --format flag to the export command supporting Markdown and JSON
GOAL: The "export" CLI command accepts a --format flag (values: "markdown", "json") and writes the extraction results to stdout in the chosen format.

INPUTS:
  - name: extraction_id
    type: int
    constraints: extraction_id > 0
    source: Positional CLI argument
  - name: format
    type: string
    constraints: format IN ["markdown", "json"]
    source: CLI flag --format, default "markdown"

OUTPUTS:
  - name: formatted_output
    type: string
    constraints: len > 0
    destination: stdout
  - name: exit_code
    type: exit_code
    constraints: exit_code IN [0, 1, 2]
    destination: Process exit code

ACCEPTANCE:
  - "aqe export 1 --format markdown" writes valid Markdown to stdout and exits 0
  - "aqe export 1 --format json" writes valid JSON (parseable by jq) to stdout and exits 0
  - "aqe export 1 --format xml" prints error to stderr and exits 1
  - "aqe export 999" (non-existent ID) prints error to stderr and exits 1
  - Default (no --format flag) produces Markdown
  - JSON output contains fields: quotes[], metadata{}, references[]
  - Markdown output contains: ## Quotes, ## References sections

DEPENDS_ON:
  - sqlite-schema-init
  - cli-base-setup

CONSTRAINTS:
  - Use cobra for flag registration (consistent with existing CLI)
  - Output goes to stdout; errors go to stderr
  - Exit codes: 0 = success, 1 = user error, 2 = system error
  - No color codes in output (must be pipe-safe)

FILES_SCOPE:
  - internal/cli/export.go
  - internal/cli/export_test.go
  - internal/formatter/markdown.go
  - internal/formatter/json.go
  - internal/formatter/formatter.go (interface definition)

NON_GOALS:
  - Do not implement CSV or HTML export (future tasks)
  - Do not add --output flag for file writing (always stdout)
  - Do not add interactive prompts

EFFECTS:
  - type: DB.Read
    target: SQLite extractions and chunks tables

ERROR_CASES:
  - condition: Extraction ID does not exist in database
    behavior: Return exit code 1
    output: "Error: extraction #999 not found"
  - condition: Database is unreachable or corrupted
    behavior: Return exit code 2
    output: "Error: database unavailable: <underlying error>"
  - condition: Invalid format value
    behavior: Return exit code 1
    output: "Error: unsupported format 'xml'. Supported: markdown, json"

PRIORITY: high
ESTIMATE: medium
NOTES: The formatter interface should be designed so adding new formats (csv, html) later requires only adding a new implementation, not modifying existing code.
```

### Example C: Integration Task

```yaml
TASK_ID: weaviate-hybrid-search
TASK_NAME: Implement hybrid BM25 + vector search via Weaviate
GOAL: A Search() function queries Weaviate using hybrid search (BM25 + vector similarity) and returns ranked chunk results with scores.

INPUTS:
  - name: ctx
    type: context.Context
    constraints: none
    source: Caller provides; must support cancellation
  - name: query
    type: string
    constraints: len > 0, len <= 2000
    source: User search query from CLI
  - name: limit
    type: int
    constraints: 1 <= limit <= 100
    source: CLI flag, default 10
  - name: alpha
    type: f64
    constraints: 0.0 <= alpha <= 1.0
    source: Config, default 0.75 (favors vector over BM25)

OUTPUTS:
  - name: results
    type: list<ChunkResult>
    constraints: len <= limit; EACH item: item.score >= 0.0
    destination: Return value
  - name: err
    type: option<error>
    constraints: none
    destination: Return value

ACCEPTANCE:
  - Querying "machine learning" against a seeded Weaviate instance returns non-empty results
  - Results are sorted by descending score
  - Each ChunkResult contains: chunk_id, text_preview (first 200 chars), score, document_id
  - Context cancellation mid-query returns context.Canceled error (not a hang)
  - Limit of 5 returns at most 5 results
  - Empty Weaviate collection returns empty list and nil error
  - Integration test passes: tests/integration/search_test.go

DEPENDS_ON:
  - weaviate-client-setup
  - weaviate-schema-init
  - embedding-pipeline

CONSTRAINTS:
  - Use official weaviate-go-client/v4 directly (no wrapper abstraction)
  - Use Weaviate's native hybrid search with alpha parameter
  - Do not cache results (caller's responsibility)
  - Function signature: func (s *SearchClient) Search(ctx context.Context, query string, limit int, alpha float64) ([]ChunkResult, error)

FILES_SCOPE:
  - internal/search/weaviate.go
  - internal/search/weaviate_test.go
  - internal/models/chunk.go (ChunkResult struct if not already defined)

NON_GOALS:
  - Do not implement result caching
  - Do not implement pagination or cursor-based iteration
  - Do not implement query preprocessing or expansion
  - Do not add filtering by document or date (future task)

EFFECTS:
  - type: Network.Out
    target: Weaviate HTTP API at localhost:8080

ERROR_CASES:
  - condition: Weaviate service unreachable
    behavior: Return wrapped error with context
    output: (programmatic) fmt.Errorf("search failed: weaviate unavailable: %w", err)
  - condition: Invalid alpha value
    behavior: Return error before making network call
    output: (programmatic) fmt.Errorf("invalid alpha %.2f: must be between 0.0 and 1.0", alpha)

PRIORITY: critical
ESTIMATE: medium
NOTES: Alpha = 0.75 means 75% vector similarity, 25% BM25 keyword matching. This is tunable and should come from config, but default to 0.75 for initial implementation.
```

### Example D: Refactoring Task

```yaml
TASK_ID: extract-formatter-interface
TASK_NAME: Refactor export formatters to use a common interface
GOAL: All export formatters (Markdown, JSON, and future formats) implement a single Formatter interface, selected by a registry function.

INPUTS:
  - name: existing_code
    type: N/A (refactoring)
    constraints: N/A
    source: Current implementation in internal/cli/export.go (inline formatting logic)

OUTPUTS:
  - name: interface_definition
    type: N/A (code artifact)
    constraints: N/A
    destination: internal/formatter/formatter.go
  - name: implementations
    type: N/A (code artifacts)
    constraints: N/A
    destination: internal/formatter/markdown.go, internal/formatter/json.go

ACCEPTANCE:
  - A Formatter interface exists with method: Format(data *ExtractionResult) ([]byte, error)
  - A NewFormatter(format string) (Formatter, error) registry function exists
  - MarkdownFormatter and JSONFormatter both implement Formatter
  - All existing tests still pass (zero behavior change)
  - export.go uses the Formatter interface instead of inline formatting
  - Adding a new format requires only: (1) new file with implementation, (2) register in NewFormatter

DEPENDS_ON:
  - cli-export-format-flag

CONSTRAINTS:
  - Zero behavior change — output must be byte-for-byte identical before and after refactor
  - No new external dependencies
  - Interface must be in its own package (internal/formatter), not in cli package

FILES_SCOPE:
  - internal/formatter/formatter.go (new — interface + registry)
  - internal/formatter/markdown.go (new — extracted from export.go)
  - internal/formatter/json.go (new — extracted from export.go)
  - internal/cli/export.go (modified — use Formatter interface)
  - internal/formatter/formatter_test.go (new)

NON_GOALS:
  - Do not add new format types (CSV, HTML)
  - Do not change the CLI flag interface
  - Do not optimize formatting performance

EFFECTS: None (pure refactoring)

ERROR_CASES:
  - condition: Unknown format string passed to NewFormatter
    behavior: Return error
    output: (programmatic) fmt.Errorf("unsupported format %q: supported formats are: markdown, json", format)

PRIORITY: medium
ESTIMATE: small
NOTES: This is a prerequisite for adding CSV and HTML export later. The interface should be minimal — resist adding methods beyond Format() until there is a concrete need.
```

---

## 10. Extending This Specification

### 10.1 Project-Specific Domain Types

Projects may define additional domain types in a `TYPES` preamble at the top of a task file:

```yaml
TYPES:
  ChunkResult: {chunk_id: int, text: string, score: f64, document_id: int}
  HarvardCitation: {authors: list<Author>, year: int, title: string, ...}
```

These types are then available for use in all Task Nodes in that file.

### 10.2 Template Inheritance

Common constraints that apply to all tasks in a project (e.g., "all code must pass `go vet`") may be defined in a `DEFAULTS` block:

```yaml
DEFAULTS:
  CONSTRAINTS:
    - All code must pass go vet and go fmt
    - All exported functions must have doc comments
    - Error messages must not expose internal paths or stack traces
  ACCEPTANCE:
    - go test ./... passes
    - go vet ./... reports no issues
```

Individual Task Nodes inherit these defaults. Task-level fields **append to** (not replace) default fields.

### 10.3 Version History

| Version | Date | Changes |
|---|---|---|
| 0.1.0 | 2026-01-31 | Initial draft |
| 0.2.0 | 2026-01-31 | Added JSON Schema validation workflow and `taskval` CLI |

---

## 11. JSON Schema Validation Workflow

### 11.1 Overview

Tasks authored by an LLM agent (or a human) can be mechanically validated before execution using the `taskval` CLI tool. This implements a two-tier validation strategy:

```
TASK (natural language) → LLM compiles → JSON → Tier 1: Schema check → Tier 2: Semantic check → PASS / FAIL + actionable errors → if FAIL: LLM refines → re-validate
```

**Tier 1 (Structural — JSON Schema):** Deterministic, zero-LLM checks for required fields, types, patterns, enum constraints, string lengths, and array minimums. Uses JSON Schema Draft 2020-12.

**Tier 2 (Semantic — Programmatic):** Cross-node and content-quality checks that require graph analysis or heuristic pattern matching. Includes DAG acyclicity (V5), dependency reference integrity (V4), goal quality (V6), acceptance criterion quality (V7), and contextual field completeness (V9).

### 11.2 JSON Representation

Task Nodes authored in the YAML-like format defined in Section 3 are converted to JSON for validation. The JSON field names use `snake_case` versions of the spec field names:

| Spec Field | JSON Key |
|---|---|
| `TASK_ID` | `task_id` |
| `TASK_NAME` | `task_name` |
| `GOAL` | `goal` |
| `INPUTS` | `inputs` |
| `OUTPUTS` | `outputs` |
| `ACCEPTANCE` | `acceptance` |
| `DEPENDS_ON` | `depends_on` |
| `CONSTRAINTS` | `constraints` |
| `FILES_SCOPE` | `files_scope` |
| `NON_GOALS` | `non_goals` |
| `EFFECTS` | `effects` |
| `ERROR_CASES` | `error_cases` |
| `PRIORITY` | `priority` |
| `ESTIMATE` | `estimate` |
| `NOTES` | `notes` |

Contextual fields that are not applicable use a structured N/A:

```json
"depends_on": {"status": "N/A", "reason": "Pure function, no external dependencies"}
```

### 11.3 Task Graph JSON Envelope

A complete task graph wraps multiple task nodes with optional metadata:

```json
{
  "version": "0.1.0",
  "types": { ... },
  "defaults": { ... },
  "milestones": [ ... ],
  "tasks": [ ... ]
}
```

### 11.4 The `taskval` CLI

```
taskval [flags] <file.json>

Flags:
  --mode=task|graph    Validate a single task node or a full task graph (default: graph)
  --output=text|json   Output format (default: text)

Exit codes:
  0  Validation passed (no errors; warnings may be present)
  1  Validation failed (one or more errors found)
  2  Usage error or internal error
```

**Text output** is designed for human and LLM readability — each error includes the rule ID, JSON path, problem description, actionable fix suggestion, and the offending value.

**JSON output** is designed for programmatic consumption — the same information as structured JSON, suitable for piping into an LLM agent's feedback loop.

### 11.5 LLM Agent Feedback Loop

The recommended workflow for LLM-compiled tasks:

1. **Agent receives** a natural language task description from the user.
2. **Agent compiles** the task into the JSON format defined by this spec.
3. **Agent runs** `taskval --mode=task --output=json <task.json>`.
4. **If valid:** The task is ready for execution per Section 7.
5. **If invalid:** The agent reads the error output (each error includes `rule`, `path`, `message`, `suggestion`) and revises the specific fields identified.
6. **Agent re-runs** `taskval`. Repeat until validation passes.
7. **Maximum 3 refinement attempts** — if the task still fails after 3 rounds, halt and escalate to the user with the remaining errors.

This loop ensures that by the time a task reaches execution, it is structurally complete, semantically consistent, and free of the most common quality defects.

### 11.6 JSON Schema Files

The authoritative JSON Schemas are located at:

- `schemas/task_node.schema.json` — Single task node validation
- `schemas/task_graph.schema.json` — Full task graph validation (references task_node schema)

These schemas enforce validation rules V1, V2 (uniqueness checked in Tier 2), V3, V8 (partially), and structural aspects of V9.

---

## Appendix A: Quick Reference Card

```
TASK_ID:      <kebab-case-unique-id>                     [REQUIRED]
TASK_NAME:    <Imperative phrase, <80 chars>              [REQUIRED]
GOAL:         <One testable outcome sentence>             [REQUIRED]
INPUTS:       [{ name, type, constraints, source }]       [REQUIRED]
OUTPUTS:      [{ name, type, constraints, destination }]   [REQUIRED]
ACCEPTANCE:   [<testable assertion>, ...]                 [REQUIRED]
DEPENDS_ON:   [<TASK_ID>, ...] | N/A                      [CONTEXTUAL]
CONSTRAINTS:  [<hard rule>, ...]                          [CONTEXTUAL]
FILES_SCOPE:  [<filepath or glob>, ...]                   [CONTEXTUAL]
NON_GOALS:    [<exclusion>, ...]                          [OPTIONAL]
EFFECTS:      [{ type, target }]                          [OPTIONAL]
ERROR_CASES:  [{ condition, behavior, output }]           [OPTIONAL]
PRIORITY:     critical | high | medium | low              [OPTIONAL]
ESTIMATE:     trivial | small | medium | large | unknown  [OPTIONAL]
NOTES:        <free text>                                 [OPTIONAL]
```
