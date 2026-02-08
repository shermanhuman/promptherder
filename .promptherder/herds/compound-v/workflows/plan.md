---
description: Autonomous planning workflow. Researches, reasons, and presents a plan with minimal back-and-forth. Respects the user's time.
---

// turbo-all

# Plan

Invoke the `compound-v-plan` skill and follow it exactly.

## Workflow-specific protocol

### Slug resolution

1. If the user provided a kebab-case slug (e.g. `/plan fix-auth`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.
4. Check if `.promptherder/convos/<slug>/plan.md` already exists. If so, read it and `decisions.md` if present, then resume from the current state (don't start over).

### Artifacts

Write to `.promptherder/convos/<slug>/`:

- `plan.md` — the plan
- `decisions.md` — full decisions table (all ideas, including rejected)

### Response handling

After presenting the plan, always end with:

> **Reply APPROVED to proceed, SHOW DECISIONS to audit my reasoning, or give feedback.**
> Task: `<slug>`

**APPROVED** → Update status to `approved` in plan.md. Reply: "Plan approved. Run `/execute` to begin. Task: `<slug>`". Do NOT implement.

**SHOW DECISIONS** → Print contents of `decisions.md`. Re-prompt for approval.

**Feedback** → Incorporate, re-research if needed, update plan and decisions, re-present.

### Deferred ideas

List ideas with future value when presenting the plan. Ask: "Should I add these to `future-tasks.md`?" Only append to `.promptherder/future-tasks.md` after user confirms.
