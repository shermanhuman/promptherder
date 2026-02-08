---
name: compound-v-plan
description: Writes an implementation plan with small steps, exact files to touch, and verification commands. Use before making non-trivial changes.
---

# Planning Skill

## When to use this skill

- any multi-file change
- any change that impacts behavior, data, auth, billing, or production workflows
- any debugging that needs systematic isolation

## Research before planning

Before writing the plan, search for context in parallel:

- `search_web` for latest docs of libraries/APIs being used (scope to versions in `stack.md`).
- `search_web` for best practices or migration guides relevant to the task.
- Read existing project files to understand current structure.

## Planning rules

- Steps should be **small** (2–10 minutes each).
- Every step must include **verification**.
- Prefer **incremental deliverables** (avoid "big bang" edits).
- Identify **rollback** and **risk controls** early.
- Group independent steps for **parallel execution** where possible.

## Plan format (use this exact structure)

### Goal

### Assumptions

### Plan

1. Step name
   - Files: `path/to/file.ext`, `...`
   - Change: (1–2 bullets)
   - Verify: (exact commands or checks)
2. ...

### Risks & mitigations

### Rollback plan
