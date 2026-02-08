# Contributing to promptherder

## Adding a New Target Agent

promptherder uses a `Target` interface to support multiple AI coding agents. Adding a new target is straightforward.

### 1. Create the target file

Create `internal/app/youragent.go`:

```go
package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type YourAgentTarget struct{}

func (t YourAgentTarget) Name() string { return "youragent" }

func (t YourAgentTarget) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
	srcRoot := filepath.Join(cfg.RepoPath, ".promptherder", "agent")

	// Walk source files and copy to your agent's target directory.
	// Return repo-relative paths of all files written.
	// Must be idempotent — safe to run multiple times.

	var installed []string
	// ... walk srcRoot, copy files to target location ...
	return installed, nil
}
```

### 2. Register the target

In `cmd/promptherder/main.go`, add your target to the registry:

```go
allTargets := []app.Target{copilot, antigravity, compoundV, yourAgent}
```

And add a case to the subcommand switch:

```go
case "youragent":
    runErr = app.RunTarget(ctx, yourAgent, cfg)
```

### 3. Add the subcommand to extractSubcommand

```go
known := map[string]bool{
    "copilot":     true,
    "antigravity":  true,
    "compound-v":  true,
    "youragent":   true,
}
```

### 4. Write tests

Create `internal/app/youragent_test.go` with tests for:

- Basic file installation
- Dry-run mode
- Idempotency (run twice, same result)
- Skipping generated files

### 5. Update docs

- Add the target to the README usage section
- Add the target to this CONTRIBUTING guide

## Guidelines

- **Idempotency**: Running any command multiple times must produce the same result.
- **Manifest tracking**: All written files must be tracked in the manifest so stale files can be cleaned up.
- **Generated file protection**: Check `manifest.isGenerated()` before overwriting files the agent may have created (e.g., `stack.md`).
- **Atomic writes**: Use `writeFile()` which wraps `AtomicWriter` for safe file writes.
