---
description: Executes an approved plan. Parallel-by-default. Runs verification after each batch. Stops on failure.
---

// turbo-all

# Execute Plan

Invoke skills as needed during execution: `compound-v-tdd`, `compound-v-debug`, `compound-v-parallel`, `compound-v-verify`.

**Announce at start:** "Executing plan: `<slug>`. [1-2 line summary of the plan]."

## Workflow-specific protocol

### Slug resolution

1. If the user provided a kebab-case slug (e.g. `/execute fix-this`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.

### Preconditions (do not skip)

1. The plan must exist at `.promptherder/convos/<slug>/plan.md`.
2. Update status to `approved` in plan.md (running `/execute` IS the approval).

If the plan file does not exist, stop and tell the user to run `/plan` first.

### Context files (read before starting)

- `.promptherder/convos/<slug>/plan.md` — the approved plan.
- `.agent/rules/stack.md` — pinned versions and tech constraints.
- `.agent/rules/structure.md` — project layout and naming conventions.
- `.promptherder/hard-rules.md` — project-level rules that must always be followed.

### Execution loop

1. Analyze step dependencies (use `compound-v-parallel`). Group independent steps into batches.
2. **Before each batch**, `search_web` for latest docs on libraries/APIs about to be used. Scope to `stack.md` versions.
3. Fire all independent steps as concurrent tool calls.
4. After each batch: run verification, append results to `.promptherder/convos/<slug>/execution.md`.
5. If verification fails: stop, switch to `compound-v-debug`. Do not continue until fixed.
6. If the plan is wrong: stop, update the plan, ask for approval if the change is material.

### Finish (required)

**Normal mode:**

1. Run `compound-v-review`. Follow its output format and action menu — do not duplicate them here.
2. After fixes (if any), provide a manual smoke test:
   - List exact commands to test the happy path end-to-end
   - List edge cases worth testing manually
   - Show expected output for each command
3. Write summary to `.promptherder/convos/<slug>/review.md`.
4. Confirm artifacts exist by listing `.promptherder/convos/<slug>/`.

**`YOLO` mode:**

1. Run `compound-v-review` in YOLO mode. It handles auto-fixing.
2. Output summary: what was built, what was found, what was fixed.
3. Write summary to `.promptherder/convos/<slug>/review.md`.
4. Confirm artifacts.

Stop after completing the finish step.
