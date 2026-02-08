# Project Structure

## Folder Layout

```
promptherder/
├── cmd/
│   └── promptherder/          # Main CLI entry point
│       └── main.go
├── internal/
│   ├── app/                   # Core application logic
│   │   ├── target.go          # Target interface
│   │   ├── antigravity.go     # Antigravity target impl
│   │   ├── compoundv.go       # Compound V (embedded) target impl
│   │   ├── copilot.go         # Copilot target impl
│   │   ├── runner.go          # Orchestration & helpers
│   │   └── manifest.go        # Manifest tracking
│   └── files/                 # File utilities
│       ├── atomic.go          # AtomicWriter
├── compound-v/                # Embedded Compound V methodology
│   ├── rules/
│   ├── skills/
│   └── workflows/
├── .promptherder/             # Source of truth for agent config
│   ├── agent/                 # Staging area for agent files
│   ├── convos/                # Per-task artifacts (brainstorms, plans, reviews)
│   │   └── <slug>/            # Each task gets its own folder
│   ├── artifacts/             # Legacy artifacts (historical)
│   └── manifest.json          # Tracks installed/generated files
├── .agent/                    # Installed Antigravity agent files
└── .github/                   # Installed Copilot agent files
```

## Where Things Belong

- **New Targets**: `internal/app/<targetname>.go` implementing `Target`.
- **Shared Logic**: `internal/app/runner.go` or specific helpers.
- **Embedded Assets**: `compound-v/` (must be registered in `embed.go`).

## Naming Conventions

- **Skill Files**: `SKILL.md` inside a subdirectory per skill (platform requirement).
  - Target-specific variants: `ANTIGRAVITY.md`, `COPILOT.md` in the same skill dir.
  - Variants replace `SKILL.md` when installing to their target.
- **Test Files**: `*_test.go` next to source.
- **Go Files**: `snake_case` (e.g., `runner_helper_test.go`).
- **Config Files**: `kebab-case` (e.g., `compound-v-plan.md`).

## DRY Location Hints

- **Frontend/Agent**: Code generation logic lives in `internal/app/`.
- **System**: File I/O helpers in `internal/files/`.
