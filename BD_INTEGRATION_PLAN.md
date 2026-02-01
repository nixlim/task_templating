# BD Integration Plan: taskval x Beads

**Version:** 0.2.0
**Status:** Approved
**Date:** 2026-01-31

---

## Overview

Four deliverables:

1. **Fix module path** (`github.com/foundry-zero/task-templating` -> `github.com/nixlim/task_templating`)
2. **Add `--create-beads` and `--dry-run` flags** to `taskval` CLI
3. **Write `TASK_CREATION_INSTRUCTIONS_AGENTS.md`** for agent workflow guidance
4. **Create `/taskify` skill** -- a Claude Code slash command that reads a spec/plan, decomposes it into structured tasks, validates them, and records them as beads

---

## Part 1: Fix Module Path

### Changes

| File | Change |
|------|--------|
| `go.mod` | `module github.com/foundry-zero/task-templating` -> `module github.com/nixlim/task_templating` |
| `cmd/taskval/main.go` | Update import path on line 30 |

This is a 2-line change. All internal packages use relative imports so only the one import in `main.go` needs updating.

---

## Part 2: `--create-beads` and `--dry-run` Flags

### 2.1 New CLI Flags

```
taskval [flags] <file.json>

Existing flags:
  --mode=task|graph     Validation mode
  --output=text|json    Output format

New flags:
  --create-beads        On validation success, create Beads issues via bd CLI
  --dry-run             Show bd commands that would be executed (requires --create-beads)
  --epic-title          Override the auto-generated epic title (graph mode only)
```

### 2.2 Behavior

**Normal flow (no `--create-beads`):** Unchanged. Validate and report.

**With `--create-beads`:**

1. Run normal validation (Tier 1 + Tier 2)
2. If validation **fails** -> print errors, exit 1 (no beads created)
3. If validation **passes** -> proceed to Beads creation:

**Graph mode (`--mode=graph --create-beads`):**

```
Step 1: Create epic issue for the graph
        bd create --title "<graph_title>" \
          --type epic \
          --description "<goal summary from all tasks>" \
          --priority <mapped priority> \
          --json --silent

Step 2: For each task in topological order:
        bd create --title "<task_name>" \
          --type task \
          --description "<composed description>" \
          --acceptance "<joined acceptance>" \
          --priority <mapped priority> \
          --estimate <mapped minutes> \
          --notes "<notes>" \
          --parent <epic_id> \
          --labels "taskval-managed" \
          --json --silent

Step 3: For each depends_on reference:
        bd dep add <task_bd_id> <dependency_bd_id> --type blocks

Step 4: Store template metadata on each issue:
        bd update <task_bd_id> --design "<_template metadata JSON>"
```

**Single task mode (`--mode=task --create-beads`):**

```
Step 1: Create single task issue
        bd create --title "<task_name>" \
          --type task \
          --description "<composed description>" \
          --acceptance "<joined acceptance>" \
          --priority <mapped priority> \
          --estimate <mapped minutes> \
          --notes "<notes>" \
          --labels "taskval-managed" \
          --json --silent

Step 2: Store template metadata:
        bd update <task_bd_id> --design "<_template metadata JSON>"
```

**With `--dry-run`:** Print each `bd` command that would be executed to stdout, prefixed with `[DRY-RUN]`, but don't execute them. Exit 0 if validation passes.

### 2.3 Field Mapping (Task -> bd create)

| Template Field | bd Flag | Transformation |
|---|---|---|
| `task_name` | `--title` | Direct (truncate to 500 chars if needed) |
| `goal` | `--description` | Compose: goal + inputs/outputs + constraints + non_goals + error_cases sections |
| `acceptance` | `--acceptance` | Join array with `\n- ` to make markdown checklist |
| `priority` | `--priority` | Map: critical->0, high->1, medium->2, low->3, default->2 |
| `estimate` | `--estimate` | Map: trivial->15, small->60, medium->240, large->480, unknown->omit |
| `notes` | `--notes` | Direct from template `notes` field (human-readable) |
| `depends_on` | `bd dep add` | After creation, link with `--type blocks` |
| `files_scope` | `--design` | Stored in `_template` metadata JSON block |
| `inputs` | `--description` | Rendered as "## Inputs" section in description |
| `outputs` | `--description` | Rendered as "## Outputs" section in description |
| `constraints` | `--description` | Rendered as "## Constraints" section in description |
| `non_goals` | `--description` | Rendered as "## Non-Goals" section in description |
| `effects` | `--design` | Stored in `_template` metadata JSON block |
| `error_cases` | `--description` | Rendered as "## Error Cases" section in description |

### 2.4 Description Composition

For the `--description` flag, compose a structured markdown document:

```markdown
<goal text>

## Inputs
- **price** (`f64`): price > 0 -- Source: Order record from database
- **discount** (`union(...)`): Fixed: value >= 0 -- Source: Promotion rules engine

## Outputs
- **total** (`f64`): total >= 0 -- Dest: Return value

## Constraints
- Pure function: no side effects, no I/O
- Result must be clamped to 0.0 minimum

## Non-Goals
- Do not implement tax calculation
- Do not handle currency conversion

## Error Cases
- **price is zero or negative**: Return error -> "invalid price: must be positive"
- **Fixed discount exceeds price**: Clamp to 0.0 -> N/A (silent clamp)
```

### 2.5 Template Metadata in Design Field

Use `bd update --design` to store machine-readable template metadata on each created issue. The `design` field is semantically appropriate for "how this was specified":

```json
{
  "_template": {
    "version": "0.2.0",
    "task_id": "calculate-discounted-total",
    "files_scope": ["internal/pricing/discount.go", "internal/pricing/discount_test.go"],
    "effects": "None",
    "inputs": [
      {"name": "price", "type": "f64", "constraints": "price > 0", "source": "Order record from database"}
    ],
    "outputs": [
      {"name": "total", "type": "f64", "constraints": "total >= 0", "destination": "Return value"}
    ]
  }
}
```

**Rationale:** Beads' `metadata` field is not settable via `bd create` or `bd update` CLI flags. The `--design` flag maps to the Beads `design` field which is intended for design notes. Storing structured template metadata here keeps `--notes` clean for human-readable content, and the JSON is parseable by agents while being clearly delineated.

### 2.6 Epic Title Generation (Graph Mode)

For graph mode, the epic needs a title. Resolution order:

1. If `--epic-title` flag is provided, use that
2. If the graph has milestones, use: `"Task Graph: <first milestone name>"`
3. Otherwise, derive from input filename: `"Task Graph: <filename>"`
4. If reading from stdin: `"Task Graph: (stdin)"`

### 2.7 New Files

| File | Purpose |
|------|--------|
| `internal/beads/beads.go` | Core Beads integration: `Creator` struct, field mapping, bd command construction |
| `internal/beads/exec.go` | Command execution: shell out to `bd`, parse JSON output, handle errors |
| `internal/beads/mapping.go` | Description composition, priority mapping, estimate mapping |
| `internal/beads/beads_test.go` | Unit tests for field mapping, description composition, command construction |

### 2.8 Modified Files

| File | Change |
|------|--------|
| `go.mod` | Module path fix |
| `cmd/taskval/main.go` | Add `--create-beads`, `--dry-run`, `--epic-title` flags; import beads package; add beads creation step after validation |
| `internal/validator/types.go` | Add `Graph *TaskGraph` field to `ValidationResult` |
| `internal/validator/validate.go` | Populate `Graph` field when validation passes |

### 2.9 Validation Result Enhancement

Currently `Validate()` returns `*ValidationResult` which only has errors/stats. The beads layer needs the parsed `TaskGraph`/`TaskNode` to do field mapping without re-parsing the JSON.

Add a `Graph` field to `ValidationResult`:

```go
type ValidationResult struct {
    Valid  bool              `json:"valid"`
    Errors []ValidationError `json:"errors,omitempty"`
    Stats  ValidationStats   `json:"stats"`
    Graph  *TaskGraph        `json:"-"` // Parsed graph, not included in JSON output
}
```

The `json:"-"` tag ensures this field is excluded from JSON output, keeping the validation output format unchanged.

### 2.10 ID Mapping

When creating beads from a task graph, we need to map template `task_id` values to Beads `bd-xxxx` IDs (returned by `bd create --silent`). This is critical for the dependency linking step.

```go
type CreationResult struct {
    EpicID   string            // bd-xxxx for the epic (graph mode only)
    TaskIDs  map[string]string // template task_id -> bd-xxxx
    Commands []string          // all bd commands executed (for logging/dry-run)
    Created  int               // number of issues created
    Deps     int               // number of dependencies linked
}
```

### 2.11 Error Handling

If any `bd` command fails mid-creation:

- Print which tasks were successfully created (with their bd IDs)
- Print which task failed and why (stderr from bd)
- Exit 2 (internal error, not validation failure)
- Do NOT attempt to roll back (Beads issues can be manually cleaned up with `bd close` or `bd delete`)

Pre-flight checks before attempting creation:

- Verify `bd` is on PATH (`exec.LookPath("bd")`)
- Verify beads is initialized (run `bd list --limit 0` and check for "no beads database" error)
- Fail fast with clear error message if either check fails

### 2.12 Exit Codes (Updated)

| Code | Meaning |
|------|---------|
| 0 | Validation passed (and beads created if `--create-beads`) |
| 1 | Validation failed |
| 2 | Usage error, internal error, bd not found, bd command failure |

### 2.13 Output (Text Mode with --create-beads)

```
VALIDATION PASSED
  Tasks validated: 3
  No errors or warnings.

BEADS CREATION
  Epic created: bd-a1b2 "Task Graph: valid_task_graph.json"
  Task created: bd-c3d4 "Implement discount calculation for order totals" (calculate-discounted-total)
  Task created: bd-e5f6 "Add --format flag to the export command" (cli-export-format-flag)
  Task created: bd-g7h8 "Implement hybrid BM25 + vector search" (weaviate-hybrid-search)
  Dependency:   bd-g7h8 blocked-by bd-c3d4
  Dependency:   bd-g7h8 blocked-by bd-e5f6

  Summary: 1 epic + 3 tasks created, 2 dependencies linked.
```

### 2.14 Output (JSON Mode with --create-beads)

```json
{
  "valid": true,
  "errors": [],
  "stats": { "total_tasks": 3, "error_count": 0, "warning_count": 0, "info_count": 0 },
  "beads": {
    "epic_id": "bd-a1b2",
    "tasks": {
      "calculate-discounted-total": "bd-c3d4",
      "cli-export-format-flag": "bd-e5f6",
      "weaviate-hybrid-search": "bd-g7h8"
    },
    "dependencies_linked": 2,
    "total_created": 4
  }
}
```

### 2.15 Output (Dry-Run Mode)

```
VALIDATION PASSED
  Tasks validated: 3
  No errors or warnings.

BEADS CREATION (DRY RUN)
  [DRY-RUN] bd create --title "Task Graph: valid_task_graph.json" --type epic --priority 2 --labels taskval-managed --json --silent
  [DRY-RUN] bd create --title "Implement discount calculation..." --type task --parent <epic-id> --priority 2 --estimate 15 --labels taskval-managed --json --silent
  [DRY-RUN] bd create --title "Add --format flag..." --type task --parent <epic-id> --priority 1 --estimate 240 --labels taskval-managed --json --silent
  [DRY-RUN] bd create --title "Implement hybrid BM25..." --type task --parent <epic-id> --priority 0 --estimate 240 --labels taskval-managed --json --silent
  [DRY-RUN] bd dep add <weaviate-id> <discount-id> --type blocks
  [DRY-RUN] bd dep add <weaviate-id> <export-id> --type blocks

  Summary: Would create 1 epic + 3 tasks, link 2 dependencies.
```

---

## Part 3: TASK_CREATION_INSTRUCTIONS_AGENTS.md

### Purpose

A definitive guide for AI coding agents on how to use the `taskval` + `bd` workflow for structured task management. Lives at the project root.

### Document Structure

```
# Task Creation Instructions for AI Agents

## 1. Overview
  - What this workflow achieves (quality-gated task tracking)
  - When to use it (new features, bug fixes, refactoring plans, any multi-step work)

## 2. Prerequisites
  - taskval binary installed and on PATH
  - bd (beads) initialized in the project (bd init)
  - JSON task files conforming to the Structured Task Template Spec

## 3. Workflow A: User-Provided Tasks
  1. User describes what they want built
  2. Agent decomposes into task graph JSON conforming to the spec
  3. Agent writes JSON to a .task.json file
  4. Agent runs: taskval --mode=graph --create-beads plan.task.json
  5. On failure: read validation errors, fix the JSON, retry
  6. On success: epic + tasks are now tracked in beads
  7. Agent runs bd ready to pick up first available task

## 4. Workflow B: Existing Task Files in Repository
  1. Agent scans for task files: find . -name "*.task.json"
  2. Agent validates each: taskval --mode=graph <file>
  3. If valid and not yet in beads: taskval --mode=graph --create-beads <file>
  4. Agent uses bd ready to find available work

## 5. Writing Valid Task JSON
  - Required fields reference table (6 required, 3 contextual, 6 optional)
  - What makes each field valid
  - Common validation failures and how to fix them:
    - SCHEMA: structural errors (missing fields, wrong types)
    - V2: duplicate task_id
    - V4: dangling depends_on reference
    - V5: dependency cycle
    - V6: goal contains forbidden words (try, explore, investigate, look into)
    - V7: vague acceptance criteria (works correctly, is good, etc.)
    - V9: missing contextual field without N/A justification
    - V10: implementation task missing files_scope
  - Complete single task example
  - Complete task graph example

## 6. Task Decomposition Guidelines
  - One task = one coherent unit of work (30 min to 4 hours)
  - Goal must be a single, testable sentence starting with a verb
  - Acceptance criteria: each must be independently verifiable
  - Dependencies must form a DAG (no cycles, no self-references)
  - Use N/A with reason for non-applicable contextual fields
  - files_scope should be as narrow as possible (blast radius control)

## 7. The Validation -> Beads Pipeline
  - Full command reference for all flag combinations
  - --dry-run to preview before creating
  - What happens on success (epic + tasks + deps created)
  - What happens on validation failure (fix and retry loop)
  - What happens on bd failure (partial creation, how to clean up)

## 8. After Tasks Are Created in Beads
  - bd ready to find unblocked work
  - bd update <id> --status in_progress to claim
  - Do the work, respecting files_scope from the template metadata
  - bd close <id> --reason "Completed" to mark done
  - bd sync at end of session
  - If new tasks emerge during work: create a new .task.json, validate, create beads

## 9. Quick Reference
  - Cheat sheet of the most common commands
  - Field reference table
  - Priority and estimate mapping tables
```

---

## Part 4: `/taskify` Skill

### 4.1 What It Does

`/taskify` is a Claude Code skill that automates the full pipeline:

```
User input (spec, plan, or text description)
    |
    v
[1] Read and understand the input
    |
    v
[2] Decompose into structured task graph JSON
    (conforming to STRUCTURED_TEMPLATE_SPEC.md)
    |
    v
[3] Write task graph to .task.json file
    |
    v
[4] Run: taskval --mode=graph <file>
    |
    +--[FAIL]--> Read errors, fix JSON, retry (up to 3 times)
    |
    +--[PASS]--> Run: taskval --mode=graph --create-beads <file>
    |
    v
[5] Report summary of created beads
```

### 4.2 Usage

```bash
# From a spec or plan file
/taskify docs/oauth-spec.md
/taskify REQUIREMENTS.md

# From inline text description
/taskify "Add OAuth2 support with Google and GitHub providers, including token refresh and session management"

# From an existing task JSON (skip decomposition, just validate and create beads)
/taskify tasks/auth-feature.task.json
```

The skill detects the input type:
- If `$ARGUMENTS` ends in `.task.json` -> skip decomposition, go straight to validation
- If `$ARGUMENTS` is a valid file path -> read and decompose
- If `$ARGUMENTS` is quoted text or doesn't match a file -> treat as inline description

### 4.3 Skill Directory Structure

```
.claude/
    skills/
        taskify/
            SKILL.md                    # Main skill file (instructions + frontmatter)
            spec-reference.md           # Condensed task template spec (field definitions, validation rules)
            task-writing-guide.md       # How to write valid task JSON (from TASK_CREATION_INSTRUCTIONS_AGENTS.md)
            examples/
                single-task.json        # Complete valid single task example
                task-graph.json         # Complete valid task graph example
    agents/
        taskify-agent.md                # Custom subagent definition
```

### 4.4 SKILL.md Frontmatter

```yaml
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
```

Key design decisions:

| Setting | Value | Rationale |
|---------|-------|-----------|
| `disable-model-invocation: true` | User-only | This creates beads (side effects). User should control when it runs. |
| `context: fork` | Forked subagent | Keeps heavy spec reading, JSON composition, and retry loops out of main conversation. Returns clean summary. |
| `agent: taskify-agent` | Custom subagent | Dedicated agent with opus model, specific tools, and preloaded skills. |
| `allowed-tools` | Restricted set | Read/Write/Edit for file ops, Bash only for `taskval` and `bd` commands, Glob/Grep for finding files. |

### 4.5 Custom Subagent: taskify-agent

The skill uses `context: fork` with a custom subagent defined at `.claude/agents/taskify-agent.md`:

```yaml
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
```

**Why a separate subagent?**

- **Model control:** Forces `opus` for highest quality task decomposition regardless of the main session's model.
- **Permission mode:** `acceptEdits` so the agent can write .task.json files and run taskval/bd without per-file prompts.
- **Preloaded skill:** The taskify skill content is injected at startup so the agent has full context from the start.

### 4.6 SKILL.md Content

The body of SKILL.md (after the frontmatter) contains the step-by-step instructions the subagent follows:

````markdown
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

- If `taskval` is not found: Tell the user to build it with `go build -o taskval ./cmd/taskval/`
- If `bd` is not found: Tell the user to install beads: `go install github.com/steveyegge/beads/cmd/bd@latest`
- If `bd init` hasn't been run: Run `bd init` automatically
- If beads creation fails partway: Report what was created and what failed
````

### 4.7 Supporting File: spec-reference.md

A condensed version of STRUCTURED_TEMPLATE_SPEC.md containing only what the agent needs:

- Field definitions table (all 15 fields with types, constraints, required/optional)
- Type vocabulary (primitives, compounds, refined, domain types)
- Validation rules reference (V1-V10 with examples of pass/fail)
- JSON Schema structure (task_node and task_graph top-level shapes)
- The N/A pattern for contextual fields

Target: ~200 lines. Self-contained. No external references needed.

### 4.8 Supporting File: task-writing-guide.md

A condensed version of TASK_CREATION_INSTRUCTIONS_AGENTS.md containing:

- Task decomposition principles (granularity, independence, testability)
- Common patterns for writing good goals, acceptance criteria, inputs/outputs
- Forbidden words list and how to rephrase
- Vague terms list and how to make specific
- Priority and estimate mapping tables
- Two complete examples (single task, task graph)

Target: ~150 lines. Practical, example-heavy.

### 4.9 Supporting Files: examples/

Copy the existing validated examples from the project:

- `examples/single-task.json` -- copy of `examples/valid_single_task.json`
- `examples/task-graph.json` -- copy of `examples/valid_task_graph.json`

These serve as concrete templates the agent can reference when composing new task JSON.

### 4.10 Interaction Flow

**User invokes:**
```
/taskify docs/oauth-spec.md
```

**What happens internally:**

1. Claude Code detects `/taskify` skill
2. Forks a new subagent context using `taskify-agent` (opus model)
3. Subagent receives SKILL.md content as its task prompt, with `$ARGUMENTS` = `docs/oauth-spec.md`
4. Subagent reads `docs/oauth-spec.md`
5. Subagent consults `spec-reference.md` and `task-writing-guide.md` for field definitions
6. Subagent decomposes the spec into a task graph JSON
7. Subagent writes `.tasks/oauth-feature.task.json`
8. Subagent runs `taskval --mode=graph .tasks/oauth-feature.task.json`
9. If validation fails: reads errors, fixes JSON, retries (up to 3x)
10. On pass: runs `taskval --mode=graph --create-beads .tasks/oauth-feature.task.json`
11. Subagent returns summary to main conversation

**User sees (in main conversation):**
```
/taskify completed:

Created 5 tasks from docs/oauth-spec.md:

Epic: bd-a1b2 "OAuth2 Feature Implementation"

Tasks:
  bd-c3d4 [P1] Add Google OAuth2 provider (add-google-oauth)
  bd-e5f6 [P1] Add GitHub OAuth2 provider (add-github-oauth)
  bd-g7h8 [P1] Implement token refresh logic (implement-token-refresh)
  bd-i9j0 [P2] Add session management middleware (add-session-middleware)
  bd-k1l2 [P2] Write integration tests for OAuth flow (write-oauth-tests)

Dependencies:
  implement-token-refresh -> add-google-oauth, add-github-oauth
  add-session-middleware -> implement-token-refresh
  write-oauth-tests -> add-session-middleware

Task file saved to: .tasks/oauth-feature.task.json
Run `bd ready` to see available work.
```

---

## Implementation Order

| Step | Description | Files | Effort |
|------|-------------|-------|--------|
| 1 | Fix module path | `go.mod`, `cmd/taskval/main.go` | 10 min |
| 2 | Add `Graph` field to `ValidationResult` | `internal/validator/types.go`, `internal/validator/validate.go` | 15 min |
| 3 | Create `internal/beads/mapping.go` | New: priority/estimate mapping, description composition | 45 min |
| 4 | Create `internal/beads/exec.go` | New: bd command execution, output parsing | 30 min |
| 5 | Create `internal/beads/beads.go` | New: `Creator` struct, orchestration | 60 min |
| 6 | Update `cmd/taskval/main.go` | Add flags, wire beads creation | 30 min |
| 7 | Create `internal/beads/beads_test.go` | Tests for mapping, commands, dry-run | 45 min |
| 8 | Update existing tests | Ensure nothing breaks | 15 min |
| 9 | Write `TASK_CREATION_INSTRUCTIONS_AGENTS.md` | New: agent workflow guide | 45 min |
| 10 | Create `.claude/skills/taskify/SKILL.md` | New: main skill file | 30 min |
| 11 | Create `.claude/skills/taskify/spec-reference.md` | New: condensed spec reference | 45 min |
| 12 | Create `.claude/skills/taskify/task-writing-guide.md` | New: task writing guide | 30 min |
| 13 | Copy examples into skill directory | `.claude/skills/taskify/examples/` | 5 min |
| 14 | Create `.claude/agents/taskify-agent.md` | New: custom subagent | 10 min |
| 15 | Manual integration testing | Test full `/taskify` flow end-to-end | 45 min |
| 16 | Update `README.md` and `CLI_COMMAND_REFERENCE.md` | Document new flags, beads workflow, /taskify | 30 min |

**Total estimated effort: ~8 hours** (5.5h for Parts 1-3 + 2.5h for Part 4)

---

## Design Decisions Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI architecture | Built into `taskval` | Single tool for agents; `--create-beads` is natural extension of "validate then act" |
| bd integration method | Shell out via `os/exec` | Simple, no library dependency, requires only `bd` on PATH |
| Graph structure in Beads | Epic + child tasks | Mirrors task graph; epic holds graph-level context; `parent-child` deps |
| Single task support | Yes, both modes | Consistent behavior; single task creates one bead without epic |
| Template metadata storage | `bd update --design` | `metadata` not settable via CLI; `design` is appropriate; keeps `notes` clean |
| Dry-run support | Yes, `--dry-run` flag | Essential for review; useful for debugging and CI |
| Module path | Fix now | Correct technical debt before adding features |
| Parsed graph in result | `Graph *TaskGraph` with `json:"-"` | Avoids double-parsing; excluded from JSON output |
| Rollback on failure | No rollback | Beads issues can be manually cleaned; partial creation better than silent rollback |
| ID format | Template `task_id` in `_template` metadata | Beads generates hash IDs; template IDs preserved for traceability |
| Instructions location | Project root | Alongside README, SPEC, CLI_COMMAND_REFERENCE |
| Skill scope | Project-level (`.claude/skills/`) | Committed with repo; available to anyone cloning the project |
| Skill execution | Forked subagent | Isolates heavy work (spec reading, JSON composition, retry loops) from main conversation |
| Subagent model | opus | Highest quality task decomposition; worth the cost for planning accuracy |
| Reference material | Supporting files in skill dir | Self-contained; spec-reference.md and task-writing-guide.md bundled with skill |
| Input flexibility | Both file and text | `/taskify plan.md` reads file; `/taskify "Add OAuth2..."` accepts inline text |
| Validation retry | Auto-retry up to 3 times | Subagent reads errors, fixes JSON, retries without user intervention |
| Model invocation | `disable-model-invocation: true` | Creates beads (side effects); user should explicitly control when it runs |
| Task file location | `.tasks/` directory | Organized, discoverable, separates task specs from source code |
