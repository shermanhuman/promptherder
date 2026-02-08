package app

import (
	"os"
	"path/filepath"
	"testing"
)

// Manifest edge case and error handling tests.

func TestReadManifest_CorruptedJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	manifestPath := filepath.Join(dir, manifestDir, manifestFile)
	mustMkdir(t, filepath.Dir(manifestPath))
	mustWrite(t, manifestPath, "{invalid json, missing quotes")

	m := readManifest(dir, testLogger(t))

	if len(m.allFiles()) != 0 {
		t.Errorf("corrupted JSON should return empty manifest, got %d files", len(m.allFiles()))
	}
	if m.Version != 0 {
		t.Errorf("corrupted JSON should have version 0, got %d", m.Version)
	}
}

func TestReadManifest_InvalidVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logger := testLogger(t)

	tests := []struct {
		name    string
		content string
	}{
		{"negative version", `{"version": -1, "targets": {}}`},
		{"very large version", `{"version": 999999, "targets": {}}`},
		{"version as string", `{"version": "2", "targets": {}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testDir := filepath.Join(dir, tt.name)
			mustMkdir(t, testDir)

			mp := filepath.Join(testDir, manifestDir, manifestFile)
			mustMkdir(t, filepath.Dir(mp))
			mustWrite(t, mp, tt.content)

			m := readManifest(testDir, logger)
			_ = m.allFiles() // must not crash
		})
	}
}

func TestSetTarget_DuplicateNames(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2}

	m.setTarget("test", []string{"file1.md", "file2.md"})
	m.setTarget("test", []string{"file3.md"})

	files := m.Targets["test"]
	if len(files) != 1 {
		t.Fatalf("expected 1 file after duplicate setTarget, got %d", len(files))
	}
	if files[0] != "file3.md" {
		t.Errorf("expected file3.md, got %s", files[0])
	}
}

func TestAllFiles_EmptyTargets(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2, Targets: map[string][]string{}}

	if len(m.allFiles()) != 0 {
		t.Errorf("empty targets should return no files, got %d", len(m.allFiles()))
	}
}

func TestAllFiles_TargetWithEmptyFileList(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2}
	m.setTarget("test", []string{})

	if len(m.allFiles()) != 0 {
		t.Errorf("target with empty file list should contribute no files, got %d", len(m.allFiles()))
	}
}

func TestIsGenerated_CaseSensitive(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2, Generated: []string{"stack.md", "structure.md"}}

	tests := []struct {
		filename string
		want     bool
	}{
		{"stack.md", true},
		{"STACK.MD", false},
		{"Stack.md", false},
		{"structure.md", true},
		{"STRUCTURE.MD", false},
		{"other.md", false},
		{"stackmd", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()
			if got := m.isGenerated(tt.filename); got != tt.want {
				t.Errorf("isGenerated(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestWriteManifest_CreatesDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	m := manifest{Version: 2}
	m.setTarget("test", []string{"file.md"})

	md := filepath.Join(dir, ".promptherder")
	if _, err := os.Stat(md); err == nil {
		t.Fatal("manifest directory should not exist yet")
	}

	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(md); err != nil {
		t.Error("writeManifest should create .promptherder directory")
	}
}

func TestSetTarget_AcceptsDuplicateFiles(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2}
	m.setTarget("test", []string{"file.md", "file.md", "other.md", "file.md"})

	// setTarget sorts but does not deduplicate; it stores what it's given.
	files := m.Targets["test"]
	if len(files) != 4 {
		t.Errorf("expected 4 entries (no dedup), got %d", len(files))
	}
}

func TestReadManifest_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	mp := filepath.Join(dir, manifestDir, manifestFile)
	mustMkdir(t, filepath.Dir(mp))
	mustWrite(t, mp, "")

	m := readManifest(dir, testLogger(t))
	if len(m.allFiles()) != 0 {
		t.Errorf("empty file should return empty manifest, got %d files", len(m.allFiles()))
	}
}

func TestReadManifest_OnlyWhitespace(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	mp := filepath.Join(dir, manifestDir, manifestFile)
	mustMkdir(t, filepath.Dir(mp))
	mustWrite(t, mp, "   \n\t\n   ")

	m := readManifest(dir, testLogger(t))
	if len(m.allFiles()) != 0 {
		t.Errorf("whitespace-only file should return empty manifest, got %d files", len(m.allFiles()))
	}
}

func TestNewManifestFrom_PreservesGenerated(t *testing.T) {
	t.Parallel()
	prev := manifest{
		Version:   1,
		Generated: []string{"stack.md", "structure.md", "custom.md"},
	}
	prev.setTarget("old", []string{"old.md"})

	cur := newManifestFrom(prev)

	if len(cur.Generated) != 3 {
		t.Errorf("generated list should be preserved, got %d items", len(cur.Generated))
	}

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

	if cur.Version != manifestVersion {
		t.Errorf("version should be %d, got %d", manifestVersion, cur.Version)
	}
	if len(cur.Targets) != 0 {
		t.Errorf("newManifestFrom should not preserve targets, got %d", len(cur.Targets))
	}
}

func TestIsGenerated_EmptyList(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2, Generated: []string{}}
	if m.isGenerated("stack.md") {
		t.Error("empty generated list should not match any file")
	}
}

func TestIsGenerated_NilList(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2, Generated: nil}
	if m.isGenerated("stack.md") {
		t.Error("nil generated list should not match any file")
	}
}

func TestSetTarget_CaseSensitiveNames(t *testing.T) {
	t.Parallel()
	m := manifest{Version: 2}
	m.setTarget("Test", []string{"file1.md"})
	m.setTarget("test", []string{"file2.md"})

	if len(m.Targets) != 2 {
		t.Errorf("target names should be case-sensitive, got %d targets", len(m.Targets))
	}
	if _, ok := m.Targets["Test"]; !ok {
		t.Error("Test target should exist")
	}
	if _, ok := m.Targets["test"]; !ok {
		t.Error("test target should exist")
	}
}

func TestReadManifest_SpecialCharactersInFilename(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logger := testLogger(t)

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

func TestWriteReadManifest_PathSeparatorRoundTrip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logger := testLogger(t)

	m := manifest{Version: 2}
	m.setTarget("test", []string{
		".github/copilot-instructions.md",
		".agent/rules/00-test.md",
	})
	if err := writeManifest(dir, m); err != nil {
		t.Fatal(err)
	}

	readBack := readManifest(dir, logger)
	files := readBack.Targets["test"]
	if len(files) != 2 {
		t.Fatalf("expected 2 files after round-trip, got %d", len(files))
	}

	// Verify forward-slash paths survive serialization round-trip
	for _, want := range m.Targets["test"] {
		found := false
		for _, got := range files {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("path %q not preserved after round-trip", want)
		}
	}
}

func TestCleanStale_PathNormalization(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	file := createTestFile(t, dir, ".github/old.md", "# Old\n")

	prev := manifest{Version: 1}
	prev.setTarget("test", []string{".github/old.md"})

	cur := manifest{Version: 2}

	err := cleanStale(dir, prev, cur, false, testLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("stale file should be removed with Unix path separators")
	}
}
