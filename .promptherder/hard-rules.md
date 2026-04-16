---
trigger: always_on
---

# Hard Rules

Project-level rules that are always active. Added by `/rule` or manually.

- **Releases: semver.** Tag format is `vX.Y.Z`. Version is injected via goreleaser ldflags — no version file to edit. To release: `git tag vX.Y.Z && git push origin master && git push origin vX.Y.Z`. Bump minor for new features. Bump patch for fixes and improvements. Do not bump major without explicit approval — major is reserved for a stability milestone (v1.0.0). Never release without running `go test ./...` first. Never release with a dirty working tree.

- **Merging is a human task.** Never merge a branch, squash-merge a PR, or push directly to the default branch. You may create branches, push to feature branches, and create PRs. When work is ready to merge, ask the user to merge. This rule has no exceptions.
