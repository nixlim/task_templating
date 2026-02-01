#!/usr/bin/env bash
set -euo pipefail

# install-taskify.sh — Bootstrap the /taskify skill into any git repository
# https://github.com/nixlim/task_templating

VERSION="0.1.0"

# =============================================================================
# Section 1: USAGE
# =============================================================================

usage() {
  cat <<USAGE
install-taskify.sh v${VERSION} — Install the /taskify skill into a project

Usage:
  bash install-taskify.sh [flags]

Flags:
  --target DIR    Target project directory (default: current directory)
  --force         Overwrite existing skill files (upgrade path)
  --skip-beads    Skip bd installation and bd init
  --dry-run       Print actions without executing them
  --help          Show this help message

Examples:
  bash install-taskify.sh
  bash install-taskify.sh --target /path/to/project
  bash install-taskify.sh --force --target /path/to/project
  bash install-taskify.sh --dry-run
USAGE
}

# =============================================================================
# Section 2: FLAG PARSING
# =============================================================================

TARGET_DIR="."
FORCE=0
SKIP_BEADS=0
DRY_RUN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      if [[ -z "${2:-}" ]]; then
        echo "[taskify] ERROR: --target requires a directory argument" >&2
        exit 1
      fi
      TARGET_DIR="$2"
      shift 2
      ;;
    --force)
      FORCE=1
      shift
      ;;
    --skip-beads)
      SKIP_BEADS=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "[taskify] ERROR: Unknown flag: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

# =============================================================================
# Section 3: UTILITY FUNCTIONS
# =============================================================================

# Colour codes (disabled if stdout is not a terminal)
if [[ -t 1 ]]; then
  _GREEN='\033[0;32m'
  _YELLOW='\033[0;33m'
  _RED='\033[0;31m'
  _RESET='\033[0m'
else
  _GREEN=''
  _YELLOW=''
  _RED=''
  _RESET=''
fi

log() {
  printf "${_GREEN}[taskify]${_RESET} %s\n" "$*"
}

warn() {
  printf "${_YELLOW}[taskify]${_RESET} %s\n" "$*" >&2
}

die() {
  printf "${_RED}[taskify]${_RESET} %s\n" "$*" >&2
  exit 1
}

# run CMD [ARGS...]
# In dry-run mode, prints the command. Otherwise, executes it.
run() {
  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "[dry-run] $*"
  else
    "$@"
  fi
}

# need_cmd NAME INSTALL_CMD
# Checks if NAME is on PATH. If not, runs INSTALL_CMD to install it.
need_cmd() {
  local name="$1"
  local install_cmd="$2"

  if command -v "$name" >/dev/null 2>&1; then
    log "$name is already installed ($(command -v "$name"))"
    return 0
  fi

  log "Installing $name ..."
  run $install_cmd
}

# file_has_marker FILE MARKER
# Returns 0 if the file exists and contains the marker string.
file_has_marker() {
  local file="$1"
  local marker="$2"

  [[ -f "$file" ]] && grep -qF "$marker" "$file"
}

# =============================================================================
# Validate target directory
# =============================================================================

# Resolve to absolute path
_ORIG_TARGET="$TARGET_DIR"
TARGET_DIR="$(cd "$TARGET_DIR" 2>/dev/null && pwd)" || die "Target directory does not exist: $_ORIG_TARGET"
unset _ORIG_TARGET

log "Target: $TARGET_DIR"
log "Version: $VERSION"
[[ "$DRY_RUN" -eq 1 ]] && log "Mode: DRY RUN (no changes will be made)"
[[ "$FORCE" -eq 1 ]] && log "Mode: FORCE (existing skill files will be overwritten)"

# =============================================================================
# Section 4: PREREQUISITE CHECKS
# =============================================================================

# 4a. Go is required for go install
if ! command -v go >/dev/null 2>&1; then
  die "Go is required. Install from https://go.dev/dl/"
fi
log "Go found: $(go version)"

# 4b. Check if target is a git repository
if [[ ! -d "$TARGET_DIR/.git" ]]; then
  warn "Target is not a git repository. Beads requires git — some features may not work."
fi

# 4c. Install taskval if missing
need_cmd taskval "go install github.com/nixlim/task_templating/cmd/taskval@latest"

# 4d. Install bd if missing (unless --skip-beads)
if [[ "$SKIP_BEADS" -eq 0 ]]; then
  need_cmd bd "go install github.com/steveyegge/beads/cmd/bd@latest"
else
  log "Skipping bd installation (--skip-beads)"
fi

# 4e. Run bd init if .beads/ does not exist (unless --skip-beads)
if [[ "$SKIP_BEADS" -eq 0 ]]; then
  if [[ ! -d "$TARGET_DIR/.beads" ]]; then
    log "Initialising beads in $TARGET_DIR ..."
    run bd init --dir "$TARGET_DIR" || warn "bd init failed — you may need to run it manually"
  else
    log "Beads already initialised ($TARGET_DIR/.beads/)"
  fi
else
  log "Skipping beads initialisation (--skip-beads)"
fi

# =============================================================================
# Section 5: FILE CREATION
# =============================================================================

# Counters for summary
FILES_CREATED=0
FILES_SKIPPED=0

# write_file DEST_PATH
# Writes stdin to DEST_PATH. Skips if file exists (unless --force).
# Respects --dry-run.
write_file() {
  local dest="$1"
  local dir
  dir="$(dirname "$dest")"

  if [[ -f "$dest" && "$FORCE" -eq 0 ]]; then
    log "Already exists (skipped): $dest"
    FILES_SKIPPED=$((FILES_SKIPPED + 1))
    cat >/dev/null  # consume stdin
    return 0
  fi

  if [[ "$DRY_RUN" -eq 1 ]]; then
    log "[dry-run] Would create: $dest"
    cat >/dev/null  # consume stdin
    FILES_CREATED=$((FILES_CREATED + 1))
    return 0
  fi

  mkdir -p "$dir"
  cat > "$dest"
  log "Created: $dest"
  FILES_CREATED=$((FILES_CREATED + 1))
}

# --- 5a. .claude/agents/taskify-agent.md ---

write_file "$TARGET_DIR/.claude/agents/taskify-agent.md" << 'HEREDOC_AGENT'
---
name: taskify-agent
description: >
  Specialized agent for decomposing specs and plans into structured task graph JSON,
  validating against the Structured Task Template Spec, and creating Beads issues.
  Used exclusively by the /taskify skill.
tools: Read, Write, Edit, Bash, Glob, Grep
model: opus
permissionMode: acceptEdits
skills:
  - taskify
---

# Taskify Agent

You are a task decomposition specialist. You read specs, plans, and descriptions,
break them into structured task graphs, validate them with `taskval`, and record
them as Beads issues with `bd`.

Your workflow is defined in the taskify skill. Follow its steps precisely.

Key principles:
- Every task must be 30 minutes to 4 hours of work
- Goals are testable outcomes, never activities
- Acceptance criteria are specific and independently verifiable
- Dependencies form a DAG (no cycles)
- Fix validation errors by reading the output and adjusting the JSON
HEREDOC_AGENT

# --- 5b. .claude/skills/taskify/SKILL.md (with patched error recovery) ---

write_file "$TARGET_DIR/.claude/skills/taskify/SKILL.md" << 'HEREDOC_SKILL'
---
name: taskify
description: >
  Decompose a spec, plan, or text description into structured task graph JSON,
  validate it against the Structured Task Template Spec, and record the tasks
  as Beads issues. Use when the user wants to break down work into trackable,
  validated tasks.
argument-hint: <spec-file or description>
disable-model-invocation: true
context: fork
agent: taskify-agent
allowed-tools: Read, Write, Edit, Bash(taskval *), Bash(bd *), Bash(cat *), Glob, Grep
---

# Taskify: Spec/Plan to Validated Beads

You are a task decomposition specialist. Your job is to read a spec, plan, or
description, break it into a structured task graph, validate it, and record the
tasks as Beads issues.

## Input Detection

Examine `$ARGUMENTS`:

1. If it ends in `.task.json`: This is a pre-composed task file. Skip to Step 3 (Validation).
2. If it is a valid file path: Read the file. Proceed to Step 1 (Analysis).
3. Otherwise: Treat as inline text description. Proceed to Step 1 (Analysis).

## Step 1: Analyse the Input

Read the spec/plan/description carefully. Identify:

- The overall goal (what is being built or changed)
- Distinct units of work (each should be 30 min to 4 hours of effort)
- Dependencies between units (what must complete before what)
- File paths that will be touched (for files_scope)
- Inputs and outputs for each unit
- Acceptance criteria (testable, specific assertions)

For detailed field definitions and validation rules, see [spec-reference.md](spec-reference.md).
For examples and common patterns, see [task-writing-guide.md](task-writing-guide.md).

## Step 2: Compose Task Graph JSON

Write a task graph JSON file conforming to the Structured Task Template Spec.

### File naming
- Save to `.tasks/<descriptive-name>.task.json` in the project root
- Create the `.tasks/` directory if it doesn't exist

### Required structure
Every task graph must have:
- `version`: "0.1.0"
- `tasks`: array of task nodes

Each task node must have these required fields:
- `task_id`: kebab-case, unique, max 60 chars (e.g., "add-oauth-google")
- `task_name`: imperative phrase starting with a verb, max 80 chars
- `goal`: single testable sentence; FORBIDDEN words: "try", "explore", "investigate", "look into"
- `inputs`: array of {name, type, constraints, source}
- `outputs`: array of {name, type, constraints, destination}
- `acceptance`: array of independently verifiable assertions

Each task must also address contextual fields (provide value or N/A with reason):
- `depends_on`: array of task_ids, or {"status": "N/A", "reason": "..."}
- `constraints`: array of strings, or {"status": "N/A", "reason": "..."}
- `files_scope`: array of file paths/globs, or {"status": "N/A", "reason": "..."}

Optional fields (include when useful):
- `non_goals`, `effects`, `error_cases`, `priority`, `estimate`, `notes`

### Quality rules to follow
- Goals must be testable outcomes, not activities
- Each acceptance criterion must be independently verifiable
- No vague terms in acceptance: avoid "works correctly", "is good", "functions properly"
- Dependencies must form a DAG (no cycles)
- Implementation tasks need files_scope (use N/A with reason if truly not applicable)

## Step 3: Validate

Run validation:

```bash
taskval --mode=graph .tasks/<name>.task.json
```

### If validation PASSES:
Proceed to Step 4.

### If validation FAILS:
1. Read the error output carefully
2. Fix each reported issue in the JSON file:
   - SCHEMA errors: fix structural problems (missing fields, wrong types)
   - V6 errors: rewrite goal to remove forbidden words, make it a testable outcome
   - V7 errors: replace vague acceptance criteria with specific assertions
   - V9 errors: add missing contextual fields or mark N/A with reason
   - V10 errors: add files_scope for implementation tasks
3. Re-run validation
4. Repeat up to 3 times total. If still failing after 3 attempts, report the
   remaining errors to the user and stop.

## Step 4: Create Beads

Once validation passes, create the Beads issues:

```bash
taskval --mode=graph --create-beads .tasks/<name>.task.json
```

If beads isn't initialized in this project, run first:
```bash
bd init
```

## Step 5: Report Results

Provide a summary to the user:

1. How many tasks were created
2. The epic ID and title (if graph mode)
3. Each task with its Beads ID, title, and priority
4. The dependency structure
5. Suggest running `bd ready` to see available work

## Error Recovery

- If `taskval` is not found: Tell the user to install it with `go install github.com/nixlim/task_templating/cmd/taskval@latest`
- If `bd` is not found: Tell the user to install beads: `go install github.com/steveyegge/beads/cmd/bd@latest`
- If `bd init` hasn't been run: Run `bd init` automatically
- If beads creation fails partway: Report what was created and what failed
HEREDOC_SKILL

# --- 5c. .claude/skills/taskify/spec-reference.md ---

write_file "$TARGET_DIR/.claude/skills/taskify/spec-reference.md" << 'HEREDOC_SPECREF'
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
HEREDOC_SPECREF

# --- 5d. .claude/skills/taskify/task-writing-guide.md ---

write_file "$TARGET_DIR/.claude/skills/taskify/task-writing-guide.md" << 'HEREDOC_GUIDE'
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
HEREDOC_GUIDE

# --- 5e. .claude/skills/taskify/examples/single-task.json ---

write_file "$TARGET_DIR/.claude/skills/taskify/examples/single-task.json" << 'HEREDOC_SINGLE'
{
  "task_id": "calculate-discounted-total",
  "task_name": "Implement discount calculation for order totals",
  "goal": "Given a price and a discount (fixed amount or percentage), return the discounted total, guaranteed non-negative.",
  "inputs": [
    {
      "name": "price",
      "type": "f64",
      "constraints": "price > 0",
      "source": "Order record from database"
    },
    {
      "name": "discount",
      "type": "union(Fixed: f64, Percentage: f64(0..1))",
      "constraints": "Fixed: value >= 0; Percentage: 0 <= value <= 1",
      "source": "Promotion rules engine"
    }
  ],
  "outputs": [
    {
      "name": "total",
      "type": "f64",
      "constraints": "total >= 0",
      "destination": "Return value, stored to order.total_amount"
    }
  ],
  "acceptance": [
    "CalculateTotal(100.0, Fixed(10.0)) == 90.0",
    "CalculateTotal(100.0, Percentage(0.1)) == 90.0",
    "CalculateTotal(50.0, Fixed(60.0)) == 0.0 (clamped, not negative)",
    "CalculateTotal(0.01, Percentage(0.99)) > 0 (no floating point underflow to negative)",
    "Unit tests pass with 100% branch coverage for this function"
  ],
  "depends_on": {
    "status": "N/A",
    "reason": "Pure function, no external dependencies"
  },
  "constraints": [
    "Pure function: no side effects, no I/O",
    "Result must be clamped to 0.0 minimum (never return negative)",
    "Use decimal-safe arithmetic if available in the language; document precision limits otherwise"
  ],
  "files_scope": [
    "internal/pricing/discount.go",
    "internal/pricing/discount_test.go"
  ],
  "non_goals": [
    "Do not implement tax calculation",
    "Do not handle currency conversion",
    "Do not persist the result (caller's responsibility)"
  ],
  "effects": "None",
  "error_cases": [
    {
      "condition": "price is zero or negative",
      "behavior": "Return error",
      "output": "invalid price: must be positive"
    },
    {
      "condition": "Fixed discount exceeds price",
      "behavior": "Clamp to 0.0 (not an error)",
      "output": "N/A (silent clamp)"
    }
  ],
  "priority": "medium",
  "estimate": "trivial",
  "notes": "This function is on the critical path for the order pipeline. Keep it simple and fast."
}
HEREDOC_SINGLE

# --- 5f. .claude/skills/taskify/examples/task-graph.json ---

write_file "$TARGET_DIR/.claude/skills/taskify/examples/task-graph.json" << 'HEREDOC_GRAPH'
{
  "version": "0.1.0",
  "types": {
    "ChunkResult": {
      "chunk_id": "int",
      "text": "string",
      "score": "f64",
      "document_id": "int"
    }
  },
  "defaults": {
    "constraints": [
      "All code must pass go vet and go fmt",
      "All exported functions must have doc comments"
    ],
    "acceptance": [
      "go test ./... passes",
      "go vet ./... reports no issues"
    ]
  },
  "milestones": [
    {
      "name": "M1 - Core Infrastructure",
      "task_ids": ["calculate-discounted-total", "cli-export-format-flag"]
    },
    {
      "name": "M2 - Search",
      "depends_on_milestones": ["M1 - Core Infrastructure"],
      "task_ids": ["weaviate-hybrid-search"]
    }
  ],
  "tasks": [
    {
      "task_id": "calculate-discounted-total",
      "task_name": "Implement discount calculation for order totals",
      "goal": "Given a price and a discount (fixed amount or percentage), return the discounted total, guaranteed non-negative.",
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
        "Unit tests pass with 100% branch coverage"
      ],
      "depends_on": {
        "status": "N/A",
        "reason": "Pure function, no external dependencies"
      },
      "constraints": [
        "Pure function: no side effects, no I/O"
      ],
      "files_scope": [
        "internal/pricing/discount.go",
        "internal/pricing/discount_test.go"
      ],
      "priority": "medium",
      "estimate": "trivial"
    },
    {
      "task_id": "cli-export-format-flag",
      "task_name": "Add --format flag to the export command supporting Markdown and JSON",
      "goal": "The export CLI command accepts a --format flag (values: markdown, json) and writes extraction results to stdout in the chosen format.",
      "inputs": [
        {
          "name": "extraction_id",
          "type": "int",
          "constraints": "extraction_id > 0",
          "source": "Positional CLI argument"
        },
        {
          "name": "format",
          "type": "string",
          "constraints": "format IN [\"markdown\", \"json\"]",
          "source": "CLI flag --format, default markdown"
        }
      ],
      "outputs": [
        {
          "name": "formatted_output",
          "type": "string",
          "constraints": "len > 0",
          "destination": "stdout"
        }
      ],
      "acceptance": [
        "aqe export 1 --format markdown writes valid Markdown to stdout and exits 0",
        "aqe export 1 --format json writes valid JSON parseable by jq to stdout and exits 0",
        "Default (no --format flag) produces Markdown"
      ],
      "depends_on": {
        "status": "N/A",
        "reason": "Standalone CLI feature"
      },
      "constraints": [
        "Use cobra for flag registration (consistent with existing CLI)",
        "Output goes to stdout; errors go to stderr"
      ],
      "files_scope": [
        "internal/cli/export.go",
        "internal/cli/export_test.go"
      ],
      "priority": "high",
      "estimate": "medium"
    },
    {
      "task_id": "weaviate-hybrid-search",
      "task_name": "Implement hybrid BM25 + vector search via Weaviate",
      "goal": "A Search() function queries Weaviate using hybrid search (BM25 + vector similarity) and returns ranked chunk results with scores.",
      "inputs": [
        {
          "name": "query",
          "type": "string",
          "constraints": "len > 0, len <= 2000",
          "source": "User search query from CLI"
        },
        {
          "name": "limit",
          "type": "int",
          "constraints": "1 <= limit <= 100",
          "source": "CLI flag, default 10"
        }
      ],
      "outputs": [
        {
          "name": "results",
          "type": "list<ChunkResult>",
          "constraints": "len <= limit; EACH item: item.score >= 0.0",
          "destination": "Return value"
        }
      ],
      "acceptance": [
        "Querying 'machine learning' against a seeded Weaviate instance returns non-empty results",
        "Results are sorted by descending score",
        "Limit of 5 returns at most 5 results"
      ],
      "depends_on": ["calculate-discounted-total", "cli-export-format-flag"],
      "constraints": [
        "Use official weaviate-go-client/v4 directly (no wrapper abstraction)"
      ],
      "files_scope": [
        "internal/search/weaviate.go",
        "internal/search/weaviate_test.go"
      ],
      "effects": [
        {
          "type": "Network.Out",
          "target": "Weaviate HTTP API at localhost:8080"
        }
      ],
      "priority": "critical",
      "estimate": "medium"
    }
  ]
}
HEREDOC_GRAPH

log "File creation complete. Created: $FILES_CREATED, Skipped: $FILES_SKIPPED"

# =============================================================================
# Section 6: AGENTS.MD UPDATE
# =============================================================================

TASKIFY_MARKER="<!-- taskify-begin -->"
AGENTS_FILE="$TARGET_DIR/AGENTS.md"

_AGENTS_CONTENT='<!-- taskify-begin -->
## Taskify — Structured Task Decomposition

This project uses the `/taskify` skill for breaking down specs and plans into
validated, trackable task graphs.

### Usage

```
/taskify <spec-file-or-description>
```

The skill reads the input, decomposes it into structured task nodes (30 min to
4 hour granularity), validates the graph with `taskval`, and creates beads
issues for tracking.

### Prerequisites

| Tool      | Install                                                                   |
|-----------|---------------------------------------------------------------------------|
| `taskval` | `go install github.com/nixlim/task_templating/cmd/taskval@latest`         |
| `bd`      | `go install github.com/steveyegge/beads/cmd/bd@latest`                    |

### Quick reference

```bash
bd ready              # Find available work after task creation
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```
<!-- taskify-end -->'

if file_has_marker "$AGENTS_FILE" "$TASKIFY_MARKER"; then
  log "AGENTS.md already has taskify section (skipped)"
elif [[ "$DRY_RUN" -eq 1 ]]; then
  if [[ -f "$AGENTS_FILE" ]]; then
    log "[dry-run] Would append taskify section to AGENTS.md"
  else
    log "[dry-run] Would create AGENTS.md with taskify section"
  fi
else
  if [[ -f "$AGENTS_FILE" ]]; then
    printf '\n%s\n' "$_AGENTS_CONTENT" >> "$AGENTS_FILE"
    log "Appended taskify section to AGENTS.md"
  else
    printf '%s\n' "$_AGENTS_CONTENT" > "$AGENTS_FILE"
    log "Created AGENTS.md with taskify section"
  fi
fi

# =============================================================================
# Section 7: CLAUDE.MD UPDATE
# =============================================================================

CLAUDE_FILE="$TARGET_DIR/CLAUDE.md"

_CLAUDE_CONTENT='<!-- taskify-begin -->
## Task Management with Taskify

- Use `/taskify` to decompose specs, plans, or descriptions into structured task graphs
- Task graphs are saved to `.tasks/` and validated with `taskval`
- Issues are tracked with beads (`bd`). Run `bd ready` for available work.
- Run `bd sync` at session end to sync beads state with git
<!-- taskify-end -->'

if file_has_marker "$CLAUDE_FILE" "$TASKIFY_MARKER"; then
  log "CLAUDE.md already has taskify section (skipped)"
elif [[ "$DRY_RUN" -eq 1 ]]; then
  if [[ -f "$CLAUDE_FILE" ]]; then
    log "[dry-run] Would append taskify section to CLAUDE.md"
  else
    log "[dry-run] Would create CLAUDE.md with taskify section"
  fi
else
  if [[ -f "$CLAUDE_FILE" ]]; then
    printf '\n%s\n' "$_CLAUDE_CONTENT" >> "$CLAUDE_FILE"
    log "Appended taskify section to CLAUDE.md"
  else
    printf '%s\n' "$_CLAUDE_CONTENT" > "$CLAUDE_FILE"
    log "Created CLAUDE.md with taskify section"
  fi
fi

# =============================================================================
# Section 8: SUMMARY
# =============================================================================

echo ""
log "============================================"
log "  Taskify installation complete!"
log "============================================"
echo ""
log "Files created: $FILES_CREATED"
log "Files skipped: $FILES_SKIPPED"

if [[ "$DRY_RUN" -eq 1 ]]; then
  log "Mode: DRY RUN — no changes were made"
fi

if command -v taskval >/dev/null 2>&1; then
  log "taskval: installed ($(command -v taskval))"
fi

if [[ "$SKIP_BEADS" -eq 0 ]]; then
  if command -v bd >/dev/null 2>&1; then
    log "bd: installed ($(command -v bd))"
  fi
  if [[ -d "$TARGET_DIR/.beads" ]]; then
    log "beads: initialised"
  fi
else
  log "beads: skipped (--skip-beads)"
fi

echo ""
log "Next steps:"
log "  1. cd $TARGET_DIR"
log "  2. Open Claude Code"
log "  3. Run: /taskify <spec-file or description>"
log "  4. Run: bd ready   (to see available work)"
echo ""
