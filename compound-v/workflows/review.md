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

## Persist (mandatory)

Write the review to `.promptherder/artifacts/review.md` using `write_to_file`.
Confirm it exists by listing `.promptherder/artifacts/`.

Do not implement changes in this workflow. Stop after persistence.
