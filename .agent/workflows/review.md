---
description: Runs a review pass with severity levels (Blocker/Major/Minor/Nit). Supports check targeting and YOLO mode.
---

# Review

Invoke the `compound-v-review` skill and follow it exactly.

## Workflow-specific protocol

### Slug resolution

1. If the user provided a kebab-case slug (e.g. `/review fix-this`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.

### Check targeting

If the user specified a check short name (e.g. `/review security`, `/review edges`), pass it to the review skill. Run only that check.

If the user specified `YOLO` (e.g. `/review YOLO`), pass that mode to the review skill.

Both can be combined with a slug: `/review fix-this security`, `/review fix-this YOLO`.

### Persist (mandatory)

Write the review to `.promptherder/convos/<slug>/review.md`.
Confirm it exists by listing `.promptherder/convos/<slug>/`.

Do not implement changes in this workflow unless `YOLO` mode is active.
