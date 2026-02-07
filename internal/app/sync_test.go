package app

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseFrontmatter ---

func TestParseFrontmatter_WithApplyTo(t *testing.T) {
	input := []byte("---\napplyTo: \"**/*.sh\"\n---\n# Shell rules\n\nDo the thing.\n")
	applyTo, body := parseFrontmatter(input)

	if applyTo != "**/*.sh" {
		t.Errorf("applyTo = %q, want %q", applyTo, "**/*.sh")
	}
	if !bytes.Contains(body, []byte("# Shell rules")) {
		t.Errorf("body missing expected content: %q", body)
	}
	if bytes.Contains(body, []byte("applyTo")) {
		t.Error("body should not contain frontmatter")
	}
}

func TestParseFrontmatter_WithoutFrontmatter(t *testing.T) {
	input := []byte("# Just a doc\n\nNo frontmatter here.\n")
	applyTo, body := parseFrontmatter(input)

	if applyTo != "" {
		t.Errorf("applyTo = %q, want empty", applyTo)
	}
	if !bytes.Equal(body, input) {
		t.Errorf("body should equal input verbatim")
	}
}

func TestParseFrontmatter_UnclosedFrontmatter(t *testing.T) {
	input := []byte("---\napplyTo: \"**/*.yaml\"\n# No closing delimiter\n")
	applyTo, body := parseFrontmatter(input)

	if applyTo != "" {
		t.Errorf("unclosed frontmatter should return empty applyTo, got %q", applyTo)
	}
	if !bytes.Equal(body, input) {
		t.Error("unclosed frontmatter should return original data as body")
	}
}

func TestParseFrontmatter_SingleQuotes(t *testing.T) {
	input := []byte("---\napplyTo: '**/*.yaml'\n---\n# Content\n")
	applyTo, _ := parseFrontmatter(input)

	if applyTo != "**/*.yaml" {
		t.Errorf("applyTo = %q, want %q", applyTo, "**/*.yaml")
	}
}

func TestParseFrontmatter_NoQuotes(t *testing.T) {
	input := []byte("---\napplyTo: **/*.yaml\n---\n# Content\n")
	applyTo, _ := parseFrontmatter(input)

	if applyTo != "**/*.yaml" {
		t.Errorf("applyTo = %q, want %q", applyTo, "**/*.yaml")
	}
}

func TestParseFrontmatter_EmptyFile(t *testing.T) {
	applyTo, body := parseFrontmatter([]byte{})
	if applyTo != "" {
		t.Errorf("applyTo = %q, want empty", applyTo)
	}
	if len(body) != 0 {
		t.Errorf("body should be empty, got %q", body)
	}
}

// --- readSources ---

func TestReadSources_Basic(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	mustMkdir(t, rulesDir)

	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n\nBe good.\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n\nSafe bash.\n")

	sources, err := readSources(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	// Sorted by filename
	if sources[0].Name != "00-general" {
		t.Errorf("sources[0].Name = %q, want %q", sources[0].Name, "00-general")
	}
	if sources[0].ApplyTo != "" {
		t.Errorf("sources[0].ApplyTo = %q, want empty", sources[0].ApplyTo)
	}
	if sources[1].Name != "01-shell" {
		t.Errorf("sources[1].Name = %q, want %q", sources[1].Name, "01-shell")
	}
	if sources[1].ApplyTo != "**/*.sh" {
		t.Errorf("sources[1].ApplyTo = %q, want %q", sources[1].ApplyTo, "**/*.sh")
	}
}

func TestReadSources_MissingDir(t *testing.T) {
	dir := t.TempDir()
	sources, err := readSources(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(sources))
	}
}

func TestReadSources_IncludeFilter(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	mustMkdir(t, rulesDir)

	mustWrite(t, filepath.Join(rulesDir, "keep.md"), "# Keep\n")
	mustWrite(t, filepath.Join(rulesDir, "skip.txt"), "skip\n")

	sources, err := readSources(dir, []string{"**/*.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if sources[0].Name != "keep" {
		t.Errorf("name = %q, want %q", sources[0].Name, "keep")
	}
}

// --- buildPlan ---

func TestBuildPlan_AllRepoWide(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# General\n")},
		{Name: "01-ops", Body: []byte("# Ops\n")},
	}

	plan := buildPlan(repoPath, sources)

	// Should produce 2 items: GEMINI.md + copilot-instructions.md
	if len(plan) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, "GEMINI.md"))
	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "copilot-instructions.md"))

	// GEMINI.md should contain both
	assertContains(t, plan[0].Content, "# General")
	assertContains(t, plan[0].Content, "# Ops")

	// copilot-instructions.md should also contain both
	assertContains(t, plan[1].Content, "# General")
	assertContains(t, plan[1].Content, "# Ops")
}

func TestBuildPlan_WithApplyTo(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# General\n")},
		{Name: "01-shell", ApplyTo: "**/*.sh", Body: []byte("# Shell\n")},
	}

	plan := buildPlan(repoPath, sources)

	// GEMINI.md + copilot-instructions.md + 01-shell.instructions.md = 3
	if len(plan) != 3 {
		t.Fatalf("expected 3 plan items, got %d", len(plan))
	}

	// GEMINI.md gets ALL sources
	assertTarget(t, plan[0], filepath.Join(repoPath, "GEMINI.md"))
	assertContains(t, plan[0].Content, "# General")
	assertContains(t, plan[0].Content, "# Shell")

	// copilot-instructions.md gets only repo-wide (no applyTo)
	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "copilot-instructions.md"))
	assertContains(t, plan[1].Content, "# General")
	assertNotContains(t, plan[1].Content, "# Shell")

	// 01-shell.instructions.md gets its own file with frontmatter
	assertTarget(t, plan[2], filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md"))
	assertContains(t, plan[2].Content, `applyTo: "**/*.sh"`)
	assertContains(t, plan[2].Content, "# Shell")
}

func TestBuildPlan_AllWithApplyTo(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-yaml", ApplyTo: "**/*.yaml", Body: []byte("# YAML\n")},
		{Name: "01-shell", ApplyTo: "**/*.sh", Body: []byte("# Shell\n")},
	}

	plan := buildPlan(repoPath, sources)

	// GEMINI.md + 2 instruction files = 3 (no copilot-instructions.md since all have applyTo)
	if len(plan) != 3 {
		t.Fatalf("expected 3 plan items, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, "GEMINI.md"))
	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "instructions", "00-yaml.instructions.md"))
	assertTarget(t, plan[2], filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md"))
}

func TestBuildPlan_GeneratedHeaders(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# Rules\n")},
	}

	plan := buildPlan(repoPath, sources)

	assertContains(t, plan[0].Content, "Auto-generated by promptherder")
	assertContains(t, plan[0].Content, "Do not edit")
	assertContains(t, plan[1].Content, "Auto-generated by promptherder")
	assertContains(t, plan[1].Content, "do not edit")
}

// --- Run integration ---

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-test.md"), "# Test\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   true,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Nothing should be written in dry-run
	if _, err := os.Stat(filepath.Join(dir, "GEMINI.md")); !os.IsNotExist(err) {
		t.Error("dry-run should not create GEMINI.md")
	}
}

func TestRun_ActualSync(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n\nBe good.\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n\nSafe bash.\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Check GEMINI.md
	gemini, err := os.ReadFile(filepath.Join(dir, "GEMINI.md"))
	if err != nil {
		t.Fatalf("GEMINI.md not created: %v", err)
	}
	if !bytes.Contains(gemini, []byte("# General")) || !bytes.Contains(gemini, []byte("# Shell")) {
		t.Error("GEMINI.md should contain all rules")
	}

	// Check copilot-instructions.md
	copilot, err := os.ReadFile(filepath.Join(dir, ".github", "copilot-instructions.md"))
	if err != nil {
		t.Fatalf("copilot-instructions.md not created: %v", err)
	}
	if !bytes.Contains(copilot, []byte("# General")) {
		t.Error("copilot-instructions.md should contain repo-wide rules")
	}
	if bytes.Contains(copilot, []byte("# Shell")) {
		t.Error("copilot-instructions.md should NOT contain applyTo rules")
	}

	// Check .github/instructions/01-shell.instructions.md
	inst, err := os.ReadFile(filepath.Join(dir, ".github", "instructions", "01-shell.instructions.md"))
	if err != nil {
		t.Fatalf("01-shell.instructions.md not created: %v", err)
	}
	if !bytes.Contains(inst, []byte(`applyTo: "**/*.sh"`)) {
		t.Error("instruction file should have applyTo frontmatter")
	}
	if !bytes.Contains(inst, []byte("# Shell")) {
		t.Error("instruction file should contain shell rules body")
	}
}

func TestRun_NoSources(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "GEMINI.md")); !os.IsNotExist(err) {
		t.Error("should not create files when no sources exist")
	}
}

func TestRun_ValidationError(t *testing.T) {
	cfg := Config{
		RepoPath: "",
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	err := Run(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "repo path") {
		t.Errorf("error = %q, want repo path validation", err)
	}
}

// --- dedupeStrings ---

func TestDedupeStrings(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b"}
	result := dedupeStrings(input)

	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	want := "a,b,c"
	got := strings.Join(result, ",")
	if got != want {
		t.Errorf("result = %q, want %q", got, want)
	}
}

// --- helpers ---

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTarget(t *testing.T, item planItem, want string) {
	t.Helper()
	if item.Target != want {
		t.Errorf("target = %q, want %q", item.Target, want)
	}
}

func assertContains(t *testing.T, data []byte, substr string) {
	t.Helper()
	if !bytes.Contains(data, []byte(substr)) {
		t.Errorf("expected content to contain %q, got:\n%s", substr, data)
	}
}

func assertNotContains(t *testing.T, data []byte, substr string) {
	t.Helper()
	if bytes.Contains(data, []byte(substr)) {
		t.Errorf("expected content NOT to contain %q, got:\n%s", substr, data)
	}
}
