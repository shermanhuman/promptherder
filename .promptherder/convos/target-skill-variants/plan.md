# Plan: Target-Specific Skill Variants

> `/plan` · **Status**: approved · `.promptherder/convos/target-skill-variants/plan.md`

## Goal

Allow skills to have target-specific variants. `SKILL.md` is the generic version; `ANTIGRAVITY.md` / `COPILOT.md` replace it when installing to that specific target as `<skill-name>/SKILL.md`.

## Plan

1. Update Antigravity install to prefer target-specific skill files
   - Files: `internal/app/antigravity.go`
   - Change: During walk, if file is `ANTIGRAVITY.md` in `skills/*/`, install as `<skill-name>/SKILL.md`. If `SKILL.md` exists alongside `ANTIGRAVITY.md`, skip generic. Skip other targets' variants.
   - Verify: `go test ./...`

2. Update Copilot install to prefer target-specific skill files
   - Files: `internal/app/copilot.go`
   - Change: In `buildCopilotSkillPrompts`, check for `COPILOT.md` before `SKILL.md`.
   - Verify: `go test ./...`

3. Add tests for target-specific skill variant selection
   - Files: `internal/app/antigravity_test.go`, `internal/app/copilot_test.go`
   - Change: Test cases with variant files. Verify variant installed as SKILL.md, generic skipped.
   - Verify: `go test ./...`

4. Create ANTIGRAVITY.md for compound-v-parallel
   - Files: `compound-v/skills/compound-v-parallel/ANTIGRAVITY.md`, `compound-v/skills/compound-v-parallel/SKILL.md`
   - Change: Move waitForPreviousTools content to ANTIGRAVITY.md. Make SKILL.md generic.
   - Verify: `go run ./cmd/promptherder && go run ./cmd/promptherder`, inspect `.agent/skills/compound-v-parallel/SKILL.md`

5. Update structure.md
   - Files: `compound-v/rules/structure.md`
   - Change: Document SKILL.md + TARGET.md convention
   - Verify: Read file

## Risks & mitigations

- **Risk**: Variant file not detected → generic installed instead. **Mitigation**: Tests explicitly verify variant preference.
- **Risk**: Variant files leak to targets (e.g. COPILOT.md appears in .agent/). **Mitigation**: Antigravity walker skips all non-matching variant files.

## Rollback plan

Revert the commit. No data migration, no schema changes.

## Deferred

- Rename CompoundVTarget to Methodology with separate Extract() interface
