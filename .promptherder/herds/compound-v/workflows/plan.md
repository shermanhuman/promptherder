---
description: Interactive planning workflow. Decomposes tasks into ideas, investigates, iterates with user, drafts plan.
---

// turbo-all

# Plan

## Task

Plan the task described by the user in their message above.

If the user's request is unclear, ask them to restate the task in one sentence and STOP.

## Rules

- DO NOT edit code during this workflow.
- You may read files to understand context.
- Never write to `stack.md` or `structure.md` without showing proposed changes and getting explicit user approval.
- **One question at a time.** If you need clarification, ask one focused question per message.
- **Multiple choice preferred.** When asking the user to decide, present numbered options with your recommendation.
- **YAGNI ruthlessly.** Remove unnecessary features and scope creep from all designs.

## Init

1. Resolve a task slug:
   - If the user provided a kebab-case slug (e.g. `/plan fix-auth`), use it.
   - If continuing a previous task, check `.promptherder/convos/` for a matching folder.
   - Otherwise, generate a short kebab-case name (2-4 words) from the task description.

2. Check if `.promptherder/convos/<slug>/plan.md` already exists. If so, read it and resume from current state.

3. Read `.agent/rules/stack.md` and `.agent/rules/structure.md` if they exist. Use pinned versions to scope web searches.

4. Research in parallel:
   - `search_web` for best practices, alternatives, and pitfalls related to the task.
   - `view_file_outline` on relevant project files to understand what exists today.

5. Create `.promptherder/convos/<slug>/plan.md` with:
   - Header line: `# Plan: <title>`
   - Status line: `> /plan · Status: **planning** · .promptherder/convos/<slug>/plan.md`
   - Empty ideas table
   - Decompose the user's request into initial ideas (state: `proposed`)

## Ideation loop

Maintain the ideas table across every message. For each user message:

- Process any ACCEPT/REJECT/DEFER decisions (update table, add rationale)
- For accepted ideas, add or update plan steps (each step: Files, Change, Verify)
- Break any new input into discrete ideas (state: `proposed`)
- Investigate proposed ideas: search web, apply first principles, build user stories / mock interfaces / diagram workflows as appropriate
- Challenge each idea: does it solve the actual problem? Is it the simplest approach? Apply YAGNI.
- Persist `plan.md` to disk immediately after every update
- Present the full ideas table at the end of every response
- If proposed ideas remain: ask the user to ACCEPT / REJECT / DEFER by number
- If no proposed ideas remain: ask "All ideas resolved. Type **DRAFT** to finalize the plan."

## Draft

When the user types DRAFT:

1. Review the full plan as a senior engineer.
2. Rewrite the plan cleanly:
   - Accepted ideas → concrete plan steps (Files, Change, Verify per step)
   - Rejected ideas → removed entirely
   - Deferred ideas → listed in a Deferred section
   - Add: Goal, Risks & mitigations, Rollback plan
3. Propose `stack.md` and `structure.md` changes if needed. Show the diff and wait for approval before writing.
4. Present the plan to the user in sections (200-300 words each), checking after each: "Does this section look right?"
   - Plan summary and goal
   - Happy-path user story
   - Interface mocks or logic diagrams (if applicable)
5. Persist `plan.md` with status `draft`.

## Approval

Ask: **Approve this plan? Reply APPROVED. Task: `<slug>`**

If the user replies APPROVED:

- Update status to `approved`.
- Persist `plan.md`.
- Do NOT implement.
- Reply: **"Plan approved. Run `/execute` to begin implementation. Task: `<slug>`"**

## Plan artifact format (use this structure)

```markdown
# Plan: <title>

> `/plan` · **Status**: planning · `.promptherder/convos/<slug>/plan.md`

## Ideas

| #   | Idea | State    | Rationale |
| --- | ---- | -------- | --------- |
| 1   | ...  | proposed | —         |

## Goal

(filled during DRAFT)

## Plan

1. Step name
   - Files: `path/to/file`
   - Change: what changes
   - Verify: command to verify

## Risks & mitigations

(filled during DRAFT)

## Rollback plan

(filled during DRAFT)

## Deferred

(ideas with state: deferred)
```
