# Plan: Herd Architecture Refactor

> `/plan` · **Status**: approved · `.promptherder/convos/methodology-refactor/plan.md`

## Goal

Replace the embedded Compound V methodology with a generic **herd** system. A herd is a versionable collection of rules, skills, and workflows pulled from any git URL. Promptherder becomes the herder — it pulls herds, merges them, and fans out to agent targets.

## Happy-path user story

```
# User installs promptherder (binary, no embedded prompts)
go install github.com/shermanhuman/promptherder/cmd/promptherder@latest

# User pulls compound-v herd from git
promptherder pull https://github.com/shermanhuman/compound-v

# This creates:
#   .promptherder/herds/compound-v/
#     herd.json           ← {"name":"compound-v","source":"https://..."}
#     rules/
#     skills/
#     workflows/

# User runs promptherder to fan out to all targets
promptherder
#   Step 1: Merge all herds → .promptherder/agent/
#   Step 2: antigravity: .promptherder/agent/ → .agent/
#   Step 3: copilot: .promptherder/agent/ → .github/

# User can also pull additional herds
promptherder pull https://github.com/someone/elixir-rules
#   → .promptherder/herds/elixir-rules/
# Next `promptherder` merges both herds (errors on file conflicts)
```

## Plan

### Phase 1: Create compound-v as standalone repo

**Step 1: Initialize compound-v repo**

- Files: `c:\Users\s\github\compound-v\`
- Change: Copy `compound-v/rules/`, `compound-v/skills/`, `compound-v/workflows/` from promptherder. Add `herd.json` with `{"name": "compound-v"}`. Init git, commit.
- Verify: `ls c:\Users\s\github\compound-v` shows `herd.json`, `rules/`, `skills/`, `workflows/`

### Phase 2: Add herd management to promptherder

**Step 2: Create `herd.go` — herd data types and discovery**

- Files: `internal/app/herd.go` (new)
- Change: Define `HerdMeta` struct (parsed from `herd.json`), `discoverHerds()` function that scans `.promptherder/herds/*/herd.json`, and `mergeHerds()` that copies all herds into `.promptherder/agent/` (erroring on conflict).
- Verify: `go vet ./...`

**Step 3: Create `pull.go` — git clone/update logic**

- Files: `internal/app/pull.go` (new)
- Change: `Pull(ctx, url, repoPath)` function. Extracts herd name from URL (last path segment sans `.git`). If `.promptherder/herds/<name>/` doesn't exist, `git clone` into it. If it does exist, `git pull` to update. Validates `herd.json` exists after clone.
- Verify: `go vet ./...`

**Step 4: Create `herd_test.go` and `pull_test.go`**

- Files: `internal/app/herd_test.go` (new), `internal/app/pull_test.go` (new)
- Change: Tests for discovery, merge, conflict detection, pull (using local git repos in tempdir).
- Verify: `go test ./...`

**Step 5: Update runner.go — merge herds before targets**

- Files: `internal/app/runner.go`
- Change: `RunAll` gains a herd-merge step before target loop. Discover herds → merge to `.promptherder/agent/` → run targets. If no herds found, log warning and continue (targets read whatever is already in `.promptherder/agent/`).
- Verify: `go test ./...`

### Phase 3: Remove embedded Compound V

**Step 6: Delete embedded FS and CompoundVTarget**

- Files: Delete `embed.go`, `internal/app/compoundv.go`, `internal/app/compoundv_test.go`. Delete `compound-v/` directory from promptherder repo.
- Change: Remove all compound-v-specific code. The `compound-v/` content now lives in its own repo.
- Verify: `go vet ./...`, `go test ./...`

**Step 7: Update main.go — remove compound-v target, add pull subcommand**

- Files: `cmd/promptherder/main.go`
- Change:
  - Remove `compoundV` from `allTargets` (only `[copilot, antigravity]`)
  - Remove `case "compound-v"` from subcommand switch
  - Add `case "pull"` that calls `app.Pull(ctx, url, repoPath)`
  - Update usage string
- Verify: `go build ./cmd/promptherder`

**Step 8: Update main_test.go**

- Files: `cmd/promptherder/main_test.go`
- Change: Remove `compound-v` test case from `TestExtractSubcommand`. Add `pull` test case.
- Verify: `go test ./cmd/...`

### Phase 4: Update documentation

**Step 9: Update CONTRIBUTING.md**

- Files: `CONTRIBUTING.md`
- Change: Remove CompoundVTarget from targets table. Add "Herds" section documenting the architecture. Update directory structure diagram.
- Verify: Read file

**Step 10: Update structure.md**

- Files: `.agent/rules/structure.md` (or `.promptherder/agent/rules/` — wherever it lives post-refactor)
- Change: Document `.promptherder/herds/` directory and `herd.json` format.
- Verify: Read file

### Phase 5: Verify end-to-end

**Step 11: Full integration test**

- Files: none (read-only verification)
- Verify:
  - `go vet ./...` passes
  - `go test ./...` passes
  - `go build ./cmd/promptherder` succeeds
  - Manual: `promptherder pull <compound-v-path>` creates `.promptherder/herds/compound-v/`
  - Manual: `promptherder` merges herd → `.promptherder/agent/` → `.agent/` + `.github/`

## Risks & mitigations

| Risk                                                             | Mitigation                                                                                                                                                          |
| ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Users with existing embedded compound-v setup get stale files    | First run of new version: stale cleanup deletes old `compound-v` target files from manifest. Log a clear message: "Run `promptherder pull <url>` to install herds." |
| `git` not on PATH                                                | `Pull` checks for `git` binary first, returns actionable error: "git is required for `promptherder pull`"                                                           |
| Herd repo has no `herd.json`                                     | `Pull` validates `herd.json` after clone, returns error with fix instructions                                                                                       |
| Two herds conflict on same file path                             | `mergeHerds` returns error listing both herds and the conflicting path                                                                                              |
| Network unavailable during pull                                  | `Pull` returns git's error directly. Existing herds in `.promptherder/herds/` still work offline.                                                                   |
| Breaking change for CI scripts calling `promptherder compound-v` | This subcommand is removed. Mitigation: release as major version bump (v1.0.0) or document migration in release notes.                                              |

## Rollback plan

```
git revert <commit>    # promptherder repo
go install github.com/shermanhuman/promptherder/cmd/promptherder@v0.7.0  # revert to embedded version
```

## Deferred

- `promptherder list herds` — show installed herds
- `promptherder remove herd <name>` — uninstall a herd
- Herd version pinning / lockfile
- `herd.json` with richer metadata (version, description, compatible targets, dependencies)
- Multi-herd conflict resolution strategies beyond "error" (namespacing, priority)
