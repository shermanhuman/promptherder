# Brainstorm: Artifact Management Strategy

## Goal

Establish a robust strategy for managing agent-generated artifacts (plans, brainstorms, reviews, execution logs) that:

1. Supports **parallel agents** writing simultaneously without collision
2. Prevents **accidental overwrites** on re-runs
3. Provides **history** via git commits
4. Maintains **workflow handoff continuity** — the slug/context must survive transitions between `/brainstorm` → `/write-plan` → `/execute` → `/review`

## Constraints

- **Filesystem-based**: Must work with standard file operations (no database).
- **Git-centric**: User explicitly prefers "simple git commit" for history.
- **Cross-platform agents**: Must work for both **Antigravity** and **VS Code Copilot** targets.
- **Compound V Integration**: Must align with existing workflows.
- **Promptherder is the generator**: Workflows live in source at `compound-v/workflows/` and get installed to target-specific paths. Any changes must flow through the promptherder manifest system.

## Known Context

### Current Workflow Handoff Chain

| Workflow      | Reads                                             | Writes                                               |
| ------------- | ------------------------------------------------- | ---------------------------------------------------- |
| `/brainstorm` | _(nothing)_                                       | `.promptherder/artifacts/brainstorm.md`              |
| `/write-plan` | _(user message)_                                  | `.promptherder/artifacts/plan.md`                    |
| `/execute`    | `.promptherder/artifacts/plan.md` **← hardcoded** | `.promptherder/artifacts/execution.md` + `review.md` |
| `/review`     | _(changed files)_                                 | `.promptherder/artifacts/review.md`                  |

The **only hard file dependency** is `/execute` reading `plan.md`. But the _implicit_ dependency is that each workflow assumes the previous one wrote to the known path.

### Platform Research: What Each Agent Knows

**Antigravity (Gemini CLI / IDE)**:

- Provides `ADDITIONAL_METADATA` with: local time, active document, cursor position, open documents, browser pages
- Conversation IDs are UUIDs stored at `~/.gemini/antigravity/conversations/<uuid>.pb`
- **No conversation ID is injected into the agent's context** — the agent does not know its own conversation ID at runtime
- Conversation summaries are provided at conversation start (title + ID)
- Sessions can be resumed via `/resume` or `--resume` flag
- Skills/rules/workflows are discovered from `.agent/` directory

**VS Code Copilot**:

- `copilot-instructions.md` persists as a system-level instruction file across all sessions
- **No session/conversation ID exposed** to prompt files or custom instructions
- Chat sessions maintain history within a single session, but context does not persist between sessions
- Prompt files (`.prompt.md`) are stateless — they don't receive runtime metadata
- The Copilot coding agent (async) creates PRs as persistent artifacts

### Critical Finding

**Neither platform exposes a conversation ID to the agent at workflow runtime.** This means we cannot use conversation IDs as automatic namespaces — the workflow instructions have no way to say "write to `conversations/<my-conversation-id>/`" because they don't know the ID.

## Risks

- **Slug amnesia**: Agent forgets the slug between workflow transitions (new conversation, context window overflow, or simply not mentioned in the prompt)
- **Zombie folders**: Proliferation of `convos/<slug>` folders that are never cleaned up
- **Complexity creep**: Over-engineering the folder structure makes it hard for humans to find things
- **Tooling breakage**: Existing references to `artifacts/plan.md` must be updated everywhere

## Options (3)

### Option A: Slug-Based Folders + CURRENT_TASK Pointer

Structure: `.promptherder/convos/<slug>/plan.md`  
Pointer: `.promptherder/CURRENT_TASK` contains the active slug

**How workflow handoffs work**:

1. `/brainstorm name=fix-login` → sets `CURRENT_TASK=fix-login`, writes to `convos/fix-login/brainstorm.md`
2. `/write-plan` → reads `CURRENT_TASK`, writes to `convos/fix-login/plan.md`
3. `/execute` → reads `CURRENT_TASK`, reads `convos/fix-login/plan.md`

**Parallel agents**: Each agent gets `name=<slug>` in its invocation. Different slugs = different folders = no collision.

**Weakness**: `CURRENT_TASK` is a **shared mutable pointer** — two parallel agents race on it. Only works if parallel agents always pass explicit slugs (and never rely on `CURRENT_TASK` as default).

| Dimension          | Rating                                        |
| ------------------ | --------------------------------------------- |
| Parallel safety    | ⚠️ Partial (safe only if slugs always passed) |
| Handoff continuity | ✅ Via CURRENT_TASK pointer                   |
| History            | ✅ Git commit per artifact                    |
| Complexity         | Low-medium                                    |

### Option B: Slug-Based Folders + Footer Breadcrumb (No Shared Pointer)

Structure: `.promptherder/convos/<slug>/plan.md`  
**No** `CURRENT_TASK` file. Instead, every artifact ends with a footer:

```markdown
---

**Task**: `fix-login`  
**Artifacts**: `.promptherder/convos/fix-login/`
```

**How workflow handoffs work**:

1. `/brainstorm name=fix-login` → writes to `convos/fix-login/brainstorm.md` with footer
2. User has the brainstorm open in editor → Antigravity sees it in `ADDITIONAL_METADATA` → agent reads the footer → knows the slug
3. `/write-plan` → reads slug from open brainstorm file → writes to `convos/fix-login/plan.md`

**Parallel agents**: Fully safe — no shared mutable state. Each agent's context comes from the files open in its own session.

**Weakness**: Relies on the user having the relevant artifact file **open in their editor**. If they close it, the slug is lost. Also, **Copilot prompt files don't receive `ADDITIONAL_METADATA`** — they can't read open files.

| Dimension          | Rating                             |
| ------------------ | ---------------------------------- |
| Parallel safety    | ✅ Full (no shared state)          |
| Handoff continuity | ⚠️ Fragile (depends on open files) |
| History            | ✅ Git commit per artifact         |
| Complexity         | Low                                |

### Option C: Slug-Based Folders + CURRENT_TASK Pointer + Footer (Hybrid)

Structure: `.promptherder/convos/<slug>/plan.md`  
Pointer: `.promptherder/CURRENT_TASK` (convenience for sequential work)  
Footer: Every artifact ends with the slug breadcrumb

**How workflow handoffs work** (resolution order):

1. If `name=<slug>` is in the invocation → use that slug
2. Else if an artifact with a footer is open in the editor → read slug from there
3. Else if `CURRENT_TASK` exists → use that slug
4. Else → ask the user for a name

**Parallel agents**:

- Safe if each agent gets an explicit `name=<slug>`
- `CURRENT_TASK` is treated as a **hint**, not a lock — parallel agents should always specify slugs
- A `convos/<slug>/task.json` metadata file can store slug, creation time, and conversation mapping

**Git history approach**:

- Each workflow commits its artifact: `git add .promptherder/convos/<slug>/<file> && git commit -m "artifact: <slug>/<file>"`
- History of changes to the same file = `git log --follow` on that path

| Dimension          | Rating                                              |
| ------------------ | --------------------------------------------------- |
| Parallel safety    | ✅ Full (when slugs specified)                      |
| Handoff continuity | ✅ Triple fallback (explicit → open file → pointer) |
| History            | ✅ Git commit per artifact                          |
| Complexity         | Medium                                              |

## Recommendation

**Option C (Hybrid)** — it provides the strongest handoff continuity via the triple fallback, parallel safety via explicit slugs, and the `CURRENT_TASK` pointer as a convenience for the common sequential case (which is the majority of usage).

### Key design decisions:

1. **Folder**: `.promptherder/convos/<slug>/` — not `conversations/` (user's preference for shorter name)
2. **Metadata**: Each slug folder gets a `task.json`:
   ```json
   {
     "slug": "fix-login",
     "created": "2026-02-08T07:50:00Z",
     "description": "Fix the login bug in auth module"
   }
   ```
3. **Footer**: Every artifact ends with:
   ```markdown
   ---

   > **Task**: `fix-login` | **Artifacts**: `.promptherder/convos/fix-login/`
   ```
4. **CURRENT_TASK**: Simple text file with just the slug. Updated by each workflow. Treated as "last used" hint.
5. **Git commit**: Each workflow includes a step to `git add` + `git commit` the artifact with a conventional message.
6. **Slug resolution** (in workflow instructions):
   - Explicit `name=` parameter → wins
   - Open artifact file with footer → fallback
   - `CURRENT_TASK` → fallback
   - None found → prompt user

## Acceptance criteria

- [ ] `.promptherder/convos/<slug>/` folder structure defined and documented
- [ ] All four workflows updated with slug resolution logic and footer generation
- [ ] `CURRENT_TASK` pointer mechanism implemented in workflow instructions
- [ ] `task.json` metadata file created per slug folder
- [ ] Git commit step added to each workflow's persist section
- [ ] Parallel safety verified: two different slugs can coexist without collision
- [ ] Backward compatibility: workflows still work if user invokes without a slug (defaults to `CURRENT_TASK` or asks)
