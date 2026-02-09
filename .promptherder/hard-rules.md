---
trigger: always_on
---

# Hard Rules

Project-level rules that are always active. Added by `/rule` or manually.

- **Releases: default to patch.** Tag format is `vX.Y.Z`. Version is injected via goreleaser ldflags — no version file to edit. To release: `git tag vX.Y.Z && git push origin main && git push origin vX.Y.Z`. Bump patch for every fix, feature, or improvement. Bump minor only for breaking changes (renamed/removed flags, changed behavior users must adapt to). Do not bump major without explicit approval — major is reserved for a stability milestone (v1.0.0). Never release without running `go test ./...` first. Never release with a dirty working tree.
