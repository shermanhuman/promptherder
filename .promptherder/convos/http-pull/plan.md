# Plan: Replace git pull with HTTP tarball download

> `/plan` · **Status**: approved · `.promptherder/convos/http-pull/plan.md`

## Ideas

| #   | Idea                                           | State    | Rationale                                              |
| --- | ---------------------------------------------- | -------- | ------------------------------------------------------ |
| 1   | Replace git clone/pull with HTTP tarball fetch | accepted | Removes git dependency, eliminates nested .git footgun |
| 2   | Remove .promptherder/herds/ from .gitignore    | accepted | No .git dir → safe to commit or not, user's choice     |
| 3   | Add archive URL builder for GitHub             | accepted | GitHub archive API: /archive/refs/heads/master.tar.gz  |
| 4   | Extract tar.gz with prefix stripping           | accepted | GitHub tarballs have owner-branch/ prefix              |
| 5   | Remove splitFlagsAndArgs from main.go          | rejected | Still needed for other flags with pull                 |

## Goal

Replace the git-based `promptherder pull` with an HTTP tarball download using only stdlib (`net/http`, `archive/tar`, `compress/gzip`). This eliminates the `git` binary dependency and the nested `.git` directory footgun that could corrupt users' repos.

## Plan

1. **Rewrite `pull.go` — HTTP tarball download**
   - Files: `internal/app/pull.go`
   - Change: Replace `git clone`/`git pull` with HTTP GET of GitHub archive tarball. Extract tar.gz with top-level prefix stripping. Remove `os/exec` import, add `net/http`, `archive/tar`, `compress/gzip`. Keep `PullConfig`, `Pull`, `herdNameFromURL`. Add `toArchiveURL` and `extractTarGz` helpers. For updates, delete existing herd dir and re-download (idempotent).
   - Verify: `go vet ./...`, `go test ./... -count=1`

2. **Update `pull_test.go` — archive URL + extraction tests**
   - Files: `internal/app/pull_test.go`
   - Change: Add `TestToArchiveURL` for URL conversion. Add `TestExtractTarGz` using an in-memory tar.gz fixture. Keep existing `TestHerdNameFromURL`.
   - Verify: `go test ./internal/app/ -run "Archive|Extract|HerdName" -v`

3. **Update `.gitignore` — remove herds exclusion**
   - Files: `.gitignore`
   - Change: Remove `.promptherder/herds/` line. No .git dirs in herds anymore, so no footgun.
   - Verify: `git check-ignore .promptherder/herds/compound-v` should fail (NOT ignored)

4. **Integration test — pull and run end-to-end**
   - Files: n/a
   - Change: Build, run `promptherder pull https://github.com/shermanhuman/compound-v`, verify no `.git` dir, run `promptherder` to merge+install.
   - Verify: `ls .promptherder/herds/compound-v/.git` should not exist. `promptherder` should succeed.

## Risks & mitigations

| Risk                                      | Mitigation                                                       |
| ----------------------------------------- | ---------------------------------------------------------------- |
| GitHub rate limit (60/hr unauthenticated) | Tarballs are small, users pull rarely. Future: support token.    |
| Non-GitHub hosts                          | Archive URL pattern differs. Start GitHub-only, document.        |
| No incremental update                     | Re-download full tarball (~few KB for rules/skills). Acceptable. |

## Rollback plan

Revert to git-based pull.go from v0.8.0.

## Deferred

- Support for GitLab/Bitbucket archive URLs
- Auth token for private repos
- Tag/version pinning (download specific tag instead of master)
