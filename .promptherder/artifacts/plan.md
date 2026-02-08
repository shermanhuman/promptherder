# Plan: Convo-Based Artifact Management

## Goal

Replace hardcoded `.promptherder/artifacts/<type>.md` paths in all four Compound V workflows with `convos/<slug>/<type>.md`. Keep it minimal — trust the LLM to pick a slug from context.

## Assumptions

- Source of truth is `compound-v/workflows/*.md`. After editing, `promptherder` reinstalls to `.agent/workflows/` and `.github/prompts/`.
- No Go code changes needed — this is workflow instructions only.
- No git commit automation — artifacts ride with normal code commits.
- No history index file — `convos/` directory is self-describing.
- Existing `artifacts/` files stay as-is (legacy).

## Plan

### Step 1: Update `brainstorm.md` workflow

- Files: `compound-v/workflows/brainstorm.md`
- Change:
  - Add slug instruction block before the persist section
  - Change persist path from `.promptherder/artifacts/brainstorm.md` → `.promptherder/convos/<slug>/brainstorm.md`
- Verify: Read file, confirm `convos/<slug>/` path and slug instructions present

### Step 2: Update `write-plan.md` workflow

- Files: `compound-v/workflows/write-plan.md`
- Change:
  - Add slug instruction block
  - Add: "If `convos/<slug>/brainstorm.md` exists, read it for context"
  - Change persist path → `.promptherder/convos/<slug>/plan.md`
  - Update approval message to include the slug
- Verify: Read file, confirm brainstorm reference, convo path, slug in approval message

### Step 3: Update `execute.md` workflow

- Files: `compound-v/workflows/execute.md`
- Change:
  - Add slug instruction block
  - Change precondition: plan must exist at `.promptherder/convos/<slug>/plan.md`
  - Change execution log path → `.promptherder/convos/<slug>/execution.md`
  - Change review output path → `.promptherder/convos/<slug>/review.md`
- Verify: Read file, confirm all three paths reference `convos/<slug>/`

### Step 4: Update `review.md` workflow

- Files: `compound-v/workflows/review.md`
- Change:
  - Add slug instruction block
  - Change persist path → `.promptherder/convos/<slug>/review.md`
- Verify: Read file, confirm path

### Step 5: Update `structure.md` rule

- Files: `.agent/rules/structure.md`
- Change: Update the folder layout tree — add `convos/` under `.promptherder/`, note that `artifacts/` is legacy
- Verify: Read file, confirm tree reflects new layout

### Step 6: Create `convos/` directory

- Files: `.promptherder/convos/.gitkeep`
- Change: Create empty `.gitkeep`
- Verify: `Test-Path .promptherder/convos/.gitkeep`

### Step 7: Run `promptherder` to reinstall

- Files: `.agent/workflows/*`, `.github/prompts/*`
- Change: `go run ./cmd/promptherder`
- Verify: Command succeeds, installed workflows match source

### Step 8: Smoke test

- Verify: `go test ./...` passes, `go vet ./...` passes

## The slug instruction block (used in steps 1-4)

This is the block that gets added to each workflow. Same wording in all four:

```markdown
## Slug (resolve before persisting)

Determine a task slug for organizing artifacts:

1. If the user provided a kebab-case slug (e.g. `/brainstorm fix-this`), use it.
2. If continuing a previous task, check `.promptherder/convos/` for a matching folder.
3. Otherwise, generate a short kebab-case name (2-4 words) from the task description.

Write all artifacts to `.promptherder/convos/<slug>/`.
```

## Risks & mitigations

| Risk                                     | Mitigation                                             |
| ---------------------------------------- | ------------------------------------------------------ |
| LLM picks a different slug than expected | Slug is visible in output; user can correct and re-run |
| `convos/` accumulates stale folders      | Manual cleanup; folders are small markdown files       |
| `/execute` can't find the right plan     | LLM lists `convos/` and asks which task to execute     |

## Rollback plan

`git checkout compound-v/workflows/` + re-run `promptherder`.
