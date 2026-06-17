# Tasks: Review Resolution Report

**Input**: Design documents from `specs/002-review-resolution-report/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`, and `source-context.md`
**Tests**: Required for code changes. Add targeted Go tests before implementation, then run focused package tests and full validation.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested as an independent increment after the foundational report metadata work.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other marked tasks because it touches different files or only reads evidence
- **[Story]**: User story label from `spec.md`
- Every task includes exact repository paths

## Phase 1: Setup (Shared Preparation)

**Purpose**: Confirm the implementation boundaries, source context, and validation contract before code changes.

- [ ] T001 Confirm the feature requirements and non-goals in specs/002-review-resolution-report/spec.md and specs/002-review-resolution-report/plan.md
- [ ] T002 Trace the current review-resolution data flow from specs/002-review-resolution-report/source-context.md into internal/pipeline/executor.go
- [ ] T003 [P] Confirm user-facing surface files from specs/002-review-resolution-report/source-context.md against internal/cli/axi_render.go, internal/tui/pipeline.go, and internal/pipeline/steps/prsummary.go
- [ ] T004 [P] Confirm no-new-dependency and validation constraints in specs/002-review-resolution-report/research.md and specs/002-review-resolution-report/quickstart.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared path, storage, metadata, and IPC contracts that all user stories depend on.

**Critical**: No user story implementation should start until these shared contracts compile and have targeted tests.

- [ ] T005 [P] Add report artifact path helper tests in internal/paths/paths_test.go
- [ ] T006 [P] Add review_resolution_reports migration and metadata persistence tests in internal/db/review_resolution_report_test.go
- [ ] T007 [P] Add ReviewResolutionReportInfo JSON compatibility tests in internal/ipc/protocol_test.go
- [ ] T008 [P] Add daemon run-info report metadata projection tests in internal/daemon/runinfo_test.go
- [ ] T009 Implement ReportsDir and RunReviewResolutionReportPath helpers in internal/paths/paths.go
- [ ] T010 Add the review_resolution_reports schema migration in internal/db/schema.go
- [ ] T011 Implement review resolution report metadata persistence helpers in internal/db/review_resolution_report.go
- [ ] T012 Add ReviewResolutionReportInfo to ipc.RunInfo and ipc.Event in internal/ipc/protocol.go
- [ ] T013 Project persisted report metadata into run info and run/review update events in internal/daemon/daemon.go
- [ ] T014 Add canonical report status, latest-outcome, and summary-count constants in internal/reviewreport/types.go

**Checkpoint**: Report metadata can be stored, loaded, projected through IPC, and represented with stable status/outcome/count keys.

---

## Phase 3: User Story 1 - Understand Review Resolution (Priority: P1) MVP

**Goal**: Generate one durable Markdown report per reportable review run that explains findings, decisions, fix attempts, applied fix summaries, latest trustworthy outcome, and remaining risks.

**Independent Test**: Run or fixture a review with at least one actionable finding selected for fix, then verify the report shows the original issue, selected action, applied fix summary, and latest review outcome without raw logs, transcripts, code excerpts, or diff hunks.

### Tests for User Story 1

- [ ] T015 [P] [US1] Add Markdown contract heading, label, and summary-count rendering tests in internal/reviewreport/render_test.go
- [ ] T016 [P] [US1] Add decision mapping, summary count, and latest-outcome precedence tests in internal/reviewreport/derive_test.go, including no-reviewable-changes, failed-after-fix, cancelled-after-fix, and superseded-after-fix latest-outcome cases
- [ ] T017 [P] [US1] Add sanitizer tests for diff-like, code-like, log-like, transcript-like, and secret-like values in internal/reviewreport/sanitize_test.go
- [ ] T018 [P] [US1] Add executor lifecycle tests for review findings, selected fixes, fix summaries, final review evidence, cancellation, and generation failure in internal/pipeline/executor_review_report_test.go
- [ ] T019 [P] [US1] Add TUI review gate report metadata rendering tests in internal/tui/pipeline_test.go

### Implementation for User Story 1

- [ ] T020 [US1] Implement report snapshot, finding, decision, fix attempt, and source evidence types in internal/reviewreport/types.go
- [ ] T021 [US1] Implement persisted review round loading and normalization from step_results and step_rounds in internal/reviewreport/rounds.go
- [ ] T022 [US1] Implement resolution decision mapping and canonical summary-count derivation in internal/reviewreport/derive.go, including an Accepted gate: emit `Accepted` only from explicit stored human/user risk-acceptance evidence; generic approve, skip, or unselected data must map to another evidence-backed state such as `Skipped`, `Still open`, `Decision not recorded`, or `Unavailable`
- [ ] T023 [US1] Implement deterministic latest-outcome precedence in internal/reviewreport/outcome.go
- [ ] T024 [US1] Implement allowlist-based report sanitization in internal/reviewreport/sanitize.go
- [ ] T025 [US1] Implement Markdown rendering with the exact contract headings from specs/002-review-resolution-report/contracts/review-resolution-report-markdown.md in internal/reviewreport/render.go
- [ ] T026 [US1] Implement report writing, metadata update, stale detection, and regeneration guard behavior in internal/reviewreport/report.go
- [ ] T027 [US1] Implement pipeline report lifecycle hooks for review evidence, decisions, fix attempts, final states, and generation errors in internal/pipeline/review_report.go
- [ ] T028 [US1] Call the report lifecycle hooks from the existing approval and fix loop boundaries in internal/pipeline/executor.go
- [ ] T029 [US1] Render report path, status, latest outcome, and summary counts in TUI review gate details in internal/tui/pipeline.go
- [ ] T030 [US1] Apply report metadata from IPC events into the TUI model in internal/tui/events.go
- [ ] T031 [US1] Enforce that report snapshots exclude raw diff, log, transcript, and code excerpt sources in internal/reviewreport/report.go

**Checkpoint**: User Story 1 is complete when a durable report can be generated from stored review data, viewed from the report path, and referenced by the TUI review gate without changing review behavior.

---

## Phase 4: User Story 2 - Preserve Origin For Future Work (Priority: P1)

**Goal**: Ensure future agents and maintainers can understand why the report exists, what evidence it uses, and which fields are unavailable or not recorded without reading raw logs.

**Independent Test**: Open the feature directory and a generated report, then verify the purpose, non-goals, source evidence, safe intent summary, and unavailable/not-recorded labels explain the review-resolution story without relying on raw logs or transcripts.

### Tests for User Story 2

- [ ] T032 [P] [US2] Add purpose, non-goal, source evidence, safe intent, unavailable, and not-recorded rendering tests in internal/reviewreport/origin_test.go
- [ ] T033 [P] [US2] Add legacy partial-data report tests for missing selected IDs, user instructions, fix summaries, severity, location, issue title, context, recommendation, action type, source, risk level, and risk rationale in internal/reviewreport/legacy_test.go; assert no inference of decision, source, action type, risk, or resolution category from adjacent fields

### Implementation for User Story 2

- [ ] T034 [US2] Render the report purpose statement and behavior-preservation non-goals in internal/reviewreport/render.go
- [ ] T035 [US2] Render source evidence labels, integrity status, generation notes, unavailable values, and not-recorded values in internal/reviewreport/render.go
- [ ] T036 [US2] Derive a sanitized safe intent summary from stored run data without using raw transcripts or logs in internal/reviewreport/report.go
- [ ] T037 [US2] Update report lifecycle, discovery, latest-outcome labels, and resolution labels in docs/src/content/docs/reference/pipeline-steps.md
- [ ] T038 [US2] Update CLI report reference behavior and status/count terminology in docs/src/content/docs/reference/cli.md
- [ ] T039 [US2] Append implementation notes that preserve origin and source-context assumptions in specs/002-review-resolution-report/source-context.md

**Checkpoint**: User Story 2 is complete when the report and docs preserve the origin, purpose, evidence boundaries, and missing-data semantics for future work.

---

## Phase 5: User Story 3 - Report Successful Runs Without Hiding Misses (Priority: P2)

**Goal**: Ensure AXI status, run, drive, and respond output references review-resolution details whenever review fixes or non-empty report metadata exist, including successful runs.

**Independent Test**: Drive a run with review findings and a fix summary, then verify successful AXI output includes the report reference, latest outcome, status, and stable summary counts.

### Tests for User Story 3

- [ ] T040 [P] [US3] Add AXI renderer tests for current, stale, unavailable, error, incomplete, and successful report metadata in internal/cli/axi_render_report_test.go
- [ ] T041 [P] [US3] Add AXI drive success-output tests for runs with report metadata and applied fixes in internal/cli/axi_drive_test.go
- [ ] T042 [P] [US3] Add AXI status/query tests for report reference, latest outcome, and count display in internal/cli/axi_test.go

### Implementation for User Story 3

- [ ] T043 [US3] Render report path, status, latest outcome, updated timestamp, stale/error state, and summary counts in internal/cli/axi_render.go
- [ ] T044 [US3] Include report metadata in AXI status and query command output paths in internal/cli/axi_query.go
- [ ] T045 [US3] Include report references in successful AXI drive and respond completion output in internal/cli/axi_drive.go
- [ ] T046 [US3] Update agent-facing success/status guidance if behavior changes in skills/no-mistakes/SKILL.md

**Checkpoint**: User Story 3 is complete when agent-facing command output no longer hides review fixes or incomplete/unavailable review outcomes on successful runs.

---

## Phase 6: User Story 4 - Support PR Review Context (Priority: P2)

**Goal**: Add concise PR-facing review-resolution context with summary counts, latest outcome, material sanitized applied-fix summaries, omitted-summary counts, and the durable report reference.

**Independent Test**: Generate a PR summary after one or more review fix rounds and verify it includes concise review-resolution context without raw report body, logs, transcripts, code excerpts, diff hunks, or unsafe summary text.

### Tests for User Story 4

- [ ] T047 [P] [US4] Add PR summary tests for report reference, counts, latest outcome, material applied-fix summaries, and omitted-summary count in internal/pipeline/steps/prsummary_report_test.go
- [ ] T048 [P] [US4] Add PR summary tests for stale, unavailable, error, multiple-fix-round, and unsafe-summary cases in internal/pipeline/steps/prsummary_report_errors_test.go

### Implementation for User Story 4

- [ ] T049 [US4] Extend PR summary input models to accept report metadata and material fix-summary data in internal/pipeline/steps/prsummary.go
- [ ] T050 [US4] Render review-resolution report reference, counts, latest outcome, material summaries, and omitted-summary count in internal/pipeline/steps/prsummary.go
- [ ] T051 [US4] Wire persisted report metadata into PR summary generation in internal/pipeline/steps/pr.go
- [ ] T052 [US4] Enforce PR summary display limits so raw report body, logs, transcripts, code excerpts, diff hunks, and unsafe fields stay out of internal/pipeline/steps/prsummary.go

**Checkpoint**: User Story 4 is complete when PR summaries expose material review-resolution context while keeping the durable report as the detailed sanitized surface.

---

## Phase 7: Polish & Cross-Cutting Validation

**Purpose**: Prove cross-surface consistency, docs correctness, generated artifact state, and full repository health.

- [ ] T053 [P] Add cross-surface summary-count consistency tests across report metadata, AXI, TUI, and PR fixtures in internal/reviewreport/integration_test.go
- [ ] T054 [P] Update TUI documentation for report references and review gate details in docs/src/content/docs/guides/tui.md
- [ ] T055 Run focused report and persistence tests from specs/002-review-resolution-report/quickstart.md for internal/reviewreport, internal/db, internal/ipc, and internal/daemon
- [ ] T056 Run focused AXI, TUI, and PR summary tests from specs/002-review-resolution-report/quickstart.md for internal/cli, internal/tui, and internal/pipeline/steps
- [ ] T057 Run full race validation from specs/002-review-resolution-report/quickstart.md with go test -race ./...
- [ ] T058 Run lint and docs validation from specs/002-review-resolution-report/quickstart.md with make lint and make docs-build
- [ ] T059 Regenerate and validate the generated skill with make skill if T046 changed skills/no-mistakes/SKILL.md
- [ ] T060 Record any validation gaps, deferred risks, or non-applicable checks in specs/002-review-resolution-report/quickstart.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1**: No dependencies.
- **Phase 2**: Depends on Phase 1 and blocks all user-story work.
- **Phase 3 (US1)**: Depends on Phase 2 and is the MVP implementation.
- **Phase 4 (US2)**: Depends on Phase 2; coordinates with US1 when touching internal/reviewreport/render.go and internal/reviewreport/report.go.
- **Phase 5 (US3)**: Depends on Phase 2; needs daemon/IPC metadata and may proceed once report metadata fixtures exist.
- **Phase 6 (US4)**: Depends on Phase 2; can proceed with report metadata fixtures before all AXI/TUI work is done.
- **Phase 7**: Depends on the selected user stories being complete.

### User Story Dependencies

- **US1 (P1)**: MVP. Required for durable report generation and TUI report references.
- **US2 (P1)**: Can be developed after the shared report contracts exist; highest risk is shared renderer/report files with US1.
- **US3 (P2)**: Depends on IPC/daemon metadata projection and reusable report rendering helpers.
- **US4 (P2)**: Depends on persisted report metadata and sanitized material fix-summary extraction.

### Within Each User Story

- Write or update tests before implementation changes.
- Implement pure package behavior before executor, daemon, CLI, TUI, or PR wiring.
- Keep report derivation centralized in internal/reviewreport before rendering summaries in other packages.
- Verify each story independently before moving to the next priority when working sequentially.

## Parallel Opportunities

- T003 and T004 can run in parallel after T001-T002 start.
- T005-T008 can run in parallel because they touch different test files.
- T015-T019 can run in parallel after Phase 2 because they add distinct test coverage.
- T032-T033, T040-T042, and T047-T048 can run in parallel within their phases.
- US3 and US4 can be staffed in parallel after Phase 2 if they use metadata fixtures and coordinate changes to shared report summary helpers.
- Documentation tasks T037, T038, T054 and generated-skill task T046 can be parallelized with code work after the surface behavior is stable.

## Parallel Example: User Story 1

```bash
Task: "T015 Add Markdown contract heading, label, and summary-count rendering tests in internal/reviewreport/render_test.go"
Task: "T016 Add decision mapping, summary count, and latest-outcome precedence tests in internal/reviewreport/derive_test.go"
Task: "T017 Add sanitizer tests for diff-like, code-like, log-like, transcript-like, and secret-like values in internal/reviewreport/sanitize_test.go"
Task: "T018 Add executor lifecycle tests for review findings, selected fixes, fix summaries, final review evidence, cancellation, and generation failure in internal/pipeline/executor_review_report_test.go"
Task: "T019 Add TUI review gate report metadata rendering tests in internal/tui/pipeline_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "T040 Add AXI renderer tests for current, stale, unavailable, error, incomplete, and successful report metadata in internal/cli/axi_render_report_test.go"
Task: "T041 Add AXI drive success-output tests for runs with report metadata and applied fixes in internal/cli/axi_drive_test.go"
Task: "T042 Add AXI status/query tests for report reference, latest outcome, and count display in internal/cli/axi_test.go"
```

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for US1.
3. Validate with focused tests for internal/reviewreport, internal/db, internal/ipc, internal/daemon, internal/pipeline, and internal/tui.
4. Stop and review the generated report against specs/002-review-resolution-report/contracts/review-resolution-report-markdown.md.

### Incremental Delivery

1. Deliver US1 so durable reports exist and TUI can reference them.
2. Deliver US2 so future agents and maintainers have preserved purpose, source evidence, and missing-data semantics.
3. Deliver US3 so agent-facing AXI output references review-resolution details on successful and incomplete runs.
4. Deliver US4 so PR summaries include concise review-resolution context.
5. Finish Phase 7 validation and docs/generated-skill checks.

### Team Strategy

After Phase 2, one engineer can own internal/reviewreport and executor wiring for US1/US2, another can own AXI output for US3, and another can own PR summary work for US4. Coordinate any shared changes to internal/reviewreport/types.go, internal/reviewreport/report.go, and internal/reviewreport/render.go before merging.

## Notes

- The report is a reporting layer only; do not change review, approval, auto-fix, push, PR, or CI semantics.
- `Recommendation` and `Applied fix` are distinct labels; do not treat suggested fixes as applied fixes.
- `Accepted` requires explicit stored human risk-acceptance evidence and must not be inferred from approve, skip, or unselected findings.
- `no issues remain` requires successfully parsed latest review evidence for the same run after the relevant fix attempt.
- The report and surfaces must not include raw logs, raw transcripts, raw diff hunks, code excerpts, or secret-bearing values.
