# Implementation Plan — Address All Review Findings

## Goal

Fix all Blockers (2), Majors (5), Minors (5), and Nits (4) identified in the Superpowers review. Copyright holder is **Sherman Boyd**. After all fixes and final testing, bump to `v0.3.0` and commit. Add a GitHub Actions release workflow (from waxseal).

## Assumptions

- All changes are in the `promptherder` repo at `c:\Users\s\github\promptherder`.
- Tests must pass after each step.
- Current version: `v0.2.0`. Next minor: `v0.3.0`.
- GoReleaser uses defaults (no `.goreleaser.yml` needed for simple CLIs).
- Release workflow triggers on `v*` tags.

---

## Plan

### Step 1 — B1: Fix `.gitignore`

**Files:** `.gitignore`  
**Change:** Remove the `.agent/` ignore line and the stale comment. The `.agent/` directory is now the source of truth, not a generated artifact.  
**Verify:** `git status` shows `.gitignore` modified. Confirm `.agent/rules/00-promptherder.md` would be tracked.

---

### Step 2 — B2: Delete `.antigravity/` and rewrite `.agent/rules/00-promptherder.md`

**Files:** `.antigravity/rules/00-promptherder.md` (delete entire `.antigravity/` dir), `.agent/rules/00-promptherder.md`  
**Change:**

- Delete `.antigravity/` directory entirely.
- Rewrite `.agent/rules/00-promptherder.md` to accurately describe current architecture: reads `.agent/rules/`, generates Copilot outputs + `.promptherder/manifest.json`, no GEMINI.md.  
  **Verify:** `dir .antigravity` returns not found. `cat .agent/rules/00-promptherder.md` shows accurate description.

---

### Step 3 — M5: Handle `sources == 0` for cleanup

**Files:** `internal/app/sync.go`  
**Change:** When no source files are found, still load the old manifest, run `cleanStale` to remove previously generated files, and write an empty manifest. Only skip the plan-building and file-writing steps.  
**Verify:** `go test ./... -count=1` passes.

---

### Step 4 — M5 test: Add test for zero-sources cleanup

**Files:** `internal/app/sync_test.go`  
**Change:** Add `TestRun_NoSources_CleansUpStaleFiles`: (1) create a source, run `Run`, verify output + manifest exist. (2) Delete the source. (3) Run `Run` again. (4) Assert the output file is deleted and the manifest is empty.  
**Verify:** `go test ./... -count=1 -run TestRun_NoSources_CleansUpStaleFiles` passes.

---

### Step 5 — M4: Add end-to-end idempotency test

**Files:** `internal/app/sync_test.go`  
**Change:** Add `TestRun_IdempotentCleanup`: (1) create sources A (repo-wide) and B (with applyTo), run `Run`. (2) Delete source B. (3) Run `Run` again. (4) Assert A's output remains, B's instruction file is deleted, manifest reflects only A.  
**Verify:** `go test ./... -count=1 -run TestRun_IdempotentCleanup` passes.

---

### Step 6 — M1: Log warning on corrupt manifest

**Files:** `internal/app/sync.go`  
**Change:** `readManifest` needs access to the logger. Change signature to `readManifest(repoPath string, logger *slog.Logger)`. On JSON parse error, log at `WARN` level before returning empty manifest. Update the call site in `Run`.  
**Verify:** `go test ./... -count=1` passes.

---

### Step 7 — M2: Remove duplicate `MkdirAll` from `writeFile`

**Files:** `internal/app/sync.go`  
**Change:** Remove the `os.MkdirAll` call from `writeFile` — `AtomicWriter.Write()` already handles directory creation.  
**Verify:** `go test ./... -count=1` passes.

---

### Step 8 — M3 + N1: Fix README and code comments re: "Gemini CLI"

**Files:** `README.md`, `internal/app/sync.go`  
**Change:**

- README: Replace "Gemini CLI reads this directory natively" with generic language. Remove the Gemini CLI row from the table or replace with "AI coding agent (reads `.agent/rules/` natively)".
- `sync.go` comment on `Run`: Change "Gemini CLI reads .agent/rules/ natively" to "The AI coding agent reads .agent/rules/ natively".
- `buildPlan` header comments in generated content: Change `.agent/rules/` reference to not mention any specific tool.  
  **Verify:** `grep -r "Gemini CLI" .` returns no results (outside `artifacts/`). `go test ./... -count=1` passes.

---

### Step 9 — m1: Make `sourceDir` configurable

**Files:** `internal/app/sync.go`, `cmd/promptherder/main.go`  
**Change:**

- Add `SourceDir string` field to `Config` struct.
- In `Run`, default to `.agent/rules` if `SourceDir` is empty.
- Use `cfg.SourceDir` wherever `sourceDir` constant was used (readSources, buildPlan header, manifest).
- Keep the const as the default value.
- Add `--source` flag to `main.go`.  
  **Verify:** `go test ./... -count=1` passes. `promptherder --help` shows `--source` flag.

---

### Step 10 — m3: Add `fsync` to `AtomicWriter`

**Files:** `internal/files/atomic.go`  
**Change:** Call `tmpFile.Sync()` after `Write` and before `Close`. This ensures data is flushed to disk before the rename.  
**Verify:** `go test ./... -count=1` passes.

---

### Step 11 — m4: Add dry-run log for manifest write

**Files:** `internal/app/sync.go`  
**Change:** In the dry-run branch, after logging planned outputs, add `cfg.Logger.Info("dry-run", "target", manifestPath)` for the manifest file.  
**Verify:** `promptherder --repo c:\Users\s\github\breakdown-infra --dry-run -v` output includes the manifest line.

---

### Step 12 — m5 + README: Document `.promptherder/` and update README

**Files:** `README.md`  
**Change:**

- Add a "Manifest" section explaining `.promptherder/manifest.json`, its purpose, and that it should be committed to git.
- Add guidance on `.gitignore` (don't ignore `.promptherder/`).  
  **Verify:** `cat README.md` shows manifest section.

---

### Step 13 — N2: Rename `new` parameter in `cleanStale`

**Files:** `internal/app/sync.go`  
**Change:** Rename `old, new manifest` to `prev, cur manifest` in `cleanStale` signature and body. Update call site.  
**Verify:** `go test ./... -count=1` passes.

---

### Step 14 — N3: Update copyright holder in LICENSE

**Files:** `LICENSE`  
**Change:** Change `Copyright (c) 2026 shermanhuman` to `Copyright (c) 2026 Sherman Boyd`.  
**Verify:** `cat LICENSE` shows correct copyright.

---

### Step 15 — N4: Verify `promptherder.exe` is gitignored

**Files:** (none modified — verification only)  
**Change:** Run `git status promptherder.exe` to confirm it's not tracked. If tracked, run `git rm --cached promptherder.exe`.  
**Verify:** `git status promptherder.exe` shows nothing or shows it as untracked.

---

### Step 16 — m2 (parseFrontmatter): No code change, add comment

**Files:** `internal/app/sync.go`  
**Change:** Add a comment to `parseFrontmatter` noting the parser is intentionally minimal (no full YAML dep) and that `applyTo` is the only recognized key.  
**Verify:** Read the comment.

---

### Step 17 — Add GitHub Actions release workflow

**Files:** `.github/workflows/release.yml`  
**Change:** Copy the release workflow from waxseal (`c:\Users\s\github\waxseal\.github\workflows\release.yml`), updating the Go version from `1.24` to `1.25` to match promptherder's `go.mod`.  
**Verify:** `cat .github/workflows/release.yml` shows correct Go version.

---

### Step 18 — Final verification

**Files:** (none)  
**Change:** Run full test suite, rebuild binary, dry-run against breakdown-infra.  
**Verify:**

```bash
go test ./... -count=1 -v
go build -o promptherder.exe ./cmd/promptherder
promptherder.exe --repo c:\Users\s\github\breakdown-infra --dry-run -v
```

---

### Step 19 — Commit and tag v0.3.0

**Files:** (all modified files)  
**Change:**

- `git add -A`
- `git commit -m "v0.3.0: manifest-based idempotency, configurable source dir, review fixes"`
- `git tag -a v0.3.0 -m "v0.3.0: manifest-based idempotency, configurable source dir, review fixes"`  
  **Verify:** `git log -1 --oneline` shows the commit. `git tag -l v0.3.0` shows the tag.

---

## Risks & mitigations

| Risk                                                       | Mitigation                                                 |
| ---------------------------------------------------------- | ---------------------------------------------------------- |
| Making `sourceDir` configurable could break header strings | Use the configured value in all generated headers          |
| Changing `readManifest` signature affects callers          | Only one call site in `Run`; update it in same step        |
| Deleting `.antigravity/` is irreversible                   | Content is stale; accurate version is in `.agent/rules/`   |
| GoReleaser may need config for ldflags                     | GoReleaser auto-detects `main.Version` etc. from Go source |
| Release workflow Go version mismatch                       | Explicitly set to `1.25` matching `go.mod`                 |

## Rollback plan

All changes are git-tracked.

- `git checkout .` reverts all uncommitted changes.
- `git tag -d v0.3.0` removes the tag if not yet pushed.
- Each step is independently testable, so partial rollback is possible.
