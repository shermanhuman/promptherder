package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
)

const (
	manifestDir  = ".promptherder"
	manifestFile = "manifest.json"
)

// manifest tracks which files promptherder owns in a target repo.
type manifest struct {
	Version     int                 `json:"version"`
	SourceDir   string              `json:"source_dir,omitempty"` // v1 compat
	GeneratedAt string              `json:"generated_at"`
	Files       []string            `json:"files,omitempty"`     // v1 compat
	Targets     map[string][]string `json:"targets,omitempty"`   // v2: "copilot", "antigravity", "compound-v"
	Generated   []string            `json:"generated,omitempty"` // filenames that the agent generates (e.g. stack.md) — never overwritten
}

// allFiles returns the union of v1 Files and all v2 Targets values.
// This provides backward compatibility with v1 manifests.
func (m manifest) allFiles() []string {
	seen := make(map[string]bool)
	var result []string

	for _, f := range m.Files {
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	for _, files := range m.Targets {
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				result = append(result, f)
			}
		}
	}
	sort.Strings(result)
	return result
}

// setTarget replaces the file list for the given target.
func (m *manifest) setTarget(name string, files []string) {
	if m.Targets == nil {
		m.Targets = make(map[string][]string)
	}
	sort.Strings(files)
	m.Targets[name] = files
}

// hasTarget returns true if the manifest has records for the given target.
func (m manifest) hasTarget(name string) bool {
	_, ok := m.Targets[name]
	return ok
}

// isGenerated returns true if the given filename (base name, no path) is in
// the Generated list and should not be overwritten.
func (m manifest) isGenerated(name string) bool {
	for _, g := range m.Generated {
		if g == name {
			return true
		}
	}
	return false
}

// readManifest loads the previous manifest from .promptherder/manifest.json.
// Returns an empty manifest if the file doesn't exist or is corrupt.
func readManifest(repoPath string, logger *slog.Logger) manifest {
	path := filepath.Join(repoPath, manifestDir, manifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest{}
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		logger.Warn("corrupt manifest, treating as empty", "path", path, "error", err)
		return manifest{}
	}
	return m
}

// writeManifest writes the manifest to .promptherder/manifest.json.
func writeManifest(repoPath string, m manifest) error {
	// Upgrade to v2.
	m.Version = 2
	// Clear v1-only fields if we have v2 targets.
	if len(m.Targets) > 0 {
		m.Files = nil
		m.SourceDir = ""
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	return writeFile(filepath.Join(repoPath, manifestDir, manifestFile), data)
}

// cleanStale removes files that were in the previous manifest but are not in
// the current one. This is the only mechanism for deleting files — promptherder
// never deletes anything it didn't previously create.
func cleanStale(repoPath string, prev, cur manifest, dryRun bool, logger *slog.Logger) error {
	curSet := make(map[string]bool)
	for _, f := range cur.allFiles() {
		curSet[f] = true
	}

	for _, f := range prev.allFiles() {
		if curSet[f] {
			continue
		}

		absPath := filepath.Join(repoPath, filepath.FromSlash(f))

		// Only remove if the file still exists.
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			continue
		}

		if dryRun {
			logger.Info("dry-run: would remove stale", "file", f)
			continue
		}

		if err := os.Remove(absPath); err != nil {
			return fmt.Errorf("remove stale %s: %w", f, err)
		}
		logger.Info("removed stale", "file", f)
	}

	return nil
}
