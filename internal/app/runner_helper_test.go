package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- setupRunner tests ---

func TestSetupRunner_InitializesLogger(t *testing.T) {
	dir := t.TempDir()

	cfg := Config{
		RepoPath: dir,
		Logger:   nil, // Explicitly nil to test logger initialization
	}

	_, _, tcfg, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify logger was created
	if cfg.Logger == nil {
		t.Error("setupRunner should create a logger when cfg.Logger is nil")
	}

	// Verify logger was propagated to TargetConfig
	if tcfg.Logger == nil {
		t.Error("setupRunner should propagate logger to TargetConfig")
	}
}

func TestSetupRunner_ResolvesAbsolutePath(t *testing.T) {
	dir := t.TempDir()

	// Use relative path
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd) //nolint

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RepoPath: ".",
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	repoPath, _, _, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify path is absolute
	if !filepath.IsAbs(repoPath) {
		t.Errorf("setupRunner should return absolute path, got: %s", repoPath)
	}

	// Verify path matches temp dir
	absDir, _ := filepath.Abs(dir)
	if repoPath != absDir {
		t.Errorf("repoPath = %q, want %q", repoPath, absDir)
	}
}

func TestSetupRunner_LoadsManifest(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Write a manifest
	m := manifest{Version: 2}
	m.setTarget("test", []string{".test/file.md"})
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	_, prevManifest, _, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify manifest was loaded
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
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := Config{
		RepoPath: dir,
		Logger:   logger,
	}

	_, prevManifest, _, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify empty manifest returned when no manifest file exists
	if len(prevManifest.allFiles()) != 0 {
		t.Errorf("expected empty manifest, got %d files", len(prevManifest.allFiles()))
	}
}

func TestSetupRunner_ErrorOnInvalidPath(t *testing.T) {
	// Use a path that will fail filepath.Abs
	// On most systems, empty string is invalid
	cfg := Config{
		RepoPath: "\x00invalid",
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
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
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg := Config{
		RepoPath: dir,
		DryRun:   true,
		Logger:   logger,
	}

	_, _, tcfg, err := setupRunner(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Verify TargetConfig fields
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
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	prev := manifest{Version: 1}
	cur := manifest{Version: 2}
	cur.setTarget("test", []string{".test/file.md"})

	err := persistAndClean(dir, prev, cur, true, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Verify manifest was NOT written in dry-run
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Error("dry-run should not write manifest file")
	}
}

func TestPersistAndClean_ActualWrite(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	prev := manifest{Version: 1}
	cur := manifest{Version: 2}
	cur.setTarget("test", []string{".test/file.md"})

	err := persistAndClean(dir, prev, cur, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Verify manifest was written
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Errorf("manifest should be written, got error: %v", err)
	}

	// Verify manifest content
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
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a stale file
	staleDir := filepath.Join(dir, ".test")
	mustMkdir(t, staleDir)
	staleFile := filepath.Join(staleDir, "old.md")
	mustWrite(t, staleFile, "# Old\n")

	// Previous manifest includes the stale file
	prev := manifest{Version: 1}
	prev.setTarget("test", []string{".test/old.md"})

	// Current manifest does not include it
	cur := manifest{Version: 2}
	cur.setTarget("test", []string{".test/new.md"})

	err := persistAndClean(dir, prev, cur, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	// Verify stale file was removed
	if _, err := os.Stat(staleFile); !os.IsNotExist(err) {
		t.Error("stale file should be removed")
	}
}

func TestPersistAndClean_WriteError(t *testing.T) {
	// Use a directory as the repo path to cause writeManifest to fail
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create manifest directory as a file to cause write error
	manifestPath := filepath.Join(dir, manifestDir)
	if err := os.WriteFile(manifestPath, []byte("block"), 0o644); err != nil {
		t.Fatal(err)
	}

	prev := manifest{Version: 1}
	cur := manifest{Version: 2}

	err := persistAndClean(dir, prev, cur, false, logger)
	if err == nil {
		t.Error("persistAndClean should return error when writeManifest fails")
	}
}
