package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestHerdNameFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https with .git", "https://github.com/shermanhuman/compound-v.git", "compound-v"},
		{"https without .git", "https://github.com/shermanhuman/compound-v", "compound-v"},
		{"trailing slash", "https://github.com/shermanhuman/compound-v/", "compound-v"},
		{"trailing slash and .git", "https://github.com/shermanhuman/compound-v.git/", "compound-v"},
		{"ssh url", "git@github.com:shermanhuman/compound-v.git", "compound-v"},
		{"simple name", "compound-v", "compound-v"},
		{"local path", "/home/user/herds/my-herd", "my-herd"},
		{"windows path", "C:\\Users\\s\\github\\compound-v", "compound-v"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := herdNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("herdNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestOwnerRepoFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
	}{
		{"https with .git", "https://github.com/shermanhuman/compound-v.git", "shermanhuman", "compound-v"},
		{"https without .git", "https://github.com/shermanhuman/compound-v", "shermanhuman", "compound-v"},
		{"trailing slash", "https://github.com/shermanhuman/compound-v/", "shermanhuman", "compound-v"},
		{"ssh url", "git@github.com:shermanhuman/compound-v.git", "shermanhuman", "compound-v"},
		{"non-github", "https://gitlab.com/user/repo", "", ""},
		{"no owner", "https://github.com/compound-v", "", ""},
		{"empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotOwner, gotRepo := ownerRepoFromURL(tt.url)
			if gotOwner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", gotRepo, tt.wantRepo)
			}
		})
	}
}

func TestToArchiveURL(t *testing.T) {
	t.Parallel()

	got := toArchiveURL("shermanhuman", "compound-v")
	want := "https://api.github.com/repos/shermanhuman/compound-v/tarball"
	if got != want {
		t.Errorf("toArchiveURL() = %q, want %q", got, want)
	}
}

func TestExtractTarGz(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Build an in-memory tar.gz with a top-level prefix (like GitHub).
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	files := []struct {
		name    string
		content string
		isDir   bool
	}{
		{"compound-v-abc123/", "", true},
		{"compound-v-abc123/herd.json", `{"name":"compound-v"}`, false},
		{"compound-v-abc123/rules/", "", true},
		{"compound-v-abc123/rules/foo.md", "# Foo\n", false},
		{"compound-v-abc123/skills/my-skill/", "", true},
		{"compound-v-abc123/skills/my-skill/SKILL.md", "# Skill\n", false},
	}

	for _, f := range files {
		if f.isDir {
			_ = tw.WriteHeader(&tar.Header{
				Name:     f.name,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			})
		} else {
			_ = tw.WriteHeader(&tar.Header{
				Name:     f.name,
				Typeflag: tar.TypeReg,
				Mode:     0644,
				Size:     int64(len(f.content)),
			})
			_, _ = tw.Write([]byte(f.content))
		}
	}
	tw.Close()
	gw.Close()

	destDir := filepath.Join(dir, "output")
	if err := extractTarGz(&buf, destDir); err != nil {
		t.Fatal(err)
	}

	// Verify files exist with prefix stripped.
	wantFiles := map[string]string{
		"herd.json":                `{"name":"compound-v"}`,
		"rules/foo.md":             "# Foo\n",
		"skills/my-skill/SKILL.md": "# Skill\n",
	}
	for relPath, wantContent := range wantFiles {
		data, err := os.ReadFile(filepath.Join(destDir, filepath.FromSlash(relPath)))
		if err != nil {
			t.Errorf("file %s not found: %v", relPath, err)
			continue
		}
		if string(data) != wantContent {
			t.Errorf("file %s content = %q, want %q", relPath, string(data), wantContent)
		}
	}
}

func TestExtractTarGz_PathTraversal(t *testing.T) {
	t.Parallel()

	// Build tar.gz with a path traversal attempt.
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	_ = tw.WriteHeader(&tar.Header{
		Name:     "prefix/../../etc/passwd",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     4,
	})
	_, _ = tw.Write([]byte("evil"))
	tw.Close()
	gw.Close()

	dir := t.TempDir()
	err := extractTarGz(&buf, dir)
	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("escape")) {
		t.Errorf("error should mention escape, got: %v", err)
	}
}
