# Taskify Launcher: Installation Script Plan

## Goal

A single shell script (`scripts/install-taskify.sh`) that bootstraps the
`/taskify` skill into any git repository. After running it, the target project
has a fully functional `/taskify` slash command in Claude Code with `taskval`
for validation and `bd` for issue tracking.

Additionally, a `/init` skill command in the task_templating project itself
invokes this script against a target directory.

---

## 1. What Gets Installed

### 1.1 Files created in the target project

```
<target-project>/
  .claude/
    agents/
      taskify-agent.md          # Subagent definition (opus model, acceptEdits)
    skills/
      taskify/
        SKILL.md                # Main skill definition (patched for go install)
        spec-reference.md       # Structured Task Template Spec quick reference
        task-writing-guide.md   # Practical guide for writing good tasks
        examples/
          single-task.json      # Complete single task example
          task-graph.json       # Complete task graph example
  AGENTS.md                     # Created or appended with taskify section
  CLAUDE.md                     # Created or appended with taskify section
```

### 1.2 Binaries installed (if missing)

| Binary    | Install command                                                          | Purpose                |
|-----------|--------------------------------------------------------------------------|------------------------|
| `taskval` | `go install github.com/nixlim/task_templating/cmd/taskval@latest`        | Task graph validation  |
| `bd`      | `go install github.com/steveyegge/beads/cmd/bd@latest`                   | Issue tracking         |

### 1.3 Beads initialisation

If `.beads/` does not exist in the target project, the script runs `bd init`.

---

## 2. Script Design

### 2.1 Invocation

```bash
# From anywhere, targeting a specific project:
bash /path/to/install-taskify.sh --target /path/to/project

# From inside the target project:
bash /path/to/install-taskify.sh

# Piped from curl (for remote installation):
curl -sL https://raw.githubusercontent.com/nixlim/task_templating/main/scripts/install-taskify.sh | bash

# With flags:
bash install-taskify.sh --force              # Overwrite existing skill files
bash install-taskify.sh --skip-beads         # Don't install or initialise beads
bash install-taskify.sh --dry-run            # Show what would be done
bash install-taskify.sh --target /path/to/dir
```

### 2.2 Flags

| Flag            | Default | Description                                                        |
|-----------------|---------|--------------------------------------------------------------------|
| `--target DIR`  | `.`     | Target project directory                                           |
| `--force`       | off     | Overwrite existing skill files (upgrade path)                      |
| `--skip-beads`  | off     | Skip `bd` installation and `bd init`                               |
| `--dry-run`     | off     | Print actions without executing them                               |
| `--help`        | -       | Show usage information                                             |

### 2.3 Script Structure (top to bottom)

```
Section 1: HEADER & USAGE           (~25 lines)
  - Shebang: #!/usr/bin/env bash
  - set -euo pipefail
  - Version string
  - usage() function

Section 2: FLAG PARSING              (~35 lines)
  - getopts-style loop for --target, --force, --skip-beads, --dry-run, --help

Section 3: UTILITY FUNCTIONS          (~40 lines)
  - log()     — green prefix "[taskify]"
  - warn()    — yellow prefix "[taskify]"
  - die()     — red prefix "[taskify]", exit 1
  - run()     — execute or print in dry-run mode
  - need_cmd() — check a command exists or install it
  - file_has_marker() — check if a file contains the taskify marker

Section 4: PREREQUISITE CHECKS        (~50 lines)
  - Verify Go is on PATH (required for go install)
  - Verify target is a git repository (warn if not)
  - Install taskval if missing: go install github.com/nixlim/task_templating/cmd/taskval@latest
  - Install bd if missing (unless --skip-beads): go install github.com/steveyegge/beads/cmd/bd@latest
  - Run bd init if .beads/ missing (unless --skip-beads)

Section 5: FILE CREATION              (~640 lines)
  - Create directories: .claude/agents/, .claude/skills/taskify/examples/
  - Write each file via heredoc (skip if exists and --force not set)
  - Files written:
    a) .claude/agents/taskify-agent.md
    b) .claude/skills/taskify/SKILL.md  (with patched error recovery)
    c) .claude/skills/taskify/spec-reference.md
    d) .claude/skills/taskify/task-writing-guide.md
    e) .claude/skills/taskify/examples/single-task.json
    f) .claude/skills/taskify/examples/task-graph.json

Section 6: AGENTS.MD UPDATE           (~35 lines)
  - If AGENTS.md does not exist: create with taskify section
  - If AGENTS.md exists but has no marker: append taskify section
  - If AGENTS.md has marker: skip (already installed)
  - Marker: <!-- taskify-begin --> / <!-- taskify-end -->

Section 7: CLAUDE.MD UPDATE           (~25 lines)
  - Same logic as AGENTS.md with its own marker pair
  - Creates project-level CLAUDE.md (not ~/.claude/CLAUDE.md)

Section 8: SUMMARY                    (~25 lines)
  - List what was created/skipped
  - Print next steps
```

**Estimated total: ~875 lines**

### 2.4 Idempotency Guarantees

The script is safe to run multiple times:

| Action                        | First run         | Subsequent runs (no --force) | With --force           |
|-------------------------------|-------------------|------------------------------|------------------------|
| Install taskval               | Installed         | Skipped (already on PATH)    | Skipped                |
| Install bd                    | Installed         | Skipped (already on PATH)    | Skipped                |
| Run bd init                   | Initialised       | Skipped (.beads/ exists)     | Skipped                |
| Create skill files            | Created           | Skipped (files exist)        | Overwritten            |
| Append to AGENTS.md           | Created/appended  | Skipped (marker present)     | Skipped (marker-based) |
| Append to CLAUDE.md           | Created/appended  | Skipped (marker present)     | Skipped (marker-based) |

**Note:** `--force` overwrites skill files but does NOT re-append to AGENTS.md or
CLAUDE.md if the marker is already present. To update those, remove the markers
manually first.

### 2.5 No Git Commits

The script creates and modifies files but never runs `git add` or `git commit`.
The user decides when and how to commit the changes.

---

## 3. SKILL.md Patch

The installed SKILL.md is identical to the source in this repository except for
the error recovery section. The original says:

```
- If `taskval` is not found: Tell the user to build it with `go build -o taskval ./cmd/taskval/`
```

The installed version says:

```
- If `taskval` is not found: Tell the user to install it with `go install github.com/nixlim/task_templating/cmd/taskval@latest`
```

This is the only difference. The patch ensures that the error recovery
instructions make sense in repositories that don't contain the taskval source.

---

## 4. Content Injected into AGENTS.md

The following section is appended (or used as the initial content):

```markdown
<!-- taskify-begin -->
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
<!-- taskify-end -->
```

---

## 5. Content Injected into CLAUDE.md

The following section is appended (or used as the initial content):

```markdown
<!-- taskify-begin -->
## Task Management with Taskify

- Use `/taskify` to decompose specs, plans, or descriptions into structured task graphs
- Task graphs are saved to `.tasks/` and validated with `taskval`
- Issues are tracked with beads (`bd`). Run `bd ready` for available work.
- Run `bd sync` at session end to sync beads state with git
<!-- taskify-end -->
```

---

## 6. The `/init` Command

In addition to the standalone shell script, the task_templating project itself
provides a `/init` skill that invokes the installer. This is useful when
Claude Code is already running in the task_templating project and the user
wants to bootstrap taskify into another project.

### 6.1 Skill definition

File: `.claude/skills/init/SKILL.md`

```yaml
---
name: init
description: >
  Install the taskify skill into a target project directory. Creates .claude/skills/taskify/,
  installs taskval and bd, initialises beads, and updates AGENTS.md and CLAUDE.md.
argument-hint: <target-project-directory>
disable-model-invocation: true
---
```

The skill body instructs Claude to run `scripts/install-taskify.sh` with the
provided arguments.

### 6.2 Behaviour

```
/init /path/to/my-project              # Install into specific directory
/init /path/to/my-project --force      # Overwrite existing files
/init /path/to/my-project --skip-beads # Skip beads installation
```

The `/init` command delegates entirely to `scripts/install-taskify.sh`. It
passes `$ARGUMENTS` directly as flags to the script.

---

## 7. Claude Code CLI References

The skill system used by this installer is documented in the official Claude
Code documentation. Key references:

### 7.1 Skills

**Source:** https://docs.anthropic.com/en/docs/claude-code/skills

Skills are defined as `SKILL.md` files in `.claude/skills/<name>/` directories.
Key points relevant to this installer:

- **Skill resolution order:** Enterprise > Personal (`~/.claude/skills/`) >
  Project (`.claude/skills/`). The installer creates project-level skills.
- **Directory structure:** Each skill is a directory with `SKILL.md` as the
  entry point, plus optional supporting files (reference docs, examples, scripts).
- **Frontmatter fields used by taskify:**
  - `name` — Skill identifier, becomes the `/slash-command` name
  - `description` — Tells Claude when to load the skill automatically
  - `argument-hint` — Shown during autocomplete (e.g., `<spec-file or description>`)
  - `disable-model-invocation` — Set to `true` to prevent Claude from auto-triggering
  - `context: fork` — Runs the skill in an isolated subagent context
  - `agent` — Which subagent to use when `context: fork` is set
  - `allowed-tools` — Tool whitelist for the skill
- **`$ARGUMENTS` substitution:** When a user types `/taskify my-spec.md`, the
  string `my-spec.md` replaces `$ARGUMENTS` in SKILL.md content.
- **Supporting files:** Referenced via relative markdown links in SKILL.md
  (e.g., `[spec-reference.md](spec-reference.md)`). Claude loads them when needed.

### 7.2 Subagents (Agents)

**Source:** https://docs.anthropic.com/en/docs/claude-code/sub-agents

The taskify skill uses a custom subagent (`taskify-agent`) defined in
`.claude/agents/taskify-agent.md`. Key points:

- **Agent file location:** `.claude/agents/` for project-level,
  `~/.claude/agents/` for user-level. The installer creates project-level agents.
- **Frontmatter fields used by taskify-agent:**
  - `name` — Unique identifier (`taskify-agent`)
  - `description` — When Claude should delegate to this agent
  - `tools` — Tool whitelist: `Read, Write, Edit, Bash, Glob, Grep`
  - `model` — `opus` for high-quality task decomposition
  - `permissionMode` — `acceptEdits` to auto-accept file writes
  - `skills` — List of skills preloaded into the agent's context (`taskify`)
- **How it connects:** The SKILL.md sets `agent: taskify-agent` and
  `context: fork`, so when `/taskify` is invoked, Claude spawns the
  taskify-agent subagent with the skill content as its prompt.
- **Preloaded skills:** The `skills: [taskify]` field in the agent frontmatter
  injects the full SKILL.md content into the subagent's context at startup,
  not just the description.

### 7.3 Memory Files (CLAUDE.md)

**Source:** https://docs.anthropic.com/en/docs/claude-code/memory

- **Project-level CLAUDE.md** at the repository root is loaded into context
  for every conversation in that project.
- The installer creates/appends to this file with taskify-related instructions.
- The user's personal `~/.claude/CLAUDE.md` is NOT modified.

### 7.4 AGENTS.md

AGENTS.md at the repository root provides instructions specifically for Claude
Code agent sessions. The installer adds the taskify usage reference here so
agents know about `/taskify` and beads commands.

---

## 8. File Delivery

All skill file contents are embedded as heredocs in the shell script. This
makes the script fully self-contained — no network access is required after
the initial download of the script itself (and the `go install` commands for
binaries).

The tradeoff is script size (~875 lines), but the benefits are:

- Works offline after initial download
- No dependency on GitHub raw URL availability
- No version drift between script and files
- Single file to distribute

---

## 9. Upgrade Path

To upgrade taskify in a project that already has it installed:

```bash
# Re-run with --force to overwrite skill files
bash install-taskify.sh --force --target /path/to/project

# Upgrade taskval binary
go install github.com/nixlim/task_templating/cmd/taskval@latest
```

The `--force` flag overwrites `.claude/skills/taskify/` and `.claude/agents/taskify-agent.md`
but preserves AGENTS.md and CLAUDE.md additions (marker-based, not overwritten).

---

## 10. Error Handling

| Condition                      | Behaviour                                                    |
|--------------------------------|--------------------------------------------------------------|
| Go not installed               | `die` with message: "Go is required. Install from https://go.dev/dl/" |
| Not a git repository           | `warn` (beads needs git) but continue                        |
| `go install taskval` fails     | `die` with the Go error output                               |
| `go install bd` fails          | `die` with the Go error output (unless --skip-beads)         |
| `bd init` fails                | `warn` and continue (bd might need manual setup)             |
| Target directory doesn't exist | `die` with message                                           |
| File write fails               | `die` with message                                           |
| --dry-run                      | All actions printed but not executed; exit 0                  |

---

## 11. Testing the Installer

### Manual smoke test

```bash
# Create a fresh test project
mkdir /tmp/test-project && cd /tmp/test-project
git init

# Run the installer
bash /path/to/task_templating/scripts/install-taskify.sh

# Verify
ls .claude/skills/taskify/SKILL.md          # Skill exists
ls .claude/agents/taskify-agent.md          # Agent exists
which taskval                               # Binary on PATH
which bd                                    # Binary on PATH
ls .beads/                                  # Beads initialised
grep "taskify-begin" AGENTS.md              # Section added
grep "taskify-begin" CLAUDE.md              # Section added

# Test idempotency
bash /path/to/task_templating/scripts/install-taskify.sh
# Should report "already exists" for everything

# Test --force
bash /path/to/task_templating/scripts/install-taskify.sh --force
# Should overwrite skill files, skip AGENTS.md/CLAUDE.md

# Test --dry-run
bash /path/to/task_templating/scripts/install-taskify.sh --dry-run
# Should print actions without executing

# Clean up
rm -rf /tmp/test-project
```

### Verify the skill works in Claude Code

After installation, open Claude Code in the target project and run:

```
/taskify "Add user authentication with JWT tokens and password hashing"
```

The skill should:
1. Analyse the description
2. Create `.tasks/user-auth.task.json`
3. Validate with `taskval --mode=graph`
4. Create beads issues
5. Report results
