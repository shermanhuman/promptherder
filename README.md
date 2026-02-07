# promptherder

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Fans out agent instructions from a single source of truth (`.antigravity/rules/`) to the file locations that different AI coding tools actually read.

## Why

You maintain your rules once in `.antigravity/rules/*.md`. promptherder generates the output files for each target system:

| Target System | Output |
|---|---|
| **Gemini** (Code Assist / CLI) | `GEMINI.md` — all rules concatenated |
| **VS Code Copilot** (repo-wide) | `.github/copilot-instructions.md` — rules without `applyTo` |
| **VS Code Copilot** (path-scoped) | `.github/instructions/<name>.instructions.md` — rules with `applyTo` frontmatter |

## Source format

Rules live in `.antigravity/rules/*.md`. Files are sorted alphabetically and processed in order.

### Repo-wide rule (no frontmatter)

```markdown
# Hard Rules

- Do not generate plaintext secrets.
- All changes go through Git.
```

Goes into both `GEMINI.md` and `.github/copilot-instructions.md`.

### Path-scoped rule (with `applyTo` frontmatter)

```markdown
---
applyTo: "**/*.sh"
---
# Shell Script Safety

- Use `set -Eeuo pipefail`.
- Never echo secrets.
```

Goes into `GEMINI.md` (all rules always do) **and** gets its own `.github/instructions/<name>.instructions.md` file with Copilot-format frontmatter.

## Usage

```bash
# Dry run (show what would be written)
promptherder --repo /path/to/repo --dry-run

# Sync for real
promptherder --repo /path/to/repo

# Include only specific source files
promptherder --repo /path/to/repo --include "**/*.md"
```

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
