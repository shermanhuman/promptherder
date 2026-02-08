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
		cfg.Logger.Info("no source directory found", "dir", antigravitySource)
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

		// Skip agent-generated files (e.g. stack.md, structure.md).
		baseName := filepath.Base(rel)
		if m.isGenerated(baseName) {
			targetPath := filepath.Join(cfg.RepoPath, antigravityTarget, rel)
			if _, err := os.Stat(targetPath); err == nil {
				cfg.Logger.Info("skipping generated file", "file", relSlash)
				return nil
			}
			// If the generated file doesn't exist yet, allow writing it.
		}

		// Read source.
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		targetPath := filepath.Join(cfg.RepoPath, antigravityTarget, rel)
		targetRel := filepath.ToSlash(filepath.Join(antigravityTarget, relSlash))

		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "target", targetRel, "source", relSlash)
		} else {
			if err := writeFile(targetPath, data); err != nil {
				return err
			}
			cfg.Logger.Info("synced", "target", targetRel, "source", relSlash)
		}

		installed = append(installed, targetRel)
		return nil
	})

	return installed, err
}

// stripFrontmatterForCopy removes YAML frontmatter for file copy targets that
// don't need it (e.g. Antigravity reads skills/workflows natively).
// Currently unused but available for future targets that need clean content.
func stripFrontmatterForCopy(data []byte) []byte {
	s := string(data)
	if !strings.HasPrefix(s, "---") {
		return data
	}
	idx := strings.Index(s[3:], "---")
	if idx == -1 {
		return data
	}
	return []byte(strings.TrimLeft(s[idx+6:], "\r\n"))
}
