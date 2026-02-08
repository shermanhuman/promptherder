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
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)

	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n\nBe good.\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n\nSafe bash.\n")

	sources, err := readSources(dir, ".promptherder/agent/rules", nil)
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
	sources, err := readSources(dir, ".promptherder/agent/rules", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(sources))
	}
}

func TestReadSources_IncludeFilter(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)

	mustWrite(t, filepath.Join(rulesDir, "keep.md"), "# Keep\n")
	mustWrite(t, filepath.Join(rulesDir, "skip.txt"), "skip\n")

	sources, err := readSources(dir, ".promptherder/agent/rules", []string{"**/*.md"})
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

// --- buildCopilotPlan ---

func TestBuildCopilotPlan_AllRepoWide(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# General\n")},
		{Name: "01-ops", Body: []byte("# Ops\n")},
	}

	plan := buildCopilotPlan(repoPath, ".promptherder/agent/rules", sources)

	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, ".github", "copilot-instructions.md"))

	assertContains(t, plan[0].Content, "# General")
	assertContains(t, plan[0].Content, "# Ops")
}

func TestBuildCopilotPlan_WithApplyTo(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# General\n")},
		{Name: "01-shell", ApplyTo: "**/*.sh", Body: []byte("# Shell\n")},
	}

	plan := buildCopilotPlan(repoPath, ".promptherder/agent/rules", sources)

	if len(plan) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, ".github", "copilot-instructions.md"))
	assertContains(t, plan[0].Content, "# General")
	assertNotContains(t, plan[0].Content, "# Shell")

	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md"))
	assertContains(t, plan[1].Content, `applyTo: "**/*.sh"`)
	assertContains(t, plan[1].Content, "# Shell")
}

func TestBuildCopilotPlan_AllWithApplyTo(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-yaml", ApplyTo: "**/*.yaml", Body: []byte("# YAML\n")},
		{Name: "01-shell", ApplyTo: "**/*.sh", Body: []byte("# Shell\n")},
	}

	plan := buildCopilotPlan(repoPath, ".promptherder/agent/rules", sources)

	if len(plan) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, ".github", "instructions", "00-yaml.instructions.md"))
	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md"))
}

func TestBuildCopilotPlan_GeneratedHeaders(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# Rules\n")},
	}

	plan := buildCopilotPlan(repoPath, ".promptherder/agent/rules", sources)

	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}
	assertContains(t, plan[0].Content, "Auto-generated by promptherder")
	assertContains(t, plan[0].Content, "do not edit")
}

// --- RunCopilot integration ---

func TestRunCopilot_DryRun(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-test.md"), "# Test\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   true,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Nothing should be written in dry-run
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error("dry-run should not create copilot-instructions.md")
	}
}

func TestRunCopilot_ActualSync(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n\nBe good.\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n\nSafe bash.\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// GEMINI.md should NOT be created
	if _, err := os.Stat(filepath.Join(dir, "GEMINI.md")); !os.IsNotExist(err) {
		t.Error("should not create GEMINI.md")
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

func TestRunCopilot_NoSources(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error("should not create files when no sources exist")
	}
}

func TestRunCopilot_ValidationError(t *testing.T) {
	cfg := Config{
		RepoPath: "",
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	err := RunCopilot(context.Background(), cfg)
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

// --- manifest & cleanStale ---

func TestCleanStale_RemovesStaleFromOldManifest(t *testing.T) {
	dir := t.TempDir()
	instDir := filepath.Join(dir, ".github", "instructions")
	mustMkdir(t, instDir)

	mustWrite(t, filepath.Join(instDir, "old-rule.instructions.md"), "# Old rule\n")
	mustWrite(t, filepath.Join(instDir, "manual.instructions.md"), "# My custom\n")
	mustWrite(t, filepath.Join(instDir, "README.md"), "# Instructions dir\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	old := manifest{
		Version: 1,
		Files:   []string{".github/instructions/old-rule.instructions.md"},
	}
	new := manifest{Version: 2}

	err := cleanStale(dir, old, new, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(instDir, "old-rule.instructions.md")); !os.IsNotExist(err) {
		t.Error("stale file from old manifest should have been removed")
	}
	if _, err := os.Stat(filepath.Join(instDir, "manual.instructions.md")); err != nil {
		t.Error("manually created file should be preserved")
	}
	if _, err := os.Stat(filepath.Join(instDir, "README.md")); err != nil {
		t.Error("non-instruction file should be preserved")
	}
}

func TestCleanStale_KeepsFilesInBothManifests(t *testing.T) {
	dir := t.TempDir()
	instDir := filepath.Join(dir, ".github", "instructions")
	mustMkdir(t, instDir)

	mustWrite(t, filepath.Join(instDir, "01-shell.instructions.md"), "# Shell\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	old := manifest{Version: 1, Files: []string{".github/instructions/01-shell.instructions.md"}}
	cur := manifest{Version: 2}
	cur.setTarget("copilot", []string{".github/instructions/01-shell.instructions.md"})

	err := cleanStale(dir, old, cur, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(instDir, "01-shell.instructions.md")); err != nil {
		t.Error("file in both manifests should be preserved")
	}
}

func TestCleanStale_DryRunDoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	instDir := filepath.Join(dir, ".github", "instructions")
	mustMkdir(t, instDir)

	stale := filepath.Join(instDir, "old-rule.instructions.md")
	mustWrite(t, stale, "# Old\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	old := manifest{Version: 1, Files: []string{".github/instructions/old-rule.instructions.md"}}
	new := manifest{Version: 2}

	err := cleanStale(dir, old, new, true, logger)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(stale); err != nil {
		t.Error("dry-run should not delete stale files")
	}
}

func TestCleanStale_EmptyOldManifest(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	err := cleanStale(dir, manifest{}, manifest{}, false, logger)
	if err != nil {
		t.Fatal(err)
	}
}

func TestManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	m := manifest{
		Version:     2,
		GeneratedAt: "2026-02-07T00:00:00Z",
	}
	m.setTarget("copilot", []string{
		".github/copilot-instructions.md",
		".github/instructions/01-shell.instructions.md",
	})

	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	got := readManifest(dir, logger)
	if got.Version != 2 {
		t.Errorf("version = %d, want 2", got.Version)
	}
	copilotFiles := got.Targets["copilot"]
	if len(copilotFiles) != 2 {
		t.Fatalf("copilot files count = %d, want 2", len(copilotFiles))
	}
	if copilotFiles[0] != ".github/copilot-instructions.md" {
		t.Errorf("copilot files[0] = %q", copilotFiles[0])
	}
}

func TestManifest_ReadMissing(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	m := readManifest(dir, logger)
	if len(m.allFiles()) != 0 {
		t.Errorf("expected empty files, got %d", len(m.allFiles()))
	}
}

func TestManifest_AllFiles_V1Compat(t *testing.T) {
	m := manifest{
		Version: 1,
		Files:   []string{".github/copilot-instructions.md"},
	}
	files := m.allFiles()
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0] != ".github/copilot-instructions.md" {
		t.Errorf("files[0] = %q", files[0])
	}
}

func TestManifest_AllFiles_V2(t *testing.T) {
	m := manifest{Version: 2}
	m.setTarget("copilot", []string{".github/copilot-instructions.md"})
	m.setTarget("antigravity", []string{".agent/rules/00-general.md"})

	files := m.allFiles()
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
}

func TestManifest_IsGenerated(t *testing.T) {
	m := manifest{Generated: []string{"stack.md", "structure.md"}}
	if !m.isGenerated("stack.md") {
		t.Error("stack.md should be generated")
	}
	if m.isGenerated("browser.md") {
		t.Error("browser.md should not be generated")
	}
}

func TestRunCopilot_WritesManifest(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	m := readManifest(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if m.Version != 2 {
		t.Errorf("manifest version = %d, want 2", m.Version)
	}
	copilotFiles := m.Targets["copilot"]
	if len(copilotFiles) != 1 {
		t.Fatalf("manifest copilot files = %d, want 1", len(copilotFiles))
	}
	if copilotFiles[0] != ".github/copilot-instructions.md" {
		t.Errorf("manifest copilot files[0] = %q", copilotFiles[0])
	}
}

func TestRunCopilot_NoSources_CleansUpStaleFiles(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := Config{RepoPath: dir, Logger: logger}

	// First run: creates output + manifest.
	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); err != nil {
		t.Fatal("output should exist after first run")
	}

	// Remove all sources.
	os.Remove(filepath.Join(rulesDir, "00-general.md"))

	// Second run: should clean up stale output.
	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error("stale output should be removed when sources are gone")
	}
}

func TestRunCopilot_IdempotentCleanup(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := Config{RepoPath: dir, Logger: logger}

	// First run: creates both outputs.
	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	inst := filepath.Join(dir, ".github", "instructions", "01-shell.instructions.md")
	if _, err := os.Stat(inst); err != nil {
		t.Fatal("instruction file should exist after first run")
	}

	// Remove source B.
	os.Remove(filepath.Join(rulesDir, "01-shell.md"))

	// Second run: should remove B's output, keep A's.
	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(inst); !os.IsNotExist(err) {
		t.Error("removed source's instruction file should be cleaned up")
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); err != nil {
		t.Error("remaining source's output should still exist")
	}

	// Manifest should reflect only A.
	m := readManifest(dir, logger)
	copilotFiles := m.Targets["copilot"]
	if len(copilotFiles) != 1 {
		t.Fatalf("manifest should have 1 copilot file, got %d", len(copilotFiles))
	}
}

// --- workflow → prompt conversion ---

func TestConvertWorkflowToPrompt_Basic(t *testing.T) {
	input := []byte("---\ndescription: Run a review pass.\n---\n\n# Review\n\nDo the review.\n")
	result := convertWorkflowToPrompt("test/workflows", "review.md", input)

	assertContains(t, result, `mode: "agent"`)
	assertContains(t, result, `description: "Run a review pass."`)
	assertContains(t, result, "# Review")
	assertContains(t, result, "Auto-generated by promptherder")
	assertNotContains(t, result, "description: Run a review pass.") // raw form should be quoted
}

func TestConvertWorkflowToPrompt_StripsTurbo(t *testing.T) {
	input := []byte("---\ndescription: Execute plan.\n---\n\n// turbo-all\n\n# Execute\n\nDo stuff.\n")
	result := convertWorkflowToPrompt("test/workflows", "execute.md", input)

	assertNotContains(t, result, "// turbo-all")
	assertContains(t, result, "# Execute")
}

func TestStripAntigravityAnnotations(t *testing.T) {
	input := []byte("line1\n// turbo\nline2\n// turbo-all\nline3\n")
	result := stripAntigravityAnnotations(input)
	s := string(result)

	if strings.Contains(s, "// turbo") {
		t.Errorf("should strip turbo annotations, got: %s", s)
	}
	if !strings.Contains(s, "line1") || !strings.Contains(s, "line2") || !strings.Contains(s, "line3") {
		t.Errorf("should keep non-annotation lines, got: %s", s)
	}
}

func TestExtractDescription(t *testing.T) {
	input := []byte("---\ndescription: My workflow.\n---\n# Body\n")
	desc := extractDescription(input)
	if desc != "My workflow." {
		t.Errorf("desc = %q, want %q", desc, "My workflow.")
	}
}

func TestExtractDescription_NoFrontmatter(t *testing.T) {
	input := []byte("# Just body\n")
	desc := extractDescription(input)
	if desc != "" {
		t.Errorf("desc = %q, want empty", desc)
	}
}

func TestBuildCopilotPrompts_Basic(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".promptherder", "agent", "workflows")
	mustMkdir(t, wfDir)
	mustWrite(t, filepath.Join(wfDir, "brainstorm.md"), "---\ndescription: Brainstorm.\n---\n\n# Brainstorm\n\nDo it.\n")
	mustWrite(t, filepath.Join(wfDir, "review.md"), "---\ndescription: Review.\n---\n\n# Review\n\nCheck.\n")

	items, err := buildCopilotPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 prompt items, got %d", len(items))
	}

	// Check filenames.
	assertTarget(t, items[0], filepath.Join(dir, ".github", "prompts", "brainstorm.prompt.md"))
	assertTarget(t, items[1], filepath.Join(dir, ".github", "prompts", "review.prompt.md"))
}

func TestBuildCopilotPrompts_MissingDir(t *testing.T) {
	dir := t.TempDir()
	items, err := buildCopilotPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items for missing dir, got %d", len(items))
	}
}

func TestRunCopilot_WithWorkflows(t *testing.T) {
	dir := t.TempDir()

	// Set up rules.
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")

	// Set up workflows.
	wfDir := filepath.Join(dir, ".promptherder", "agent", "workflows")
	mustMkdir(t, wfDir)
	mustWrite(t, filepath.Join(wfDir, "brainstorm.md"), "---\ndescription: Brainstorm.\n---\n\n# Brainstorm\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Check prompt file was created.
	promptFile := filepath.Join(dir, ".github", "prompts", "brainstorm.prompt.md")
	data, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("prompt file not created: %v", err)
	}
	assertContains(t, data, `mode: "agent"`)
	assertContains(t, data, "# Brainstorm")

	// Check manifest includes prompt file.
	m := readManifest(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	copilotFiles := m.Targets["copilot"]
	found := false
	for _, f := range copilotFiles {
		if f == ".github/prompts/brainstorm.prompt.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("manifest should include prompt file, got: %v", copilotFiles)
	}
}

// --- skill → prompt conversion ---

func TestBuildCopilotSkillPrompts_Basic(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "compound-v-tdd")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: compound-v-tdd\ndescription: TDD skill.\n---\n\n# TDD\n\nRed green refactor.\n")

	items, err := buildCopilotSkillPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill prompt, got %d", len(items))
	}

	assertTarget(t, items[0], filepath.Join(dir, ".github", "prompts", "compound-v-tdd.prompt.md"))
	assertContains(t, items[0].Content, `mode: "agent"`)
	assertContains(t, items[0].Content, `description: "TDD skill."`)
	assertContains(t, items[0].Content, "# TDD")
	assertContains(t, items[0].Content, "Red green refactor")
}

func TestBuildCopilotSkillPrompts_MissingDir(t *testing.T) {
	dir := t.TempDir()
	items, err := buildCopilotSkillPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestBuildCopilotSkillPrompts_SkipsDirWithoutSKILLmd(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "empty-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "README.md"), "# Not a skill\n")

	items, err := buildCopilotSkillPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items for dir without SKILL.md, got %d", len(items))
	}
}

func TestRunCopilot_WithSkills(t *testing.T) {
	dir := t.TempDir()

	// Set up a skill.
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-review")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: my-review\ndescription: Review code.\n---\n\n# Review\n\nCheck things.\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := RunCopilot(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Check prompt file was created.
	promptFile := filepath.Join(dir, ".github", "prompts", "my-review.prompt.md")
	data, err := os.ReadFile(promptFile)
	if err != nil {
		t.Fatalf("skill prompt file not created: %v", err)
	}
	assertContains(t, data, `mode: "agent"`)
	assertContains(t, data, "# Review")

	// Check manifest.
	m := readManifest(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	copilotFiles := m.Targets["copilot"]
	found := false
	for _, f := range copilotFiles {
		if f == ".github/prompts/my-review.prompt.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("manifest should include skill prompt file, got: %v", copilotFiles)
	}
}

func TestBuildCopilotSkillPrompts_PrefersCopilotVariant(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: my-skill\ndescription: Generic skill.\n---\n\n# Generic\n\nGeneric content.\n")
	mustWrite(t, filepath.Join(skillDir, "COPILOT.md"), "---\nname: my-skill\ndescription: Copilot skill.\n---\n\n# Copilot\n\nCopilot-specific content.\n")

	items, err := buildCopilotSkillPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill prompt, got %d", len(items))
	}

	// Should use COPILOT.md content, not SKILL.md.
	assertContains(t, items[0].Content, "Copilot-specific content")
	assertNotContains(t, items[0].Content, "Generic content")
	assertContains(t, items[0].Content, `description: "Copilot skill."`)

	// Source label should reference COPILOT.md.
	assertContains(t, items[0].Content, "my-skill/COPILOT.md")
}

func TestBuildCopilotSkillPrompts_FallsBackToGeneric(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: my-skill\ndescription: Generic skill.\n---\n\n# Generic\n\nGeneric content.\n")

	items, err := buildCopilotSkillPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill prompt, got %d", len(items))
	}

	// Should use SKILL.md when no COPILOT.md exists.
	assertContains(t, items[0].Content, "Generic content")
	assertContains(t, items[0].Content, "my-skill/SKILL.md")
}

func TestBuildCopilotSkillPrompts_IgnoresAntigravityVariant(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "---\nname: my-skill\ndescription: Generic.\n---\n\n# Generic\n")
	mustWrite(t, filepath.Join(skillDir, "ANTIGRAVITY.md"), "---\nname: my-skill\ndescription: Antigravity.\n---\n\n# Antigravity\n")

	items, err := buildCopilotSkillPrompts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 skill prompt, got %d", len(items))
	}

	// Should use generic SKILL.md, not ANTIGRAVITY.md (that's for Antigravity target).
	assertContains(t, items[0].Content, "# Generic")
	assertNotContains(t, items[0].Content, "# Antigravity")
}
