# Research: Review Resolution Report

## Inputs Read

- `specs/002-review-resolution-report/spec.md`
- `specs/002-review-resolution-report/red-team-findings-applied-2026-06-18-232904.md`
- `internal/pipeline/executor.go`
- `internal/pipeline/pipeline.go`
- `internal/pipeline/steps/review.go`
- `internal/pipeline/steps/common_fix.go`
- `internal/db/schema.go`
- `internal/db/round.go`
- `internal/paths/paths.go`
- `internal/pipeline/steps/pr.go`
- `internal/pipeline/steps/prsummary.go`
- `internal/types/findings.go`
- `internal/ipc/protocol.go`
- `internal/cli/axi_render.go`
- `internal/tui/pipeline.go`

## Decisions

### Use persisted Review rounds as the source of truth

**Decision**: Rebuild the report from Review `step_results` and `step_rounds`, plus added decision/fix evidence.

**Rationale**: The executor already persists every round's findings, trigger type, selected finding IDs, user-vs-auto selection source, merged user findings, and fix summary. This is the durable source required by the spec and avoids parsing logs or transcripts.

**Rejected**: Parse logs, agent transcripts, or PR summaries. These are lossy, privacy-sensitive, and do not preserve terminal decision provenance reliably.

### Add a focused review-report package

**Decision**: Create `internal/reviewreport` for classification, sanitization, Markdown rendering, and atomic refresh.

**Rationale**: The rules are cross-cutting. Centralizing them keeps executor, PR, AXI, and TUI code from duplicating status/count logic.

**Rejected**: Put classification directly in the executor. That would couple lifecycle orchestration to report rendering and make PR/UI drift more likely.

### Keep report artifact local under `$NM_HOME`

**Decision**: Write Markdown to `$NM_HOME/reports/<runID>/review-resolution.md`.

**Rationale**: This matches the reconciled spec direction and keeps private local evidence out of the target repository and public PR body.

**Rejected**: Repo-local `no-mistakes/<branch-slug>/review-resolution.md`. The spec explicitly rejects staging, force-adding, or committing local reports.

### Persist compact metadata separately

**Decision**: Add `review_resolution_reports` keyed by `run_id`.

**Rationale**: AXI, TUI, and PR summaries need status/counts/path without reparsing Markdown. A dedicated row also supports lifecycle timestamps, watermarks, integrity fields, and degraded/evidence-unavailable states.

**Rejected**: Store metadata only inside Markdown front matter. This would force every surface to parse an untrusted human-readable file and would be harder to keep atomic with DB state.

### Add explicit decision and fix evidence

**Decision**: Persist terminal Review decision evidence and fix commit/no-commit evidence.

**Rationale**: Existing final step status and selected finding IDs are not enough to prove accepted-without-fix versus still-open after abort/failure/supersede. Existing `FixSummary` is also insufficient to distinguish no-op, failed, and no-commit fix attempts.

**Rejected**: Infer acceptance from a completed/skipped step alone. The red-team-applied spec specifically tightened this trust boundary.

### Extend fix schema while preserving legacy behavior

**Decision**: Keep `summary` required and add optional `resolutions[]`.

**Rationale**: Existing agents and tests expect a one-line summary. Optional per-finding detail adds richer report content without breaking old fix agents.

**Rejected**: Require `resolutions[]` immediately. That would break legacy agent responses and make no-structured-output cases impossible to represent honestly.

### Classify by normalized finding ID

**Decision**: Use finding ID as the primary entry identity. Repeated same-ID sightings update one entry; changed/ambiguous IDs remain still open or verification-inconclusive.

**Rationale**: `types.NormalizeFindings` already gives deterministic IDs when missing. The spec rejects coalescing by file, line, or description because those heuristics can silently misclassify findings.

**Rejected**: Match by file/line/description. This risks false resolution or false acceptance when findings shift between review rounds.

### Surface metadata with privacy boundaries

**Decision**: AXI/TUI may show local report path; PR summaries only show compact status/counts.

**Rationale**: AXI/TUI are local surfaces; PR body may be public and must not expose local filesystem details, report excerpts, or fake blob links.

**Rejected**: Link to local report in PR body. `$NM_HOME` artifacts are not public repository files.

### Fail required write failures

**Decision**: Once Review findings exist, failure to write the report or metadata must fail the Review step/run with an actionable error.

**Rationale**: The report is required evidence. Silent degradation would let the pipeline claim review resolution evidence exists when it does not.

**Rejected**: Best-effort warning-only generation. This violates FR-025 and makes UI/PR status untrustworthy.

## Consequences

- DB migrations need careful additive tests against older schemas.
- Executor hooks need to distinguish clean Review from Review-with-findings at several lifecycle points.
- Report refresh must be atomic across Markdown and metadata from the perspective of readers.
- Sanitization needs unit tests for Markdown controls, raw diff/log/code blocks, transcript-like content, secret-like values, and oversized fields.
