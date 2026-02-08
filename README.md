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

| Target          | Source                       | Output                                                      | Command                    |
| --------------- | ---------------------------- | ----------------------------------------------------------- | -------------------------- |
| **Copilot**     | `.promptherder/agent/rules/` | `.github/copilot-instructions.md` + `.github/instructions/` | `promptherder copilot`     |
| **Antigravity** | `.promptherder/agent/`       | `.agent/` (rules, skills, workflows)                        | `promptherder antigravity` |
| **Compound V**  | Embedded in binary           | `.agent/` (skills, workflows, rules)                        | `promptherder compound-v`  |

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

## Credits

- [obra/superpowers](https://github.com/obra/superpowers) — the original AI coding methodology that Compound V is based on
- [anthonylee991/gemini-superpowers-antigravity](https://github.com/anthonylee991/gemini-superpowers-antigravity) — the starting point for the Compound V Antigravity adaptation
