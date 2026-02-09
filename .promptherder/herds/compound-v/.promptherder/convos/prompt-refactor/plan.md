# Plan: prompt-refactor

> **Status**: approved

## Goal

Refactor all Compound V prompts so they produce output that is readable at a glance, enforce DRY/YAGNI/TDD sequencing, and give the user enough visual context (filesystem trees, interface sketches, happy paths) to understand what's being built before approving.

---

## Context

After dogfooding the methodology on the hard-rules feature, the user identified concrete pain points:

1. **Plan was unclear** — hard to understand what was being built without visual anchors
2. **Output was hard to read** — wall-of-text, no structural hierarchy, hard to scan
3. **Review findings hard to cross-reference** — "m2" was mentioned but hard to locate
4. **Review checklist was vague** — "readability" means nothing without concrete criteria
5. **Missing domains** — YAGNI/overengineering/DRY not checked during review
6. **TDD discipline** — tests came after code, not before
7. **No smoke test guidance** — execute finishes without manual validation walkthrough
8. **Parallel not enforced** — some skills say "parallel" but don't all enforce it

---

## Plan

### 1. Add output formatting rules to `compound-v.md`

- **Files:** `rules/compound-v.md`
- **Change:**
  - Add a `## Output formatting` section to the always-on rule (all workflows/skills inherit it)
  - Rules: header hierarchy (H1 for titles, H2 for sections, H3 for subsections), `---` dividers between major sections, tables for structured data (findings, decisions, comparisons), bold for key terms, code blocks for paths/commands
  - Severity icons use braille dot patterns (visual fill-level = severity):
    - `⠿` **Blocker** (red if color available)
    - `⠷` **Major** (orange if color available)
    - `⠴` **Minor** (yellow if color available)
    - `⠠` **Nit** (gray if color available)
  - Color: use `<span style="color:...">` for HTML-capable renderers; design must work without color (braille dots carry the hierarchy on their own)
  - Finding IDs are mandatory in reviews: `⠿ **B1**`, `⠷ **M2**`, `⠴ **m3**`, `⠠ **n1**`
  - **Decision prompts** must use markdown blockquotes (`>`) for visual consistency. Every point where the user must choose an action gets a blockquote. Specific prompts:
    - **After plan:** `> Run /execute <slug> to proceed, SHOW DECISIONS to audit, DECLINE to reject, or give feedback.`
    - **After review findings:** `> Run fix to fix all, fix blockers for ⠿ only, or give feedback. Task: <slug>`
    - **Deferred ideas:** `> Add these to future-tasks.md? yes / no`
  - **Short names first:** When referencing review domains, workflows, or features that have a short name, always lead with the short name in backticks followed by a brief description. This teaches users the vocabulary progressively. Examples:
    - ✅ `edges` — boundary conditions and error handling
    - ✅ `perf` — performance pitfalls
    - ✅ `YOLO` — full autonomous mode
    - ❌ "Edge cases & error handling" (user doesn't learn the shortcut)
- **Verify:** Visual review of the rule file

### 2. Enhance plan presentation template in `compound-v-plan` skill

- **Files:** `skills/compound-v-plan/SKILL.md`
- **Change:**
  - Add to Phase 4 template: `## What this builds` section with:
    - **Happy path** — numbered walkthrough of the user-facing flow after implementation
    - **Filesystem tree** — `tree` showing files created/modified (before → after)
    - **Interface sketch** (when applicable) — ASCII or markdown description of any UI/CLI changes
  - Add to template: `## References` section with links to `plan.md` and `decisions.md` paths
  - Add TDD sequencing rule: each plan step must show test→code→verify order
  - Add: "If the plan changes the codebase structure, show a before/after filesystem tree"
  - In Phase 3 (Think), add explicit YAGNI/DRY checks: "For each step, ask: does this duplicate existing logic? Is this the simplest approach?"
- **Verify:** Visual review; compare template against the hard-rules plan to check improvements

### 2b. Update plan workflow approval flow

- **Files:** `workflows/plan.md`
- **Change:**
  - Replace the current prompt with:
    > Run `/execute <slug>` to proceed, `SHOW DECISIONS` to audit, `DECLINE` to reject, or give feedback.
  - Remove the APPROVED handler. Running `/execute` IS the approval.
  - **Status tracking in plan.md:**
    - `draft` — plan presented, awaiting response
    - `approved` — set when `/execute` is run (execute workflow sets this as a precondition step)
    - `declined` — set when user says DECLINE
  - **Response handlers:**
    - `/execute` → Status updated to `approved` by execute workflow. Proceed.
    - `SHOW DECISIONS` → Print `decisions.md`. Re-prompt.
    - `DECLINE` → Update status to `declined` in plan.md. Reply: "Plan declined. Task: `<slug>`". Stop.
    - Feedback → Incorporate, re-research if needed, update plan and decisions, re-present.
- **Verify:** Visual review

### 3. Overhaul review skill with concrete criteria

- **Files:** `skills/compound-v-review/SKILL.md`
- **Change:**
  - Add senior engineer persona: "Act as a senior engineer performing a thorough code review."
  - Replace vague checklist with specific, actionable criteria organized into **10 domains**
  - Each domain runs as an independent parallel check
  - **Research phase upgrade:** Before reviewing, read `stack.md` and `search_web` for the **specific pinned versions** of libraries/frameworks used. Download version-specific docs where available. This catches deprecated APIs, breaking changes, and version-specific gotchas.
  - **Domain targeting:** User can run a single domain by its **short name**: `/review security`, `/review edges`, `/review perf`, etc. When a short name is provided, run ONLY that domain (skip the rest). If no domain specified, run all 10.
  - **Domain short names:**

    | #   | Domain                      | Short name    |
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

  - **YOLO mode:** The `YOLO` flag (all caps) enables full autonomous operation. It cascades through the pipeline:
    - `/review YOLO` — Skip findings presentation. Auto-fix ALL findings (⠿→⠠) without asking. Output summary of what was fixed.
    - `/execute YOLO` — Execute the plan, then auto-run review at finish, then auto-fix all findings. No interaction.
    - `/plan YOLO` — Full pipeline: plan → auto-approve → execute → auto-review → auto-fix. Zero interaction, start to finish.
    - In all YOLO modes, still output a summary at the end (what was built, what was found, what was fixed).
  - New checklist structure:

  #### 🎯 1. Correctness (`correctness`)

  Verify the implementation matches what was planned, agreed upon, and doesn't violate project rules.
  - Does the code fulfill every step in `.promptherder/convos/<slug>/plan.md`?
  - Cross-check `.promptherder/future-tasks.md` — did we accidentally skip a related deferred idea that should have been included?
  - **Check `.promptherder/hard-rules.md`** — does the code violate any hard rules?
  - Are return values, status codes, and error types correct for the contracts defined?
  - Do conditional branches cover all expected states (not just the happy path)?

  #### 🚧 2. Edge cases & error handling (`edges`) 🔍

  Verify the code handles the boundaries and failure modes of the stack.
  **Stack research:** `search_web` for common edge cases and error patterns for the specific versions in `stack.md` (e.g., "Go 1.22 common gotchas", "Phoenix 1.7 error handling patterns").
  - Boundary conditions: nil, empty, zero, max, negative, off-by-one?
  - Resource cleanup: files, connections, goroutines — released on ALL paths (including error paths)?
  - Error propagation: errors wrapped with context (`fmt.Errorf %w`), not swallowed or silently ignored?
  - Race conditions: concurrent access to shared state guarded (mutexes, channels, atomic)?
  - Partial failure: if step 3 of 5 fails, are steps 1-2 cleaned up or left dangling?
  - External dependency failure: what happens when the API/DB/filesystem is unavailable or slow?
  - Type coercion / encoding: charset, timezone, integer overflow for the language version in use?

  #### 🛡️ 3. Security (`security`) 🔍

  Verify the code doesn't introduce vulnerabilities specific to the stack.
  **Stack research:** `search_web` for CVEs, security advisories, and common misconfigurations for the specific versions in `stack.md` (e.g., "Go 1.22 security advisories", "Ecto SQL injection patterns").
  - Secrets hardcoded or logged?
  - User input sanitized before use in queries, commands, file paths, templates?
  - Auth/authz checked on every entry point (including background jobs, webhooks, admin routes)?
  - Unsafe defaults (permissive CORS, debug mode in prod, open ports, wildcard origins)?
  - Dependency vulnerabilities: are any imported packages at versions with known CVEs?
  - Cryptographic misuse: weak algorithms, hardcoded IVs, insufficient key lengths?

  #### ⚡ 4. Performance (`perf`) 🔍

  Verify there are no obvious performance issues specific to the stack.
  **Stack research:** `search_web` for common performance pitfalls for the specific versions in `stack.md` (e.g., "Go 1.22 performance anti-patterns", "Ecto N+1 prevention").
  - N+1 queries or unbounded loops over external data?
  - Unnecessary allocations or copies in hot paths?
  - Missing pagination, limits, or timeouts on external calls (DB, HTTP, file I/O)?
  - Blocking operations in async/concurrent contexts?
  - Large payloads loaded fully into memory where streaming would work?

  #### 🧪 5. Tests (`tests`)

  Verify test quality — not just coverage counts, but meaningful assertions.
  - New behavior covered by tests?
  - Tests assert behavior, not implementation details (e.g., testing return values, not internal method calls)?
  - Edge cases tested (empty, nil, max, concurrent, error paths)?
  - Error paths tested explicitly (not just happy path)?
  - Tests actually run and pass? (execute them, don't assume)
  - Test names describe the scenario, not the function (`TestEmptyInputReturnsError` not `TestProcess`)?

  #### 📐 6. Design & maintainability (`design`)

  Verify the code's structure supports long-term comprehension and modification.
  - Single responsibility: does each function/type do one thing?
  - Names describe purpose (not implementation): `fetchUser` not `getData`, `hardRulesFile` not `file2`?
  - Constants/config extracted — no magic numbers, no magic strings?
  - Separation of concerns: business logic separate from I/O, transport, presentation?
  - Low coupling: can you change module A without touching module B?
  - Cognitive load: can a new team member understand this function in under 2 minutes?
  - Patterns match latest framework best practices for the versions in `stack.md`?
  - **Idiomatic code** 🔍: Does the code use the natural idioms and conventions of the language?
    `search_web` for "idiomatic [language] [version]" (e.g., "idiomatic Go 1.22", "idiomatic Elixir 1.16").
    - Uses language-native constructs (e.g., list comprehensions in Python, channels in Go, pattern matching in Elixir)?
    - Follows the language's style guide (Effective Go, PEP 8, Elixir formatter)?
    - Leverages standard library over reinventing (e.g., `slices.Contains` in Go 1.21+ vs manual loop)?
    - Naming follows language conventions (camelCase vs snake_case, exported vs unexported)?
    - Error handling follows language idioms (Go: return errors, Elixir: ok/error tuples, Python: exceptions)?

  #### 🔁 7. DRY (`dry`)

  Verify there's no unjustified duplication (but respect the Rule of Three — don't abstract too early).
  - Duplicated logic that appears 3+ times? (below 3, duplication may be acceptable)
  - Existing utility rewritten instead of reused?
  - Copy-pasted code with minor variations that could be parameterized?
  - Shared constants defined in one place, not scattered?

  #### 🪓 8. YAGNI & overengineering (`yagni`)

  Verify every abstraction and extension point serves a current, concrete use case.
  **Technique:** `grep`/search for actual callers before accepting an abstraction. If nothing calls it, flag it.
  - Abstractions (interfaces, generics, factories) serving no current caller?
  - "Future-proofing" for unconfirmed requirements?
  - Sophisticated patterns (strategy, visitor, plugin system) where a simple function would work?
  - Config or extension points nobody asked for?
  - Premature optimization without measured evidence of a bottleneck?

  #### 📋 9. Logging & observability (`logging`)

  Verify the code is debuggable in production without redeployment.
  - Log level appropriate? (INFO for expected operations, WARN for recoverable issues, ERROR for failures requiring attention)
  - Sensitive data excluded from logs (secrets, tokens, PII, full request bodies)?
  - Sufficient context for debugging (request IDs, relevant parameter values, not just "error occurred")?
  - Structured logging format consistent with project conventions (slog, zerolog, etc.)?
  - Silent failures: are there code paths that swallow errors without logging?

  #### 📝 10. Documentation (`docs`)

  Verify the code communicates its intent to future readers.
  - Public APIs documented (function signatures, expected inputs/outputs)?
  - Complex logic has comments explaining _why_ the approach was chosen (not restating _what_ the code does)?
  - README/docs updated if user-facing behavior, CLI flags, or configuration options changed?
  - Deprecated patterns or migration steps documented if replacing old behavior?

  - **Review method:** Use `git diff` against the pre-implementation baseline for the review scope. Read changed files in parallel.
  - **Output format:**
    1. **Strengths** — What's well done? Be specific (file:line).
    2. **Findings** — Grouped by severity (⠿ → ⠷ → ⠴ → ⠠). Each finding must include:
       - Unique ID: `⠿ **B1**`, `⠷ **M2**`, `⠴ **m3**`, `⠠ **n1**`
       - File:line reference
       - What's wrong
       - Why it matters
       - How to fix (if not obvious)
    3. **Verdict** — "Ready to merge? **Yes** / **No** / **With fixes**" + 1-2 sentence reasoning
  - **Findings-first workflow:** Present findings and verdict. Do NOT auto-fix. Wait for user to say "fix" or give feedback. If any finding is unclear, clarify ALL unclear items before fixing ANY.
  - **Fix triage (when user approves fixing):**
    - ⠿ Blockers: fix immediately
    - ⠷ Majors: fix before proceeding
    - ⠴ Minors: note for later (or fix if quick)
    - ⠠ Nits: optional, fix only if trivial
    - Test after each fix. Verify no regressions.

- **Verify:** Visual review; compare against the hard-rules review output

### 4. Update execute workflow: smoke test + findings-first flow

- **Files:** `workflows/execute.md`
- **Change:**
  - Update the finish section to reflect the findings-first flow:

    ```
    ### Finish (required)

    **Normal mode:**
    1. Run review pass (use compound-v-review). Present findings and verdict.
    2. Wait for user response:
       - "fix" / "fix all" → Fix in severity order (⠿ → ⠷ → ⠴ → ⠠), test after each.
       - "fix blockers" → Fix only ⠿ findings.
       - Feedback → Discuss, then fix agreed items.
       - "ship it" → Skip fixes, confirm artifacts.
    3. Manual smoke test (when applicable):
       - List exact commands to test the happy path end-to-end
       - List edge cases worth testing manually
       - Show expected output for each command
    4. Confirm artifacts exist by listing .promptherder/convos/<slug>/.

    **YOLO mode:**
    1. Run review pass (use compound-v-review).
    2. Auto-fix ALL findings (⠿ → ⠷ → ⠴ → ⠠), test after each. No user interaction.
    3. Output summary: what was built, what was found, what was fixed.
    4. Confirm artifacts.
    ```

- **Verify:** Visual review of the workflow

### 5. Enforce parallel research in all skills

- **Files:** `skills/compound-v-tdd/SKILL.md`, `skills/compound-v-debug/SKILL.md`, `skills/compound-v-review/SKILL.md`
- **Change:**
  - TDD: add "Do research calls **in parallel** (multiple `search_web` + `view_file` calls)"
  - Debug: already says "parallel" in step 2 and 4 ✅ — just verify it's explicit about `waitForPreviousTools`
  - Review: already says "in parallel" ✅ — add "Use `waitForPreviousTools: false` for all research calls"
  - Add parallel note to review domains: "Run all domain checks in parallel. Each domain is independent."
- **Verify:** `grep -i parallel` across all skills

---

## Risks & mitigations

- **Risk:** Longer prompts increase token usage → mitigation: keep rules concise; use bullet points not prose
- **Risk:** Too many emoji becomes clownish → mitigation: fixed palette of 8 semantic emoji only, no decorative use
- **Risk:** Filesystem trees / happy paths add work to planning → mitigation: only required for multi-file changes

## Rollback plan

`git revert` — all changes are in the compound-v repo, single commit per batch.

---

## What this builds

### Happy path

1. User types `/plan add widget caching`
2. Agent researches silently, then presents a plan with:
   - Clear goal statement
   - Numbered steps with test→code→verify order
   - Filesystem tree showing what files change
   - Happy path walkthrough ("after this, when a user calls `/widgets`, the response is cached for 60s")
   - References: "Full plan: `.promptherder/convos/widget-cache/plan.md`"
3. User runs `/execute widget-cache` (this IS the approval)
4. Agent executes with TDD (tests first), logs each batch
5. At finish: review pass presents:
   - **Strengths:** "Clean cache invalidation logic at `cache.go:42-58`"
   - **Findings:** `⠿ **B1**: cache.go:35 — TTL not configurable, hardcoded 60s. Fix: extract to config.`
   - **Verdict:** "Ready to merge? **With fixes** — B1 is a blocker."
6. User says "fix" → agent fixes B1, tests pass
7. Smoke test: "Run `go test ./...`, then `curl localhost:8080/widgets` twice — second should be faster"
8. Artifacts confirmed

### Filesystem tree (what changes)

```
compound-v/
├── rules/
│   └── compound-v.md          ← + output formatting section
├── skills/
│   ├── compound-v-plan/
│   │   └── SKILL.md           ← + visual anchors, TDD sequencing, references
│   ├── compound-v-review/
│   │   └── SKILL.md           ← full overhaul: concrete criteria, parallel domains
│   ├── compound-v-tdd/
│   │   └── SKILL.md           ← + parallel research note
│   └── compound-v-debug/
│       └── SKILL.md           ← + parallel note (minor)
└── workflows/
    └── execute.md             ← + smoke test section
```
