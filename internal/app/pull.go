package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PullConfig holds the configuration for a pull operation.
type PullConfig struct {
	RepoPath string       // absolute path to the repo root
	DryRun   bool         // if true, log what would happen but don't clone
	Logger   *slog.Logger // structured logger
}

// Pull clones or updates a herd from a git URL.
// The herd name is derived from the URL's last path segment (sans .git).
// After clone/pull, it validates that herd.json exists.
func Pull(ctx context.Context, gitURL string, cfg PullConfig) error {
	// Check git is available.
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required for `promptherder pull`: %w", err)
	}

	name := herdNameFromURL(gitURL)
	if name == "" {
		return fmt.Errorf("cannot derive herd name from URL: %s", gitURL)
	}

	herdPath := filepath.Join(cfg.RepoPath, herdsDir, name)

	if cfg.DryRun {
		if isDirectory(herdPath) {
			cfg.Logger.Info("dry-run: would update herd", "name", name, "url", gitURL)
		} else {
			cfg.Logger.Info("dry-run: would clone herd", "name", name, "url", gitURL, "path", herdPath)
		}
		return nil
	}

	if isDirectory(herdPath) {
		// Update existing herd.
		cfg.Logger.Info("updating herd", "name", name, "url", gitURL)
		cmd := exec.CommandContext(ctx, "git", "pull")
		cmd.Dir = herdPath
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git pull in %s: %w", herdPath, err)
		}
	} else {
		// Clone new herd.
		cfg.Logger.Info("pulling herd", "name", name, "url", gitURL)
		cmd := exec.CommandContext(ctx, "git", "clone", gitURL, herdPath)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone %s: %w", gitURL, err)
		}
	}

	// Validate herd.json exists.
	metaPath := filepath.Join(herdPath, herdMetaFile)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("herd %q has no %s — is this a valid herd repository?", name, herdMetaFile)
	}

	cfg.Logger.Info("herd ready", "name", name, "path", herdPath)
	return nil
}

// herdNameFromURL extracts the herd name from a git URL.
// e.g. "https://github.com/shermanhuman/compound-v.git" → "compound-v"
// e.g. "https://github.com/shermanhuman/compound-v" → "compound-v"
func herdNameFromURL(gitURL string) string {
	// Strip trailing slashes and .git suffix.
	u := strings.TrimRight(gitURL, "/\\")
	u = strings.TrimSuffix(u, ".git")

	// Take the last path segment (handle both / and \ separators).
	if idx := strings.LastIndexAny(u, "/\\"); idx >= 0 {
		return u[idx+1:]
	}
	return u
}
