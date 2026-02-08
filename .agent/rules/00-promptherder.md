# promptherder

Go CLI that fans out agent instructions from `.agent/rules/` (source of truth) to the file locations that different AI coding tools read (e.g. VS Code Copilot).

## Targets

| Target                | Output                                                                           |
| --------------------- | -------------------------------------------------------------------------------- |
| AI coding agent       | _(reads `.agent/rules/` natively — no generation needed)_                        |
| Copilot (repo‑wide)   | `.github/copilot-instructions.md` — rules without `applyTo`                      |
| Copilot (path‑scoped) | `.github/instructions/<name>.instructions.md` — rules with `applyTo` frontmatter |

## Manifest

promptherder writes `.promptherder/manifest.json` to track which files it owns. This enables idempotent cleanup — stale outputs are removed when source rules are renamed or deleted.

## Project rules

- **Go 1.25**, single external dependency: `github.com/bmatcuk/doublestar/v4`.
- Module path: `github.com/shermanhuman/promptherder`.
- Atomic file writes: temp → fsync → chmod → rename (see `internal/files/atomic.go`).
- All I/O functions accept `context.Context`.
- Errors: wrap with `fmt.Errorf("context: %w", err)`.
- Source files may have optional YAML frontmatter with `applyTo` field.

## Structure

```
cmd/promptherder/main.go    — entry point, flag parsing, ldflags version
internal/app/sync.go        — source reading, frontmatter parsing, plan builder, manifest, fan‑out
internal/files/atomic.go    — atomic writer
.promptherder/manifest.json — tracks generated files for idempotent cleanup
```

## Testing

- `go test ./... -count=1` — run all tests.
- Use `t.TempDir()` for filesystem tests.
- Test frontmatter parsing (with/without `applyTo`), plan building, manifest round-trip, and end‑to‑end sync.

## Build

```bash
go build -ldflags "-X main.Version=... -X main.Commit=... -X main.BuildDate=..." -o promptherder ./cmd/promptherder
```
