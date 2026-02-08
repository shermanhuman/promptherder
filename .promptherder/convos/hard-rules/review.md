# Review: methodology-refactor + hard-rules

> Scope: compound-v thin-link refactor + promptherder hard-rules feature

## Blockers

**B1: Antigravity hard-rules runs even when Walk fails** (`antigravity.go:106-124`)

Walk error captured in `err`, but hard-rules block executes regardless. Can partially succeed (write hard-rules) while returning a Walk error. Fix: add `if err != nil { return installed, err }` before hard-rules block.

## Majors

**M1: `hardRulesFile` constant lives in `antigravity.go` but is used by `copilot.go`** — package-level constant works but creates a non-obvious dependency. Move to shared location.

**M2: No stale cleanup for hard-rules.md** — if user removes `.promptherder/hard-rules.md`, the installed `.agent/rules/hard-rules.md` persists until manually deleted. Consistent with generated files, but may surprise users.

**M3: Copilot source count includes injected hard-rules** (`copilot.go:92`) — log shows inflated source count.

## Minors

**m1: `trigger: always_on` in hard-rules is effectively dead metadata for Copilot** — works fine, just not actively consumed.

**m2: Announce pattern duplicated in review workflow AND review skill** — workflow should defer to skill.

**m3: Skill says decisions.md path is "determined by the calling workflow"** — implicit contract, could be documented more explicitly.

## Nits

**n1: `/execute` has announce in workflow, not in a skill** — inconsistent with other workflows.

**n2: `/idea` and `/rule` don't announce** — lightweight, but inconsistent.

**n3: `/rule` doesn't document no-argument behavior.**

## Next Actions

1. **Fix B1** — early return on Walk error (1 line)
2. **Fix M1** — move constant to shared location
3. **Consider M2** — stale cleanup policy
4. **Optional m2** — remove announce duplication
