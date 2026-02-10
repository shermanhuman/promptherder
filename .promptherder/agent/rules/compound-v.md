---
trigger: always_on
---

# Compound V

You have the Compound V methodology available. Use these workflows and skills:

## Pipeline

1. `/plan` — autonomous planning with `/execute` approval
2. `/execute` — parallel-by-default execution with checkpointing
3. `/review` — severity-graded review pass
4. `/idea` — add to future tasks (lightweight, any time)
5. `/rule` — add a hard rule to the always-on prompt

## Skills (auto-activated)

- `compound-v-plan` — autonomous planning methodology
- `compound-v-review` — severity-graded review with 10 parallel checks
- `compound-v-tdd` — tests-first discipline
- `compound-v-debug` — systematic debugging
- `compound-v-parallel` — parallel execution reasoning
- `compound-v-verify` — verification before completion
- `compound-v-persist` — resolves conversation slugs and paths

## Manual rules

- `browser.md` — browser-based UI testing (manual trigger)

## Output formatting

All workflows and skills must follow these formatting rules:

### Structure

- **H1** for titles, **H2** for sections, **H3** for subsections
- `---` dividers between major sections
- Tables for structured data (findings, decisions, comparisons)
- Ordered lists for sequential steps. Unordered lists when order doesn't matter.

### Semantic text formatting

- **Bold** for key terms and action verbs in steps: "**Read** the file. **Append** the rule."
- `Inline code` for anything the user might copy: commands, paths, filenames, flags, slugs
- _Italic_ for caveats, assumptions, and de-emphasized metadata: _"This assumes the plan exists."_
- Blockquotes (`>`) for prompting the user — questions, decisions, and action menus. Not for informational text.

### Severity indicators (braille dot patterns)

Visual fill-level = severity. Works without color.

- `⠿` **Blocker** — wrong behavior, security issue, data loss, broken build
- `⠷` **Major** — likely bug, missing edge case, poor reliability
- `⠴` **Minor** — style, clarity, small maintainability issue
- `⠠` **Nit** — optional polish

Finding IDs are mandatory in reviews: `⠿ **B1**`, `⠷ **M2**`, `⠴ **m3**`, `⠠ **n1**`

### Decision prompts

Examples:

- **After plan:** `> Run /execute <slug> to proceed, SHOW DECISIONS to audit, DECLINE to reject, or give feedback.`
- **After review findings:** `> FIX to fix ⠿⠷, FIX ALL to fix everything, SKIP to move on, or give feedback.`
- **Deferred ideas:** `> Add these to future-tasks.md? yes / no`

Task slugs go on the next line in italics: _Task: `<slug>`_

### Short names first

When referencing any concept that has a short name, lead with the short name in backticks followed by a brief description. This teaches users the vocabulary progressively.

- ✅ `edges` — boundary conditions and error handling
- ✅ `perf` — performance pitfalls
- ✅ `YOLO` — full autonomous mode
- ❌ "Edge cases & error handling" (user doesn't learn the shortcut)

### YOLO mode

The `YOLO` flag (all caps) enables full autonomous operation. It cascades through the pipeline:

- `/review YOLO` — Auto-fix ALL findings (⠿→⠠) without asking. Output summary.
- `/execute YOLO` — Execute, auto-review, auto-fix all findings. No interaction.
- `/plan YOLO` — Full pipeline: plan → execute → review → fix. Zero interaction.

In all YOLO modes, still output a summary at the end (what was built, what was found, what was fixed).
