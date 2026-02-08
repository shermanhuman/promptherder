# Execution Log: Target-Specific Skill Variants

## Batch 1: Core logic changes (parallel)

### Step 1: Antigravity target (`internal/app/antigravity.go`)

- Added `skillVariantFiles` map and `isInSkillDir` helper
- Walk logic: ANTIGRAVITY.md → installed as SKILL.md, COPILOT.md → skipped, SKILL.md → skipped if variant exists
- Verification: `go vet ./...` ✅, `go test ./...` ✅

### Step 2: Copilot target (`internal/app/copilot.go`)

- `buildCopilotSkillPrompts` now checks for COPILOT.md before SKILL.md
- Source label updated to track which file was used
- Verification: `go vet ./...` ✅, `go test ./...` ✅

## Batch 2: Tests

### Step 3: Tests (`internal/app/antigravity_test.go`, `internal/app/copilot_test.go`)

- 5 Antigravity tests: prefers variant, skips copilot, falls back to generic, all three files, isInSkillDir
- 3 Copilot tests: prefers copilot variant, falls back to generic, ignores antigravity variant
- Verification: `go test ./... -v -count=1 -run "Variant|IsInSkill|PrefersCopilot|FallsBack|IgnoresAntigravity"` ✅ (8/8 pass)
- Full suite: `go test ./...` ✅ (all pass)

## Batch 3: Content + docs (parallel)

### Step 4: Skill variant content

- `compound-v/skills/compound-v-parallel/SKILL.md` → generic (no agent-specific APIs)
- `compound-v/skills/compound-v-parallel/ANTIGRAVITY.md` → waitForPreviousTools details
- Verification: `go run ./cmd/promptherder && go run ./cmd/promptherder` ✅
  - `.agent/skills/compound-v-parallel/SKILL.md` contains `waitForPreviousTools` ✅
  - Only `SKILL.md` at target (no ANTIGRAVITY.md leak) ✅
  - Copilot prompt uses generic (no `waitForPreviousTools`) ✅

### Step 5: structure.md

- Added variant convention docs to naming section
- Verification: read file ✅
