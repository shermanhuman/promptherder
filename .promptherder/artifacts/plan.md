# Plan — Compound V Migration & promptherder v2

## Goal

1. **Migrate the intelligence from `superpowers` (and `gemini-superpowers-antigravity`) into `compound-v`**, a streamlined, native version for Antigravity.
2. **Architect `promptherder` as an idempotent agent configuration manager** that fans out these capabilities.
3. Establish `.promptherder/agent/` as the single source of truth for project rules.
4. Implement extensible "Target" architecture (Copilot, Antigravity, Compound V) with easy contribution flow.

## Core Philosophy

- **Configuration Management, not Package Management:** We manage the _state_ of AI agent configurations (.agent/, .github/) from a single source (.promptherder/). We don't fetch remote dependencies.
- **Migration, not Reinvention:** We are moving the _proven_ prompts from `superpowers`. We are only changing their _packaging_ (Python wrapper → native Antigravity files) and _location_.
- **Idempotency:** Running `promptherder` once or 100 times must result in the exact same state. No duplicate entries, no half-written files.
- **Single Source of Truth:** Humans allow list rules in `.promptherder/agent/`. Machines generate `.agent/` and `.github/`.

## Architecture: The "Fan-Out"

```
[SOURCE]                                      [TARGETS]
.promptherder/agent/ (Project Rules) ──┬──→ .agent/  (Antigravity)
                                       ├──→ .github/ (Copilot)
                                       └──→ ???      (Future: Windsurf, etc.)

[SOURCE]
Embedded "Compound V" (Methodology) ───→ .agent/  (Antigravity)
```

## Plan

### Phase 1: MIGRATE Superpowers Content (Compound V)

**Source:** existing `superpowers-*` files + `gemini-superpowers-antigravity` starting point.
**Modifications:** Apply ONLY the improvements agreed upon in `artifacts/compound-v/brainstorm.md`. Do NOT arbitrarily shorten or rewrite proven prompts.

**Batch 1: Skills (Parallel)**

1.  **Migrate Planning:** `superpowers-plan` → `compound-v/skills/compound-v-plan/SKILL.md`
    - _Direct Port:_ Keep all proven planning instructions.
2.  **Migrate Review:** `superpowers-review` → `compound-v/skills/compound-v-review/SKILL.md`
    - _Direct Port:_ Keep severity levels and review checklist.
3.  **Migrate TDD:** `superpowers-tdd` → `compound-v/skills/compound-v-tdd/SKILL.md`
    - _Direct Port:_ Keep Red/Green/Refactor discipline.
4.  **Migrate Debugging:** `superpowers-debug` → `compound-v/skills/compound-v-debug/SKILL.md`
    - _Direct Port:_ Keep systematic isolation strategy.
5.  **Migrate Brainstorming:** `superpowers-brainstorm` → `compound-v/skills/compound-v-brainstorm/SKILL.md`
    - _Direct Port:_ Keep the rigorous options analysis.
6.  **Create Parallel Skill:** `compound-v/skills/compound-v-parallel/SKILL.md` (New)
    - _New:_ Implements the parallel execution logic previously hidden in workflows.

**Batch 2: Workflows (Parallel)**

7.  **Migrate Brainstorm Workflow:** `superpowers-brainstorm.md` → `compound-v/workflows/brainstorm.md`
    - _Diff:_ Use native `write_to_file`. Artifacts to `.promptherder/artifacts/`.
8.  **Migrate Plan Workflow:** `superpowers-write-plan.md` → `compound-v/workflows/write-plan.md`
    - _Diff:_ Add `stack.md` (tables first, rules second) and `structure.md` generation steps as per `brainstorm.md`.
9.  **Migrate Execution Workflow:** `superpowers-execute-plan.md` → `compound-v/workflows/execute.md`
    - _Diff:_ Merge sequential + parallel logic. Use `// turbo-all`.
10. **Migrate Review Workflow:** `superpowers-review.md` → `compound-v/workflows/review.md`
    - _Diff:_ Native file writing.

**Batch 3: Rules (Parallel)**

11. **Create Core Rule:** `compound-v/rules/00-compound-v.md`
    - _Content:_ Binds the methodology together (Awareness of the pipeline).
12. **Create Browser Rule:** `compound-v/rules/browser.md`
    - _Content:_ Manual trigger (`@browser.md`) for browser testing.

### Phase 2: Cleanup Old Superpowers (Parallel)

13. Delete `.agent/skills/superpowers-*` (all 9 dirs + scripts).
14. Delete `.agent/workflows/superpowers-*.md` (all 8 files).
15. Delete `.agent/rules/00-promptherder.md`.

### Phase 3: Go Implementation (promptherder v2)

_Goal: Idempotent configuration management._

16. **Embed Source:** Create `embed.go` at repo root (`//go:embed compound-v`).
17. **Define Interfaces:**
    - Create `internal/app/target.go`: `Target` interface (`Name()`, `Install()`).
    - Ensure `Install` methods are distinct and idempotent (check content hash or atomic overwrite).
18. **Implement Targets:**
    - `CopilotTarget`: Syncs `.promptherder/agent/rules` → `.github/` (replaces old logic).
    - `AntigravityTarget`: Syncs `.promptherder/agent/` → `.agent/` (New).
    - `CompoundVTarget`: Extracts embedded `compound-v` → `.agent/` (New).
    - _Constraint:_ `Antigravity` and `CompoundV` targets must **skip** files listed in `manifest.Generated` (e.g., `stack.md`, `structure.md`) to avoid overwriting agent work.
19. **Manifest v2:** Update `internal/app/manifest.go` to track files per-target.
20. **CLI Router:** Update `cmd/promptherder/main.go`:
    - `promptherder` (idempotent copy all)
    - `promptherder <target>` (idempotent copy specific)
    - Flags: `--dry-run`, `-v`, `--version`. (No `--repo`, use cwd).

### Phase 4: Verification

21. **Unit Tests:** Update existing tests, add tests for new Targets and idempotency.
22. **Integration Test:**
    - Run `promptherder` in a fresh temp dir. Check file structure.
    - Run `promptherder` _again_. Check that nothing changed/broke (idempotency).
    - Run `promptherder compound-v`. Check only compound-v files update.

### Phase 5: Documentation & Polish

23. **README:** Credit `obra/superpowers` and `anthonylee991`. Document new flow.
24. **CONTRIBUTING:** Add guide for adding new Targets.
25. **Cleanup:** Remove old `artifacts/` folder.

## Dependency Graph

```
[Phase 1: Migrate Content] ──┐
[Phase 2: Cleanup Old] ──────┴──→ [Phase 3: Go Code] ──→ [Phase 4: Verify] ──→ [Phase 5: Docs]
```
