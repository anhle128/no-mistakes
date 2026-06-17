# Review Resolution Report Source Context

**Created**: 2026-06-17  
**Feature**: `review-resolution-report`  
**Purpose**: Preserve the source-backed origin and implementation context behind `spec.md` so future `/speckit-plan`, `/speckit-tasks`, and implementation work can start from known facts.

## Origin

The user requested `/speckit-specify` for `review-resolution-report` with explicit instructions to:

- read the requirement and create a detailed spec;
- save a reference so future tasks know the origin and purpose;
- spawn dedicated sub-agents for help;
- scout the source code to understand context before writing the spec.

This file is the saved source-context reference. The product-facing requirements remain in `spec.md`.

## Current Product Context

The feature exists because review-resolution details are currently distributed across multiple surfaces:

- review findings and suggested fixes;
- user selection and per-finding instructions;
- review fix rounds and fix summaries;
- live IPC events;
- TUI approval and fix-review rendering;
- AXI status, drive, respond, and success output;
- PR summaries and pipeline docs.

Earlier gotchas in `brain/Gotchas.md` warn that user-visible review data must be traced end-to-end. In particular, proposed fixes (`suggested_fix`) are not the same as applied fixes (`fix_summaries`), and the exact screen or surface matters.

## Source Facts

- `internal/types/findings.go`: Defines `Finding`, `Findings`, finding actions, user-authored findings, user instructions, merge/filter helpers, and actionable-finding semantics.
- `internal/types/types.go`: Defines review step identity, step statuses such as `awaiting_approval`, `fixing`, and `fix_review`, plus approval actions.
- `internal/pipeline/pipeline.go`: Defines `StepOutcome`, including `Findings` and `FixSummary`.
- `internal/pipeline/steps/review.go`: Runs the review step, prompts for structured review findings, enters fix mode, and returns fix summaries.
- `internal/pipeline/executor.go`: Normalizes findings, persists rounds, records selected finding IDs and user findings, emits findings/diff/fix-summary events, and manages approval/fix loops.
- `internal/db/round.go`: Stores per-step execution rounds, selected finding IDs, selection source, user findings, and fix summaries.
- `internal/db/stats.go`: Computes reported and fixed finding counts from step rounds.
- `internal/ipc/protocol.go`: Exposes run/step info and live events with findings, diff, reported/fixed counts, and fix summaries.
- `internal/daemon/daemon.go`: Projects stored run and step data into IPC `RunInfo` and `StepResultInfo`.
- `internal/daemon/manager.go`: Routes user responses with selected finding IDs, instructions, and added findings to the active executor.
- `internal/tui/events.go`: Applies live IPC events to the TUI model, including reported/fixed counts and fix summaries.
- `internal/tui/pipeline.go`: Renders approval action bar and fix-review applied-fix summary.
- `internal/tui/review.go`: Renders review findings and labels `suggested_fix` as `Solution`.
- `internal/cli/axi_render.go`: Renders run/gate output and flattens fix summaries into `fixes`.
- `internal/cli/axi_drive.go`: Drives headless runs, resolves gates, and instructs agents to report applied fixes after successful runs.
- `docs/src/content/docs/reference/pipeline-steps.md`: Documents review findings, approval semantics, auto-fix history, and fix commit semantics.
- `docs/src/content/docs/reference/cli.md`: Documents AXI command behavior and agent-facing workflow.
- `plans/grill-me/review-file-handoff.md`: Prior plan for moving review issue handling into a Markdown handoff file and preserving review responses.
- `plans/grill-me/review-subphase-presentation-labels.md`: Prior plan for clearer review sub-phase labels without changing backend behavior.

## Data Flow To Preserve

1. Review step produces structured findings and optional risk assessment.
2. Executor normalizes findings and stores the round.
3. If approval is required, user or automation selects findings and may add instructions or user-authored findings.
4. Executor records selected finding IDs, selection source, and merged user findings.
5. Fix mode runs and returns a one-line fix summary.
6. Follow-up review produces the latest findings and risk state.
7. Stored rounds and step stats are projected into IPC events and run info.
8. TUI, AXI, and PR-facing surfaces render the latest state and applied fixes.

The report should be derived from this flow and must not require raw terminal output, raw logs, or raw transcript text.

## Defaults Chosen For This Spec

- Report format: generated Markdown artifact.
- Scope: review step only.
- Creation trigger: review findings, review fixes, approval gates, or no-reviewable-changes review outcomes.
- Surface exposure: existing TUI/AXI/PR-facing surfaces may include or reference the report; no new workflow is required by the spec.
- Persistence rule: report should remain available even if later steps fail, the run is cancelled, or the live event was missed.
- Privacy rule: safe intent summaries are allowed; raw logs and raw transcripts are out of scope.

## Sub-Agent Contributions

Two dedicated sub-agents were spawned during specification:

- Source scout: mapped source files and confirmed that `step_rounds`, finding stats, fix summaries, IPC projections, TUI events, and AXI renderers are the likely source-of-truth path.
- Requirements analyst: drafted candidate user stories, requirements, edge cases, success criteria, and risks. Open questions were resolved into the defaults above to avoid blocking the spec.

## Planning Warnings

- Do not conflate `suggested_fix` / `Recommendation` with an applied fix. Applied fixes come from fix summaries and the follow-up review state.
- Do not claim that all review issues are resolved unless the latest review evidence proves it.
- Do not add a new review step or alter raw step statuses unless a later plan explicitly justifies that scope expansion.
- Do not expose raw transcripts or raw logs in the report.
- Keep review reporting labels user-facing; avoid leaking internal field names into the user surface.

## Implementation Notes

- `internal/reviewreport` owns derivation, sanitization, latest-outcome classification, Markdown rendering, artifact writing, and metadata persistence for the durable report.
- `internal/db/review_resolution_report.go` stores one metadata row per run in `review_resolution_reports`; the Markdown artifact remains the detailed human/agent reference.
- `internal/paths` resolves the report location as `$NM_HOME/reports/<runID>/review-resolution.md`.
- `internal/pipeline/review_report.go` updates the report after review evidence, selections, fix attempts, terminal review states, and run completion/failure. Update failures are persisted as safe report metadata and logged without changing pipeline outcome.
- `internal/ipc.ReviewResolutionReportInfo` is the compact display contract for daemon, executor events, TUI, AXI, and PR surfaces. It must not carry the full report body, raw logs, raw transcripts, raw diff hunks, code excerpts, or raw user instructions.
- `internal/tui/pipeline.go` renders only the report path/reference, status, latest outcome, selected count highlights, and stale/error state. Detailed finding context remains in the sanitized Markdown report.
