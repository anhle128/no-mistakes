# Feature Specification: Review Resolution Report

**Feature Branch**: `002-review-resolution-report`  
**Created**: 2026-06-18  
**Status**: Clarified  
**Input**: User description: "Read requirements in `plans/grill-me/review-resolution-report.md`, create a detailed spec, save the origin reference so the next phase knows the purpose, and scout source code context."

## Clarifications

### Batch Applied 2026-06-18

- **Data model**: Store report metadata in a dedicated `review_resolution_reports` SQLite table keyed by `run_id`, with path, lifecycle timestamps, resolved/accepted/informational/still-open counts, explicit status enum, and integrity fields.
- **Finding identity**: Use normalized finding ID as the primary identity. Repeated same-ID findings update the same report entry; ambiguous changed IDs remain in `Still Open Issues` instead of being coalesced by file, line, or description.
- **Report timing**: Create the report immediately after first Review findings are recorded, with unresolved items marked still open, then refresh after fix rounds and terminal Review decisions.
- **Persistence failure**: Required report or metadata write failures fail the Review step/run with an actionable error.
- **AXI/TUI display**: Show compact status, counts, and local report path only; keep the full narrative in the Markdown report.
- **PR content**: Include compact counts/status only; never publish local paths, report excerpts, or private filesystem details.
- **Evidence privacy**: Do not include raw diff hunks or transcripts; use structured resolution text, changed-file lists, fix commit SHA when available, and sanitized summaries.
- **Content limits**: Enforce per-field caps and a total full-detail report budget with explicit truncation markers. These caps apply to fix-agent `summary`, `applied_solution`, `why_this_solution`, changed-file labels, user instructions, findings, and inferred summaries, and the system MUST reject or degrade raw transcript, log, diff, or code-block content before persistence. When the full-detail budget is exceeded, every finding ID MUST still appear exactly once as a compact ID- and provenance-preserving stub.
- **No fix commit**: Store nullable `fix_commit_sha` plus a short `no_commit_reason` so no-op, failed, and missing-evidence cases are distinguishable.
- **Docs scope**: Update core Review, Auto-Fix, Pipeline, TUI, AXI, PR docs, and affected generated `/no-mistakes` skill text.

## Origin Reference *(mandatory for next phase)*

This spec is derived from [plans/grill-me/review-resolution-report.md](../../plans/grill-me/review-resolution-report.md). The source file is historical: its header says it was superseded by `specs/002-review-resolution-report/plan.md` and the implemented review-report direction. That referenced `plan.md` was not present in this checkout when this spec was written, so this spec preserves the source file's durable purpose and reconciles it with the header's shipped-direction note. The next phase must preserve the durable product purpose while rejecting historical mechanics that conflict with the header.

### Preserved Origin Purpose

- A Review step that finds issues needs a durable, human-readable explanation of every issue and what happened to it.
- The report must let a reviewer understand the problem, the solution if any, why that solution was chosen, and whether the issue is resolved, accepted without a fix, or still open.
- The original reviewability gap is that rich pre-fix review findings exist, while post-fix data is only a short commit-oriented summary.
- Source anchors: `plans/grill-me/review-resolution-report.md` lines 13-44.

### Reconciled Product Direction

- The durable Markdown artifact belongs under `$NM_HOME/reports/<runID>/review-resolution.md`.
- Compact report metadata belongs in SQLite so AXI, TUI, and PR summary rendering can reference the report or its status without reparsing logs.
- The feature must not commit repo-local `no-mistakes/<branch-slug>/review-resolution.md` artifacts, force-add ignored report files, or require one review finding per fix commit.
- Source anchors: `plans/grill-me/review-resolution-report.md` lines 3-11.

### Historical Requirements Not Carried Forward

- Repo-local committed report path and branch-slug path rules from lines 55-89.
- One issue, one commit, amend-on-retry, and per-issue auto-fix budget requirements from lines 144-205.
- Same-commit report update and force-add behavior from lines 271-329 and 396-399.
- GitHub blob links to committed report files from lines 343-359.

### Requirements Carried Forward

- Stable Markdown structure with run context and per-issue sections from lines 91-130, including a report format version marker, fixed heading hierarchy, exact field labels, outcome section order, and count semantics covered by golden-file tests.
- Include every Review issue and separate resolved, accepted, informational, and still-open outcomes from lines 132-142 and 304-319.
- Prefer structured fix-agent resolution details, fallback honestly to inferred-from-diff detail, and label the source from lines 206-256.
- Persist enough metadata to avoid guessing the relevant fix from later git history from lines 258-269 and 361-372.
- Default behavior: create the report when Review findings exist, omit it for clean review from lines 331-341.
- Minimum coverage expectations for report generation and PR/summary surfacing from lines 388-404, adjusted for the reconciled `$NM_HOME` artifact model.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read Review Outcomes After a Gate Run (Priority: P1)

As a developer who pushed through `git push no-mistakes`, I want a durable review-resolution report when the Review step found issues, so I can understand what was found and how each issue ended without reconstructing the pipeline from logs.

**Why this priority**: This is the core reviewability gap from the source report. Without this, Review auto-fixes and approvals remain hard to audit.

**Independent Test**: Run a gate scenario where Review returns two findings, fix one, and approve the other. The run produces `$NM_HOME/reports/<runID>/review-resolution.md` with both findings in separate outcome sections.

**Acceptance Scenarios**:

1. **Given** a Review step emits one fixable finding, **When** the finding is fixed and a comparable follow-up Review round no longer reports the same normalized finding ID, **Then** the report contains the finding under `Resolved Issues` with problem, applied solution, rationale, fix source, verification text, follow-up round ID, scope-equivalence note, and verifier source.
2. **Given** a Review step emits a finding that the user approves as-is, **When** the Review step completes, **Then** the report contains that finding under `Accepted Without Fix` with the acceptance action, actor/source, and approval reason available to the system. If `skip` has materially different semantics from approval in the persisted decision record, the entry MUST show that skip action distinctly instead of collapsing it into generic accepted wording.
3. **Given** a Review step emits findings, **When** the run is later inspected through TUI or AXI, **Then** those surfaces expose that a review-resolution report exists and provide the local report path or compact status.

---

### User Story 2 - Preserve Honest Status for Incomplete Runs (Priority: P1)

As a developer reviewing a failed, aborted, or superseded gate run, I want the report to avoid calling unresolved findings accepted, so I can trust the artifact during interrupted workflows.

**Why this priority**: The source explicitly warns that accepted/unfixed status must be based on a real user or pipeline decision, not inferred after failure.

**Independent Test**: Run a Review scenario with two findings, fix one, then abort before approving the remaining finding. The report marks the fixed finding as resolved and the remaining finding as still open.

**Acceptance Scenarios**:

1. **Given** Review findings remain when the user aborts the step, **When** the report is generated or refreshed, **Then** those remaining findings appear under `Still Open Issues`.
2. **Given** the pipeline fails after Review started but before remaining findings are accepted, **When** the report is inspected, **Then** no remaining finding is listed under `Accepted Without Fix` solely because the run stopped.
3. **Given** a fix attempt produces no commit or no parseable resolution detail, **When** the report renders the issue, **Then** it states the missing evidence rather than claiming a verified fix.

---

### User Story 3 - See Applied Solution Details Beyond Commit Summary (Priority: P2)

As a PR reviewer or maintainer, I want the report to show what the fix agent actually changed and why, so a one-line commit summary is not the only record of the resolution.

**Why this priority**: This turns the source report's desired issue-to-solution explanation into a durable artifact while staying compatible with existing round summaries.

**Independent Test**: Use a fake review fix agent that returns structured `resolutions[]`; verify the report uses those fields. Use another fake agent that returns only `summary`; verify the report uses inferred text with an explicit fallback source label.

**Acceptance Scenarios**:

1. **Given** the review fix agent returns a `resolutions[]` entry matching a selected finding ID, **When** the report is generated, **Then** the issue shows `Applied Solution Source: fix agent structured output`.
2. **Given** the review fix agent omits `resolutions[]` or omits a selected finding ID, **When** the report is generated, **Then** the issue shows `Applied Solution Source: inferred from fix commit diff because structured resolution details were unavailable`.
3. **Given** a resolution entry includes changed files, **When** the report renders the issue, **Then** those changed files are shown as part of the applied-solution evidence.

---

### User Story 4 - Reference Review Resolution From PR Summary (Priority: P3)

As a reviewer reading the generated PR body, I want a concise indication that Review findings were resolved, accepted, or left open, so the PR summary points me to the resolution evidence without exposing local-only file links as if they were public URLs.

**Why this priority**: PR summaries already render pipeline narratives from round data; this story connects the new report metadata to that user-facing surface.

**Independent Test**: Build PR content for runs with and without a review-resolution report. The PR body includes review-resolution status only when the report exists, and never emits a broken GitHub blob link for a local `$NM_HOME` report.

**Acceptance Scenarios**:

1. **Given** a review-resolution report exists, **When** PR content is generated, **Then** the `## Pipeline` section includes compact review-resolution counts or status from metadata.
2. **Given** no Review findings occurred, **When** PR content is generated, **Then** no review-resolution report status or link is shown.
3. **Given** the report has only a local filesystem path, **When** PR content is generated, **Then** the PR body does not publish that local path as a public link.

### Edge Cases

- Clean Review step: no report file and no report metadata should be created.
- Review step skipped before producing findings: no report file should be created unless prior Review findings for that run already exist.
- User selects a subset of findings for fix: selected findings can be resolved or still open; unselected findings become accepted only after an explicit user approval/skip decision or documented pipeline-policy acceptance completes the Review step. Configured skipped steps, automated convergence approvals, canceled runs, superseded runs, and deferred/unselected findings without such a persisted terminal decision MUST NOT be rendered as accepted.
- Multiple selected findings fixed in one round: all selected findings may share one fix round and commit unless richer metadata distinguishes them; the feature must not require one finding per commit.
- Follow-up Review returns changed finding IDs: exact ID matching is preferred; ambiguous matches must be labeled verification-inconclusive or still open instead of silently resolved.
- Repeated Review output uses the same normalized finding ID: the report updates one entry for that ID rather than adding duplicate sections.
- User-authored findings from TUI or AXI: include them as Review issues with source `user` when they are dispatched through the Review fix flow, and label any user reason or instruction in the report as explicit approval input, explicit AXI/TUI finding input, explicit per-finding instruction, or unavailable. The report MUST NOT infer approval reasons or user intent summaries from raw transcripts unless the text is stored only as a sanitized bounded summary with its source label.
- Fix agent returns invalid JSON, no `resolutions[]`, no changed files, or no commit: report must preserve the issue, label missing or inferred evidence honestly, and store `no_commit_reason` when no fix commit exists.
- Report content exceeds configured field limits or the full-detail report budget: the report must keep the item, truncate or compact with explicit markers, and avoid malformed Markdown.
- `$NM_HOME/reports/<runID>/` cannot be written or report metadata cannot be persisted: the Review step/run must fail with an actionable error; it must not silently claim report generation succeeded.
- Existing SQLite databases: migrations must be additive and must preserve prior runs and round history.
- Non-GitHub remotes, skipped PR creation, or unauthenticated provider CLI: report generation still works; PR-specific status is omitted when no PR body is generated.
- Superseded or canceled runs: generated report metadata remains tied to the run ID and must not be overwritten by a newer run.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create a durable Markdown review-resolution report at `$NM_HOME/reports/<runID>/review-resolution.md` immediately after the Review step first records one or more findings.
- **FR-002**: System MUST NOT create, stage, force-add, or commit a repo-local `no-mistakes/<branch-slug>/review-resolution.md` report file for this feature.
- **FR-003**: System MUST omit the review-resolution report and report metadata when the Review step completes with no findings across all Review rounds.
- **FR-004**: System MUST include a `Run Context` section with run ID, repo identifier or path, branch, base commit, current/final head commit at generation time, Review step status, report lifecycle state, first generated timestamp, last refreshed timestamp, finalized timestamp when applicable, and local report path.
- **FR-005**: System MUST include every normalized Review issue known for the run, including agent-produced findings selected or unselected by the user and user-authored findings dispatched through Review fix flow.
- **FR-006**: System MUST render each Review issue as its own section and MUST NOT collapse multiple findings into a single broad summary.
- **FR-007**: System MUST classify report entries into `Resolved Issues`, `Accepted Without Fix`, `Still Open Issues`, and `Informational / No Action Required`. Informational entries are limited to Review findings whose effective action is `no-op` and MUST NOT be counted as resolved fixes or accepted risks.
- **FR-008**: System MUST classify an issue as `Resolved` only when a valid parsed follow-up Review result over an equivalent Review scope has comparable normalized IDs and shows that the same finding is no longer present, or when a persisted verification event explicitly records equivalent human/pipeline confirmation for that finding. Clean Review completion without comparable parsed follow-up evidence is verification-inconclusive and MUST leave the issue in `Still Open Issues`.
- **FR-009**: System MUST classify an issue as `Accepted Without Fix` only after a persisted Review terminal-decision record or equivalent round metadata proves that a user or documented pipeline policy approved, skipped, or otherwise accepted the issue without fixing it. That record MUST include action, actor/source, timestamp, affected finding IDs, and reason when available; if it is absent during regeneration, unresolved findings MUST stay `Still Open` with acceptance evidence unavailable.
- **FR-010**: System MUST classify an issue as `Still Open` when it remains after attempted fixes, when verification is inconclusive, or when the run fails, aborts, cancels, or is superseded before acceptance.
- **FR-011**: System MUST preserve the original finding details available in Review output, including ID, severity, file, line, action, source, description, context, suggested or proposed fix, risk level, risk rationale, and user instructions when present. When historical or legacy findings lack context or suggested/proposed fix fields, the report MUST render an explicit `not recorded` or `unavailable in historical data` marker rather than silently omitting the field.
- **FR-012**: System MUST extend the Review fix agent structured response contract to allow a short `summary` plus optional `resolutions[]` entries containing `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`.
  - When `resolutions[]` is present, the system MUST validate it as untrusted structured data: each entry must have a non-empty `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`; `finding_id` values must be unique; unknown or duplicate IDs must not be used as finding-specific evidence; and missing selected-finding coverage must be recorded as degraded structured evidence rather than silently treated as a complete agent report.
- **FR-013**: System MUST continue accepting legacy fix-agent responses that only contain `summary`.
- **FR-014**: System MUST prefer matching validated `resolutions[]` data for applied solution and rationale when a resolution entry maps to the finding ID, but only as descriptive fix-agent evidence. The report MUST render those fields as applied-solution language only when the entry is tied to persisted fix-round evidence and a resolved or verified-attempt outcome; otherwise it MUST label the entry as attempted, not applied, or evidence unavailable as appropriate.
- **FR-015**: System MUST infer applied-solution details from the relevant fix round and commit diff when structured resolution details are missing, and MUST label that fallback source explicitly. Diff-derived fallback text MUST be a high-level sanitized summary supported by changed-file paths and fix commit SHA when available; it MUST NOT include raw hunks, code snippets, secret-bearing literals, or near-verbatim code excerpts.
- **FR-016**: System MUST NOT present inferred applied-solution text as if it came from the fix agent's structured output.
- **FR-017**: System MUST persist compact report metadata in a dedicated SQLite `review_resolution_reports` table keyed by `run_id`, sufficient for AXI, TUI, and PR summary code to know whether a report exists, where it is located locally, when it was generated, its surface-facing status, and how many issues are resolved, accepted, informational, and still open.
  - The metadata status MUST use an explicit enum with transition rules covering at least `in_progress`, `final`, `incomplete`, `stale`, `degraded`, and `evidence_unavailable`; AXI, TUI, and PR summaries MUST render this enum rather than deriving their own success wording from counts alone.
- **FR-018**: System MUST persist nullable `fix_commit_sha` and `no_commit_reason` evidence for Review fix rounds, so later Document, Lint, Push, or CI commits do not become inferred Review fix commits and no-commit cases remain distinguishable.
- **FR-019**: System MUST support multiple findings sharing one fix round or commit when the existing fix flow handles selected findings together. When structured `resolutions[]` data is absent and one commit or fix round covers multiple findings, inferred evidence MUST be labeled commit-level or round-level evidence, not finding-specific applied-solution detail, unless exact verification evidence supports the per-finding mapping.
- **FR-020**: System MUST update or refresh the report when Review findings are first recorded, after Review fix rounds, when the Review step reaches approve/skip/abort/failure outcomes, when a run with Review findings becomes failed, cancelled, superseded, or stale-recovered, and before PR summary generation uses report metadata. If terminal Review evidence cannot be reconstructed during reconciliation, unresolved findings MUST remain `Still Open` and the report status MUST indicate incomplete or evidence-unavailable state.
- **FR-021**: System MUST keep report generation scoped to the Review step for this version and MUST NOT add all-step evidence reports.
- **FR-022**: System MUST expose compact report status, issue counts, and local report path through AXI and TUI run detail surfaces when a report exists, and MUST NOT inline the full report narrative in those surfaces. Surface wording MUST be derived from the persisted report status and counts; any nonzero `still_open_count` or evidence-unavailable status MUST use non-success language.
- **FR-023**: System MUST update generated PR content to reference compact review-resolution counts/status from metadata when a report exists, without publishing local-only filesystem paths, report excerpts, or private filesystem details.
- **FR-024**: System MUST sanitize and bound report content sourced from agent output, user instructions, findings, diffs, and commit metadata to avoid malformed Markdown control content, accidental raw transcript storage, raw logs, raw code blocks, raw diff hunks, common secret formats, and unbounded report size; truncation MUST use explicit markers.
  - Sanitization MUST be field-level: preserve IDs, severity, action, source, file, and line as structured fields; redact secret-like values; escape Markdown controls; strip or summarize raw transcript/log/diff/code blocks; and store bounded summaries for descriptions, risk rationale, user instructions, applied solution, and reasons.
  - The report-level size limit is a full-detail budget: once exceeded, later entries MUST render compact stubs that preserve finding ID, outcome, and classification provenance rather than omitting entries.
- **FR-025**: System MUST fail the Review step/run with an actionable error when required report writing or metadata persistence fails for a run with Review findings. Report Markdown and `review_resolution_reports` metadata refresh MUST be atomic from the perspective of consumers; metadata MUST include integrity fields such as report version, entry count, source round range or watermark, and last refresh result, and surfaces MUST show degraded or evidence-unavailable status instead of confident counts when consistency checks fail.
- **FR-026**: System MUST update core Review, Auto-Fix, Pipeline, TUI, AXI, PR documentation, and affected generated `/no-mistakes` skill text when user-visible report behavior, AXI/TUI output, or PR summary content changes.

### Key Entities

- **Review Resolution Report**: Local durable Markdown artifact for one run, stored at `$NM_HOME/reports/<runID>/review-resolution.md`.
- **Review Resolution Metadata**: Compact SQLite record in `review_resolution_reports`, keyed by `run_id`, for report path, lifecycle timestamps, issue counts, status enum, source watermark, integrity fields, and last refresh result. It lets UI and PR code reference the report without reparsing the Markdown.
- **Resolution Entry**: One report section for one normalized Review issue, including original finding fields, outcome category, applied solution, rationale, verification, source labels, changed files, fix commit when known, no-commit reason when no fix commit exists, and outcome-decision provenance. Provenance MUST include decision actor/source, action, timestamp when available, Review round ID, selection source, and the evidence reference used to classify the entry.
- **Fix Resolution Detail**: Optional structured fix-agent output item keyed by finding ID, used as the preferred applied-solution source.
- **Review Round**: Existing step-round data for Review executions, including findings, selected finding IDs, selection source, user findings payload, fix summary, and any added fix commit or no-commit metadata.
- **Report Reference**: Surface-specific pointer to the report or its status. AXI/TUI show compact status, counts, and local path; PR summaries show compact counts/status and avoid local filesystem links or report excerpts.

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: The feature preserves the fixed gate order and does not change what it means for a branch to pass. It adds Review evidence after findings occur and does not alter `origin` behavior or push semantics.
- **Isolation/User Control**: Intentional writes go under `$NM_HOME/reports/<runID>/` and SQLite state, not the user's day-to-day worktree. The report reflects user approval, skip, fix, and abort decisions without inventing acceptance.
- **Evidence Plan**: Implementation should include focused unit tests for report generation, metadata migration, Review fix output parsing, PR summary rendering, and failure/abort classification. Standard Go validation remains `gofmt`, targeted `go test`, `go test -race ./...`, and `make lint` unless the implementation plan records a temporary exception.
- **Agent/Interface Contracts**: The Review fix schema remains backward-compatible with `summary` and gains optional validated `resolutions[]`. AXI/TUI labels should use user-facing terms: resolved, accepted without fix, informational, still open, evidence unavailable, report path.
- **Docs/Generated Artifacts**: Update docs that describe Review, Auto-Fix Loop, Pipeline, AXI output, TUI behavior, PR summaries, local state paths, and any generated `/no-mistakes` skill text affected by the new report.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In tests with Review findings, 100% of normalized finding IDs from Review rounds appear exactly once in the report; repeated same-ID sightings update the entry, and ambiguous changed IDs remain in `Still Open Issues` instead of being coalesced heuristically.
- **SC-002**: In tests with clean Review output, no report file and no report metadata are created.
- **SC-003**: In tests with structured fix-agent `resolutions[]`, the report uses the structured applied solution and rationale for the matching finding IDs.
- **SC-004**: In tests without structured resolution details, the report includes an inferred-source label and never claims the text came from structured agent output.
- **SC-005**: In tests where a run aborts or fails with unresolved Review findings, 100% of unresolved findings appear under `Still Open Issues`.
- **SC-006**: In tests for existing databases, additive migration preserves old runs and permits the dedicated `review_resolution_reports` table to be added without rebuilding the DB.
- **SC-007**: PR body generation tests show compact review-resolution counts/status when metadata exists, omit the section when no report exists, and never include local paths or report excerpts. Integration tests MUST compare the Markdown section counts, SQLite metadata counts/status, AXI/TUI run-detail output, and PR summary output for the same run to prove they come from the same persisted classified snapshot.
- **SC-008**: In tests where report or metadata persistence fails after Review findings exist, the Review step/run fails with an actionable error.
- **SC-009**: In tests with oversized finding, agent, or diff-derived content, the report remains valid Markdown, uses explicit truncation or compaction markers, and preserves every finding ID exactly once.

## Assumptions

- `$NM_HOME` is the durable local state boundary for this artifact; the report is not intended to be committed to the target repository in this reconciled version.
- PR reviewers benefit from compact report status in the PR body even when the full local report is not publicly linkable.
- Existing persisted Review step rounds are the authoritative regeneration source for report generation; the feature should add decision, fix, and report metadata only where the current round data is insufficient. Regeneration MUST rebuild from persisted Review rounds plus persisted decision/fix metadata, not transient events or latest findings alone, and metadata MUST store a source-round watermark or content hash so stale Markdown/SQLite mismatches are detectable.

## Source Context For Planning *(non-requirement reference)*

The following source files were scouted while writing this spec and should be revisited during `/speckit-plan`:

- `internal/pipeline/executor.go`: inserts step rounds, records selected finding IDs, handles approval actions, fix loops, and run/step finalization.
- `internal/pipeline/pipeline.go`: defines `StepContext` and `StepOutcome`, including `FixSummary`.
- `internal/pipeline/steps/review.go`: Review prompt, Review fix prompt, current `summary`-only fix schema usage, follow-up Review execution.
- `internal/pipeline/steps/common_fix.go`: `executeFixMode`, `commitAgentFixes`, and commit summary parsing.
- `internal/db/round.go`, `internal/db/step.go`, `internal/db/run.go`, and `internal/db/schema.go`: run, step, step-round persistence and additive migrations.
- `internal/paths/paths.go`: current `$NM_HOME` directories; no report directory helper exists yet.
- `internal/pipeline/steps/pr.go` and `internal/pipeline/steps/prsummary.go`: PR content generation and deterministic pipeline summary rendering from rounds.
- `internal/types/findings.go`: finding identity, action, source, user instructions, and JSON compatibility helpers.
- `internal/ipc/protocol.go`, `internal/ipc/server.go`, `internal/tui/app.go`, and `internal/cli/axi_render.go`: daemon-to-UI/AXI contracts likely affected by report metadata surfacing.
- `internal/daemon/manager.go`: run lifecycle and superseded/canceled-run behavior.
- `docs/src/content/docs/reference/pipeline-steps.md`, `docs/src/content/docs/concepts/auto-fix.md`, and `docs/src/content/docs/concepts/gate-model.md`: user-visible behavior that will need updates.
- `Makefile`: canonical verification targets include `make build`, `make test`, `make lint`, `make e2e`, `make skill`, and docs targets.
