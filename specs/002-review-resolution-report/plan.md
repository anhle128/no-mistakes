# Implementation Plan: Review Resolution Report

**Branch**: `002-review-resolution-report` | **Date**: 2026-06-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/002-review-resolution-report/spec.md`

## Summary

Create a durable Review-only resolution report whenever the Review step records findings. The implementation will rebuild report state from persisted Review step rounds, classify each normalized finding once, write a repo-local Markdown artifact under `no-mistakes/<branch-slug>/review-resolution.md`, persist compact metadata in SQLite, expose compact status/counts/path through AXI/TUI surfaces, and include compact PR status plus the repo-relative committed report path without leaking absolute local filesystem details.

The core design is a new review-report domain package backed by additive DB metadata and repo-local path helpers. The pipeline executor refreshes the report at Review lifecycle boundaries and before PR summary generation. The push step force-adds exactly the current run's report artifact when it exists. The Review fix agent contract remains backward-compatible with the existing `summary` field while accepting validated optional `resolutions[]` details.

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: Standard library, SQLite driver already used by `internal/db`, Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, TOON renderer, existing Git/provider helpers
**Storage**: `$NM_HOME/state.sqlite`, repo-local `no-mistakes/<branch-slug>/review-resolution.md`, existing step results/rounds, local logs/evidence directories
**Testing**: Targeted Go tests, golden Markdown fixtures, IPC/AXI/TUI render tests, tagged E2E journey tests, then `go test -race ./...`, `make lint`, `make skill`
**Target Platform**: macOS, Linux, Windows CLI/daemon
**Project Type**: Go CLI/daemon with docs site and generated agent skill text
**Performance Goals**: Report refresh is bounded by one run's Review rounds; AXI/TUI/PR read compact metadata rather than reparsing Markdown; gate push remains daemon-backed and responsive
**Constraints**: Do not modify `origin`; create and commit only the exact repo-local report artifact for the current run; force-add that report if ignored without force-adding unrelated `no-mistakes/` files; report writes and metadata writes must be atomic from consumer perspective; report content must not store raw transcripts, raw logs, raw hunks, code blocks, or secret-like values
**Scale/Scope**: Per-run local artifact and metadata for branch-scoped gate runs; no all-step evidence report in this feature

## Constitution Check

*GATE: Passed before Phase 0 research; re-checked after Phase 1 design.*

- **Explicit Gate Semantics**: PASS. The design adds Review evidence and does not change pass/fail semantics, step order, `origin`, or push behavior.
- **Isolation and User Control**: PASS. Report writes are intentionally bounded to repo-local `no-mistakes/<branch-slug>/` because the report is PR evidence; SQLite metadata remains local runtime support. Acceptance without fix requires persisted user or documented policy provenance; aborted/superseded/canceled runs keep unresolved findings still open.
- **Evidence-First Quality**: PASS. The verification plan includes unit, DB migration, pipeline lifecycle, IPC/AXI/TUI, PR summary, and E2E coverage.
- **Agent-Agnostic Contracts**: PASS. The fix schema extends the existing structured JSON contract while preserving legacy `summary`; surfaces render persisted enum/counts instead of recomputing success language.
- **Simplicity and Recovery**: PASS. A single review-report package owns classification/rendering; DB migrations are additive; report refresh returns actionable errors on required write failures.
- **Docs and Generated Artifacts**: PASS. Docs and `skills/no-mistakes/SKILL.md` are in scope because user-facing Review, AXI/TUI, PR, and generated skill behavior changes.
- **Red Team Gate**: SATISFIED. The feature directory contains `red-team-findings-applied-2026-06-18-232904.md`, satisfying the installed before-plan gate for this qualifying spec.

## Project Structure

### Documentation (this feature)

```text
specs/002-review-resolution-report/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── review-resolution-report.md
└── tasks.md              # Generated later by /speckit-tasks
```

### Source Code

```text
internal/db/                  # Additive schema, report metadata accessors, fix/decision evidence persistence
internal/reviewreport/        # Repo-local path helpers plus report classification, sanitization, rendering, atomic refresh
internal/pipeline/            # Executor lifecycle hooks for report refresh/failure propagation
internal/pipeline/steps/      # Review fix schema extension, exact report staging, and PR summary integration
internal/ipc/                 # Report metadata wire contract for local surfaces
internal/cli/                 # AXI compact report rendering
internal/tui/                 # TUI compact report rendering
docs/src/content/docs/        # Review, auto-fix, pipeline, TUI/AXI, PR documentation
skills/no-mistakes/           # Generated agent-facing behavior text
internal/e2e/                 # Tagged end-to-end report scenarios
```

**Structure Decision**: Add a focused `internal/reviewreport` package rather than spreading classification/rendering rules across executor, PR summary, AXI, and TUI. DB remains the compact state authority; Markdown is the human-readable repo-local artifact.

## Complexity Tracking

No constitution violations are expected.

No complexity-tracking rows are needed.

## Phase 0: Research

Completed in [research.md](./research.md).

Key decisions:

- Rebuild report entries from persisted Review `step_rounds`, not logs or transcripts.
- Use normalized finding IDs as the primary identity.
- Add a dedicated `review_resolution_reports` metadata table keyed by `run_id`.
- Persist fix-round evidence on rounds and terminal acceptance provenance in explicit decision metadata.
- Keep PR content absolute-local-path-free and narrative-free while showing the repo-relative report path; AXI/TUI may show the repo-local report path.
- Treat report generation as required once Review findings exist; write/metadata failures fail the Review step or run.

## Phase 1: Design

Completed artifacts:

- [data-model.md](./data-model.md)
- [contracts/review-resolution-report.md](./contracts/review-resolution-report.md)
- [quickstart.md](./quickstart.md)

### Core Design

1. **Persistence and paths**
   - Add `internal/reviewreport` helpers for `no-mistakes/<branch-slug>/review-resolution.md` path derivation from the run branch/workdir, using the grill-me branch safety model.
   - Add `review_resolution_reports` to `internal/db/schema.go` and migrations for existing databases.
   - Add DB accessors for report metadata upsert/read and integrity status.
   - Add fix evidence fields to `step_rounds`: `fix_commit_sha`, `no_commit_reason`, and optional validated `fix_resolution_details_json`.
   - Add decision evidence for Review terminal actions, keyed by run/finding/round, so approve/skip/abort/fix provenance is reconstructable without transcripts.

2. **Report domain package**
   - New `internal/reviewreport` package owns:
     - loading Review step and rounds,
     - normalizing/reconciling findings by ID,
     - classifying `resolved`, `accepted`, `informational`, and `still_open`,
     - validating optional fix-agent `resolutions[]`,
     - sanitizing and truncating untrusted content,
     - rendering deterministic Markdown from a stable format version,
     - atomically refreshing Markdown and metadata with a consumer-safe protocol: render and classify the Review snapshot in memory; write Markdown bytes to a temporary file in the target report directory; close/fsync the file and atomically rename it over `no-mistakes/<branch-slug>/review-resolution.md`; compute `content_hash` and `source_watermark` from the exact rendered bytes and source rounds/decisions; then upsert `review_resolution_reports` in one DB transaction that preserves `first_generated_at` and updates counts, watermark, hash, and `last_refresh_result`. If the file write or rename fails, leave previous Markdown/metadata untouched and return an actionable Review error. If the DB transaction fails after rename, consumers must detect the hash/watermark mismatch and render degraded or evidence-unavailable status instead of confident counts. Refresh tests must assert this order and both partial-failure paths.
   - The package must keep raw diffs/logs/transcripts out of persisted report data. Diff-derived fallback is a short sanitized summary plus changed-file labels and commit SHA when available.

3. **Pipeline lifecycle hooks**
   - In `internal/pipeline/executor.go`, refresh the report after the Review step first persists findings, after fix rounds, after approve/skip/abort/failure decisions, and when a run with Review findings becomes failed/canceled/superseded/stale.
   - If Review findings exist and required report/metadata refresh fails, return an actionable error and fail the Review step/run.
   - Do not refresh or create a report for clean Review runs with no findings across all Review rounds.
   - Before PR summary generation, reconcile metadata once so PR content reflects the latest persisted classified snapshot.

4. **Fix agent contract**
   - Extend the Review fix schema from `{"summary": string}` to `{"summary": string, "resolutions": [...]}`.
   - Preserve legacy `summary`-only responses.
   - Validate `resolutions[]` as untrusted: required non-empty fields, unique selected finding IDs, no duplicate/unknown IDs, bounded strings, bounded changed-file labels.
   - Store degraded evidence when structured output is partial or invalid; do not silently treat it as complete.

5. **Surfaces**
   - Extend IPC `RunInfo` or a nested report metadata object with status, counts, and repo-local path.
   - AXI and TUI render compact status/counts/path only when a report exists.
   - PR summaries render compact status/counts plus the repo-relative report path only. They must never include absolute local paths or report excerpts.
   - Push staging force-adds exactly the current run's report path if present, even when `no-mistakes/` is ignored, and rejects unrelated report paths.

6. **Docs and generated skill text**
   - Update Review, auto-fix, pipeline, gate, TUI/AXI, PR, local state, and generated `/no-mistakes` skill documentation where behavior changes are user-visible.

## Phase 2: Task Planning Approach

`/speckit-tasks` should split work into independently testable slices:

1. DB/repo-local path foundation and migrations.
2. Review report domain model, classification, sanitization, and golden Markdown rendering.
3. Review fix schema extension and fix evidence persistence.
4. Executor lifecycle refresh/failure behavior.
5. IPC, AXI, TUI, and PR summary surfacing.
6. E2E journeys and documentation/generated skill updates.

Parallelization is possible after the DB/path contract lands: report rendering, PR summary rendering, AXI/TUI rendering, and docs can proceed with fixtures against the shared metadata contract.

## Verification Plan

Targeted tests:

- `internal/reviewreport`: classification, repeated IDs, changed IDs, no-op informational entries, structured vs inferred evidence, sanitization, truncation, deterministic Markdown golden files.
- `internal/db`: migration preserves existing runs/steps/rounds and creates metadata; metadata upsert/read/status behavior; integrity mismatch degrades status.
- `internal/pipeline`: report creation after first Review findings; refresh after fix/approve/skip/abort/failure; required write failure fails the Review step/run.
- `internal/pipeline/steps`: PR summary includes compact metadata plus repo-relative report path, omits absolute paths/excerpts, and push force-adds only the exact current report.
- `internal/ipc`, `internal/cli`, `internal/tui`: report metadata wire shape and compact rendering.
- `internal/e2e`: mixed resolved/accepted run, clean Review run, aborted unresolved run, structured `resolutions[]`, legacy `summary` fallback, PR privacy.

Commands:

```sh
gofmt -w <changed go files>
go test ./internal/reviewreport ./internal/db ./internal/pipeline ./internal/pipeline/steps ./internal/ipc ./internal/cli ./internal/tui ./internal/daemon
go test -tags=e2e ./internal/e2e -run 'ReviewResolution|Axi'
go test -race ./...
make lint
make skill
```

## Open Risks

- Existing round data does not yet include enough terminal-decision provenance to distinguish approve vs skip vs policy acceptance for every finding. The implementation must add explicit persisted decision evidence rather than infer from final step status.
- `commitAgentFixes` currently updates the run head but does not return/store the fix commit SHA on the round. The implementation must capture this deterministically, including no-op/no-commit cases.
- Follow-up Review scope equivalence is not encoded today. The report classifier must record verification as inconclusive when comparable parsed follow-up evidence cannot be established.
- Generated PR summary and local AXI/TUI renderers must share persisted metadata to avoid count/status drift.
