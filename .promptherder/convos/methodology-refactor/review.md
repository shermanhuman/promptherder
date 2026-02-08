# Review: Herd Architecture Refactor (Post-fixes)

## Blockers

None.

## Majors

None.

## Minors

None. All addressed:

- ~~**M1**: `updateHerdSource` writes to git working tree~~ → **Fixed**: removed `updateHerdSource` entirely. Source URL lives in `.git/config` (git remote origin). `HerdMeta.Source` field removed.
- ~~**M2**: Non-herd files (README.md, LICENSE) merged into agent dir~~ → **Fixed**: `mergeHerds` now only walks into `rules/`, `skills/`, `workflows/` directories via `herdContentDirs` allowlist. Top-level files and unknown directories are skipped.

## Nits

None. All addressed:

- ~~**N1**: Missing test for `.git` directory skip~~ → **Fixed**: `TestMergeHerds_SkipsGitDir` added.
- ~~**N2**: `pull` subcommand doesn't support `--dry-run`~~ → **Fixed**: Added `DryRun` to `PullConfig`, early-return with log message. Also fixed flag ordering with `splitFlagsAndArgs` so `pull <url> -dry-run` works.
- ~~**N3**: Empty parent dirs left after cleanup~~ → **Fixed**: `cleanAgentDir` now walks up parent directories removing empty ones until reaching `agentDir`.

## Verification

| Metric                             | Result                                                        |
| ---------------------------------- | ------------------------------------------------------------- |
| `go vet ./...`                     | ✅ pass                                                       |
| `go test ./...`                    | ✅ pass (all 3 packages)                                      |
| `go build`                         | ✅ pass                                                       |
| `promptherder pull <url>`          | ✅ clones, no dirty working tree                              |
| `promptherder pull <url> -dry-run` | ✅ logs without cloning                                       |
| `promptherder` (bare)              | ✅ merges only rules/skills/workflows, no README/LICENSE leak |
| `.git` contents                    | ✅ not merged                                                 |
| Empty dir cleanup                  | ✅ cleaned                                                    |

## New tests added

| Test                                       | Coverage                                                          |
| ------------------------------------------ | ----------------------------------------------------------------- |
| `TestMergeHerds_SkipsGitDir`               | `.git/` directory and nested contents not copied                  |
| `TestMergeHerds_SkipsNonContentDirs`       | README.md, LICENSE excluded; rules/, skills/, workflows/ included |
| `TestCleanAgentDir_RemovesEmptyParentDirs` | Empty parent dirs cleaned after file removal                      |
| `TestSplitFlagsAndArgs` (7 cases)          | Flags separated from positional args regardless of order          |

## Files changed in this pass

| File                            | Change                                                                                           |
| ------------------------------- | ------------------------------------------------------------------------------------------------ |
| `internal/app/herd.go`          | Added `herdContentDirs`, content dir filter, empty dir cleanup, removed `Source` from `HerdMeta` |
| `internal/app/pull.go`          | Removed `updateHerdSource`, `json` import; added `DryRun` to `PullConfig`                        |
| `internal/app/herd_test.go`     | Added 3 new tests                                                                                |
| `internal/app/pull_test.go`     | Unchanged                                                                                        |
| `cmd/promptherder/main.go`      | Added `splitFlagsAndArgs`, `DryRun` in `PullConfig`, positional arg handling                     |
| `cmd/promptherder/main_test.go` | Added `TestSplitFlagsAndArgs` (7 cases)                                                          |
