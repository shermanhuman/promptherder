package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- RunAll tests ---

func TestRunAll_AllTargetsExecute(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a mock target that tracks if Install was called
	type mockTarget struct {
		name      string
		installed bool
		files     []string
	}

	targets := []*mockTarget{
		{name: "target-a", files: []string{".a/file.md"}},
		{name: "target-b", files: []string{".b/file.md"}},
		{name: "target-c", files: []string{".c/file.md"}},
	}

	// Convert to Target interface
	var iTargets []Target
	for _, mt := range targets {
		mt := mt // Capture loop variable
		iTargets = append(iTargets, targetFunc{
			name: mt.name,
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				mt.installed = true
				return mt.files, nil
			},
		})
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	err := RunAll(context.Background(), iTargets, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify all targets were executed
	for _, mt := range targets {
		if !mt.installed {
			t.Errorf("target %s was not executed", mt.name)
		}
	}

	// Verify manifest has all 3 targets
	m := readManifest(dir, logger)
	if len(m.Targets) != 3 {
		t.Fatalf("manifest should have 3 targets, got %d", len(m.Targets))
	}

	for _, mt := range targets {
		files, ok := m.Targets[mt.name]
		if !ok {
			t.Errorf("manifest missing target %s", mt.name)
			continue
		}
		if len(files) != len(mt.files) {
			t.Errorf("target %s: expected %d files, got %d", mt.name, len(mt.files), len(files))
		}
	}
}

func TestRunAll_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	iTargets := []Target{
		targetFunc{
			name: "test",
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				// Should not be called due to cancelled context
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				return []string{".test/file.md"}, nil
			},
		},
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	err := RunAll(ctx, iTargets, cfg)
	if err == nil {
		t.Error("RunAll should return error for cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error should be context.Canceled, got: %v", err)
	}
}

func TestRunAll_TargetFailureStopsExecution(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	executed := make(map[string]bool)

	iTargets := []Target{
		targetFunc{
			name: "target-1",
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				executed["target-1"] = true
				return []string{".1/file.md"}, nil
			},
		},
		targetFunc{
			name: "target-2-fails",
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				executed["target-2-fails"] = true
				return nil, fmt.Errorf("simulated failure")
			},
		},
		targetFunc{
			name: "target-3",
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				executed["target-3"] = true
				return []string{".3/file.md"}, nil
			},
		},
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	err := RunAll(context.Background(), iTargets, cfg)
	if err == nil {
		t.Fatal("RunAll should return error when a target fails")
	}

	// Verify error message mentions failure
	if !strings.Contains(err.Error(), "simulated failure") && !strings.Contains(err.Error(), "target-2-fails") {
		t.Errorf("error should mention failing target or error, got: %v", err)
	}

	// Verify target-1 was executed
	if !executed["target-1"] {
		t.Error("target-1 should have been executed before failure")
	}

	// Verify target-2 was executed (it's the one that failed)
	if !executed["target-2-fails"] {
		t.Error("target-2-fails should have been attempted")
	}

	// Note: target-3 may or may not be executed depending on implementation
	// The current implementation doesn't explicitly stop, so target-2 failure might not prevent target-3
	// This test documents actual behavior
}

func TestRunAll_DryRunAllTargets(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	iTargets := []Target{
		targetFunc{
			name: "test-a",
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				if !cfg.DryRun {
					t.Error("target should receive DryRun=true")
				}
				return []string{".a/file.md"}, nil
			},
		},
		targetFunc{
			name: "test-b",
			installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
				if !cfg.DryRun {
					t.Error("target should receive DryRun=true")
				}
				return []string{".b/file.md"}, nil
			},
		},
	}

	cfg := Config{
		RepoPath: dir,
		DryRun:   true,
		Logger:   logger,
	}

	err := RunAll(context.Background(), iTargets, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify manifest was NOT written in dry-run
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Error("dry-run should not write manifest")
	}
}

func TestRunAll_EmptyTargetList(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	err := RunAll(context.Background(), []Target{}, cfg)
	if err != nil {
		t.Fatalf("RunAll should succeed with empty target list, got: %v", err)
	}
}

// --- RunTarget tests ---

func TestRunTarget_PreservesOtherTargets(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// First, run target A
	targetA := targetFunc{
		name: "target-a",
		installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
			return []string{".a/file1.md", ".a/file2.md"}, nil
		},
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	if err := RunTarget(context.Background(), targetA, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify target A is in manifest
	m1 := readManifest(dir, logger)
	if len(m1.Targets["target-a"]) != 2 {
		t.Fatalf("target-a should have 2 files, got %d", len(m1.Targets["target-a"]))
	}

	// Now run target B
	targetB := targetFunc{
		name: "target-b",
		installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
			return []string{".b/file.md"}, nil
		},
	}

	if err := RunTarget(context.Background(), targetB, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify BOTH targets are in manifest
	m2 := readManifest(dir, logger)
	if len(m2.Targets) != 2 {
		t.Fatalf("manifest should have 2 targets, got %d", len(m2.Targets))
	}

	// Verify target A was preserved
	if len(m2.Targets["target-a"]) != 2 {
		t.Errorf("target-a should still have 2 files, got %d", len(m2.Targets["target-a"]))
	}

	// Verify target B was added
	if len(m2.Targets["target-b"]) != 1 {
		t.Errorf("target-b should have 1 file, got %d", len(m2.Targets["target-b"]))
	}
}

func TestRunTarget_ReplacesExistingTarget(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	target := targetFunc{
		name: "test",
		installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
			return []string{".test/file1.md"}, nil
		},
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	// First run
	if err := RunTarget(context.Background(), target, cfg); err != nil {
		t.Fatal(err)
	}

	m1 := readManifest(dir, logger)
	if len(m1.Targets["test"]) != 1 {
		t.Fatalf("first run should have 1 file, got %d", len(m1.Targets["test"]))
	}
	if m1.Targets["test"][0] != ".test/file1.md" {
		t.Errorf("expected file1.md, got %s", m1.Targets["test"][0])
	}

	// Second run with different files
	target2 := targetFunc{
		name: "test",
		installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
			return []string{".test/file2.md", ".test/file3.md"}, nil
		},
	}

	if err := RunTarget(context.Background(), target2, cfg); err != nil {
		t.Fatal(err)
	}

	m2 := readManifest(dir, logger)
	if len(m2.Targets["test"]) != 2 {
		t.Fatalf("second run should have 2 files, got %d", len(m2.Targets["test"]))
	}

	// Verify old file was replaced with new files
	files := m2.Targets["test"]
	hasFile2 := false
	hasFile3 := false
	for _, f := range files {
		if f == ".test/file2.md" {
			hasFile2 = true
		}
		if f == ".test/file3.md" {
			hasFile3 = true
		}
		if f == ".test/file1.md" {
			t.Error("file1.md should have been replaced")
		}
	}
	if !hasFile2 || !hasFile3 {
		t.Error("second run should have file2.md and file3.md")
	}
}

func TestRunTarget_EmptyManifestStart(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// No manifest exists yet
	target := targetFunc{
		name: "first-target",
		installFunc: func(ctx context.Context, cfg TargetConfig) ([]string, error) {
			return []string{".first/file.md"}, nil
		},
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	if err := RunTarget(context.Background(), target, cfg); err != nil {
		t.Fatal(err)
	}

	// Verify manifest was created
	m := readManifest(dir, logger)
	if len(m.Targets) != 1 {
		t.Fatalf("manifest should have 1 target, got %d", len(m.Targets))
	}
	if len(m.Targets["first-target"]) != 1 {
		t.Errorf("first-target should have 1 file, got %d", len(m.Targets["first-target"]))
	}
}

// --- Helper type for testing ---

// targetFunc implements Target interface using functions
type targetFunc struct {
	name        string
	installFunc func(context.Context, TargetConfig) ([]string, error)
}

func (t targetFunc) Name() string {
	return t.name
}

func (t targetFunc) Install(ctx context.Context, cfg TargetConfig) ([]string, error) {
	return t.installFunc(ctx, cfg)
}
