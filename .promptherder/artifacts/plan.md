# Plan: Naming & Namespace Cleanup

## Goal

Clean up technical debt identified in the review, focusing on obtuse naming (`SKILL.md`, `sync.go`) and repo clutter (`coverage*`, `artifacts/`).

## Assumptions

- Antigravity/Gemini CLI **requires** `SKILL.md` filename and subdirectory structure (platform convention).
- `promptherder.exe` was already gitignored and not tracked.

## Plan

### Batch 1: Quick Cleanup (Zero Risk)

1.  **Untrack Binary**
    - [ ] `git rm --cached promptherder.exe`
    - [ ] Verify: `git status` shows deleted from index.

2.  **Ignored Files**
    - [ ] Add `coverage*` to `.gitignore`.
    - [ ] Add `promptherder.exe` to `.gitignore` (if not present).
    - [ ] Verify: `git check-ignore coverage` returns true.

### Batch 2: Artifact Consolidation

3.  **Move Coverage Files**
    - [ ] `mkdir -p .promptherder/artifacts/coverage`
    - [ ] Move `coverage*` files to `.promptherder/artifacts/coverage/`.
    - [ ] Verify: Files exist in new location.

4.  **Consolidate Root Artifacts**
    - [ ] Move `artifacts/compound-v/*` to `.promptherder/artifacts/compound-v/`.
    - [ ] Move `artifacts/superpowers/*` to `.promptherder/artifacts/superpowers/`.
    - [ ] Remove `artifacts/` root directory.
    - [ ] Verify: Root `artifacts/` is gone.

### Batch 3: Rename `sync.go` (Low Risk)

5.  **Rename Copilot Implementation**
    - [ ] Rename `internal/app/sync.go` → `internal/app/copilot.go`.
    - [ ] Rename `internal/app/sync_test.go` → `internal/app/copilot_test.go`.
    - [ ] Update any internal references (search for "sync.go").
    - [ ] Verify: `go test ./internal/app/...` passes.

### Batch 4: Documentation (Low Risk)

6.  **Add Skills Documentation**
    - [ ] Create `compound-v/skills/README.md`.
    - [ ] Document platform constraints:
      - Antigravity requires `SKILL.md` filename for discovery.
      - Antigravity requires subdirectory structure (package).
      - Copilot requires translation to `.prompt.md`.
    - [ ] Verify: File exists and explains the "why".

## Risks & Mitigations

- **Risk**: User confusion about `SKILL.md` ubiquity.
  - **Mitigation**: The new README.md directly addresses this.

## Rollback Plan

1.  Undo `sync.go` rename.
2.  Restore `coverage` files.
3.  Delete `README.md`.

## Persist

- [x] Plan written to `.promptherder/artifacts/plan.md`.
