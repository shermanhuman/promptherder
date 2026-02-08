# Review: Naming & Namespace Cleanup

## Blockers

None.

## Majors

None.

## Minors

1. **`CONTRIBUTING.md` line 200 referenced `sync_test.go`** — missed in initial rename pass. Fixed.
2. **`structure.md` line 15 still said `sync.go (TO BE RENAMED)`** — stale marker left behind. Fixed.
3. **`structure.md` line 40 still described abandoned `<name>_SKILL.md` convention** — leftover from the plan iteration. Fixed.
4. **`plan.md` assumptions still referenced SKILL.md renaming** — stale after decision to keep filenames. Fixed.

## Nits

1. **`promptherder.exe` was already gitignored and untracked** — Step 1 (`git rm --cached`) was a no-op. No harm, but the plan assumed it was tracked.

## Verification

| Command                                | Result                            |
| -------------------------------------- | --------------------------------- |
| `go vet ./...`                         | Pass                              |
| `go test ./...`                        | Pass (all packages)               |
| `Test-Path artifacts`                  | `False` (root artifacts/ removed) |
| `Get-ChildItem coverage*` (root)       | No output (moved)                 |
| `.promptherder/artifacts/coverage/`    | 5 files present                   |
| `.promptherder/artifacts/compound-v/`  | 1 file present                    |
| `.promptherder/artifacts/superpowers/` | 2 files present                   |

## Summary of Changes

| Change                                                         | Files                                                     |
| -------------------------------------------------------------- | --------------------------------------------------------- |
| Added `coverage*` to `.gitignore`                              | `.gitignore`                                              |
| Moved coverage files to `.promptherder/artifacts/coverage/`    | 5 files                                                   |
| Consolidated root `artifacts/` into `.promptherder/artifacts/` | 3 files, removed `artifacts/` dir                         |
| Renamed `sync.go` → `copilot.go`                               | `internal/app/copilot.go`, `internal/app/copilot_test.go` |
| Updated all `sync.go` references                               | `CONTRIBUTING.md` (5 locations)                           |
| Created skills README                                          | `compound-v/skills/README.md`                             |
| Fixed stale structure rules                                    | `.agent/rules/structure.md` (2 locations)                 |
| Fixed stale plan assumptions                                   | `.promptherder/artifacts/plan.md`                         |

## Follow-ups

- Consider running `promptherder` end-to-end to verify the full install pipeline with the renamed file.
- The `00-compound-v.md` numeric prefix (Minor from original review) was not addressed — low priority.
- Coverage files in `.promptherder/artifacts/coverage/` should be added to `.gitignore` or deleted if not needed for reference.
