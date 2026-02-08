package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// mustMkdir creates a directory and all parent directories.
// Fails the test if directory creation fails.
func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

// mustWrite writes content to a file, creating parent directories if needed.
// Fails the test if write fails.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// assertTarget verifies that a plan item has the expected target name.
func assertTarget(t *testing.T, item planItem, want string) {
	t.Helper()
	if item.Target != want {
		t.Errorf("target = %q, want %q", item.Target, want)
	}
}

// assertContains verifies that data contains the expected substring.
func assertContains(t *testing.T, data []byte, substr string) {
	t.Helper()
	if !bytes.Contains(data, []byte(substr)) {
		t.Errorf("expected content to contain %q, got:\n%s", substr, data)
	}
}

// assertNotContains verifies that data does NOT contain the expected substring.
func assertNotContains(t *testing.T, data []byte, substr string) {
	t.Helper()
	if bytes.Contains(data, []byte(substr)) {
		t.Errorf("expected content NOT to contain %q, got:\n%s", substr, data)
	}
}

// setupTestRepo creates a temporary directory and returns its path.
// The directory is automatically cleaned up when the test completes.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// writeTestManifest is a convenience helper for writing a manifest in tests.
func writeTestManifest(t *testing.T, repoPath string, m manifest) {
	t.Helper()
	if err := writeManifest(repoPath, m); err != nil {
		t.Fatal(err)
	}
}

// readTestFile is a convenience helper for reading a file in tests.
func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// assertFileExists verifies that a file exists at the given path.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist at %s: %v", path, err)
	}
}

// assertFileNotExists verifies that a file does NOT exist at the given path.
func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should not exist at %s", path)
	}
}

// createTestFile creates a file with the given content in the test repo.
// Returns the absolute path to the created file.
func createTestFile(t *testing.T, repoPath, relPath, content string) string {
	t.Helper()
	fullPath := filepath.Join(repoPath, relPath)
	dir := filepath.Dir(fullPath)
	mustMkdir(t, dir)
	mustWrite(t, fullPath, content)
	return fullPath
}
