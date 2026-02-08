package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// RunAll runs all registered targets and writes a unified manifest.
// This is the bare `promptherder` command — copies everything everywhere.
func RunAll(ctx context.Context, targets []Target, cfg Config) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	repoPath, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	prevManifest := readManifest(repoPath, cfg.Logger)

	tcfg := TargetConfig{
		RepoPath: repoPath,
		DryRun:   cfg.DryRun,
		Logger:   cfg.Logger,
	}

	curManifest := manifest{
		Version:     2,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Generated:   prevManifest.Generated,
	}

	for _, t := range targets {
		if err := ctx.Err(); err != nil {
			return err
		}

		cfg.Logger.Info("installing target", "name", t.Name())
		installed, err := t.Install(ctx, tcfg)
		if err != nil {
			return fmt.Errorf("target %s: %w", t.Name(), err)
		}
		curManifest.setTarget(t.Name(), installed)
	}

	if cfg.DryRun {
		cfg.Logger.Info("dry-run", "target", filepath.Join(repoPath, manifestDir, manifestFile))
	} else {
		if err := writeManifest(repoPath, curManifest); err != nil {
			return err
		}
	}

	return cleanStale(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger)
}

// RunTarget runs a single named target and updates the manifest.
func RunTarget(ctx context.Context, target Target, cfg Config) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	repoPath, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	prevManifest := readManifest(repoPath, cfg.Logger)

	tcfg := TargetConfig{
		RepoPath: repoPath,
		DryRun:   cfg.DryRun,
		Logger:   cfg.Logger,
	}

	cfg.Logger.Info("installing target", "name", target.Name())
	installed, err := target.Install(ctx, tcfg)
	if err != nil {
		return fmt.Errorf("target %s: %w", target.Name(), err)
	}

	// Build manifest: preserve all other targets, update this one.
	curManifest := manifest{
		Version:     2,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Generated:   prevManifest.Generated,
	}
	for name, files := range prevManifest.Targets {
		if name != target.Name() {
			curManifest.setTarget(name, files)
		}
	}
	curManifest.setTarget(target.Name(), installed)

	if cfg.DryRun {
		cfg.Logger.Info("dry-run", "target", filepath.Join(repoPath, manifestDir, manifestFile))
	} else {
		if err := writeManifest(repoPath, curManifest); err != nil {
			return err
		}
	}

	return cleanStale(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger)
}
