package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	compoundVPrefix = "compound-v"
	compoundVTarget = ".promptherder/agent" // install to source of truth, not .agent/
)

// CompoundVTarget implements the Target interface for the embedded Compound V
// methodology. It extracts files from the embedded FS to .promptherder/agent/
// (the source of truth). The user then runs `promptherder` or
// `promptherder antigravity` to fan out to .agent/.
type CompoundVTarget struct {
	FS fs.FS // embedded filesystem (e.g. promptherder.CompoundVFS)
}

func (t CompoundVTarget) Name() string { return "compound-v" }

func (t CompoundVTarget) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
	if t.FS == nil {
		return nil, fmt.Errorf("compound-v: embedded FS is nil")
	}

	// Load manifest to check for generated files we must not overwrite.
	m := readManifest(cfg.RepoPath, cfg.Logger)

	var installed []string
	err := fs.WalkDir(t.FS, compoundVPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// path is e.g. "compound-v/rules/compound-v.md"
		// Strip "compound-v/" prefix → "rules/compound-v.md"
		rel := path[len(compoundVPrefix)+1:]

		// Skip agent-generated files that already exist.
		baseName := filepath.Base(rel)
		if m.isGenerated(baseName) {
			targetPath := filepath.Join(cfg.RepoPath, filepath.FromSlash(compoundVTarget), filepath.FromSlash(rel))
			if _, err := os.Stat(targetPath); err == nil {
				cfg.Logger.Info("skipping generated file", "file", rel)
				return nil
			}
		}

		// Read from embedded FS.
		data, err := fs.ReadFile(t.FS, path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		targetPath := filepath.Join(cfg.RepoPath, filepath.FromSlash(compoundVTarget), filepath.FromSlash(rel))
		targetRel := filepath.ToSlash(filepath.Join(compoundVTarget, rel))

		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "target", targetRel, "source", path)
		} else {
			if err := writeFile(targetPath, data); err != nil {
				return err
			}
			cfg.Logger.Info("synced", "target", targetRel, "source", path)
		}

		installed = append(installed, targetRel)
		return nil
	})

	if err == nil && !cfg.DryRun && len(installed) > 0 {
		cfg.Logger.Info("compound-v installed to .promptherder/agent/ — run `promptherder` to fan out to agent targets")
	}

	return installed, err
}
