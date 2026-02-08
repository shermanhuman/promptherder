package app

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// testLogger returns a quiet logger (discards output) for tests that don't
// inspect log content. Tests that need to capture logs should create their
// own logger with a strings.Builder.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

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
