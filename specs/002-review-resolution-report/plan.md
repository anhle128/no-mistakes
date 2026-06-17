# Implementation Plan: Review Resolution Report

**Branch**: `002-review-resolution-report` | **Date**: 2026-06-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/002-review-resolution-report/spec.md`

## Summary

Add a run-scoped, durable Markdown report that explains review findings, selected decisions, fix attempts, applied fix summaries, and the latest trustworthy review outcome. The implementation will preserve existing review, approval, auto-fix, push, PR, and CI behavior by deriving the report from existing persisted review step state (`step_results` and `step_rounds`) plus a new report metadata record, then exposing only a compact report reference and summary metadata through AXI, TUI, and PR surfaces.

The report generator is a reporting layer, not a new review engine. It must fail closed when review data is missing, unreadable, stale, or internally inconsistent, and report generation failures must not change the pipeline outcome.

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: Existing Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, SQLite, Git/provider CLIs, and TOON output support; no new dependency is planned.
**Storage**: SQLite under `NM_HOME` plus a generated Markdown artifact under `$NM_HOME/reports/<runID>/review-resolution.md`. Existing `$NM_HOME/logs/<runID>` remains for logs, and isolated worktrees remain under `$NM_HOME/worktrees`.
**Testing**: Targeted Go tests for report generation, DB migration/persistence, IPC projection, AXI/TUI rendering, and PR summary insertion; then `go test -race ./...`, `make lint`, and `make docs-build` because user-visible docs change.
**Target Platform**: macOS, Linux, and Windows CLI/daemon.
**Project Type**: Go CLI/daemon with terminal UI, headless agent interface, provider PR integration, generated agent skill, and Astro docs.
**Performance Goals**: Report generation must be bounded by persisted review data size and must not block long-running agent work. AXI/TUI/PR surfaces should read persisted metadata and not parse the full Markdown report on hot paths.
**Constraints**: Preserve `origin` behavior, fixed pipeline order, approval semantics, auto-fix limits, push/PR/CI behavior, and transcript/log privacy boundaries. Raw diffs, raw logs, raw transcripts, and code excerpts are never report inputs.
**Scale/Scope**: One current report per run, updated as review-resolution evidence changes. Historical regeneration is supported from stored run data with stale/unavailable labels for partial evidence.

## Constitution Check

*GATE: Passed before Phase 0 research. Re-check passed after Phase 1 design.*

- **Explicit Gate Semantics**: PASS. The feature reports review state only; it does not alter `git push no-mistakes`, step order, approval actions, auto-fix limits, provider pushes, PR creation, or CI monitoring.
- **Isolation and User Control**: PASS. Report files live under `NM_HOME`, not the user's working tree. `Accepted` requires explicit stored human risk-acceptance evidence and is not inferred from `approve`, `skip`, or unselected findings.
- **Evidence-First Quality**: PASS. The plan identifies focused tests for report content, partial data, unreadable final findings, multiple rounds, user-authored findings, report-generation failure, IPC/TUI/AXI/PR surfaces, and docs. Full validation remains `go test -race ./...`, `make lint`, and `make docs-build`.
- **Agent-Agnostic Contracts**: PASS. The report uses existing structured finding fields and a versioned Markdown contract. It does not depend on Claude, Codex, Rovo Dev, OpenCode, Pi, or ACP-specific transcript behavior.
- **Simplicity and Recovery**: PASS. The design adds one report generator package, one metadata table, and one artifact path helper. Generation errors are persisted as report metadata and do not fail the pipeline.
- **Docs and Generated Artifacts**: PASS. Docs must explain report creation, discovery, labels, latest-outcome meanings, stale/error states, and AXI/TUI/PR references. `skills/no-mistakes/SKILL.md` should be regenerated if agent-facing AXI guidance changes.

## Project Structure

### Documentation (this feature)

```text
specs/002-review-resolution-report/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   |-- review-resolution-report-markdown.md
|   `-- review-resolution-report-ipc.md
|-- source-context.md
|-- clarifications-applied-2026-06-17-145000.md
|-- red-team-findings-applied-2026-06-17-151656.md
`-- checklists/
    `-- requirements.md
```

### Source Code (repository root)

```text
internal/reviewreport/     # New report builder, sanitizer, outcome precedence, metadata/count derivation
internal/db/               # New review_resolution_reports table, persistence helpers, migrations, tests
internal/paths/            # ReportsDir and RunReportDir path helpers
internal/pipeline/         # Executor report lifecycle hooks after review evidence/decisions/final states
internal/ipc/              # ReviewResolutionReportInfo contract on run/events
internal/daemon/           # DB-to-IPC projection of report metadata
internal/cli/              # AXI status/run/respond output and success help/reference rendering
internal/tui/              # Review gate details and event state for report references/counts/outcomes
internal/pipeline/steps/   # PR summary section/reference insertion
docs/src/content/docs/     # CLI and pipeline-step documentation
skills/no-mistakes/        # Regenerated only if AXI agent instructions change
```

**Structure Decision**: Add `internal/reviewreport` as the owner of report derivation and Markdown rendering so DB, executor, daemon, CLI, TUI, and PR code do not duplicate resolution rules. Store only report metadata and summary counts in SQLite; the full durable human-readable body is the Markdown artifact. Surface code must render compact metadata directly and link to the artifact rather than duplicating sensitive-prone report content.

## Phase 0 Research

Completed in [research.md](research.md).

Key decisions:

- Use `$NM_HOME/reports/<runID>/review-resolution.md` plus a `review_resolution_reports` metadata table.
- Derive report truth from persisted `step_results` and `step_rounds`, not live IPC events, terminal logs, raw diffs, or transcripts.
- Treat `FixSummary` as an agent-reported fix attempt summary, not proof of resolution.
- Use fail-closed latest-outcome precedence and integrity checks before any "no issues remain" claim.
- Expose minimal metadata through IPC/AXI/TUI/PR surfaces: report path, status, latest outcome, stable summary counts, stale/error state, and updated timestamp.

## Phase 1 Design

Completed artifacts:

- [data-model.md](data-model.md): Entities, metadata table shape, state transitions, decision mapping, outcome precedence, validation rules.
- [contracts/review-resolution-report-markdown.md](contracts/review-resolution-report-markdown.md): Versioned Markdown headings, section order, allowed labels, summary-count keys, extraction rules.
- [contracts/review-resolution-report-ipc.md](contracts/review-resolution-report-ipc.md): IPC/run metadata contract and direct-display limits.
- [quickstart.md](quickstart.md): Implementation validation scenarios and command checklist.

## Implementation Sequence

1. Add report artifact path helpers in `internal/paths` and DB migration/helpers for `review_resolution_reports`.
2. Create `internal/reviewreport` with pure derivation tests first: parse rounds, sanitize fields, map decisions, compute counts, choose latest outcome, and render the contract headings.
3. Wire executor lifecycle hooks so the report updates after review findings, selection metadata, fix attempts, final review evidence, terminal failure/cancellation, and report-generation failures.
4. Project metadata through daemon/IPC and live events without breaking old readers.
5. Render compact references in AXI status/run/respond success output and TUI review gate details.
6. Add PR summary integration that includes counts, latest outcome, material sanitized fix summaries, and the report reference.
7. Update docs and regenerate the agent skill only if AXI success/reporting guidance changes.

## Verification Plan

Targeted tests before full validation:

- `internal/reviewreport`: one-fix run, partial selection, user instructions, user-authored findings, missing fix summary, multiple fix rounds, informational-only findings, no-reviewable-changes, invalid final findings JSON, inconsistent selected IDs/counts, cancelled/failed after fix, stale regeneration, and sanitizer rejection of code/diff/log/secret-like content.
- `internal/db`: migration idempotence, metadata persistence, stale/error status, latest successful report path, summary counts JSON.
- `internal/ipc` and `internal/daemon`: report metadata round-trip and DB projection.
- `internal/cli`: AXI status/run/respond displays report reference and counts for success, failure, incomplete, stale, and unavailable states.
- `internal/tui`: review gate detail shows report reference/counts/latest outcome and does not duplicate unsafe report fields.
- `internal/pipeline/steps`: PR summary includes report reference, latest outcome, material fix summaries, and omitted-summary counts.
- `docs`: report lifecycle, labels, latest outcomes, and discovery paths documented.

Full validation:

```sh
go test -race ./...
make lint
make docs-build
```

## Complexity Tracking

No constitution violations are planned. New storage and one internal package are justified by the durable report/reference requirement and the need to keep fail-closed report semantics centralized across DB, IPC, AXI, TUI, and PR surfaces.
