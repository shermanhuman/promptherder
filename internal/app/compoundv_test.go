package app

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCompoundVTarget_NilFS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := CompoundVTarget{FS: nil}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	_, err := target.Install(context.Background(), cfg)
	if err == nil {
		t.Error("nil FS should return error")
	}
	if !strings.Contains(err.Error(), "embedded FS is nil") {
		t.Errorf("error should mention nil FS, got: %v", err)
	}
}

func TestCompoundVTarget_InstallsFromEmbeddedFS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFS := fstest.MapFS{
		"compound-v/rules/00-test.md":           {Data: []byte("# Test Rule\n")},
		"compound-v/rules/01-shell.md":          {Data: []byte("# Shell Rule\n")},
		"compound-v/skills/test-skill/SKILL.md": {Data: []byte("# Test Skill\n")},
		"compound-v/workflows/test.md":          {Data: []byte("# Test Workflow\n")},
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 4 {
		t.Fatalf("expected 4 files installed, got %d", len(installed))
	}

	base := filepath.Join(dir, ".promptherder", "agent")
	for _, rel := range []string{
		"rules/00-test.md", "rules/01-shell.md",
		"skills/test-skill/SKILL.md", "workflows/test.md",
	} {
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s should be installed", rel)
		}
	}

	data, err := os.ReadFile(filepath.Join(base, "rules", "00-test.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Test Rule") {
		t.Error("file content should match embedded FS")
	}
}

func TestCompoundVTarget_SkipsGeneratedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFS := fstest.MapFS{
		"compound-v/rules/stack.md":  {Data: []byte("# Embedded Stack\n")},
		"compound-v/rules/normal.md": {Data: []byte("# Normal\n")},
	}
	base := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, base)
	mustWrite(t, filepath.Join(base, "stack.md"), "# Existing Stack\n")

	m := manifest{Version: 2, Generated: []string{"stack.md"}}
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("expected 1 file (skipping stack.md), got %d", len(installed))
	}
	data, err := os.ReadFile(filepath.Join(base, "stack.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Existing Stack") {
		t.Error("generated file should not be overwritten")
	}
	if _, err := os.Stat(filepath.Join(base, "normal.md")); err != nil {
		t.Error("normal.md should be installed")
	}
}

func TestCompoundVTarget_DryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {Data: []byte("# Test\n")},
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, DryRun: true, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 1 {
		t.Fatalf("dry-run should return file list, got %d", len(installed))
	}
	f := filepath.Join(dir, ".promptherder", "agent", "rules", "test.md")
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
}

func TestCompoundVTarget_ContextCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {Data: []byte("# Test\n")},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	_, err := target.Install(ctx, cfg)
	if err == nil {
		t.Error("cancelled context should return error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestCompoundVTarget_InstallationMessage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {Data: []byte("# Test\n")},
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, Logger: logger}

	if _, err := target.Install(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logBuf.String(), "fan out to agent targets") {
		t.Error("installation should log 'fan out to agent targets' message")
	}
}

func TestCompoundVTarget_NoMessageOnDryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {Data: []byte("# Test\n")},
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, DryRun: true, Logger: logger}

	if _, err := target.Install(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logBuf.String(), "fan out") {
		t.Error("dry-run should not log 'fan out' message")
	}
}

func TestCompoundVTarget_PreservesDirectoryStructure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFS := fstest.MapFS{
		"compound-v/rules/subdir/nested.md":        {Data: []byte("# Nested\n")},
		"compound-v/skills/skill-a/subdir/deep.md": {Data: []byte("# Deep\n")},
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(installed) != 2 {
		t.Fatalf("expected 2 files, got %d", len(installed))
	}
	base := filepath.Join(dir, ".promptherder", "agent")
	if _, err := os.Stat(filepath.Join(base, "rules", "subdir", "nested.md")); err != nil {
		t.Error("nested directory structure should be preserved")
	}
	if _, err := os.Stat(filepath.Join(base, "skills", "skill-a", "subdir", "deep.md")); err != nil {
		t.Error("deep nested structure should be preserved")
	}
}

func TestCompoundVTarget_Name(t *testing.T) {
	t.Parallel()
	target := CompoundVTarget{}
	if target.Name() != "compound-v" {
		t.Errorf("Name() = %q, want %q", target.Name(), "compound-v")
	}
}

func TestCompoundVTarget_EmptyFS(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := CompoundVTarget{FS: fstest.MapFS{}}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	_, err := target.Install(context.Background(), cfg)
	if err == nil {
		t.Error("empty FS should return error when trying to walk compound-v")
	}
}

func TestCompoundVTarget_WalkRootError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mockFS := fstest.MapFS{
		"wrong-prefix/rules/test.md": {Data: []byte("# Test\n")},
	}
	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{RepoPath: dir, Logger: testLogger(t)}

	_, err := target.Install(context.Background(), cfg)
	if err == nil {
		t.Error("walk with missing root should return error")
	}
}
