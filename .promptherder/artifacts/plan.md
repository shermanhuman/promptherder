# Implementation Plan: Complete Test Coverage for promptherder

**Created**: 2026-02-08T06:33  
**Goal**: Add comprehensive test coverage for untested code paths identified in review  
**Source**: `.promptherder/artifacts/review.md` (Test Coverage Review)

---

## Goal

Achieve **85% statement coverage** for `internal/app` and **60% coverage** for `cmd/promptherder` by adding tests for:

1. Recent refactoring (setupRunner, persistAndClean) - **Critical blocker**
2. Multi-target orchestration (RunAll) - **Critical blocker**
3. Antigravity target implementation - **Major gap**
4. CompoundV target implementation - **Major gap**
5. CLI entry point logic - **Major gap**

This brings the test suite from **64% coverage** to **85% coverage** and adds regression protection for all user-facing features.

---

## Assumptions

- All existing tests must continue to pass
- Test files follow existing patterns (`*_test.go` in same package)
- Use `t.Run` for subtests and table-driven tests where appropriate
- Use real filesystem (`t.TempDir()`) instead of mocks
- Tests should be fast (<1s total suite runtime)

---

## Plan

### **Phase 1: Critical Blockers** (2-3 hours)

#### Step 1: Add runner helper tests (M3 - Critical)

**Priority**: Blocker  
**Estimated time**: 30 minutes

**Files**:

- `internal/app/runner_test.go` (new tests)

**Change**:

1. Add `TestSetupRunner_InitializesLogger` - verify nil logger is created
2. Add `TestSetupRunner_ResolvesAbsolutePath` - verify path resolution
3. Add `TestSetupRunner_LoadsManifest` - verify manifest loaded correctly
4. Add `TestSetupRunner_ErrorHandling` - verify error on invalid repo path
5. Add `TestPersistAndClean_DryRun` - verify no manifest written in dry-run
6. Add `TestPersistAndClean_ActualWrite` - verify manifest written + cleanup
7. Add `TestPersistAndClean_WriteError` - verify error propagation

**Verify**:

```powershell
go test ./internal/app -v -run TestSetupRunner
go test ./internal/app -v -run TestPersistAndClean
go test ./internal/app -coverprofile=coverage.out
go tool cover -func=coverage.out | findstr runner.go
```

**Success criteria**:

- 7 new tests pass
- `runner.go` coverage increases from ~40% to ~85%
- `setupRunner` and `persistAndClean` fully covered

---

#### Step 2: Add RunAll orchestration tests (M1 - Critical)

**Priority**: Blocker  
**Estimated time**: 45 minutes

**Files**:

- `internal/app/runner_test.go` (new tests)

**Change**:

1. Add `TestRunAll_AllTargetsExecute` - verify all 3 targets called
2. Add `TestRunAll_ManifestMergesTargets` - verify manifest has 3 target entries
3. Add `TestRunAll_ContextCancellation` - verify early exit on ctx.Done()
4. Add `TestRunAll_TargetFailureStopsExecution` - verify stops on first error
5. Add `TestRunAll_DryRunAllTargets` - verify dry-run doesn't write files

**Verify**:

```powershell
go test ./internal/app -v -run TestRunAll
.\promptherder.exe --dry-run -v
go test ./internal/app -coverprofile=coverage.out
```

**Success criteria**:

- 5 new tests pass
- Multi-target orchestration has regression protection
- Manifest merging logic verified

---

#### Step 3: Add RunTarget preservation tests (m2)

**Priority**: High  
**Estimated time**: 20 minutes

**Files**:

- `internal/app/runner_test.go` (new tests)

**Change**:

1. Add `TestRunTarget_PreservesOtherTargets` - run copilot, then antigravity, verify both in manifest
2. Add `TestRunTarget_ReplacesExistingTarget` - run copilot twice, verify only latest entry
3. Add `TestRunTarget_EmptyManifestStart` - run antigravity with no existing manifest

**Verify**:

```powershell
go test ./internal/app -v -run TestRunTarget
```

**Success criteria**:

- 3 new tests pass
- Target preservation logic verified

---

### **Phase 2: Major Gaps** (3-4 hours)

#### Step 4: Create Antigravity target tests (M2 - Major)

**Priority**: High  
**Estimated time**: 50 minutes

**Files**:

- `internal/app/antigravity_test.go` (new file)

**Change**:

1. Add `TestAntigravityTarget_BasicInstall` - sync .promptherder/agent → .agent
2. Add `TestAntigravityTarget_SkipsGeneratedFiles` - verify stack.md, structure.md skipped if they exist
3. Add `TestAntigravityTarget_DryRun` - verify no files written
4. Add `TestAntigravityTarget_MissingSource` - verify returns nil when .promptherder/agent missing
5. Add `TestAntigravityTarget_ContextCancellation` - verify stops on ctx.Done()
6. Add `TestAntigravityTarget_PreservesDirectoryStructure` - verify subdirs copied correctly
7. Add `TestAntigravityTarget_GeneratedFileFirstInstall` - verify generated files ARE written if they don't exist

**Verify**:

```powershell
go test ./internal/app -v -run TestAntigravity
.\promptherder.exe antigravity --dry-run -v
go test ./internal/app -coverprofile=coverage.out
go tool cover -func=coverage.out | findstr antigravity.go
```

**Success criteria**:

- 7 new tests pass
- `antigravity.go` coverage goes from 0% to ~90%
- All code paths tested

---

#### Step 5: Create CompoundV target tests (M2 - Major)

**Priority**: High  
**Estimated time**: 50 minutes

**Files**:

- `internal/app/compoundv_test.go` (new file)

**Change**:

1. Add `TestCompoundVTarget_NilFS` - verify error when FS is nil
2. Add `TestCompoundVTarget_InstallsFromEmbeddedFS` - use `fstest.MapFS` to test embedded FS sync
3. Add `TestCompoundVTarget_SkipsGeneratedFiles` - verify generated files skipped
4. Add `TestCompoundVTarget_DryRun` - verify dry-run behavior
5. Add `TestCompoundVTarget_ContextCancellation` - verify stops on ctx.Done()
6. Add `TestCompoundVTarget_InstallationMessage` - verify "fan out" message logged
7. Add `TestCompoundVTarget_PreservesDirectoryStructure` - verify rules/, skills/, workflows/ preserved

**Verify**:

```powershell
go test ./internal/app -v -run TestCompoundV
.\promptherder.exe compound-v --dry-run -v
go test ./internal/app -coverprofile=coverage.out
go tool cover -func=coverage.out | findstr compoundv.go
```

**Success criteria**:

- 7 new tests pass
- `compoundv.go` coverage goes from 0% to ~90%
- `fstest.MapFS` pattern documented for future embedded FS tests

---

#### Step 6: Add CLI unit tests (M3 - Major)

**Priority**: Medium  
**Estimated time**: 40 minutes

**Files**:

- `cmd/promptherder/main_test.go` (new file)

**Change**:

1. Add `TestExtractSubcommand` - table-driven test for all subcommand cases
2. Add `TestExtractSubcommand_UnknownCommand` - verify unknown commands return ""
3. Add `TestParseIncludePatterns` - table-driven test for CSV parsing
4. Add `TestParseIncludePatterns_EmptyString` - verify nil returned
5. Add `TestParseIncludePatterns_Whitespace` - verify trimming

**Verify**:

```powershell
go test ./cmd/promptherder -v
go test ./cmd/promptherder -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**Success criteria**:

- 5 new tests pass
- `main.go` coverage increases from 0% to ~40% (helpers covered)
- Table-driven test pattern demonstrated

---

#### Step 7: Add CLI integration tests (m3)

**Priority**: Medium  
**Estimated time**: 30 minutes

**Files**:

- `cmd/promptherder/integration_test.go` (new file, build tag: `//go:build integration`)

**Change**:

1. Add `TestCLI_Version` - verify `--version` output
2. Add `TestCLI_UnknownSubcommand` - verify exit code 2
3. Add `TestCLI_DryRunAllTargets` - verify `--dry-run` works with no subcommand
4. Add `TestCLI_DryRunSingleTarget` - verify `promptherder copilot --dry-run`

**Verify**:

```powershell
go build -o promptherder.exe ./cmd/promptherder
go test ./cmd/promptherder -v -tags=integration
```

**Success criteria**:

- 4 integration tests pass
- Tests run actual binary via `os/exec`
- Exit codes verified

---

### **Phase 3: Polish** (1-2 hours)

#### Step 8: Extract shared test helpers (m1)

**Priority**: Low  
**Estimated time**: 25 minutes

**Files**:

- `internal/app/testing_helpers_test.go` (new file)

**Change**:

1. Move `mustMkdir`, `mustWrite`, `assertContains`, `assertNotContains`,`assertTarget` from `sync_test.go`
2. Add `setupTestRepo(t *testing.T) (dir string)` helper
3. Add godoc comments for each helper
4. Update `sync_test.go` to use extracted helpers (already does, just remove duplicates)

**Verify**:

```powershell
go test ./internal/app -v
```

**Success criteria**:

- All tests still pass
- Test helpers consolidated
- sync_test.go reduced by ~30 lines

---

#### Step 9: Add manifest negative tests (m2)

**Priority**: Low  
**Estimated time**: 20 minutes

**Files**:

- `internal/app/manifest_test.go` (extend existing)

**Change**:

1. Add `TestReadManifest_CorruptedJSON` - verify returns empty manifest + logs warning
2. Add `TestReadManifest_InvalidVersion` - verify handles negative/large versions
3. Add `TestManifest_SetTarget_DuplicateNames` - verify last write wins

**Verify**:

```powershell
go test ./internal/app -v -run TestManifest
```

**Success criteria**:

- 3 new tests pass
- Edge case handling documented with tests

---

#### Step 10: Generate coverage report and update docs

**Priority**: Low  
**Estimated time**: 15 minutes

**Files**:

- `CONTRIBUTING.md` (update)
- `coverage.out` (generated)

**Change**:

1. Run full coverage analysis:
   ```powershell
   go test ./... -coverprofile=coverage.out -covermode=atomic
   go tool cover -html=coverage.out -o coverage.html
   go tool cover -func=coverage.out
   ```
2. Update `CONTRIBUTING.md` with:
   - Coverage targets (85% for internal/app, 60% for cmd)
   - How to run tests
   - How to check coverage
3. Document table-driven test pattern with example

**Verify**:

```powershell
start coverage.html
go tool cover -func=coverage.out | findstr total
```

**Success criteria**:

- HTML coverage report generated
- Total coverage ~80-85%
- CONTRIBUTING.md updated

---

## Risks & Mitigations

| Risk                                        | Likelihood | Impact | Mitigation                                                 |
| ------------------------------------------- | ---------- | ------ | ---------------------------------------------------------- |
| Tests break existing functionality          | Low        | High   | Run full test suite after each phase                       |
| Integration tests fail on CI                | Medium     | Medium | Use `//go:build integration` tag, document how to run      |
| fstest.MapFS doesn't match real embedded FS | Low        | Medium | Verify with actual embedded FS in CompoundV tests          |
| Coverage targets too ambitious              | Low        | Low    | Adjust targets if 85% proves unrealistic                   |
| Tests slow down development                 | Low        | Medium | Keep tests fast (<1s total), use `t.Parallel()` where safe |

---

## Rollback Plan

1. **Git**: Commit after Phase 1 (critical blockers)
2. **Git**: Commit after Phase 2 (major gaps)
3. **Git**: Commit after Phase 3 (polish)

If issues arise:

- **Phase 1**: Critical for v0.4.2 safety - must fix, not rollback
- **Phase 2**: Can ship without if time-constrained (document gaps)
- **Phase 3**: Pure polish, safe to defer

---

## Total Estimated Time

- **Phase 1 (Critical)**: Steps 1-3 = ~1 hour 35 minutes
- **Phase 2 (Major)**: Steps 4-7 = ~3 hours 10 minutes
- **Phase 3 (Polish)**: Steps 8-10 = ~1 hour

**Total**: ~5 hours 45 minutes (conservative estimate)

**Minimum viable**: Phase 1 only (~1.5 hours) addresses blockers

---

## Success Metrics

- [ ] All tests pass (100% green)
- [ ] `internal/app` coverage: 64% → 85%
- [ ] `cmd/promptherder` coverage: 0% → 60%
- [ ] `runner.go` helpers: 0% → 100% coverage
- [ ] `antigravity.go`: 0% → 90% coverage
- [ ] `compoundv.go`: 0% → 90% coverage
- [ ] Zero behavior changes (tests document existing behavior)
- [ ] Test suite runtime: <2 seconds

---

## Architecture Clarity Note

**Corrected**: The system has **2 agents** (Copilot, Antigravity) and **3 targets**:

1. **compound-v target** → Installs methodology to `.promptherder/agent/` (source of truth)
2. **copilot target** → Syncs `.promptherder/agent/` → `.github/` (Copilot agent)
3. **antigravity target** → Syncs `.promptherder/agent/` → `.agent/` (Antigravity agent)

Running `promptherder` executes all 3 targets, but only 2 are AI agents.

---

**Plan Status**: Ready for approval  
**Next Step**: Reply **APPROVED** to begin implementation with `/execute`
