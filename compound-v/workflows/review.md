---
description: Runs a review pass with severity levels (Blocker/Major/Minor/Nit).
---

# Review

Read and apply the `compound-v-review` skill.

Review all changes made in the current session (or the scope specified by the user).

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
