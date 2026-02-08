# Review: Convo-Based Artifact Management

## Blockers

None.

## Majors

None.

## Minors

1. **First `go run` used cached binary** — The embedded FS wasn't updated on the first run. Required a second `go run` to pick up the changes. This is a known Go build cache behavior — not a bug in our code, but worth noting for the workflow: after editing `compound-v/` source files, run `go run` twice or use `go run -a` to force rebuild.

## Nits

1. **`00-compound-v.md` rule reference** — The rules file at `.agent/rules/compound-v.md` (previously `00-compound-v.md`) still references `.promptherder/artifacts/` in its workflow descriptions. This is cosmetic — the actual workflow files are correct.

## Verification

| Command                                              | Result                    |
| ---------------------------------------------------- | ------------------------- |
| `go vet ./...`                                       | Pass                      |
| `go test ./...`                                      | Pass (all packages)       |
| `Select-String artifacts/` in `.agent/workflows/`    | No matches                |
| `Select-String artifacts/` in `.github/prompts/`     | No matches                |
| `Select-String convos/<slug>` in `.agent/workflows/` | 20 matches across 4 files |
| `Test-Path .promptherder/convos/.gitkeep`            | True                      |

## Summary of Changes

| Change                                    | Files                                                                            |
| ----------------------------------------- | -------------------------------------------------------------------------------- |
| Added slug resolution block               | `compound-v/workflows/brainstorm.md`, `write-plan.md`, `execute.md`, `review.md` |
| Changed persist paths to `convos/<slug>/` | Same 4 files                                                                     |
| Updated folder layout tree                | `.agent/rules/structure.md`                                                      |
| Created convos directory                  | `.promptherder/convos/.gitkeep`                                                  |
| Reinstalled to all targets                | `.agent/workflows/*`, `.github/prompts/*`, `.promptherder/agent/workflows/*`     |

## Follow-ups

- Test the workflow in a real conversation — invoke `/brainstorm` and verify it creates a `convos/<slug>/` folder.
- Consider updating the `compound-v.md` rule file to reference `convos/` instead of `artifacts/`.
