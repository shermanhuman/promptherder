package app

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/shermanhuman/promptherder/internal/files"
)

var ErrValidation = errors.New("validation error")

const (
	sourceDir      = ".agent/rules"
	copilotTarget  = ".github/copilot-instructions.md"
	copilotInstDir = ".github/instructions"
	manifestDir    = ".promptherder"
	manifestFile   = "manifest.json"
)

// manifest tracks which files promptherder owns in a target repo.
type manifest struct {
	Version     int      `json:"version"`
	SourceDir   string   `json:"source_dir"`
	GeneratedAt string   `json:"generated_at"`
	Files       []string `json:"files"` // repo-relative paths
}

// Config controls a sync run.
type Config struct {
	RepoPath  string
	SourceDir string // defaults to ".agent/rules" if empty
	Include   []string
	DryRun    bool
	Logger    *slog.Logger
}

// sourceFile represents a parsed rule from .agent/rules/.
type sourceFile struct {
	Path    string // absolute path
	Name    string // stem without extension, e.g. "00-breakdown-infra"
	ApplyTo string // from frontmatter; empty means repo-wide
	Body    []byte // content after frontmatter is stripped
}

// planItem represents a single output file to write.
type planItem struct {
	Target  string
	Content []byte
	Sources []string // names, for logging
}

// Run reads the source rules directory and fans out to Copilot targets.
// The AI coding agent reads .agent/rules/ natively; no extra output is needed for it.
func Run(ctx context.Context, cfg Config) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return fmt.Errorf("repo path: %w", ErrValidation)
	}

	srcDir := cfg.SourceDir
	if srcDir == "" {
		srcDir = sourceDir
	}

	repoPath, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	sources, err := readSources(repoPath, srcDir, cfg.Include)
	if err != nil {
		return err
	}

	// Load previous manifest (if any) for stale file cleanup.
	prevManifest := readManifest(repoPath, cfg.Logger)

	if len(sources) == 0 {
		cfg.Logger.Info("no source files found", "dir", srcDir)
		// Still clean up any previously generated files.
		curManifest := manifest{Version: 1, SourceDir: srcDir, GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
		if !cfg.DryRun {
			if err := writeManifest(repoPath, curManifest); err != nil {
				return err
			}
		}
		return cleanStale(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger)
	}

	plan := buildPlan(repoPath, srcDir, sources)

	cfg.Logger.Info("plan", "sources", len(sources), "outputs", len(plan))
	for _, item := range plan {
		if err := ctx.Err(); err != nil {
			return err
		}

		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "target", item.Target, "sources", item.Sources)
			continue
		}

		if err := writeFile(item.Target, item.Content); err != nil {
			return err
		}
		cfg.Logger.Info("synced", "target", item.Target, "sources", item.Sources)
	}

	// Build and write the new manifest.
	curManifest := buildManifest(repoPath, srcDir, plan)
	if cfg.DryRun {
		cfg.Logger.Info("dry-run", "target", filepath.Join(repoPath, manifestDir, manifestFile))
	} else {
		if err := writeManifest(repoPath, curManifest); err != nil {
			return err
		}
	}

	// Clean up files from previous manifest that are no longer current.
	if err := cleanStale(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger); err != nil {
		return err
	}

	return nil
}

// readSources discovers and parses all rule files under the given source directory.
func readSources(repoPath, srcDir string, include []string) ([]sourceFile, error) {
	root := filepath.Join(repoPath, filepath.FromSlash(srcDir))

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("check source dir %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	includes := include
	if len(includes) == 0 {
		includes = []string{"**/*"}
	}

	var matches []string
	for _, inc := range includes {
		pattern := filepath.ToSlash(inc)
		found, err := doublestar.Glob(os.DirFS(root), pattern)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", inc, err)
		}
		matches = append(matches, found...)
	}

	sort.Strings(matches)
	matches = dedupeStrings(matches)

	var sources []sourceFile
	for _, match := range matches {
		absPath := filepath.Join(root, filepath.FromSlash(match))
		if isDirectory(absPath) {
			continue
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", absPath, err)
		}

		applyTo, body := parseFrontmatter(data)
		name := strings.TrimSuffix(filepath.Base(match), filepath.Ext(match))

		sources = append(sources, sourceFile{
			Path:    absPath,
			Name:    name,
			ApplyTo: applyTo,
			Body:    body,
		})
	}

	return sources, nil
}

// buildPlan creates output plan items for Copilot targets.
//
// Targets:
//
//	.github/copilot-instructions.md              ← sources WITHOUT applyTo
//	.github/instructions/<name>.instructions.md  ← each source WITH applyTo
func buildPlan(repoPath, srcDir string, sources []sourceFile) []planItem {
	var plan []planItem

	// 1. .github/copilot-instructions.md — sources WITHOUT applyTo.
	var copilotParts [][]byte
	var copilotSources []string
	for _, s := range sources {
		if s.ApplyTo == "" {
			copilotParts = append(copilotParts, s.Body)
			copilotSources = append(copilotSources, s.Name)
		}
	}
	if len(copilotParts) > 0 {
		plan = append(plan, planItem{
			Target: filepath.Join(repoPath, filepath.FromSlash(copilotTarget)),
			Content: concatWithHeader(
				fmt.Sprintf("<!-- Auto-generated by promptherder from %s/ \u2014 do not edit -->\n", srcDir),
				copilotParts,
			),
			Sources: copilotSources,
		})
	}

	// 2. .github/instructions/<name>.instructions.md — each source WITH applyTo.
	for _, s := range sources {
		if s.ApplyTo == "" {
			continue
		}

		header := fmt.Sprintf("---\napplyTo: %q\n---\n<!-- Auto-generated by promptherder from %s/%s.md — do not edit -->\n",
			s.ApplyTo, srcDir, s.Name)

		var buf bytes.Buffer
		buf.WriteString(header)
		buf.WriteByte('\n')
		buf.Write(bytes.TrimSpace(s.Body))
		buf.WriteByte('\n')

		plan = append(plan, planItem{
			Target:  filepath.Join(repoPath, filepath.FromSlash(copilotInstDir), s.Name+".instructions.md"),
			Content: buf.Bytes(),
			Sources: []string{s.Name},
		})
	}

	return plan
}

// parseFrontmatter extracts the applyTo value from YAML frontmatter.
// The parser is intentionally minimal (no YAML dependency). Only the "applyTo"
// key is recognized; all other frontmatter keys are silently ignored.
// Returns the applyTo value (empty if none) and the body after frontmatter.
func parseFrontmatter(data []byte) (applyTo string, body []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return "", data
	}

	var fmLines []string
	closed := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		fmLines = append(fmLines, line)
	}

	if !closed {
		return "", data
	}

	for _, line := range fmLines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "applyTo:") {
			val := strings.TrimPrefix(line, "applyTo:")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			applyTo = val
		}
	}

	var bodyBuf bytes.Buffer
	for scanner.Scan() {
		bodyBuf.WriteString(scanner.Text())
		bodyBuf.WriteByte('\n')
	}

	return applyTo, bodyBuf.Bytes()
}

// concatWithHeader joins body parts with a leading header comment.
func concatWithHeader(header string, parts [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(header)
	buf.WriteByte('\n')

	for i, part := range parts {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.Write(bytes.TrimSpace(part))
		buf.WriteByte('\n')
	}

	return buf.Bytes()
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

// buildManifest creates a new manifest from the current plan.
func buildManifest(repoPath, srcDir string, plan []planItem) manifest {
	var relPaths []string
	for _, item := range plan {
		rel, err := filepath.Rel(repoPath, item.Target)
		if err != nil {
			rel = item.Target
		}
		relPaths = append(relPaths, filepath.ToSlash(rel))
	}
	sort.Strings(relPaths)
	return manifest{
		Version:     1,
		SourceDir:   srcDir,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Files:       relPaths,
	}
}

// writeManifest writes the manifest to .promptherder/manifest.json.
func writeManifest(repoPath string, m manifest) error {
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
	curSet := make(map[string]bool, len(cur.Files))
	for _, f := range cur.Files {
		curSet[f] = true
	}

	for _, f := range prev.Files {
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

func writeFile(target string, content []byte) error {
	writer := files.AtomicWriter{Path: target, Perm: 0o644}
	if err := writer.Write(content); err != nil {
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

func dedupeStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
