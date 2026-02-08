---
description: Structured brainstorm. Produces goal/constraints/risks/options/recommendation/acceptance criteria.
---

# Brainstorm

## Task

Brainstorm the task described by the user in their message above.

If the user's request is unclear, ask them to restate the task in one sentence and STOP.

## Research phase (parallel)

Before brainstorming, gather context. Fire these in parallel:

1. `search_web` for current best practices and alternatives related to the task.
2. `search_web` for common pitfalls or known issues in the problem domain.
3. `view_file_outline` on relevant project files to understand what exists today.
4. Read `stack.md` and `structure.md` rules if they exist — scope searches to pinned versions.

## Output sections (use exactly)

## Goal

## Constraints

## Known context

## Risks

## Options (2–4)

## Recommendation

## Acceptance criteria

## Persist (mandatory)

After generating the brainstorm, write it to disk:

1. Use `write_to_file` to save the brainstorm markdown to `.promptherder/artifacts/brainstorm.md`.
2. Confirm it exists by listing `.promptherder/artifacts/`.

Do not implement changes in this workflow. Stop after persistence.
