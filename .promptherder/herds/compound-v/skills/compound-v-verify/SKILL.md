---
name: compound-v-verify
description: Mandatory checklist before claiming a task is done. Ensures verification, clean code, and accurate reporting. Use before saying "done" or "complete".
---

# Verification Before Completion

Before reporting a task as done, run through this checklist.

## When to use this skill

- before telling the user a task is complete
- before moving to the next plan step
- before writing a review or execution summary

## The checklist

1. **Requirements met?** Re-read the task description. Have you missed details or edge cases?
2. **Tests passing?** Run the full test suite, not just the tests you wrote.
3. **Code clean?** No commented-out code. No debug prints. No placeholder TODOs.
4. **Warnings?** Check for linter warnings, compiler warnings, or deprecation notices.
5. **Verification commands?** Run the exact verification commands from the plan step.
6. **Documentation?** If you changed behavior or APIs, update relevant docs.

## Statement of completion

When you announce completion, include:

- What you did (1-2 lines)
- How you verified it (exact commands + results)

Example: "Step 3 complete. Added auth middleware to `lib/auth/plug.ex`. Verified: `mix test test/auth/` — 8 tests, 0 failures."

## Never

- Say "done" without running verification commands
- Skip the checklist because "it's simple"
- Move to step N+1 if step N is not verified
