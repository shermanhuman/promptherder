package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriter_Write(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.md")

	w := AtomicWriter{Path: target, Perm: 0o644}
	content := []byte("hello world\n")

	if err := w.Write(content); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestAtomicWriter_CreatesDirs(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "sub", "deep", "output.md")

	w := AtomicWriter{Path: target, Perm: 0o644}
	if err := w.Write([]byte("nested")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("nested file not created: %v", err)
	}
}

func TestAtomicWriter_Overwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.md")

	// Write initial content
	w := AtomicWriter{Path: target, Perm: 0o644}
	if err := w.Write([]byte("v1")); err != nil {
		t.Fatal(err)
	}

	// Overwrite
	if err := w.Write([]byte("v2")); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Errorf("content = %q, want %q", got, "v2")
	}
}

func TestAtomicWriter_NoTempLeftBehind(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "output.md")

	w := AtomicWriter{Path: target, Perm: 0o644}
	if err := w.Write([]byte("clean")); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected 1 file, got %d: %v", len(entries), names)
	}
}

func TestAtomicWriter_EmptyPath(t *testing.T) {
	w := AtomicWriter{Path: "", Perm: 0o644}
	if err := w.Write([]byte("fail")); err == nil {
		t.Error("expected error for empty path")
	}
}
