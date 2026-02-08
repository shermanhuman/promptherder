# Execution Log: Herd Architecture Refactor

> `.promptherder/convos/methodology-refactor/execution.md`

## Batch 1: Compound-v repo + herd.go + pull.go (Steps 1-3)

- Created `c:\Users\s\github\compound-v\` with content from `promptherder/compound-v/`
- Added `herd.json` with `{"name":"compound-v"}`
- Initialized git, committed
- Created `internal/app/herd.go`: `HerdMeta`, `discoverHerds()`, `mergeHerds()`, `cleanAgentDir()`
- Created `internal/app/pull.go`: `Pull()`, `herdNameFromURL()`, `updateHerdSource()`
- ✅ `go vet ./...` passes
- ✅ `git init && commit` succeeds

## Batch 2: Tests + runner.go (Steps 4-5)

- Created `internal/app/herd_test.go`: 10 tests (discovery, merge, conflict, dry-run, context, cleanup)
- Created `internal/app/pull_test.go`: `TestHerdNameFromURL` (8 cases including Windows paths)
- Fixed `herdNameFromURL` to handle `\` separator (Windows)
- Updated `internal/app/runner.go`: `RunAll` now merges herds before running targets
- ✅ `go test ./...` all pass

## Batch 3: Remove embedded + update main (Steps 6-8) + compound-v repo

- Deleted: `embed.go`, `compoundv.go`, `compoundv_test.go`, `compound-v/` dir
- Updated `cmd/promptherder/main.go`:
  - Removed `promptherder` import (embedded FS)
  - Removed `CompoundVTarget` from targets
  - Added `case "pull"` subcommand
  - Updated usage string
- Updated `cmd/promptherder/main_test.go`: replaced `compound-v` with `pull` test case
- Added `README.md` and `LICENSE` (MIT, Sherman Boyd) to compound-v repo
- Created GitHub repo: `gh repo create shermanhuman/compound-v --public`
- ✅ `go vet`, `go test ./...`, `go build` all pass

## Batch 4: Docs + integration (Steps 9-11)

- Updated `CONTRIBUTING.md`: removed CompoundVTarget, added Herds section
- Updated `.agent/rules/structure.md`: replaced embedded layout with herds
- Fixed `.git` directory leak in `mergeHerds` (added `filepath.SkipDir`)
- Integration test:
  - `promptherder pull https://github.com/shermanhuman/compound-v` ✅ clones to `.promptherder/herds/compound-v/`
  - `promptherder` (bare) ✅ merges herd → `.promptherder/agent/` → `.agent/` + `.github/`
  - No `.git` files leaked
  - Skills variant selection still works (ANTIGRAVITY.md → SKILL.md)
