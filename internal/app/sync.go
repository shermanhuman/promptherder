package app

import (
	"bufio"
	"bytes"
	"context"
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
	defaultSourceDir  = ".promptherder/agent/rules"
	workflowSourceDir = ".promptherder/agent/workflows"
	skillSourceDir    = ".promptherder/agent/skills"
	copilotTarget     = ".github/copilot-instructions.md"
	copilotInstDir    = ".github/instructions"
	copilotPromptsDir = ".github/prompts"
)

// Config controls a sync run.
type Config struct {
	RepoPath  string
	SourceDir string // defaults to ".promptherder/agent/rules" if empty
	Include   []string
	DryRun    bool
	Logger    *slog.Logger
}

// sourceFile represents a parsed rule from the source directory.
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

// CopilotTarget implements the Target interface for GitHub Copilot.
type CopilotTarget struct {
	SourceDir string   // override source dir; defaults to defaultSourceDir
	Include   []string // glob patterns for source files
}

func (t CopilotTarget) Name() string { return "copilot" }

func (t CopilotTarget) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
	srcDir := t.SourceDir
	if srcDir == "" {
		srcDir = defaultSourceDir
	}

	var written []string

	// 1. Rules → copilot-instructions.md + instruction files.
	sources, err := readSources(cfg.RepoPath, srcDir, t.Include)
	if err != nil {
		return nil, err
	}

	if len(sources) > 0 {
		plan := buildCopilotPlan(cfg.RepoPath, srcDir, sources)
		cfg.Logger.Info("plan", "target", "copilot/rules", "sources", len(sources), "outputs", len(plan))
		for _, item := range plan {
			if err := ctx.Err(); err != nil {
				return written, err
			}
			rel, _ := filepath.Rel(cfg.RepoPath, item.Target)
			relSlash := filepath.ToSlash(rel)
			if cfg.DryRun {
				cfg.Logger.Info("dry-run", "target", relSlash, "sources", item.Sources)
			} else {
				if err := writeFile(item.Target, item.Content); err != nil {
					return written, err
				}
				cfg.Logger.Info("synced", "target", relSlash, "sources", item.Sources)
			}
			written = append(written, relSlash)
		}
	} else {
		cfg.Logger.Info("no source files found", "dir", srcDir)
	}

	// 2. Workflows → .github/prompts/*.prompt.md.
	promptItems, err := buildCopilotPrompts(cfg.RepoPath)
	if err != nil {
		return written, err
	}

	// 3. Skills → .github/prompts/*.prompt.md.
	skillItems, err := buildCopilotSkillPrompts(cfg.RepoPath)
	if err != nil {
		return written, err
	}
	promptItems = append(promptItems, skillItems...)

	if len(promptItems) > 0 {
		cfg.Logger.Info("plan", "target", "copilot/prompts", "workflows", len(promptItems)-len(skillItems), "skills", len(skillItems))
	}
	for _, item := range promptItems {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		rel, _ := filepath.Rel(cfg.RepoPath, item.Target)
		relSlash := filepath.ToSlash(rel)
		if cfg.DryRun {
			cfg.Logger.Info("dry-run", "target", relSlash, "sources", item.Sources)
		} else {
			if err := writeFile(item.Target, item.Content); err != nil {
				return written, err
			}
			cfg.Logger.Info("synced", "target", relSlash, "sources", item.Sources)
		}
		written = append(written, relSlash)
	}

	return written, nil
}

// RunCopilot is the legacy entry point that runs the Copilot target with
// manifest management. Preserved for backward compatibility.
func RunCopilot(ctx context.Context, cfg Config) error {
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return fmt.Errorf("repo path: %w", ErrValidation)
	}

	repoPath, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	target := CopilotTarget{SourceDir: cfg.SourceDir, Include: cfg.Include}
	tcfg := TargetConfig{RepoPath: repoPath, DryRun: cfg.DryRun, Logger: cfg.Logger}

	// Load previous manifest for stale cleanup.
	prevManifest := readManifest(repoPath, cfg.Logger)

	installed, err := target.Install(ctx, tcfg)
	if err != nil {
		return err
	}

	// Build new manifest.
	curManifest := manifest{
		Version:     2,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	curManifest.setTarget("copilot", installed)

	// Preserve other targets from previous manifest.
	for name, files := range prevManifest.Targets {
		if name != "copilot" {
			curManifest.setTarget(name, files)
		}
	}
	// Preserve generated list.
	curManifest.Generated = prevManifest.Generated

	if cfg.DryRun {
		cfg.Logger.Info("dry-run", "target", filepath.Join(repoPath, manifestDir, manifestFile))
	} else {
		if err := writeManifest(repoPath, curManifest); err != nil {
			return err
		}
	}

	return cleanStale(repoPath, prevManifest, curManifest, cfg.DryRun, cfg.Logger)
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

// buildCopilotPlan creates output plan items for Copilot targets.
func buildCopilotPlan(repoPath, srcDir string, sources []sourceFile) []planItem {
	var plan []planItem

	// .github/copilot-instructions.md — sources WITHOUT applyTo.
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
				fmt.Sprintf("<!-- Auto-generated by promptherder from %s/ — do not edit -->\n", srcDir),
				copilotParts,
			),
			Sources: copilotSources,
		})
	}

	// .github/instructions/<name>.instructions.md — each source WITH applyTo.
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

// buildCopilotPrompts reads workflow files from .promptherder/agent/workflows/
// and converts them to .github/prompts/*.prompt.md for Copilot Chat.
//
// Conversion:
//   - Antigravity frontmatter (description) → Copilot frontmatter (mode, description)
//   - Strips Antigravity-specific annotations (// turbo, // turbo-all)
//   - Renames: brainstorm.md → brainstorm.prompt.md
func buildCopilotPrompts(repoPath string) ([]planItem, error) {
	wfRoot := filepath.Join(repoPath, filepath.FromSlash(workflowSourceDir))

	if _, err := os.Stat(wfRoot); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(wfRoot)
	if err != nil {
		return nil, fmt.Errorf("read workflows dir: %w", err)
	}

	var plan []planItem
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(wfRoot, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read workflow %s: %w", entry.Name(), err)
		}

		promptContent := convertWorkflowToPrompt(entry.Name(), data)
		stem := strings.TrimSuffix(entry.Name(), ".md")

		plan = append(plan, planItem{
			Target:  filepath.Join(repoPath, filepath.FromSlash(copilotPromptsDir), stem+".prompt.md"),
			Content: promptContent,
			Sources: []string{stem},
		})
	}

	return plan, nil
}

// buildCopilotSkillPrompts reads skill files from .promptherder/agent/skills/*/SKILL.md
// and converts them to .github/prompts/*.prompt.md for Copilot Chat.
//
// Each skill directory contains a SKILL.md file with name/description frontmatter.
// The directory name becomes the prompt file name (e.g., compound-v-tdd → compound-v-tdd.prompt.md).
func buildCopilotSkillPrompts(repoPath string) ([]planItem, error) {
	skillsRoot := filepath.Join(repoPath, filepath.FromSlash(skillSourceDir))

	if _, err := os.Stat(skillsRoot); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	var plan []planItem
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillFile := filepath.Join(skillsRoot, entry.Name(), "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue // skip directories without SKILL.md
			}
			return nil, fmt.Errorf("read skill %s: %w", entry.Name(), err)
		}

		promptContent := convertWorkflowToPrompt("skills/"+entry.Name()+"/SKILL.md", data)

		plan = append(plan, planItem{
			Target:  filepath.Join(repoPath, filepath.FromSlash(copilotPromptsDir), entry.Name()+".prompt.md"),
			Content: promptContent,
			Sources: []string{entry.Name()},
		})
	}

	return plan, nil
}

// convertWorkflowToPrompt transforms an Antigravity workflow file into a
// Copilot .prompt.md file.
func convertWorkflowToPrompt(filename string, data []byte) []byte {
	// Parse frontmatter to extract description.
	_, body := parseFrontmatter(data)
	desc := extractDescription(data)

	// Strip Antigravity-specific annotations.
	body = stripAntigravityAnnotations(body)

	// Build Copilot prompt file.
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.WriteString("mode: \"agent\"\n")
	if desc != "" {
		buf.WriteString(fmt.Sprintf("description: %q\n", desc))
	}
	buf.WriteString("---\n")
	buf.WriteString(fmt.Sprintf("<!-- Auto-generated by promptherder from %s/%s — do not edit -->\n\n",
		workflowSourceDir, filename))
	buf.Write(bytes.TrimSpace(body))
	buf.WriteByte('\n')

	return buf.Bytes()
}

// extractDescription pulls the description value from YAML frontmatter.
func extractDescription(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return ""
	}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}
		if strings.HasPrefix(line, "description:") {
			val := strings.TrimPrefix(line, "description:")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			return val
		}
	}
	return ""
}

// stripAntigravityAnnotations removes lines like "// turbo" and "// turbo-all"
// that are Antigravity-specific and meaningless to Copilot.
func stripAntigravityAnnotations(body []byte) []byte {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "// turbo" || trimmed == "// turbo-all" {
			continue
		}
		lines = append(lines, line)
	}
	return []byte(strings.Join(lines, "\n"))
}

// parseFrontmatter extracts the applyTo value from YAML frontmatter.
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
