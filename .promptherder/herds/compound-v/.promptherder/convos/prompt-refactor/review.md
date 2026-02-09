# Review: compound-v v0.6.0

## Strengths

- Consistent terminology: zero "domain" in source files. Clean migration to "check".
- Imperative style throughout: "Do X. Flag Y." — no passive questions remain.
- Short names table (SKILL.md:23-34): clean, scannable, memorizable.
- Research-backed gaps from actual industry checklists (OWASP ASVS, SPOT, canonical events).
- Parallel execution clearly stated with `waitForPreviousTools` references.
- Plan skill anti-patterns (compound-v-plan/SKILL.md:157-164): explicit guardrails.
- Lightweight workflows (idea.md, rule.md): zero-ceremony, appropriate friction.
- YOLO cascade (compound-v.md:69-77): clean 3-tier with preserved summaries.
- Blockquote decision prompts consistent across all workflows.

## Findings

### ⠿ B1: plan.md artifact — stale "domain" references

Historical plan artifact still uses "domain" 20+ times. Not a source file, but could confuse LLM during context reads. Acceptable as historical record.

### ⠷ M1: SKILL.md:72 — Go-centric error wrapping

`edges` check hardcodes `fmt.Errorf %w` but compound-v is language-agnostic. Other checks correctly multi-language.
Fix: Generalize with multi-language examples.

### ⠷ M2: SKILL.md:40 — stack.md assumed to exist

Review skill research phase doesn't handle missing stack.md. Plan skill says "if it exists" but review doesn't.
Fix: Add fallback: infer from go.mod/mix.exs/package.json.

### ⠷ M3: compound-v.md:20 — stale skill description

"review checklist" undersells the refactored skill.
Fix: Update description.

### ⠴ m1: compound-v-plan/SKILL.md:53-55 — orphaned passive questions

3 bullet points still in passive question style after imperative filters.
Fix: Rewrite as imperative.

### ⠴ m2: compound-v-verify/SKILL.md:18 — passive question style

Verify skill checklist not converted to imperative style.
Fix: Rewrite as imperative.

### ⠴ m3: execute.md:5 — turbo-all undocumented

`// turbo-all` annotation not explained in compound-v docs (promptherder convention).

### ⠴ m4: herd.json — version field undocumented

No schema docs for herd.json fields. Promptherder concern.

### ⠠ n1: SKILL.md:232-234 — triage labels imply check-severity mapping

Example uses check names as severity labels, suggesting each severity maps to a check.
Fix: Drop check names from triage labels.

### ⠠ n2: skills/README.md — possibly stale

Not reviewed this session.

## Verdict

Ready to merge? **With fixes** — M1 (Go-centrism) and M2 (missing stack.md fallback) could confuse non-Go projects.
