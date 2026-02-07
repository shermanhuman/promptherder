package files

import (
	"fmt"
	"os"
	"path/filepath"
)

type AtomicWriter struct {
	Path string
	Perm os.FileMode
}

func (w AtomicWriter) Write(data []byte) error {
	if w.Path == "" {
		return fmt.Errorf("path is required")
	}
	perm := w.Perm
	if perm == 0 {
		perm = 0o644
	}

	dir := filepath.Dir(w.Path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), w.Path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
