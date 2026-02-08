# Execution: Convo-Based Artifact Management

## Batch 1 (parallel — Steps 1-6)

### Step 1: brainstorm.md workflow

- Files: `compound-v/workflows/brainstorm.md`
- Added slug resolution block, changed persist path to `convos/<slug>/brainstorm.md`
- Verify: ✅ `convos/<slug>` references present, zero `artifacts/` references

### Step 2: write-plan.md workflow

- Files: `compound-v/workflows/write-plan.md`
- Added slug block, brainstorm context loading, changed persist path, updated approval message with slug
- Verify: ✅ All paths updated

### Step 3: execute.md workflow

- Files: `compound-v/workflows/execute.md`
- Added slug block, updated precondition, execution log path, review output path (5 path changes)
- Verify: ✅ All 5 paths reference `convos/<slug>/`

### Step 4: review.md workflow

- Files: `compound-v/workflows/review.md`
- Added slug block, changed persist path
- Verify: ✅ Path updated

### Step 5: structure.md rule

- Files: `.agent/rules/structure.md`
- Added `convos/` and `<slug>/` to folder tree, marked `artifacts/` as legacy
- Verify: ✅ Tree reflects new layout

### Step 6: Create convos directory

- Files: `.promptherder/convos/.gitkeep`
- Verify: ✅ `Test-Path` returns True

## Batch 2 (Step 7)

### Promptherder reinstall

- First run: used cached binary — installed old embedded content
- Second run: recompiled, all 3 targets updated (copilot, antigravity, compound-v)
- Verify: ✅ All installed workflows contain `convos/<slug>` references
- Verify: ✅ Zero `artifacts/` references remain in any installed workflow

## Batch 3 (Step 8)

### Smoke test

- `go vet ./...`: ✅ Pass
- `go test ./...`: ✅ Pass (all packages)
- Stale references check: ✅ Zero `artifacts/` paths in `.agent/workflows/` or `.github/prompts/`
