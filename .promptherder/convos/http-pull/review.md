# Review: HTTP Tarball Pull

## Blockers

None.

## Majors

None.

## Minors

### M1: GitHub-only URL parsing

`ownerRepoFromURL` only handles `github.com` URLs. Non-GitHub URLs (GitLab, Bitbucket) will get a clear error: `"cannot parse owner/repo from URL"`. This is acceptable for v1 and documented in the plan's deferred items.

## Nits

### N1: No `io.LimitReader` on HTTP response body

`extractTarGz` reads the entire response body without a size limit. A malicious or misconfigured server could send an unbounded stream. Low risk since we're reading from GitHub's API, but a `LimitReader` would be defensive.

### N2: File permissions not preserved from tar headers

`os.Create` uses default permissions (0666 minus umask). The tar headers contain mode bits but they're not applied. Not an issue for markdown/text files in herds.

## Verification

| Metric                             | Result                             |
| ---------------------------------- | ---------------------------------- |
| `go vet ./...`                     | ✅ pass                            |
| `go test ./...`                    | ✅ pass (all 3 packages)           |
| `go build`                         | ✅ pass                            |
| `promptherder pull <url>`          | ✅ downloads via HTTP, no .git     |
| `promptherder pull <url> -dry-run` | ✅ logs without downloading        |
| `promptherder` (bare run)          | ✅ merges and installs all targets |
| No `.git` in herds                 | ✅ confirmed                       |
| `.gitignore` updated               | ✅ herds not ignored               |
| Path traversal test                | ✅ rejected                        |

## Summary of changes

| File                        | Change                                                                   |
| --------------------------- | ------------------------------------------------------------------------ |
| `internal/app/pull.go`      | Replaced git clone/pull with HTTP tarball download (stdlib only)         |
| `internal/app/pull_test.go` | Added ownerRepoFromURL, toArchiveURL, extractTarGz, path traversal tests |
| `.gitignore`                | Removed `.promptherder/herds/` exclusion                                 |

## Follow-ups

- GitLab/Bitbucket archive URL support (deferred)
- Auth token for private repos (deferred)
- Tag/version pinning (deferred)
- `io.LimitReader` on response body (N1)
