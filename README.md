# promptherder

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Distribute AI coding instructions from one source to every agent you use.

```
.promptherder/agent/     (you edit here)
        │
        │  promptherder
        │
        ├──→ .agent/     (Antigravity)
        ├──→ .github/    (VS Code Copilot)
        └──→ ???/        (easy to add)
```

## Install

```bash
go install github.com/shermanhuman/promptherder/cmd/promptherder@latest
```

## Quick Start

```bash
# Install Compound V methodology + sync to all agents
promptherder compound-v
promptherder

# Or pull a herd from GitHub
promptherder pull https://github.com/shermanhuman/compound-v
promptherder
```

## Commands

| Command | What it does |
|---|---|
| `promptherder` | Sync to all targets |
| `promptherder copilot` | Sync to `.github/` only |
| `promptherder antigravity` | Sync to `.agent/` only |
| `promptherder compound-v` | Install Compound V methodology |
| `promptherder pull <url>` | Pull a herd from GitHub |
| `promptherder --dry-run` | Show what would be written |

## Source Format

Rules live in `.promptherder/agent/rules/*.md`:

```markdown
# Hard Rules

- Do not generate plaintext secrets.
- All changes go through Git.
```

Path-scoped rules use frontmatter:

```markdown
---
applyTo: "**/*.sh"
---

# Shell Script Safety

Use `set -Eeuo pipefail`.
```

## Compound V

promptherder ships with **Compound V**, an AI coding methodology forked from [obra/superpowers](https://github.com/obra/superpowers).

Where Superpowers emphasizes control through subagents, checkpoints, and gated reviews, Compound V prioritizes getting results with minimal human effort — autonomous research, parallel execution, and decisions surfaced only when genuinely needed.

See the [Compound V README](https://github.com/shermanhuman/compound-v) for details.

## Manifest

promptherder tracks written files in `.promptherder/manifest.json` for idempotent cleanup. Commit this file.

## Credits

- [obra/superpowers](https://github.com/obra/superpowers) — the original methodology
- [anthonylee991/gemini-superpowers-antigravity](https://github.com/anthonylee991/gemini-superpowers-antigravity) — Antigravity adaptation
