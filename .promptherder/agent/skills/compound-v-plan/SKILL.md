---
name: compound-v-plan
description: Autonomous planning with internal reasoning. Researches, evaluates ideas, presents a plan. Minimizes user round-trips. Use before making non-trivial changes.
---

# Planning Skill

**Announce at start:** "I'm using the planning skill to work through this."

## When to use this skill

- any multi-file change
- any change that impacts behavior, data, auth, billing, or production workflows
- any debugging that needs systematic isolation
- any design decision with multiple viable approaches

## Core pattern: think first, ask smart

Planning is **LLM-driven**, not turn-by-turn. Follow these phases:

### Phase 1: Establish the goal

Determine the **desired end result** — the single sentence that defines success.

- If the user stated a clear goal, use it.
- If the goal is ambiguous, ask ONE question: "What's the desired end result?" and STOP.
- Do NOT ask multiple clarifying questions. Infer what you can and note assumptions.

### Phase 2: Research (autonomous — no user interaction)

Do all of this **in parallel** (multiple `search_web` + `view_file` calls with `waitForPreviousTools: false`):

- `search_web` for best practices, alternatives, and pitfalls. Scope to versions in `stack.md`.
- `view_file_outline` on relevant project files.
- Read `.promptherder/future-tasks.md` if it exists — check if any deferred ideas are relevant.
- Read `.agent/rules/stack.md` and `.agent/rules/structure.md` if they exist.
- Read `.promptherder/hard-rules.md` if it exists — all rules must be respected.

### Phase 3: Think (autonomous — no user interaction)

For each approach you consider, evaluate:

- **Idea**: what is it?
- **Pros**: why it's good
- **Cons**: why it's risky or complex
- **Verdict**: accept / reject / ask (need user input)

Apply these filters yourself:

- **DRY** — Identify repeated logic across the plan. Extract shared patterns into reusable steps. If two steps do the same thing, merge them. Update logic in one place — changes reflect everywhere.
- **YAGNI** — Only plan features you actually need right now. Focus on current requirements, not hypothetical future ones. If a step exists "just in case," cut it.
- **Don't overengineer** — Focus on the simplest solution that solves the core problem. Ask: "What's the minimum viable version?" If a step adds complexity without clear necessity, skip it. Keep it lean, functional, and maintainable.
- Confirm the approach solves the actual problem, not a hypothetical one.
- Identify risks and verify they are manageable.
- Ensure rollback options exist.

**Reject bad ideas yourself.** Only surface ideas with verdict `ask` to the user.

#### Persist decisions

Write the full decisions table to `decisions.md`. The calling workflow determines the full path (typically `.promptherder/convos/<slug>/decisions.md`):

```markdown
# Decisions: <title>

| #   | Idea | Verdict  | Pros | Cons | Rationale |
| --- | ---- | -------- | ---- | ---- | --------- |
| 1   | ...  | accepted | ...  | ...  | ...       |
| 2   | ...  | rejected | ...  | ...  | ...       |
| 3   | ...  | ask      | ...  | ...  | ...       |
```

Update this file whenever decisions change (feedback, re-planning, etc.).

**Verdicts:**

- `accepted` — you're confident this is right. Include in plan.
- `rejected` — you've killed it. Visible when user says SHOW DECISIONS.
- `ask` — you genuinely can't decide without user input. Surface as a batch question.

### Phase 4: Present the plan

Present the plan in a **single response**:

```markdown
# Plan: <title>

> **Status**: draft

## Goal

<one sentence>

## Plan

### 1. Step name

- **Files:** `path/to/file`
- **Change:** what changes (1-2 bullets)
  - Test → Code → Verify order
- **Verify:** command to verify

## What this builds

### Happy path

1. [numbered walkthrough of user-facing flow after implementation]

### Filesystem tree

[tree showing files created/modified]

## Risks & mitigations

- Risk → mitigation

## Rollback plan

<how to undo>

## References

- Full plan: `.promptherder/convos/<slug>/plan.md`
- Decisions: `.promptherder/convos/<slug>/decisions.md`
```

#### Batch questions

After the plan, present **all** decisions you need input on in one block:

```
**I need your input on:**

1. <question> — I recommend (A) because ...
   - (A) option
   - (B) option
2. <question>
   - (A) ...
   - (B) ...
```

If the path is clear, skip this section.

#### Deferred ideas

If you identified ideas with future value, list them:

```
**Ideas I'd defer to future tasks:**
- <idea> — <brief rationale>
```

> Add these to `future-tasks.md`? `yes` / `no`

**Only append after the user confirms.**

## Anti-patterns (don't do these)

- ❌ Asking the user to ACCEPT/REJECT individual ideas one at a time
- ❌ Showing the decisions table before you've done your own thinking
- ❌ Asking clarifying questions you could answer by reading the code
- ❌ Multiple rounds of "does this look right?" on sections of the plan
- ❌ Deferring the goal statement to a later phase
- ❌ Silently adding deferred ideas without confirmation

## Good patterns

- ✅ Research first, then present a confident plan
- ✅ Batch all questions into one block at the end
- ✅ Offer SHOW DECISIONS audit on demand (not by default)
- ✅ Kill bad ideas yourself with rationale (visible in audit)
- ✅ State assumptions explicitly so the user can correct them
- ✅ Confirm deferred ideas with the user before adding to `future-tasks.md`
- ✅ Include "What this builds" section with happy path and filesystem tree

## Planning rules

- Steps should be **small** (2–10 minutes each).
- Every step must include **verification**.
- Each step must show **test → code → verify** order where applicable.
- If the plan changes the codebase structure, show a **before/after filesystem tree**.
- Prefer **incremental deliverables** (avoid "big bang" edits).
- Identify **rollback** and **risk controls** early.
- Group independent steps for **parallel execution** where possible.
- Never write to `stack.md` or `structure.md` without user approval.
- State the **goal** first. Everything else flows from it.

## Plan step format

```
### 1. Step name
- **Files:** `path/to/file.ext`, `...`
- **Change:** (1–2 bullets)
  - Test: what test to write first
  - Code: what to implement
- **Verify:** (exact commands or checks)
```
