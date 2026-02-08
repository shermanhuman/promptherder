# Senior Engineer Test Review

**Reviewer**: Senior Go Engineer (simulated)  
**Date**: 2026-02-08  
**Scope**: All new test files from the test coverage initiative  
**Files Reviewed**: 7 test files + 5 production source files

---

## Blockers

### B1: `os.Chdir` in `TestSetupRunner_ResolvesAbsolutePath` is not parallel-safe

**File**: `runner_helper_test.go:37-71`  
**Severity**: Blocker

`os.Chdir` mutates global process state. If this test runs concurrently with any other test — including via `go test -count=N` or when `t.Parallel()` is later added — it will corrupt the working directory for every other goroutine in the process. The `defer os.Chdir(cwd)` is a race condition, not a safety mechanism.

**Impact**: Prevents safely adding `t.Parallel()` to the entire suite. Could cause rare, maddening CI flakes.

**Fix**: Eliminate `os.Chdir` entirely. Instead, test that `filepath.Abs(".")` produces the expected path in a separate helper, or just verify the function's contract directly:

```go
func TestSetupRunner_ResolvesAbsolutePath(t *testing.T) {
    dir := t.TempDir()
    cfg := Config{
        RepoPath: dir, // already absolute from TempDir
        Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
    }
    repoPath, _, _, err := setupRunner(&cfg)
    if err != nil {
        t.Fatal(err)
    }
    if !filepath.IsAbs(repoPath) {
        t.Errorf("expected absolute path, got %s", repoPath)
    }
}
```

If you truly need to test relative-to-absolute resolution, use `t.Setenv("PWD", ...)` or accept that it's an implementation detail of `filepath.Abs` that doesn't need testing.

---

### B2: Unused import `"strings"` in `manifest_edge_test.go`

**File**: `manifest_edge_test.go:7`  
**Severity**: Blocker

`"strings"` is imported but never used. **This will fail `go vet`** and will break CI on any pipeline that runs `go vet ./...` or golangci-lint.

**Fix**: Remove `"strings"` from the import block.

---

## Majors

### M1: Zero use of `t.Parallel()` across all new test files

**Files**: All 7 new test files  
**Severity**: Major

Not a single test calls `t.Parallel()`. The user specifically asked about parallel execution.

**Current runtime**: 0.5s — acceptable for now, but this won't scale.

**Which tests CAN safely be parallelized** (each uses its own `t.TempDir()`):

- ✅ Every test in `antigravity_test.go` (except if `os.Chdir` B1 is present nearby in the package)
- ✅ Every test in `compoundv_test.go`
- ✅ Every test in `manifest_edge_test.go`
- ✅ Every test in `runner_orchestration_test.go`
- ✅ Most tests in `runner_helper_test.go` (except `TestSetupRunner_ResolvesAbsolutePath` — see B1)
- ✅ Every test in `cmd/promptherder/main_test.go` (pure function tests, no I/O)

**Which tests CANNOT be parallelized** without fix:

- ❌ `TestSetupRunner_ResolvesAbsolutePath` — `os.Chdir` (process-global)

**Fix**: Fix B1 first, then add `t.Parallel()` to every top-level `Test*` function and every `t.Run` subtest. Pattern:

```go
func TestFoo(t *testing.T) {
    t.Parallel()
    // ...
}

func TestBar_TableDriven(t *testing.T) {
    t.Parallel()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // ...
        })
    }
}
```

---

### M2: Extracted test helpers `setupTestRepo`, `writeTestManifest`, `readTestFile`, `assertFileExists`, `assertFileNotExists` are never called

**File**: `testing_helpers_test.go:52-102`  
**Severity**: Major

Six helper functions were created in the extraction step but are **dead code** — only `mustMkdir`, `mustWrite`, `assertTarget`, `assertContains`, `assertNotContains`, and `createTestFile` are actually used by the test suite. The others were aspirational additions that no test calls.

Dead test helper code creates confusion: a future reader sees `assertFileExists` and wonders why no one uses it, or assumes it's called indirectly. It also signals that the helpers were designed without being driven by actual test needs (the opposite of TDD).

**Fix**: Either:

1. **Delete** `setupTestRepo`, `writeTestManifest`, `readTestFile`, `assertFileExists`, `assertFileNotExists` — they aren't adding value today.
2. Or **refactor existing tests** to use them (e.g., many tests inline `t.TempDir()` and `writeManifest()` calls that could use these helpers).

I recommend option 1. YAGNI. Bring them back when a test actually needs them.

---

### M3: `TestManifest_IsGenerated_CaseInsensitive` is misnamed

**File**: `manifest_edge_test.go:109`  
**Severity**: Major

The test is named `TestManifest_IsGenerated_CaseInsensitive` but actually verifies that `isGenerated` is **case-sensitive**. The test expectations confirm `"STACK.MD"` returns `false`. This is a documentation lie — test names ARE documentation.

**Fix**: Rename to `TestManifest_IsGenerated_CaseSensitive`. Then the comments `// isGenerated is case-sensitive` are redundant and can be removed (the test name says it all).

---

### M4: Error matching via `strings.Contains(err.Error(), ...)` instead of `errors.Is`

**Files**: `antigravity_test.go:226`, `compoundv_test.go:30,211`, `runner_helper_test.go:141`, `runner_orchestration_test.go:157`  
**Severity**: Major

Six places match errors by string content. This is fragile: if the error message changes, the test breaks. For sentinel errors like `context.Canceled`, the idiomatic Go approach is:

```go
if !errors.Is(err, context.Canceled) {
    t.Errorf("expected context.Canceled, got %v", err)
}
```

`runner_orchestration_test.go:111` already does this correctly! The rest of the suite should follow the same pattern.

For wrapped errors like `"resolve repo path: ..."`, `strings.Contains` is acceptable IF there's no exported sentinel. But the context cancellation tests (antigravity and compoundv) should absolutely use `errors.Is`.

**Fix**: Replace `strings.Contains(err.Error(), "context canceled")` with `errors.Is(err, context.Canceled)` in `antigravity_test.go` and `compoundv_test.go`.

---

## Minors

### m1: `TestManifest_TargetFilesDeduplication` doesn't actually assert anything

**File**: `manifest_edge_test.go:186-206`  
**Severity**: Minor

This test has conditional branches that either `t.Log` or check deduplication, but neither branch calls `t.Error` or `t.Fatal` for the "4 files" case. It's a test that never fails. A test that never fails is not a test — it's a comment.

**Fix**: Pick the behavior you want and assert it:

```go
func TestManifest_SetTarget_AcceptsDuplicates(t *testing.T) {
    m := manifest{Version: 2}
    m.setTarget("test", []string{"file.md", "file.md", "other.md"})

    // setTarget does not deduplicate; it stores what it's given.
    if len(m.Targets["test"]) != 3 {
        t.Errorf("expected 3 entries (no dedup), got %d", len(m.Targets["test"]))
    }
}
```

---

### m2: `TestManifest_WindowsPathSeparators` doesn't assert anything meaningful

**File**: `manifest_edge_test.go:356-371`  
**Severity**: Minor

The test stores mixed path separators and then logs them with `t.Logf`. It verifies `len(files) == 2` (trivially true — we stored 2 paths) but doesn't test any real behavior like normalization or round-trip through write/read.

**Fix**: Either assert something about how these paths interact with `cleanStale` (cross-platform normalization), or delete the test. A test without a meaningful assertion is cargo-culting.

---

### m3: Repeated boilerplate: `logger := slog.New(slog.NewTextHandler(os.Stderr, nil))`

**Files**: All test files (appears ~30 times)  
**Severity**: Minor

Every single test creates the same `slog.Logger` to `os.Stderr`. This is a prime candidate for the test helpers file.

**Fix**: Add to `testing_helpers_test.go`:

```go
// testLogger returns a logger that writes to os.Stderr.
// Use discardLogger for tests that capture log output.
func testLogger(t *testing.T) *slog.Logger {
    t.Helper()
    return slog.New(slog.NewTextHandler(os.Stderr, nil))
}
```

Then replace all 30 instances with `logger := testLogger(t)`.

---

### m4: `TestRunAll_TargetFailureStopsExecution` has a comment that hedges on behavior

**File**: `runner_orchestration_test.go:171-173`

```go
// Note: target-3 may or may not be executed depending on implementation
// The current implementation doesn't explicitly stop, so target-2 failure might not prevent target-3
// This test documents actual behavior
```

But looking at `runner.go:63-64`, it clearly returns on the first error:

```go
if err != nil {
    return fmt.Errorf("target %s: %w", t.Name(), err)
}
```

Target-3 will **never** execute after target-2 fails. The comment confuses readers. The test should assert `!executed["target-3"]`.

**Fix**:

```go
// target-3 should NOT have executed because target-2 returned an error.
if executed["target-3"] {
    t.Error("target-3 should not execute after target-2 fails")
}
```

---

### m5: `TestRunAll_ContextCancellation` checks cancellation before loop, but Install() also checks context

**File**: `runner_orchestration_test.go:81-113`  
**Severity**: Minor

The mock's `installFunc` checks `ctx.Err()` itself, but the real `RunAll` loop checks `ctx.Err()` _before_ calling `Install()`. So the mock's Install is never reached — the context check at `runner.go:57` returns first. The mock's internal context check is dead code and misleading.

**Fix**: Simplify the mock — it doesn't need to check context since it will never be called:

```go
installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
    t.Error("Install should not be called with cancelled context")
    return nil, nil
},
```

---

### m6: `failingFS` in `compoundv_test.go` doesn't implement `fs.FS` correctly for `WalkDir`

**File**: `compoundv_test.go:367-377`  
**Severity**: Minor

`failingFS.Open()` only returns errors or `fs.ErrNotExist`. `fs.WalkDir` calls `Open` on the root directory first and expects a `fs.ReadDirFile`. Since `Open("compound-v")` returns `fs.ErrNotExist`, the walk fails before ever reaching `test.md`.

The test passes because it expects _any_ error — but it's not testing the read-failure scenario it claims to test. It's actually testing the "walk root doesn't exist" error.

**Fix**: Either:

1. Rename to `TestCompoundVTarget_WalkRootError` to match what it actually tests.
2. Or implement a proper `fs.FS` that lets the walk succeed but fails on `ReadFile`.

---

## Nits

### N1: Test names in `manifest_edge_test.go` use `Read`/`Write` prefix inconsistently

Some tests say `TestReadManifest_...`, others say `TestManifest_...`, and some say `TestWrite Manifest_...`. In the same file, `TestCleanStale_PathNormalization` follows a different convention.

**Fix**: Pick one convention: `TestManifest_<Method>_<Scenario>` or `Test<Function>_<Scenario>`. Be consistent.

---

### N2: `\\n` literal strings in test content

**Files**: `antigravity_test.go`, `compoundv_test.go`

Many tests write content like `"# Test\\n"` — this writes the literal characters `\n` (backslash + n), not a newline. It works because the tests don't care about content parsing, but it's confusing to read.

**Fix**: Use `"# Test\n"` (actual newline) for clarity. If the content genuinely doesn't matter, consider `"irrelevant content"`.

---

### N3: `runner_orchestration_test.go:36` loop variable capture is unnecessary on Go 1.22+

```go
mt := mt // Capture loop variable
```

Since Go 1.22, the loop variable is re-scoped per iteration. With Go 1.25 in `stack.md`, this line is a no-op. It's not harmful but is dated.

**Fix**: Remove the `mt := mt` line and the comment.

---

### N4: `TestWriteManifest_PermissionError` — keep it or kill it

**File**: `manifest_edge_test.go:158-183`

The test is permanently skipped on the primary development platform (Windows). A test that never runs is noise.

**Fix**: Either:

1. Replace with a UNIX-only build tag (`//go:build !windows`) to eliminate the skip noise.
2. Or delete entirely — `chmod` tests are platform-specific and inherently flaky.

---

## Things Done Well ✅

### Correct patterns

- **Table-driven tests**: CLI tests (`main_test.go`) are excellent examples of well-structured table-driven tests.
- **Real filesystem testing**: Using `t.TempDir()` everywhere instead of mocking the filesystem. This is the right approach for a file-syncing tool.
- **`fstest.MapFS`**: The CompoundV tests demonstrate the correct, idiomatic way to test against embedded filesystems without needing real `embed` directives.
- **`targetFunc` adapter**: Clean interface-to-function adapter for mocking `Target`. Avoids heavy mock frameworks.
- **Generated-file protection is well tested**: Both first-install and skip-when-exists scenarios are covered for both targets.
- **DryRun is tested everywhere**: Every target has a dry-run test confirming no writes occur.
- **Missing-source is graceful**: AntigravityTarget handles missing dirs without error — and there's a test for it.
- **`t.Helper()` on all helpers**: Every function in `testing_helpers_test.go` calls `t.Helper()`, so failure lines point to the calling test.
- **Log capture pattern**: `TestCompoundVTarget_InstallationMessage` captures logs via `strings.Builder` — clean and dependency-free.

### Correct test boundaries

- Tests test **behavior**, not implementation details (with minor exceptions noted above).
- Tests use the production APIs (`Install`, `RunAll`, `RunTarget`) rather than reaching into internals.
- Manifest edge tests probe the serialization boundary correctly (write → read → verify round-trip).

### Things we should NOT be testing (and correctly aren't)

- ✅ `filepath.Abs` itself
- ✅ JSON marshal/unmarshal correctness
- ✅ `os.MkdirAll` actually creating dirs
- ✅ The Go `slog` package

---

## Missing Edge Cases

### E1: `RunAll` with a nil target list (not empty — nil)

```go
err := RunAll(ctx, nil, cfg)
```

Currently only `[]Target{}` is tested.

### E2: `AntigravityTarget.Install` with a symlink in the source tree

Symlinks inside `.promptherder/agent/` could cause `filepath.Walk` to follow them or skip them. This is relevant on UNIX CI.

### E3: `CompoundVTarget.Install` with a file containing zero bytes

An empty file in the embedded FS might exercise different code paths (e.g., `len(installed) > 0` is still true).

### E4: `cleanStale` dry-run verification

`TestPersistAndClean_DryRun` verifies the manifest wasn't written, but doesn't verify that stale files were NOT deleted during dry-run. The dry-run test for `persistAndClean` should create a stale file and confirm it survives.

### E5: `runTarget` error from `setupRunner`

No test covers the path where `RunTarget` or `RunAll` receive an invalid `Config.RepoPath`. The `setupRunner` error test exists but only tests `setupRunner` directly — the orchestration-level error propagation isn't tested.

---

## Summary + Next Actions

### Summary

The test suite is **solid at the behavioral level** — it tests the right things at the right boundaries and achieves excellent coverage (90.1%). The biggest gap is the **complete absence of `t.Parallel()`**, which the user specifically asked about and which is standard Go practice for I/O-heavy tests. There's also a process-global `os.Chdir` that makes the entire package unsafe for parallel execution, an unused import that will fail `go vet`, and several dead-code helpers.

### Prioritized Next Actions

1. **Fix B1**: Remove `os.Chdir` from `TestSetupRunner_ResolvesAbsolutePath`
2. **Fix B2**: Remove unused `"strings"` import from `manifest_edge_test.go`
3. **Fix M1**: Add `t.Parallel()` to all tests and subtests
4. **Fix M2**: Delete unused helper functions from `testing_helpers_test.go`
5. **Fix M3**: Rename `TestManifest_IsGenerated_CaseInsensitive` → `CaseSensitive`
6. **Fix M4**: Use `errors.Is(err, context.Canceled)` instead of string matching
7. **Fix m4**: Assert `target-3` was not executed in failure-stops test
8. (Optional) Add missing edge cases E1-E5

---

**Overall Grade**: **B+**

Strong behavioral coverage and correct test boundaries. Falls short on Go idioms (`t.Parallel`, `errors.Is`) and has some housekeeping issues (dead helpers, naming, unused imports) that would get flagged in any serious code review.
