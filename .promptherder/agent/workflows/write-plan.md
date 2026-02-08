---
description: Plan gate. Writes a small-step plan with files + verification. Generates stack.md and structure.md rules. Must ask for approval before coding.
---

# Write Plan

## Task

Plan the task described by the user in their message above.

If the user's request is unclear, ask them to restate the task in one sentence and STOP.

## Rules

- DO NOT edit code.
- You may read files to understand context, but produce the plan and then stop.
- Plan steps must be small (2–10 minutes each) and include verification commands.

## Research phase (parallel)

Before writing the plan, gather context in parallel:

1. `search_web` for latest stable versions of each technology in the project.
2. `view_file_outline` on key project files to understand current structure.
3. Read existing `.promptherder/agent/rules/` if they exist.

## Generate rules

After research, generate two always-on rules:

### `.agent/rules/stack.md`

- **Tables first**: version table (pin major.minor, let patch float)
- **Then bullet rules**: key constraints and "never do" items

### `.agent/rules/structure.md`

- Folder layout tree
- Where things belong (components, contexts, tests, assets)
- Naming conventions
- DRY location hints

Write both rules using `write_to_file`.

## Output format (use exactly)

## Goal

## Assumptions

## Plan

(Each step must include: Files, Change, Verify)

## Risks & mitigations

## Rollback plan

## Persist (mandatory)

Write the plan to `.promptherder/artifacts/plan.md` using `write_to_file`.
Confirm it exists by listing `.promptherder/artifacts/`.

## Approval

Ask: **Approve this plan? Reply APPROVED if it looks good.**

If the user replies APPROVED:

- Do NOT implement yet.
- Reply: **"Plan approved. Run `/execute` to begin implementation."**
