---
description: Executes an approved plan. Parallel-by-default. Runs verification after each batch. Stops on failure.
---

// turbo-all

# Execute Plan

## Slug (resolve before starting)

Determine a task slug for organizing artifacts:

1. If the user provided a kebab-case slug (e.g. `/execute fix-this`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.

Write all artifacts to `.promptherder/convos/<slug>/`.

## Preconditions (do not skip)

1. The user must have replied **APPROVED** to a written plan.
2. The approved plan must exist at `.promptherder/convos/<slug>/plan.md`.

If the plan file does not exist, stop and tell the user to run `/plan` first.

## Load the plan

- Read `.promptherder/convos/<slug>/plan.md`.
- Restate the plan briefly (1–2 lines) before making changes.

## Dependency analysis

After loading, analyze step dependencies:

1. Build dependency graph (file overlap → sequential).
2. Group independent steps into **parallel batches**.
3. Steps that read output of a previous step → sequential.

## Skills to apply as needed

Read and apply these skills when relevant:

- `compound-v-tdd` (preferred for implementation)
- `compound-v-debug` (if issues occur)
- `compound-v-review` (at the end)
- `compound-v-parallel` (for batch reasoning)
- `compound-v-verify` (before reporting each step complete)

## Context files (read before starting)

- `.agent/rules/stack.md` — pinned versions and tech constraints. Use these versions in all `search_web` queries.
- `.agent/rules/structure.md` — project layout and naming conventions. Follow when creating or moving files.

## Execution rules (strict)

1. **Before each batch**, `search_web` for the latest documentation of libraries/APIs you are about to use. Scope queries to the versions in `stack.md` (e.g., "Phoenix 1.8 LiveView streams" not just "Phoenix streams"). Run searches in parallel.
2. For each batch, fire all independent steps as concurrent tool calls.
3. After each batch:
   - Run verification commands.
   - Append a short note to `.promptherder/convos/<slug>/execution.md`:
     - Step name, files changed, what changed (1–3 bullets)
     - Verification command(s) and result (pass/fail)
4. If verification fails:
   - Stop. Switch to systematic debugging (use `compound-v-debug`).
   - Do not continue until fixed and verified.
5. Keep changes minimal and scoped to the plan. If the plan is wrong:
   - Stop, update the plan, ask for approval again if the change is material.

## Finish (required)

At the end:

1. Run a review pass (Blocker/Major/Minor/Nit).
2. Write a final summary to `.promptherder/convos/<slug>/review.md`:
   - Verification commands run + results
   - Summary of changes
   - Follow-ups (if any)
3. Confirm artifacts exist by listing `.promptherder/convos/<slug>/`.

Stop after completing the finish step.
