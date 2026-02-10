---
name: compound-v-review
description: Reviews changes for correctness, edge cases, style, security, and maintainability with severity levels. 10 parallel checks with version-specific research. Use before finalizing changes.
---

# Review Skill

Act as a senior engineer performing a thorough code review.

**Announce at start:** "Running review pass on [scope]."

## When to use this skill

- before delivering final code changes
- after implementing a planned set of steps
- before merging or shipping

## Check targeting

User can run a single check by short name: `/review security`, `/review edges`, `/review perf`.
If no check specified, run all 10.

| #   | Check                       | Short name    |
| --- | --------------------------- | ------------- |
| 1   | Correctness                 | `correctness` |
| 2   | Edge cases & error handling | `edges`       |
| 3   | Security                    | `security`    |
| 4   | Performance                 | `perf`        |
| 5   | Tests                       | `tests`       |
| 6   | Design & maintainability    | `design`      |
| 7   | DRY                         | `dry`         |
| 8   | YAGNI & overengineering     | `yagni`       |
| 9   | Logging & observability     | `logging`     |
| 10  | Documentation               | `docs`        |

## Research before reviewing

Before running the checks, do all research **in parallel** (`waitForPreviousTools: false`):

1. Read `.agent/rules/stack.md` for pinned versions. If it doesn't exist, infer versions from `go.mod`, `mix.exs`, `package.json`, or equivalent.
2. Use `git diff` against the pre-implementation baseline for the review scope.
3. Read all changed files in parallel to build full context.
4. `search_web` for version-specific docs, gotchas, and best practices scoped to `stack.md` versions.
5. Read `.promptherder/hard-rules.md` if it exists.
6. Read `.promptherder/convos/<slug>/plan.md` and `.promptherder/future-tasks.md`.

## Review checks (run in parallel)

Each check is independent. Fire all checks concurrently.
Checks marked 🔍 require `search_web` using the specific versions from `stack.md`.

---

### 🎯 1. `correctness` — does it do what was asked?

Walk through every step in `.promptherder/convos/<slug>/plan.md`. Confirm the code delivers each one.
Cross-check `.promptherder/future-tasks.md` — catch any deferred idea we accidentally skipped.
Read `.promptherder/hard-rules.md` — flag any violation. Hard rules are non-negotiable.
Verify return values, status codes, and error types match the contracts defined.
Trace every conditional branch. Confirm all expected states are handled, not just the happy path.
Check pre/post conditions: what must be true before calling a function, and what must be true after it returns? Flag any violations.
Flag hardcoded solutions that only work for specific test inputs. The code must solve the general problem.

---

### 🚧 2. `edges` — what breaks at the boundaries? 🔍

**Stack research:** `search_web` for common edge cases and error patterns for the specific versions in `stack.md`.

Test the boundaries: nil, empty, zero, max, negative, off-by-one. If any are unhandled, flag them.
Trace resource lifecycle — files, connections, goroutines must be released on ALL paths, including error paths.
Follow every error. It must be wrapped with context (e.g., `fmt.Errorf %w` in Go, `raise ... from` in Python, `Kernel.reraise` in Elixir), never swallowed or silently ignored.
Check for race conditions. Shared state needs guards: mutexes, channels, or atomics. Maps, slices, and dicts shared across goroutines without synchronization are bugs.
Simulate partial failure: if step 3 of 5 fails, are steps 1-2 cleaned up or left dangling?
Consider external failures: what happens when the API/DB/filesystem is unavailable or slow?
Watch for type coercion traps: charset, timezone (DST, leap years, leap seconds), integer overflow for the language version in use.
Check for retry/recovery: transient failures (network timeouts, 503s) should have retry mechanisms where appropriate.

---

### 🛡️ 3. `security` — close the doors 🔍

**Stack research:** `search_web` for CVEs, security advisories, and OWASP ASVS misconfigurations for the specific versions in `stack.md`.

Search for hardcoded secrets and credentials. Search logs for leaked tokens or PII. Flag both.
Trace user input from entry to use. Sanitize before queries, commands, file paths, and templates.
Check every entry point for auth/authz — including background jobs, webhooks, and admin routes.
Look for unsafe defaults: permissive CORS, debug mode in prod, open ports, wildcard origins. Tighten them.
Check dependencies against known CVEs. Flag any at vulnerable versions.
Verify cryptographic usage: no weak algorithms, no hardcoded IVs, sufficient key lengths.
Check session management: session IDs regenerated on auth, HttpOnly/Secure cookie flags, session termination on logout/inactivity.
Check IaC files too: Kubernetes manifests, Dockerfiles, Terraform — not just application code.

---

### ⚡ 4. `perf` — don't waste cycles 🔍

**Stack research:** `search_web` for common performance pitfalls for the specific versions in `stack.md`.

Find N+1 queries and unbounded loops over external data. Batch or paginate them.
Look for unnecessary allocations or copies in hot paths. Eliminate them.
Verify every external call (DB, HTTP, file I/O) has pagination, limits, and timeouts. Add what's missing.
Check for blocking operations in async/concurrent contexts. They stall everything.
Flag large payloads loaded fully into memory. Use streaming where the data size is unbounded.
Look for caching opportunities: repeated expensive computations or fetches that could be cached.
Check database queries for missing indexes. Queries on unindexed columns are silent performance killers.

---

### 🧪 5. `tests` — prove it works

Run the tests. Don't assume they pass — execute them.
Check that new behavior has corresponding tests. Missing coverage = missing confidence.
Verify tests assert behavior, not implementation. Test return values and outcomes, not internal method calls.
Confirm edge cases are tested: empty, nil, max, concurrent, error paths.
Check error paths explicitly. Happy-path-only tests give false confidence.
Read the test names. Each should describe the scenario: `TestEmptyInputReturnsError`, not `TestProcess`.
Verify test independence: tests must run in any order without shared state. Shared mutable state between tests causes flaky failures.
Flag flaky indicators: tests that depend on timing, network availability, or filesystem state without proper setup/teardown.

---

### 📐 6. `design` — keep it simple and idiomatic 🔍

Give each function and type a single responsibility. If it does two things, split it.
Use names that describe purpose, not implementation: `fetchUser` not `getData`, `hardRulesFile` not `file2`.
Extract constants and config. No magic numbers, no magic strings scattered through the code.
Separate concerns: business logic apart from I/O, transport, and presentation.
Minimize coupling: changing module A shouldn't require touching module B.
Apply the 2-minute rule: can a new team member understand this function in under 2 minutes? If not, simplify.
Match patterns to the latest framework best practices for the versions in `stack.md`.

**Idiomatic code** 🔍 — `search_web` for "idiomatic [language] [version]":

Use language-native constructs: list comprehensions in Python, channels in Go, pattern matching in Elixir.
Follow the language's style guide: Effective Go, PEP 8, Elixir formatter.
Prefer standard library over reinventing. Use `slices.Contains` in Go 1.21+, not a manual loop.
Match naming conventions: camelCase vs snake_case, exported vs unexported.
Handle errors the language's way: Go returns errors, Elixir uses ok/error tuples, Python raises exceptions.

---

### 🔁 7. `dry` — single point of truth

Apply DRY (Single Point of Truth): every piece of knowledge should have one unambiguous, authoritative representation in the codebase.
Identify repeated code — same logic in multiple functions, same constant in multiple files.
Extract shared logic into a reusable function, module, or variable.
Call the reusable unit instead of duplicating code. Update logic in one place — changes reflect everywhere.
Respect the Rule of Three: below 3 occurrences, duplication may be acceptable. Don't abstract too early.
Check for copy-pasted code with minor variations. Parameterize the differences.

---

### 🪓 8. `yagni` — build it now, not "just in case"

Apply YAGNI: only implement features when you actually need them. Focus on current requirements, not hypothetical future ones.

**Technique:** `grep`/search for actual callers before accepting an abstraction. If nothing calls it, flag it.

Flag abstractions (interfaces, generics, factories) that serve no current caller. Remove or simplify them.
Cut "future-proofing" for unconfirmed requirements. Refactor later when real needs arise.
Replace sophisticated patterns (strategy, visitor, plugin system) with simple functions wherever possible.
Remove config or extension points nobody asked for. Keep the code lean and maintainable.
Don't optimize without evidence. Flag premature optimization that lacks measured bottleneck data.
Generalize upon second use, not first. The first occurrence is not a pattern — it's just code.
Flag deep inheritance hierarchies (3+ levels) — they usually indicate premature abstraction.

---

### 📋 9. `logging` — make it debuggable in production

Use correct log levels: INFO for expected operations, WARN for recoverable issues, ERROR for failures requiring attention.
Exclude sensitive data from logs: secrets, tokens, PII, full request bodies. Search and flag any leaks.
Include sufficient context: request IDs, correlation/trace IDs, relevant parameter values, timestamps. "Error occurred" alone is useless.
Keep structured logging format consistent with project conventions (slog, zerolog, etc.).
Find silent failures: code paths that swallow errors without logging. Every error needs a trace.
Consolidate related log entries: prefer canonical/wide events (one structured log per request) over scattered individual log lines.
Watch log volume: excessive logging causes cost and performance issues in production. Flag unnecessary INFO/DEBUG in hot paths.

---

### 📝 10. `docs` — explain the why, not the what

Document public APIs: function signatures, expected inputs, outputs, and error conditions.
Add comments that explain _why_ the approach was chosen. Don't restate what the code does — the code already says that.
Update README/docs when user-facing behavior, CLI flags, or configuration options change.
Document migration steps and deprecation notes when replacing old behavior.
Flag and remove commented-out code. It's not documentation — it's clutter. Use version control.
Flag outdated comments that don't match current code behavior. A wrong comment is worse than no comment.

---

## Output format

Keep the review **scannable**. The user should grasp the full picture in 10 seconds from the table, then drill into details only where needed.

### 1. Strengths (3-5 items max)

Highlight what's well done. Be specific with file:line references. Don't list everything good — pick the top 3-5.

---

### 2. Check coverage

**Always include this table.** Every check must appear — no silent omissions.

```
| # | Check | Result |
|---|-------|--------|
| 1 | `correctness` | ✅ Clean |
| 2 | `edges` | ⠷ M1 |
| 3 | `security` | N/A — no auth or input handling |
| ... | ... | ... |
```

Mark each check: `✅ Clean` (no issues), finding IDs (e.g. `⠷ M1, ⠴ m2`), or `N/A — reason`.

---

### 3. Findings

**Always start with a summary table.** This is the scannable overview:

```
| ID | Sev | Location | Issue |
|----|-----|----------|-------|
| B1 | ⠿ | file.go:42 | Brief description |
| M1 | ⠷ | auth.go:15 | Brief description |
| m1 | ⠴ | utils.go:8 | Brief description |
```

Detail a finding ONLY if the fix is non-obvious or the reasoning is nuanced. If someone can understand the issue and fix from the table row alone, do NOT expand it below.

```
⠷ **M1**: auth.go:15 — Why this matters (1-2 lines)
  Fix: specific remediation.
```

Finding IDs: `⠿ **B1**` (blocker), `⠷ **M1**` (major), `⠴ **m1**` (minor), `⠠ **n1**` (nit).

---

### 4. Persistence (before verdict)

Write review to `.promptherder/convos/<slug>/review-<description>.md`.

- `<description>` should match the review scope (e.g. `review-login-fix.md`, `review-security.md`).
- Do not overwrite previous reviews unless explicitly requested.

The persisted file contains strengths, check coverage, findings table, details, and assessment — but NOT the action menu. The menu is conversational, not archival.

**Overwrite guard:** (Managed by `compound-v-persist` skill usage + dynamic filename)

Confirm the file exists by listing `.promptherder/convos/<slug>/`.

---

### 5. Verdict

State your assessment in 1-2 sentences (what you found, what matters most). Then present the action menu:

> FIX to fix ⠿⠷ (blockers + majors), FIX ALL to fix everything, SKIP to move on without fixes, or give feedback.

_Task: `<slug>`_

**The action menu appears exactly ONCE, at the very end of the response.** It comes AFTER persistence. Do not repeat it after file operations or any other step.

If any finding is unclear, clarify ALL unclear items before fixing ANY.

---

## Fix triage (when user approves)

- FIX ALL → Fix in severity order (⠿ → ⠷ → ⠴ → ⠠), test after each.
- FIX → Fix ⠿ blockers and ⠷ majors. Note ⠴⠠ for later.
- SKIP → Confirm artifacts, move on.
- Feedback → Discuss, then fix agreed items.

**`YOLO` mode:**

Skip presentation. Auto-fix ALL findings (⠿ → ⠷ → ⠴ → ⠠). Output summary of what was fixed.
