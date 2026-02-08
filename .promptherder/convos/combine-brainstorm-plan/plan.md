# Plan: Merge Brainstorm + Write-Plan into Interactive `/plan` Workflow

> `/plan` · **Status**: planning · `.promptherder/convos/combine-brainstorm-plan/plan.md`

## Goal

Replace `/brainstorm` and `/write-plan` with a single `/plan` workflow that uses an interactive ideation loop: decompose → investigate → present → user decides → iterate → draft → approve.

## Assumptions

- Source of truth is `compound-v/` — after editing, `promptherder` reinstalls to all targets.
- No Go code changes — this is workflow and skill markdown only.
- The workflow file instructs the LLM to maintain loop state via the `plan.md` artifact (ideas table + status).
- The LLM can maintain loop behavior across multiple messages within a single conversation.

## Plan

### Phase 1: Write the new `/plan` workflow (the core work)

**Step 1: Create `compound-v/workflows/plan.md`**

- Files: `compound-v/workflows/plan.md` (new)
- Change: Write the full interactive planning workflow with these sections:

  **1. Init section:**
  - Resolve slug (kebab-case from user or auto-generated)
  - Check if `convos/<slug>/plan.md` exists → load and resume
  - Read `stack.md` and `structure.md` if they exist
  - Research: parallel web searches + file reading
  - Decompose user's initial request into discrete ideas
  - Create `plan.md` artifact with header, empty ideas table, initial ideas as `proposed`

  **2. Ideation loop section** (~15 lines, not a state machine):

  Maintain the ideas table across every message. For each user message:
  - Process any ACCEPT/REJECT/DEFER decisions (update table, add rationale)
  - For accepted ideas, add or update plan steps
  - Break any new input into discrete ideas (state: proposed)
  - Investigate proposed ideas: search web, apply first principles, build user stories / mock interfaces / diagram workflows as appropriate
  - Persist plan.md to disk immediately
  - Present the full ideas table
  - If proposed ideas remain: ask the user to ACCEPT/REJECT by number
  - If no proposed ideas: ask "Type DRAFT to finalize the plan"

  **3. Draft section:**
  - Senior engineer review of the plan
  - Rewrite: accepted → concrete steps, rejected → removed, deferred → listed
  - **Propose** `stack.md` and `structure.md` changes — show diff to user, only write after explicit approval
  - Present: summary, happy-path user story, mocks, diagrams
  - Persist with status `draft`

  **4. Approval section:**
  - User types APPROVED → "Run `/execute` to begin implementation."
  - Status updates to `approved`

- Verify: Read the file, confirm all 4 sections are present, loop order matches spec

### Phase 2: Clean up old workflows and skills

**Step 2: Delete `compound-v/workflows/brainstorm.md`**

- Files: `compound-v/workflows/brainstorm.md` (delete)
- Change: Remove the file
- Verify: `Test-Path compound-v/workflows/brainstorm.md` returns False

**Step 3: Delete `compound-v/workflows/write-plan.md`**

- Files: `compound-v/workflows/write-plan.md` (delete)
- Change: Remove the file
- Verify: `Test-Path compound-v/workflows/write-plan.md` returns False

**Step 4: Delete `compound-v/skills/compound-v-brainstorm/SKILL.md`**

- Files: `compound-v/skills/compound-v-brainstorm/SKILL.md` (delete)
- Change: Remove the file and directory
- Verify: `Test-Path compound-v/skills/compound-v-brainstorm` returns False

**Step 5: Update `compound-v/skills/compound-v-plan/SKILL.md`**

- Files: `compound-v/skills/compound-v-plan/SKILL.md`
- Change: Rewrite to describe the ideation loop pattern. Key content:
  - When to use: any multi-file change, any design decision, any debugging
  - Ideas table format and states (proposed/accepted/rejected/deferred)
  - Planning rules: small steps, verification, incremental delivery
  - Reinforce: always present ideas table, always ask ACCEPT/REJECT/DEFER
  - Plan step format: Files, Change, Verify
- Verify: Read the file, confirm ideation loop described

**Step 6: Update `compound-v/rules/compound-v.md`**

- Files: `compound-v/rules/compound-v.md`
- Change:
  - Pipeline: remove `/brainstorm` and `/write-plan`, add `/plan`
  - Skills: remove `compound-v-brainstorm`
  - Pipeline becomes: `1. /plan → 2. /execute → 3. /review`
- Verify: Read the file, confirm pipeline has 3 steps, no brainstorm references

### Phase 3: Reinstall and verify

**Step 7: Run promptherder**

- Files: All installed locations
- Change: `go run ./cmd/promptherder`
- Verify:
  - Command succeeds
  - `Test-Path .agent/workflows/plan.md` returns True
  - `Test-Path .agent/workflows/brainstorm.md` returns False
  - `Test-Path .agent/workflows/write-plan.md` returns False
  - `Test-Path .agent/skills/compound-v-brainstorm` returns False

**Step 8: Verify installed content**

- Files: None (read-only)
- Verify:
  - `Select-String "convos/<slug>" .agent/workflows/plan.md` has matches
  - `Select-String "ACCEPT" .agent/workflows/plan.md` has matches
  - `Select-String "brainstorm" .agent/rules/compound-v.md` has NO matches
  - `.github/prompts/plan.prompt.md` exists
  - `.github/prompts/brainstorm.prompt.md` does NOT exist
  - `.github/prompts/write-plan.prompt.md` does NOT exist
  - `go test ./...` passes
  - `go vet ./...` passes

## Risks & mitigations

| Risk                                            | Mitigation                                                                                           |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| Workflow file too long (100+ lines)             | Keep instructions concise — the loop is a numbered list, not prose                                   |
| LLM doesn't maintain loop across messages       | The ideas table in plan.md IS the state — LLM reads it each cycle and sees what's proposed/accepted  |
| Old workflow references in user memory files    | compound-v.md rule is the primary reference; update it and the user's `.gemini` memory will catch up |
| Copilot prompt files can't do interactive loops | Copilot operates per-message anyway; the workflow instructions still apply per invocation            |
| Stale installed files not cleaned up            | Promptherder's cleanStale mechanism handles files in the manifest that are no longer generated       |

## Rollback plan

```
git checkout compound-v/
go run ./cmd/promptherder
```

Restores all workflows, skills, and rules to pre-change state.
