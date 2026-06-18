---
description: "Task list for Review Resolution Report implementation"
---

# Tasks: Review Resolution Report

**Input**: Design documents from `specs/002-review-resolution-report/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/review-resolution-report.md`, `quickstart.md`

**Tests**: Required for all code changes. Add targeted Go tests before implementation tasks, then run focused packages, E2E coverage, race tests, lint, and generated skill validation.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested as an independent increment after shared foundations land.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or only adds independent tests/fixtures
- **[Story]**: User story label for story-phase tasks only
- Every task includes exact file paths or package paths

## Phase 1: Setup (Shared Orientation)

**Purpose**: Confirm scope, source boundaries, and validation targets before editing implementation files.

- [ ] T001 Review the active feature inputs in `specs/002-review-resolution-report/spec.md`, `specs/002-review-resolution-report/plan.md`, and `specs/002-review-resolution-report/contracts/review-resolution-report.md`
- [ ] T002 Check `git status --short` and note unrelated user changes before editing files under `internal/`, `docs/src/content/docs/`, and `skills/no-mistakes/SKILL.md`
- [ ] T003 [P] Confirm verification commands from `specs/002-review-resolution-report/quickstart.md` for `internal/reviewreport`, `internal/db`, `internal/pipeline`, `internal/pipeline/steps`, `internal/ipc`, `internal/cli`, `internal/tui`, `internal/daemon`, and `internal/e2e`
- [ ] T004 [P] Create implementation and fixture directories `internal/reviewreport/` and `internal/reviewreport/testdata/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared DB, path, IPC, and sanitization contracts needed by every user story.

**Critical**: No user story implementation should begin until these contracts are complete.

- [ ] T005 [P] Add path helper tests for `ReportsDir`, `ReviewResolutionReportPath`, and `EnsureDirs` in `internal/paths/paths_test.go`
- [ ] T006 Implement `ReportsDir()` and `ReviewResolutionReportPath(runID string)` and ensure reports directory creation in `internal/paths/paths.go`
- [ ] T007 [P] Add additive migration tests for `review_resolution_reports`, `review_resolution_decisions`, and new `step_rounds` evidence columns in `internal/db/review_resolution_report_test.go`
- [ ] T008 Add `review_resolution_reports`, `review_resolution_decisions`, `fix_commit_sha`, `no_commit_reason`, and `fix_resolution_details_json` schema/migrations in `internal/db/schema.go`
- [ ] T009 Implement review-resolution metadata model and upsert/read/delete accessors in `internal/db/review_resolution_report.go`
- [ ] T010 Implement Review terminal decision model and insert/query accessors in `internal/db/review_resolution_decision.go`
- [ ] T011 Extend `StepRound`, `InsertStepRound`, `GetRoundsByStep`, and setter helpers for fix evidence fields in `internal/db/round.go`
- [ ] T012 Add round persistence tests for fix commit, no-commit reason, structured detail JSON, and legacy `StepFixSummaries` behavior in `internal/db/round_test.go`
- [ ] T013 [P] Add IPC round-trip tests for review-resolution metadata wire fields in `internal/ipc/protocol_test.go`
- [ ] T014 Add `ReviewResolutionReportInfo` and optional run-level report metadata to `internal/ipc/protocol.go`
- [ ] T015 Hydrate report metadata in daemon `runToInfo`/`stepToInfo` paths in `internal/daemon/daemon.go`
- [ ] T016 Add daemon run-info metadata hydration tests in `internal/daemon/runinfo_test.go`
- [ ] T017 [P] Define review report status, count, entry, decision, and refresh result structs in `internal/reviewreport/types.go`
- [ ] T018 [P] Add field-level sanitization, truncation, secret redaction, Markdown escape, raw diff/log/transcript stripping, and code-fence tests in `internal/reviewreport/sanitize_test.go`
- [ ] T019 Implement field-level sanitization and bounded string helpers in `internal/reviewreport/sanitize.go`

**Checkpoint**: DB/path/IPC/report primitives exist and tests can compile; user story work can proceed.

---

## Phase 3: User Story 1 - Read Review Outcomes After a Gate Run (Priority: P1) MVP

**Goal**: Create a durable local report after Review findings and expose compact report status/counts/path through local run-detail surfaces.

**Independent Test**: Run a Review scenario with two findings, one fixed and one explicitly approved; verify `$NM_HOME/reports/<runID>/review-resolution.md`, SQLite metadata, AXI output, and TUI output agree on both findings.

### Tests for User Story 1

- [ ] T020 [P] [US1] Add classifier tests for mixed resolved/accepted findings, repeated same-ID findings, and exact normalized ID matching in `internal/reviewreport/classifier_test.go`
- [ ] T021 [P] [US1] Add golden Markdown tests for required headings, field labels, counts, section order, full original finding field preservation (context, suggested/proposed fix, risk level/rationale, user instructions), and explicit `not recorded` / `unavailable in historical data` markers in `internal/reviewreport/render_test.go` and `internal/reviewreport/testdata/mixed_resolved_accepted.golden.md`
- [ ] T022 [P] [US1] Add DB-backed refresh tests for atomic Markdown write plus metadata upsert in `internal/reviewreport/refresh_test.go`
- [ ] T023 [P] [US1] Add executor lifecycle tests for report creation after first Review findings, refresh after fix/approve, and negative cases where Test, Lint, or Document findings do not create or refresh `review_resolution_reports` rows or Markdown reports in `internal/pipeline/executor_review_report_test.go`
- [ ] T024 [P] [US1] Add AXI compact status/count/path rendering tests in `internal/cli/axi_render_test.go`
- [ ] T025 [P] [US1] Add TUI compact status/count/path rendering tests in `internal/tui/pipeline_test.go`

### Implementation for User Story 1

- [ ] T026 [US1] Implement Review step/run/round source loading from persisted DB records in `internal/reviewreport/source.go`
- [ ] T027 [US1] Implement normalized finding reconciliation and resolved/accepted classification in `internal/reviewreport/classifier.go`
- [ ] T028 [US1] Implement deterministic Markdown rendering with report format version, Run Context, Counts, Resolved Issues, Accepted Without Fix, Informational / No Action Required, and Still Open Issues in `internal/reviewreport/render.go`
- [ ] T029 [US1] Implement atomic report refresh, content hash, source watermark, status/count metadata, and write failure propagation in `internal/reviewreport/refresh.go`
- [ ] T030 [US1] Persist Review approve/skip decision provenance with action, actor/source, affected finding IDs, timestamp, and reason fields in `internal/pipeline/executor.go`
- [ ] T031 [US1] Call `reviewreport.Refresh` after initial Review findings, after Review fix rounds, and after approve/skip completion in `internal/pipeline/executor.go`
- [ ] T032 [US1] Render AXI report status, resolved/accepted/informational/still-open counts, and local path in `internal/cli/axi_render.go`
- [ ] T033 [US1] Render TUI report status, counts, and local path in `internal/tui/pipeline.go`
- [ ] T034 [US1] Ensure report metadata reaches AXI/TUI through `internal/daemon/daemon.go` and `internal/ipc/protocol.go`

**Checkpoint**: US1 is independently demonstrable with a mixed fixed/approved Review run.

---

## Phase 4: User Story 2 - Preserve Honest Status for Incomplete Runs (Priority: P1)

**Goal**: Keep unresolved findings still open during failed, aborted, canceled, superseded, stale, or no-evidence workflows.

**Independent Test**: Run a Review scenario with two findings, fix one, abort before approving the second; verify the remaining finding is still open and metadata status is not final/success.

### Tests for User Story 2

- [ ] T035 [P] [US2] Add classifier tests for abort, failure, canceled, superseded, verification-inconclusive, and still-open outcomes in `internal/reviewreport/classifier_test.go`
- [ ] T036 [P] [US2] Add status transition tests for `incomplete`, `stale`, `degraded`, and `evidence_unavailable` in `internal/reviewreport/refresh_test.go`
- [ ] T037 [P] [US2] Add executor tests for abort/failure report refresh and required write failure behavior in `internal/pipeline/executor_review_report_test.go`
- [ ] T038 [P] [US2] Add daemon cancellation and stale-run recovery tests for report reconciliation in `internal/daemon/manager_test.go`
- [ ] T039 [P] [US2] Add no-commit evidence tests for no-op, no changes, failed commit, and missing evidence in `internal/pipeline/steps/common_fix_test.go`

### Implementation for User Story 2

- [ ] T040 [US2] Persist abort/failure decision and evidence state for Review findings in `internal/pipeline/executor.go`
- [ ] T041 [US2] Reconcile reports when runs become failed, cancelled, superseded, or stale-recovered in `internal/pipeline/executor.go` and `internal/daemon/manager.go`
- [ ] T042 [US2] Extend fix commit handling to distinguish no-op, no changes, failed commit, and missing evidence in `internal/pipeline/steps/common_fix.go`
- [ ] T043 [US2] Persist `fix_commit_sha` and `no_commit_reason` on fix rounds through `internal/pipeline/pipeline.go`, `internal/pipeline/executor.go`, and `internal/db/round.go`
- [ ] T044 [US2] Implement stale/degraded/evidence-unavailable integrity checks for missing Markdown, hash mismatch, source watermark drift, and partial refreshes in `internal/reviewreport/refresh.go`
- [ ] T045 [US2] Render non-success wording whenever status is incomplete/stale/degraded/evidence-unavailable or still-open count is nonzero in `internal/cli/axi_render.go` and `internal/tui/pipeline.go`

**Checkpoint**: US2 is independently demonstrable with aborted and failed Review runs that never misclassify unresolved findings as accepted.

---

## Phase 5: User Story 3 - See Applied Solution Details Beyond Commit Summary (Priority: P2)

**Goal**: Capture structured fix-agent resolution details when available and honestly label inferred or degraded solution evidence.

**Independent Test**: Use one fake Review fix agent that returns `resolutions[]` and another that returns only `summary`; verify structured entries are preferred and legacy entries are labeled as inferred/degraded without raw diffs or code excerpts.

### Tests for User Story 3

- [ ] T046 [P] [US3] Add fix detail validation tests for required fields, duplicate IDs, unknown IDs, missing selected IDs, changed files, and per-field caps in `internal/reviewreport/fixdetail_test.go`
- [ ] T047 [P] [US3] Add Review fix schema and backward-compatible prompt tests in `internal/pipeline/steps/review_test.go`
- [ ] T048 [P] [US3] Add structured-resolution and legacy-summary golden tests in `internal/reviewreport/render_test.go` and `internal/reviewreport/testdata/structured_resolution.golden.md`
- [ ] T049 [P] [US3] Add diff-derived fallback privacy tests for raw hunks, code snippets, secret literals, and near-verbatim code excerpts in `internal/reviewreport/sanitize_test.go`

### Implementation for User Story 3

- [ ] T050 [US3] Extend the Review fix JSON schema and prompt to accept `summary` plus optional `resolutions[]` in `internal/pipeline/steps/review.go` and `internal/pipeline/steps/common_fix.go`
- [ ] T051 [US3] Parse, validate, sanitize, and classify optional structured fix resolution details in `internal/reviewreport/fixdetail.go` and `internal/pipeline/steps/common_fix.go`
- [ ] T052 [US3] Add fix detail, fix commit, and no-commit evidence fields to `StepOutcome` and pass them through executor round persistence in `internal/pipeline/pipeline.go` and `internal/pipeline/executor.go`
- [ ] T053 [US3] Apply validated structured resolution details to matching report entries only when tied to persisted fix-round evidence in `internal/reviewreport/classifier.go`
- [ ] T054 [US3] Implement high-level sanitized diff/commit fallback summaries and changed-file labels in `internal/reviewreport/fallback.go`
- [ ] T055 [US3] Label structured, inferred, round-level, commit-level, degraded, unavailable, attempted, and not-applied evidence explicitly in `internal/reviewreport/render.go`

**Checkpoint**: US3 is independently demonstrable with structured and legacy fake-agent Review fix outputs.

---

## Phase 6: User Story 4 - Reference Review Resolution From PR Summary (Priority: P3)

**Goal**: Add compact review-resolution status/counts to generated PR content without exposing local paths, report excerpts, or fake public links.

**Independent Test**: Build PR content for runs with and without report metadata; verify the PR body includes compact counts/status only when metadata exists and never includes `$NM_HOME`, absolute local paths, report excerpts, or blob links for `review-resolution.md`.

### Tests for User Story 4

- [ ] T056 [P] [US4] Add PR summary tests for compact report metadata rendering, omit-when-absent behavior, and non-success wording in `internal/pipeline/steps/prsummary_test.go`
- [ ] T057 [P] [US4] Add PR and push/evidence staging tests for pre-summary reconciliation, local-path privacy, and proving review-resolution reports stay under `$NM_HOME` metadata paths and are never staged or force-added from a repo-local `no-mistakes/<branch-slug>/review-resolution.md` path in `internal/pipeline/steps/pr_test.go` and `internal/pipeline/steps/push_test.go`
- [ ] T058 [P] [US4] Add clean-run omission tests for IPC/AXI report metadata in `internal/ipc/protocol_test.go` and `internal/cli/axi_render_test.go`

### Implementation for User Story 4

- [ ] T059 [US4] Reconcile review report metadata before PR summary generation uses it in `internal/pipeline/steps/pr.go`
- [ ] T060 [US4] Add compact `Review resolution` status/count rendering to the generated `## Pipeline` section in `internal/pipeline/steps/prsummary.go`
- [ ] T061 [US4] Prevent `$NM_HOME`, absolute local paths, `review-resolution.md` local paths, report excerpts, and generated GitHub blob links from PR content in `internal/pipeline/steps/prsummary.go` and `internal/pipeline/steps/pr.go`
- [ ] T062 [US4] Omit review-resolution status entirely for clean Review runs and runs without report metadata in `internal/pipeline/steps/prsummary.go`

**Checkpoint**: US4 is independently demonstrable with PR body fixtures for report-present and report-absent runs.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: End-to-end validation, docs, generated skill updates, and full repository checks.

- [ ] T063 [P] Add E2E mixed resolved/accepted and clean Review scenarios in `internal/e2e/review_resolution_test.go`
- [ ] T064 [P] Add E2E aborted unresolved, structured `resolutions[]`, legacy `summary`, and AXI detail scenarios in `internal/e2e/review_resolution_test.go`
- [ ] T065 [P] Update Review, Auto-Fix, pipeline, and gate model docs in `docs/src/content/docs/reference/pipeline-steps.md`, `docs/src/content/docs/concepts/auto-fix.md`, `docs/src/content/docs/concepts/pipeline.md`, and `docs/src/content/docs/concepts/gate-model.md`
- [ ] T066 [P] Update TUI, AXI, PR, and local state docs in `docs/src/content/docs/guides/tui.md`, `docs/src/content/docs/reference/cli.md`, and `docs/src/content/docs/concepts/daemon.md`
- [ ] T067 Regenerate generated agent guidance in `skills/no-mistakes/SKILL.md` using `make skill`
- [ ] T068 Run `gofmt` on changed files under `internal/reviewreport/`, `internal/db/`, `internal/paths/`, `internal/pipeline/`, `internal/ipc/`, `internal/cli/`, `internal/tui/`, and `internal/daemon/`
- [ ] T069 Run targeted tests for `./internal/reviewreport`, `./internal/db`, `./internal/paths`, `./internal/pipeline`, `./internal/pipeline/steps`, `./internal/ipc`, `./internal/cli`, `./internal/tui`, and `./internal/daemon`
- [ ] T070 Run E2E tests for `./internal/e2e` with `go test -tags=e2e ./internal/e2e -run 'ReviewResolution|Axi'`
- [ ] T071 Run full validation for repository paths `internal/`, `cmd/`, `docs/src/content/docs/`, and `skills/no-mistakes/SKILL.md` with `go test -race ./...`, `make lint`, and `make skill`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational.
- **User Story 2 (Phase 4)**: Depends on Foundational; coordinates with US1 on `internal/reviewreport/classifier.go`, `internal/reviewreport/refresh.go`, and `internal/pipeline/executor.go`.
- **User Story 3 (Phase 5)**: Depends on Foundational; can use fixture metadata but integrates most cleanly after US1 report rendering exists.
- **User Story 4 (Phase 6)**: Depends on Foundational metadata accessors; can use metadata fixtures before full lifecycle wiring.
- **Polish (Phase 7)**: Depends on all selected user stories.

### User Story Dependencies

- **US1 (P1)**: Delivers MVP report creation, local metadata, AXI/TUI exposure.
- **US2 (P1)**: Delivers trust-preserving incomplete/failure/cancel behavior; product behavior is independent from US1 but shares report classifier and refresh files.
- **US3 (P2)**: Adds richer fix evidence while preserving legacy summary-only behavior.
- **US4 (P3)**: Adds PR summary surfacing after compact metadata exists.

### Within Each User Story

- Write tests/golden fixtures first and confirm they fail where feasible.
- Implement shared contracts before lifecycle hooks.
- Implement report classification before rendering/surface claims.
- Implement DB metadata before AXI/TUI/PR consumption.
- Run targeted tests for the story before moving to the next priority.

---

## Parallel Opportunities

- T003 and T004 can run in parallel after T001.
- T005, T007, T013, T017, and T018 can run in parallel because they add independent tests/types.
- T020 through T025 can run in parallel after Phase 2.
- T035 through T039 can run in parallel after Phase 2.
- T046 through T049 can run in parallel after Phase 2.
- T056 through T058 can run in parallel after Phase 2.
- T063 through T066 can run in parallel after story behavior stabilizes.

## Parallel Example: User Story 1

```bash
# Tests that can be authored together:
Task: "T020 [US1] Add classifier tests in internal/reviewreport/classifier_test.go"
Task: "T021 [US1] Add golden Markdown tests in internal/reviewreport/render_test.go and internal/reviewreport/testdata/mixed_resolved_accepted.golden.md"
Task: "T024 [US1] Add AXI compact rendering tests in internal/cli/axi_render_test.go"
Task: "T025 [US1] Add TUI compact rendering tests in internal/tui/pipeline_test.go"

# Implementation slices that can proceed after report metadata contracts exist:
Task: "T026 [US1] Implement DB source loading in internal/reviewreport/source.go"
Task: "T028 [US1] Implement Markdown rendering in internal/reviewreport/render.go"
Task: "T032 [US1] Render AXI compact report metadata in internal/cli/axi_render.go"
Task: "T033 [US1] Render TUI compact report metadata in internal/tui/pipeline.go"
```

## Implementation Strategy

### MVP First (US1 + Required Foundation)

1. Complete Phase 1 and Phase 2.
2. Complete US1 tests and implementation.
3. Validate with `go test ./internal/reviewreport ./internal/db ./internal/pipeline ./internal/ipc ./internal/cli ./internal/tui ./internal/daemon`.
4. Stop and inspect the mixed fixed/approved report artifact before expanding scope.

### Trust Boundary Next (US2)

1. Add incomplete/failure/cancel/stale tests.
2. Implement decision provenance and lifecycle reconciliation.
3. Validate that unresolved findings never move to accepted without persisted authority.

### Evidence Detail Then PR Surface

1. Add US3 structured fix details while preserving legacy `summary`.
2. Add US4 compact PR status/counts using persisted metadata only.
3. Keep PR content local-path-free and narrative-free.

### Final Validation

1. Run targeted tests for changed packages.
2. Run E2E Review Resolution and AXI journeys.
3. Run `go test -race ./...`, `make lint`, and `make skill`.

## Notes

- `[P]` tasks must not edit the same files concurrently.
- `internal/reviewreport` owns classification, sanitization, rendering, and refresh rules; other packages should consume its metadata or refresh API instead of duplicating status logic.
- PR summaries must never publish `$NM_HOME`, local report paths, report excerpts, or generated blob links for `review-resolution.md`.
- AXI/TUI may show the local report path because those are local run-detail surfaces.
- Any report or metadata write failure after Review findings exist must fail the Review step/run with an actionable error.
