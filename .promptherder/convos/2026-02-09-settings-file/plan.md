# Plan: Settings File

> **Status**: complete

## Goal

Add a `.promptherder/settings.json` file that controls command prefix naming and optional git auto-commit after syncing artifacts. Document the config in `-h` output.

## Settings file format

```json
{
  "command_prefix": "v-",
  "command_prefix_enabled": true,
  "git_auto_commit": false
}
```

- `command_prefix` — string prepended to output filenames for workflows/prompts (default: `""`)
- `command_prefix_enabled` — toggle (default: `false`)
- `git_auto_commit` — run `git add + commit` on managed files after sync (default: `false`)

All fields optional. Missing file = all defaults. Fully backward compatible.

## Plan

### 1. Define settings struct and loader

- **Files:** `internal/app/settings.go`, `internal/app/settings_test.go`
- **Change:**
  - `Settings` struct: `CommandPrefix`, `CommandPrefixEnabled`, `GitAutoCommit`
  - `LoadSettings(repoPath)` reads `.promptherder/settings.json`, returns defaults if missing
  - Validation: empty prefix + enabled = treated as disabled
  - Test: missing file → defaults; valid file → parsed; partial file → defaults for missing fields; empty prefix + enabled → disabled
- **Verify:** `go test ./internal/app/ -run TestSettings -v`

### 2. Wire settings into TargetConfig and runner

- **Files:** `internal/app/target.go`, `internal/app/runner.go`
- **Change:**
  - Add `Settings` field to `TargetConfig`
  - `setupRunner` calls `LoadSettings` and populates the config
- **Verify:** `go test ./... -count=1`

### 3. Apply command prefix in Copilot target

- **Files:** `internal/app/copilot.go`, `internal/app/copilot_test.go`
- **Change:**
  - In `buildCopilotPrompts`: if prefix enabled, `plan.prompt.md` → `v-plan.prompt.md`
  - In `buildCopilotSkillPrompts`: skills NOT prefixed (they already have unique names)
  - Test: with prefix → `v-plan.prompt.md`; without → `plan.prompt.md`
- **Verify:** `go test ./internal/app/ -run TestCopilot -v`

### 4. Apply command prefix in Antigravity target

- **Files:** `internal/app/antigravity.go`, `internal/app/antigravity_test.go`
- **Change:**
  - Workflow files: prepend prefix if enabled (`plan.md` → `v-plan.md`)
  - Skills: NOT prefixed
  - Test: with prefix → `v-plan.md`; without → `plan.md`
- **Verify:** `go test ./internal/app/ -run TestAntigravity -v`

### 5. Implement git auto-commit

- **Files:** `internal/app/commit.go`, `internal/app/commit_test.go`
- **Change:**
  - `gitAutoCommit(repoPath, files, logger)` runs `git add <files>; git commit -m "promptherder: sync"`
  - Gracefully warns if git binary not found
  - Test in temp dir with real git repo
- **Verify:** `go test ./internal/app/ -run TestAutoCommit -v`

### 6. Call auto-commit from runner

- **Files:** `internal/app/runner.go`
- **Change:**
  - After `persistAndClean`, if `GitAutoCommit == true`, call `gitAutoCommit`
  - Log result with pretty output
- **Verify:** `go test ./... -count=1` + manual run

### 7. Add help text with config documentation

- **Files:** `cmd/promptherder/main.go`
- **Change:**
  - Replace default `flag.Usage` with a custom usage function
  - Include: usage line, subcommands, flags, and a **Settings** section documenting `.promptherder/settings.json` fields with defaults
  - Target output:

    ```
    promptherder — sync agent configuration across AI coding tools

    Usage:
      promptherder [flags]              Sync all targets
      promptherder <target> [flags]     Sync a single target (copilot, antigravity)
      promptherder pull <git-url>       Install a herd from a Git repository

    Flags:
      -dry-run     Show actions without writing files
      -include     Comma-separated glob patterns to include (default: all)
      -v           Verbose logging (structured output to stderr)
      -version     Print version and exit

    Settings (.promptherder/settings.json):
      command_prefix           Prefix for command filenames (default: "")
      command_prefix_enabled   Enable prefix (default: false)
      git_auto_commit          Auto-commit managed files after sync (default: false)

    Examples:
      promptherder                              Sync all targets
      promptherder pull https://github.com/user/herd
      promptherder copilot -dry-run             Preview copilot sync
    ```

- **Verify:** `promptherder -h`

## What this builds

### Happy path

1. User creates `.promptherder/settings.json` with `"command_prefix": "v-"` and `"command_prefix_enabled": true`
2. Runs `promptherder`
3. Output:
   ```
   ✓ loaded settings (prefix: v-, auto-commit: off)
   ✓ synced .agent/workflows/v-plan.md ← workflows/plan.md
   ✓ synced .github/prompts/v-plan.prompt.md sources=[plan]
   ```
4. Skills remain unprefixed: `compound-v-tdd/SKILL.md`
5. `promptherder -h` shows full usage with settings documentation

### Filesystem tree (with prefix enabled)

```
.agent/workflows/
├── v-execute.md       ← prefixed
├── v-plan.md          ← prefixed
├── v-review.md        ← prefixed
└── v-rule.md          ← prefixed

.github/prompts/
├── v-execute.prompt.md    ← prefixed
├── v-plan.prompt.md       ← prefixed
├── compound-v-tdd.prompt.md  ← NOT prefixed (skill)
└── ...
```

## Risks & mitigations

- **Breaking existing repos** → Missing file = defaults. Zero impact.
- **Git not available** → Auto-commit logs warning and continues.
- **Empty prefix + enabled** → Treated as disabled (validated in loader).
- **Stale files after prefix change** → Manifest cleanup handles this automatically.

## Rollback plan

Delete `settings.go`, `commit.go`, revert the `Settings` field additions. No schema changes.

## References

- Full plan: `.promptherder/convos/2026-02-09-settings-file/plan.md`
- Decisions: `.promptherder/convos/2026-02-09-settings-file/decisions.md`
