package app

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PullConfig holds the configuration for a pull operation.
type PullConfig struct {
	RepoPath string       // absolute path to the repo root
	DryRun   bool         // if true, log what would happen but don't download
	Logger   *slog.Logger // structured logger
}

// Pull downloads a herd from a GitHub repository archive.
// The herd name is derived from the URL's last path segment (sans .git).
// After extraction, it validates that herd.json exists.
// No git binary required — uses net/http + archive/tar + compress/gzip.
func Pull(ctx context.Context, gitURL string, cfg PullConfig) error {
	name := herdNameFromURL(gitURL)
	if name == "" {
		return fmt.Errorf("cannot derive herd name from URL: %s", gitURL)
	}

	owner, repo := ownerRepoFromURL(gitURL)
	if owner == "" || repo == "" {
		return fmt.Errorf("cannot parse owner/repo from URL: %s (expected https://github.com/OWNER/REPO)", gitURL)
	}

	archiveURL := toArchiveURL(owner, repo)
	herdPath := filepath.Join(cfg.RepoPath, herdsDir, name)

	if cfg.DryRun {
		if isDirectory(herdPath) {
			cfg.Logger.Info("dry-run: would update herd", "name", name, "url", archiveURL)
		} else {
			cfg.Logger.Info("dry-run: would download herd", "name", name, "url", archiveURL, "path", herdPath)
		}
		return nil
	}

	// Remove existing herd dir for a clean re-download.
	if isDirectory(herdPath) {
		cfg.Logger.Info("updating herd (re-downloading)", "name", name)
		if err := os.RemoveAll(herdPath); err != nil {
			return fmt.Errorf("remove existing herd %s: %w", name, err)
		}
	}

	cfg.Logger.Info("downloading herd", "name", name, "url", archiveURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", archiveURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", archiveURL, resp.StatusCode)
	}

	if err := extractTarGz(resp.Body, herdPath); err != nil {
		return fmt.Errorf("extract herd %s: %w", name, err)
	}

	// Validate herd.json exists.
	metaPath := filepath.Join(herdPath, herdMetaFile)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("herd %q has no %s — is this a valid herd repository?", name, herdMetaFile)
	}

	cfg.Logger.Info("herd ready", "name", name, "path", herdPath)
	return nil
}

// herdNameFromURL extracts the herd name from a git URL.
// e.g. "https://github.com/shermanhuman/compound-v.git" → "compound-v"
// e.g. "https://github.com/shermanhuman/compound-v" → "compound-v"
func herdNameFromURL(gitURL string) string {
	// Strip trailing slashes and .git suffix.
	u := strings.TrimRight(gitURL, "/\\")
	u = strings.TrimSuffix(u, ".git")

	// Take the last path segment (handle both / and \ separators).
	if idx := strings.LastIndexAny(u, "/\\"); idx >= 0 {
		return u[idx+1:]
	}
	return u
}

// ownerRepoFromURL extracts the owner and repo from a GitHub URL.
// Supports: https://github.com/OWNER/REPO[.git]
func ownerRepoFromURL(gitURL string) (owner, repo string) {
	u := strings.TrimRight(gitURL, "/\\")
	u = strings.TrimSuffix(u, ".git")

	// Handle HTTPS URLs.
	if idx := strings.Index(u, "github.com/"); idx >= 0 {
		path := u[idx+len("github.com/"):]
		parts := strings.SplitN(path, "/", 3)
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1]
		}
	}

	// Handle SSH URLs: git@github.com:OWNER/REPO
	if strings.HasPrefix(u, "git@github.com:") {
		path := u[len("git@github.com:"):]
		parts := strings.SplitN(path, "/", 3)
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1]
		}
	}

	return "", ""
}

// toArchiveURL builds the GitHub API archive URL for a repo's default branch.
// Returns: https://api.github.com/repos/OWNER/REPO/tarball
func toArchiveURL(owner, repo string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/tarball", owner, repo)
}

// extractTarGz extracts a tar.gz stream into destDir, stripping the
// top-level directory prefix (GitHub archives have a "repo-branch/" prefix).
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		// Strip the top-level directory prefix.
		// GitHub tarballs have entries like "owner-repo-sha/" at the top.
		name := hdr.Name
		if idx := strings.IndexByte(name, '/'); idx >= 0 {
			name = name[idx+1:]
		}
		if name == "" {
			continue // skip the root dir entry itself
		}

		target := filepath.Join(destDir, filepath.FromSlash(name))

		// Path traversal protection.
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("tar entry %q tries to escape destination", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", target, err)
			}
			f, err := os.Create(target)
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("write %s: %w", target, err)
			}
			f.Close()
		}
	}

	return nil
}
