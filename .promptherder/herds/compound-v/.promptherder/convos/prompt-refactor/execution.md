# Execution: prompt-refactor

## Batch 1 (all steps — independent files)

### Step 1: Output formatting rules in `compound-v.md` ✅

- Added `## Output formatting` section with severity indicators, decision prompts, short-names-first, YOLO mode
- File: `rules/compound-v.md`

### Step 2: Plan skill enhancements ✅

- Added visual anchors (happy path, filesystem tree, references) to Phase 4 template
- Added TDD sequencing rule to plan steps
- Added YAGNI/DRY checks to Phase 3
- Added parallel research with `waitForPreviousTools: false`
- Added hard-rules.md read to research phase
- Added blockquote deferred ideas prompt
- File: `skills/compound-v-plan/SKILL.md`

### Step 2b: Plan workflow approval flow ✅

- Replaced APPROVED prompt with `/execute` (running /execute IS the approval)
- Added DECLINE option with status tracking
- All prompts use blockquotes
- File: `workflows/plan.md`

### Step 3: Review skill overhaul ✅

- 10 parallel domains with short names (correctness, edges, security, perf, tests, design, dry, yagni, logging, docs)
- Senior engineer persona
- Domain targeting (`/review security`)
- Version-specific research from stack.md per domain
- Findings-first workflow: strengths → findings → verdict
- File:line references mandatory
- "How to fix" per finding
- YOLO mode
- Git diff review method
- Idiomatic code check in design domain
- YAGNI grep technique
- Hard-rules check in correctness domain
- Fix triage by severity
- File: `skills/compound-v-review/SKILL.md`

### Step 4: Execute workflow update ✅

- /execute IS the approval (sets status to approved)
- Findings-first finish: present → user response → fix
- YOLO mode finish: auto-review → auto-fix → summary
- Smoke test section
- Reads hard-rules.md as context
- Blockquote decision prompt
- File: `workflows/execute.md`

### Step 5: Parallel research enforcement ✅

- TDD: explicit `waitForPreviousTools: false` note
- Debug: explicit `waitForPreviousTools: false` note
- Review: already explicit ✅
- Plan: already explicit ✅
- Files: `skills/compound-v-tdd/SKILL.md`, `skills/compound-v-debug/SKILL.md`

### Step (bonus): Review workflow update ✅

- Added domain targeting support
- Added YOLO mode support
- Added slug+domain combination syntax
- File: `workflows/review.md`

## Verification

- `parallel` keyword present in all 4 skills ✅
- `APPROVED` removed from all user-facing prompts ✅
- All decision prompts use blockquotes (`>`) ✅
- All domains have short names ✅
