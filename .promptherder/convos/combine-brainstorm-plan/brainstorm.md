# Brainstorm: Combine Brainstorm + Plan Workflows?

## Goal

Determine whether the separate `/brainstorm` and `/write-plan` workflows should remain distinct or be merged into a single `/plan` workflow, optimizing for context efficiency and practical usage patterns.

## Constraints

- Must work within LLM context windows — more workflow steps = more context consumed
- Must support the human-in-the-loop approval gate before execution
- Must remain useful for both "I need to think through options" and "I know what I want, just plan it"
- Must work across Antigravity and Copilot targets

## Known context

### What each workflow does today

**`/brainstorm`** (56 lines):

- Research phase (web search + file reading)
- Outputs: Goal, Constraints, Known context, Risks, Options (2-4), Recommendation, Acceptance criteria
- Explicitly says "do not implement"
- Purpose: **Divergent thinking** — explore the problem space

**`/write-plan`** (84 lines):

- Research phase (web search + file reading)
- Generates `stack.md` and `structure.md` rules
- Outputs: Goal, Assumptions, Plan (steps with files/change/verify), Risks & mitigations, Rollback plan
- Requires approval before execution
- Purpose: **Convergent thinking** — commit to a specific implementation path

### How they're actually used

Looking at this conversation as a case study:

1. `/brainstorm` ran → produced options analysis for artifact management
2. Multiple rounds of discussion refined the approach
3. `/write-plan` ran → produced implementation steps
4. Plan was reviewed, simplified, re-written
5. `/execute` ran → implemented

The brainstorm consumed significant context (~10K tokens of output) that then needed to be re-read or summarized when writing the plan. The plan itself duplicated some research (another web search + file reading phase).

### Context window research

Key finding: LLM performance degrades as context fills ("context rot"). Each workflow transition wastes context by:

- Re-reading the same files
- Re-doing overlapping web searches
- Restating context the agent already discovered
- The conversation history itself grows with each step

## Risks

- **Merging loses the "explore vs commit" distinction** — brainstorms are for when you don't know what to do; plans are for when you do
- **A combined workflow becomes too long** — 100+ lines of instructions the LLM must follow
- **Loss of the standalone brainstorm** — sometimes you just want to think, not plan
- **Approval gate timing** — in a combined flow, when does the human approve?

## Options (3)

### Option A: Keep separate (status quo)

Two distinct workflows, each with its own research phase and output format.

| Dimension          | Assessment                             |
| ------------------ | -------------------------------------- |
| Context efficiency | ❌ Duplicates research, re-reads files |
| Flexibility        | ✅ Can brainstorm without planning     |
| Complexity         | Low (each file is simple)              |
| Approval gate      | ✅ Clear — at end of plan              |

### Option B: Merge into single `/plan` workflow

One workflow that does it all: research → options → recommendation → plan → approval.

The output would be:

```
## Goal
## Constraints
## Options (if multiple approaches exist; skip if obvious)
## Chosen approach + rationale
## Plan (steps with files/change/verify)
## Risks & mitigations
## Rollback plan
```

| Dimension          | Assessment                                                     |
| ------------------ | -------------------------------------------------------------- |
| Context efficiency | ✅ Single research phase, no duplication                       |
| Flexibility        | ⚠️ Can't brainstorm without planning (but you can just... ask) |
| Complexity         | Medium (longer workflow file, ~80 lines)                       |
| Approval gate      | ✅ At end, same as today                                       |

### Option C: `/plan` with optional expand flag

One workflow, but if the user says `/plan` it goes straight to planning. If they say `/plan --explore` or the task is ambiguous, it does the options analysis first.

| Dimension          | Assessment                                         |
| ------------------ | -------------------------------------------------- |
| Context efficiency | ✅ Single research phase                           |
| Flexibility        | ✅ Supports both modes                             |
| Complexity         | ⚠️ Conditional logic in workflow (LLM must decide) |
| Approval gate      | ✅ At end                                          |

## Recommendation

**Option B: Merge into `/plan`.**

Here's why:

1. **The brainstorm-then-plan pipeline was always a single conversation anyway.** You don't brainstorm in one conversation and plan in a different one — they're the same thinking process. The only time you'd brainstorm without planning is when you're genuinely unsure and want to stop and think. But you can do that by just... asking the LLM to brainstorm. You don't need a formal workflow for "think about this."

2. **The options section is optional, not a separate workflow.** For obvious tasks ("add a test for X"), the plan can skip options. For ambiguous tasks ("should we restructure the auth module?"), the plan naturally includes an options analysis. The LLM is already smart enough to decide which — we don't need a flag.

3. **Context efficiency matters.** Every workflow transition wastes context re-reading files and re-doing searches. One research phase → one output is cleaner.

4. **Simpler mental model.** Users remember one command (`/plan`) not two (`/brainstorm` then `/write-plan`). The pipeline becomes: `/plan` → approve → `/execute` → `/review`.

The merged workflow would include a line like:

> _If the task has multiple viable approaches, include an Options section with 2-4 alternatives and your recommendation before writing the plan steps. If the approach is obvious, skip Options and go straight to the plan._

## Acceptance criteria

- [ ] Single `/plan` workflow replaces both `/brainstorm` and `/write-plan`
- [ ] Options analysis is included when the task is ambiguous, skipped when obvious
- [ ] Research phase runs once (not duplicated)
- [ ] Approval gate preserved at the end
- [ ] `stack.md` and `structure.md` rule generation preserved
- [ ] Old `/brainstorm` workflow removed from source and all targets
- [ ] Pipeline simplifies to: `/plan` → `/execute` → `/review`
