package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shermanhuman/promptherder/internal/compiler"
)

type archiveTransport func(*http.Request) (*http.Response, error)

func (f archiveTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPullStagesAndVerifiesLockedArchive(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, ".promptherder/herds/example")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "herd.json"), []byte(`{"name":"original"}`), 0644); err != nil {
		t.Fatal(err)
	}
	old := httpClient
	t.Cleanup(func() { httpClient = old })
	body := []byte("invalid archive")
	httpClient = &http.Client{Transport: archiveTransport(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "/commits/HEAD") {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"sha":"` + strings.Repeat("a", 40) + `"}`)), Request: r}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
	})}
	cfg := PullConfig{RepoPath: root, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := Pull(context.Background(), "https://github.com/owner/example", cfg); err == nil {
		t.Fatal("invalid archive accepted")
	}
	data, _ := os.ReadFile(filepath.Join(dest, "herd.json"))
	if string(data) != `{"name":"original"}` {
		t.Fatal("failed download changed installed herd")
	}
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	tw := tar.NewWriter(gz)
	for _, file := range []struct {
		name, data string
		mode       int64
	}{
		{"herd.json", `{"name":"example","version":"1"}`, 0644},
		{"skills/example/SKILL.md", "---\nname: example\ndescription: Example\n---\nBody\n", 0644},
		{"skills/example/scripts/run.sh", "#!/bin/sh\nexit 0\n", 0755},
	} {
		if err := tw.WriteHeader(&tar.Header{Name: "root/" + file.name, Mode: file.mode, Size: int64(len(file.data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(file.data))
	}
	_ = tw.Close()
	_ = gz.Close()
	body = b.Bytes()
	if err := Pull(context.Background(), "https://github.com/owner/example", cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dest, "skills/example/scripts/run.sh"))
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("executable mode lost", err)
	}
	sum, err := compiler.HerdDigest(dest)
	if err != nil {
		t.Fatal(err)
	}
	revision := strings.Repeat("a", 40)
	locks := map[string]compiler.HerdLock{"example": {Repository: "https://github.com/owner/example", Revision: revision, Digest: sum}}
	data, _ = json.Marshal(locks)
	if err := os.WriteFile(filepath.Join(root, ".promptherder/lock.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	cfg.Locked = true
	httpClient.Transport = archiveTransport(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/"+revision) {
			t.Error("did not request immutable revision", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
	})
	if err := Pull(context.Background(), "https://github.com/owner/example", cfg); err != nil {
		t.Fatal(err)
	}
	locks["example"] = compiler.HerdLock{Repository: "https://github.com/owner/example", Revision: revision, Digest: "incorrect"}
	data, _ = json.Marshal(locks)
	_ = os.WriteFile(filepath.Join(root, ".promptherder/lock.json"), data, 0644)
	if err := Pull(context.Background(), "https://github.com/owner/example", cfg); err == nil {
		t.Fatal("wrong locked digest accepted")
	}
	after, _ := compiler.HerdDigest(dest)
	if after != sum {
		t.Fatal("failed verification changed installed herd")
	}
}
