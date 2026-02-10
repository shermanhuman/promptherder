---
description: Autonomous planning workflow. Researches, reasons, and presents a plan with minimal back-and-forth. Respects the user's time.
---

// turbo-all

# Plan

Invoke the `compound-v-plan` skill and follow it exactly.

## Workflow-specific protocol

### Slug resolution

Invoke the `compound-v-persist` skill to resolve the target `<slug>` folder.

**Logic:**

- If updating an existing plan, target the existing folder.
- If creating a **New Plan**, the skill will generate a new `YYYY-MM-DD-slug`.
- If the user provided a specific slug, pass it to the skill.

### Artifacts

Write to `.promptherder/convos/<slug>/`:

- `plan.md` — the plan
- `decisions.md` — full decisions table (all ideas, including rejected)

### Response handling

After presenting the plan, always end with:

> Run `/execute <slug>` to proceed, `SHOW DECISIONS` to audit, `DECLINE` to reject, or give feedback.

_Task: `<slug>`_

**`/execute`** → The user running `/execute` IS the approval. The execute workflow sets status to `approved`. Do NOT implement.

**`SHOW DECISIONS`** → Print contents of `decisions.md`. Re-prompt.

**`DECLINE`** → Update status to `declined` in plan.md. Reply: "Plan declined. Task: `<slug>`". Stop.

**Feedback** → Incorporate, re-research if needed, update plan and decisions, re-present.

### Deferred ideas

List ideas with future value when presenting the plan.

> Add these to `future-tasks.md`? `yes` / `no`

Only append to `.promptherder/future-tasks.md` after user confirms.
