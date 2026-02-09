package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	antigravitySource = ".promptherder/agent"
	antigravityTarget = ".agent"
)

// AntigravityTarget implements the Target interface for Google Antigravity.
// It copies files from .promptherder/agent/ to .agent/, preserving directory structure.
type AntigravityTarget struct{}

func (t AntigravityTarget) Name() string { return "antigravity" }

func (t AntigravityTarget) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
	srcRoot := filepath.Join(cfg.RepoPath, filepath.FromSlash(antigravitySource))

	if _, err := os.Stat(srcRoot); os.IsNotExist(err) {
		cfg.Logger.Debug("no source directory found", "dir", antigravitySource)
		return nil, nil
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
