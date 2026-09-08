package app

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shermanhuman/promptherder/internal/compiler"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// maxExtractFileSize is the per-file size limit during herd archive extraction.
const maxExtractFileSize = 10 << 20 // 10 MB

// httpClient is used for all outbound HTTP requests. Unlike http.DefaultClient,
// it has a timeout to prevent indefinite hangs.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// ResolveAndPull resolves a pull argument (URL or alias) and pulls herds.
// If the argument is a URL, it pulls directly.
// If it's a short name, it looks up aliases (user config → embedded defaults).
// On first alias use, writes default config to disk for discoverability.
func ResolveAndPull(ctx context.Context, arg string, cfg PullConfig) error {
	// URL detection — pass through to Pull directly.
	if isURL(arg) {
		return Pull(ctx, arg, cfg)
	}

	// Load aliases.
	aliases, source, err := LoadAliases()
	if err != nil {
		return fmt.Errorf("load aliases: %w", err)
	}

	// Auto-scaffold config file on first alias use.
	if !cfg.DryRun {
		EnsureAliasesConfig(source, cfg.Logger.Info)
	}

	// Resolve alias.
	urls := ResolveAlias(arg, aliases)
	if urls == nil {
		names := SortedAliasNames(aliases)
		return fmt.Errorf("unknown herd %q — not a URL and no matching alias.\n\nAvailable: %s\nRun 'promptherder list' to see descriptions.", arg, strings.Join(names, ", "))
	}

	// Pull each URL in the alias.
	var errs []error
	for _, u := range urls {
		if pullErr := Pull(ctx, u, cfg); pullErr != nil {
			errs = append(errs, pullErr)
			cfg.Logger.Error("pull failed", "url", u, "error", pullErr)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("pull %q: %d of %d herds failed", arg, len(errs), len(urls))
	}
	return nil
}

// PullConfig holds the configuration for a pull operation.
type PullConfig struct {
	Locked   bool         // Require the recorded immutable revision and content digest.
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
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("cannot derive herd name from URL: %s", gitURL)
	}

	owner, repo := ownerRepoFromURL(gitURL)
	if owner == "" || repo == "" {
		return fmt.Errorf("cannot parse owner/repo from URL: %s (expected https://github.com/OWNER/REPO)", gitURL)
	}

	archiveURL := toArchiveURL(owner, repo)
	var locked compiler.HerdLock
	if cfg.Locked {
		data, err := os.ReadFile(filepath.Join(cfg.RepoPath, ".promptherder/lock.json"))
		if err != nil {
			return err
		}
		var locks map[string]compiler.HerdLock
		if err := json.Unmarshal(data, &locks); err != nil {
			return err
		}
		locked = locks[name]
		if !regexp.MustCompile(`^[a-fA-F0-9]{40}$`).MatchString(locked.Revision) || locked.Repository != "https://github.com/"+owner+"/"+repo {
			return fmt.Errorf("herd %s has no matching immutable revision in lock.json; pull and sync first", name)
		}
		archiveURL += "/" + locked.Revision
	}
	herdPath := filepath.Join(cfg.RepoPath, herdsDir, name)

	if cfg.DryRun {
		if isDirectory(herdPath) {
			cfg.Logger.Info("dry-run: would update herd", "name", name, "url", archiveURL)
		} else {
			cfg.Logger.Info("dry-run: would download herd", "name", name, "url", archiveURL, "path", herdPath)
		}
		return nil
	}

	revision := locked.Revision
	if revision == "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+owner+"/"+repo+"/commits/HEAD", nil)
		if err != nil {
			return err
		}
		response, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("resolve immutable herd revision: %w", err)
		}
		var commit struct {
			SHA string `json:"sha"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&commit)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("resolve herd revision: HTTP %d", response.StatusCode)
		}
		if decodeErr != nil || !regexp.MustCompile(`^[a-fA-F0-9]{40}$`).MatchString(commit.SHA) {
			return fmt.Errorf("GitHub returned an invalid immutable revision")
		}
		revision = commit.SHA
		archiveURL += "/" + revision
	}
	cfg.Logger.Info("downloading herd", "name", name, "url", archiveURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", archiveURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", archiveURL, resp.StatusCode)
	}

	parent := filepath.Dir(herdPath)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(filepath.Dir(parent), ".pull-")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.RemoveAll(staging)
		}
	}()
	download := filepath.Join(staging, "new")
	if err := extractTarGz(resp.Body, download); err != nil {
		return fmt.Errorf("extract herd %s: %w", name, err)
	}
	data, err := os.ReadFile(filepath.Join(download, herdMetaFile))
	if err != nil {
		return fmt.Errorf("herd %q has no readable %s: %w", name, herdMetaFile, err)
	}
	var meta struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("invalid herd.json: %w", err)
	}

	sum, err := compiler.HerdDigest(download)
	if err != nil {
		return err
	}
	if cfg.Locked && sum != locked.Digest {
		return fmt.Errorf("herd %s does not match locked content digest", name)
	}
	source, err := json.MarshalIndent(compiler.HerdLock{Repository: "https://github.com/" + owner + "/" + repo, Revision: revision, Version: meta.Version, Digest: sum}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(download, ".source.json"), source, 0644); err != nil {
		return err
	}
	backup := filepath.Join(staging, "previous")
	hadPrevious := false
	if _, err := os.Lstat(herdPath); err == nil {
		if err := os.Rename(herdPath, backup); err != nil {
			return err
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(download, herdPath); err != nil {
		if hadPrevious {
			if restoreErr := os.Rename(backup, herdPath); restoreErr != nil {
				keepBackup = true
				return fmt.Errorf("install failed: %v; previous herd retained at %s: %w", err, backup, restoreErr)
			}
		}
		return err
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
			mode := os.FileMode(0644)
			if hdr.Mode&0111 != 0 {
				mode = 0755
			}
			f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			n, err := io.Copy(f, io.LimitReader(tr, maxExtractFileSize+1))
			f.Close()
			if err != nil {
				_ = os.Remove(target)
				return fmt.Errorf("write %s: %w", target, err)
			}
			if n > maxExtractFileSize {
				_ = os.Remove(target)
				return fmt.Errorf("file %q exceeds size limit (%d bytes)", hdr.Name, maxExtractFileSize)
			}
		}
	}

	return nil
}
