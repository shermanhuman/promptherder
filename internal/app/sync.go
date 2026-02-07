package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/shermanhuman/promptherder/internal/files"
)

var ErrValidation = errors.New("validation error")

type Config struct {
	RepoPath string
	Source   string
	Include  []string
	DryRun   bool
	Logger   *slog.Logger
}

type mapping struct {
	Name       string
	SourceRoot string
	TargetRoot string
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return fmt.Errorf("repo path: %w", ErrValidation)
	}
	if strings.TrimSpace(cfg.Source) == "" {
		return fmt.Errorf("source: %w", ErrValidation)
	}

	repoPath, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	plan, err := buildPlan(repoPath, cfg.Source, cfg.Include)
	if err != nil {
		return err
	}

	cfg.Logger.Info("plan", "source", cfg.Source, "files", len(plan))
	for _, item := range plan {
		if err := ctx.Err(); err != nil {
			return err
		}

		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "copy", item.Source, "to", item.Target)
			continue
		}

		if err := copyFile(item.Source, item.Target); err != nil {
			return err
		}
		cfg.Logger.Info("synced", "source", item.Source, "target", item.Target)
	}

	return nil
}

type planItem struct {
	Source string
	Target string
}

func buildPlan(repoPath, source string, include []string) ([]planItem, error) {
	mappings, err := mappingsFor(source)
	if err != nil {
		return nil, err
	}

	includes := include
	if len(includes) == 0 {
		includes = []string{"**/*"}
	}

	var plan []planItem
	for _, m := range mappings {
		root := filepath.Join(repoPath, filepath.FromSlash(m.SourceRoot))
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("check source root %s: %w", root, err)
		}

		if !info.IsDir() {
			continue
		}

		for _, inc := range includes {
			pattern := filepath.ToSlash(inc)
			matches, err := doublestar.Glob(os.DirFS(root), pattern)
			if err != nil {
				return nil, fmt.Errorf("glob %s: %w", inc, err)
			}

			for _, match := range matches {
				if strings.HasSuffix(match, "/") {
					continue
				}
				src := filepath.Join(root, filepath.FromSlash(match))
				if isDirectory(src) {
					continue
				}
				targetRoot := filepath.Join(repoPath, filepath.FromSlash(m.TargetRoot))
				target := filepath.Join(targetRoot, filepath.FromSlash(match))
				plan = append(plan, planItem{Source: src, Target: target})
			}
		}
	}

	return dedupePlan(plan), nil
}

// mappingsFor returns the file sync mappings for a given source of truth.
//
// "antigravity" (default): .antigravity/ is the canonical source.
//
//	Rules  → .agent/rules/  (where Antigravity actually reads them)
//	Skills → .agent/skills/ (where Antigravity actually reads them)
//
// "agent": .agent/ is the canonical source, syncs back to .antigravity/.
func mappingsFor(source string) ([]mapping, error) {
	key := strings.ToLower(strings.TrimSpace(source))
	switch key {
	case "antigravity":
		return []mapping{
			{Name: "rules", SourceRoot: ".antigravity/rules", TargetRoot: ".agent/rules"},
			{Name: "skills", SourceRoot: ".antigravity/skills", TargetRoot: ".agent/skills"},
		}, nil
	case "agent":
		return []mapping{
			{Name: "rules", SourceRoot: ".agent/rules", TargetRoot: ".antigravity/rules"},
			{Name: "skills", SourceRoot: ".agent/skills", TargetRoot: ".antigravity/skills"},
		}, nil
	default:
		return nil, fmt.Errorf("unknown source %q (valid: antigravity, agent): %w", source, ErrValidation)
	}
}

func copyFile(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("read %s: %w", source, err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", target, err)
	}

	writer := files.AtomicWriter{Path: target, Perm: 0o644}
	if err := writer.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}

	return nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func dedupePlan(items []planItem) []planItem {
	seen := make(map[string]planItem, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		if _, exists := seen[item.Target]; !exists {
			order = append(order, item.Target)
		}
		seen[item.Target] = item
	}

	result := make([]planItem, 0, len(order))
	for _, key := range order {
		result = append(result, seen[key])
	}
	return result
}
