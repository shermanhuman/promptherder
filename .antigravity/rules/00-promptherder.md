# promptherder

Go CLI that fans out agent instructions from `.antigravity/rules/` (source of truth) to the file locations that Gemini and VS Code Copilot actually read.

## Targets

| Target | Output |
|---|---|
| Gemini | `GEMINI.md` — all rules concatenated |
| Copilot (repo‑wide) | `.github/copilot-instructions.md` — rules without `applyTo` |
| Copilot (path‑scoped) | `.github/instructions/<name>.instructions.md` — rules with `applyTo` frontmatter |

## Project rules

- **Go 1.25**, single external dependency: `github.com/bmatcuk/doublestar/v4`.
- Module path: `github.com/shermanhuman/promptherder`.
- Atomic file writes: temp → chmod → rename (see `internal/files/atomic.go`).
- All I/O functions accept `context.Context`.
- Errors: wrap with `fmt.Errorf("context: %w", err)`.
- Source files may have optional YAML frontmatter with `applyTo` field.

## Structure

```
cmd/promptherder/main.go   — entry point, flag parsing, ldflags version
internal/app/sync.go       — source reading, frontmatter parsing, plan builder, fan‑out
internal/files/atomic.go   — atomic writer
```

## Testing

- `go test ./... -count=1` — run all tests.
- Use `t.TempDir()` for filesystem tests.
- Test frontmatter parsing (with/without `applyTo`), plan building, and end‑to‑end sync.

## Build

```bash
go build -ldflags "-X main.Version=... -X main.Commit=... -X main.BuildDate=..." -o promptherder ./cmd/promptherder
```
