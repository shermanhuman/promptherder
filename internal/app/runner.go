package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

// setupRunner initializes logger, resolves paths, and loads manifest.
// Returns the absolute repo path, previous manifest, target config, and any error.
func setupRunner(cfg *Config) (repoPath string, prevManifest manifest, tcfg TargetConfig, err error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	repoPath, err = filepath.Abs(cfg.RepoPath)
	if err != nil {
		return "", manifest{}, TargetConfig{}, fmt.Errorf("resolve repo path: %w", err)
	}

	prevManifest = readManifest(repoPath, cfg.Logger)

	tcfg = TargetConfig{
		RepoPath: repoPath,
		DryRun:   cfg.DryRun,
		Logger:   cfg.Logger,
	}

	return repoPath, prevManifest, tcfg, nil
}

// persistAndClean writes the manifest and cleans stale files.
func persistAndClean(repoPath string, prev, cur manifest, dryRun bool, logger *slog.Logger) error {
	if dryRun {
		logger.Info("dry-run", "target", filepath.Join(repoPath, manifestDir, manifestFile))
	} else {
		if err := writeManifest(repoPath, cur); err != nil {
			return err
		}
	}
	return cleanStale(repoPath, prev, cur, dryRun, logger)
}

// RunAll runs all registered targets and writes a unified manifest.
// This is the bare `promptherder` command. It merges herds first,
// then fans out to all agent targets.
func RunAll(ctx context.Context, targets []Target, cfg Config) error {
	repoPath, prevManifest, tcfg, err := setupRunner(&cfg)
	if err != nil {
		return err
	}

	curManifest := newManifestFrom(prevManifest)

	// --- Herd merge step ---
	herds, err := discoverHerds(repoPath)
	if err != nil {
		return fmt.Errorf("discover herds: %w", err)
	}

	if len(herds) == 0 {
		cfg.Logger.Warn("no herds found — run `promptherder pull <url>` to install one")
	} else {
		// Clean previous herd files from agent dir before re-merging.
		if err := cleanAgentDir(repoPath, prevManifest, cfg.DryRun, cfg.Logger); err != nil {
			return fmt.Errorf("clean agent dir: %w", err)
		}

		herdNames := make([]string, len(herds))
		for i, h := range herds {
			herdNames[i] = h.Meta.Name
		}
		cfg.Logger.Info("merging herds", "herds", herdNames)

		installed, err := mergeHerds(ctx, repoPath, herds, prevManifest, tcfg)
		if err != nil {
			return fmt.Errorf("merge herds: %w", err)
		}
		curManifest.setTarget("herds", installed)
	}

	// --- Target install step ---
	for _, t := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}

		cfg.Logger.Info("target", "name", t.Name())
		installed, err := t.Install(ctx, tcfg)
		if err != nil {
			return fmt.Errorf("target %s: %w", t.Name(), err)
		}
		curManifest.setTarget(t.Name(), installed)
	}

	return persistAndClean(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger)
}

// RunTarget runs a single named target and updates the manifest.
func RunTarget(ctx context.Context, target Target, cfg Config) error {
	repoPath, prevManifest, tcfg, err := setupRunner(&cfg)
	if err != nil {
		return err
	}

	cfg.Logger.Info("target", "name", target.Name())
	installed, err := target.Install(ctx, tcfg)
	if err != nil {
		return fmt.Errorf("target %s: %w", target.Name(), err)
	}

	// Build manifest: preserve all other targets, update this one.
	curManifest := newManifestFrom(prevManifest)
	for name, files := range prevManifest.Targets {
		if name != target.Name() {
			curManifest.setTarget(name, files)
		}
	}
	curManifest.setTarget(target.Name(), installed)

	return persistAndClean(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger)
}
