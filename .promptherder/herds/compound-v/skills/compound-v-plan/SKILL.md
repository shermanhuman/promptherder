---
name: compound-v-plan
description: Interactive planning with ideation loop. Maintains ideas table, investigates proposals, iterates with user decisions. Use before making non-trivial changes.
---

# Planning Skill

## When to use this skill

- any multi-file change
- any change that impacts behavior, data, auth, billing, or production workflows
- any debugging that needs systematic isolation
- any design decision with multiple viable approaches

## Core pattern: ideation loop

Planning is interactive, not one-shot. The LLM maintains an **ideas table** across messages:

| #   | Idea                   | State    | Rationale                  |
| --- | ---------------------- | -------- | -------------------------- |
| 1   | Use middleware pattern | accepted | Testable, clean separation |
| 2   | Use hooks instead      | rejected | Too magical                |

**States**: `proposed` → `accepted` / `rejected` / `deferred`

Every response should:

1. Process user decisions (ACCEPT/REJECT/DEFER by number)
2. Update the ideas table and plan steps
3. Investigate any new proposed ideas
4. Present the full table and ask for decisions

## Investigation techniques

When investigating proposed ideas:

- `search_web` for patterns, pitfalls, best practices (scope to versions in `stack.md`)
- Apply first-principles thinking — does this solve the actual problem?
- Challenge with YAGNI — is this the simplest approach? Remove unnecessary complexity.
- Build user stories for UX-facing ideas
- Mock interfaces for visual ideas
- Diagram workflows for process ideas

## Conversation principles

- **One question at a time.** Don't overwhelm the user — ask one focused question per message.
- **Multiple choice preferred.** Present numbered options with your recommendation.
- **YAGNI ruthlessly.** Remove unnecessary features from all designs.

## Planning rules

- Steps should be **small** (2–10 minutes each).
- Every step must include **verification**.
- Prefer **incremental deliverables** (avoid "big bang" edits).
- Identify **rollback** and **risk controls** early.
- Group independent steps for **parallel execution** where possible.
- Never write to `stack.md` or `structure.md` without user approval.

## Plan step format

```
1. Step name
   - Files: `path/to/file.ext`, `...`
   - Change: (1–2 bullets)
   - Verify: (exact commands or checks)
```
