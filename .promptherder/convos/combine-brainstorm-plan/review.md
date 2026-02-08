# Review: Merge Brainstorm + Write-Plan → /plan

## Blockers

None.

## Majors

None.

## Minors

1. **cleanStale leaves empty directories** — When promptherder removes stale files, it doesn't prune the now-empty parent directories (e.g. `.agent/skills/compound-v-brainstorm/` was left as an empty folder). Manually cleaned up. This is a pre-existing promptherder bug, not introduced by this change.

2. **Go build cache requires double-run** — First `go run` after editing `compound-v/` source files uses a cached binary with stale embedded FS. Second run picks up changes. Known issue from previous execution.

## Nits

1. **User memory file** — The user's `.gemini` memory file (`compound-v.md`) still references the old 4-step pipeline. It will auto-update on next conversation start, since the installed `.agent/rules/compound-v.md` is the source of truth.

## Verification

| Command                                         | Result    |
| ----------------------------------------------- | --------- |
| `go vet ./...`                                  | Pass      |
| `go test ./...`                                 | Pass      |
| `Test-Path` checks for existence/absence        | All pass  |
| `Select-String` for ACCEPT in plan.md           | 4 matches |
| `Select-String` for brainstorm in compound-v.md | 0 matches |

## Summary of Changes

| Action    | File                                                                 |
| --------- | -------------------------------------------------------------------- |
| Created   | `compound-v/workflows/plan.md` — interactive ideation loop workflow  |
| Deleted   | `compound-v/workflows/brainstorm.md`                                 |
| Deleted   | `compound-v/workflows/write-plan.md`                                 |
| Deleted   | `compound-v/skills/compound-v-brainstorm/SKILL.md`                   |
| Updated   | `compound-v/skills/compound-v-plan/SKILL.md` — ideation loop pattern |
| Updated   | `compound-v/rules/compound-v.md` — 3-step pipeline                   |
| Cleaned   | Empty directories from stale removal                                 |
| Installed | All 3 targets via `go run ./cmd/promptherder`                        |

## Follow-ups

- Fix cleanStale to prune empty parent directories after removing files.
- Consider adding `go run -a` or cache-busting to the execute workflow to avoid the double-run issue.
- Smoke test: run `/plan` in a new conversation to verify the ideation loop works end-to-end.
