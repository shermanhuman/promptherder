package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- setupRunner tests ---

func TestSetupRunner_InitializesLogger(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		Logger:   nil, // Explicitly nil to test logger initialization
	}

	_, _, tcfg, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Logger == nil {
		t.Error("setupRunner should create a logger when cfg.Logger is nil")
	}
	if tcfg.Logger == nil {
		t.Error("setupRunner should propagate logger to TargetConfig")
	}
}

func TestSetupRunner_ResolvesAbsolutePath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // already absolute

	cfg := Config{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	repoPath, _, _, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !filepath.IsAbs(repoPath) {
		t.Errorf("setupRunner should return absolute path, got: %s", repoPath)
	}
	if repoPath != dir {
		t.Errorf("repoPath = %q, want %q", repoPath, dir)
	}
}

func TestSetupRunner_LoadsManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	m := manifest{Version: 2}
	m.setTarget("test", []string{".test/file.md"})
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	_, prevManifest, _, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	if prevManifest.Version != 2 {
		t.Errorf("manifest version = %d, want 2", prevManifest.Version)
	}

	testFiles := prevManifest.Targets["test"]
	if len(testFiles) != 1 {
		t.Fatalf("expected 1 test file in manifest, got %d", len(testFiles))
	}
	if testFiles[0] != ".test/file.md" {
		t.Errorf("test file = %q, want .test/file.md", testFiles[0])
	}
}

func TestSetupRunner_EmptyManifest(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		Logger:   testLogger(t),
	}

	_, prevManifest, _, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	if len(prevManifest.allFiles()) != 0 {
		t.Errorf("expected empty manifest, got %d files", len(prevManifest.allFiles()))
	}
}

func TestSetupRunner_ErrorOnInvalidPath(t *testing.T) {
	t.Parallel()
	cfg := Config{
		RepoPath: "\x00invalid",
		Logger:   testLogger(t),
	}

	_, _, _, err := setupRunner(&cfg)
	if err == nil {
		t.Error("setupRunner should return error for invalid repo path")
	}

	if !strings.Contains(err.Error(), "resolve repo path") {
		t.Errorf("error should mention 'resolve repo path', got: %v", err)
	}
}

func TestSetupRunner_CreatesTargetConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		DryRun:   true,
		Logger:   testLogger(t),
	}

	_, _, tcfg, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !filepath.IsAbs(tcfg.RepoPath) {
		t.Error("TargetConfig.RepoPath should be absolute")
	}
	if !tcfg.DryRun {
		t.Error("TargetConfig.DryRun should be true")
	}
	if tcfg.Logger == nil {
		t.Error("TargetConfig.Logger should not be nil")
	}
}

// --- persistAndClean tests ---

func TestPersistAndClean_DryRun(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create a stale file to verify it survives dry-run
	staleDir := filepath.Join(dir, ".test")
	mustMkdir(t, staleDir)
	staleFile := filepath.Join(staleDir, "old.md")
	mustWrite(t, staleFile, "# Old\n")

	prev := manifest{Version: 1}
	prev.setTarget("test", []string{".test/old.md"})

	cur := manifest{Version: 2}
	cur.setTarget("test", []string{".test/file.md"})

	err := persistAndClean(dir, prev, cur, true, testLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	// Verify manifest was NOT written in dry-run
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Error("dry-run should not write manifest file")
	}

	// Verify stale file was NOT deleted in dry-run
	if _, err := os.Stat(staleFile); os.IsNotExist(err) {
		t.Error("dry-run should not delete stale files")
	}
}

func TestPersistAndClean_ActualWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logger := testLogger(t)

	prev := manifest{Version: 1}
	cur := manifest{Version: 2}
	cur.setTarget("test", []string{".test/file.md"})

	err := persistAndClean(dir, prev, cur, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest should be written, got error: %v", err)
	}

	readBack := readManifest(dir, logger)
	if readBack.Version != 2 {
		t.Errorf("manifest version = %d, want 2", readBack.Version)
	}
	testFiles := readBack.Targets["test"]
	if len(testFiles) != 1 {
		t.Fatalf("expected 1 test file, got %d", len(testFiles))
	}
}

func TestPersistAndClean_CleansStaleFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	staleDir := filepath.Join(dir, ".test")
	mustMkdir(t, staleDir)
	staleFile := filepath.Join(staleDir, "old.md")
	mustWrite(t, staleFile, "# Old\n")

	prev := manifest{Version: 1}
	prev.setTarget("test", []string{".test/old.md"})

	cur := manifest{Version: 2}
	cur.setTarget("test", []string{".test/new.md"})

	err := persistAndClean(dir, prev, cur, false, testLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Error("stale file should be removed")
	}
}

func TestPersistAndClean_WriteError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create manifest directory as a file to cause write error
	manifestPath := filepath.Join(dir, manifestDir)
	if err := os.WriteFile(manifestPath, []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}

	prev := manifest{Version: 1}
	cur := manifest{Version: 2}

	err := persistAndClean(dir, prev, cur, false, testLogger(t))
	if err == nil {
		t.Error("persistAndClean should return error when writeManifest fails")
	}
}
