# promptherder

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Fans out agent instructions from a single source of truth to the file locations that different AI coding tools actually read.

## Why

You maintain your rules once in a source directory (default: `.agent/rules/*.md`). Your AI coding agent reads this directory natively — promptherder generates the output files for VS Code Copilot:

| Target System                     | Output                                                                           |
| --------------------------------- | -------------------------------------------------------------------------------- |
| **AI coding agent**               | _(reads `.agent/rules/` natively — no generation needed)_                        |
| **VS Code Copilot** (repo-wide)   | `.github/copilot-instructions.md` — rules without `applyTo`                      |
| **VS Code Copilot** (path-scoped) | `.github/instructions/<name>.instructions.md` — rules with `applyTo` frontmatter |

## Source format

Rules live in your source directory (default: `.agent/rules/*.md`). Files are sorted alphabetically and processed in order.

### Repo-wide rule (no frontmatter)

```markdown
# Hard Rules

- Do not generate plaintext secrets.
- All changes go through Git.
```

Goes into `.github/copilot-instructions.md`.

### Path-scoped rule (with `applyTo` frontmatter)

```markdown
---
applyTo: "**/*.sh"
---

# Shell Script Safety

- Use `set -Eeuo pipefail`.
- Never echo secrets.
```

Gets its own `.github/instructions/<name>.instructions.md` file with Copilot-format frontmatter.

## Manifest

promptherder writes `.promptherder/manifest.json` to track which files it owns in the target repo. This enables idempotent cleanup — when a source rule is renamed or deleted, the corresponding output file is automatically removed on the next run.

**Commit `.promptherder/manifest.json` to your repo** so cleanup state persists across machines.

## Usage

```bash
# Dry run (show what would be written)
promptherder --repo /path/to/repo --dry-run

# Sync for real
promptherder --repo /path/to/repo

# Include only specific source files
promptherder --repo /path/to/repo --include "**/*.md"

# Use a custom source directory
promptherder --repo /path/to/repo --source ".gemini/rules"
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
