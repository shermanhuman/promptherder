---
description: Structured brainstorm. Produces goal/constraints/risks/options/recommendation/acceptance criteria.
---

# Brainstorm

## Task

Brainstorm the task described by the user in their message above.

If the user's request is unclear, ask them to restate the task in one sentence and STOP.

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
