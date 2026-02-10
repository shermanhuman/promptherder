---
description: Runs a review pass with severity levels (Blocker/Major/Minor/Nit). Supports check targeting and YOLO mode.
---

// turbo-all

# Review

Invoke the `compound-v-review` skill and follow it exactly.

## Workflow-specific protocol

### Slug resolution

Invoke the `compound-v-persist` skill to resolve the target `<slug>` folder.

**Logic:**

- If invoked as part of `/execute`, reuse the execution slug.
- If standalone, default to the most recent conversation (the skill handles this).
- Only create a new slug if the user explicitly requests a fresh context.

### Check targeting

If the user specified a check short name (e.g. `/review security`, `/review edges`), pass it to the review skill. Run only that check.

If the user specified `YOLO` (e.g. `/review YOLO`), pass that mode to the review skill.

Both can be combined with a slug: `/review fix-this security`, `/review fix-this YOLO`.

### Persist (mandatory)

Write the review to `.promptherder/convos/<slug>/review-<description>.md`.

- Use the user's provided slug (e.g. `/review fix-auth` → `review-fix-auth.md`).
- If no slug provided, generate a short 1-3 word description (e.g. `review-security.md`, `review-wip.md`).

Confirm it exists by listing `.promptherder/convos/<slug>/`.

Do not implement changes in this workflow unless `YOLO` mode is active.
