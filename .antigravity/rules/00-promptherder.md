# promptherder

Go CLI that syncs Antigravity agent rules/skills between `.antigravity/` (source of truth) and `.agent/` (tool‑read location).

## Project rules

- **Go 1.25**, single external dependency: `github.com/bmatcuk/doublestar/v4`.
- Module path: `github.com/shermanhuman/promptherder`.
- Atomic file writes: temp → chmod → rename (see `internal/files/atomic.go`).
- All I/O functions accept `context.Context`.
- Errors: wrap with `fmt.Errorf("context: %w", err)`.
- No Copilot concepts in this codebase — Antigravity‑only mappings.

## Structure

```
cmd/promptherder/main.go   — entry point, flag parsing, ldflags version
internal/app/sync.go       — mappings, plan builder, file sync
internal/files/atomic.go   — atomic writer
```

## Testing

- `go test ./... -count=1` — run all tests.
- Use `t.TempDir()` for filesystem tests.
- Test both mapping directions (`antigravity` and `agent` sources).

## Build

```bash
go build -ldflags "-X main.Version=... -X main.Commit=... -X main.BuildDate=..." -o promptherder ./cmd/promptherder
```
