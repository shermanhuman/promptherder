# Contributing to promptherder

## Adding a New Target Agent

promptherder uses a `Target` interface to support multiple AI coding agents. Here's how to add one.

The interface is minimal — just two methods:

```go
type Target interface {
    Name() string
    Install(ctx context.Context, cfg TargetConfig) ([]string, error)
}
```

`Install` reads from `.promptherder/agent/` and writes to wherever your agent expects its config. It returns repo-relative paths of everything it wrote (for manifest tracking and stale cleanup).

### Example: CopilotTarget

CopilotTarget transforms content from the shared `.promptherder/agent/` format into the formats Copilot expects.

#### What Copilot reads

Copilot expects three types of files:

| Copilot file                             | Purpose                                                     |
| ---------------------------------------- | ----------------------------------------------------------- |
| `.github/copilot-instructions.md`        | Always-on repo-wide instructions (single concatenated file) |
| `.github/instructions/*.instructions.md` | Path-scoped instructions with `applyTo` frontmatter         |
| `.github/prompts/*.prompt.md`            | Reusable slash commands in Copilot Chat                     |

#### What promptherder stores

```
.promptherder/agent/
├── rules/
│   ├── 00-general.md          # No frontmatter → repo-wide
│   ├── shell-safety.md        # Has applyTo: "**/*.sh" → path-scoped
│   └── docker.md              # Has applyTo: "Dockerfile*" → path-scoped
├── workflows/
│   ├── brainstorm.md          # Has description frontmatter
│   └── review.md
└── skills/
    └── compound-v-parallel/
        ├── SKILL.md            # Generic skill (name + description frontmatter)
        ├── ANTIGRAVITY.md      # Antigravity-specific variant (optional)
        └── COPILOT.md          # Copilot-specific variant (optional)
```

#### The translation

```
Source                                     →  Output
─────────────────────────────────────────────────────────────────────
rules/00-general.md (no applyTo)           →  .github/copilot-instructions.md (concatenated)
rules/shell-safety.md (applyTo: **/*.sh)   →  .github/instructions/shell-safety.instructions.md
rules/docker.md (applyTo: Dockerfile*)     →  .github/instructions/docker.instructions.md
workflows/brainstorm.md                    →  .github/prompts/brainstorm.prompt.md
workflows/review.md                        →  .github/prompts/review.prompt.md
skills/compound-v-parallel/COPILOT.md      →  .github/prompts/compound-v-parallel.prompt.md  (variant preferred)
skills/compound-v-parallel/SKILL.md        →  .github/prompts/compound-v-parallel.prompt.md  (fallback)
```

#### Step 1: The target struct and registration (`copilot.go`)

```go
type CopilotTarget struct {
    SourceDir string   // override source dir; defaults to ".promptherder/agent/rules"
    Include   []string // glob patterns for source files
}

func (t CopilotTarget) Name() string { return "copilot" }
```

Registered in `cmd/promptherder/main.go`:

```go
copilot := app.CopilotTarget{Include: cfg.Include}
allTargets := []app.Target{copilot, antigravity, compoundV}

case "copilot":
    runErr = app.RunTarget(ctx, copilot, cfg)
```

And in `extractSubcommand`:

```go
known := map[string]bool{
    "copilot":     true,
    "antigravity": true,
    "compound-v":  true,
}
```

#### Step 2: The Install method (`copilot.go`)

```go
func (t CopilotTarget) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
    var written []string

    // --- Phase 1: Rules → copilot-instructions.md + .instructions.md files ---

    // readSources() discovers all .md files and parses their frontmatter.
    // Each sourceFile has: Name, ApplyTo (from frontmatter), Body (content after frontmatter).
    sources, err := readSources(cfg.RepoPath, defaultSourceDir, t.Include)
    if err != nil {
        return nil, err
    }

    if len(sources) > 0 {
        // buildCopilotPlan() does the actual translation:
        //   - Rules WITHOUT applyTo → concatenated into one copilot-instructions.md
        //   - Rules WITH applyTo    → each gets its own .instructions.md with applyTo frontmatter
        plan := buildCopilotPlan(cfg.RepoPath, defaultSourceDir, sources)
        written, err = writeItems(ctx, cfg, plan, written)
        if err != nil {
            return written, err
        }
    }

    // --- Phase 2: Workflows → .github/prompts/*.prompt.md ---

    // Reads .promptherder/agent/workflows/*.md
    // Strips Antigravity annotations (// turbo, // turbo-all)
    // Rewrites frontmatter: description → mode: "agent" + description
    promptItems, err := buildCopilotPrompts(cfg.RepoPath)
    if err != nil {
        return written, err
    }

    // --- Phase 3: Skills → .github/prompts/*.prompt.md (same conversion) ---

    skillItems, err := buildCopilotSkillPrompts(cfg.RepoPath)
    if err != nil {
        return written, err
    }
    promptItems = append(promptItems, skillItems...)

    written, err = writeItems(ctx, cfg, promptItems, written)
    if err != nil {
        return written, err
    }

    return written, nil
}
```

#### Step 3: The translation functions

**Rule concatenation** — `buildCopilotPlan()` splits rules by whether they have `applyTo`:

```go
// Rules WITHOUT applyTo → combined into one file
var copilotParts [][]byte
for _, s := range sources {
    if s.ApplyTo == "" {
        copilotParts = append(copilotParts, s.Body)
    }
}
plan = append(plan, planItem{
    Target:  filepath.Join(repoPath, ".github/copilot-instructions.md"),
    Content: concatWithHeader("<!-- Auto-generated by promptherder -->", copilotParts),
})

// Rules WITH applyTo → each gets its own .instructions.md
for _, s := range sources {
    if s.ApplyTo == "" {
        continue
    }
    header := fmt.Sprintf("---\napplyTo: %q\n---\n", s.ApplyTo)
    plan = append(plan, planItem{
        Target:  filepath.Join(repoPath, ".github/instructions", s.Name+".instructions.md"),
        Content: append([]byte(header), s.Body...),
    })
}
```

**Workflow/skill → prompt conversion** — `convertWorkflowToPrompt()` rewrites frontmatter and strips agent-specific annotations:

```go
// Input (Antigravity workflow):
// ---
// description: Run a review pass.
// ---
// // turbo-all
//
// # Review
// Do the review.

// Output (Copilot prompt):
// ---
// mode: "agent"
// description: "Run a review pass."
// ---
// <!-- Auto-generated by promptherder -->
//
// # Review
// Do the review.
```

The `// turbo` and `// turbo-all` annotations are stripped because they're Antigravity-specific (they control auto-approval of commands).

#### Step 4: Tests (`copilot_test.go`)

The Copilot target has tests covering every scenario:

```go
func TestRunCopilot_ConcatenatesRepoWideRules(t *testing.T) {
    // Two rules without applyTo → both end up in copilot-instructions.md
    dir := t.TempDir()
    rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
    mustMkdir(t, rulesDir)
    mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n\n- Be helpful.\n")
    mustWrite(t, filepath.Join(rulesDir, "01-style.md"), "# Style\n\n- Use gofmt.\n")

    RunCopilot(context.Background(), Config{RepoPath: dir, Logger: logger})

    content, _ := os.ReadFile(filepath.Join(dir, ".github", "copilot-instructions.md"))
    assertContains(t, content, "Be helpful")
    assertContains(t, content, "Use gofmt")
    assertContains(t, content, "Auto-generated by promptherder")
}

func TestRunCopilot_ScopedRulesGetOwnFiles(t *testing.T) {
    // Rule with applyTo → gets its own .instructions.md
    dir := t.TempDir()
    rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
    mustMkdir(t, rulesDir)
    mustWrite(t, filepath.Join(rulesDir, "shell.md"),
        "---\napplyTo: \"**/*.sh\"\n---\n\n- Use set -e.\n")

    RunCopilot(context.Background(), Config{RepoPath: dir, Logger: logger})

    content, _ := os.ReadFile(filepath.Join(dir, ".github", "instructions", "shell.instructions.md"))
    assertContains(t, content, `applyTo: "**/*.sh"`)
    assertContains(t, content, "Use set -e")
}

func TestConvertWorkflowToPrompt_Basic(t *testing.T) {
    input := []byte("---\ndescription: Run a review pass.\n---\n\n# Review\n\nDo the review.\n")
    result := convertWorkflowToPrompt("test/workflows", "review.md", input)

    assertContains(t, result, `mode: "agent"`)
    assertContains(t, result, `description: "Run a review pass."`)
    assertContains(t, result, "# Review")
}

func TestConvertWorkflowToPrompt_StripsTurbo(t *testing.T) {
    input := []byte("---\ndescription: Execute.\n---\n\n// turbo-all\n\n# Execute\n")
    result := convertWorkflowToPrompt("test/workflows", "execute.md", input)

    assertNotContains(t, result, "// turbo-all")
    assertContains(t, result, "# Execute")
}

func TestRunCopilot_DryRun(t *testing.T) {
    dir := t.TempDir()
    rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
    mustMkdir(t, rulesDir)
    mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")

    RunCopilot(context.Background(), Config{RepoPath: dir, DryRun: true, Logger: logger})

    // File should NOT exist in dry-run mode.
    if _, err := os.Stat(filepath.Join(dir, ".github")); !os.IsNotExist(err) {
        t.Error("dry-run should not write files")
    }
}
```

### Available Helpers

| Helper                                        | What it does                                                       | Defined in    |
| --------------------------------------------- | ------------------------------------------------------------------ | ------------- |
| `readSources(repoPath, srcDir, include)`      | Discovers + parses all `.md` files, extracts `applyTo` frontmatter | `copilot.go`  |
| `concatWithHeader(header, parts)`             | Joins body parts with a leading comment                            | `copilot.go`  |
| `writeFile(path, content)`                    | Atomic write via temp file + rename                                | `copilot.go`  |
| `writeItems(ctx, cfg, items, written)`        | Batch write with dry-run + context cancellation                    | `copilot.go`  |
| `readManifest(repoPath, logger)`              | Load previous manifest (for generated file checks)                 | `manifest.go` |
| `convertWorkflowToPrompt(srcDir, name, data)` | Rewrite frontmatter + strip annotations                            | `copilot.go`  |

### Existing targets as reference

| Target              | File             | Pattern                                            | Good example of...                                       |
| ------------------- | ---------------- | -------------------------------------------------- | -------------------------------------------------------- |
| `CopilotTarget`     | `copilot.go`     | Rules → concatenated + per-rule + prompt files     | Complex multi-output target with frontmatter translation |
| `AntigravityTarget` | `antigravity.go` | Mirror directory tree with skill variant selection | 1:1 copy with target-specific skill preference           |

## Target-Specific Skill Variants

Skills can have target-specific variants that replace the generic `SKILL.md` when installing to a particular agent target.

### File structure

```
compound-v/skills/compound-v-parallel/
├── SKILL.md              ← Generic (methodology, no agent-specific APIs)
├── ANTIGRAVITY.md        ← Replaces SKILL.md when installing to Antigravity
└── COPILOT.md            ← Replaces SKILL.md when installing to Copilot (optional)
```

### How it works

1. **Herds** are pulled via `promptherder pull <url>` into `.promptherder/herds/<name>/`. The merge step copies all herd content into `.promptherder/agent/skills/<name>/`.
2. **Antigravity** walks `.promptherder/agent/` and for each skill directory:
   - If `ANTIGRAVITY.md` exists → installs it as `<name>/SKILL.md` at `.agent/`
   - If `COPILOT.md` exists → skips it (not for this target)
   - If no variant exists → installs generic `SKILL.md`
3. **Copilot** checks each skill directory for `COPILOT.md` before falling back to `SKILL.md`.

Variant filenames use **uppercase target names** to match the `SKILL.md` convention and stand out in directory listings.

### When to create a variant

Create a variant when the skill needs **agent-specific API details**. Example:

| Content                                       | Where it belongs                   |
| --------------------------------------------- | ---------------------------------- |
| "Group independent steps into batches"        | `SKILL.md` (methodology)           |
| "Use `waitForPreviousTools: false`"           | `ANTIGRAVITY.md` (Antigravity API) |
| "Use concurrent tool calls in the same block" | `COPILOT.md` (Copilot API)         |

### Adding a variant for a new target

If you add a new target (e.g., `CursorTarget`):

1. Add the variant filename to `SkillVariantFiles` in `target.go`:

```go
var SkillVariantFiles = map[string]string{
    "ANTIGRAVITY.md": "antigravity",
    "COPILOT.md":     "copilot",
    "CURSOR.md":      "cursor",  // ← add this
}
```

2. In your target's Install method, check for `CURSOR.md` before falling back to `SKILL.md`.
3. Create `compound-v/skills/<skill-name>/CURSOR.md` for skills that need Cursor-specific instructions.

## Guidelines

- **Idempotency**: Running any command multiple times must produce the same result.
- **Manifest tracking**: All written files must be tracked in the manifest so stale files can be cleaned up. Return repo-relative paths from `Install`.
- **Generated file protection**: Check `manifest.isGenerated()` before overwriting files the agent may have created (e.g., `stack.md`, `structure.md`).
- **Atomic writes**: Use `writeFile()` which wraps `AtomicWriter` for safe file writes.
- **Context cancellation**: Check `ctx.Err()` periodically in loops for graceful shutdown.
- **Dry-run support**: When `cfg.DryRun` is true, log what would happen but don't write. Still return the paths.
- **Skill variants**: When adding a new target, register its variant filename in `SkillVariantFiles` (in `target.go`) and implement variant preference in the target's Install method.

## Herds

A **herd** is a versionable collection of rules, skills, and workflows pulled from any git URL.

### Architecture

```
promptherder pull <url>
  → git clone → .promptherder/herds/<name>/

promptherder (bare)
  → discover herds in .promptherder/herds/*/
  → merge herds → .promptherder/agent/
  → copilot: .promptherder/agent/ → .github/
  → antigravity: .promptherder/agent/ → .agent/
```

### herd.json

Each herd repo must contain a `herd.json` at its root:

```json
{
  "name": "my-herd"
}
```

### Conflict resolution

If two herds provide the same file path (e.g. `rules/foo.md`), `mergeHerds` returns an error naming both herds and the conflicting path.

### Key files

| File        | Purpose                                          |
| ----------- | ------------------------------------------------ |
| `herd.go`   | `HerdMeta`, `discoverHerds`, `mergeHerds`        |
| `pull.go`   | `Pull` — git clone/update + herd.json validation |
| `runner.go` | `RunAll` — merge herds before target install     |
