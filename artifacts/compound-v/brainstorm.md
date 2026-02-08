# Brainstorm — Compound V: Agent Workflow Prompts for Antigravity

## Goal

Rebrand "superpowers" as **Compound V** (`compound-v`), migrate prompts into the `promptherder` repo, refactor for Antigravity's native tool model, strip the `superpowers-` prefix for faster `/` autocomplete, make everything shorter and tighter, and kill the Python scripts.

## Constraints

- Must work in **Antigravity only** (not Gemini CLI).
- No Python scripts — use native tools (`write_to_file`, `multi_replace_file_content`, parallel `run_command`).
- Workflows triggered via `/slash-commands` — naming must be terse for autocomplete.
- Skills are agent-triggered (auto-discovered by name+description match).
- Antigravity's `waitForPreviousTools` parameter controls parallel vs sequential tool calls.
- Artifacts go to `.promptherder/artifacts/`.
- Prompts ≤40 lines. Skills ≤35 lines. Workflows ≤50 lines.
- Parallelism is the **default** execution strategy.

## Known context

1. **Current inventory** (9 skills, 8 workflows) — all designed for Gemini CLI, not Antigravity.
2. **Python scripts** (3) — all dead weight. `write_artifact.py` → `write_to_file`. `spawn_subagent.py` → native parallel tool calls. `record_activation.py` → unused.
3. **Antigravity native parallelism:** `waitForPreviousTools: false` (default) fires concurrent tool calls.
4. **`// turbo` / `// turbo-all`** annotations auto-run `run_command` steps without user approval.
5. **Rule activation modes:** always-on, manual (@mention), model-decision, glob-pattern.
6. **12,000 char limit per rule file.** Skills have no such limit but should stay short.
7. **Reload is unnecessary** — Antigravity loads skills/workflows fresh each conversation.
8. **Persistent context:** Knowledge Items (KIs) carry distilled knowledge between conversations.
9. **browser_subagent:** Native tool for UI testing — click, type, screenshot, record video.
10. **Tech stack rules are currently hand-written** (e.g., `phx-decapcms.md` at 9KB). These should be generated during planning.

## Key Design Insight: Planning Generates Rules

The `/write-plan` workflow should not just produce a plan — it should **generate project rules** as artifacts:

### Stack Rule (`.agent/rules/stack.md`)

Generated during planning:

1. Research latest stable versions of chosen technologies via `search_web`
2. Pin to **major.minor**, let patch float
3. Write a compact version table + key constraints
4. This rule is always-on — agent sees it every conversation

### Structure Rule (`.agent/rules/structure.md`)

Generated during planning:

1. Design folder layout based on the tech stack and project scope
2. Define where things belong (separation of concerns)
3. Naming conventions, module organization
4. DRY hints ("shared components go here", "context modules go here")
5. This rule is always-on — prevents the agent from scattering code

### Why this matters

- **Today:** You hand-write 300-line rule files and burn 9KB of the 12KB rule budget on every conversation, even when irrelevant.
- **With Compound V:** Planning generates focused, compact rules (~50 lines each). Framework-specific reference patterns become skills (loaded on-demand). You save context window budget and get more precise agent behavior.

## Risks

| Risk                                      | Impact | Mitigation                                                  |
| ----------------------------------------- | ------ | ----------------------------------------------------------- |
| Too-terse prompts lose guardrails         | High   | Trim filler, keep structural constraints. Test each prompt. |
| Parallel-by-default causes file conflicts | Medium | Agent checks file overlap before parallelizing.             |
| Generated rules drift from reality        | Medium | `/review` workflow validates rules still match codebase.    |
| Skills not reliably activated             | Medium | Write specific descriptions with action verbs + keywords.   |
| Stack versions outdated                   | Low    | `/write-plan` always researches latest before pinning.      |

## Options

### Option A: Minimal port — rebrand and strip prefix only

- **Pro:** Fast. **Con:** Doesn't fix broken execution or add value.

### Option B: Clean rewrite — everything from scratch

- **Pro:** Cleanest. **Con:** Most work, may lose proven guardrails.

### Option C: Hybrid — port tight skills, rewrite workflows, add rule generation

- Port skills as-is (already tight at 30-36 lines)
- Rewrite workflows for Antigravity (parallel-by-default execution)
- Add rule generation to `/write-plan` (stack + structure)
- Add browser verification skill
- Kill dead weight
- **Pro:** Best effort-to-quality ratio.

## Recommendation

**Option C: Hybrid port with rule generation.**

### Final Inventory

**Skills** (agent-triggered, `compound-v-` prefix):

| Name                    | Lines | Notes                                                  |
| ----------------------- | ----- | ------------------------------------------------------ |
| `compound-v-plan`       | ~30   | Planning methodology — small steps, verification       |
| `compound-v-review`     | ~34   | Review checklist — Blocker/Major/Minor/Nit             |
| `compound-v-tdd`        | ~31   | Red/green/refactor discipline                          |
| `compound-v-debug`      | ~36   | Systematic debugging: reproduce, isolate, fix, prevent |
| `compound-v-brainstorm` | ~25   | Structured brainstorm template                         |
| `compound-v-parallel`   | ~25   | Teaches `waitForPreviousTools` batch reasoning         |

**Manual rules** (user-activated via `@mention`):

| Name         | Trigger       | Notes                                                |
| ------------ | ------------- | ---------------------------------------------------- |
| `browser.md` | `@browser.md` | Tells agent to use `browser_subagent` for UI testing |

**Workflows** (user `/commands`, bare names):

| Name            | Slash Command | Lines | Notes                                         |
| --------------- | ------------- | ----- | --------------------------------------------- |
| `brainstorm.md` | `/brainstorm` | ~25   | Structured brainstorm → persist to artifacts  |
| `write-plan.md` | `/write-plan` | ~45   | Plan + research stack + generate rules        |
| `execute.md`    | `/execute`    | ~45   | Parallel-by-default execution, `// turbo-all` |
| `review.md`     | `/review`     | ~20   | Review pass → persist findings                |

**Generated artifacts** (from `/write-plan`):

| Artifact       | Location                          | Purpose                                       |
| -------------- | --------------------------------- | --------------------------------------------- |
| `plan.md`      | `.promptherder/artifacts/plan.md` | Implementation plan                           |
| `stack.md`     | `.agent/rules/stack.md`           | Version-pinned tech stack (always-on rule)    |
| `structure.md` | `.agent/rules/structure.md`       | Folder layout + organization (always-on rule) |

**Deleted (7 items):**

- `superpowers-python-automation` skill — too generic
- `superpowers-rest-automation` skill — too generic
- `superpowers-finish` skill — folded into `/execute`
- `superpowers-workflow` skill + all Python scripts
- `superpowers-execute-plan-parallel` workflow — merged into `/execute`
- `superpowers-reload` workflow — unnecessary
- `superpowers-finish` / `superpowers-debug` workflows — skills sufficient

### `/write-plan` Workflow Design (with rule generation)

```
1. Parse the user's task description
2. Research: search_web for latest stable versions of relevant technologies
3. Generate .agent/rules/stack.md:
   - **Tables first**: version table (major.minor pinned, patch floats)
   - **Then bullet rules**: key constraints and "never do" items
   - ≤50 lines
4. Generate .agent/rules/structure.md:
   - Folder layout tree
   - Where things belong (components, contexts, tests, assets)
   - Naming conventions
   - DRY location hints
   - ≤50 lines
5. Write the implementation plan to .promptherder/artifacts/plan.md:
   - Small steps (2-10 min each)
   - Files per step
   - Verification commands
   - Dependency analysis for parallel execution
6. Ask for APPROVED before proceeding
```

### `/execute` Workflow Design (parallel-by-default)

```
// turbo-all

1. Load .promptherder/artifacts/plan.md
2. Analyze step dependencies:
   - Build dependency graph (file overlap → sequential)
   - Group independent steps into parallel batches
3. For each batch:
   - Fire all steps as concurrent tool calls (waitForPreviousTools: false)
   - After batch: run verification commands
   - Checkpoint to .promptherder/artifacts/execution.md
   - If any fail: stop, use compound-v-debug skill
4. After all batches:
   - Run compound-v-review checklist
   - Write summary to .promptherder/artifacts/review.md
   - List changed files + verification results
```

### Browser Testing Rule (`@browser.md`)

A **manual rule** (not a skill) — activated by typing `@browser.md` in chat.
Skills are agent-triggered and can't be manually invoked. Rules support manual activation via `@mention`.

```markdown
---
trigger: manual
---

# Browser Testing

Use `browser_subagent` to test the behavior described by the user.
Reproduce the issue, verify the fix, and report what you see.
```

3 lines. User types `@browser.md` + describes the bug → agent fires up browser subagent.
Not prescriptive — the agent already knows how to click, type, screenshot.

### Stack Rule Format (`stack.md`)

Generated by `/write-plan`. Tables first for quick reference, then bullet rules:

```markdown
---
trigger: always_on
---

# Tech Stack

| Component | Version | Component | Version |
| --------- | ------- | --------- | ------- |
| Phoenix   | 1.8.x   | LiveView  | 1.1.x   |
| Tailwind  | 4.1.x   | DaisyUI   | 5.0.x   |
| Ecto      | 3.13.x  | Postgres  | 15+     |

## Rules

- Pin major.minor, let patch float
- Use DaisyUI semantic colors (`primary`, `success`, `base-*`)
- Never use `@apply` in templates
- Use `tailwind` Hex package, not npm
```

### Artifact Directory Structure

```
.promptherder/
├── manifest.json              # promptherder file ownership
└── artifacts/
    ├── brainstorm.md          # /brainstorm output
    ├── plan.md                # /write-plan output
    ├── execution.md           # /execute step log
    └── review.md              # /review output

.agent/
├── rules/
│   ├── 00-compound-v.md       # Core methodology (always-on, ~10 lines)
│   ├── browser.md             # Manual: @browser.md for UI testing
│   ├── stack.md               # Generated: version-pinned tech stack (tables + rules)
│   └── structure.md           # Generated: folder layout + org rules
├── skills/
│   └── compound-v-*/SKILL.md  # 6 skills
└── workflows/
    ├── brainstorm.md           # 4 workflows
    ├── write-plan.md
    ├── execute.md
    └── review.md
```

## Acceptance Criteria

1. All skills/workflows live in `promptherder/.agent/`.
2. Brand is "Compound V" / `compound-v` throughout.
3. No `superpowers` prefix in any filename.
4. No Python scripts.
5. Workflows use bare names: `/brainstorm`, `/write-plan`, `/execute`, `/review`.
6. Skills use `compound-v-` prefix for discovery.
7. `/execute` is parallel-by-default via native `waitForPreviousTools`.
8. `/execute` annotated with `// turbo-all`.
9. `/write-plan` generates `stack.md` and `structure.md` rules.
10. Stack versions researched via `search_web`, pinned major.minor, patch floats.
11. Structure rule includes folder layout, naming conventions, DRY hints.
12. `browser.md` manual rule activates browser_subagent via `@browser.md`.
13. Artifacts in `.promptherder/artifacts/`.
14. All prompts ≤50 lines.
15. `reload` and `execute-plan-parallel` workflows deleted.
16. A minimal always-on rule (`00-compound-v.md`, ~10 lines) ensures agent awareness of the pipeline.
