# Plan: Per-repo hard rules

> `/plan` · **Status**: approved · `.promptherder/convos/hard-rules/plan.md`

## Goal

Add a per-repo `.promptherder/rules.md` file whose contents are always injected into every target agent, plus a `/rule` workflow to append rules to it.

## Plan

1. Implement rules.md support in AntigravityTarget
   - Files: `internal/app/antigravity.go`
   - Change: During Install, read `.promptherder/rules.md` (if it exists) and prepend its content to the installed `rules/` output (or write it as a standalone rules file in the target dir).
   - Verify: `go vet ./...`, `go test ./internal/app/ -run Antigravity -v`

2. Implement rules.md support in CopilotTarget
   - Files: `internal/app/copilot.go`
   - Change: Same pattern — read `.promptherder/rules.md` and inject into Copilot's target output.
   - Verify: `go vet ./...`, `go test ./internal/app/ -run Copilot -v`

3. Add tests for rules.md injection
   - Files: `internal/app/antigravity_test.go`, `internal/app/copilot_test.go`
   - Change: Add test cases with a `.promptherder/rules.md` fixture, verify the content appears in target output. Add test for missing rules.md (no error, no injection).
   - Verify: `go test ./internal/app/ -v -count=1`

4. Add `/rule` workflow to compound-v herd
   - Files: `workflows/rule.md` (in compound-v repo)
   - Change: Lightweight workflow (like `/idea`) — appends a rule to `.promptherder/rules.md`. Creates the file with a header if it doesn't exist.
   - Verify: Manual — read the file, confirm format.

5. Update compound-v.md to list `/rule`
   - Files: `rules/compound-v.md` (in compound-v repo)
   - Change: Add `/rule` to the Pipeline section.
   - Verify: Visual — read the file.

## Risks & mitigations

- **Content collision**: rules.md content could conflict with herd rules → mitigate by placing it as a separate file in the target (e.g., `.agent/rules/_project.md`), not merging into herd rule files.
- **Missing file**: rules.md doesn't exist → no-op, not an error.

## Rollback plan

Remove the rules.md reading logic from the target Install methods. Delete the `/rule` workflow.
