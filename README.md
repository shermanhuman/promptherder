# promptherder

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

An agent configuration manager that fans out AI coding instructions from a single source of truth to the file locations that different AI coding tools actually read.

## Why

You maintain your rules once in `.promptherder/agent/`. promptherder distributes them to every AI coding agent you use:

```
.promptherder/agent/ (you edit here)
        │
        │  promptherder
        │
        ├──→ .agent/     (Antigravity)
        ├──→ .github/    (VS Code Copilot)
        └──→ ???/        (easy to add — see CONTRIBUTING.md)
```

### Compound V

promptherder ships with **Compound V**, a methodology for AI-assisted development embedded directly in the binary. It provides:

- **Skills**: planning, review, TDD, debugging, brainstorming, parallel execution
- **Workflows**: `/brainstorm`, `/write-plan`, `/execute`, `/review`
- **Rules**: always-on methodology awareness + manual browser testing

Run `promptherder compound-v` to install it into `.agent/`.

## Targets

| Target          | Source                 | Output                                                                           | Command                    |
| --------------- | ---------------------- | -------------------------------------------------------------------------------- | -------------------------- |
| **Copilot**     | `.promptherder/agent/` | `.github/copilot-instructions.md` + `.github/instructions/` + `.github/prompts/` | `promptherder copilot`     |
| **Antigravity** | `.promptherder/agent/` | `.agent/` (rules, skills, workflows)                                             | `promptherder antigravity` |
| **Compound V**  | Embedded in binary     | `.promptherder/agent/` (source of truth)                                         | `promptherder compound-v`  |

## Source format

Rules live in `.promptherder/agent/rules/*.md`. Files are sorted alphabetically and processed in order.

### Repo-wide rule (no frontmatter)

```markdown
# Hard Rules

- Do not generate plaintext secrets.
- All changes go through Git.
```

Goes into `.github/copilot-instructions.md` (Copilot) and `.agent/rules/` (Antigravity).

### Path-scoped rule (with `applyTo` frontmatter)

```markdown
---
applyTo: "**/*.sh"
---

# Shell Script Safety

- Use `set -Eeuo pipefail`.
- Never echo secrets.
```

Gets its own `.github/instructions/<name>.instructions.md` file for Copilot.

## Manifest

promptherder writes `.promptherder/manifest.json` to track which files it owns per target. This enables idempotent cleanup — when a source rule is renamed or deleted, the corresponding output file is automatically removed on the next run.

**Commit `.promptherder/manifest.json` to your repo** so cleanup state persists across machines.

### Generated file protection

Files listed in `manifest.generated` (e.g., `stack.md`, `structure.md`) are never overwritten by promptherder. These are files the AI agent itself creates during `/write-plan`.

## Usage

```bash
# Install everything (all targets)
promptherder

# Install a specific target
promptherder copilot
promptherder antigravity
promptherder compound-v

# Dry run (show what would be written)
promptherder --dry-run

# Verbose logging
promptherder -v

# Include only specific source files (Copilot)
promptherder copilot --include "**/*.md"
```

All commands run from the repo root (current working directory).

## Build

```bash
go build -o promptherder ./cmd/promptherder

# With version info
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildDate=$BUILD_DATE" \
  -o promptherder ./cmd/promptherder
```

## Install

```bash
go install github.com/shermanhuman/promptherder/cmd/promptherder@latest
```

## Compound V vs. Superpowers

Compound V is a fork of [obra/superpowers](https://github.com/obra/superpowers), adapted for multi-agent distribution via promptherder. The core methodology (brainstorm → plan → execute → review) is the same, but the implementation diverges in several ways:

| Aspect                  | Superpowers                                                                                                                      | Compound V                                                                                                                                                                                     |
| ----------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Target agent**        | Claude Code (plugin system)                                                                                                      | Any agent — Antigravity, Copilot, others via `Target` interface                                                                                                                                |
| **Installation**        | `/install` plugin marketplace                                                                                                    | `promptherder compound-v` → fans out to all agents                                                                                                                                             |
| **Distribution**        | Single agent, single format                                                                                                      | One source → multiple agent formats (`.agent/`, `.github/`, `.github/prompts/`)                                                                                                                |
| **Research**            | Not prompted                                                                                                                     | Every skill/workflow searches the web for latest docs and best practices before acting, scoped to pinned versions in `stack.md`                                                                |
| **Parallel execution**  | Subagent-driven (Claude spawns subagents)                                                                                        | Native parallel tool calls (`waitForPreviousTools: false`) — no subagent overhead                                                                                                              |
| **Git worktrees**       | Built-in skill for branch isolation                                                                                              | Not included — most agents don't have direct git integration                                                                                                                                   |
| **Subagent-driven dev** | Core workflow (Claude dispatches subagents per task)                                                                             | Replaced by parallel batches — agent executes directly with `// turbo-all`                                                                                                                     |
| **Python scripts**      | Used for artifact writing and automation                                                                                         | Eliminated — all artifact persistence uses native `write_to_file`                                                                                                                              |
| **Stack awareness**     | Not explicit                                                                                                                     | `/write-plan` generates `stack.md` (pinned versions) and `structure.md` (project layout) as always-on rules                                                                                    |
| **Copilot support**     | None                                                                                                                             | Workflows → `.github/prompts/*.prompt.md`, Skills → `#prompt:` commands, Rules → `copilot-instructions.md`                                                                                     |
| **Skills kept**         | brainstorming, writing-plans, executing-plans, TDD, debugging, code review, git worktrees, subagent dispatch, finishing branches | brainstorm, plan, execute, review, TDD, debug, parallel (new)                                                                                                                                  |
| **Skills dropped**      | —                                                                                                                                | `using-git-worktrees`, `finishing-a-development-branch`, `subagent-driven-development`, `receiving-code-review`, `writing-skills`, `verification-before-completion` (merged into other skills) |
| **Philosophy**          | Identical                                                                                                                        | Identical — TDD, systematic over ad-hoc, verify before declaring success                                                                                                                       |

### Why fork?

Superpowers is designed around Claude Code's plugin system and subagent architecture. Compound V adapts the proven methodology for agents that:

1. **Don't have a plugin marketplace** — skills/workflows are installed as files, not plugins
2. **Can't spawn subagents** — parallel execution uses native concurrent tool calls instead
3. **Need multi-agent support** — the same methodology works in Copilot, Antigravity, and future agents
4. **Benefit from web research** — every step searches for latest documentation instead of relying on training data

## Credits

- [obra/superpowers](https://github.com/obra/superpowers) — the original AI coding methodology that Compound V is based on
- [anthonylee991/gemini-superpowers-antigravity](https://github.com/anthonylee991/gemini-superpowers-antigravity) — the starting point for the Compound V Antigravity adaptation
