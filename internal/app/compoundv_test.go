package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCompoundVTarget_NilFS(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	target := CompoundVTarget{FS: nil}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	_, err := target.Install(context.Background(), cfg)
	if err == nil {
		t.Error("nil FS should return error")
	}

	if !strings.Contains(err.Error(), "embedded FS is nil") {
		t.Errorf("error should mention nil FS, got: %v", err)
	}
}

func TestCompoundVTarget_InstallsFromEmbeddedFS(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create mock embedded FS using fstest.MapFS
	mockFS := fstest.MapFS{
		"compound-v/rules/00-test.md": {
			Data: []byte("# Test Rule\\n"),
		},
		"compound-v/rules/01-shell.md": {
			Data: []byte("# Shell Rule\\n"),
		},
		"compound-v/skills/test-skill/SKILL.md": {
			Data: []byte("# Test Skill\\n"),
		},
		"compound-v/workflows/test.md": {
			Data: []byte("# Test Workflow\\n"),
		},
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify 4 files were installed
	if len(installed) != 4 {
		t.Fatalf("expected 4 files installed, got %d", len(installed))
	}

	// Verify files exist in .promptherder/agent/
	targetBase := filepath.Join(dir, ".promptherder", "agent")

	if _, err := os.Stat(filepath.Join(targetBase, "rules", "00-test.md")); err != nil {
		t.Error("rules/00-test.md should be installed")
	}
	if _, err := os.Stat(filepath.Join(targetBase, "rules", "01-shell.md")); err != nil {
		t.Error("rules/01-shell.md should be installed")
	}
	if _, err := os.Stat(filepath.Join(targetBase, "skills", "test-skill", "SKILL.md")); err != nil {
		t.Error("skills/test-skill/SKILL.md should be installed")
	}
	if _, err := os.Stat(filepath.Join(targetBase, "workflows", "test.md")); err != nil {
		t.Error("workflows/test.md should be installed")
	}

	// Verify content
	data, err := os.ReadFile(filepath.Join(targetBase, "rules", "00-test.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Test Rule") {
		t.Error("file content should match embedded FS")
	}
}

func TestCompoundVTarget_SkipsGeneratedFiles(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create mock FS with generated file
	mockFS := fstest.MapFS{
		"compound-v/rules/stack.md": {
			Data: []byte("# Embedded Stack\\n"),
		},
		"compound-v/rules/normal.md": {
			Data: []byte("# Normal\\n"),
		},
	}

	// Create existing generated file
	targetBase := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, targetBase)
	mustWrite(t, filepath.Join(targetBase, "stack.md"), "# Existing Stack\\n")

	// Mark stack.md as generated
	m := manifest{Version: 2, Generated: []string{"stack.md"}}
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify only normal.md was installed
	if len(installed) != 1 {
		t.Fatalf("expected 1 file (skipping stack.md), got %d", len(installed))
	}

	// Verify stack.md was not overwritten
	data, err := os.ReadFile(filepath.Join(targetBase, "stack.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Existing Stack") {
		t.Error("generated file should not be overwritten")
	}

	// Verify normal.md was installed
	if _, err := os.Stat(filepath.Join(targetBase, "normal.md")); err != nil {
		t.Error("normal.md should be installed")
	}
}

func TestCompoundVTarget_DryRun(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {
			Data: []byte("# Test\\n"),
		},
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		DryRun:   true,
		Logger:   logger,
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify files list returned
	if len(installed) != 1 {
		t.Fatalf("dry-run should return file list, got %d", len(installed))
	}

	// Verify no files actually written
	targetFile := filepath.Join(dir, ".promptherder", "agent", "rules", "test.md")
	if _, err := os.Stat(targetFile); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
}

func TestCompoundVTarget_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {
			Data: []byte("# Test\\n"),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	_, err := target.Install(ctx, cfg)
	if err == nil {
		t.Error("cancelled context should return error")
	}

	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("error should mention context cancellation, got: %v", err)
	}
}

func TestCompoundVTarget_InstallationMessage(t *testing.T) {
	dir := t.TempDir()

	// Capture log output
	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {
			Data: []byte("# Test\\n"),
		},
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		DryRun:   false,
		Logger:   logger,
	}

	_, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify "fan out" message logged
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "fan out to agent targets") {
		t.Error("installation should log 'fan out to agent targets' message")
	}
}

func TestCompoundVTarget_NoMessageOnDryRun(t *testing.T) {
	dir := t.TempDir()

	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	mockFS := fstest.MapFS{
		"compound-v/rules/test.md": {
			Data: []byte("# Test\\n"),
		},
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		DryRun:   true,
		Logger:   logger,
	}

	_, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify NO "fan out" message on dry-run
	logOutput := logBuf.String()
	if strings.Contains(logOutput, "fan out") {
		t.Error("dry-run should not log 'fan out' message")
	}
}

func TestCompoundVTarget_PreservesDirectoryStructure(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Complex nested structure
	mockFS := fstest.MapFS{
		"compound-v/rules/subdir/nested.md": {
			Data: []byte("# Nested\\n"),
		},
		"compound-v/skills/skill-a/subdir/deep.md": {
			Data: []byte("# Deep\\n"),
		},
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	installed, err := target.Install(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(installed) != 2 {
		t.Fatalf("expected 2 files, got %d", len(installed))
	}

	// Verify structure preserved
	targetBase := filepath.Join(dir, ".promptherder", "agent")
	if _, err := os.Stat(filepath.Join(targetBase, "rules", "subdir", "nested.md")); err != nil {
		t.Error("nested directory structure should be preserved")
	}
	if _, err := os.Stat(filepath.Join(targetBase, "skills", "skill-a", "subdir", "deep.md")); err != nil {
		t.Error("deep nested structure should be preserved")
	}
}

func TestCompoundVTarget_Name(t *testing.T) {
	target := CompoundVTarget{}
	if target.Name() != "compound-v" {
		t.Errorf("Name() = %q, want %q", target.Name(), "compound-v")
	}
}

func TestCompoundVTarget_EmptyFS(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Empty MapFS (no compound-v directory)
	mockFS := fstest.MapFS{}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	_, err := target.Install(context.Background(), cfg)
	if err == nil {
		t.Error("empty FS should return error when trying to walk compound-v")
	}
}

func TestCompoundVTarget_ReadError(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create FS that will fail on read
	mockFS := &failingFS{
		files: map[string]bool{
			"compound-v/rules/test.md": true,
		},
	}

	target := CompoundVTarget{FS: mockFS}
	cfg := TargetConfig{
		RepoPath: dir,
		Logger:   logger,
	}

	_, err := target.Install(context.Background(), cfg)
	if err == nil {
		t.Error("FS read error should be propagated")
	}
}

// Failing FS for error testing
type failingFS struct {
	files map[string]bool
}

func (f *failingFS) Open(name string) (fs.File, error) {
	if f.files[name] {
		return nil, fmt.Errorf("simulated read failure")
	}
	return nil, fs.ErrNotExist
}
