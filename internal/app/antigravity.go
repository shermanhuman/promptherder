package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	antigravitySource       = ".promptherder/agents"
	antigravitySourceLegacy = ".promptherder/agent"
	antigravityTarget       = ".agents"
	antigravityTargetLegacy = ".agent"
)

// AntigravityTarget implements the Target interface for Google Antigravity.
// It copies files from .promptherder/agents/ to .agents/, preserving directory structure.
//
// Legacy path support: if .promptherder/agents/ does not exist but .promptherder/agent/
// does, the legacy source is used automatically (backward compat for repos that haven't
// renamed yet). Similarly, if .agent/ exists without .agents/, an interactive migration
// prompt is offered.
type AntigravityTarget struct{}

func (t AntigravityTarget) Name() string { return "antigravity" }

func (t AntigravityTarget) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
	// Resolve source: prefer new path, fall back to legacy.
	srcRoot := filepath.Join(cfg.RepoPath, filepath.FromSlash(antigravitySource))
	if _, err := os.Stat(srcRoot); os.IsNotExist(err) {
		legacySrc := filepath.Join(cfg.RepoPath, filepath.FromSlash(antigravitySourceLegacy))
		if _, err := os.Stat(legacySrc); err == nil {
			cfg.Logger.Debug("using legacy source directory", "dir", antigravitySourceLegacy)
			srcRoot = legacySrc
		}
	}

	if _, err := os.Stat(srcRoot); os.IsNotExist(err) {
		cfg.Logger.Debug("no source directory found", "dir", antigravitySource)
		return nil, nil
	}

	// Offer interactive migration if legacy target exists and new target does not.
	if err := maybeMigrateLegacyTarget(cfg); err != nil {
		cfg.Logger.Warn("migration failed (continuing)", "err", err)
	}

	// Load manifest to check for generated files we must not overwrite.
	m := readManifest(cfg.RepoPath, cfg.Logger)

	var installed []string
	err := filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Compute relative path from source root.
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return fmt.Errorf("rel path: %w", err)
		}
		relSlash := filepath.ToSlash(rel)
		baseName := filepath.Base(rel)
		sourceSlash := relSlash // preserve original for logging

		// --- Skill variant logic ---
		// If this file is in a skills/*/ directory, apply variant selection.
		if isInSkillDir(relSlash) {
			if targetName, isVariant := SkillVariantFiles[baseName]; isVariant {
				if targetName != "antigravity" {
					// Skip other targets' variant files (e.g. COPILOT.md).
					return nil
				}
				// This is our variant — install it as SKILL.md.
				rel = filepath.Join(filepath.Dir(rel), "SKILL.md")
				relSlash = filepath.ToSlash(rel)
			} else if baseName == "SKILL.md" {
				// Check if our variant file exists; if so, skip the generic.
				variantPath := filepath.Join(filepath.Dir(path), "ANTIGRAVITY.md")
				if _, err := os.Stat(variantPath); err == nil {
					cfg.Logger.Debug("skipping generic skill (variant exists)", "file", relSlash)
					return nil
				}
			}
		}

		// Skip agent-generated files (e.g. stack.md, structure.md).
		if m.isGenerated(baseName) {
			targetPath := filepath.Join(cfg.RepoPath, antigravityTarget, rel)
			if _, err := os.Stat(targetPath); err == nil {
				cfg.Logger.Debug("skipping generated file", "file", relSlash)
				return nil
			}
			// If the generated file doesn't exist yet, allow writing it.
		}

		// Read source.
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		// Apply command prefix to workflow files (not skills).
		outputRel := rel
		if isInWorkflowDir(relSlash) {
			dir := filepath.Dir(rel)
			base := filepath.Base(rel)
			outputRel = filepath.Join(dir, cfg.Settings.PrefixCommand(base))
		}

		targetPath := filepath.Join(cfg.RepoPath, antigravityTarget, outputRel)
		targetRel := filepath.ToSlash(filepath.Join(antigravityTarget, filepath.ToSlash(outputRel)))

		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "target", targetRel, "source", sourceSlash)
		} else {
			if err := writeFile(targetPath, data); err != nil {
				return err
			}
			cfg.Logger.Info("synced", "target", targetRel, "source", sourceSlash)
		}

		installed = append(installed, targetRel)
		return nil
	})
	if err != nil {
		return installed, err
	}

	// Copy hard-rules.md if it exists.
	hardRulesPath := filepath.Join(cfg.RepoPath, filepath.FromSlash(hardRulesFile))
	if data, err := os.ReadFile(hardRulesPath); err == nil {
		targetPath := filepath.Join(cfg.RepoPath, antigravityTarget, "rules", "hard-rules.md")
		targetRel := filepath.ToSlash(filepath.Join(antigravityTarget, "rules", "hard-rules.md"))
		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "target", targetRel, "source", hardRulesFile)
		} else {
			if err := writeFile(targetPath, data); err != nil {
				return installed, err
			}
			cfg.Logger.Info("synced", "target", targetRel, "source", hardRulesFile)
		}
		installed = append(installed, targetRel)
	}

	return installed, err
}

// maybeMigrateLegacyTarget checks if the legacy .agent/ target exists without
// .agents/ and, if running interactively, prompts the user to migrate both the
// target directory and the source staging directory.
//
// If the user declines or the environment is non-interactive, migration is skipped
// and a note is logged. Install proceeds to .agents/ regardless.
func maybeMigrateLegacyTarget(cfg TargetConfig) error {
	newTarget := filepath.Join(cfg.RepoPath, antigravityTarget)
	legacyTarget := filepath.Join(cfg.RepoPath, antigravityTargetLegacy)

	// No migration needed if new target already exists or legacy doesn't.
	if _, err := os.Stat(newTarget); err == nil {
		return nil
	}
	if _, err := os.Stat(legacyTarget); os.IsNotExist(err) {
		return nil
	}

	// Non-interactive or dry-run: skip silently with a log note.
	if cfg.DryRun || !isInteractive() {
		cfg.Logger.Info("legacy .agent/ detected — run promptherder interactively to migrate to .agents/")
		return nil
	}

	fmt.Println()
	fmt.Println("⚠  Legacy .agent/ directory detected.")
	fmt.Println()
	fmt.Println("   Antigravity now uses .agents/ as its default path (since v1.14).")
	fmt.Println("   Promptherder will install files to .agents/ going forward.")
	fmt.Println()
	fmt.Println("   Migrating will rename:")
	fmt.Printf("     %s/  →  %s/\n", antigravityTargetLegacy, antigravityTarget)

	// Also offer to rename the source staging dir if the legacy one is in use.
	newSrc := filepath.Join(cfg.RepoPath, antigravitySource)
	legacySrc := filepath.Join(cfg.RepoPath, antigravitySourceLegacy)
	migrateSrc := false
	if _, err := os.Stat(newSrc); os.IsNotExist(err) {
		if _, err := os.Stat(legacySrc); err == nil {
			migrateSrc = true
			fmt.Printf("     .promptherder/agent/  →  .promptherder/agents/\n")
		}
	}

	fmt.Println()
	fmt.Println("   This will appear as a rename in git — expected, since you're")
	fmt.Println("   already updating your agent configuration at this point.")
	fmt.Println()
	fmt.Print("   Migrate now? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		// EOF or read error — treat as no.
		fmt.Println()
		cfg.Logger.Info("migration skipped (no input)")
		return nil
	}

	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		cfg.Logger.Info("migration skipped — .agent/ will remain as an orphaned directory")
		return nil
	}

	// Perform migration. Non-fatal on failure — install continues to .agents/ regardless.
	if err := os.Rename(legacyTarget, newTarget); err != nil {
		fmt.Fprintf(os.Stderr, "\n   \u26a0\u26a0  Migration failed: %v\n   Files remain in .agent/ — re-run to retry.\n\n", err)
		return nil
	}
	cfg.Logger.Info("migrated", "from", antigravityTargetLegacy, "to", antigravityTarget)

	if migrateSrc {
		if err := os.Rename(legacySrc, newSrc); err != nil {
			// Non-fatal: log and continue. Install will fall back to reading new target.
			cfg.Logger.Info("could not rename source dir (non-fatal)", "err", err)
		} else {
			cfg.Logger.Info("migrated", "from", ".promptherder/agent/", "to", ".promptherder/agents/")
		}
	}

	fmt.Println()
	return nil
}

// isInteractive returns true if stdin is connected to a terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// isInSkillDir returns true if the slash-separated relative path is inside a
// skills/*/ directory (e.g. "skills/compound-v-tdd/SKILL.md").
func isInSkillDir(relSlash string) bool {
	return strings.HasPrefix(relSlash, "skills/") && strings.Count(relSlash, "/") >= 2
}

// isInWorkflowDir returns true if the slash-separated relative path is inside
// the workflows/ directory (e.g. "workflows/plan.md").
func isInWorkflowDir(relSlash string) bool {
	return strings.HasPrefix(relSlash, "workflows/")
}
