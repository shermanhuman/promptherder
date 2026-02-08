# Execution: Merge Brainstorm + Write-Plan → /plan

## Batch 1 (parallel — Steps 1-6)

### Step 1: Create `compound-v/workflows/plan.md`

- Created new interactive planning workflow (~100 lines)
- Sections: Init, Ideation Loop, Draft, Approval, Plan Artifact Format
- Verify: ✅ File exists

### Step 2: Delete `compound-v/workflows/brainstorm.md`

- Verify: ✅ `Test-Path` returns False

### Step 3: Delete `compound-v/workflows/write-plan.md`

- Verify: ✅ `Test-Path` returns False

### Step 4: Delete `compound-v/skills/compound-v-brainstorm/`

- Verify: ✅ `Test-Path` returns False

### Step 5: Update `compound-v/skills/compound-v-plan/SKILL.md`

- Rewrote to describe ideation loop pattern, ideas table, investigation techniques
- Verify: ✅ File contains ideation loop description

### Step 6: Update `compound-v/rules/compound-v.md`

- Pipeline: 3 steps (`/plan` → `/execute` → `/review`)
- Skills: 5 skills (no brainstorm)
- Verify: ✅ No "brainstorm" references

## Batch 2 (Step 7)

### Run promptherder

- First run: cached binary installed old files, but cleanStale removed stale from `.promptherder/agent/`
- Second run: fresh compile, all 3 targets updated. cleanStale removed:
  - `.agent/skills/compound-v-brainstorm/SKILL.md`
  - `.agent/workflows/brainstorm.md`
  - `.agent/workflows/write-plan.md`
  - `.github/prompts/brainstorm.prompt.md`
  - `.github/prompts/compound-v-brainstorm.prompt.md`
  - `.github/prompts/write-plan.prompt.md`
- Manually removed empty directories left by cleanStale

## Batch 3 (Step 8)

### Verification

| Check                                         | Result       |
| --------------------------------------------- | ------------ |
| `.agent/workflows/plan.md` exists             | ✅ True      |
| `.agent/workflows/brainstorm.md` exists       | ✅ False     |
| `.agent/workflows/write-plan.md` exists       | ✅ False     |
| `.agent/skills/compound-v-brainstorm` exists  | ✅ False     |
| `.github/prompts/plan.prompt.md` exists       | ✅ True      |
| `.github/prompts/brainstorm.prompt.md` exists | ✅ False     |
| `.github/prompts/write-plan.prompt.md` exists | ✅ False     |
| "ACCEPT" in `.agent/workflows/plan.md`        | ✅ 4 matches |
| "brainstorm" in `.agent/rules/compound-v.md`  | ✅ 0 matches |
| `go vet ./...`                                | ✅ Pass      |
| `go test ./...`                               | ✅ Pass      |
