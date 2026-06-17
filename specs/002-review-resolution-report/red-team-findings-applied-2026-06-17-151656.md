# Red Team Findings: Review Resolution Report

Status: ARCHIVED
**Applied:** 2026-06-17-151656

Session: RT-002-review-resolution-report-2026-06-17
Target: `specs/002-review-resolution-report/spec.md`
Feature directory: `specs/002-review-resolution-report`
Invocation: `/speckit.red-team.run specs/002-review-resolution-report/spec.md --yes`
Date: 2026-06-17
Maintainer: Codex
Selection method: auto (`--yes`)

Selected lenses:

- Agent Contract Integrity Adversary
- Partial Evidence Recovery Adversary
- Privacy and Transcript Exposure Adversary
- Review Trust-Boundary Adversary
- User Surface Misrepresentation Adversary

Supporting context:

- `specs/002-review-resolution-report/source-context.md`
- `specs/002-review-resolution-report/checklists/requirements.md`
- `.specify/memory/constitution.md`

## 1. Session Summary

Applied and archived. This red-team pass found 25 findings: 1 CRITICAL, 11 HIGH, 12 MEDIUM, and 1 LOW. The strongest risks cluster around fail-closed data integrity, report contract stability, explicit decision provenance, sanitized report inputs, and stale or incomplete review evidence.

## 2. Findings

| ID | Lens | Severity | Location | Finding | Suggested Resolution | Status |
| --- | --- | --- | --- | --- | --- | --- |
| F-RT-002-review-resolution-report-2026-06-17-001 | Agent Contract Integrity Adversary | CRITICAL | FR-009/FR-011/FR-023/SC-002 | The spec prevents confident "all resolved" claims when final review evidence is missing or unreadable, but it does not require a fail-closed integrity state when stored review rounds are malformed, partially parsed, or internally inconsistent. A report could still present confident summary counts or a latest outcome while the per-finding list, selected IDs, fix summaries, or stats disagree. | Add a required data-integrity outcome such as "review data inconsistent" and require the report to surface source parse errors, count mismatches, and omitted records before any resolved/unresolved summary. Define validation rules that compare findings, selected IDs, fix attempts, and summary counts. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-002 | Agent Contract Integrity Adversary | HIGH | FR-025/SC-010/Key Entities: Review Resolution Report | The stable extraction contract is only described as "stable Markdown headings, user-facing labels, and summary counts" without naming the exact headings, count keys, required sections, ordering, or version. Future agents and tests can silently drift by matching different labels or interpreting changed headings as equivalent. | Define a versioned report contract with exact heading text, required section order, canonical summary count names, and allowed label values. Require tests to assert those exact anchors. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-003 | Agent Contract Integrity Adversary | HIGH | FR-008/Key Entities: Fix Attempt/source-context.md Data Flow | The spec calls fix summaries "applied review fix summaries", but the source context says fix mode returns a one-line fix summary from an agent. Without a required evidence link from each summary to selected finding IDs and follow-up review results, an agent-written summary can be treated as proof that a fix was applied even if it was only attempted or incompletely verified. | Separate "fix attempted", "agent-reported fix summary", and "verified resolved by follow-up review" fields. Require each fix attempt to record selected finding IDs, summary source, and verification status. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-004 | Partial Evidence Recovery Adversary | HIGH | FR-009 / Clarifications lines 24-25 / Edge Cases lines 105-109 | The latest-outcome labels are enumerated, but the spec does not define precedence rules for overlapping partial-evidence states. For example, a cancelled, failed, or superseded run after fixes may also have unreadable final findings, stale pre-fix findings, or no persisted final round; without a decision table, implementations can choose a reassuring label such as "no issues remain" or a vague label inconsistently. | Add a latest-outcome precedence table mapping run status, review status, final-evidence availability, parseability, and user-decision state to exactly one label. Make "review resolution incomplete" mandatory whenever trustworthy post-fix evidence is absent after a fix attempt, including superseded runs. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-005 | Partial Evidence Recovery Adversary | HIGH | FR-021 / FR-023 / Assumptions line 185 / source-context Data Flow lines 56-65 | Regeneration from stored run state is required, but the spec does not prevent regenerated output from overwriting a newer live-updated report or silently diverging from missed IPC events. The source context explicitly has both live IPC events and stored rounds/run info, yet the spec does not require snapshot identity, round IDs, timestamps, or freshness checks in the report. | Require report metadata to include generation mode, source snapshot timestamp, run/step round identifiers, and latest included review/fix event. Regeneration must not overwrite a report generated from newer evidence and must label any suspected stale or partial source state. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-006 | Privacy and Transcript Exposure Adversary | HIGH | FR-004 / FR-016 | FR-004 requires the report to include finding context, recommended resolution, source, and user instructions, while FR-016 only says to avoid raw transcripts, logs, secrets, code excerpts, and diff hunks. The spec does not define how these report fields are sanitized, even though review findings and user instructions can naturally contain copied code, diff snippets, logs, or secrets. | Define an explicit allowlist and sanitizer for each report field. Require unsafe or untrusted values to be redacted, summarized, or replaced with `unavailable`. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-007 | Privacy and Transcript Exposure Adversary | HIGH | User Story 2 Acceptance Scenario 3 / FR-016 / Key Entities: Run Context | The spec permits a safe intent summary when user intent is available, but does not define the source boundary for that summary. This conflicts with the constitution's requirement that transcript-derived intent be treated as untrusted, redact secrets, and avoid storing raw transcript text. | Specify that intent may only come from an already-sanitized user-facing intent field, never raw transcript text. Add required redaction and an `unsafe or unavailable` fallback. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-008 | Privacy and Transcript Exposure Adversary | HIGH | source-context.md Data Flow / FR-016 | The source context notes that executor and IPC paths emit findings, diff, and fix-summary events, while FR-016 forbids diff hunks in the report. The spec does not require the report generator to avoid diff event payloads or use a strict source allowlist, so implementation could accidentally persist raw diff content from adjacent live event data. | Add a source allowlist for report generation that excludes raw diff events, raw logs, and transcripts. Require tests that inject diff-like content and prove it is not emitted. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-009 | Review Trust-Boundary Adversary | HIGH | FR-013 / Key Entities: Resolution Decision / User Story 1 Acceptance Scenario 3 | The spec makes `Accepted` the canonical label for deliberate risk acceptance, but does not require an explicit human/user acceptance event before a finding may be labeled Accepted. A report generator or automation could convert an unselected finding into Accepted, implying approval that was never granted. | Require `Accepted` only when stored run data contains an explicit user/human risk-acceptance decision. Otherwise label unselected actionable findings as skipped, deferred, still open, not recorded, or unavailable according to evidence. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-010 | Review Trust-Boundary Adversary | HIGH | FR-004, FR-005, FR-006, FR-007 / Key Entities: Resolution Decision and Fix Attempt | The spec requires finding source and distinguishes user-selected fixes from automatically selected fixes, but it does not require every resolution decision to carry actor/provenance. Accepted, skipped, deferred, informational, still-open, selected-for-fix, and user-instruction fields could be misattributed between user, automation, reviewer agent, report generator, and regenerated historical data. | Add a required `decision source/actor` and evidence reference for every per-finding resolution decision and fix attempt. Prohibit deriving the actor from the label or from report-generation heuristics. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-011 | User Surface Misrepresentation Adversary | HIGH | FR-014/FR-015/FR-022 | Surface exposure is conditioned mostly on successful runs with applied review fixes or PRs where review fixes occurred. A run can still have accepted, skipped, deferred, informational, unresolved, unavailable, or incomplete review outcomes without an applied fix, allowing success/status, TUI, or PR surfaces to omit the report reference and make the run look cleaner than it is. | Require AXI, TUI, and PR/user-facing summaries to expose the report reference whenever a report exists or review resolution state is non-empty, not only when fixes were applied. Require those surfaces to show unresolved/incomplete/accepted-risk state alongside success labels. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-012 | User Surface Misrepresentation Adversary | HIGH | FR-015/FR-025/SC-002/SC-010 | The spec requires summary counts but does not define a canonical count taxonomy or source of truth across the report, run metadata, TUI, AXI, and PR summaries. Counts such as total findings, actionable findings, selected for fix, applied fixes, accepted, skipped, informational, deferred, still open, and unavailable can diverge while still satisfying the current wording. | Define required count fields, meanings, and derivation rules once, then require every surface that displays counts to use that same persisted report/run metadata source. Add validation that cross-surface counts match for representative runs. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-013 | Agent Contract Integrity Adversary | MEDIUM | FR-004/FR-006/FR-013/Key Entities: Resolution Decision | The spec mixes original finding fields such as action type with later resolution states such as selected for fix, accepted, skipped, informational, deferred, and still open. It does not define a canonical per-finding state machine or precedence rules when a finding is selected, fixed, reappears, accepted, or lacks historical selected IDs. | Add a canonical decision enum with definitions, allowed transitions, source of decision, and precedence rules for regenerated reports. Include an explicit unknown or not-recorded state rather than overloading skipped or accepted. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-014 | Agent Contract Integrity Adversary | MEDIUM | FR-015/SC-006 | PR summaries must include "material applied-fix summaries", but materiality is undefined and can depend on an agent summary that omits important fix details. This creates a silent omission path where the durable report may contain fix history but the PR-facing extraction drops details without a detectable rule violation. | Define materiality deterministically, such as including every HIGH or CRITICAL fix and count of any omitted lower-severity fixes. Require PR summaries to include total fix-attempt counts and a report reference whenever any details are summarized rather than listed. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-015 | Partial Evidence Recovery Adversary | MEDIUM | FR-023 / FR-004 / FR-010 / Key Entities lines 147-150 | The partial-history rule only names missing selected finding IDs, user instructions, and fix summaries, while the report also depends on severity, location, title, context, recommended resolution, action type, source, risk level, and risk rationale. Missing source or action fields are especially dangerous because they affect whether a finding is agent-produced, user-authored, informational, actionable, accepted, or still open. | Expand FR-023 to cover every report field derived from historical review data. For each missing field, require an explicit "not recorded" or "unavailable" label and prohibit deriving source, action type, risk, or resolution category from adjacent fields. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-016 | Partial Evidence Recovery Adversary | MEDIUM | FR-006 / FR-013 / FR-023 / SC-002 | The spec requires every finding to appear in the correct category and to show selected versus not selected, but older data may lack selected finding IDs. In that case, the current labels risk forcing unknown findings into accepted, skipped, fixed, or still-open counts, which would infer decisions that FR-023 forbids. | Add an explicit unknown-selection or decision-not-recorded category for regenerated partial data, with summary counts that separate unknown decisions from accepted, skipped, selected-for-fix, and unresolved findings. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-017 | Partial Evidence Recovery Adversary | MEDIUM | FR-024 / FR-020 / FR-014 / FR-015 / FR-022 | Report-generation failure is covered, but user-facing surfaces are not told how to behave when no durable report path exists or an older report path exists after a later generation failure. AXI, TUI, and PR summaries could omit the failure, point to a stale report, or imply the durable report is current. | Require run metadata to persist report generation status, error detail safe for display, last successful report path if any, and whether that report is stale. AXI, TUI, and PR surfaces should display "report unavailable" or "report stale" instead of a normal report reference when generation failed. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-018 | Privacy and Transcript Exposure Adversary | MEDIUM | FR-015 / User Story 4 Acceptance Scenarios 2-3 | PR summaries must include material applied-fix summaries and may summarize multiple fix rounds, but the spec only says not to duplicate the full report body. It does not limit PR-visible fix summaries to sanitized, non-secret, non-code content, creating a second exposure surface outside the durable report. | Constrain PR summaries to counts, latest outcome, sanitized one-line fix summaries, and the report reference. Require PR summaries to use the same sanitizer as the report. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-019 | Privacy and Transcript Exposure Adversary | MEDIUM | FR-014 / FR-022 | AXI and TUI surfaces must include or expose review-resolution information, but the spec does not define which fields may be duplicated directly versus only referenced. This leaves room for agent-facing output or TUI details to copy sensitive finding context, user instructions, or fix summaries instead of linking to the sanitized report. | Define a minimal direct-display contract for AXI and TUI: report path, summary counts, latest outcome, and sanitized labels only. Require detailed sensitive-prone fields to stay behind the sanitized report renderer. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-020 | Review Trust-Boundary Adversary | MEDIUM | FR-014, FR-015, FR-025 / User Story 3 and User Story 4 | Agent-facing output and PR summaries must include summary counts and latest outcome, but the spec does not require counts to be separated by fixed, accepted, skipped, deferred, informational, and still-open categories. A later surface could collapse non-fixed states into a generic resolved count and make skipped, deferred, accepted, or informational findings look like applied fixes. | Require all summary counts in AXI, TUI, PR, and report headings to preserve the resolution categories separately. Ban aggregate labels such as `resolved` unless they explicitly exclude skipped, deferred, informational, and still-open findings. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-021 | Review Trust-Boundary Adversary | MEDIUM | FR-009, FR-011 / Data Flow To Preserve step 6 | The spec says the report must show the latest post-fix review result and must not claim all issues are resolved when evidence is missing or unreadable, but it does not define the authoritative evidence source for `no issues remain`. An agent-authored fix summary, PR summary, or report regeneration could be mistaken for final review evidence. | Define `no issues remain` as valid only from a successfully parsed latest review pass for the same run after the relevant fix attempt. Require the report to show the evidence source and fall back to incomplete/unavailable states for summaries or stale data. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-022 | Review Trust-Boundary Adversary | MEDIUM | FR-016 / Assumptions: safe run intent / Constitution Alignment: Isolation/User Control | The report may include safe run intent, but the spec does not explicitly prevent safe intent summaries from being treated as user instructions or approval decisions. Transcript-derived or summarized intent is lower-authority context and could be confused with an actual TUI/AXI approval event. | State that safe run intent is contextual only and must never populate resolution decisions, user instructions, selected finding IDs, or accepted-risk labels. Require actual approval surfaces or stored response data for those fields. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-023 | User Surface Misrepresentation Adversary | MEDIUM | FR-009/FR-013 | The latest outcome labels include phrases such as `final findings unavailable`, `final findings unreadable`, and `review resolution incomplete`, while FR-013 separately lists user-facing workflow labels. The spec does not say which outcome labels are user-facing, which are internal states, or how they should be rendered in AXI/TUI/PR surfaces, creating room for internal concepts to leak or for users to confuse incomplete evidence with accepted risk. | Create a canonical user-facing outcome-label table with definitions and allowed renderings for report, AXI, TUI, and PR surfaces. Distinguish evidence-state labels from decision labels such as `Accepted`, `Skipped`, and `Still open`. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-024 | User Surface Misrepresentation Adversary | MEDIUM | FR-020/FR-021/FR-024 | The spec says the report path is persisted and one current report is updated, but it does not define when the reference becomes trustworthy or what surfaces must show during report generation failure. A surface could show a stale report reference, omit that generation failed, or summarize newer review state while linking an older report. | Require report metadata to include generation/update timestamp, source run/round identifiers, and generation error state. Require every surface that references the report to indicate when the report is unavailable, stale, or failed to generate. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-17-025 | User Surface Misrepresentation Adversary | LOW | FR-004/FR-008/FR-013/source-context Planning Warnings | The spec requires `recommended resolution` and `applied fix summary`, but only source-context warns not to conflate suggested fixes with applied fixes. Without making this distinction explicit in the functional requirements and label definitions, user surfaces may label recommendations as solutions or fixes and imply work was performed. | Add a requirement that recommendations/proposed fixes must never be labeled as applied fixes. Define separate labels for `Recommendation` and `Applied fix` across report, TUI, AXI, documentation, and PR summaries. | spec-fix |

## 3. Resolutions Log

Resolution categories: `spec-fix`, `new-OQ`, `accepted-risk`, `out-of-scope`, or `skipped`.

### F-RT-002-review-resolution-report-2026-06-17-001

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:128` says "The report MUST never claim that all review issues are resolved when final review evidence is missing or unreadable," but it does not cover malformed, partially parsed, or internally inconsistent stored data, so the finding's premise holds.
    Evidence: `internal/db/stats.go:155` derives reported counts from parsed findings while `internal/db/round.go:23` stores selected IDs separately, so count, ID, fix-summary, and parse-state disagreement is a real contract shape the report must handle.
    Category choice: `spec-fix`, not `new-OQ`, because the spec already establishes a fail-closed truthfulness rule and the local data model shows the fields that need consistency checks.
    Long-term vs band-aid: rejected band-aid of hiding counts only after a parser error; the durable fix is a report-level integrity outcome that fails closed whenever the source records disagree.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-011**: The report MUST never claim that all review issues are resolved when final review evidence is missing or unreadable.
  After: |
    - **FR-011**: The report MUST never claim that all review issues are resolved when final review evidence is missing, unreadable, malformed, partially parsed, or internally inconsistent.
    - **FR-011A**: When stored review data cannot be trusted because findings, selected finding IDs, fix summaries, source parse results, or summary counts disagree, the report MUST use a fail-closed `review data inconsistent` latest outcome, surface the inconsistency, and omit confident resolved/unresolved totals until the inconsistent records are identified.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:128-129

### F-RT-002-review-resolution-report-2026-06-17-002

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:142` says the report format must provide "stable Markdown headings, user-facing labels, and summary counts," but it does not name exact headings, section order, count keys, labels, or a version, so the finding's premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:173` requires tests and future agents to locate headings, labels, and counts "without parsing prose-only content," which needs concrete anchors rather than a loose stability promise.
    Category choice: `spec-fix`, not `skipped`, because the current text is directionally correct but underspecified for the stated extraction contract.
    Long-term vs band-aid: rejected band-aid of telling tests to use fuzzy label matching; the durable fix is a versioned Markdown contract with exact anchors.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-025**: The report format MUST provide stable Markdown headings, user-facing labels, and summary counts sufficient for targeted validation and future-agent extraction.
  After: |
    - **FR-025**: The report format MUST provide a versioned stable Markdown contract sufficient for targeted validation and future-agent extraction, including exact heading text, required section order, canonical summary count names, and allowed user-facing label values.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:143

### F-RT-002-review-resolution-report-2026-06-17-003

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:125` requires "each applied review fix summary," but `specs/002-review-resolution-report/source-context.md:60` says fix mode returns "a one-line fix summary," so the premise holds that a summary can be mistaken for proof of resolution.
    Evidence: `internal/db/round.go:30` defines `FixSummary` as "the agent's one-line commit summary" while `specs/002-review-resolution-report/source-context.md:61` separately names the follow-up review as the latest findings and risk state.
    Category choice: `spec-fix`, not `new-OQ`, because the local contract already separates selected IDs, agent-reported summaries, and follow-up review evidence.
    Long-term vs band-aid: rejected band-aid of renaming the summary label only; the durable fix records selected IDs, summary source, and verification status as distinct fix-attempt facts.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **Fix Attempt**: A review fix cycle with selected findings, decision source, optional user instructions, and applied fix summary; missing historical fields are shown as `not recorded` or `unavailable` rather than inferred.
  After: |
    - **Fix Attempt**: A review fix cycle with selected finding IDs, selection source, optional user instructions, agent-reported fix summary, and verification status from the follow-up review result; missing historical fields are shown as `not recorded` or `unavailable` rather than inferred, and an agent-reported fix summary alone never proves the finding was resolved.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:150

### F-RT-002-review-resolution-report-2026-06-17-004

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:126` enumerates latest outcome labels, and `specs/002-review-resolution-report/spec.md:107` says cancelled or failed runs after review fixes use "review resolution incomplete," but there is no precedence rule when multiple partial-evidence states overlap, so the premise holds.
    Evidence: `specs/002-review-resolution-report/clarifications-applied-2026-06-17-145000.md:101` chose "review resolution incomplete" for missing trustworthy final evidence, while `specs/002-review-resolution-report/spec.md:105` also names missing or unparsable latest review data.
    Category choice: `spec-fix`, not `accepted-risk`, because inconsistent outcome selection is directly within this feature's reporting contract and cheap to specify now.
    Long-term vs band-aid: rejected band-aid of letting implementation choose the first matching label; the durable fix is a deterministic precedence table that yields exactly one outcome.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-009**: The report MUST show the latest post-fix review result as one of: no issues remain, unresolved findings remain, no reviewable changes, awaiting user decision, final findings unavailable, final findings unreadable, or review resolution incomplete.
  After: |
    - **FR-009**: The report MUST show exactly one latest post-fix review result chosen by a deterministic precedence table across run status, review status, final-evidence availability, parseability, and user-decision state; allowed labels are: no issues remain, unresolved findings remain, no reviewable changes, awaiting user decision, final findings unavailable, final findings unreadable, review data inconsistent, or review resolution incomplete.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:126

### F-RT-002-review-resolution-report-2026-06-17-005

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:138` requires "a single current report per run" and `specs/002-review-resolution-report/spec.md:185` says regeneration should work from stored run state, but neither requires snapshot identity, round IDs, timestamps, or freshness checks, so the premise holds.
    Evidence: `specs/002-review-resolution-report/source-context.md:56` through `specs/002-review-resolution-report/source-context.md:63` names a flow from structured findings to stored rounds to IPC events and rendered surfaces, which is exactly where stale live-vs-stored divergence can occur.
    Category choice: `spec-fix`, not `new-OQ`, because the feature already chose one current report and stored-run regeneration; the missing piece is the metadata needed to preserve that contract.
    Long-term vs band-aid: rejected band-aid of never regenerating once a live report exists; the durable fix records source snapshot identity and refuses to overwrite newer evidence silently.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-021**: System MUST maintain a single current report per run, updating it as review decisions, fix attempts, and final outcomes become available while preserving chronological fix history.
  After: |
    - **FR-021**: System MUST maintain a single current report per run, updating it as review decisions, fix attempts, and final outcomes become available while preserving chronological fix history, generation mode, source snapshot timestamp, source run and review/fix round identifiers, and the latest included review/fix event. Regeneration MUST NOT overwrite a report generated from newer evidence and MUST label suspected stale or partial source state.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:139

### F-RT-002-review-resolution-report-2026-06-17-006

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:121` says the report lists finding "context, recommended resolution, action type, source, and user instructions," while `specs/002-review-resolution-report/spec.md:133` only says it must avoid raw transcripts, logs, secrets, code excerpts, and diff hunks, so the premise holds that sanitization is undefined.
    Evidence: `internal/types/findings.go:28` through `internal/types/findings.go:33` shows finding description, context, suggested fix, source, and user instructions are structured fields that could carry unsafe text unless the report applies a field-level sanitizer.
    Category choice: `spec-fix`, not `accepted-risk`, because privacy-safe report rendering is a direct requirement of this milestone and not a later hardening concern.
    Long-term vs band-aid: rejected band-aid of redacting only obvious secret strings; the durable fix is an explicit allowlist plus sanitizer behavior for every report field.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-016**: The report MUST avoid raw agent transcripts, raw logs, secret-bearing data, code excerpts, and diff hunks; it may include finding locations, safe finding context, decisions, summaries, safe run intent, or summarized context already intended for user-facing surfaces.
  After: |
    - **FR-016**: The report MUST avoid raw agent transcripts, raw logs, secret-bearing data, code excerpts, and diff hunks; it may include only allowlisted, sanitized fields: finding locations, safe finding context, decisions, summaries, safe run intent, or summarized context already intended for user-facing surfaces. Unsafe, secret-bearing, code-like, diff-like, log-like, or transcript-derived values MUST be redacted, summarized, or shown as `unavailable`.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:134

### F-RT-002-review-resolution-report-2026-06-17-007

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:62` says the report includes the "safe intent summary" when user intent is available, and `specs/002-review-resolution-report/spec.md:133` allows safe run intent, but neither defines the source boundary, so the premise holds.
    Evidence: `.specify/memory/constitution.md:70` says transcript-derived intent must be treated as untrusted and `.specify/memory/constitution.md:71` requires transcript readers to redact secrets and avoid raw transcript storage.
    Category choice: `spec-fix`, not `new-OQ`, because the constitution already answers the authority boundary: raw transcript text is not an acceptable source for the report.
    Long-term vs band-aid: rejected band-aid of adding a generic "safe" adjective; the durable fix restricts intent to an already-sanitized user-facing intent field with an unavailable fallback.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    3. **Given** user intent is available for a run, **When** the report is viewed, **Then** it includes the safe intent summary or explicitly states that no safe intent summary is available.
  After: |
    3. **Given** user intent is available for a run, **When** the report is viewed, **Then** it includes a safe intent summary only from an already-sanitized user-facing intent field, or explicitly states that no safe intent summary is available.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:62

### F-RT-002-review-resolution-report-2026-06-17-008

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:133` forbids "code excerpts, and diff hunks," while the spec's source context says live and stored data include adjacent diff/finding/fix-summary events, so the premise holds that a source allowlist is missing.
    Evidence: `internal/ipc/protocol.go:241` defines `Findings` for step events and `internal/ipc/protocol.go:242` defines `Diff` as "unified diff for fix_review events," which the report must not copy.
    Category choice: `spec-fix`, not `skipped`, because the concern is not hypothetical broad archaeology; it follows from the local IPC contract cited by the feature context.
    Long-term vs band-aid: rejected band-aid of stripping lines that look like diff hunks after rendering; the durable fix excludes raw diff, log, and transcript sources from report generation.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - Reports must not include code excerpts or diff hunks; finding locations, safe finding context, decisions, and summaries provide traceability.
  After: |
    - Reports must not include code excerpts or diff hunks; report generation must use an explicit source allowlist that excludes raw diff events, raw logs, and transcripts, so finding locations, safe finding context, decisions, and summaries provide traceability without copying adjacent raw event payloads.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:185

### F-RT-002-review-resolution-report-2026-06-17-009

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:130` says "`Accepted` is the canonical label for deliberate risk acceptance," but it does not require an explicit user/human acceptance event before applying that label, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:45` groups accepted, skipped, informational, and deferred as possible not-selected outcomes, while `internal/db/round.go:88` through `internal/db/round.go:90` records selected findings and whether the selection came from the user or auto-fix, not a blanket acceptance decision.
    Category choice: `spec-fix`, not `new-OQ`, because the spec already defines Accepted as deliberate risk acceptance and the constitution requires user control for approval-like decisions.
    Long-term vs band-aid: rejected band-aid of treating every unselected item as accepted; the durable fix requires stored human/user acceptance evidence or a lower-authority label.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-013**: User-facing surfaces that reference the report MUST use plain workflow labels such as `Issue`, `Recommendation`, `Selected for fix`, `Applied fix`, `Still open`, `Accepted`, `Skipped`, and `Risk`; `Accepted` is the canonical label for deliberate risk acceptance.
  After: |
    - **FR-013**: User-facing surfaces that reference the report MUST use plain workflow labels such as `Issue`, `Recommendation`, `Selected for fix`, `Applied fix`, `Still open`, `Accepted`, `Skipped`, and `Risk`; `Accepted` is the canonical label for deliberate risk acceptance and MUST be used only when stored run data contains an explicit human/user risk-acceptance decision.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:131

### F-RT-002-review-resolution-report-2026-06-17-010

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:124` distinguishes user-selected fixes from automatically selected fixes, but the cited requirements do not require actor/provenance for accepted, skipped, deferred, informational, still-open, or not-recorded decisions, so the premise holds.
    Evidence: `internal/db/round.go:88` through `internal/db/round.go:90` records selection source for fixed findings, and `internal/types/findings.go:16` through `internal/types/findings.go:19` records finding source, which proves provenance already exists locally but is not required for every report decision.
    Category choice: `spec-fix`, not `skipped`, because the proposed provenance requirement extends an existing local contract instead of inventing a new approval system.
    Long-term vs band-aid: rejected band-aid of deriving actor from labels such as Accepted or Skipped; the durable fix records decision source/actor and evidence reference per resolution decision.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-007**: The report MUST distinguish user-selected fixes from automatically selected fixes.
  After: |
    - **FR-007**: The report MUST distinguish user-selected fixes from automatically selected fixes and MUST show the decision source/actor plus evidence reference for every selected-for-fix, accepted, skipped, deferred, informational, still-open, and not-recorded resolution decision. The report MUST NOT infer actor or authority from the label alone.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:124

### F-RT-002-review-resolution-report-2026-06-17-011

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:131` limits agent-facing output to successful runs that "applied review fixes," and `specs/002-review-resolution-report/spec.md:132` limits PR summaries to when "review fixes occurred," so the premise holds for AXI/PR even though TUI is partly broader at `specs/002-review-resolution-report/spec.md:139`.
    Evidence: `specs/002-review-resolution-report/clarifications-applied-2026-06-17-145000.md:63` selected AXI success/status, TUI review gate details, and PR summaries "when review resolution occurred," which is broader than only applied fixes.
    Category choice: `spec-fix`, not `out-of-scope`, because exposing the durable report reference on the named surfaces is explicitly part of this feature's scope.
    Long-term vs band-aid: rejected band-aid of adding a warning only to successful fix runs; the durable fix exposes report state whenever report data exists or review-resolution state is non-empty.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-014**: The AXI success/status output and other agent-facing command surfaces MUST include or reference review-resolution information whenever a successful run applied review fixes.
  After: |
    - **FR-014**: The AXI success/status output and other agent-facing command surfaces MUST include or reference review-resolution information whenever a report exists or review resolution state is non-empty, including successful runs with applied fixes and runs with accepted, skipped, deferred, informational, unresolved, unavailable, or incomplete review outcomes.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:132

### F-RT-002-review-resolution-report-2026-06-17-012

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:142` requires "summary counts" and `specs/002-review-resolution-report/spec.md:173` requires future agents to locate them, but no canonical count taxonomy or source of truth is defined, so the premise holds.
    Evidence: `internal/ipc/protocol.go:205` and `internal/ipc/protocol.go:206` expose only reported/fixed finding counts today, which is narrower than the report's accepted, skipped, deferred, unavailable, and still-open states.
    Category choice: `spec-fix`, not `new-OQ`, because the existing report purpose names the exact resolution facts users need and no product policy decision is needed to keep those counts separate.
    Long-term vs band-aid: rejected band-aid of recomputing counts independently in every surface; the durable fix defines one persisted taxonomy derived from the report/run metadata source.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - Report content will use stable Markdown headings, user-facing labels, and summary counts rather than prose-only structure or a separate machine-readable sidecar.
  After: |
    - Report content will use a single persisted summary-count taxonomy rather than prose-only structure or a separate machine-readable sidecar; required counts include total findings, actionable findings, selected for fix, fix attempts, applied fix summaries, accepted, skipped, informational, deferred, still open, unavailable, and decision not recorded, all derived from the same report/run metadata source.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:180

### F-RT-002-review-resolution-report-2026-06-17-013

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:148` defines Resolution Decision as "selected for fix, accepted, skipped, informational, deferred, or still open," but it does not define precedence when selected, fixed, reappearing, accepted, or missing-selection facts overlap, so the premise holds.
    Evidence: `internal/types/findings.go:9` through `internal/types/findings.go:13` defines original finding actions separately from later resolution decisions, which confirms the report needs an explicit decision state rather than overloading action type.
    Category choice: `spec-fix`, not `skipped`, because this closes a real contract gap between original finding fields and report resolution states.
    Long-term vs band-aid: rejected band-aid of mapping unknown historical data to Skipped or Accepted; the durable fix adds a canonical decision enum with actor, evidence, and not-recorded handling.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **Resolution Decision**: The decision made for a finding: selected for fix, accepted, skipped, informational, deferred, or still open. `Accepted` means deliberate risk acceptance and is distinct from skipped, deferred, or unresolved work.
  After: |
    - **Resolution Decision**: The decision made for a finding with an explicit actor/source and evidence reference; allowed states are selected for fix, accepted, skipped, informational, deferred, still open, and decision not recorded. `Accepted` means deliberate risk acceptance and is distinct from skipped, deferred, unresolved work, or missing historical selection data; when multiple facts apply, the report must use the canonical precedence rules rather than inferring a higher-authority decision.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:149

### F-RT-002-review-resolution-report-2026-06-17-014

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:132` requires "material applied-fix summaries," but it never defines materiality, so the premise holds.
    Evidence: `internal/db/round.go:44` through `internal/db/round.go:45` returns one fix-summary entry per fix round in round order, so the local contract can support deterministic inclusion and omission counts.
    Category choice: `spec-fix`, not `new-OQ`, because the spec already chose bounded PR summaries rather than full report duplication, and a deterministic materiality rule is enough.
    Long-term vs band-aid: rejected band-aid of letting each PR renderer choose what feels material; the durable fix defines material summaries and requires omitted-detail counts plus a report reference.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-015**: PR-facing summaries MUST include summary counts, the latest review outcome, material applied-fix summaries, and a reference to the durable report when review fixes occurred.
  After: |
    - **FR-015**: PR-facing summaries MUST include summary counts, the latest review outcome, material applied-fix summaries, and a reference to the durable report whenever a report exists or review fixes occurred. A material applied-fix summary is any recorded HIGH/CRITICAL fix summary, any summary needed to explain the latest outcome, and a count of omitted lower-severity or unavailable summaries.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:133

### F-RT-002-review-resolution-report-2026-06-17-015

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:140` limits missing historical-field handling to "selected finding IDs, user instructions, or fix summaries," while `specs/002-review-resolution-report/spec.md:121` requires many more per-finding fields, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:127` also requires latest review risk level and rationale when available, which are not covered by the current partial-history rule.
    Category choice: `spec-fix`, not `accepted-risk`, because regenerated historical reports are an explicit edge case for this milestone and the structural fix is just to cover every report-derived field.
    Long-term vs band-aid: rejected band-aid of filling missing source/action/risk from adjacent fields; the durable fix labels every missing historical report field and forbids inference of source, action, risk, or resolution category.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-023**: Reports regenerated from older or partial run data MUST label missing selected finding IDs, user instructions, or fix summaries as `not recorded` or `unavailable` and MUST NOT infer decisions from incomplete historical data.
  After: |
    - **FR-023**: Reports regenerated from older or partial run data MUST label any missing historical field used by the report as `not recorded` or `unavailable`, including selected finding IDs, user instructions, fix summaries, severity, location, issue title, context, recommended resolution, action type, source, risk level, and risk rationale. The report MUST NOT infer decisions, source, action type, risk, or resolution category from incomplete historical data.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:141

### F-RT-002-review-resolution-report-2026-06-17-016

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:123` requires the report to show selected versus not selected findings, and `specs/002-review-resolution-report/spec.md:140` forbids inferring decisions from incomplete historical data, but no explicit unknown-selection category is defined, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:109` says missing historical selected IDs are labeled `not recorded` or `unavailable`, which requires those unknowns to stay separate from accepted, skipped, fixed, or still-open counts.
    Category choice: `spec-fix`, not `skipped`, because the current spec contains the no-inference principle but lacks the durable user-facing category that preserves it.
    Long-term vs band-aid: rejected band-aid of placing unknowns into Skipped or Still open; the durable fix adds decision-not-recorded/unknown-selection counts.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - Older run data lacks some resolution metadata.
  After: |
    - Older run data lacks some resolution metadata; the report separates `decision not recorded` / unknown-selection counts from accepted, skipped, selected-for-fix, informational, deferred, and still-open counts.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:112

### F-RT-002-review-resolution-report-2026-06-17-017

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:141` says report-generation failure must surface an error and preserve captured data, but it does not define stale path or unavailable-report behavior for user-facing surfaces, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:137` persists the report path for AXI, TUI, PR summaries, regeneration, and future agents, which means those surfaces need safe metadata when the latest generation failed.
    Category choice: `spec-fix`, not `accepted-risk`, because showing stale/unavailable report state is necessary to prevent misleading references in the first implementation milestone.
    Long-term vs band-aid: rejected band-aid of printing a transient console error only; the durable fix persists generation status, safe error detail, and stale/unavailable report state.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-024**: If report generation fails after review data has been captured, the pipeline MUST continue with review semantics unchanged, surface a report-generation error, and preserve captured review data for later regeneration.
  After: |
    - **FR-024**: If report generation fails after review data has been captured, the pipeline MUST continue with review semantics unchanged, persist a report-generation status with safe error detail, preserve captured review data for later regeneration, and surface whether any last successful report path is stale or unavailable.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:142

### F-RT-002-review-resolution-report-2026-06-17-018

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:93` says PR summaries should summarize multiple fix rounds "without exposing excessive raw round detail," and `specs/002-review-resolution-report/spec.md:133` sanitizes the report, but the PR summary exposure boundary is not explicit, so the premise holds.
    Evidence: `internal/ipc/protocol.go:245` exposes `FixSummaries` to event consumers and `internal/ipc/protocol.go:241` through `internal/ipc/protocol.go:242` keeps findings and diff event data nearby, so PR rendering needs the same sanitization boundary.
    Category choice: `spec-fix`, not `out-of-scope`, because PR summaries are one of the named report-reference surfaces for this feature.
    Long-term vs band-aid: rejected band-aid of truncating PR text length; the durable fix requires sanitized PR-visible summaries and excludes raw code, diff, log, transcript, secret, or unsafe text.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    2. **Given** multiple review fix rounds occurred, **When** the PR summary is generated, **Then** it summarizes the chain without exposing excessive raw round detail.
  After: |
    2. **Given** multiple review fix rounds occurred, **When** the PR summary is generated, **Then** it summarizes the sanitized chain without exposing excessive raw round detail, raw code, diff hunks, logs, transcripts, secrets, or unsafe fix-summary text.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:93

### F-RT-002-review-resolution-report-2026-06-17-019

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:139` says TUI review gate details expose a direct report reference, and `specs/002-review-resolution-report/spec.md:131` says agent-facing surfaces include or reference review-resolution information, but neither says which fields may be duplicated outside the sanitized report, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:133` puts the no-raw-transcript/no-diff boundary on the report itself, leaving direct AXI/TUI detail display as an unclosed exposure path.
    Category choice: `spec-fix`, not `accepted-risk`, because limiting direct-display fields is the smallest durable way to preserve the privacy boundary across required surfaces.
    Long-term vs band-aid: rejected band-aid of relying on each surface to remember the report sanitizer; the durable fix defines a minimal direct-display contract and keeps sensitive-prone details behind sanitized report rendering.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-022**: TUI review gate details MUST expose a direct report reference when review resolution occurred or a report exists for the run.
  After: |
    - **FR-022**: TUI review gate details MUST expose a direct report reference when review resolution occurred or a report exists for the run, and direct TUI/AXI detail outside the report MUST be limited to the report path, summary counts, latest outcome, and sanitized labels unless rendered through the sanitized report content.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:140

### F-RT-002-review-resolution-report-2026-06-17-020

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:76` and `specs/002-review-resolution-report/spec.md:92` require review-resolution information and PR review fixes, but no requirement separates fixed, accepted, skipped, deferred, informational, still-open, unavailable, and not-recorded counts, so the premise holds.
    Evidence: `internal/db/stats.go:155` through `internal/db/stats.go:157` calculates reported and fixed counts, which is useful but insufficient for the report's richer resolution labels.
    Category choice: `spec-fix`, not `skipped`, because the feature's whole purpose is to avoid hiding accepted, skipped, or unresolved review outcomes behind cleaner summaries.
    Long-term vs band-aid: rejected band-aid of adding one generic `resolved` total; the durable fix preserves categories separately and bans misleading aggregate resolved counts.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **SC-010**: In targeted validation, generated reports expose stable Markdown headings, user-facing labels, and summary counts that tests and future agents can locate without parsing prose-only content.
  After: |
    - **SC-010**: In targeted validation, generated reports and every surface that displays report counts expose the same stable Markdown headings, user-facing labels, and separate summary counts for selected-for-fix, fix attempts, accepted, skipped, informational, deferred, still-open, unavailable, and decision-not-recorded states; aggregate `resolved` counts are absent or explicitly exclude skipped, deferred, informational, accepted-risk, unavailable, and still-open findings.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:174

### F-RT-002-review-resolution-report-2026-06-17-021

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:126` allows "no issues remain" and `specs/002-review-resolution-report/spec.md:128` forbids that claim when final evidence is missing or unreadable, but the authoritative evidence source is not defined, so the premise holds.
    Evidence: `specs/002-review-resolution-report/source-context.md:61` says the follow-up review produces the latest findings and risk state, while `internal/pipeline/pipeline.go:45` through `internal/pipeline/pipeline.go:49` says fix summaries are one-line agent summaries, not review evidence.
    Category choice: `spec-fix`, not `new-OQ`, because the feature's data flow already identifies follow-up review as the source of truth.
    Long-term vs band-aid: rejected band-aid of trusting an agent-authored fix or PR summary when it sounds conclusive; the durable fix makes no-issues-remain valid only from a parsed latest review pass for the same run after the relevant fix.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **SC-004**: In targeted validation, reports never state "no issues remain" when the latest review evidence is missing or unreadable.
  After: |
    - **SC-004**: In targeted validation, reports never state "no issues remain" unless that outcome comes from a successfully parsed latest review pass for the same run after the relevant fix attempt; missing, unreadable, stale, regenerated, summary-only, or agent-authored fix-summary evidence yields an incomplete or unavailable outcome instead.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:168

### F-RT-002-review-resolution-report-2026-06-17-022

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:183` says "Safe run intent may be shown," but it does not explicitly prevent that context from populating user instructions, selected IDs, accepted-risk labels, or approval decisions, so the premise holds.
    Evidence: `.specify/memory/constitution.md:46` through `.specify/memory/constitution.md:54` requires human control for approval-like decisions, and `.specify/memory/constitution.md:70` says transcript-derived intent is untrusted context.
    Category choice: `spec-fix`, not `skipped`, because the current spec's allowance for safe intent needs the authority boundary that the constitution already requires.
    Long-term vs band-aid: rejected band-aid of adding a disclaimer in prose only; the durable fix states safe intent is contextual and cannot populate decision, instruction, selection, risk-acceptance, or approval fields.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - Safe run intent may be shown when already available to user-facing pipeline surfaces; raw transcripts and raw logs remain out of scope.
  After: |
    - Safe run intent may be shown when already available to user-facing pipeline surfaces, but it is contextual only: it must never populate resolution decisions, user instructions, selected finding IDs, accepted-risk labels, or approval decisions. Raw transcripts and raw logs remain out of scope.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:184

### F-RT-002-review-resolution-report-2026-06-17-023

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:126` lists latest-outcome labels such as final findings unavailable and review resolution incomplete, while `specs/002-review-resolution-report/spec.md:130` lists workflow labels such as Accepted and Skipped, but it does not define which are user-facing renderings versus evidence states, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:135` requires documentation to explain resolution labels, which is the natural place to add latest-outcome label definitions and renderings.
    Category choice: `spec-fix`, not `new-OQ`, because the labels are already chosen by the spec; the missing work is classification and rendering rules.
    Long-term vs band-aid: rejected band-aid of renaming only the most awkward labels in one surface; the durable fix requires docs and surfaces to distinguish outcome/evidence labels from decision labels.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-018**: Documentation MUST explain when the report is created, how to find it, and what each resolution label means.
  After: |
    - **FR-018**: Documentation MUST explain when the report is created, how to find it, what each resolution label and latest-outcome label means, and which labels are user-facing renderings versus evidence-state/internal source states.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:136

### F-RT-002-review-resolution-report-2026-06-17-024

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:137` persists the report path/reference and `specs/002-review-resolution-report/spec.md:138` maintains one current report, but neither defines when the reference is trustworthy after generation failure or stale regeneration, so the premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:141` requires report-generation errors to surface and captured data to remain available, which implies surfaces need generation status rather than a bare path.
    Category choice: `spec-fix`, not `accepted-risk`, because stale report references would directly undermine the user-facing report contract in this milestone.
    Long-term vs band-aid: rejected band-aid of deleting old report paths on failure; the durable fix persists generation/update timestamp, source IDs, status, and stale/unavailable/error indicators.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-020**: System MUST persist the durable report path or reference with the run metadata so later AXI output, TUI surfaces, PR summaries, regeneration, and future agents can locate the same report.
  After: |
    - **FR-020**: System MUST persist the durable report path or reference with the run metadata so later AXI output, TUI surfaces, PR summaries, regeneration, and future agents can locate the same report; the metadata MUST include generation/update timestamp, source run and round identifiers, generation status, and safe stale/unavailable/error indicators for surfaces that reference it.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:138

### F-RT-002-review-resolution-report-2026-06-17-025

- Category: spec-fix
- Payload:
  Reasoning: |
    Verification: `specs/002-review-resolution-report/spec.md:121` requires "recommended resolution" and `specs/002-review-resolution-report/spec.md:125` requires "applied review fix summary," but only the source context explicitly warns not to conflate them, so the premise holds.
    Evidence: `specs/002-review-resolution-report/source-context.md:85` says "Do not conflate `suggested_fix` / `Recommendation` with an applied fix," and `internal/types/findings.go:30` stores `SuggestedFix` separately from fix summaries.
    Category choice: `spec-fix`, not `skipped`, because the spec has the right labels but lacks the invariant that prevents user surfaces from implying work was performed.
    Long-term vs band-aid: rejected band-aid of changing one UI label from Suggested fix to Solution; the durable fix forbids recommendations, suggested fixes, or proposed resolutions from being labeled as applied fixes anywhere the report contract is rendered.
  Target: specs/002-review-resolution-report/spec.md
  Before: |
    - **FR-008**: The report MUST show each applied review fix summary in chronological order and provide an explicit "fix applied, no summary recorded" label when a fix summary is missing.
  After: |
    - **FR-008**: The report MUST show each applied review fix summary in chronological order, provide an explicit "fix applied, no summary recorded" label when a fix summary is missing, and never label a recommendation, suggested fix, or proposed resolution as an applied fix.
- Status: applied
- Applied-at: 2026-06-17T15:16:56+07:00
- Downstream-ref: specs/002-review-resolution-report/spec.md:125

## 5. Session Metadata

```yaml
session_id: RT-002-review-resolution-report-2026-06-17
target: specs/002-review-resolution-report/spec.md
feature_directory: specs/002-review-resolution-report
date: 2026-06-17
invocation: "/speckit.red-team.run specs/002-review-resolution-report/spec.md --yes"
requested_feature_directory_argument: specs/002-review-resolution-report
target_spec_path_used: specs/002-review-resolution-report/spec.md
selection_method: auto
yes_used: true
matched_triggers:
  - ai_llm
  - contracts
  - immutability_audit
  - multi_party
  - regulatory_path
preflight:
  before_speckit_red_team_run_hooks: none
  constitution_red_team_trigger_criteria: absent_bootstrap_mode
  lens_catalog: .specify/extensions/red-team/red-team-lenses.yml
supporting_context:
  - specs/002-review-resolution-report/source-context.md
  - specs/002-review-resolution-report/checklists/requirements.md
  - .specify/memory/constitution.md
lenses:
  Agent Contract Integrity Adversary:
    status: completed
    findings_retained: 5
  Partial Evidence Recovery Adversary:
    status: completed
    findings_retained: 5
  Privacy and Transcript Exposure Adversary:
    status: completed
    findings_retained: 5
  Review Trust-Boundary Adversary:
    status: completed
    findings_retained: 5
  User Surface Misrepresentation Adversary:
    status: completed
    findings_retained: 5
counts:
  total: 25
  by_severity:
    CRITICAL: 1
    HIGH: 11
    MEDIUM: 12
    LOW: 1
  by_lens:
    Agent Contract Integrity Adversary: 5
    Partial Evidence Recovery Adversary: 5
    Privacy and Transcript Exposure Adversary: 5
    Review Trust-Boundary Adversary: 5
    User Surface Misrepresentation Adversary: 5
resolution_counts:
  unresolved: 0
  spec_fix: 25
  new_OQ: 0
  accepted_risk: 0
  out_of_scope: 0
  skipped: 0
notes:
  - The installed command requires a target spec path; the target was derived from the requested feature directory by appending spec.md.
  - The shipped red-team lens template was replaced with repo-specific lenses before dispatch because the placeholder catalog had insufficient matching lens coverage for this feature.
apply:
  applied_at: 2026-06-17T15:16:56+07:00
  applied_by: Codex
  resolutions:
    spec_fix: 25
    new_OQ: 0
    accepted_risk: 0
    out_of_scope: 0
    skipped: 0
  unresolved: 0
  allow_historical_edits: true
  historical_edits_applied:
    - F-RT-002-review-resolution-report-2026-06-17-001:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-002:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-003:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-004:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-005:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-006:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-007:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-008:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-009:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-010:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-011:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-012:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-013:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-014:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-015:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-016:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-017:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-018:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-019:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-020:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-021:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-022:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-023:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-024:specs/002-review-resolution-report/spec.md
    - F-RT-002-review-resolution-report-2026-06-17-025:specs/002-review-resolution-report/spec.md
```
