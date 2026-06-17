# Feature Specification: Review Resolution Report

**Feature Branch**: `002-review-resolution-report`  
**Created**: 2026-06-17  
**Status**: Draft  
**Input**: User description: "Create detailed spec for review-resolution-report; save origin and purpose reference for future tasks; scout source code context before implementation."

## Origin & Purpose

Users and future agents need a durable way to understand what happened during the review step after findings are fixed, accepted, skipped, or left unresolved. Today, that understanding is spread across live terminal state, agent-facing command output, PR summary fragments, and stored run data. The report must turn the review-resolution story into one readable reference: what the review found, what the user or automation chose to fix, what was actually fixed, what remains open, and why the latest review state is acceptable or still needs attention.

This feature is a reporting/reference capability. It must not change the review engine, approval semantics, auto-fix limits, push behavior, PR behavior, or CI behavior.

Detailed source-code context for planning and implementation is stored in `source-context.md` beside this specification.

## Clarifications

### Session 2026-06-17

- Q: Where should the durable Markdown report live relative to run data? → A: Store one run-scoped Markdown artifact and persist its path/reference with the run metadata.
- Q: When review-resolution state changes during one run, how should the report lifecycle behave? → A: Maintain one report per run, updating it as review decisions, fixes, and final outcomes become available.
- Q: Which user-facing surfaces must expose a direct report reference? → A: AXI success/status output, TUI review gate details, and PR summaries when review resolution occurred.
- Q: May the report include code excerpts or diff hunks from reviewed changes? → A: No; include finding locations, safe finding context, decisions, and summaries, but no code excerpts or diff hunks.
- Q: How should the report label a run that is cancelled or fails after review fixes but before trustworthy final review evidence exists? → A: Mark the latest outcome as "review resolution incomplete" and show the latest trustworthy evidence.
- Q: How should regenerated reports handle older or partial run data that lacks selected finding IDs, user instructions, or fix summaries? → A: Generate the report and label missing fields as "not recorded" or "unavailable" without inferring decisions.
- Q: What is the canonical label for actionable findings the user chose not to fix but deliberately accepted as risk? → A: Accepted.
- Q: What stable report contract should future agents and tests rely on? → A: Stable Markdown headings, user-facing labels, and summary counts.
- Q: When a PR summary references a durable report, what should it include directly? → A: Summary counts, latest review outcome, material applied-fix summaries, and a report reference.
- Q: What should happen if report generation itself fails after review data has been captured? → A: Continue the pipeline, surface a report-generation error, and preserve captured review data for later regeneration.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Understand review resolution (Priority: P1)

As a user reviewing a paused or completed gate run, I want a clear review-resolution report so I can understand which review findings were fixed, accepted, skipped, or still open without reconstructing the run from logs or terminal state.

**Why this priority**: This is the core value of the feature. A report that does not clearly explain review resolution fails the user's stated goal.

**Independent Test**: Run a review that produces at least one actionable finding, request a fix, and verify the report explains the original finding, selected action, applied fix, and final review outcome.

**Acceptance Scenarios**:

1. **Given** a review finding was selected for fixing, **When** the report is viewed, **Then** it shows the original issue, the selected resolution action, and the applied fix summary.
2. **Given** a finding was selected with user instructions, **When** the report is viewed, **Then** it includes those instructions with that finding.
3. **Given** a finding was deliberately not selected for fixing, **When** the report is viewed, **Then** it is marked as accepted, skipped, informational, or deferred instead of disappearing.
4. **Given** the latest review pass still has findings, **When** the report is viewed, **Then** it clearly distinguishes unresolved findings from findings that were fixed.

---

### User Story 2 - Preserve origin for future work (Priority: P1)

As a future coding agent or maintainer, I want the report and source-context reference to capture why this feature exists and which review-resolution facts matter so I do not repeat prior context gathering or misinterpret the review outcome.

**Why this priority**: The user explicitly asked to save the origin and purpose for later tasks. This is also necessary to avoid repeating earlier mistakes around partial review data paths.

**Independent Test**: Open the feature directory without reading raw logs and verify the spec plus source-context reference explain the feature purpose, current product behavior, and expected report outcome.

**Acceptance Scenarios**:

1. **Given** a future task starts from this feature directory, **When** the maintainer reads the spec, **Then** they can identify the report's purpose and non-goals.
2. **Given** a future task needs implementation context, **When** the maintainer reads the source-context reference, **Then** they can identify the current review, approval, reporting, and user-visible surfaces that shaped the spec.
3. **Given** user intent is available for a run, **When** the report is viewed, **Then** it includes a safe intent summary only from an already-sanitized user-facing intent field, or explicitly states that no safe intent summary is available.

---

### User Story 3 - Report successful runs without hiding misses (Priority: P2)

As an agent using the headless command surface, I want successful run output to reference review-resolution details when review fixes occurred so I can tell the user what the original change missed and what the pipeline fixed.

**Why this priority**: Agent-driven runs must close the loop with the user. A passing run should not hide the fact that review fixes were applied.

**Independent Test**: Drive a run through review findings and a fix, then verify the successful output includes or points to the review-resolution report.

**Acceptance Scenarios**:

1. **Given** a run passes after review fixes, **When** the agent-facing success output is produced, **Then** it includes review-resolution information or a clear report reference.
2. **Given** a run passes without review findings or review fixes, **When** the report is viewed, **Then** it states that no review findings required resolution rather than inventing a resolution narrative.
3. **Given** a run has informational review findings, **When** the report is viewed, **Then** they are labeled informational and not counted as applied fixes.

---

### User Story 4 - Support PR review context (Priority: P2)

As a reviewer reading a PR, I want a concise review-resolution summary so I can see what the gate caught and what changed before merge without reading full run logs.

**Why this priority**: The PR is the final human review surface before merge, so it should not omit material review fixes.

**Independent Test**: Generate a PR summary after one or more review fix rounds and verify it includes a concise review-resolution summary or references the durable report.

**Acceptance Scenarios**:

1. **Given** a PR includes pipeline-applied review fixes, **When** the PR summary is generated, **Then** it includes the review fixes and the latest review outcome.
2. **Given** multiple review fix rounds occurred, **When** the PR summary is generated, **Then** it summarizes the sanitized chain without exposing excessive raw round detail, raw code, diff hunks, logs, transcripts, secrets, or unsafe fix-summary text.
3. **Given** a durable report exists, **When** the PR summary is generated, **Then** it references the detailed report instead of duplicating the full report body.

### Edge Cases

- Review produces findings but the user has not responded yet.
- Review produces findings and the user fixes only a subset.
- Review produces only informational findings.
- Review has no reviewable changes after ignored paths are applied.
- A fix round records no summary.
- Multiple review fix rounds occur in one run.
- User-authored findings are added during the review gate.
- The latest review data is missing or cannot be parsed.
- A run is cancelled, superseded, or fails after review resolution but before pipeline completion.
- A run is cancelled or fails after review fixes but before trustworthy final review evidence exists; the report labels the latest outcome as "review resolution incomplete" and shows only the latest trustworthy evidence.
- A report is regenerated from stored run data after the live terminal event was missed.
- A report is regenerated from older or partial run data that lacks selected finding IDs, user instructions, or fix summaries; missing fields are labeled as `not recorded` or `unavailable` without inferred decisions.
- Report generation fails after review data has been captured; the pipeline continues with review semantics unchanged, the report-generation error is surfaced, and captured review data remains available for later regeneration.
- User intent is unavailable or unsafe to show.
- Older run data lacks some resolution metadata; the report separates `decision not recorded` / unknown-selection counts from accepted, skipped, selected-for-fix, informational, deferred, and still-open counts.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST produce one run-scoped durable Markdown review-resolution report for every run where the review step produced findings, entered a fix cycle, required human approval, or reached a no-reviewable-changes outcome.
- **FR-002**: The report MUST identify the run, branch, reviewed commits when available, review status, run status, and latest review outcome.
- **FR-003**: The report MUST include a concise purpose statement explaining that it records review findings, decisions, fixes, and remaining risks for later human and agent review.
- **FR-004**: The report MUST list each review finding with severity, location when available, issue title, context, recommended resolution, action type, source, and user instructions when present.
- **FR-005**: The report MUST distinguish agent-produced findings from user-authored findings.
- **FR-006**: The report MUST show which findings were selected for fixing and which were not selected.
- **FR-007**: The report MUST distinguish user-selected fixes from automatically selected fixes and MUST show the decision source/actor plus evidence reference for every selected-for-fix, accepted, skipped, deferred, informational, still-open, and not-recorded resolution decision. The report MUST NOT infer actor or authority from the label alone.
- **FR-008**: The report MUST show each applied review fix summary in chronological order, provide an explicit "fix applied, no summary recorded" label when a fix summary is missing, and never label a recommendation, suggested fix, or proposed resolution as an applied fix.
- **FR-009**: The report MUST show exactly one latest post-fix review result chosen by a deterministic precedence table across run status, review status, final-evidence availability, parseability, and user-decision state; allowed labels are: no issues remain, unresolved findings remain, no reviewable changes, awaiting user decision, final findings unavailable, final findings unreadable, review data inconsistent, or review resolution incomplete.
- **FR-010**: The report MUST include the latest review risk level and risk rationale when available.
- **FR-011**: The report MUST never claim that all review issues are resolved when final review evidence is missing, unreadable, malformed, partially parsed, or internally inconsistent.
- **FR-011A**: When stored review data cannot be trusted because findings, selected finding IDs, fix summaries, source parse results, or summary counts disagree, the report MUST use a fail-closed `review data inconsistent` latest outcome, surface the inconsistency, and omit confident resolved/unresolved totals until the inconsistent records are identified.
- **FR-012**: The report MUST remain available when later pipeline steps fail, the run is cancelled, or the run is superseded.
- **FR-013**: User-facing surfaces that reference the report MUST use plain workflow labels such as `Issue`, `Recommendation`, `Selected for fix`, `Applied fix`, `Still open`, `Accepted`, `Skipped`, and `Risk`; `Accepted` is the canonical label for deliberate risk acceptance and MUST be used only when stored run data contains an explicit human/user risk-acceptance decision.
- **FR-014**: The AXI success/status output and other agent-facing command surfaces MUST include or reference review-resolution information whenever a report exists or review resolution state is non-empty, including successful runs with applied fixes and runs with accepted, skipped, deferred, informational, unresolved, unavailable, or incomplete review outcomes.
- **FR-015**: PR-facing summaries MUST include summary counts, the latest review outcome, material applied-fix summaries, and a reference to the durable report whenever a report exists or review fixes occurred. A material applied-fix summary is any recorded HIGH/CRITICAL fix summary, any summary needed to explain the latest outcome, and a count of omitted lower-severity or unavailable summaries.
- **FR-016**: The report MUST avoid raw agent transcripts, raw logs, secret-bearing data, code excerpts, and diff hunks; it may include only allowlisted, sanitized fields: finding locations, safe finding context, decisions, summaries, safe run intent, or summarized context already intended for user-facing surfaces. Unsafe, secret-bearing, code-like, diff-like, log-like, or transcript-derived values MUST be redacted, summarized, or shown as `unavailable`.
- **FR-017**: The report MUST preserve existing review, approval, auto-fix, push, PR, and CI behavior.
- **FR-018**: Documentation MUST explain when the report is created, how to find it, what each resolution label and latest-outcome label means, and which labels are user-facing renderings versus evidence-state/internal source states.
- **FR-019**: The feature directory MUST retain a source-context reference that records the origin, purpose, and current source-code surfaces used to create this spec.
- **FR-020**: System MUST persist the durable report path or reference with the run metadata so later AXI output, TUI surfaces, PR summaries, regeneration, and future agents can locate the same report; the metadata MUST include generation/update timestamp, source run and round identifiers, generation status, and safe stale/unavailable/error indicators for surfaces that reference it.
- **FR-021**: System MUST maintain a single current report per run, updating it as review decisions, fix attempts, and final outcomes become available while preserving chronological fix history, generation mode, source snapshot timestamp, source run and review/fix round identifiers, and the latest included review/fix event. Regeneration MUST NOT overwrite a report generated from newer evidence and MUST label suspected stale or partial source state.
- **FR-022**: TUI review gate details MUST expose a direct report reference when review resolution occurred or a report exists for the run, and direct TUI/AXI detail outside the report MUST be limited to the report path, summary counts, latest outcome, and sanitized labels unless rendered through the sanitized report content.
- **FR-023**: Reports regenerated from older or partial run data MUST label any missing historical field used by the report as `not recorded` or `unavailable`, including selected finding IDs, user instructions, fix summaries, severity, location, issue title, context, recommended resolution, action type, source, risk level, and risk rationale. The report MUST NOT infer decisions, source, action type, risk, or resolution category from incomplete historical data.
- **FR-024**: If report generation fails after review data has been captured, the pipeline MUST continue with review semantics unchanged, persist a report-generation status with safe error detail, preserve captured review data for later regeneration, and surface whether any last successful report path is stale or unavailable.
- **FR-025**: The report format MUST provide a versioned stable Markdown contract sufficient for targeted validation and future-agent extraction, including exact heading text, required section order, canonical summary count names, and allowed user-facing label values.

### Key Entities

- **Review Resolution Report**: One run-scoped durable Markdown artifact that summarizes review findings, user or automatic decisions, applied fixes, and the latest review outcome for one run; its path/reference is persisted with run metadata and its structure uses stable headings, labels, and summary counts.
- **Review Finding**: A reported review issue or informational note with severity, location, issue title, context, recommended resolution, action type, source, and optional user instructions.
- **Resolution Decision**: The decision made for a finding with an explicit actor/source and evidence reference; allowed states are selected for fix, accepted, skipped, informational, deferred, still open, and decision not recorded. `Accepted` means deliberate risk acceptance and is distinct from skipped, deferred, unresolved work, or missing historical selection data; when multiple facts apply, the report must use the canonical precedence rules rather than inferring a higher-authority decision.
- **Fix Attempt**: A review fix cycle with selected finding IDs, selection source, optional user instructions, agent-reported fix summary, and verification status from the follow-up review result; missing historical fields are shown as `not recorded` or `unavailable` rather than inferred, and an agent-reported fix summary alone never proves the finding was resolved.
- **Run Context**: The branch, commits, status, safe intent summary, review risk information, report reference, and report-generation error state needed to interpret the report.

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: This feature reports review-resolution history only. It must not alter `git push no-mistakes`, review ordering, approval behavior, auto-fix limits, push, PR, or CI semantics.
- **Isolation/User Control**: The report must reflect user-selected findings, user-authored findings, accepted risks, skipped findings, and user instructions without converting them into automatic approval.
- **Evidence Plan**: Planning should include targeted tests for report content, missing/unreadable final review data, partial selections, multiple fix attempts, user-authored findings, agent-facing output, PR-facing summary behavior, and documentation.
- **Agent/Interface Contracts**: Existing structured finding, approval, and fix-summary contracts must remain compatible. Any new report format must be readable by humans and stable enough for future agents.
- **Docs/Generated Artifacts**: User-visible reporting behavior requires docs updates. Generated agent guidance should be updated if agent-facing success or status instructions change.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can identify the original review issue, selected resolution, applied fix, and latest review outcome from a report for a one-fix run in under 60 seconds without reading raw logs.
- **SC-002**: In targeted validation, every review finding from reportable review data appears exactly once in the correct report category.
- **SC-003**: In targeted validation, reports for multiple fix attempts list applied fix summaries in chronological order.
- **SC-004**: In targeted validation, reports never state "no issues remain" unless that outcome comes from a successfully parsed latest review pass for the same run after the relevant fix attempt; missing, unreadable, stale, regenerated, summary-only, or agent-authored fix-summary evidence yields an incomplete or unavailable outcome instead.
- **SC-005**: Successful agent-facing output includes or references review-resolution information for 100% of tested runs where review fixes occurred.
- **SC-006**: PR-facing summaries include a concise review-resolution summary or detailed-report reference for 100% of tested PR runs where review fixes occurred.
- **SC-007**: Existing review approval and auto-fix behavior remains unchanged in targeted regression tests.
- **SC-008**: Documentation explains report creation, discovery, and resolution labels without conflicting with existing pipeline-step behavior.
- **SC-009**: In targeted validation, report-generation failure after captured review data does not change the pipeline outcome, surfaces a report-generation error, and leaves captured review data available for later regeneration.
- **SC-010**: In targeted validation, generated reports and every surface that displays report counts expose the same stable Markdown headings, user-facing labels, and separate summary counts for selected-for-fix, fix attempts, accepted, skipped, informational, deferred, still-open, unavailable, and decision-not-recorded states; aggregate `resolved` counts are absent or explicitly exclude skipped, deferred, informational, accepted-risk, unavailable, and still-open findings.

## Assumptions

- The durable report will be one run-scoped generated Markdown artifact because it is readable by users, friendly to PR review, and suitable as a future-agent reference.
- The report path/reference will be stored with run metadata so AXI output, TUI review gate details, PR summaries, regeneration, and future agents can locate the same artifact.
- Report content will use a single persisted summary-count taxonomy rather than prose-only structure or a separate machine-readable sidecar; required counts include total findings, actionable findings, selected for fix, fix attempts, applied fix summaries, accepted, skipped, informational, deferred, still open, unavailable, and decision not recorded, all derived from the same report/run metadata source.
- The first version is scoped to the review step only; fixes from test, document, lint, PR, or CI may remain in their existing summaries.
- Reports are generated for review runs with findings, fixes, approvals, or no-reviewable-changes outcomes; routine runs with a clean review and no findings do not need a detailed artifact unless referenced by another user-facing surface.
- Existing review results are the source of report truth; this feature should not require a new review output contract.
- Safe run intent may be shown when already available to user-facing pipeline surfaces, but it is contextual only: it must never populate resolution decisions, user instructions, selected finding IDs, accepted-risk labels, or approval decisions. Raw transcripts and raw logs remain out of scope.
- Reports must not include code excerpts or diff hunks; report generation must use an explicit source allowlist that excludes raw diff events, raw logs, and transcripts, so finding locations, safe finding context, decisions, and summaries provide traceability without copying adjacent raw event payloads.
- Report regeneration should be possible from stored run state when live events are missed, and missing historical fields should be labeled rather than inferred.
