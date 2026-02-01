# Compatibility Analysis: Structured Task Templates x Beads (bd)

**Version:** 0.1.0
**Status:** Draft  
**Date:** 2026-01-31

---

## 1. Executive Summary

This document analyses the compatibility between our **Structured Task Template** specification (v0.2.0) and **Beads (bd)**, Steve Yegge's distributed, git-backed graph issue tracker for AI coding agents.

**Core finding:** The two systems are complementary, not competing. They occupy different positions in the task lifecycle:

| Dimension | Structured Templates | Beads (bd) |
|-----------|---------------------|------------|
| **Purpose** | Define *what to build* with machine-verifiable quality | Track *execution state* across agents and sessions |
| **Phase** | Authoring / planning | Execution / coordination |
| **Validation** | Schema + semantic checks at write time | Runtime status lifecycle + dependency resolution |
| **Persistence** | Static JSON documents | Git-backed JSONL with SQLite cache |
| **Agent model** | Single agent consumes one task | Multi-agent swarm with session handoff |

Templates ensure task quality *before* work begins. Beads manages task state *while* work executes. A validated template can produce high-quality Beads issues; Beads issues can be validated retroactively against template quality rules. Neither replaces the other.

**Significance: HIGH.** Integration is feasible with moderate effort and would give us authoring-time quality guarantees (templates) combined with runtime execution tracking (Beads) -- a capability neither system provides alone.

---

## 2. Beads Overview

### 2.1 Architecture

Beads stores issues as append-only JSONL in `.beads/issues.jsonl`, backed by a local SQLite cache for fast queries. It uses git as the distribution mechanism: issues are committed, branched, and merged alongside code. Hash-based IDs (`bd-a1b2`) prevent merge collisions in multi-agent workflows.

Key architectural decisions:
- **Git as database** -- no external service, works offline, branches isolate work
- **JSONL as interchange** -- human-readable, diffable, git-friendly
- **SQLite as query engine** -- fast local operations, rebuilt from JSONL on import
- **Daemon for auto-sync** -- background export/commit/push with 30-second debounce

### 2.2 Data Model

The Beads `Issue` struct contains approximately 70+ fields organized into 18 logical groups:

| Group | Fields | Purpose |
|-------|--------|---------|
| Core Identification | `id`, `content_hash` | Unique identity, dedup |
| Content | `title`, `description`, `design`, `acceptance_criteria`, `notes` | What the work is |
| Status & Workflow | `status`, `priority`, `issue_type` | Current state |
| Assignment | `assignee`, `owner`, `estimated_minutes` | Who does it |
| Timestamps | `created_at`, `updated_at`, `closed_at`, `close_reason` | When things happened |
| Scheduling | `due_at`, `defer_until` | Time-based coordination |
| External Integration | `external_ref`, `source_system` | Cross-system links |
| Custom Metadata | `metadata` (JSON blob) | Arbitrary extension point |
| Compaction | `compaction_level`, `compacted_at`, `original_size` | Memory decay |
| Labels | `labels[]` | Tagging |
| Dependencies | `dependencies[]` (typed edges) | Graph relationships |
| Comments | `comments[]` | Discussion |
| Tombstones | `deleted_at`, `deleted_by`, `delete_reason` | Soft-delete |
| Messaging | `sender`, `ephemeral` | Inter-agent communication |
| Agent Identity | `hook_bead`, `role_bead`, `agent_state`, `role_type`, `rig` | Agent-as-bead |
| Molecule/Swarm | `mol_type`, `work_type`, `bonded_from[]` | Swarm coordination |
| Gates/Slots | `await_type`, `await_id`, `timeout`, `holder` | Async coordination |
| HOP Entity Tracking | `creator`, `validations[]`, `quality_score`, `crystallizes` | CV chains |

### 2.3 Agent Workflow

Beads defines a structured agent session protocol:

1. `bd ready` -- find unblocked work (dependency-aware)
2. `bd update <id> --status in_progress` -- claim work
3. Execute the task
4. `bd close <id>` -- mark complete
5. `bd sync` -- flush to JSONL, commit, push

The `bd ready` command is the key differentiator: it automatically resolves the dependency graph and surfaces only tasks whose blockers are all closed. This eliminates manual dependency tracking that plagues markdown-based planning.

### 2.4 Key Differentiators

**What makes Beads distinctive for AI agent workflows:**

1. **Dependency-aware scheduling** -- `bd ready` computes the frontier of unblocked work
2. **18+ typed dependency edges** -- not just "blocks", but parent-child, conditional-blocks, waits-for, replies-to, authored-by, attests, etc.
3. **Compaction** -- semantic "memory decay" summarizes old closed tasks to save context window budget
4. **Multi-agent coordination** -- agent state tracking, molecule types (swarm/patrol/work), gates for async coordination, slots for exclusive access
5. **Git-native distribution** -- no server, no API, just git push/pull
6. **Hierarchical IDs** -- `bd-a3f8.1.1` for epic/task/subtask decomposition

---

## 3. Field Mapping

### 3.1 Template Fields to Beads Issue Fields

| Template Field | Category | Beads Field | Mapping Quality | Notes |
|----------------|----------|-------------|-----------------|-------|
| `task_id` | Required | `id` | **Direct** | Format differs: our kebab-case (`add-auth-middleware`) vs Beads hash (`bd-a1b2`). Beads IDs are auto-generated; ours are human-authored. |
| `task_name` | Required | `title` | **Direct** | Our constraint (imperative phrase, max 80 chars, starts with verb) is stricter than Beads (max 500 chars, any format). |
| `goal` | Required | `description` | **Semantic** | Our `goal` is a single testable sentence with forbidden words. Beads `description` is free-form markdown. Quality degrades without template validation. |
| `inputs` | Required | `metadata` | **Transform** | No native Beads field. Must serialize `[{name, type, constraints, source}]` into the `metadata` JSON blob. |
| `outputs` | Required | `metadata` | **Transform** | Same as inputs -- no native field, serialize to `metadata`. |
| `acceptance` | Required | `acceptance_criteria` | **Direct** | Both systems have this field. Beads stores it as a single string; our spec uses `list<string>` with per-item verifiability rules. Join with newlines on export. |
| `depends_on` | Contextual | `dependencies` | **Subset** | Our simple `[task_id, ...]` maps to Beads `Dependency` with `type: "blocks"`. Beads supports 18+ edge types; ours only models blocking. |
| `constraints` | Contextual | `description` or `metadata` | **Lossy** | No native field. Append to description as a "## Constraints" section, or store in `metadata`. |
| `files_scope` | Contextual | `metadata` | **Transform** | No native field. Store in `metadata` as `{"files_scope": ["src/**/*.go"]}`. |
| `non_goals` | Optional | `description` or `metadata` | **Lossy** | Append as "## Non-Goals" section in description, or store in `metadata`. |
| `effects` | Optional | `metadata` | **Transform** | No native field. Serialize `[{type, target}]` to `metadata`. |
| `error_cases` | Optional | `metadata` | **Transform** | No native field. Serialize `[{condition, behavior, output}]` to `metadata`. |
| `priority` | Optional | `priority` | **Transform** | Our enum (`critical`, `high`, `medium`, `low`) maps to Beads int (0-4). Mapping: critical=0, high=1, medium=2, low=3. Our `low` has no Beads equivalent for "backlog" (4). |
| `estimate` | Optional | `estimated_minutes` | **Transform** | Our enum (`trivial`, `small`, `medium`, `large`, `unknown`) must be converted to minutes. Requires project-specific mapping (e.g., trivial=15, small=60, medium=240, large=480). |
| `notes` | Optional | `notes` | **Direct** | Both are free-text fields. Direct mapping. |

### 3.2 Mapping Quality Summary

| Quality | Count | Fields |
|---------|-------|--------|
| **Direct** (1:1) | 4 | task_id, task_name, acceptance, notes |
| **Semantic** (same concept, different rules) | 1 | goal -> description |
| **Transform** (requires conversion) | 6 | inputs, outputs, files_scope, effects, error_cases, estimate |
| **Lossy** (information lost without metadata) | 2 | constraints, non_goals |
| **Subset** (our model is a strict subset) | 1 | depends_on |
| **No mapping needed** | 1 | priority (enum -> int) |

**Conclusion:** 4 of 15 fields map directly. 7 require the `metadata` JSON blob as an escape hatch. The `metadata` field is Beads' explicit extension point for this purpose -- it accepts arbitrary JSON and is validated for well-formedness. This is the correct integration surface.

### 3.3 Beads Fields with No Template Equivalent

Approximately 55+ Beads fields have no template equivalent. These fall into categories our spec intentionally does not address:

| Category | Example Fields | Why Templates Don't Cover This |
|----------|---------------|-------------------------------|
| Runtime State | `status`, `assignee`, `agent_state` | Templates define work, not track execution |
| Timestamps | `created_at`, `updated_at`, `closed_at` | Generated at runtime |
| Compaction | `compaction_level`, `compacted_at` | Runtime optimization |
| Agent Identity | `hook_bead`, `role_bead`, `rig` | Multi-agent runtime concern |
| Swarm Coordination | `mol_type`, `work_type`, `bonded_from` | Execution topology |
| Async Primitives | `await_type`, `timeout`, `holder` | Runtime synchronization |
| HOP Tracking | `creator`, `validations`, `quality_score` | Post-execution attribution |
| Soft Delete | `deleted_at`, `delete_reason` | Lifecycle management |

This is not a gap -- it is the expected boundary between a *specification format* and a *runtime tracker*.

---

## 4. Dependency Model Comparison

### 4.1 Model Structures

**Structured Templates:**
```json
{
  "depends_on": ["setup-database", "create-user-model"]
}
```
- Simple list of task IDs
- Implicit semantics: "blocks" (must complete before this task starts)
- DAG enforced by `taskval` (Kahn's algorithm for cycle detection)
- Self-dependencies rejected

**Beads:**
```json
{
  "dependencies": [
    {"issue_id": "bd-a1b2", "depends_on_id": "bd-c3d4", "type": "blocks"},
    {"issue_id": "bd-a1b2", "depends_on_id": "bd-e5f6", "type": "parent-child"},
    {"issue_id": "bd-a1b2", "depends_on_id": "bd-g7h8", "type": "related"}
  ]
}
```
- Typed edges with metadata
- 18+ relationship types
- Only 4 types affect `bd ready` (blocks, parent-child, conditional-blocks, waits-for)
- Custom edge types allowed (up to 50 chars)

### 4.2 Edge Type Coverage

| Beads Dependency Type | Affects Ready? | Template Equivalent | Notes |
|-----------------------|----------------|--------------------|----|
| `blocks` | Yes | `depends_on` | Direct match |
| `parent-child` | Yes | None | Templates have milestones but no parent-child hierarchy |
| `conditional-blocks` | Yes | None | "B runs only if A fails" -- not expressible in templates |
| `waits-for` | Yes | None | Fanout gate for dynamic children |
| `related` | No | None | Informational link |
| `discovered-from` | No | None | Provenance tracking |
| `replies-to` | No | None | Conversation threading |
| `relates-to` | No | None | Knowledge graph edge |
| `duplicates` | No | None | Deduplication |
| `supersedes` | No | None | Version chain |
| `authored-by` | No | None | HOP entity relationship |
| `assigned-to` | No | None | Assignment relationship |
| `approved-by` | No | None | Approval chain |
| `attests` | No | None | Skill attestation |
| `tracks` | No | None | Convoy cross-project reference |
| `until` | No | None | Active-until-target-closes |
| `caused-by` | No | None | Audit trail |
| `validates` | No | None | Approval/validation |
| `delegated-from` | No | None | Delegation chain |

**Significance: Our dependency model is a strict subset of Beads.** Our `depends_on` maps cleanly to Beads' `blocks` type. The 17 additional Beads edge types represent runtime coordination, provenance tracking, and multi-agent relationships that are outside the scope of task specification.

### 4.3 Implications for Integration

When exporting validated templates to Beads:
- Each `depends_on` reference becomes a `Dependency{type: "blocks"}`
- Template milestones could map to `parent-child` dependencies (epic -> tasks)
- No information is lost; the mapping is injective (one-to-one, into)

When validating Beads issues against template rules:
- Only `blocks` dependencies are relevant for DAG validation
- Other edge types should be ignored by `taskval`
- Beads issues may have valid cycles through non-blocking edges (e.g., `related`, `replies-to`) -- these must not trigger cycle detection

---

## 5. Workflow Gap Analysis

### 5.1 What Templates Provide That Beads Does Not

| Capability | Template Implementation | Beads Status | Significance |
|------------|------------------------|--------------|--------------|
| **Goal quality enforcement** | Forbidden words (try, explore, investigate), activity phrasing check | No equivalent -- `title` and `description` accept any string | **HIGH** -- prevents vague, untestable goals that waste agent context |
| **Acceptance criteria structure** | `list<string>`, each independently verifiable, vagueness detection (V7) | Single string field, no quality checks | **HIGH** -- unclear acceptance criteria are the #1 cause of agent-human misalignment |
| **Input/Output contracts** | Typed `{name, type, constraints, source}` specs | No equivalent (free-form description) | **MEDIUM** -- enables mechanical verification of task completion |
| **Effects declaration** | Explicit `{type, target}` for DB, network, filesystem, subprocess | No equivalent | **MEDIUM** -- enables safety review, sandboxing decisions |
| **Error case specification** | `{condition, behavior, output}` triples | No equivalent | **LOW** -- useful for robustness but not strictly required |
| **Schema validation** | JSON Schema (Draft 2020-12) with `oneOf` for contextual N/A | Beads validates via Go struct methods (title length, priority range, status enum) | **MEDIUM** -- catches structural errors before they become runtime surprises |
| **Constraint language** | Formal comparison/logical/set operators, field references | No equivalent | **LOW** -- rarely used in practice |
| **Blast radius control** | `files_scope` with glob patterns | No equivalent | **HIGH** -- prevents agents from modifying files outside their mandate |

### 5.2 What Beads Provides That Templates Do Not

| Capability | Beads Implementation | Template Status | Significance |
|------------|---------------------|-----------------|--------------|
| **Dependency-aware scheduling** | `bd ready` computes unblocked frontier | Templates define dependencies but don't schedule | **CRITICAL** -- without this, agents must manually resolve execution order |
| **Status lifecycle** | open -> in_progress -> blocked -> closed -> tombstone | No runtime state | **CRITICAL** -- essential for multi-session work |
| **Multi-agent coordination** | Agent state, molecules, gates, slots, swarm types | No agent model | **HIGH** -- required for parallel agent execution |
| **Context compaction** | Semantic summarization of old closed tasks | No compaction | **HIGH** -- prevents context window exhaustion on long-horizon projects |
| **Git-backed persistence** | JSONL + SQLite, auto-sync, merge-safe hash IDs | Static JSON files | **HIGH** -- templates have no persistence story |
| **Hierarchical decomposition** | `bd-a3f8.1.1` hierarchical IDs, parent-child | Flat task graph with milestones | **MEDIUM** -- Beads decomposition is more natural |
| **Session handoff** | "Land the plane" protocol, `bd sync`, daemon | No session concept | **MEDIUM** -- enables multi-session continuity |
| **Typed relationships** | 18+ edge types beyond blocking | Only `blocks` | **MEDIUM** -- richer graph for knowledge and provenance |
| **Audit trail** | Events table (created, updated, commented, etc.) | No history | **LOW** -- useful for debugging but not for task definition |

### 5.3 Gap Summary

The systems have **zero overlap** in their core value propositions:

- **Templates**: Authoring-time quality guarantees (goal clarity, acceptance verifiability, input/output contracts, blast radius control)
- **Beads**: Runtime execution management (scheduling, status, multi-agent coordination, persistence, compaction)

This clean separation means integration adds value without redundancy.

---

## 6. Integration Architecture

### 6.1 Path A: Templates as Planning Layer (Recommended)

**Flow:** LLM decomposes work -> produces task graph JSON -> `taskval` validates -> approved tasks feed into `bd create`

```
 User Request
      |
      v
 LLM Planning Agent
      |
      v
 Task Graph JSON ------> taskval --mode=graph
      |                        |
      |                   [PASS]  [FAIL]
      |                     |        |
      |                     v        v
      |               bd create   LLM refines
      |               (for each   (feedback loop)
      |                task)
      v
 Beads manages execution
 (bd ready, status, sync)
```

**Implementation sketch:**

```bash
# 1. LLM produces task graph
llm-agent plan "Add OAuth2 support" > plan.json

# 2. Validate
taskval --mode=graph plan.json
# Exit 0 = pass

# 3. Convert to Beads issues
task2bd plan.json
# Creates bd issues with:
#   title     <- task_name
#   description <- goal + constraints + non_goals (as sections)
#   acceptance_criteria <- acceptance (joined)
#   priority  <- priority (enum to int)
#   metadata  <- {inputs, outputs, effects, error_cases, files_scope}
#   dependencies <- depends_on (as "blocks" type)
```

**Advantages:**
- Clean separation of concerns
- Template validation catches quality issues before they enter Beads
- Beads handles everything after planning (scheduling, status, sync)
- No modifications needed to Beads itself

**Effort:** ~2-3 days to build `task2bd` CLI tool

### 6.2 Path B: Beads Metadata Extension

**Flow:** Store validated template fields in Beads' `metadata` JSON blob, enabling post-hoc quality checks and richer agent context.

```json
{
  "id": "bd-a1b2",
  "title": "Add rate limiting middleware",
  "description": "Add rate limiting to all public API endpoints...",
  "acceptance_criteria": "Rate limiter returns 429 after threshold...",
  "metadata": {
    "_template": {
      "version": "0.2.0",
      "task_id": "add-rate-limiting",
      "inputs": [
        {"name": "request", "type": "http.Request", "constraints": "non-null"}
      ],
      "outputs": [
        {"name": "response", "type": "http.Response", "constraints": "429 or passthrough"}
      ],
      "effects": [
        {"type": "DB.Read", "target": "rate_limit_counters"},
        {"type": "DB.Write", "target": "rate_limit_counters"}
      ],
      "files_scope": ["internal/middleware/*.go", "internal/ratelimit/*.go"],
      "error_cases": [
        {"condition": "Redis unavailable", "behavior": "fail open", "output": "passthrough"}
      ]
    }
  }
}
```

**Advantages:**
- No new tools needed beyond `taskval` and a thin converter
- Template data travels with the issue through git sync
- Agents can read `metadata._template.files_scope` to self-limit blast radius
- Enables `taskval --mode=beads-audit` to validate existing Beads issues against template quality rules

**Disadvantages:**
- Metadata is opaque to Beads -- no native queries on template fields
- Requires agents to know about the `_template` metadata convention

**Effort:** ~1 day for converter, ~1 day for audit mode in `taskval`

### 6.3 Path C: Bidirectional Validation

**Flow:** `taskval` learns to read Beads issues (via `bd show --json`) and validate them against template quality rules.

```bash
# Validate a Beads issue against template quality rules
bd show bd-a1b2 --json | taskval --mode=beads-issue

# Validate all ready work
bd ready --json | taskval --mode=beads-batch

# Output:
# bd-a1b2: WARN V6: goal contains forbidden word "try"
# bd-a1b2: WARN V7: acceptance_criteria contains vague term "should work"
# bd-c3d4: PASS (all quality checks)
```

**Advantages:**
- Works with existing Beads issues (no upfront migration)
- Catches quality problems in issues created without template discipline
- Can be integrated into CI/CD or agent pre-flight checks

**Disadvantages:**
- Many template fields (inputs, outputs, effects) won't exist in Beads issues unless Path B is also used
- Limited to validating fields that Beads natively stores (title, description, acceptance_criteria)

**Effort:** ~1-2 days to add Beads issue parsing to `taskval`

### 6.4 Recommended Integration Path

**Start with Path A + B combined.** This gives:

1. **Quality gate at planning time** -- LLM produces template, `taskval` validates, rejects vague goals and untestable acceptance criteria
2. **Rich metadata in Beads** -- template fields preserved in `metadata._template` for agent consumption
3. **Full runtime management** -- Beads handles everything after planning

Path C can be added later as a quality audit tool for existing Beads issues.

---

## 7. Suitability Matrix

### 7.1 When to Use Each System

| Scenario | Templates Only | Beads Only | Templates + Beads |
|----------|---------------|------------|-------------------|
| **Single agent, single session** | Sufficient | Overkill | Overkill |
| **Single agent, multi-session** | Insufficient (no state) | Sufficient | Ideal |
| **Multi-agent coordination** | Insufficient | Sufficient | Ideal |
| **High-stakes tasks (security, data)** | Good for spec quality | Missing quality gates | **Best** |
| **Rapid prototyping** | Overkill | Sufficient | Overkill |
| **Large codebase refactoring** | Good for planning | Good for execution | **Best** |
| **LLM task decomposition** | Core use case | Not designed for this | **Best** |
| **Backlog triage** | Not applicable | Core use case | Beads only |
| **Agent session handoff** | Not applicable | Core use case | Beads only |
| **CI/CD quality gate** | Good (validate PR-linked tasks) | Not designed for this | Templates only |

### 7.2 Decision Framework

Use **templates alone** when:
- You need a quality gate for LLM-generated task decompositions
- The work is single-session and doesn't need persistent tracking
- You're building a CI/CD validation pipeline for task specifications

Use **Beads alone** when:
- You need runtime issue tracking for multi-agent or multi-session work
- Task quality is less critical than execution coordination
- The project already uses Beads and adding templates would create friction

Use **both** when:
- High-stakes work where vague goals or unclear acceptance criteria are costly
- LLM agents decompose large features into task graphs that feed into execution
- You want authoring-time quality guarantees *and* runtime execution management
- Multiple agents collaborate on a shared task graph over multiple sessions

---

## 8. Recommendation

### 8.1 Immediate Action

Build a `task2bd` CLI tool that:
1. Reads a validated task graph JSON (output of `taskval --mode=graph`)
2. Creates Beads issues via `bd create` with proper field mapping
3. Stores template-specific fields in `metadata._template`
4. Creates `blocks` dependencies for each `depends_on` reference
5. Maps milestones to Beads epics with `parent-child` dependencies

### 8.2 Module Path Fix

Before building new tooling, fix the module path mismatch:
- Current: `github.com/foundry-zero/task-templating`
- Actual repo: `github.com/nixlim/task_templating`

This was flagged in the previous session but not yet resolved.

### 8.3 Future Extensions

1. **`taskval --mode=beads-audit`** -- validate existing Beads issues against template quality rules (Path C)
2. **`bd recipe` integration** -- register `taskval` as a Beads recipe so `bd prime` includes template validation context
3. **Compaction-aware templates** -- when Beads compacts old issues, preserve `metadata._template` fields in the summary
4. **LLM feedback loop** -- when `taskval` rejects a template, pipe the validation errors back to the LLM with structured guidance for refinement (already documented in spec Section 11, extend to Beads workflow)

### 8.4 Effort Estimate

| Item | Effort | Priority |
|------|--------|----------|
| Fix module path | 30 minutes | High |
| `task2bd` CLI tool | 2-3 days | High |
| Beads audit mode in `taskval` | 1-2 days | Medium |
| Beads recipe integration | 0.5 days | Low |
| Compaction-aware metadata | 1 day | Low |

**Total estimated effort for core integration (Path A + B): 3-4 days.**

---

## Appendix A: Priority Mapping

| Template `priority` | Beads `priority` | Meaning |
|---------------------|------------------|---------|
| `critical` | `0` | Security, data loss, broken builds |
| `high` | `1` | Major features, important bugs |
| `medium` | `2` | Default, nice-to-have |
| `low` | `3` | Polish, optimization |
| *(none)* | `4` | Backlog (no template equivalent) |

## Appendix B: Estimate Mapping (Suggested)

| Template `estimate` | Suggested Minutes | Rationale |
|---------------------|-------------------|-----------|
| `trivial` | 15 | Config change, typo fix |
| `small` | 60 | Single function, single file |
| `medium` | 240 | Multi-file feature, ~half day |
| `large` | 480 | Multi-day feature, cross-cutting |
| `unknown` | *(omit)* | Do not set `estimated_minutes` |

## Appendix C: Metadata Schema Convention

Template fields stored in Beads `metadata` should use the `_template` namespace to avoid collisions with other metadata consumers:

```json
{
  "metadata": {
    "_template": {
      "version": "0.2.0",
      "task_id": "original-template-task-id",
      "inputs": [...],
      "outputs": [...],
      "effects": [...],
      "error_cases": [...],
      "files_scope": [...],
      "constraints": [...],
      "non_goals": [...]
    }
  }
}
```

The `_` prefix follows the convention of internal/reserved namespaces and signals to other metadata consumers that this key is managed by the template system.
