---
description: Runs a review pass with severity levels (Blocker/Major/Minor/Nit).
---

# Review

Invoke the `compound-v-review` skill and follow it exactly.

**Announce at start:** "Running review pass on [scope]."

## Workflow-specific protocol

### Slug resolution

1. If the user provided a kebab-case slug (e.g. `/review fix-this`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.

### Persist (mandatory)

Write the review to `.promptherder/convos/<slug>/review.md`.
Confirm it exists by listing `.promptherder/convos/<slug>/`.

Do not implement changes in this workflow. Stop after persistence.
