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
