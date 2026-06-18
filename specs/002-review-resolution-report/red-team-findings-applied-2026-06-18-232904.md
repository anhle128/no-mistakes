# Red Team Findings: Review Resolution Report

Session ID: `RT-002-review-resolution-report-2026-06-18`  
Target: `specs/002-review-resolution-report/spec.md`  
Date: 2026-06-18  
Maintainer: Codex  
Selection method: auto (`--yes`)  
Matched triggers: `ai_llm`, `immutability_audit`, `multi_party`, `contracts`  
Selected lenses: Agent Contract Integrity Adversary; Partial Evidence Recovery Adversary; Privacy and Transcript Exposure Adversary; Review Trust-Boundary Adversary; User Surface Misrepresentation Adversary  
Supporting context: `.specify/memory/constitution.md`  
Wall-clock: not recorded
Status: ARCHIVED
**Applied:** 2026-06-18-232904

## 1. Session Summary

Applied 23 `spec-fix` resolutions to `specs/002-review-resolution-report/spec.md` with `--allow-historical-edits`; skipped 2 findings whose payloads conflicted with, or were already covered by, mandatory requirements.

## 2. Findings

| ID | Lens | Severity | Location | Finding | Suggested Resolution | Status |
| --- | --- | --- | --- | --- | --- | --- |
| F-RT-002-review-resolution-report-2026-06-18-001 | Agent Contract Integrity Adversary | HIGH | FR-012 through FR-016 / lines 152-156 | The new fix-agent `resolutions[]` contract is underspecified: it names fields but does not require schema validation, duplicate-ID rejection, selected-finding coverage checks, per-field length limits, or explicit handling for unknown finding IDs. Because legacy `summary` remains accepted, malformed or partial structured output can silently fall back to inferred text while still producing a polished report. | Define a versioned response schema with validation rules: required fields when `resolutions[]` is present, unique `finding_id`, allowed unknown/duplicate behavior, selected-ID coverage reporting, and explicit degraded-source labels when validation fails. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-002 | Agent Contract Integrity Adversary | HIGH | FR-008 / line 148 | The spec allows an issue to be classified `Resolved` when the Review step completes cleanly after a fix round, even without requiring parsed follow-up Review evidence that the same normalized finding disappeared. If follow-up structured review output is missing, malformed, or lacks comparable IDs, this can turn absence of evidence into confident resolution. | Require a valid parsed follow-up Review result with comparable normalized IDs for automatic resolution; otherwise classify the issue as `Still Open` or `verification-inconclusive` unless an explicit human/pipeline acceptance action exists. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-003 | Agent Contract Integrity Adversary | HIGH | FR-012, FR-014, Key Entities / lines 152, 154, 172-174 | `applied_solution` and `why_this_solution` are too ambiguous to prove that a fix was actually applied rather than proposed, attempted, skipped as illegitimate, or only summarized by the agent. The report can therefore attribute material fix details to the agent without a per-finding applied/attempted/no-op status or verification basis. | Extend each resolution entry with explicit outcome fields such as `status`, `evidence_source`, `verification_summary`, and optional `not_applied_reason`; only render applied-solution language when `status` confirms an applied change and evidence exists. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-004 | Partial Evidence Recovery Adversary | HIGH | FR-020 / line 160; Edge Cases / line 135 | The spec requires refreshes when Review reaches approve/skip/abort/failure, but it does not cover cancellation or supersession while Review is waiting, fixing, or between persisted events, nor stale-run recovery after a daemon crash. Because FR-001 creates a report immediately after first findings, an interrupted run can leave earlier counts/status visible as if they are the latest durable outcome. | Require report finalization/reconciliation on any run transition to failed, cancelled, superseded, or stale-recovered after Review findings exist. If terminal Review evidence cannot be reconstructed, force unresolved findings to Still Open and mark the report status as incomplete or evidence-unavailable. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-005 | Partial Evidence Recovery Adversary | HIGH | FR-009 / line 149; FR-020 / line 160 | Accepted status depends on a user or pipeline approve/skip decision, but the spec does not require that decision, actor/source, timestamp, affected finding IDs, or approval reason to be persisted. A live refresh could classify accepted issues correctly, while later regeneration from stored rounds may only see completed/skipped step state and infer acceptance incorrectly or lose the reason. | Add a persisted Review terminal-decision record or round metadata for approve/skip actions, including source, affected finding IDs, timestamp, and reason when available. Require regenerated reports to label acceptance evidence as unavailable and leave findings Still Open if that record is absent. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-006 | Partial Evidence Recovery Adversary | HIGH | FR-017 / line 157; FR-023 / line 163; Key Entities / line 171 | The metadata `surface-facing status` is underspecified, so AXI/TUI/PR summaries can publish compact counts without distinguishing terminal, in-progress, incomplete, stale, verification-inconclusive, or evidence-unavailable states. This is especially risky when final Review evidence is absent but a report exists from an earlier refresh. | Define an explicit status enum and transition rules for report metadata, including incomplete/evidence-unavailable states. Require PR/TUI/AXI labels to use those states instead of presenting unresolved counts as final outcomes. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-007 | Privacy and Transcript Exposure Adversary | HIGH | FR-011 / line 151 and FR-024 / line 164 | FR-011 requires preserving original finding descriptions, risk rationales, and user instructions, all of which may be agent- or transcript-derived and may contain secrets, raw logs, code excerpts, or copied diff hunks. FR-024 requires sanitization but does not define the source boundary, redaction rules, or which fields must be dropped versus summarized, leaving implementers to preserve unsafe raw content while believing they satisfied the report requirement. | Define a field-level sanitization contract for original finding fields: redact secret patterns, strip raw transcript/log/diff/code blocks, escape Markdown controls, and store only bounded summaries for user instructions and rationale. Add tests proving raw transcript text, diff hunk markers, code fences, and common secret formats are not emitted. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-008 | Privacy and Transcript Exposure Adversary | HIGH | FR-015 / line 155 and User Story 3 / lines 101-103 | The spec requires inferred applied-solution details from the relevant fix commit diff when structured resolution details are missing. It forbids raw diff hunks in FR-024, but it does not prohibit or constrain code-level excerpts, secret-bearing literals, or near-verbatim paraphrases derived from the diff, so the fallback path can still leak sensitive implementation content into the durable report. | Require diff-derived fallback text to be a high-level sanitized summary only, with changed-file paths and commit SHA as evidence, not code snippets or literals. Add explicit redaction and summarization tests for secret-bearing diffs and large code hunks. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-009 | Review Trust-Boundary Adversary | HIGH | FR-009 / lines 149 and 125 | The spec allows a finding to become `Not Fixed / Accepted` after a `user or pipeline action approves, skips, or otherwise accepts` it, and says unselected findings become accepted after an approve or skip decision. `pipeline action`, `skip`, and `otherwise accepts` are not authority-scoped, so an automated skip, configured skipped step, or lower-authority workflow transition could be rendered as acceptance without explicit user approval. | Define the exact acceptance-authority events that may produce `Accepted`, including actor type and source surface. Treat automated skips, canceled/superseded runs, deferred/unselected findings, and preconfigured skipped steps as separate non-accepted states unless tied to an explicit user or documented policy decision. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-010 | Review Trust-Boundary Adversary | HIGH | FR-011, FR-017 / lines 151 and 157 | The spec preserves original finding source but does not require preserving outcome-decision provenance: who selected it, who approved/skipped it, which agent claimed the fix, or which review round verified it. Without per-entry decision provenance, reports and PR summaries can misattribute acceptance or resolution between the user, automation, the review agent, and the fix agent. | Add per-resolution-entry fields for decision_actor, decision_source, action, timestamp, round_id, selection_source, and evidence reference. Require the Markdown report and metadata consumers to label user, pipeline policy, review-agent, and fix-agent decisions distinctly. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-011 | User Surface Misrepresentation Adversary | HIGH | FR-017, FR-022, FR-023 / lines 157, 162-163 | The spec requires a `surface-facing status` and `compact status` but never defines the allowed status values or how they are derived from resolved, accepted, and still-open counts. This lets AXI, TUI, or PR summaries use success-like wording such as `review resolution report generated` even when still-open findings remain. | Define an explicit status enum and derivation rule, with still-open findings taking precedence over any success wording. Require all user-facing surfaces to render non-success language whenever `still_open_count` is greater than zero. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-012 | User Surface Misrepresentation Adversary | HIGH | FR-017, FR-020, SC-001, SC-007 / lines 157, 160, 189, 195 | The report, SQLite metadata, AXI/TUI output, and PR summary can all expose counts, but the spec does not require a consistency invariant across those surfaces. It tests report inclusion and PR rendering separately, leaving room for stale metadata or mismatched counts after refreshes, aborts, or failed writes. | Add a requirement that metadata counts must be generated from the same classified entry set as the Markdown report and that AXI, TUI, and PR summaries must read that same persisted snapshot. Add an integration test that compares report section counts, metadata counts, AXI/TUI detail output, and PR summary output for the same run. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-013 | Agent Contract Integrity Adversary | MEDIUM | FR-017, FR-020, SC-007 / lines 157, 160, 195 | The SQLite metadata requirement stores aggregate counts and a surface-facing status, but does not require consistency checks against the per-issue report entries or a degraded/partial state. AXI, TUI, and PR summaries could display confident resolved/accepted/still-open counts from stale or partially generated metadata even when the Markdown report is missing entries or failed mid-refresh. | Require metadata refresh to be atomic with report generation and include integrity fields such as report version, entry count, source round range, last refresh result, and a `partial/degraded` status that surfaces instead of confident counts when consistency checks fail. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-014 | Agent Contract Integrity Adversary | MEDIUM | Requirements Carried Forward, FR-007, User Story 3 / lines 50, 147, 101-102 | The spec requires stable high-level headings but does not define a complete Markdown extraction contract for heading levels, exact field labels, section order, count derivation, or versioning. Future agents that parse the human report could break silently if labels like `Applied Solution Source` or outcome headings drift. | Add a canonical report template with version marker, fixed heading hierarchy, exact labels, count semantics, and golden-file tests that fail on incompatible heading or label drift. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-015 | Partial Evidence Recovery Adversary | MEDIUM | FR-011 / line 151; Origin lines 30-35 and carried-forward line 50 | The original source lists `context` and `suggested_fix`, and the carried-forward structure includes proposed solution before fix, but FR-011 only requires ID, severity, file, line, action, source, description, risk fields, and user instructions. Missing historical fields could silently disappear instead of being labeled unavailable or not recorded. | Extend FR-011 and the report entry schema to include original context and suggested/proposed fix when present. When legacy findings lack those fields, render explicit `not recorded` or `unavailable in historical data` markers. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-016 | Partial Evidence Recovery Adversary | MEDIUM | Assumptions / line 203; FR-017 / line 157; FR-020 / line 160 | The spec says existing round history is the foundation but does not define an authoritative regeneration source, freshness marker, or atomic relationship between Markdown and SQLite metadata. Regenerating from live events, latest step findings, or partially updated metadata can diverge from persisted Review rounds and produce mismatched counts or stale report content. | Specify that regeneration must rebuild from persisted Review step rounds plus persisted decision/fix metadata, not transient events or latest findings alone. Store a round watermark or content hash with metadata and require Markdown plus metadata updates to be atomic or clearly marked stale on mismatch. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-017 | Privacy and Transcript Exposure Adversary | MEDIUM | FR-012 through FR-014 / lines 152-154 | The new fix-agent `summary`, `applied_solution`, and `why_this_solution` fields are trusted as preferred report content, but agents often echo prompts, transcripts, logs, or diffs in structured responses. The spec does not require schema validation beyond shape, per-field maximums, redaction, or rejection of raw transcript/log/diff content before these fields are written to Markdown. | Extend the response contract with per-field length caps and content rules that ban raw transcript/log/diff/code blocks and require sanitization before persistence. Treat structured agent output as untrusted input even when it matches the schema. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-018 | Privacy and Transcript Exposure Adversary | MEDIUM | Acceptance Scenario 1.2 / line 70, Edge Case / line 129, FR-011 / line 151 | The report may include approval reasons, user-authored findings, and user instructions, but the spec does not distinguish explicit user-provided text from transcript-derived intent summaries or unavailable inferred intent. This creates a path for private conversation details or speculative user intent to be recorded as durable report evidence. | Require user-facing report text to label the source of any user reason or instruction as explicit approval input, explicit AXI/TUI finding, or unavailable. Forbid deriving approval reasons or user intent summaries from raw transcripts unless they pass the same redaction and bounded-summary rules. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-019 | Review Trust-Boundary Adversary | MEDIUM | FR-007 to FR-010 / lines 147-150 | The three required buckets force every finding into resolved, accepted, or still open, but the source model includes informational or no-op findings that may not need a fix or approval. A later surface could therefore render informational/no-action items as accepted or resolved fixes, overstating both user approval and remediation. | Add an explicit `Informational / No Action Required` or equivalent non-fix, non-acceptance status. Require AXI/TUI/PR counts to keep informational and deferred findings separate from resolved and accepted findings. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-020 | Review Trust-Boundary Adversary | MEDIUM | FR-008 / line 148 | `Resolved` may be assigned when the Review step completes cleanly after a relevant fix round, even without requiring a concrete evidence record for the specific finding. A clean completion can be produced by a changed review scope, skipped follow-up, or agent omission, which risks treating lower-authority absence-of-reporting as proof of fix. | Require each resolved entry to cite a specific verification event: follow-up review round ID, exact normalized finding ID absence, scope equivalence, and whether the verifier was an agent or user. If that evidence is missing or scope changed, classify the item as verification-inconclusive or still open. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-021 | Review Trust-Boundary Adversary | MEDIUM | FR-015 and FR-019 / lines 155 and 159 | The spec requires inferring applied-solution details from the relevant fix round and commit diff when structured details are missing, while also allowing multiple findings to share one fix round or commit. In multi-finding commits, per-finding inferred solution text can easily attribute the wrong change or rationale to a finding. | When structured `resolutions[]` data is absent and a commit covers multiple findings, label the evidence as commit-level inferred evidence, not finding-specific solution detail. Only render per-finding applied-solution claims when structured mapping or exact verification evidence supports the mapping. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-022 | User Surface Misrepresentation Adversary | MEDIUM | FR-001, FR-004, FR-020, FR-022 / lines 141, 144, 160, 162 | The report is created immediately after first Review findings, then refreshed later, but the spec only requires a generated timestamp and does not distinguish preliminary, refreshed, and finalized report states. During an active or interrupted run, AXI/TUI could show the report path and counts without making clear whether the report is final or still evolving. | Persist and surface lifecycle fields such as `report_state`, `first_generated_at`, `last_refreshed_at`, and `finalized_at`. Specify AXI/TUI wording for in-progress, aborted, failed, superseded, and terminal successful runs. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-023 | User Surface Misrepresentation Adversary | MEDIUM | FR-007 through FR-010, Constitution Alignment / lines 147-150, 182 | The label `Not Fixed / Accepted Issues` compresses approve, skip, and other acceptance paths into one user-facing bucket. Users may read `accepted` as reviewed and safe even when the underlying action was a skip or pipeline acceptance without a fix. | Use a clearer label such as `Accepted Without Fix` and require each entry to show the acceptance action and reason. If skip has materially different semantics, give skipped findings a distinct label or substatus. | spec-fix |
| F-RT-002-review-resolution-report-2026-06-18-024 | Privacy and Transcript Exposure Adversary | LOW | FR-004 / line 144, FR-017 / line 157, FR-022 / line 162 | The spec requires storing and displaying repo path/local report path in the report metadata and AXI/TUI run detail surfaces. Even without full report narrative, local paths can expose usernames, client names, private repository names, or sensitive directory structure, and the spec does not say when to prefer a repo identifier over a full path. | Prefer sanitized repo identifiers in Run Context and metadata, and only show full local paths in explicitly local-only surfaces where needed to open the file. Define path redaction rules for AXI output that may be copied into logs or automation. | skipped |
| F-RT-002-review-resolution-report-2026-06-18-025 | User Surface Misrepresentation Adversary | LOW | Acceptance Scenario 1.3 and FR-022 / lines 71, 162 | The user story says AXI/TUI should provide the local report path or compact status, while FR-022 requires compact status, counts, and local report path. The weaker acceptance wording could lead implementers to omit either the path or counts, reducing traceability from user-facing output to the durable report. | Align the acceptance scenario with FR-022 by requiring AXI and TUI run detail surfaces to show status, counts, and local report path whenever a report exists. | skipped |

## 3. Resolutions Log

### F-RT-002-review-resolution-report-2026-06-18-001

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The spec says the fix-agent contract "allow[s] a short `summary` plus optional `resolutions[]` entries" with four named fields, and the finding's premise holds because no validation, duplicate, unknown-ID, or selected-coverage rule is stated in FR-012 through FR-016.
    Evidence: `specs/002-review-resolution-report/spec.md:152` names `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`; `internal/pipeline/steps/common_fix.go:29-35` shows the current schema only requires `summary`, so the new structured contract must be specified rather than inferred.
    Why this category over alternatives: This is not `new-OQ` because the local contract points to a straightforward compatibility-preserving schema extension, and not `skipped` because the spec is genuinely silent on validation.
    Long-term vs band-aid: A band-aid would silently ignore malformed entries and fall back to diff inference; the durable fix is to make invalid structured data visible as degraded evidence while keeping legacy `summary` compatibility.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-012**: System MUST extend the Review fix agent structured response contract to allow a short `summary` plus optional `resolutions[]` entries containing `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`.
  ```
  After:
  ```markdown
  - **FR-012**: System MUST extend the Review fix agent structured response contract to allow a short `summary` plus optional `resolutions[]` entries containing `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`.
    - When `resolutions[]` is present, the system MUST validate it as untrusted structured data: each entry must have a non-empty `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`; `finding_id` values must be unique; unknown or duplicate IDs must not be used as finding-specific evidence; and missing selected-finding coverage must be recorded as degraded structured evidence rather than silently treated as a complete agent report.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-002

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-008 currently says an issue can be `Resolved` when "the Review step completes cleanly after the relevant fix round without ambiguous matching", so the finding correctly identifies a path where clean completion can substitute for comparable follow-up evidence.
    Evidence: `specs/002-review-resolution-report/spec.md:148` contains that clean-completion clause, while `specs/002-review-resolution-report/spec.md:127` already says changed IDs must be "verification-inconclusive or still open" instead of silently resolved.
    Why this category over alternatives: This is not `new-OQ` because the spec's own edge case already defines the safe answer for ambiguous matching, and not `accepted-risk` because the fix is a narrow invariant.
    Long-term vs band-aid: A band-aid would add a defensive guard at rendering time; the durable fix is to define the resolution classification rule so every producer and surface shares the same evidence threshold.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-008**: System MUST classify an issue as `Resolved` only when available follow-up Review evidence indicates that the same finding is no longer present, or when the Review step completes cleanly after the relevant fix round without ambiguous matching.
  ```
  After:
  ```markdown
  - **FR-008**: System MUST classify an issue as `Resolved` only when a valid parsed follow-up Review result over an equivalent Review scope has comparable normalized IDs and shows that the same finding is no longer present, or when a persisted verification event explicitly records equivalent human/pipeline confirmation for that finding. Clean Review completion without comparable parsed follow-up evidence is verification-inconclusive and MUST leave the issue in `Still Open Issues`.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-003

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-014 says the report must prefer matching `resolutions[]` data, but it does not say that agent text is only descriptive evidence and cannot prove the fix outcome by itself.
    Evidence: `specs/002-review-resolution-report/spec.md:154` says to "prefer matching `resolutions[]` data"; `internal/pipeline/steps/review.go:51-58` currently asks the fix agent to verify work but only returns a short summary, so the new fields must not become authoritative without persisted fix and verification evidence.
    Why this category over alternatives: This is not `skipped` because the ambiguity is real, and not `new-OQ` because the invariant is clear: agent-provided prose is untrusted explanation, not the source of truth for applied status.
    Long-term vs band-aid: A band-aid would add cautious wording to the Markdown renderer; the durable fix is to bind structured agent details to actual fix-round and verification evidence before rendering applied-solution language.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-014**: System MUST prefer matching `resolutions[]` data for applied solution and rationale when a resolution entry maps to the finding ID.
  ```
  After:
  ```markdown
  - **FR-014**: System MUST prefer matching validated `resolutions[]` data for applied solution and rationale when a resolution entry maps to the finding ID, but only as descriptive fix-agent evidence. The report MUST render those fields as applied-solution language only when the entry is tied to persisted fix-round evidence and a resolved or verified-attempt outcome; otherwise it MUST label the entry as attempted, not applied, or evidence unavailable as appropriate.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-004

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-020 lists first-recorded findings, fix rounds, approve/skip/abort/failure outcomes, and PR summary use, but it does not include cancellation, supersession, or stale-run recovery.
    Evidence: `specs/002-review-resolution-report/spec.md:160` omits those transitions even though `specs/002-review-resolution-report/spec.md:135` says canceled and superseded runs keep metadata tied to the run ID; local recovery code marks stale running runs failed in `internal/db/run.go:204-231`.
    Why this category over alternatives: This is not `new-OQ` because local lifecycle states are known (`failed`, `cancelled`, stale recovery), and not `accepted-risk` because an interrupted report is a core trust problem for this feature.
    Long-term vs band-aid: A band-aid would patch only explicit user aborts; the durable fix is to reconcile the report on every terminal or recovered run transition after Review findings exist.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-020**: System MUST update or refresh the report when Review findings are first recorded, after Review fix rounds, when the Review step reaches approve/skip/abort/failure outcomes, and before PR summary generation uses report metadata.
  ```
  After:
  ```markdown
  - **FR-020**: System MUST update or refresh the report when Review findings are first recorded, after Review fix rounds, when the Review step reaches approve/skip/abort/failure outcomes, when a run with Review findings becomes failed, cancelled, superseded, or stale-recovered, and before PR summary generation uses report metadata. If terminal Review evidence cannot be reconstructed during reconciliation, unresolved findings MUST remain `Still Open` and the report status MUST indicate incomplete or evidence-unavailable state.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-005

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-009 requires acceptance after a user or pipeline action, but the spec does not require that action to be persisted with actor, source, timestamp, affected IDs, or reason.
    Evidence: `specs/002-review-resolution-report/spec.md:149` is the only acceptance rule; local `approvalResponse` in `internal/pipeline/executor.go:26-31` carries action and IDs only in memory, and the persisted `step_rounds` schema in `internal/db/schema.go:40-52` has no terminal-decision column.
    Why this category over alternatives: This is not `new-OQ` because the existing flow already defines approve, skip, fix, and abort actions, and not `skipped` because regeneration from persisted data would otherwise be under-specified.
    Long-term vs band-aid: A band-aid would infer acceptance from completed or skipped step status; the durable fix is to persist the acceptance decision itself and require reports to treat absent decision evidence as still open.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-009**: System MUST classify an issue as `Not Fixed / Accepted` only after a user or pipeline action approves, skips, or otherwise accepts the issue without fixing it.
  ```
  After:
  ```markdown
  - **FR-009**: System MUST classify an issue as `Accepted Without Fix` only after a persisted Review terminal-decision record or equivalent round metadata proves that a user or documented pipeline policy approved, skipped, or otherwise accepted the issue without fixing it. That record MUST include action, actor/source, timestamp, affected finding IDs, and reason when available; if it is absent during regeneration, unresolved findings MUST stay `Still Open` with acceptance evidence unavailable.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-006

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-017 requires a "surface-facing status" but does not define allowed values or transition rules, so the finding's premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:157` names path, timestamp, counts, and status; `specs/002-review-resolution-report/spec.md:163` separately says PR content uses compact counts/status, so status semantics must be shared before surfaces render it.
    Why this category over alternatives: This is not `new-OQ` because the required states follow directly from the spec's lifecycle: in-progress, finalized, incomplete, stale/degraded, and evidence-unavailable.
    Long-term vs band-aid: A band-aid would make each UI choose wording independently; the durable fix is a single persisted enum that all surfaces consume.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-017**: System MUST persist compact report metadata in a dedicated SQLite `review_resolution_reports` table keyed by `run_id`, sufficient for AXI, TUI, and PR summary code to know whether a report exists, where it is located locally, when it was generated, its surface-facing status, and how many issues are resolved, accepted, and still open.
  ```
  After:
  ```markdown
  - **FR-017**: System MUST persist compact report metadata in a dedicated SQLite `review_resolution_reports` table keyed by `run_id`, sufficient for AXI, TUI, and PR summary code to know whether a report exists, where it is located locally, when it was generated, its surface-facing status, and how many issues are resolved, accepted, informational, and still open.
    - The metadata status MUST use an explicit enum with transition rules covering at least `in_progress`, `final`, `incomplete`, `stale`, `degraded`, and `evidence_unavailable`; AXI, TUI, and PR summaries MUST render this enum rather than deriving their own success wording from counts alone.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-007

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-011 requires preserving finding descriptions, risk rationale, and user instructions, while FR-024 only gives a generic sanitization requirement, so the reviewer is right that field-level source boundaries are missing.
    Evidence: `specs/002-review-resolution-report/spec.md:151` lists original finding fields; `specs/002-review-resolution-report/spec.md:164` bans raw transcripts and diff hunks but does not say how to redact or summarize each field.
    Why this category over alternatives: This is not `skipped` because the spec creates a real preservation-versus-privacy tension, and not `new-OQ` because the conservative long-term rule is to treat all such content as untrusted.
    Long-term vs band-aid: A band-aid would truncate suspicious text after rendering; the durable fix is a field-level sanitization contract before persistence and Markdown generation.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-024**: System MUST sanitize and bound report content sourced from agent output, user instructions, findings, diffs, and commit metadata to avoid malformed Markdown control content, accidental raw transcript storage, raw diff hunks, and unbounded report size; truncation MUST use explicit markers.
  ```
  After:
  ```markdown
  - **FR-024**: System MUST sanitize and bound report content sourced from agent output, user instructions, findings, diffs, and commit metadata to avoid malformed Markdown control content, accidental raw transcript storage, raw logs, raw code blocks, raw diff hunks, common secret formats, and unbounded report size; truncation MUST use explicit markers.
    - Sanitization MUST be field-level: preserve IDs, severity, action, source, file, and line as structured fields; redact secret-like values; escape Markdown controls; strip or summarize raw transcript/log/diff/code blocks; and store bounded summaries for descriptions, risk rationale, user instructions, applied solution, and reasons.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-008

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-015 requires inference from commit diff when structured details are missing, and FR-024 bans raw diff hunks, but the spec does not constrain code-level excerpts or near-verbatim literals from that diff.
    Evidence: `specs/002-review-resolution-report/spec.md:155` requires diff-derived inference; `specs/002-review-resolution-report/spec.md:164` mentions raw diff hunks but not code snippets or secret-bearing literals.
    Why this category over alternatives: This is not `new-OQ` because the privacy rule is already established by the spec; it only needs to be applied to the fallback path.
    Long-term vs band-aid: A band-aid would redact a few known secret patterns after the summary is generated; the durable fix is to constrain diff-derived fallback to high-level sanitized summaries plus file and commit evidence.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-015**: System MUST infer applied-solution details from the relevant fix round and commit diff when structured resolution details are missing, and MUST label that fallback source explicitly.
  ```
  After:
  ```markdown
  - **FR-015**: System MUST infer applied-solution details from the relevant fix round and commit diff when structured resolution details are missing, and MUST label that fallback source explicitly. Diff-derived fallback text MUST be a high-level sanitized summary supported by changed-file paths and fix commit SHA when available; it MUST NOT include raw hunks, code snippets, secret-bearing literals, or near-verbatim code excerpts.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-009

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The spec says unselected findings become accepted after approve or skip, but it does not scope the authority or distinguish user skip from configured/automated skip.
    Evidence: `specs/002-review-resolution-report/spec.md:125` contains the unselected-finding rule; local approval actions include `approve`, `fix`, `skip`, and `abort` at `internal/types/types.go:121-129`, while configured skipped steps can bypass execution in `internal/pipeline/executor.go:145-150`.
    Why this category over alternatives: This is not `new-OQ` because the local action model is explicit, and not `skipped` because configured skip and acceptance are materially different contracts.
    Long-term vs band-aid: A band-aid would special-case one automated path; the durable fix is to require an authority-scoped terminal decision before acceptance is rendered.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - User selects a subset of findings for fix: selected findings can be resolved or still open; unselected findings become accepted only after an approve or skip decision completes the Review step.
  ```
  After:
  ```markdown
  - User selects a subset of findings for fix: selected findings can be resolved or still open; unselected findings become accepted only after an explicit user approval/skip decision or documented pipeline-policy acceptance completes the Review step. Configured skipped steps, automated convergence approvals, canceled runs, superseded runs, and deferred/unselected findings without such a persisted terminal decision MUST NOT be rendered as accepted.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-010

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The spec preserves original finding source but does not require outcome-decision provenance on each report entry, so the finding's premise holds.
    Evidence: `specs/002-review-resolution-report/spec.md:172` lists original fields, outcome, applied solution, and evidence; `internal/db/round.go:23-33` persists selected IDs, selection source, user findings, and fix summary but not terminal acceptance provenance.
    Why this category over alternatives: This is not `new-OQ` because the provenance fields are dictated by existing actions and round state, and not `accepted-risk` because misattribution undermines the report's core purpose.
    Long-term vs band-aid: A band-aid would add vague prose like "by pipeline" to rendered text; the durable fix is to make decision actor/source, round ID, action, and evidence references explicit entry fields.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **Resolution Entry**: One report section for one normalized Review issue, including original finding fields, outcome category, applied solution, rationale, verification, source labels, changed files, fix commit when known, and no-commit reason when no fix commit exists.
  ```
  After:
  ```markdown
  - **Resolution Entry**: One report section for one normalized Review issue, including original finding fields, outcome category, applied solution, rationale, verification, source labels, changed files, fix commit when known, no-commit reason when no fix commit exists, and outcome-decision provenance. Provenance MUST include decision actor/source, action, timestamp when available, Review round ID, selection source, and the evidence reference used to classify the entry.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-011

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-022 requires compact status and counts on AXI/TUI but does not define how still-open counts affect wording, so success-like UI language remains possible.
    Evidence: `specs/002-review-resolution-report/spec.md:162` says AXI/TUI expose compact status, counts, and path; `specs/002-review-resolution-report/spec.md:157` requires a surface-facing status without defining derivation.
    Why this category over alternatives: This is not `new-OQ` because the safe derivation rule follows from FR-010: still-open findings are not success.
    Long-term vs band-aid: A band-aid would tweak one string in AXI or TUI; the durable fix is a shared surface wording rule driven by persisted metadata status and counts.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-022**: System MUST expose compact report status, issue counts, and local report path through AXI and TUI run detail surfaces when a report exists, and MUST NOT inline the full report narrative in those surfaces.
  ```
  After:
  ```markdown
  - **FR-022**: System MUST expose compact report status, issue counts, and local report path through AXI and TUI run detail surfaces when a report exists, and MUST NOT inline the full report narrative in those surfaces. Surface wording MUST be derived from the persisted report status and counts; any nonzero `still_open_count` or evidence-unavailable status MUST use non-success language.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-012

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The spec tests PR rendering and report contents separately, but it does not require all surfaces to read the same classified snapshot.
    Evidence: `specs/002-review-resolution-report/spec.md:195` only mentions PR body tests; `specs/002-review-resolution-report/spec.md:189` covers Markdown entry inclusion, leaving metadata and AXI/TUI consistency implicit.
    Why this category over alternatives: This is not `new-OQ` because the architecture already says SQLite metadata lets surfaces avoid reparsing logs, and not `skipped` because stale counts would contradict the feature purpose.
    Long-term vs band-aid: A band-aid would compare counts in one renderer; the durable fix is to make Markdown, metadata, AXI/TUI, and PR consume the same persisted classified entry set.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **SC-007**: PR body generation tests show compact review-resolution counts/status when metadata exists, omit the section when no report exists, and never include local paths or report excerpts.
  ```
  After:
  ```markdown
  - **SC-007**: PR body generation tests show compact review-resolution counts/status when metadata exists, omit the section when no report exists, and never include local paths or report excerpts. Integration tests MUST compare the Markdown section counts, SQLite metadata counts/status, AXI/TUI run-detail output, and PR summary output for the same run to prove they come from the same persisted classified snapshot.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-013

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-025 fails the run on required write or persistence failure, but it does not define atomicity or partial/degraded metadata when report generation starts and fails mid-refresh.
    Evidence: `specs/002-review-resolution-report/spec.md:165` covers failures broadly; `internal/db/schema.go:63-75` shows additive migrations are idempotent but there is no report metadata table yet, so its atomic refresh contract belongs in the spec.
    Why this category over alternatives: This is not `new-OQ` because the invariant is mechanical: never show confident counts from a partial write.
    Long-term vs band-aid: A band-aid would delete a half-written report on errors; the durable fix is atomic report/metadata refresh with integrity fields and degraded state when a previous snapshot is retained.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-025**: System MUST fail the Review step/run with an actionable error when required report writing or metadata persistence fails for a run with Review findings.
  ```
  After:
  ```markdown
  - **FR-025**: System MUST fail the Review step/run with an actionable error when required report writing or metadata persistence fails for a run with Review findings. Report Markdown and `review_resolution_reports` metadata refresh MUST be atomic from the perspective of consumers; metadata MUST include integrity fields such as report version, entry count, source round range or watermark, and last refresh result, and surfaces MUST show degraded or evidence-unavailable status instead of confident counts when consistency checks fail.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-014

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The spec carries forward "Stable Markdown structure" but only names high-level sections, so the finding's premise holds for exact labels, heading levels, and count semantics.
    Evidence: `specs/002-review-resolution-report/spec.md:50` carries stable structure forward; `plans/grill-me/review-resolution-report.md:91-127` shows a concrete template with labels such as `Applied Solution Source`.
    Why this category over alternatives: This is not `skipped` because stable Markdown is explicitly in scope, but the fix should stay simple and not create a separate machine parser requirement.
    Long-term vs band-aid: A band-aid would rely on writer convention; the durable fix is a versioned canonical template with golden-file tests for compatible headings and labels.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - Stable Markdown structure with run context and per-issue sections from lines 91-130.
  ```
  After:
  ```markdown
  - Stable Markdown structure with run context and per-issue sections from lines 91-130, including a report format version marker, fixed heading hierarchy, exact field labels, outcome section order, and count semantics covered by golden-file tests.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-015

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-011 omits `context` and `suggested_fix`, while the origin source explicitly lists them as current pre-fix review fields and the spec carries forward proposed solution structure.
    Evidence: `specs/002-review-resolution-report/spec.md:151` lists preserved fields; `plans/grill-me/review-resolution-report.md:30-35` lists `description`, `context`, `suggested_fix`, `action`, and risk fields.
    Why this category over alternatives: This is not `new-OQ` because the origin artifact answers the field intent, and not `skipped` because the omission would lose source evidence.
    Long-term vs band-aid: A band-aid would render only fields present in the current Go struct; the durable fix is to define optional original context and suggested/proposed fix handling with explicit unavailable markers.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-011**: System MUST preserve the original finding details available in Review output, including ID, severity, file, line, action, source, description, risk level, risk rationale, and user instructions when present.
  ```
  After:
  ```markdown
  - **FR-011**: System MUST preserve the original finding details available in Review output, including ID, severity, file, line, action, source, description, context, suggested or proposed fix, risk level, risk rationale, and user instructions when present. When historical or legacy findings lack context or suggested/proposed fix fields, the report MUST render an explicit `not recorded` or `unavailable in historical data` marker rather than silently omitting the field.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-016

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The assumptions say round history is the foundation, but they do not make persisted rounds plus decision/fix metadata the authoritative regeneration source or define freshness markers.
    Evidence: `specs/002-review-resolution-report/spec.md:203` states the foundation; `internal/db/round.go:125-143` provides ordered persisted rounds, and `internal/db/run.go:204-231` shows stale recovery operates from persisted DB state.
    Why this category over alternatives: This is not `new-OQ` because the existing local persistence model answers the architecture question, and not `accepted-risk` because regenerating from transient state would undermine report trust.
    Long-term vs band-aid: A band-aid would refresh from the latest step findings when needed; the durable fix is to require regeneration from persisted Review rounds and metadata watermarks.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - Existing round history remains the foundation for report generation; the feature should add metadata only where the current round data is insufficient.
  ```
  After:
  ```markdown
  - Existing persisted Review step rounds are the authoritative regeneration source for report generation; the feature should add decision, fix, and report metadata only where the current round data is insufficient. Regeneration MUST rebuild from persisted Review rounds plus persisted decision/fix metadata, not transient events or latest findings alone, and metadata MUST store a source-round watermark or content hash so stale Markdown/SQLite mismatches are detectable.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-017

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The clarification requires content limits, but it does not say structured fix-agent fields are untrusted inputs subject to length caps and raw transcript/log/diff/code rejection.
    Evidence: `specs/002-review-resolution-report/spec.md:19` says per-field and total-report caps; `internal/pipeline/steps/review.go:58-60` currently constrains only the legacy summary prompt, not future `resolutions[]` fields.
    Why this category over alternatives: This is not `new-OQ` because the privacy and bounds policy is already chosen, and not `skipped` because structured agent output is a real input path for durable Markdown.
    Long-term vs band-aid: A band-aid would sanitize only Markdown after assembly; the durable fix is to validate and sanitize each structured field before persistence and rendering.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **Content limits**: Enforce per-field and total-report caps with explicit truncation markers.
  ```
  After:
  ```markdown
  - **Content limits**: Enforce per-field and total-report caps with explicit truncation markers. These caps apply to fix-agent `summary`, `applied_solution`, `why_this_solution`, changed-file labels, user instructions, findings, and inferred summaries, and the system MUST reject or degrade raw transcript, log, diff, or code-block content before persistence.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-018

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The spec includes approval reasons, user-authored findings, and user instructions, but it does not require labels distinguishing explicit user input from unavailable or inferred intent.
    Evidence: `specs/002-review-resolution-report/spec.md:70` mentions approval reason available to the system; `specs/002-review-resolution-report/spec.md:129` includes user-authored findings; `internal/types/findings.go:154-201` shows user instructions and added findings are merged into Review findings.
    Why this category over alternatives: This is not `new-OQ` because the source labels already exist locally (`agent` and `user`), and not `skipped` because transcript-derived intent would be unsafe durable evidence.
    Long-term vs band-aid: A band-aid would hide all user text; the durable fix is to persist and render explicit source labels and unavailable markers while applying the same sanitization rules.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - User-authored findings from TUI or AXI: include them as Review issues with source `user` when they are dispatched through the Review fix flow.
  ```
  After:
  ```markdown
  - User-authored findings from TUI or AXI: include them as Review issues with source `user` when they are dispatched through the Review fix flow, and label any user reason or instruction in the report as explicit approval input, explicit AXI/TUI finding input, explicit per-finding instruction, or unavailable. The report MUST NOT infer approval reasons or user intent summaries from raw transcripts unless the text is stored only as a sanitized bounded summary with its source label.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-019

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-007 has only three buckets, while local Review findings have `action: "no-op"` for informational items that do not require fix or acceptance.
    Evidence: `specs/002-review-resolution-report/spec.md:147` lists `Resolved Issues`, `Not Fixed / Accepted Issues`, and `Still Open Issues`; `internal/types/findings.go:213-218` states no-op findings are informational and need no fix.
    Why this category over alternatives: This is not `skipped` because the local model proves informational findings exist, and not `out-of-scope` because FR-005 requires every normalized Review issue known for the run.
    Long-term vs band-aid: A band-aid would stuff no-op items into accepted; the durable fix is to add a separate informational/no-action outcome that does not imply approval or remediation.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-007**: System MUST classify report entries into `Resolved Issues`, `Not Fixed / Accepted Issues`, and `Still Open Issues`.
  ```
  After:
  ```markdown
  - **FR-007**: System MUST classify report entries into `Resolved Issues`, `Accepted Without Fix`, `Still Open Issues`, and `Informational / No Action Required`. Informational entries are limited to Review findings whose effective action is `no-op` and MUST NOT be counted as resolved fixes or accepted risks.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-020

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: Acceptance Scenario 1.1 says follow-up Review no longer reports the finding, but it does not require a specific verification event, round ID, equivalent scope, or normalized ID comparison.
    Evidence: `specs/002-review-resolution-report/spec.md:69` is the scenario; `internal/pipeline/steps/review.go:28-31` shows review scope differs between initial and fix mode, so scope equivalence must be explicit.
    Why this category over alternatives: This is not `new-OQ` because the verifier model is already Review rounds, and not `skipped` because scope drift is a real local concern.
    Long-term vs band-aid: A band-aid would assume the next clean review is comparable; the durable fix is to require every resolved entry to cite the verification event and scope basis.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  1. **Given** a Review step emits one fixable finding, **When** the finding is fixed and a follow-up Review no longer reports it, **Then** the report contains the finding under `Resolved Issues` with problem, applied solution, rationale, fix source, and verification text.
  ```
  After:
  ```markdown
  1. **Given** a Review step emits one fixable finding, **When** the finding is fixed and a comparable follow-up Review round no longer reports the same normalized finding ID, **Then** the report contains the finding under `Resolved Issues` with problem, applied solution, rationale, fix source, verification text, follow-up round ID, scope-equivalence note, and verifier source.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-021

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-015 requires diff inference and FR-019 allows multiple findings per fix round or commit, so the finding correctly identifies a per-finding attribution risk.
    Evidence: `specs/002-review-resolution-report/spec.md:155` requires inferred applied-solution details; `specs/002-review-resolution-report/spec.md:159` allows multiple findings sharing one fix round or commit.
    Why this category over alternatives: This is not `new-OQ` because the safe attribution rule is clear, and not `accepted-risk` because mislabeled per-finding fixes would directly undermine the report.
    Long-term vs band-aid: A band-aid would add vague "may include other fixes" wording; the durable fix is to forbid per-finding inferred claims unless structured mapping or exact evidence supports them.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-019**: System MUST support multiple findings sharing one fix round or commit when the existing fix flow handles selected findings together.
  ```
  After:
  ```markdown
  - **FR-019**: System MUST support multiple findings sharing one fix round or commit when the existing fix flow handles selected findings together. When structured `resolutions[]` data is absent and one commit or fix round covers multiple findings, inferred evidence MUST be labeled commit-level or round-level evidence, not finding-specific applied-solution detail, unless exact verification evidence supports the per-finding mapping.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-022

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: FR-004 includes a generated timestamp, but it does not distinguish first generation, refresh, or finalization state even though FR-001 and FR-020 make the report lifecycle incremental.
    Evidence: `specs/002-review-resolution-report/spec.md:141` creates the report immediately after first findings, and `specs/002-review-resolution-report/spec.md:144` only names "generated timestamp".
    Why this category over alternatives: This is not `new-OQ` because lifecycle states follow from the already-required refresh points, and not `skipped` because AXI/TUI display depends on whether the report is final.
    Long-term vs band-aid: A band-aid would append "(draft)" to some in-progress displays; the durable fix is to persist lifecycle timestamps and report state.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  - **FR-004**: System MUST include a `Run Context` section with run ID, repo identifier or path, branch, base commit, current/final head commit at generation time, Review step status, generated timestamp, and local report path.
  ```
  After:
  ```markdown
  - **FR-004**: System MUST include a `Run Context` section with run ID, repo identifier or path, branch, base commit, current/final head commit at generation time, Review step status, report lifecycle state, first generated timestamp, last refreshed timestamp, finalized timestamp when applicable, and local report path.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-023

- Category: spec-fix
- Payload:
  Reasoning:
    Verification: The label `Not Fixed / Accepted Issues` is used in the acceptance scenario, and it compresses approve and skip paths without requiring action/reason display.
    Evidence: `specs/002-review-resolution-report/spec.md:70` uses `Not Fixed / Accepted Issues`; `internal/types/types.go:125-128` distinguishes approve, skip, fix, and abort, so the report should preserve which action produced acceptance.
    Why this category over alternatives: This is not `new-OQ` because the local action vocabulary is explicit, and not `skipped` because the wording can misrepresent a skip as a reviewed-safe acceptance.
    Long-term vs band-aid: A band-aid would rename only the heading; the durable fix is to use clearer heading language and render the acceptance action and reason per entry.
  Target: specs/002-review-resolution-report/spec.md
  Before:
  ```markdown
  2. **Given** a Review step emits a finding that the user approves as-is, **When** the Review step completes, **Then** the report contains that finding under `Not Fixed / Accepted Issues` with the approval reason available to the system.
  ```
  After:
  ```markdown
  2. **Given** a Review step emits a finding that the user approves as-is, **When** the Review step completes, **Then** the report contains that finding under `Accepted Without Fix` with the acceptance action, actor/source, and approval reason available to the system. If `skip` has materially different semantics from approval in the persisted decision record, the entry MUST show that skip action distinctly instead of collapsing it into generic accepted wording.
  ```

Status: applied
Applied-at: 2026-06-18T23:29:04+07:00
Downstream-ref: specs/002-review-resolution-report/spec.md

### F-RT-002-review-resolution-report-2026-06-18-024

- Category: skipped
- Payload:
  Reasoning:
    Verification: The finding's privacy concern is real in the abstract, but its proposed direction conflicts with the verified feature requirement to expose the local report path on local AXI/TUI surfaces.
    Evidence: `specs/002-review-resolution-report/spec.md:162` says the system "MUST expose compact report status, issue counts, and local report path through AXI and TUI"; `specs/002-review-resolution-report/spec.md:163` separately forbids publishing local-only filesystem paths in PR content.
    Why this category over alternatives: This is not `spec-fix` because removing or redacting the local path from AXI/TUI would contradict FR-022, and not `accepted-risk` because the spec already confines public exposure by prohibiting local paths in PR output.
    Long-term vs band-aid: A band-aid would hide paths in some AXI output modes; the durable choice is to preserve the local path requirement for explicitly local surfaces and rely on FR-023 for non-local PR privacy.
  Reason: Skipped because the suggested fix conflicts with `specs/002-review-resolution-report/spec.md:162` ("MUST expose ... local report path through AXI and TUI") and the public leakage path is already blocked at `specs/002-review-resolution-report/spec.md:163` ("without publishing local-only filesystem paths").

Status: skipped
Applied-at: 2026-06-18T23:29:04+07:00

### F-RT-002-review-resolution-report-2026-06-18-025

- Category: skipped
- Payload:
  Reasoning:
    Verification: The finding relies on weaker acceptance-scenario wording, but the mandatory functional requirement already requires all three fields: status, counts, and local path.
    Evidence: `specs/002-review-resolution-report/spec.md:71` says AXI/TUI provide local path or compact status, while `specs/002-review-resolution-report/spec.md:162` says AXI/TUI "MUST expose compact report status, issue counts, and local report path".
    Why this category over alternatives: This is not `spec-fix` because the mandatory requirement already resolves the implementation contract, and not `new-OQ` because no human decision is missing.
    Long-term vs band-aid: A band-aid would duplicate FR-022 text into the user story; the durable interpretation is to let the mandatory FR govern weaker scenario prose.
  Reason: Skipped because `specs/002-review-resolution-report/spec.md:162` already requires "compact report status, issue counts, and local report path" whenever a report exists, so the cited acceptance wording does not weaken the implementation contract.

Status: skipped
Applied-at: 2026-06-18T23:29:04+07:00

## 5. Session Metadata

```yaml
schema_version: red-team-findings/v1
session_id: RT-002-review-resolution-report-2026-06-18
target: specs/002-review-resolution-report/spec.md
feature_id: 002-review-resolution-report
date: 2026-06-18
maintainer: Codex
command: speckit.red-team.run
arguments:
  target: specs/002-review-resolution-report
  resolved_target_spec: specs/002-review-resolution-report/spec.md
  yes: true
  lenses: null
  dry_run: false
selection:
  method: auto
  matched_triggers:
    - ai_llm
    - immutability_audit
    - multi_party
    - contracts
  selected_lenses:
    - Agent Contract Integrity Adversary
    - Partial Evidence Recovery Adversary
    - Privacy and Transcript Exposure Adversary
    - Review Trust-Boundary Adversary
    - User Surface Misrepresentation Adversary
summary:
  total_findings: 25
  by_severity:
    CRITICAL: 0
    HIGH: 12
    MEDIUM: 11
    LOW: 2
  by_lens:
    Agent Contract Integrity Adversary: 5
    Partial Evidence Recovery Adversary: 5
    Privacy and Transcript Exposure Adversary: 5
    Review Trust-Boundary Adversary: 5
    User Surface Misrepresentation Adversary: 5
agent_failures: []
dropped_findings: 0
resolution_counts:
  spec-fix: 23
  new-OQ: 0
  accepted-risk: 0
  out-of-scope: 0
  skipped: 2
  unresolved: 0
apply:
  applied_at: 2026-06-18T23:29:04+07:00
  applied_by: Codex
  resolutions:
    spec_fix: 23
    new_OQ: 0
    accepted_risk: 0
    out_of_scope: 0
    skipped: 2
  unresolved: 0
  allow_historical_edits: true
  historical_edits_applied:
      - F-RT-002-review-resolution-report-2026-06-18-001:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-002:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-003:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-004:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-005:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-006:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-007:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-008:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-009:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-010:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-011:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-012:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-013:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-014:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-015:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-016:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-017:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-018:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-019:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-020:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-021:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-022:specs/002-review-resolution-report/spec.md
      - F-RT-002-review-resolution-report-2026-06-18-023:specs/002-review-resolution-report/spec.md
notes:
  - The requested feature_directory was specs/002-review-resolution-report; the extension protocol requires a target spec, so the resolved target spec was specs/002-review-resolution-report/spec.md.
  - No hooks were registered under hooks.before_speckit_red_team_run.
  - The constitution does not declare a dedicated Red Team Trigger Criteria section; default trigger categories were used.
```
