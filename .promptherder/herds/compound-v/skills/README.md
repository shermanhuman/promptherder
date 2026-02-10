# Agent Skills

This directory contains the skills that define the capabilities of our AI agents. The structure of these skills is strictly defined by platform requirements.

## Directory Structure

Each skill must be contained in its own subdirectory and must include a primary definition file named `SKILL.md`.

```
skills/
└── skill-name/       # The package for the skill
    ├── SKILL.md      # REQUIRED: Skill definition and instructions
    ├── scripts/      # Optional: Executable scripts
    └── examples/     # Optional: Example usage
```

## Why `SKILL.md`?

The filename `SKILL.md` is a **hard requirement** for the Antigravity agent platform (and the open Agent Skills standard).

- **Discovery**: Agents scan for this specific filename to identify available skills.
- **Context Loading**: The agent reads the `name` and `description` from the YAML frontmatter of `SKILL.md` to determine relevance before loading the full instructions.

## Why Subdirectories?

Skills are treated as packages. The subdirectory serves as the container for the skill's definition (`SKILL.md`) and any supporting resources (scripts, templates, etc.) that the agent might need to reference or execute.

## Copilot Support

For GitHub Copilot, these skills are automatically translated into prompt files (`.github/prompts/*.prompt.md`) by the promptherder `copilot` target. Copilot does not natively consume `SKILL.md` files, so promptherder handles the translation to maintain a single source of truth across agent platforms.
