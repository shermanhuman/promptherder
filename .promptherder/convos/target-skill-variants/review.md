# Review: Target-Specific Skill Variants

## Blockers

None.

## Majors

None.

## Minors

### M1: `skillVariantFiles` lives in `antigravity.go` but is a cross-target concern

**File**: `internal/app/antigravity.go:19-22`

The `skillVariantFiles` map defines variant filenames for _all_ targets (including Copilot), but lives in the Antigravity implementation file. If someone adds a new target, they'd need to know to update `antigravity.go` — which is non-obvious.

**Options**: (a) Move to a shared file like `target.go` or a new `variants.go`. (b) Leave as-is and rely on CONTRIBUTING.md docs (which already mention this). (c) Each target defines its own variant list.

**Recommendation**: Option (a), low effort, reduces surprise. But not blocking.

### M2: Log source path is misleading when variant is installed

**File**: `internal/app/antigravity.go:99-107`

When ANTIGRAVITY.md is installed as SKILL.md, the `source` field in the log says `skills/compound-v-parallel/SKILL.md` because `rel` was already rewritten. The actual source was `ANTIGRAVITY.md`. This makes debugging harder — you can't tell from the log whether the variant or generic was used.

```
synced target=.agent/skills/compound-v-parallel/SKILL.md source=skills/compound-v-parallel/SKILL.md
```

**Fix**: Track the original `relSlash` before rewriting and use it in the log's `source` field. Something like:

```go
sourceRel := relSlash  // capture before rewrite
// ... (rewrite rel to SKILL.md)
cfg.Logger.Info("synced", "target", targetRel, "source", sourceRel)
```

### M3: TOCTOU window in Antigravity variant check

**File**: `internal/app/antigravity.go:74-78`

The `os.Stat(variantPath)` check on line 74 has a theoretical TOCTOU (time-of-check-time-of-use) race: the variant file could be deleted between the Stat and the Walk reaching ANTIGRAVITY.md. However, this is a CLI tool running single-threaded on local files — the race is not practically exploitable. **No action needed**, but worth noting.

## Nits

### N1: `isInSkillDir` could use `strings.HasPrefix`

**File**: `internal/app/antigravity.go:119-122`

```go
func isInSkillDir(relSlash string) bool {
    parts := strings.SplitN(relSlash, "/", 3)
    return len(parts) >= 3 && parts[0] == "skills"
}
```

This works but allocates a slice. A simpler check:

```go
func isInSkillDir(relSlash string) bool {
    return strings.HasPrefix(relSlash, "skills/") && strings.Count(relSlash, "/") >= 2
}
```

Both are equivalent for the expected input. Current implementation is clear and correct. Pure style nit.

### N2: Copilot tests missing `t.Parallel()`

**File**: `internal/app/copilot_test.go:823,847,866`

The three new Copilot variant tests don't call `t.Parallel()`, unlike the Antigravity variant tests which do. Consistent parallel execution is preferred.

### N3: CONTRIBUTING.md uses `AntigravityTarget` description "Simple 1:1 copy"

**File**: `CONTRIBUTING.md:287`

```
| `AntigravityTarget` | `antigravity.go` | Mirror directory tree | Simple 1:1 copy (no translation needed) |
```

With the variant logic, it's no longer a pure "simple 1:1 copy" — it now does variant selection. Consider updating to "Mirror with variant selection" or similar.

## Summary + Next Actions

**Overall**: Clean, well-tested implementation. The variant selection logic is minimal and correct. `filepath.Walk` guarantees lexical order, so ANTIGRAVITY.md is always visited before SKILL.md — the variant installs as SKILL.md first, then the generic is correctly skipped. 8 new tests cover all important scenarios. CONTRIBUTING.md documentation is thorough and includes a concrete example for extending the pattern.

**Next actions (ordered by priority)**:

1. Fix M2 (log source path) — quick, improves debuggability
2. Fix N2 (add `t.Parallel()` to Copilot tests) — one-liner each
3. Consider M1 (move `skillVariantFiles` to shared file) — optional
4. Consider N3 (update CONTRIBUTING table description) — optional
