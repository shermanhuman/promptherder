---
description: Runs a review pass with severity levels (Blocker/Major/Minor/Nit).
---

# Review

Read and apply the `compound-v-review` skill.

Review all changes made in the current session (or the scope specified by the user).

## Research phase (parallel)

Before reviewing, gather context. Fire these in parallel:

1. Read all changed files using `view_file` (parallel calls).
2. `search_web` for latest best practices for the primary libraries/frameworks being used. Scope to versions in `stack.md` if it exists.
3. `search_web` for known issues or deprecations in the APIs or patterns used in the changes.

## Output

- Blockers
- Majors
- Minors
- Nits
- Summary + next actions

## Slug (resolve before persisting)

Determine a task slug for organizing artifacts:

1. If the user provided a kebab-case slug (e.g. `/review fix-this`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.

Write all artifacts to `.promptherder/convos/<slug>/`.

## Persist (mandatory)

Write the review to `.promptherder/convos/<slug>/review.md` using `write_to_file`.
Confirm it exists by listing `.promptherder/convos/<slug>/`.

Do not implement changes in this workflow. Stop after persistence.
