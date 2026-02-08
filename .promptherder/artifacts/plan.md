# Implementation Plan: Eliminate DRY Violations

**Created**: 2026-02-07T22:54  
**Goal**: Refactor promptherder codebase to eliminate all identified code duplication  
**Source**: `.promptherder/artifacts/review.md` (DRY-focused review)

---

## Goal

Eliminate all DRY (Don't Repeat Yourself) violations in the promptherder codebase by extracting duplicated code into shared helpers and abstractions. This will:

1. Reduce ~150 lines of duplicated code to ~50 lines of shared helpers
2. Make bug fixes and enhancements easier (change in one place instead of 2-3)
3. Improve testability and maintainability
4. Set patterns for future target implementations

---

## Assumptions

- All existing tests must pass after each change
- No changes to public API or CLI interface
- Refactoring is internal to `internal/app/` package
- The `Target` interface remains unchanged
- Test coverage sufficient to catch regressions

---

## Plan

### Step 1: Extract manifest initialization helper (M3)

**Priority**: High (smallest, safest change - good warmup)  
**Estimated time**: 10 minutes

**Files**:

- `internal/app/manifest.go`
- `internal/app/runner.go`
- `internal/app/sync.go`

**Change**:

1. Add `newManifestFrom(prev manifest) manifest` function to `manifest.go`
2. Replace 3 instances of manual manifest initialization with helper call:
   - `runner.go:32-36` → `curManifest := newManifestFrom(prevManifest)`
   - `runner.go:88-92` → `curManifest := newManifestFrom(prevManifest)`
   - `sync.go:134-138` → `curManifest := newManifestFrom(prevManifest)`

**Verify**:

```powershell
go test ./internal/app/... -v
go build ./cmd/promptherder
```

**Success criteria**: All tests pass, binary builds

---

### Step 2: Extract runner setup/teardown helpers (M1)

**Priority**: High (high impact, well-isolated)  
**Estimated time**: 20 minutes

**Files**:

- `internal/app/runner.go`

**Change**:

1. Add `setupRunner(cfg *Config) (repoPath string, prevManifest manifest, tcfg TargetConfig, err error)` function
2. Add `persistAndClean(repoPath string, prev, cur manifest, dryRun bool, logger *slog.Logger) error` function
3. Refactor `RunAll` to use both helpers
4. Refactor `RunTarget` to use both helpers

**Verify**:

```powershell
go test ./internal/app/... -v -run TestRun
.\promptherder.exe --dry-run
.\promptherder.exe antigravity --dry-run
```

**Success criteria**:

- All runner tests pass
- Dry-run commands work correctly
- Code in `RunAll` and `RunTarget` reduced from ~50 lines each to ~25 lines each

---

### Step 3: Add manifest version constant (n1)

**Priority**: Low (quick win)  
**Estimated time**: 5 minutes

**Files**:

- `internal/app/manifest.go`

**Change**:

1. Add `const manifestVersion = 2` at package level
2. Replace all hard-coded `2` values with `manifestVersion`:
   - `writeManifest` function (line 95)
   - `newManifestFrom` helper (from Step 1)

**Verify**:

```powershell
go test ./internal/app/... -v -run TestManifest
```

**Success criteria**: All manifest tests pass

---

### Step 4: Extract default logger helper (m1)

**Priority**: Medium (reduces test boilerplate)  
**Estimated time**: 15 minutes

**Files**:

- `internal/app/logger.go` (new file)
- `internal/app/runner.go`
- `internal/app/sync_test.go`

**Change**:

1. Create `internal/app/logger.go` with:
   ```go
   func defaultLogger() *slog.Logger {
       return slog.New(slog.NewTextHandler(os.Stderr, nil))
   }
   ```
2. Replace pattern in `setupRunner` (from Step 2)
3. Replace pattern in 3-5 test files (grep search confirmed at least 6 instances)

**Verify**:

```powershell
go test ./internal/app/... -v
```

**Success criteria**: All tests pass, logger creation uses helper

---

### Step 5: Extract generic file installer (M2 - Part 1: Create abstraction)

**Priority**: High (most complex, highest value)  
**Estimated time**: 30 minutes

**Files**:

- `internal/app/install.go` (new file)

**Change**:

1. Create `install.go` with documented file installer abstraction:

   ```go
   // fileWalker abstracts over filesystem and embedded FS walking
   type fileWalker func(visit func(path string, isDir bool) error) error

   // fileReader abstracts over reading from filesystem or embedded FS
   type fileReader func(path string) ([]byte, error)

   // installFiles handles the common file installation pattern
   func installFiles(
       ctx context.Context,
       cfg TargetConfig,
       m manifest,
       walker fileWalker,
       reader fileReader,
       relPath func(path string) (string, error),
       targetDir string,
   ) ([]string, error)
   ```

2. Implement full logic with all shared patterns:
   - Context cancellation checking
   - Skip directories
   - Generated file checking
   - Dry-run vs actual write
   - Logging
   - Installed tracking

**Verify**:

```powershell
go build ./internal/app/...
```

**Success criteria**: Package compiles, helper function is complete and documented

---

### Step 6: Refactor AntigravityTarget to use installer (M2 - Part 2)

**Priority**: High  
**Estimated time**: 20 minutes

**Files**:

- `internal/app/antigravity.go`

**Change**:

1. Adapt `AntigravityTarget.Install()` to use `installFiles` helper
2. Create wrapper functions for:
   - Walker: `filepath.Walk` adapter
   - Reader: `os.ReadFile` wrapper
   - RelPath: `filepath.Rel` wrapper
3. Reduce `Install()` from ~50 lines to ~15-20 lines

**Verify**:

```powershell
go test ./internal/app/... -v -run TestAntigravity
.\promptherder.exe antigravity --dry-run -v
```

**Success criteria**:

- All Antigravity tests pass
- Dry-run shows correct output
- Files are still synced correctly

---

### Step 7: Refactor CompoundVTarget to use installer (M2 - Part 3)

**Priority**: High  
**Estimated time**: 20 minutes

**Files**:

- `internal/app/compoundv.go`

**Change**:

1. Adapt `CompoundVTarget.Install()` to use `installFiles` helper
2. Create wrapper functions for:
   - Walker: `fs.WalkDir` adapter
   - Reader: `fs.ReadFile` wrapper
   - RelPath: embedded path extraction
3. Reduce `Install()` from ~50 lines to ~15-20 lines

**Verify**:

```powershell
go test ./internal/app/... -v -run TestCompound
.\promptherder.exe compound-v --dry-run -v
```

**Success criteria**:

- All Compound V tests pass
- Installation message still appears
- Files are synced correctly

---

### Step 8: Add test helper for repo setup (m3)

**Priority**: Low (improves test maintenance)  
**Estimated time**: 25 minutes

**Files**:

- `internal/app/testing_helpers_test.go` (new file, test-only)

**Change**:

1. Create test helper file with `// +build test` tag
2. Add `setupTestRepo(t *testing.T) (dir string, cleanup func())`
3. Add `mustWrite(t *testing.T, path, content string)`
4. Refactor 3-4 test functions to use helpers as proof of concept

**Verify**:

```powershell
go test ./internal/app/... -v
```

**Success criteria**: All tests pass with less boilerplate

---

### Step 9: Run full test suite and build

**Priority**: Critical  
**Estimated time**: 10 minutes

**Files**: All

**Change**: None (verification only)

**Verify**:

```powershell
# Full test suite
go test ./... -v -race -coverprofile=coverage.out

# Build binary
go build -o promptherder.exe ./cmd/promptherder

# Smoke test
.\promptherder.exe --version
.\promptherder.exe --dry-run -v
.\promptherder.exe compound-v
.\promptherder.exe antigravity
.\promptherder.exe copilot

# Verify manifest
cat .promptherder\manifest.json
```

**Success criteria**:

- All tests pass
- Binary builds
- All commands work
- Manifest is valid JSON
- Coverage remains >= current level

---

### Step 10: Update manifest to track generated rules

**Priority**: Low (finish cleanup)  
**Estimated time**: 5 minutes

**Files**:

- `.promptherder/manifest.json`

**Change**:

1. Add `stack.md` and `structure.md` to `manifest.generated` list
2. This prevents promptherder from overwriting these agent-generated rules

**Verify**:

```powershell
cat .promptherder\manifest.json | jq .generated
```

**Success criteria**: JSON contains `["stack.md", "structure.md"]` in generated array

---

## Risks & Mitigations

| Risk                             | Likelihood | Impact | Mitigation                                                           |
| -------------------------------- | ---------- | ------ | -------------------------------------------------------------------- |
| Break existing tests             | Medium     | High   | Run tests after each step; rollback if failures                      |
| Change behavior of targets       | Low        | High   | Extensive manual testing with --dry-run; compare before/after output |
| Generic helper too complex       | Medium     | Medium | Keep abstraction minimal; only extract truly shared code             |
| Introduce performance regression | Low        | Low    | File operations already I/O bound; abstraction overhead negligible   |

---

## Rollback Plan

1. **Git**: Commit after Steps 1-4 (low-risk changes)
2. **Git**: Commit after Steps 5-7 (target refactoring)
3. **Git**: Tag before Step 10 (final state)

If issues arise:

- **Steps 1-4**: Simple revert (pure extraction, no behavior change)
- **Steps 5-7**: Revert to pre-refactoring commit; targets are independent
- **Steps 8-10**: Low risk, test-only and metadata changes

---

## Total Estimated Time

- **High priority**: Steps 1-2, 5-7 = ~2 hours 15 minutes
- **Medium priority**: Step 4 = ~15 minutes
- **Low priority**: Steps 3, 8, 10 = ~35 minutes
- **Verification**: Step 9 = ~10 minutes

**Total**: ~3 hours 15 minutes (conservative estimate with testing)

---

## Success Metrics

- [ ] All tests pass (100% CI green)
- [ ] Binary builds successfully
- [ ] All CLI commands work (manual smoke test)
- [ ] Code duplication reduced from ~150 to ~50 lines
- [ ] Test coverage maintained or improved
- [ ] `RunAll` and `RunTarget` reduced from ~50 lines each to ~25 lines
- [ ] Target `Install()` methods reduced from ~50 lines to ~15-20 lines
- [ ] Zero behavior changes (output identical to before)

---

**Plan Status**: Ready for approval  
**Next Step**: Wait for APPROVED, then run `/execute`
