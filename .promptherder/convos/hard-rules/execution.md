# Execution: hard-rules

## Step 1–2: Inject hard-rules.md in both targets (parallel)

- `antigravity.go`: Added `hardRulesFile` constant, reads `.promptherder/hard-rules.md` and copies as `.agent/rules/hard-rules.md`. No-op if missing.
- `copilot.go`: Reads `.promptherder/hard-rules.md`, strips frontmatter, prepends body as first source in `copilot-instructions.md`.
- Verified: `go vet ./...` — clean.

## Step 3: Tests

- `antigravity_test.go`: Added `TestAntigravityTarget_HardRules` and `TestAntigravityTarget_NoHardRules`.
- `copilot_test.go`: Added `TestCopilotTarget_HardRules` and `TestCopilotTarget_NoHardRules`.
  - Hard rules test verifies content appears AND ordering (hard-rules before regular rules).
  - No-hard-rules test verifies no error and no hard-rules content.
- Verified: `go test ./... -count=1` — all pass (100+ tests).

## Step 4–5: /rule workflow + pipeline update (parallel, compound-v repo)

- Created `workflows/rule.md` — lightweight append workflow (matches `/idea` pattern).
- Updated `rules/compound-v.md` — added `/rule` to pipeline list.
- Verified: visual review.

## All steps verified. Committed and pushed both repos.
