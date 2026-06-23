# Analyze Findings - Review Resolution Report

**Status:** APPLIED
**Generated:** 2026-06-18T23:52:50+07:00
**Applied:** 2026-06-19T00:01:11+07:00
**Spec:** spec.md
**Plan:** plan.md
**Tasks:** tasks.md
**Mode:** batch

**Instructions:**
- Review Section 2 Findings table.
- For each finding, edit the matching `### <ID>` block in Section 3 Resolutions Log.
  Fill `Category:` with one of: `spec-fix`, `new-OQ`, `accepted-risk`, `out-of-scope`, `skipped`.
  Fill `Payload:` per the category contract.
- Save the file, then run `/analyzebatch --apply` (add `--allow-historical-edits` if any
  `spec-fix` targets `specs/<feature-id>/spec.md` / `plan.md` / `tasks.md`).
- Pass `--dry-run` to preview the integration plan without writing.

---

## 1. Session Summary

The analysis found no duplication or constitution conflicts. The risk pattern is task coverage: several mandatory spec constraints are stated clearly in `spec.md`, `plan.md`, or the contract, but the task list does not yet create explicit implementation/test work for those exact guarantees. One atomicity requirement also needs a concrete refresh protocol before implementation.

## 2. Findings

| ID | Category | Severity | Location(s) | Summary | Recommendation | Status |
|----|----------|----------|-------------|---------|----------------|--------|
| V1 | Coverage | HIGH | spec.md:L142; plan.md:L21; contracts/review-resolution-report.md:L11; tasks.md:L37-L45,L155-L157 | The spec forbids creating, staging, force-adding, or committing repo-local `no-mistakes/<branch-slug>/review-resolution.md`, but tasks only cover `$NM_HOME` path helpers and PR-body privacy. No task directly proves the old repo-local/force-add behavior cannot reappear. | Add a focused task/test that asserts review-resolution reports are only written under `$NM_HOME`, are never staged or force-added by push/evidence code, and cannot produce a repo-local `review-resolution.md`. | spec-fix |
| V2 | Coverage | HIGH | spec.md:L151; contracts/review-resolution-report.md:L59-L77; tasks.md:L65-L77 | FR-011 requires preserving original finding fields and explicit unavailable markers for legacy gaps, while the task list has generic render/source tasks but no explicit test for context, suggested/proposed fix, risk rationale, user instructions, or unavailable markers. | Add a report-rendering/source-loading test task for full original-field preservation and explicit `not recorded` / `unavailable in historical data` markers, then bind implementation work to that fixture. | spec-fix |
| U1 | Underspecification | MEDIUM | spec.md:L168; plan.md:L21,L112-L120; data-model.md:L204-L210; tasks.md:L67-L78 | The artifacts require Markdown and SQLite metadata refresh to be atomic from the consumer perspective, but they do not specify the write ordering or recovery contract across the file write, metadata upsert, content hash, and watermark. | Specify the refresh protocol, including temp-file/rename order, DB transaction boundaries, hash/watermark validation, and what consumers see during partial failure; add tests around that exact order. | spec-fix |
| V3 | Coverage | MEDIUM | spec.md:L163; data-model.md:L33; plan.md:L22; tasks.md:L74-L83,L168-L176 | FR-021 scopes this feature to Review only, but the task list does not include a negative test proving Test/Lint/Document findings do not create or refresh a review-resolution report. Existing tasks wire Review source loading but do not explicitly guard other step findings. | Add a targeted unit or E2E task that seeds non-Review findings and verifies no `review_resolution_reports` row or Markdown report is created for them. | spec-fix |

**Coverage Summary:**

| Requirement Key | Has Task? | Task IDs | Notes |
|-----------------|-----------|----------|-------|
| FR-001 | yes | T023, T029, T031 | Report creation/refresh after Review findings. |
| FR-002 | partial | T006, T061 | `$NM_HOME` and PR privacy covered; staging/force-add guard missing (V1). |
| FR-003 | yes | T058, T062 | Clean Review omission. |
| FR-004 | yes | T021, T028 | Run Context rendered through golden tests/rendering. |
| FR-005 | yes | T020, T026, T027 | Source loading and ID reconciliation. |
| FR-006 | yes | T021, T028 | One section per issue through golden/render tests. |
| FR-007 | yes | T020, T027, T028 | Outcome categories including informational. |
| FR-008 | yes | T020, T027 | Resolved classification evidence. |
| FR-009 | yes | T010, T030, T040 | Persisted terminal decision evidence. |
| FR-010 | yes | T035, T040, T041, T045 | Still-open and non-success outcomes. |
| FR-011 | partial | T021, T026, T028 | Original-field preservation test missing (V2). |
| FR-012 | yes | T046, T050, T051 | Structured fix-agent contract. |
| FR-013 | yes | T047, T048, T050 | Legacy summary compatibility. |
| FR-014 | yes | T046, T053, T055 | Structured detail validation and labeling. |
| FR-015 | yes | T049, T054, T055 | Diff/commit fallback and privacy. |
| FR-016 | yes | T048, T055 | Source labeling. |
| FR-017 | yes | T007, T009, T014, T015, T017, T036 | Metadata table/status. |
| FR-018 | yes | T008, T012, T039, T042, T043 | Fix commit/no-commit evidence. |
| FR-019 | yes | T020, T054, T055 | Shared round/commit evidence labeling. |
| FR-020 | yes | T023, T031, T037, T041, T059 | Lifecycle refresh and PR reconciliation. |
| FR-021 | partial | T026, T031 | Review source loading covered; non-Review negative test missing (V3). |
| FR-022 | yes | T013, T014, T024, T025, T032, T033, T045 | AXI/TUI compact metadata. |
| FR-023 | yes | T056, T057, T060, T061, T062 | PR compact/privacy behavior. |
| FR-024 | yes | T018, T019, T049 | Sanitization/truncation/privacy. |
| FR-025 | partial | T022, T029, T037, T044 | Atomicity intent covered; concrete cross-resource protocol underspecified (U1). |
| FR-026 | yes | T065, T066, T067 | Docs and generated skill updates. |
| SC-001 | yes | T020, T021, T027, T028 | Normalized finding ID coverage. |
| SC-002 | yes | T058, T062 | Clean Review omission. |
| SC-003 | yes | T046, T048, T053 | Structured fix-agent details. |
| SC-004 | yes | T048, T055 | Inferred-source labeling. |
| SC-005 | yes | T035, T040, T041 | Aborted/failed still-open behavior. |
| SC-006 | yes | T007, T008 | Additive migration coverage. |
| SC-007 | yes | T022, T024, T025, T056, T059, T060 | Markdown/SQLite/AXI/TUI/PR count agreement. |
| SC-008 | yes | T029, T037 | Report/metadata write failures. |
| SC-009 | yes | T018, T019, T049 | Oversized/untrusted content. |

**Constitution Alignment Issues:** None.

**Unmapped Tasks:** Setup and validation tasks T001-T004 and T063-T071 are intentionally cross-cutting. No task appears unrelated to the feature.

**Metrics:**

- Total Requirements: 35
- Total Tasks: 71
- Coverage % (requirements with direct task coverage): 91% (32/35 direct, 3 partial)
- Ambiguity Count: 0
- Duplication Count: 0
- Critical Issues Count: 0

## 3. Resolutions Log

### V1
  Category: spec-fix
  Payload:
    Target: specs/002-review-resolution-report/tasks.md
    Before:
    ```markdown
    - [ ] T057 [P] [US4] Add PR step tests for pre-summary reconciliation and local-path privacy in `internal/pipeline/steps/pr_test.go`
    ```
    After:
    ```markdown
    - [ ] T057 [P] [US4] Add PR and push/evidence staging tests for pre-summary reconciliation, local-path privacy, and proving review-resolution reports stay under `$NM_HOME` metadata paths and are never staged or force-added from a repo-local `no-mistakes/<branch-slug>/review-resolution.md` path in `internal/pipeline/steps/pr_test.go` and `internal/pipeline/steps/push_test.go`
    ```
    Rationale: The feature contract already rejects repo-local report artifacts: `specs/002-review-resolution-report/spec.md` says the durable report belongs under `$NM_HOME/reports/<runID>/review-resolution.md` and must not create, stage, force-add, or commit `no-mistakes/<branch-slug>/review-resolution.md`; the contract repeats that the file is local-only evidence and must not be staged, committed, force-added, or publicly linked. A narrow source check found `internal/pipeline/steps/push.go` is the existing force-add path for in-repo evidence, so expanding the PR/privacy test task to also cover push/evidence staging is the smallest task-level guard against the historical mechanics returning.

Status: applied
Applied-at: 2026-06-19T00:01:11+07:00
Downstream-ref: specs/002-review-resolution-report/tasks.md

### V2
  Category: spec-fix
  Payload:
    Target: specs/002-review-resolution-report/tasks.md
    Before:
    ```markdown
    - [ ] T021 [P] [US1] Add golden Markdown tests for required headings, field labels, counts, and section order in `internal/reviewreport/render_test.go` and `internal/reviewreport/testdata/mixed_resolved_accepted.golden.md`
    ```
    After:
    ```markdown
    - [ ] T021 [P] [US1] Add golden Markdown tests for required headings, field labels, counts, section order, full original finding field preservation (context, suggested/proposed fix, risk level/rationale, user instructions), and explicit `not recorded` / `unavailable in historical data` markers in `internal/reviewreport/render_test.go` and `internal/reviewreport/testdata/mixed_resolved_accepted.golden.md`
    ```
    Rationale: `FR-011` requires original finding details including context, suggested/proposed fix, risk rationale, and user instructions, and the Markdown contract lists those per-issue fields plus explicit unavailable markers. Local finding types already persist severity, file, line, action, source, user instructions, and risk fields, while legacy data may not have newer context/fix fields; a golden rendering test is the simplest durable fixture because it proves both preservation and honest marker output without adding a new abstraction.

Status: applied
Applied-at: 2026-06-19T00:01:11+07:00
Downstream-ref: specs/002-review-resolution-report/tasks.md

### U1
  Category: spec-fix
  Payload:
    Target: specs/002-review-resolution-report/plan.md
    Before:
    ```markdown
    - atomically writing Markdown and metadata.
    ```
    After:
    ```markdown
    - atomically refreshing Markdown and metadata with a consumer-safe protocol: render and classify the Review snapshot in memory; write Markdown bytes to a temporary file in the target report directory; close/fsync the file and atomically rename it over `$NM_HOME/reports/<runID>/review-resolution.md`; compute `content_hash` and `source_watermark` from the exact rendered bytes and source rounds/decisions; then upsert `review_resolution_reports` in one DB transaction that preserves `first_generated_at` and updates counts, watermark, hash, and `last_refresh_result`. If the file write or rename fails, leave previous Markdown/metadata untouched and return an actionable Review error. If the DB transaction fails after rename, consumers must detect the hash/watermark mismatch and render degraded or evidence-unavailable status instead of confident counts. Refresh tests must assert this order and both partial-failure paths.
    ```
    Rationale: `FR-025` requires consumer-perspective atomicity and degraded/evidence-unavailable status on consistency failure; the plan already puts atomic refresh inside `internal/reviewreport`, and `data-model.md` defines `content_hash`, `source_watermark`, `last_refresh_result`, and the invariant that consumers treat missing Markdown, hash mismatch, or stale watermark as degraded/evidence-unavailable. Writing and renaming the Markdown before committing metadata keeps consumers from seeing new confident counts for an unwritten file, while the hash/watermark check covers the only unavoidable split-brain case: a successful rename followed by a failed metadata transaction.

Status: applied
Applied-at: 2026-06-19T00:01:11+07:00
Downstream-ref: specs/002-review-resolution-report/plan.md

### V3
  Category: spec-fix
  Payload:
    Target: specs/002-review-resolution-report/tasks.md
    Before:
    ```markdown
    - [ ] T023 [P] [US1] Add executor lifecycle tests for report creation after first Review findings and refresh after fix/approve in `internal/pipeline/executor_review_report_test.go`
    ```
    After:
    ```markdown
    - [ ] T023 [P] [US1] Add executor lifecycle tests for report creation after first Review findings, refresh after fix/approve, and negative cases where Test, Lint, or Document findings do not create or refresh `review_resolution_reports` rows or Markdown reports in `internal/pipeline/executor_review_report_test.go`
    ```
    Rationale: `FR-021` and the plan scope limit this feature to Review-only reporting, while `data-model.md` says only the Review step is in scope. Local step names show Review, Test, Document, and Lint are distinct pipeline steps, so the right fix is not a broader all-step evidence design; it is a focused executor regression test proving non-Review findings do not trigger report rows or Markdown artifacts.

Status: applied
Applied-at: 2026-06-19T00:01:11+07:00
Downstream-ref: specs/002-review-resolution-report/tasks.md

---

## 5. Session Metadata

```yaml
session:
  generated_at: 2026-06-18T23:52:50+07:00
  feature_dir: specs/002-review-resolution-report
  artifacts_analyzed:
    - spec.md
    - plan.md
    - tasks.md
    - .specify/memory/constitution.md
  supporting_context:
    - contracts/review-resolution-report.md
    - data-model.md
    - quickstart.md
    - research.md
  findings:
    total: 4
    by_severity:
      critical: 0
      high: 2
      medium: 2
      low: 0
    by_category:
      duplication: 0
      ambiguity: 0
      underspecification: 1
      constitution: 0
      coverage: 3
      inconsistency: 0
    overflow_dropped: 0
apply:
  applied_at: 2026-06-19T00:01:11+07:00
  applied_by: Codex
  resolutions:
    spec_fix: 4
    new_OQ: 0
    accepted_risk: 0
    out_of_scope: 0
    skipped: 0
  unresolved: 0
  allow_historical_edits: true
  historical_edits_applied:
      - V1:specs/002-review-resolution-report/tasks.md
      - V2:specs/002-review-resolution-report/tasks.md
      - U1:specs/002-review-resolution-report/plan.md
      - V3:specs/002-review-resolution-report/tasks.md
```

## Correction Note - 2026-06-21

The V1 analysis and its downstream task edits treated an invalid `superseded`
header in `plans/grill-me/review-resolution-report.md` as authoritative. That
was wrong. The binding source is the full `## Decisions` section in the
grill-me file.

Any statement in this analysis that recommends
`$NM_HOME/reports/<runID>/review-resolution.md`, rejects repo-local
`no-mistakes/<branch-slug>/review-resolution.md`, rejects exact force-add
staging, or removes PR evidence links is superseded by this correction. This
file remains historical audit context only; current specs and implementation
must carry forward the repo-local committed report contract.
