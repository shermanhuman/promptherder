# Herd compatibility review

Reviewed September 8, 2026 against the Promptherder 1.0.0 compiler. Companion changes are on `feat/native-herd-alignment` in Compound V, Grugg, and Oh. This is a source and compiler review, not evidence from running paid models or live API operations.

| Finding | Correction |
|---|---|
| Planning wrappers stopped execution even after full-pipeline authorization | Separate plan-only behavior from callable pipeline/execution helpers; carry task scope across phases |
| Review offered FIX, then prohibited all changes except YOLO | Ordinary fix authorization now permits relevant fixes; review-only requests remain read-only |
| Deferred tasks became implicit acceptance criteria | Compare against the accepted task; deferred ideas stay optional |
| Slug selection chose the most recently edited unrelated task | Resolve the active context or a unique matching slug; preserve existing task dates |
| Stack updates wrote generated Antigravity files | Author observed versions in `.promptherder/stack.md`; distinguish inventory from upgrades |
| Added hard rules claimed immediate automatic enforcement | Sync configured output and report any pending sync or host reload |
| Host-specific execution mechanics obscured the intended methodology | Retain mandatory research, batch checkpoints, full final test suite, templates, and ten review checks; translate tool mechanics and respect real dependencies |
| Grugg reset normal mode and conflicted with review/PR formats | Latest style choice persists; artifact schemas, evidence, and meaningful uncertainty take precedence |
| Grugg documented nonexistent environment/config activation | Remove unsupported runtime configuration claims; document explicit source overrides |
| Compression overwrote source before validating, and repair exceptions left it changed | Validate in private staging, retain exact backup bytes, detect intervening edits, then replace atomically |
| Compression silently selected Claude; validator accepted changed headings/paths/inline code | Require a local candidate or explicit provider; fail on protected structural changes |
| Version preparation could tag/push and insisted on main/clean-tree approval | Separate metadata preparation from requested publishing; inspect actual version sources and CI |
| General MCP preference conflicted with the specific GitHub tooling rule | Keep mise-first policy; GitHub explicitly uses `mise exec -- gh`, while MCP-first applies to other operations |
| API examples contained incorrect webhook placement, ignored HTTP failures, tuple/list misuse, and unbounded sync assumptions | Correct examples and scope verification; preserve dated sandbox observations as historical evidence |
| Docker guidance claimed cached apk upgrades refresh packages | Document cache invalidation explicitly; remove assumptions about CI scanners and image automation |
| Large API guides always loaded with the skill | Move focused guides and the unchanged Tekmetric catalog into referenced resources |
| Prefixed workflows failed manifest dependency lookup | Resolve canonical workflow declarations to generated IDs in the compiler |
| Interpreter caches entered native bundles | Exclude Python runtime/test cache directories during source discovery |

## Versions

- Compound V: 0.10.0 → 1.0.0 (native entry points and phase semantics)
- Grugg: 1.1.1 → 2.0.0 (explicit compression inputs/provider and corrected style boundaries)
- Oh: 0.2.4 → 1.0.0 (native compiler guidance, release boundaries, and resource layout)

## Verification

- All 25 skill entry points pass the skill-creator validator.
- Twelve offline compression regressions cover failure, retries, protected content, exact backups, permissions, and concurrent edits.
- Combined Codex/Claude compilation passes strict validation with the restored, condensed policies at 7,967 bytes, under the 8 KiB quality target. All seven profiles compile with documented duplicate-discovery and manual-invocation fallback warnings.
- Plan/apply hashes match. A selected, prefixed planning workflow includes its callable pipeline/execution/review dependency closure.
- The relocated Tekmetric endpoint catalog is byte-for-byte identical to its prior tracked version.
- Promptherder Go tests and vet pass, including the prefix/cache regression.
- Benchmark CLI dry-run performs no API calls and requires explicit provider/model selection. No fresh token-savings or model-adherence claim is made.

## Behavioral cases reviewed in the instructions

| Request | Expected transition |
|---|---|
| “Plan this; do not implement” | Produce plan and stop |
| “Implement that plan” on a later turn | Reuse task and implement without a special command requirement |
| “Complete this task autonomously” | Plan → execute → verify → review/fix within existing authorization |
| “Review only” | Report findings without code changes |
| “Review and fix” or later “FIX” | Repair relevant findings; no YOLO keyword required |
| “Normal mode” after Grugg | Keep normal prose until a new style request |
| “Bump major” during feature work | Update version metadata; do not tag, merge, publish, or deploy |
| “Record the stack” | Record observed versions; do not silently upgrade them |
| Compress an instruction file | Review meaning in a separate candidate before validated replacement |

These transitions still need running-host trials, including compaction, to establish model behavior. Enterprise/personal overrides and host settings may change which instructions load.

## Evidence for corrected service examples

- [Docker cache invalidation](https://docs.docker.com/build/cache/invalidation/)
- [Postmark email API](https://postmarkapp.com/developer/api/email-api) and [error codes](https://postmarkapp.com/developer/api/overview)
- [Groq speech-to-text models and limits](https://console.groq.com/docs/speech-to-text)
- [Telnyx Voice API webhook envelope](https://developers.telnyx.com/docs/voice/programmable-voice/voice-api-webhooks)
- [Claude CLI tool controls](https://code.claude.com/docs/en/cli-reference)

WaxSeal's local command source was inspected for command shape and the sequence of remote/local writes. No secrets were retrieved or changed; no real email, phone call, release, or deployment was performed.

## Repository policy preservation

A second review restored concrete instructions that the initial alignment had over-generalized. Oh requires mise-first tooling, `mise exec -- gh`, version bumps before PRs, and human merging. Compound V retains its plan template, decision table, output formatting, detailed ten-check review, parallel batches, research, checkpoints, and final full-suite verification. Grugg retains its explicit terse default and intensity/persistence rules. Docker guidance retains the preferred Alpine strategy and concrete build recipe, with the cache-freshness correction. Reduced context size is not a reason to remove these policies; longer conditional checklists belong in referenced resources.
