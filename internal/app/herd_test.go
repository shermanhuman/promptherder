package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverHerds_Empty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herds, err := discoverHerds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(herds) != 0 {
		t.Errorf("expected 0 herds, got %d", len(herds))
	}
}

func TestDiscoverHerds_FindsHerds(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create two herds.
	for _, name := range []string{"alpha-rules", "beta-rules"} {
		herdDir := filepath.Join(dir, herdsDir, name)
		mustMkdir(t, herdDir)
		meta := HerdMeta{Name: name}
		data, _ := json.Marshal(meta)
		mustWrite(t, filepath.Join(herdDir, "herd.json"), string(data))
	}

	herds, err := discoverHerds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(herds) != 2 {
		t.Fatalf("expected 2 herds, got %d", len(herds))
	}
	// Should be sorted alphabetically.
	if herds[0].Meta.Name != "alpha-rules" {
		t.Errorf("first herd = %q, want alpha-rules", herds[0].Meta.Name)
	}
	if herds[1].Meta.Name != "beta-rules" {
		t.Errorf("second herd = %q, want beta-rules", herds[1].Meta.Name)
	}
}

func TestDiscoverHerds_SkipsDirsWithoutMeta(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Dir without herd.json.
	mustMkdir(t, filepath.Join(dir, herdsDir, "no-meta"))
	// Dir with herd.json.
	withMeta := filepath.Join(dir, herdsDir, "has-meta")
	mustMkdir(t, withMeta)
	mustWrite(t, filepath.Join(withMeta, "herd.json"), `{"name":"has-meta"}`)

	herds, err := discoverHerds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(herds) != 1 {
		t.Fatalf("expected 1 herd, got %d", len(herds))
	}
	if herds[0].Meta.Name != "has-meta" {
		t.Errorf("herd name = %q, want has-meta", herds[0].Meta.Name)
	}
}

func TestDiscoverHerds_UseDirNameIfNameEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herdDir := filepath.Join(dir, herdsDir, "my-herd")
	mustMkdir(t, herdDir)
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{}`)

	herds, err := discoverHerds(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(herds) != 1 {
		t.Fatalf("expected 1 herd, got %d", len(herds))
	}
	if herds[0].Meta.Name != "my-herd" {
		t.Errorf("herd name = %q, want my-herd", herds[0].Meta.Name)
	}
}

func TestMergeHerds_SingleHerd(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a herd with some files.
	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustMkdir(t, filepath.Join(herdDir, "skills", "my-skill"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "foo.md"), "# Foo\n")
	mustWrite(t, filepath.Join(herdDir, "skills", "my-skill", "SKILL.md"), "# Skill\n")

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2}

	installed, err := mergeHerds(context.Background(), dir, herds, m, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected 2 installed files, got %d", len(installed))
	}

	// Verify files exist in .promptherder/agent/.
	agentRoot := filepath.Join(dir, agentDir)
	if _, err := os.Stat(filepath.Join(agentRoot, "rules", "foo.md")); err != nil {
		t.Error("rules/foo.md should be installed")
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "skills", "my-skill", "SKILL.md")); err != nil {
		t.Error("skills/my-skill/SKILL.md should be installed")
	}
}

func TestMergeHerds_ConflictDetection(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Two herds with the same file.
	for _, name := range []string{"herd-a", "herd-b"} {
		herdDir := filepath.Join(dir, herdsDir, name)
		mustMkdir(t, filepath.Join(herdDir, "rules"))
		mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"`+name+`"}`)
		mustWrite(t, filepath.Join(herdDir, "rules", "conflict.md"), "# From "+name+"\n")
	}

	herds, err := discoverHerds(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2}

	_, err = mergeHerds(context.Background(), dir, herds, m, cfg)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("error should mention conflict, got: %v", err)
	}
	if !strings.Contains(err.Error(), "herd-a") || !strings.Contains(err.Error(), "herd-b") {
		t.Errorf("error should name both herds, got: %v", err)
	}
}

func TestMergeHerds_SkipsHerdJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "rule.md"), "# Rule\n")

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2}

	installed, err := mergeHerds(context.Background(), dir, herds, m, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Only rules/rule.md, not herd.json.
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed file, got %d: %v", len(installed), installed)
	}

	// herd.json should NOT be in .promptherder/agent/.
	agentRoot := filepath.Join(dir, agentDir)
	if _, err := os.Stat(filepath.Join(agentRoot, "herd.json")); !os.IsNotExist(err) {
		t.Error("herd.json should not be copied to agent dir")
	}
}

func TestMergeHerds_SkipsGeneratedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Pre-create a generated file in agent dir.
	agentRoot := filepath.Join(dir, agentDir, "rules")
	mustMkdir(t, agentRoot)
	mustWrite(t, filepath.Join(agentRoot, "stack.md"), "# Existing Stack\n")

	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "stack.md"), "# Herd Stack\n")
	mustWrite(t, filepath.Join(herdDir, "rules", "other.md"), "# Other\n")

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2, Generated: []string{"stack.md"}}
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	installed, err := mergeHerds(context.Background(), dir, herds, m, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Only other.md should be installed (stack.md is generated and exists).
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed file, got %d: %v", len(installed), installed)
	}

	// Verify stack.md was NOT overwritten.
	data, err := os.ReadFile(filepath.Join(agentRoot, "stack.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Existing Stack") {
		t.Error("generated stack.md should not be overwritten")
	}
}

func TestMergeHerds_DryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "rule.md"), "# Rule\n")

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, DryRun: true, Logger: testLogger(t)}
	m := manifest{Version: 2}

	installed, err := mergeHerds(context.Background(), dir, herds, m, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("dry-run should return file list, got %d", len(installed))
	}

	// File should NOT actually exist.
	agentRoot := filepath.Join(dir, agentDir)
	if _, err := os.Stat(filepath.Join(agentRoot, "rules", "rule.md")); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
}

func TestMergeHerds_ContextCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "rule.md"), "# Rule\n")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2}

	_, err := mergeHerds(ctx, dir, herds, m, cfg)
	if err == nil {
		t.Error("cancelled context should return error")
	}
}

func TestCleanAgentDir_RemovesTrackedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a file in agent dir.
	agentRoot := filepath.Join(dir, agentDir, "rules")
	mustMkdir(t, agentRoot)
	mustWrite(t, filepath.Join(agentRoot, "old-rule.md"), "# Old\n")

	prev := manifest{
		Version: 2,
		Targets: map[string][]string{
			"herds": {".promptherder/agent/rules/old-rule.md"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := cleanAgentDir(dir, prev, false, logger); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(agentRoot, "old-rule.md")); !os.IsNotExist(err) {
		t.Error("old-rule.md should have been removed")
	}
}

func TestCleanAgentDir_PreservesGeneratedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	agentRoot := filepath.Join(dir, agentDir, "rules")
	mustMkdir(t, agentRoot)
	mustWrite(t, filepath.Join(agentRoot, "stack.md"), "# Stack\n")

	prev := manifest{
		Version:   2,
		Generated: []string{"stack.md"},
		Targets: map[string][]string{
			"herds": {".promptherder/agent/rules/stack.md"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := cleanAgentDir(dir, prev, false, logger); err != nil {
		t.Fatal(err)
	}

	// stack.md should still exist.
	if _, err := os.Stat(filepath.Join(agentRoot, "stack.md")); err != nil {
		t.Error("generated stack.md should be preserved")
	}
}

func TestMergeHerds_SkipsGitDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustMkdir(t, filepath.Join(herdDir, ".git", "objects"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "rule.md"), "# Rule\n")
	mustWrite(t, filepath.Join(herdDir, ".git", "HEAD"), "ref: refs/heads/master\n")
	mustWrite(t, filepath.Join(herdDir, ".git", "objects", "pack.idx"), "binary")

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2}

	installed, err := mergeHerds(context.Background(), dir, herds, m, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Only rules/rule.md, no .git files.
	if len(installed) != 1 {
		t.Fatalf("expected 1 installed file, got %d: %v", len(installed), installed)
	}

	// .git files should NOT exist in agent dir.
	agentRoot := filepath.Join(dir, agentDir)
	if _, err := os.Stat(filepath.Join(agentRoot, ".git", "HEAD")); !os.IsNotExist(err) {
		t.Error(".git/HEAD should not be copied to agent dir")
	}
	if _, err := os.Stat(filepath.Join(agentRoot, ".git", "objects", "pack.idx")); !os.IsNotExist(err) {
		t.Error(".git/objects/pack.idx should not be copied to agent dir")
	}
}

func TestMergeHerds_SkipsNonContentDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	herdDir := filepath.Join(dir, herdsDir, "test-herd")
	mustMkdir(t, filepath.Join(herdDir, "rules"))
	mustMkdir(t, filepath.Join(herdDir, "skills", "my-skill"))
	mustMkdir(t, filepath.Join(herdDir, "workflows"))
	mustWrite(t, filepath.Join(herdDir, "herd.json"), `{"name":"test-herd"}`)
	mustWrite(t, filepath.Join(herdDir, "rules", "rule.md"), "# Rule\n")
	mustWrite(t, filepath.Join(herdDir, "skills", "my-skill", "SKILL.md"), "# Skill\n")
	mustWrite(t, filepath.Join(herdDir, "workflows", "plan.md"), "# Plan\n")
	// Non-content files that should be excluded:
	mustWrite(t, filepath.Join(herdDir, "README.md"), "# README\n")
	mustWrite(t, filepath.Join(herdDir, "LICENSE"), "MIT\n")

	herds := []herdOnDisk{
		{Meta: HerdMeta{Name: "test-herd"}, Path: herdDir},
	}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}
	m := manifest{Version: 2}

	installed, err := mergeHerds(context.Background(), dir, herds, m, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Only 3 files from rules/, skills/, workflows/ — not README.md or LICENSE.
	if len(installed) != 3 {
		t.Fatalf("expected 3 installed files, got %d: %v", len(installed), installed)
	}

	agentRoot := filepath.Join(dir, agentDir)
	// Content files should exist.
	if _, err := os.Stat(filepath.Join(agentRoot, "rules", "rule.md")); err != nil {
		t.Error("rules/rule.md should be installed")
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "skills", "my-skill", "SKILL.md")); err != nil {
		t.Error("skills/my-skill/SKILL.md should be installed")
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "workflows", "plan.md")); err != nil {
		t.Error("workflows/plan.md should be installed")
	}
	// Non-content files should NOT exist.
	if _, err := os.Stat(filepath.Join(agentRoot, "README.md")); !os.IsNotExist(err) {
		t.Error("README.md should not be copied to agent dir")
	}
	if _, err := os.Stat(filepath.Join(agentRoot, "LICENSE")); !os.IsNotExist(err) {
		t.Error("LICENSE should not be copied to agent dir")
	}
}

func TestCleanAgentDir_RemovesEmptyParentDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a nested file in agent dir.
	skillDir := filepath.Join(dir, agentDir, "skills", "old-skill")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# Old\n")

	prev := manifest{
		Version: 2,
		Targets: map[string][]string{
			"herds": {".promptherder/agent/skills/old-skill/SKILL.md"},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := cleanAgentDir(dir, prev, false, logger); err != nil {
		t.Fatal(err)
	}

	// File should be gone.
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); !os.IsNotExist(err) {
		t.Error("SKILL.md should have been removed")
	}
	// Empty parent dir (old-skill/) should also be gone.
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("empty old-skill/ dir should have been removed")
	}
}
