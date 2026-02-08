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
	rulesDir := filepath.Join(dir, ".agent", "rules")
	mustMkdir(t, rulesDir)

	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n\nBe good.\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n\nSafe bash.\n")

	sources, err := readSources(dir, ".agent/rules", nil)
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
	sources, err := readSources(dir, ".agent/rules", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("expected 0 sources, got %d", len(sources))
	}
}

func TestReadSources_IncludeFilter(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".agent", "rules")
	mustMkdir(t, rulesDir)

	mustWrite(t, filepath.Join(rulesDir, "keep.md"), "# Keep\n")
	mustWrite(t, filepath.Join(rulesDir, "skip.txt"), "skip\n")

	sources, err := readSources(dir, ".agent/rules", []string{"**/*.md"})
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

	plan := buildPlan(repoPath, ".agent/rules", sources)

	// Should produce 1 item: copilot-instructions.md (no GEMINI.md)
	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, ".github", "copilot-instructions.md"))

	// copilot-instructions.md should contain both
	assertContains(t, plan[0].Content, "# General")
	assertContains(t, plan[0].Content, "# Ops")
}

func TestBuildPlan_WithApplyTo(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# General\n")},
		{Name: "01-shell", ApplyTo: "**/*.sh", Body: []byte("# Shell\n")},
	}

	plan := buildPlan(repoPath, ".agent/rules", sources)

	// copilot-instructions.md + 01-shell.instructions.md = 2 (no GEMINI.md)
	if len(plan) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan))
	}

	// copilot-instructions.md gets only repo-wide (no applyTo)
	assertTarget(t, plan[0], filepath.Join(repoPath, ".github", "copilot-instructions.md"))
	assertContains(t, plan[0].Content, "# General")
	assertNotContains(t, plan[0].Content, "# Shell")

	// 01-shell.instructions.md gets its own file with frontmatter
	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md"))
	assertContains(t, plan[1].Content, `applyTo: "**/*.sh"`)
	assertContains(t, plan[1].Content, "# Shell")
}

func TestBuildPlan_AllWithApplyTo(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-yaml", ApplyTo: "**/*.yaml", Body: []byte("# YAML\n")},
		{Name: "01-shell", ApplyTo: "**/*.sh", Body: []byte("# Shell\n")},
	}

	plan := buildPlan(repoPath, ".agent/rules", sources)

	// 2 instruction files only (no GEMINI.md, no copilot-instructions.md since all have applyTo)
	if len(plan) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan))
	}

	assertTarget(t, plan[0], filepath.Join(repoPath, ".github", "instructions", "00-yaml.instructions.md"))
	assertTarget(t, plan[1], filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md"))
}

func TestBuildPlan_GeneratedHeaders(t *testing.T) {
	repoPath := "/repo"
	sources := []sourceFile{
		{Name: "00-general", Body: []byte("# Rules\n")},
	}

	plan := buildPlan(repoPath, ".agent/rules", sources)

	// Only copilot-instructions.md (no GEMINI.md)
	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}
	assertContains(t, plan[0].Content, "Auto-generated by promptherder")
	assertContains(t, plan[0].Content, "do not edit")
}

// --- Run integration ---

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".agent", "rules")
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
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error("dry-run should not create copilot-instructions.md")
	}
}

func TestRun_ActualSync(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".agent", "rules")
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

	// GEMINI.md should NOT be created (agent reads .agent/rules/ natively)
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

func TestRun_NoSources(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
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

// --- manifest & cleanStale ---

func TestCleanStale_RemovesStaleFromOldManifest(t *testing.T) {
	dir := t.TempDir()
	instDir := filepath.Join(dir, ".github", "instructions")
	mustMkdir(t, instDir)

	// Create a file that was in the old manifest but won't be in the new one.
	mustWrite(t, filepath.Join(instDir, "old-rule.instructions.md"), "# Old rule\n")

	// Create a file NOT in any manifest (user-created).
	mustWrite(t, filepath.Join(instDir, "manual.instructions.md"), "# My custom\n")

	// Create a non-instruction file.
	mustWrite(t, filepath.Join(instDir, "README.md"), "# Instructions dir\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	old := manifest{
		Version: 1,
		Files:   []string{".github/instructions/old-rule.instructions.md"},
	}
	new := manifest{Version: 1, Files: nil} // empty = nothing planned

	err := cleanStale(dir, old, new, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Stale file (in old manifest, not in new) should be removed.
	if _, err := os.Stat(filepath.Join(instDir, "old-rule.instructions.md")); !os.IsNotExist(err) {
		t.Error("stale file from old manifest should have been removed")
	}

	// Manual file (not in any manifest) should be preserved.
	if _, err := os.Stat(filepath.Join(instDir, "manual.instructions.md")); err != nil {
		t.Error("manually created file should be preserved")
	}

	// Non-instruction file should be preserved.
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
	new := manifest{Version: 1, Files: []string{".github/instructions/01-shell.instructions.md"}}

	err := cleanStale(dir, old, new, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	// File in both manifests should still exist.
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
	new := manifest{Version: 1, Files: nil}

	err := cleanStale(dir, old, new, true, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Dry-run should NOT delete.
	if _, err := os.Stat(stale); err != nil {
		t.Error("dry-run should not delete stale files")
	}
}

func TestCleanStale_EmptyOldManifest(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Empty old manifest = nothing to clean.
	err := cleanStale(dir, manifest{}, manifest{}, false, logger)
	if err != nil {
		t.Fatal(err)
	}
}

func TestManifest_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	m := manifest{
		Version:     1,
		SourceDir:   ".agent/rules",
		GeneratedAt: "2026-02-07T00:00:00Z",
		Files: []string{
			".github/copilot-instructions.md",
			".github/instructions/01-shell.instructions.md",
		},
	}

	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	got := readManifest(dir, logger)
	if got.Version != 1 {
		t.Errorf("version = %d, want 1", got.Version)
	}
	if got.SourceDir != ".agent/rules" {
		t.Errorf("source_dir = %q, want %q", got.SourceDir, ".agent/rules")
	}
	if len(got.Files) != 2 {
		t.Fatalf("files count = %d, want 2", len(got.Files))
	}
	if got.Files[0] != ".github/copilot-instructions.md" {
		t.Errorf("files[0] = %q", got.Files[0])
	}
}

func TestManifest_ReadMissing(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	m := readManifest(dir, logger)
	if len(m.Files) != 0 {
		t.Errorf("expected empty files, got %d", len(m.Files))
	}
}

func TestBuildManifest(t *testing.T) {
	repoPath := t.TempDir()
	plan := []planItem{
		{Target: filepath.Join(repoPath, ".github", "copilot-instructions.md")},
		{Target: filepath.Join(repoPath, ".github", "instructions", "01-shell.instructions.md")},
	}

	m := buildManifest(repoPath, ".agent/rules", plan)
	if m.Version != 1 {
		t.Errorf("version = %d, want 1", m.Version)
	}
	if len(m.Files) != 2 {
		t.Fatalf("files count = %d, want 2", len(m.Files))
	}
	// Files should be repo-relative with forward slashes.
	if m.Files[0] != ".github/copilot-instructions.md" {
		t.Errorf("files[0] = %q", m.Files[0])
	}
}

func TestRun_WritesManifest(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")

	cfg := Config{
		RepoPath: dir,
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Manifest should exist.
	m := readManifest(dir, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if m.Version != 1 {
		t.Errorf("manifest version = %d, want 1", m.Version)
	}
	if len(m.Files) != 1 {
		t.Fatalf("manifest files = %d, want 1", len(m.Files))
	}
	if m.Files[0] != ".github/copilot-instructions.md" {
		t.Errorf("manifest files[0] = %q", m.Files[0])
	}
}

func TestRun_NoSources_CleansUpStaleFiles(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := Config{RepoPath: dir, Logger: logger}

	// First run: creates output + manifest.
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); err != nil {
		t.Fatal("output should exist after first run")
	}

	// Remove all sources.
	os.Remove(filepath.Join(rulesDir, "00-general.md"))

	// Second run: should clean up stale output.
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".github", "copilot-instructions.md")); !os.IsNotExist(err) {
		t.Error("stale output should be removed when sources are gone")
	}

	// Manifest should exist but be empty.
	m := readManifest(dir, logger)
	if len(m.Files) != 0 {
		t.Errorf("manifest should have 0 files, got %d", len(m.Files))
	}
}

func TestRun_IdempotentCleanup(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, ".agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-general.md"), "# General\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "---\napplyTo: \"**/*.sh\"\n---\n# Shell\n")

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := Config{RepoPath: dir, Logger: logger}

	// First run: creates both outputs.
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	inst := filepath.Join(dir, ".github", "instructions", "01-shell.instructions.md")
	if _, err := os.Stat(inst); err != nil {
		t.Fatal("instruction file should exist after first run")
	}

	// Remove source B.
	os.Remove(filepath.Join(rulesDir, "01-shell.md"))

	// Second run: should remove B's output, keep A's.
	if err := Run(context.Background(), cfg); err != nil {
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
	if len(m.Files) != 1 {
		t.Fatalf("manifest should have 1 file, got %d", len(m.Files))
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
