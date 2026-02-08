# Execution: Replace git pull with HTTP tarball download

## Batch 1 — pull.go rewrite + .gitignore cleanup (parallel)

### Step 1: Rewrite pull.go

- Files: `internal/app/pull.go`
- Replaced `git clone`/`git pull` with HTTP GET of GitHub archive tarball
- Added `ownerRepoFromURL` to parse GitHub URLs
- Added `toArchiveURL` to build archive API URL
- Added `extractTarGz` with prefix stripping and path traversal protection
- Removed `os/exec` dependency — now pure stdlib (`net/http`, `archive/tar`, `compress/gzip`)
- Verify: `go vet ./...` ✅

### Step 3: Remove herds from .gitignore

- Files: `.gitignore`
- Removed `.promptherder/herds/` entry — no .git dirs anymore
- Verify: `git check-ignore .promptherder/herds/compound-v` → NOT IGNORED ✅

## Batch 2 — Tests

### Step 2: Update pull_test.go

- Files: `internal/app/pull_test.go`
- Added `TestOwnerRepoFromURL` (7 cases: https, ssh, trailing slash, non-github, edge cases)
- Added `TestToArchiveURL`
- Added `TestExtractTarGz` (in-memory tar.gz fixture with prefix stripping)
- Added `TestExtractTarGz_PathTraversal` (security: rejects `../../` paths)
- Kept existing `TestHerdNameFromURL`
- Verify: `go test ./internal/app/ -run "HerdName|OwnerRepo|ArchiveURL|ExtractTar" -v` → all PASS ✅

## Batch 3 — Integration

### Step 4: End-to-end test

- Removed old git-cloned herd, rebuilt binary
- `promptherder pull https://github.com/shermanhuman/compound-v` → downloaded via HTTP ✅
- `.promptherder/herds/compound-v/.git` → does NOT exist ✅
- `.promptherder/herds/compound-v/herd.json` → exists ✅
- `promptherder` (bare) → merged herds, installed all targets ✅
- `promptherder pull ... -dry-run` → logs without downloading ✅

## Full suite

- `go vet ./...` ✅
- `go test ./... -count=1` → all 3 packages PASS ✅
