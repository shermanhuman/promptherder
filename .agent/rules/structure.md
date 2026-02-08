# Project Structure

## Folder Layout

```
promptherder/
├── cmd/
│   └── promptherder/          # Main entry point (CLI)
│       └── main.go
├── internal/
│   ├── app/                   # Core application logic
│   │   ├── target.go          # Target interface
│   │   ├── antigravity.go     # Antigravity target impl
│   │   ├── compoundv.go       # Compound V target impl
│   │   ├── sync.go            # Copilot target impl (legacy: RunCopilot)
│   │   ├── runner.go          # RunAll/RunTarget orchestration
│   │   ├── manifest.go        # Manifest tracking
│   │   └── *_test.go          # Tests alongside code
│   └── files/                 # File utilities
│       ├── atomic.go          # AtomicWriter
│       └── atomic_test.go
├── compound-v/                # Embedded Compound V methodology
│   ├── rules/
│   ├── skills/
│   └── workflows/
├── .promptherder/             # Source of truth for agent config
│   ├── agent/                 # Synced from compound-v or user-created
│   │   ├── rules/
│   │   ├── skills/
│   │   └── workflows/
│   ├── artifacts/             # Build artifacts, plans, reviews
│   └── manifest.json          # Tracks generated files
├── .agent/                    # Antigravity target output
└── .github/                   # Copilot target output
    ├── copilot-instructions.md
    ├── instructions/
    └── prompts/
```

## Where Things Belong

### New Target Implementations

- **Location**: `internal/app/<targetname>.go`
- **Must implement**: `Target` interface (Name() + Install())
- **Register in**: `cmd/promptherder/main.go` (allTargets slice)

### Shared Helpers

- **Runner helpers**: `internal/app/runner.go` (setupRunner, persistAndClean)
- **File helpers**: `internal/files/` package
- **Manifest helpers**: `internal/app/manifest.go`

### Tests

- **Convention**: `*_test.go` in same package as source
- **Test fixtures**: Define helpers in test files (e.g., `mustWrite`, `setupTestRepo`)

### Generated Files

- **Plans**: `.promptherder/artifacts/plan.md`
- **Reviews**: `.promptherder/artifacts/review.md`
- **User artifacts**: `.promptherder/artifacts/*` (tracked in manifest.json)

## Naming Conventions

- **Target structs**: `<AgentName>Target` (e.g., `AntigravityTarget`, `CopilotTarget`)
- **Config structs**: `Config`, `TargetConfig` (reused across targets)
- **Helper functions**: Lowercase, descriptive (e.g., `readManifest`, `writeFile`)
- **Constants**: Lowercase with package prefix (e.g., `antigravitySource`, `manifestDir`)

## DRY Location Hints

When extracting duplicated code, place it in:

- **Runner boilerplate** → `internal/app/runner.go` (shared setup/teardown)
- **File walking patterns** → New file: `internal/app/install.go` (generic file installer)
- **Logger creation** → `internal/app/logger.go` or inline in existing helpers
- **Test utilities** → `internal/app/testing.go` (test-only build tag)
- **Path builders** → Methods on `TargetConfig` (e.g., `AgentRulesDir()`)

## Import Organization

1. Standard library imports (alphabetical)
2. Blank line
3. External dependencies (alphabetical)
4. Blank line
5. Internal packages (alphabetical)

Example:

```go
import (
    "context"
    "fmt"
    "os"

    "github.com/bmatcuk/doublestar/v4"

    "github.com/shermanhuman/promptherder/internal/files"
)
```
