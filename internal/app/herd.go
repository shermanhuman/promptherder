package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	herdsDir      = ".promptherder/herds"
	herdMetaFile  = "herd.json"
	agentDir      = ".promptherder/agent"
	hardRulesFile = ".promptherder/hard-rules.md"
)

// herdContentDirs are the only top-level directories merged from a herd.
// Files outside these dirs (README.md, LICENSE, etc.) are not copied.
var herdContentDirs = map[string]bool{
	"rules":     true,
	"skills":    true,
	"workflows": true,
}

// HerdMeta is the metadata parsed from a herd's herd.json file.
type HerdMeta struct {
	Name string `json:"name"`
}

// herdOnDisk pairs metadata with its filesystem location.
type herdOnDisk struct {
	Meta HerdMeta
	Path string // absolute path to the herd root (e.g. .promptherder/herds/compound-v)
}

// discoverHerds scans .promptherder/herds/ for installed herds.
// Returns herds sorted by name for deterministic merge order.
func discoverHerds(repoPath string) ([]herdOnDisk, error) {
	root := filepath.Join(repoPath, herdsDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read herds dir: %w", err)
	}

	var herds []herdOnDisk
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		herdPath := filepath.Join(root, e.Name())
		metaPath := filepath.Join(herdPath, herdMetaFile)

		data, err := os.ReadFile(metaPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue // skip dirs without herd.json
			}
			return nil, fmt.Errorf("read %s: %w", metaPath, err)
		}

		var meta HerdMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parse %s: %w", metaPath, err)
		}
		if meta.Name == "" {
			meta.Name = e.Name()
		}

		herds = append(herds, herdOnDisk{Meta: meta, Path: herdPath})
	}

	sort.Slice(herds, func(i, j int) bool {
		return herds[i].Meta.Name < herds[j].Meta.Name
	})

	return herds, nil
}

// mergeHerds copies all herd content into .promptherder/agent/, erroring on conflict.
// It respects generated files that already exist and should not be overwritten.
func mergeHerds(ctx context.Context, repoPath string, herds []herdOnDisk, m manifest, cfg TargetConfig) ([]string, error) {
	agentRoot := filepath.Join(repoPath, agentDir)

	// Track which herd owns each relative path, for conflict detection.
	ownership := make(map[string]string) // relSlash → herd name
	var installed []string

	for _, herd := range herds {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		err := filepath.WalkDir(herd.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			// Skip .git directory entirely.
			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}
			// Only walk into known herd content directories at the top level.
			if d.IsDir() {
				rel, err := filepath.Rel(herd.Path, path)
				if err != nil {
					return fmt.Errorf("rel path: %w", err)
				}
				// Allow the herd root itself.
				if rel == "." {
					return nil
				}
				// At top level, skip dirs not in herdContentDirs.
				topDir := strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
				if !herdContentDirs[topDir] {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip herd.json itself.
			if d.Name() == herdMetaFile {
				return nil
			}

			rel, err := filepath.Rel(herd.Path, path)
			if err != nil {
				return fmt.Errorf("rel path: %w", err)
			}
			relSlash := filepath.ToSlash(rel)

			// Skip files not under a known content directory.
			topDir := strings.SplitN(relSlash, "/", 2)[0]
			if !herdContentDirs[topDir] {
				return nil
			}

			// Conflict detection: does another herd already provide this file?
			if owner, exists := ownership[relSlash]; exists {
				return fmt.Errorf("conflict: %s provided by both herd %q and %q", relSlash, owner, herd.Meta.Name)
			}

			// Skip agent-generated files that already exist.
			baseName := filepath.Base(rel)
			if m.isGenerated(baseName) {
				targetPath := filepath.Join(agentRoot, filepath.FromSlash(relSlash))
				if _, err := os.Stat(targetPath); err == nil {
					cfg.Logger.Debug("skipping generated file", "file", relSlash, "herd", herd.Meta.Name)
					return nil
				}
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}

			targetPath := filepath.Join(agentRoot, filepath.FromSlash(relSlash))
			targetRel := filepath.ToSlash(filepath.Join(agentDir, relSlash))

			if cfg.DryRun {
				cfg.Logger.Info("dry-run", "target", targetRel, "source", relSlash, "herd", herd.Meta.Name)
			} else {
				if err := writeFile(targetPath, data); err != nil {
					return err
				}
				cfg.Logger.Info("synced", "target", targetRel, "source", relSlash, "herd", herd.Meta.Name)
			}

			ownership[relSlash] = herd.Meta.Name
			installed = append(installed, targetRel)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("herd %s: %w", herd.Meta.Name, err)
		}
	}

	return installed, nil
}

// cleanAgentDir removes all files from .promptherder/agent/ that are tracked
// in the manifest under the herds target, preparing for a fresh merge.
func cleanAgentDir(repoPath string, prev manifest, dryRun bool, logger interface{ Info(string, ...any) }) error {
	herdFiles := prev.Targets["herds"]
	if len(herdFiles) == 0 {
		return nil
	}

	for _, relSlash := range herdFiles {
		// Only clean files under .promptherder/agent/
		if !strings.HasPrefix(relSlash, agentDir+"/") {
			continue
		}
		absPath := filepath.Join(repoPath, filepath.FromSlash(relSlash))

		// Skip generated files — those are user-owned.
		baseName := filepath.Base(relSlash)
		if prev.isGenerated(baseName) {
			continue
		}

		if dryRun {
			logger.Info("dry-run: would remove", "file", relSlash)
			continue
		}
		if err := os.Remove(absPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clean %s: %w", relSlash, err)
		}

		// Clean empty parent directories up to agentDir.
		parentDir := filepath.Dir(absPath)
		agentRoot := filepath.Join(repoPath, agentDir)
		for parentDir != agentRoot && parentDir != "." {
			entries, err := os.ReadDir(parentDir)
			if err != nil || len(entries) > 0 {
				break
			}
			_ = os.Remove(parentDir)
			parentDir = filepath.Dir(parentDir)
		}
	}

	return nil
}
