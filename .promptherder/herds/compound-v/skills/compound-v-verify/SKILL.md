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

1. **Requirements** — Re-read the task description. Confirm all requirements are met and no details or edge cases were missed.
2. **Tests** — Run the full test suite, not just the tests you wrote. Confirm all pass.
3. **Code cleanliness** — Remove commented-out code, debug prints, and placeholder TODOs.
4. **Warnings** — Check for and resolve linter warnings, compiler warnings, and deprecation notices.
5. **Verification commands** — Run the exact verification commands from the plan step. Confirm expected output.
6. **Documentation** — Update relevant docs if you changed behavior, APIs, CLI flags, or configuration.

## Statement of completion

When you announce completion, include:

- What you did (1-2 lines)
- How you verified it (exact commands + results)

Example: "Step 3 complete. Added auth middleware to `lib/auth/plug.ex`. Verified: `mix test test/auth/` — 8 tests, 0 failures."

## Never

- Say "done" without running verification commands
- Skip the checklist because "it's simple"
- Move to step N+1 if step N is not verified
