# Compound V

A coding methodology herd for [promptherder](https://github.com/shermanhuman/promptherder).

Compound V provides rules, skills, and workflows that define a structured approach to AI-assisted software development.

## Install

```bash
# Install promptherder
go install github.com/shermanhuman/promptherder/cmd/promptherder@latest

# Pull this herd
promptherder pull https://github.com/shermanhuman/compound-v

# Sync to agent targets
promptherder
```

## What's Included

- **Rules** — Behavioral guidelines for AI coding agents
- **Skills** — Reusable capability modules (planning, debugging, TDD, review, etc.)
- **Workflows** — Step-by-step processes activated by `/plan`, `/execute`, `/review`

## Structure

```
compound-v/
├── herd.json           # Herd metadata
├── rules/
│   ├── browser.md      # Browser-based UI testing rules
│   └── compound-v.md   # Core methodology rules
├── skills/
│   ├── compound-v-debug/
│   ├── compound-v-parallel/
│   ├── compound-v-plan/
│   ├── compound-v-review/
│   ├── compound-v-tdd/
│   └── compound-v-verify/
└── workflows/
    ├── execute.md
    ├── plan.md
    └── review.md
```

## License

MIT License — Copyright (c) 2026 Sherman Boyd
