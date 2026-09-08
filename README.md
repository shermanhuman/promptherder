# promptherder

Write shared coding instructions once and compile them into native host rules and skill bundles. Codex and Claude Code are the primary integration targets; no target is enabled by default.

Version **1.0.0** replaces flattened prompt exports with a validated build plan. Everyday rules stay in a small baseline. Skill bodies, references, assets, and executable scripts remain in native skill directories and load when needed.

## Install and select targets

```bash
go install github.com/shermanhuman/promptherder/cmd/promptherder@latest

# In your project; lists targets with nothing preselected
promptherder install

# Or configure explicitly for automation
promptherder install codex claude
promptherder pull compound-v
promptherder plan
promptherder
```

The source checkout contains the upcoming 1.0.0 changes; `@latest` follows the most recent published tag. Build this checkout with `go build ./cmd/promptherder` to try the feature branch.

First interactive pull or sync offers setup if targets have never been selected. Noninteractive use requires explicit setup. `install none` saves an intentionally empty selection. Existing explicit selections remain intact; old implicit defaults require setup once.

```bash
promptherder target list
promptherder target add cursor
promptherder target remove cursor
promptherder                       # apply selection and remove unchanged stale output
```

`add` and `remove` accept multiple names and are idempotent. `agent` remains an alias for `target`. Selection commands save preferences; sync applies them. Direct `promptherder codex` adds Codex to the installed output plan without changing preferences and rebuilds installed consumers together to protect shared artifacts.

## Installation scope

Repository scope is the default. Personal installation is explicit and currently supports Codex and Claude Code:

```bash
promptherder install codex claude --scope user
promptherder pull compound-v --scope user
promptherder plan --scope user --json
promptherder --scope user
```

User-scope sources, settings, and ownership records live in `~/.promptherder/`. Codex uses `~/.codex/AGENTS.md` and `~/.agents/skills/`; Claude uses `~/.claude/CLAUDE.md`, rules, and skills. When both are selected, Claude imports the shared personal baseline. Claude alone gets its own baseline. Use `--scope user` consistently for selection, synchronization, diagnostics, and recovery. Nondefault `CODEX_HOME` layouts are rejected for user installation. Other targets currently support repository installation only.

## Native output

| Target | Baseline / scoped rules | Skill bundles |
|---|---|---|
| Codex | `AGENTS.md`; conditional references for arbitrary globs | `.agents/skills/` |
| Claude Code | `CLAUDE.md` imports `AGENTS.md`; `.claude/rules/` | `.claude/skills/` |
| VS Code Copilot | `AGENTS.md`; `.github/instructions/` | `.github/skills/` or a selected shared root |
| Cursor | `AGENTS.md`; `.cursor/rules/*.mdc` | `.agents/skills/` |
| Windsurf | `.windsurf/rules/` | `.windsurf/skills/` |
| Cline | `AGENTS.md`; `.clinerules/` | `.cline/skills/` or selected `.claude/skills/` |
| Antigravity | `.agents/rules/` | `.agents/skills/` |

Gemini CLI has been removed. Copilot means the documented VS Code profile, not every Copilot surface. Capability evidence and verification dates are included in JSON plans. Host versions and personal/enterprise configuration can alter discovery.

Codex arbitrary glob rules use a conditional reference with a compatibility warning. `required: true` rejects that fallback. Manual skills use Claude's `disable-model-invocation` and Codex's `agents/openai.yaml` policy. Hosts without verified manual controls get a warning and explicit-invocation guidance. `--strict` turns compatibility warnings into errors.

When hosts share discovery roots, identical files have multiple owners. Conflicting output or visible variants fail the entire plan before writes. Identical skills in separate discoverable roots produce a warning; Promptherder does not assume every host deduplicates them.

## Commands

| Command | Purpose |
|---|---|
| `promptherder` | Plan and apply all selected targets |
| `install [targets...]` | Select targets interactively or explicitly; `none` disables all |
| `target list/add/remove` | Manage the saved selection |
| `list` | List built-in and user herd aliases |
| `pull <alias-or-url>` | Download and validate a replacement herd before installing it |
| `pull <alias-or-url> --locked` | Fetch the recorded immutable revision and verify its digest |
| `plan [--json]` | Preview outputs, content, origins, hashes, deletions, and diagnostics |
| `check [--strict]` | Validate deterministically without writing |
| `doctor` | Also inspect personal Codex/Claude instruction and skill overlap |
| `explain <id>` | Trace a selected rule or skill to its output |
| `recover` | Roll an interrupted apply back, preserving subsequent edits |

`--targets codex,claude` overrides target selection for read-only commands. `--dry-run` previews sync or selection changes without writing. `--include` filters rule filenames with comma-separated globs. JSON plan output includes full artifact content for review; errors go to stderr.

## Sources and settings

Downloaded herds live under `.promptherder/herds/<name>/`. Local sources live in `.promptherder/agent/`; `.promptherder/agents/` is also accepted for migration. Planning reads sources directly without merging downloaded files into local authoring directories.

```text
.promptherder/agent/
  rules/working-agreements.md
  rules/shell.md
  skills/my-task/
    SKILL.md
    references/guide.md
    scripts/check.sh
  workflows/plan.md
```

Rules without frontmatter are always active. Scoped rules use real YAML:

```yaml
---
id: shell-safety
activation: paths
paths: ["**/*.sh"]
required: false
---
Use set -Eeuo pipefail in shell scripts.
```

Supported activation modes are `always`, `paths`, `manual`, and `relevance`. Legacy `applyTo`, `globs`, and `trigger` fields are translated. `.promptherder/hard-rules.md` is loaded first as baseline content. Invalid YAML, duplicate IDs, conflicting sources, and unsupported required semantics fail validation.

Each skill needs a standard `SKILL.md` with a lowercase hyphenated name matching its directory, and a description. Resources are copied with relative paths and executable modes preserved. Markdown links into `references/`, `scripts/`, and `assets/` are checked. Uppercase host variants such as `CLAUDE.md` and `CODEX.md` remain supported as full replacements.

A herd may declare dependencies, invocation policy, and full host overlays in `herd.json`:

```json
{
  "name": "my-herd",
  "version": "1.0.0",
  "skills": {
    "my-task": {
      "requires": ["my-helper"],
      "invocation": "manual",
      "overlays": {"claude": "overlays/claude/my-task.md"}
    }
  }
}
```

Invocation is `manual` or `automatic`; omitted policy preserves skill metadata. Selected skills include their declared dependencies. Overlays must have valid skill frontmatter. Do not define an uppercase variant and a manifest overlay for the same host and skill.

Repository preferences live in `.promptherder/settings.json`:

```json
{
  "agents": ["codex", "claude"],
  "skills": ["my-task"],
  "overrides": ["rules/working-agreements.md"],
  "project_doc_max_bytes": 32768
}
```

Missing or null `agents` means unconfigured; `[]` means no targets. Missing or null `skills` selects all; `[]` selects no source skills. Manual/relevance rules remain governed by rule selection. Local content differing from a herd requires an explicit source-path override. The project-document budget is a configured ceiling, not proof of the remaining runtime context budget.

## Workflow migration

Legacy workflows become manual native skills named `workflow-<name>`. Existing `command_prefix` settings prefix this ID. For example, `workflows/plan.md` becomes:

- Codex: `$workflow-plan`
- Claude Code: `/workflow-plan`

Generated workflow text explains legacy `/plan`-style references and bounds instructions to their active phase. It does not create legacy slash-command aliases or grant additional authorization. Automatic pipeline helpers should be ordinary callable skills, or explicitly declared `automatic`, with dependencies recorded in `herd.json`.

## Upgrade from 0.x

1. Build or install the 1.0.0 binary and explicitly select your targets.
2. Run `promptherder plan --adopt --json` and review the proposed content and deletions.
3. Run `promptherder --adopt` to perform the reviewed migration.
4. Run `promptherder check` and `promptherder doctor`.

Legacy generated files lack trustworthy content hashes, so replacing them requires `--adopt`. Existing unmanaged `AGENTS.md` and `CLAUDE.md` content is preserved during adoption. Later user edits to tracked output cause conflicts, including during cleanup. Removing a target restores preserved baseline text and retains files still needed by another target.

The version 3 manifest records hashes, modes, origins, and every consuming target. Apply uses per-file atomic writes and a recovery journal; a whole multi-file sync is not filesystem-atomic. If interrupted, run `recover` before syncing. Recovery refuses to overwrite subsequent user edits or race a live sync.

`.promptherder/lock.json` records content digests plus repository, revision, and version when available. Review updated herds with `plan --update-lock`, then sync with `--update-lock`. `pull --locked` requires a recorded full Git revision; manually copied herds can have content locks without a remotely reproducible revision.

Commit sources, settings, lockfile, and manifest. Commit generated native artifacts too if remote agents need them, or regenerate deterministically before the host starts. Blanket-ignoring `.agents/` or `.claude/` can hide skills from cloud agents. Do not commit `.promptherder/apply-journal.json` or `.promptherder/sync.lock`.

## Herds and architecture

Built-in aliases include [Compound V](https://github.com/shermanhuman/compound-v), [Grugg](https://github.com/shermanhuman/grugg), and [Oh](https://github.com/shermanhuman/oh). `list` exposes their descriptions and user alias configuration location. Pulling these herds preserves their authored policies and dependencies; it does not rewrite their external tool implementations.

See [architecture and implementation status](ARCHITECTURE-PROPOSAL.md) and [contributor guide](CONTRIBUTING.md). This release implements repository compilation and explicit Codex/Claude personal installation. Herd content refactoring and model-specific tuning remain separate work; static checks do not establish live agent adherence.

## Credits

- [obra/superpowers](https://github.com/obra/superpowers) — original Compound V methodology
- [Antigravity adaptation](https://github.com/anthonylee991/gemini-superpowers-antigravity)
