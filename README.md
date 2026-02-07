# promptherder

Syncs Antigravity agent rules/skills from a source-of-truth directory (`.antigravity/`) to the locations where tools actually read them (`.agent/`).

## Why

Google Antigravity reads rules from `.agent/rules/` and skills from `.agent/skills/` in your workspace root. But you may want to keep your canonical agent instructions in a separate `.antigravity/` directory so they're clearly your source of truth and don't get accidentally edited in the tool-read locations.

promptherder copies files from source → target, preserving directory structure.

## Mappings

### `--source=antigravity` (default)

| Source | Target |
|---|---|
| `.antigravity/rules/**` | `.agent/rules/` |
| `.antigravity/skills/**` | `.agent/skills/` |

### `--source=agent`

| Source | Target |
|---|---|
| `.agent/rules/**` | `.antigravity/rules/` |
| `.agent/skills/**` | `.antigravity/skills/` |

## Usage

```bash
# Dry run (show what would be synced)
promptherder --repo /path/to/repo --dry-run

# Sync for real
promptherder --repo /path/to/repo

# Include only specific files
promptherder --repo /path/to/repo --include "**/*.md"

# Reverse direction (agent → antigravity)
promptherder --repo /path/to/repo --source agent
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
go install github.com/breakdown-analytics/promptherder/cmd/promptherder@latest
```
