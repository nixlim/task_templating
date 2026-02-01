---
name: init
description: >
  Install the taskify skill into a target project directory. Creates .claude/skills/taskify/,
  installs taskval and bd, initialises beads, and updates AGENTS.md and CLAUDE.md.
argument-hint: <target-project-directory>
disable-model-invocation: true
---

# Install Taskify into a Target Project

Run the install script with the provided arguments:

```bash
bash scripts/install-taskify.sh $ARGUMENTS
```

This will:
1. Verify Go is installed and install `taskval` and `bd` if missing
2. Create `.claude/skills/taskify/` with skill files in the target project
3. Create `.claude/agents/taskify-agent.md` in the target project
4. Update `AGENTS.md` and `CLAUDE.md` with taskify usage instructions
5. Initialise beads (`bd init`) if not already done

Pass flags after the target directory:
- `--force` to overwrite existing skill files
- `--skip-beads` to skip bd installation and initialisation
- `--dry-run` to preview changes without executing them
