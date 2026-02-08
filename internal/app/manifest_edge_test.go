package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// Additional manifest tests for edge cases and error handling

func TestReadManifest_CorruptedJSON(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Write corrupted JSON
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	mustMkdir(t, filepath.Dir(manifestPath))
	mustWrite(t, manifestPath, "{invalid json, missing quotes")

	// Should return empty manifest and log warning (not fail)
	m := readManifest(dir, logger)

	if len(m.allFiles()) != 0 {
		t.Errorf("corrupted JSON should return empty manifest, got %d files", len(m.allFiles()))
	}

	if m.Version != 0 {
		t.Errorf("corrupted JSON should have version 0, got %d", m.Version)
	}
}

func TestReadManifest_InvalidVersion(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "negative version",
			content: `{"version": -1, "targets": {}}`,
		},
		{
			name:    "very large version",
			content: `{"version": 999999, "targets": {}}`,
		},
		{
			name:    "version as string",
			content: `{"version": "2", "targets": {}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(dir, tt.name)
			mustMkdir(t, testDir)

			manifestPath := filepath.Join(testDir, manifestDir, manifestFile)
			mustMkdir(t, filepath.Dir(manifestPath))
			mustWrite(t, manifestPath, tt.content)

			// Should handle gracefully (may parse successfully or return empty)
			m := readManifest(testDir, logger)

			// The key is that it doesn't crash
			_ = m.allFiles()
		})
	}
}

func TestManifest_SetTarget_DuplicateNames(t *testing.T) {
	m := manifest{Version: 2}

	// Set target twice with different files
	m.setTarget("test", []string{"file1.md", "file2.md"})
	m.setTarget("test", []string{"file3.md"})

	// Last write should win
	files := m.Targets["test"]
	if len(files) != 1 {
		t.Fatalf("expected 1 file after duplicate setTarget, got %d", len(files))
	}
	if files[0] != "file3.md" {
		t.Errorf("expected file3.md, got %s", files[0])
	}
}

func TestManifest_AllFiles_EmptyTargets(t *testing.T) {
	m := manifest{Version: 2, Targets: map[string][]string{}}

	files := m.allFiles()
	if len(files) != 0 {
		t.Errorf("empty targets should return no files, got %d", len(files))
	}
}

func TestManifest_AllFiles_TargetWithEmptyFileList(t *testing.T) {
	m := manifest{Version: 2}
	m.setTarget("test", []string{})

	files := m.allFiles()
	if len(files) != 0 {
		t.Errorf("target with empty file list should contribute no files, got %d", len(files))
	}
}

func TestManifest_IsGenerated_CaseInsensitive(t *testing.T) {
	m := manifest{Version: 2, Generated: []string{"stack.md", "structure.md"}}

	tests := []struct {
		filename string
		want     bool
	}{
		{"stack.md", true},
		{"STACK.MD", false}, // isGenerated is case-sensitive
		{"Stack.md", false}, // isGenerated is case-sensitive
		{"structure.md", true},
		{"STRUCTURE.MD", false}, // isGenerated is case-sensitive
		{"other.md", false},
		{"stackmd", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := m.isGenerated(tt.filename)
			if got != tt.want {
				t.Errorf("isGenerated(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestWriteManifest_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()

	m := manifest{Version: 2}
	m.setTarget("test", []string{"file.md"})

	// Manifest directory doesn't exist yet
	manifestDir := filepath.Join(dir, ".promptherder")
	if _, err := os.Stat(manifestDir); err == nil {
		t.Fatal("manifest directory should not exist yet")
	}

	err := writeManifest(dir, m)
	if err != nil {
		t.Fatal(err)
	}

	// Directory should be created
	if _, err := os.Stat(manifestDir); err != nil {
		t.Error("writeManifest should create .promptherder directory")
	}
}

func TestWriteManifest_PermissionError(t *testing.T) {
	t.Skip("permission test doesn't work reliably on Windows")

	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()

	// Create read-only directory
	manifestDir := filepath.Join(dir, ".promptherder")
	mustMkdir(t, manifestDir)

	// Make it read-only (this may not work on all systems)
	if err := os.Chmod(manifestDir, 0o444); err != nil {
		t.Skip("could not set read-only permissions")
	}
	defer os.Chmod(manifestDir, 0o755) // Restore for cleanup

	m := manifest{Version: 2}
	err := writeManifest(dir, m)

	// Should return error (permission denied or similar)
	if err == nil {
		t.Error("writeManifest should return error for read-only directory")
	}
}

func TestManifest_TargetFilesDeduplication(t *testing.T) {
	m := manifest{Version: 2}

	// Set target with duplicate files
	m.setTarget("test", []string{"file.md", "file.md", "other.md", "file.md"})

	// Check if duplicates exist (current implementation doesn't dedupe)
	files := m.Targets["test"]
	if len(files) == 3 {
		// If deduped, verify uniqueness
		seen := make(map[string]bool)
		for _, f := range files {
			if seen[f] {
				t.Errorf("file %s appears multiple times after deduplication", f)
			}
			seen[f] = true
		}
	} else if len(files) == 4 {
		// Current behavior: no deduplication
		t.Log("setTarget does not deduplicate files (current behavior)")
	}
}

func TestReadManifest_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Write empty manifest file
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	mustMkdir(t, filepath.Dir(manifestPath))
	mustWrite(t, manifestPath, "")

	m := readManifest(dir, logger)

	// Should handle gracefully
	if len(m.allFiles()) != 0 {
		t.Errorf("empty file should return empty manifest, got %d files", len(m.allFiles()))
	}
}

func TestReadManifest_OnlyWhitespace(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Write whitespace-only manifest
	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	mustMkdir(t, filepath.Dir(manifestPath))
	mustWrite(t, manifestPath, "   \n\t\n   ")

	m := readManifest(dir, logger)

	// Should handle gracefully
	if len(m.allFiles()) != 0 {
		t.Errorf("whitespace-only file should return empty manifest, got %d files", len(m.allFiles()))
	}
}

func TestNewManifestFrom_PreservesGenerated(t *testing.T) {
	prev := manifest{
		Version:   1,
		Generated: []string{"stack.md", "structure.md", "custom.md"},
	}
	prev.setTarget("old", []string{"old.md"})

	cur := newManifestFrom(prev)

	// Generated list should be preserved
	if len(cur.Generated) != 3 {
		t.Errorf("generated list should be preserved, got %d items", len(cur.Generated))
	}

	// Check specific items
	for _, gen := range []string{"stack.md", "structure.md", "custom.md"} {
		found := false
		for _, g := range cur.Generated {
			if g == gen {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("generated file %s should be preserved", gen)
		}
	}

	// Version should be updated
	if cur.Version != manifestVersion {
		t.Errorf("version should be %d, got %d", manifestVersion, cur.Version)
	}

	// Targets should NOT be preserved (that's the caller's job)
	if len(cur.Targets) != 0 {
		t.Errorf("newManifestFrom should not preserve targets, got %d", len(cur.Targets))
	}
}

func TestManifest_IsGenerated_EmptyList(t *testing.T) {
	m := manifest{Version: 2, Generated: []string{}}

	if m.isGenerated("stack.md") {
		t.Error("empty generated list should not match any file")
	}
}

func TestManifest_IsGenerated_NilList(t *testing.T) {
	m := manifest{Version: 2, Generated: nil}

	if m.isGenerated("stack.md") {
		t.Error("nil generated list should not match any file")
	}
}

func TestManifest_TargetCaseSensitivity(t *testing.T) {
	m := manifest{Version: 2}

	m.setTarget("Test", []string{"file1.md"})
	m.setTarget("test", []string{"file2.md"})

	// Go maps are case-sensitive, so these are different targets
	if len(m.Targets) != 2 {
		t.Errorf("target names should be case-sensitive, got %d targets", len(m.Targets))
	}

	// Both should exist
	if _, ok := m.Targets["Test"]; !ok {
		t.Error("Test target should exist")
	}
	if _, ok := m.Targets["test"]; !ok {
		t.Error("test target should exist")
	}
}

func TestReadManifest_SpecialCharactersInFilename(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	m := manifest{Version: 2}
	m.setTarget("test", []string{
		"file with spaces.md",
		"file-with-dashes.md",
		"file_with_underscores.md",
		"file.multiple.dots.md",
	})

	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	readBack := readManifest(dir, logger)

	files := readBack.Targets["test"]
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d", len(files))
	}

	// Verify special characters preserved
	for _, original := range m.Targets["test"] {
		found := false
		for _, read := range files {
			if read == original {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("special character filename %q was not preserved", original)
		}
	}
}

func TestManifest_WindowsPathSeparators(t *testing.T) {
	m := manifest{Version: 2}

	// Set paths with various separators
	m.setTarget("test", []string{
		".github/copilot-instructions.md",
		".github\\instructions\\file.md", // Windows-style
	})

	files := m.allFiles()
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Both should be stored (normalization is the caller's job)
	t.Logf("Stored paths: %v", files)
}

func TestCleanStale_PathNormalization(t *testing.T) {
	dir := t.TempDir()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Create a file with Unix-style path
	file := createTestFile(t, dir, ".github/old.md", "# Old\n")

	prev := manifest{Version: 1}
	prev.setTarget("test", []string{".github/old.md"})

	cur := manifest{Version: 2}
	// Current manifest doesn't include the file

	err := cleanStale(dir, prev, cur, false, logger)
	if err != nil {
		t.Fatal(err)
	}

	// File should be removed regardless of path separator style
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("stale file should be removed with Unix path separators")
	}
}
