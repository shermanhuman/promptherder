package app

import (
	"context"
	"log/slog"
)

// Target knows how to install agent configuration files to a specific
// AI coding agent's location (e.g. .agent/ for Antigravity, .github/ for Copilot).
type Target interface {
	// Name returns the target identifier used in manifests and CLI subcommands.
	Name() string

	// Install copies configuration files to the target location.
	// Returns the repo-relative paths of all files written.
	// Install must be idempotent — calling it multiple times must produce the same result.
	Install(ctx context.Context, cfg TargetConfig) ([]string, error)
}

// TargetConfig holds the configuration for a target install run.
type TargetConfig struct {
	RepoPath string       // absolute path to the repo root
	DryRun   bool         // if true, log what would happen but don't write
	Logger   *slog.Logger // structured logger
	Settings Settings     // user settings from .promptherder/settings.json
}

// SkillVariantFiles maps uppercase variant filenames to their target names.
// When a skill directory contains a variant file matching the current target,
// it is installed as SKILL.md, replacing the generic version.
//
// To add a new target variant, add an entry here and implement variant
// preference in your target's Install method (see CONTRIBUTING.md).
var SkillVariantFiles = map[string]string{
	"ANTIGRAVITY.md": "antigravity",
	"COPILOT.md":     "copilot",
}
