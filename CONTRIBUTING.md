# Contributing to promptherder

The CLI compiles all selected targets together through `internal/compiler`. Extend this pipeline for new behavior. Older exporters in `internal/app` remain internal migration-era code; the CLI no longer calls them.

## Pipeline

- `content.go`: real YAML parsing, deterministic source resolution, herd declarations, dependency and resource validation, provenance.
- `plan.go`: capability profiles, host rendering, shared artifact ownership, discovery and budget diagnostics, lock validation.
- `install.go`: ownership checks, adoption, stale cleanup, concurrency checks, journaled atomic writes, recovery.
- `doctor.go`: reviewable JSON output and read-only personal discovery diagnostics.
- `cmd/promptherder`: explicit target setup, command routing, and settings.
- `internal/app/pull.go`: staged herd downloads with revision and digest verification.

`Build` must remain read-only. Adapters return artifacts; only `Apply` writes them. Keep dry-run and real sync on the same build path. Check the union of discovery roots before applying any output. Do not silently overwrite user files, flatten skill bodies into the baseline, discard activation metadata, or enable new targets by default.

## Adding or changing a host

Record official documentation and verification dates in `Capabilities`. Add native rendering and update `visible`/`skillRoot` for every root the host discovers, including roots shared with other hosts. Register the explicit target in settings and CLI dispatch. Update help, README, and cross-host fixtures.

Unsupported semantics need a diagnostic. Required behavior must fail when a host can only approximate it. Tests should cover collisions across hosts, resource preservation, invocation controls, scoped activation, and removal of one consumer of shared files.

Repository fixtures prove file behavior. Running-agent evaluations are separate evidence: record the host version, model, settings, task, artifacts actually loaded, and outcome. Do not claim model adherence based on a passing compiler test.

## Validation

```bash
go test ./...
go vet ./...
go build ./cmd/promptherder
```

Compiler tests cover real parser failures, bundle copying, modes, dependency closure, variants, scoped fallbacks, ownership, adoption, recovery, and locks. Downloader tests verify that failed downloads and digest mismatches preserve the previous herd.

For integration checks, copy herds into a temporary project's `.promptherder/herds/`, select targets explicitly, compare `plan --json` with installed artifacts, run sync twice, and remove targets. Avoid modifying actual user host configuration during tests.

Release versions are stored in `VERSION` and injected by GoReleaser. The 1.0.0 major bump covers native output layouts, explicit target setup, namespaced workflow invocation, conflict handling, and manifest/lock changes. Publishing a release is a separate action.
