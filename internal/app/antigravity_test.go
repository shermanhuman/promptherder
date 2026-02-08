package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAntigravityTarget_BasicInstall(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent")
	rulesDir := filepath.Join(srcDir, "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "00-test.md"), "# Test\n")
	mustWrite(t, filepath.Join(rulesDir, "01-shell.md"), "# Shell\n")

	skillsDir := filepath.Join(srcDir, "skills", "my-skill")
	mustMkdir(t, skillsDir)
	mustWrite(t, filepath.Join(skillsDir, "SKILL.md"), "# Skill\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(installed) != 3 {
		t.Fatalf("expected 3 files installed, got %d", len(installed))
	}

	targetRulesDir := filepath.Join(dir, ".agent", "rules")
	if _, err := os.Stat(filepath.Join(targetRulesDir, "00-test.md")); err != nil {
		t.Error("00-test.md should be copied to .agent/rules/")
	}
	if _, err := os.Stat(filepath.Join(targetRulesDir, "01-shell.md")); err != nil {
		t.Error("01-shell.md should be copied to .agent/rules/")
	}

	targetSkillFile := filepath.Join(dir, ".agent", "skills", "my-skill", "SKILL.md")
	if _, err := os.Stat(targetSkillFile); err != nil {
		t.Error("SKILL.md should be copied to .agent/skills/my-skill/")
	}
}

func TestAntigravityTarget_SkipsGeneratedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "stack.md"), "# Stack\n")
	mustWrite(t, filepath.Join(srcDir, "normal.md"), "# Normal\n")

	targetDir := filepath.Join(dir, ".agent", "rules")
	mustMkdir(t, targetDir)
	mustWrite(t, filepath.Join(targetDir, "stack.md"), "# Existing Stack\n")

	m := manifest{Version: 2, Generated: []string{"stack.md"}}
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(installed) != 1 {
		t.Fatalf("expected 1 file installed (skipping stack.md), got %d", len(installed))
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "stack.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Existing Stack") {
		t.Error("generated file stack.md should not be overwritten")
	}

	if _, err := os.Stat(filepath.Join(targetDir, "normal.md")); err != nil {
		t.Error("normal.md should be copied")
	}
}

func TestAntigravityTarget_GeneratedFileFirstInstall(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "stack.md"), "# Stack\n")

	m := manifest{Version: 2, Generated: []string{"stack.md"}}
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(installed) != 1 {
		t.Fatalf("expected 1 file installed, got %d", len(installed))
	}

	targetFile := filepath.Join(dir, ".agent", "rules", "stack.md")
	if _, err := os.Stat(targetFile); err != nil {
		t.Error("stack.md should be installed on first run even if marked generated")
	}
}

func TestAntigravityTarget_DryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "test.md"), "# Test\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		DryRun:   true,
		Logger:   testLogger(t),
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(installed) != 1 {
		t.Fatalf("dry-run should return files list, got %d", len(installed))
	}

	targetFile := filepath.Join(dir, ".agent", "rules", "test.md")
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
}

func TestAntigravityTarget_MissingSource(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatalf("missing source should not error, got: %v", err)
	}

	if len(installed) != 0 {
		t.Errorf("missing source should return empty list, got %d files", len(installed))
	}
}

func TestAntigravityTarget_ContextCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "test.md"), "# Test\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	_, err := target.Install(ctx, cfg)
	if err == nil {
		t.Error("cancelled context should return error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestAntigravityTarget_PreservesDirectoryStructure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent")
	rulesDir := filepath.Join(srcDir, "rules", "subdir")
	skillsDir := filepath.Join(srcDir, "skills", "skill-a", "examples")
	mustMkdir(t, rulesDir)
	mustMkdir(t, skillsDir)

	mustWrite(t, filepath.Join(rulesDir, "nested.md"), "# Nested\n")
	mustWrite(t, filepath.Join(skillsDir, "example.md"), "# Example\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(installed) != 2 {
		t.Fatalf("expected 2 files, got %d", len(installed))
	}

	if _, err := os.Stat(filepath.Join(dir, ".agent", "rules", "subdir", "nested.md")); err != nil {
		t.Error("nested directory structure should be preserved")
	}
	if _, err := os.Stat(filepath.Join(dir, ".agent", "skills", "skill-a", "examples", "example.md")); err != nil {
		t.Error("deep nested structure should be preserved")
	}
}

func TestAntigravityTarget_Name(t *testing.T) {
	t.Parallel()
	target := AntigravityTarget{}
	if target.Name() != "antigravity" {
		t.Errorf("Name() = %q, want %q", target.Name(), "antigravity")
	}
}

func TestAntigravityTarget_SkillVariant_PrefersAntigravityMD(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# Generic Skill\n")
	mustWrite(t, filepath.Join(skillDir, "ANTIGRAVITY.md"), "# Antigravity-Specific Skill\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Should install ANTIGRAVITY.md as SKILL.md, not the generic.
	targetFile := filepath.Join(dir, ".agent", "skills", "my-skill", "SKILL.md")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("SKILL.md should exist at target: %v", err)
	}
	if !strings.Contains(string(data), "Antigravity-Specific") {
		t.Errorf("target SKILL.md should contain variant content, got: %s", data)
	}
	if strings.Contains(string(data), "Generic Skill") {
		t.Error("target SKILL.md should NOT contain generic content")
	}

	// ANTIGRAVITY.md should NOT appear at the target.
	antigravityFile := filepath.Join(dir, ".agent", "skills", "my-skill", "ANTIGRAVITY.md")
	if _, err := os.Stat(antigravityFile); !os.IsNotExist(err) {
		t.Error("ANTIGRAVITY.md should not be copied to target — it's installed as SKILL.md")
	}

	// Verify installed list contains the SKILL.md path, not ANTIGRAVITY.md.
	for _, f := range installed {
		if strings.Contains(f, "ANTIGRAVITY.md") {
			t.Errorf("installed list should not contain ANTIGRAVITY.md, got: %v", installed)
		}
	}
}

func TestAntigravityTarget_SkillVariant_SkipsCopilotMD(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# Generic Skill\n")
	mustWrite(t, filepath.Join(skillDir, "COPILOT.md"), "# Copilot-Specific Skill\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// COPILOT.md should NOT be installed.
	copilotFile := filepath.Join(dir, ".agent", "skills", "my-skill", "COPILOT.md")
	if _, err := os.Stat(copilotFile); !os.IsNotExist(err) {
		t.Error("COPILOT.md should not be copied to Antigravity target")
	}

	// Generic SKILL.md should be installed (no ANTIGRAVITY.md variant).
	targetFile := filepath.Join(dir, ".agent", "skills", "my-skill", "SKILL.md")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("SKILL.md should exist at target: %v", err)
	}
	if !strings.Contains(string(data), "Generic Skill") {
		t.Errorf("should fall back to generic SKILL.md, got: %s", data)
	}

	// Verify installed list doesn't contain COPILOT.md.
	for _, f := range installed {
		if strings.Contains(f, "COPILOT.md") {
			t.Errorf("installed list should not contain COPILOT.md, got: %v", installed)
		}
	}
}

func TestAntigravityTarget_SkillVariant_FallsBackToGeneric(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# Generic Skill\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Generic SKILL.md should be installed when no variant exists.
	targetFile := filepath.Join(dir, ".agent", "skills", "my-skill", "SKILL.md")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("SKILL.md should exist at target: %v", err)
	}
	if !strings.Contains(string(data), "Generic Skill") {
		t.Errorf("should use generic SKILL.md, got: %s", data)
	}

	if len(installed) != 1 {
		t.Errorf("expected 1 file installed, got %d", len(installed))
	}
}

func TestAntigravityTarget_SkillVariant_AllThreeFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# Generic\n")
	mustWrite(t, filepath.Join(skillDir, "ANTIGRAVITY.md"), "# Antigravity\n")
	mustWrite(t, filepath.Join(skillDir, "COPILOT.md"), "# Copilot\n")

	target := AntigravityTarget{}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Only ANTIGRAVITY.md content should be installed as SKILL.md.
	targetFile := filepath.Join(dir, ".agent", "skills", "my-skill", "SKILL.md")
	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Antigravity") {
		t.Errorf("should install variant, got: %s", data)
	}

	// Only one file should be installed for this skill dir.
	skillFiles := 0
	for _, f := range installed {
		if strings.Contains(f, "my-skill") {
			skillFiles++
		}
	}
	if skillFiles != 1 {
		t.Errorf("expected 1 skill file installed, got %d (installed: %v)", skillFiles, installed)
	}
}

func TestIsInSkillDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want bool
	}{
		{"skills/compound-v-tdd/SKILL.md", true},
		{"skills/my-skill/ANTIGRAVITY.md", true},
		{"skills/my-skill/subdir/file.md", true},
		{"rules/compound-v.md", false},
		{"skills/README.md", false}, // not inside a skill subdir
		{"workflows/plan.md", false},
	}
	for _, tt := range tests {
		if got := isInSkillDir(tt.path); got != tt.want {
			t.Errorf("isInSkillDir(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
