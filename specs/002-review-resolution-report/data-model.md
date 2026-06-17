# Data Model: Review Resolution Report

## Entity: ReviewResolutionReportMetadata

One SQLite row per run, stored in a new `review_resolution_reports` table.

Fields:

- `run_id TEXT PRIMARY KEY`: references `runs(id)` with cascade delete.
- `report_path TEXT`: absolute path to `$NM_HOME/reports/<runID>/review-resolution.md` when a report was written.
- `status TEXT NOT NULL`: one of `current`, `stale`, `unavailable`, `error`.
- `contract_version TEXT NOT NULL`: `review-resolution-report/v1`.
- `latest_outcome TEXT NOT NULL`: one allowed latest-outcome label from the Markdown contract.
- `summary_counts_json TEXT NOT NULL`: JSON object with all canonical summary-count keys.
- `generation_mode TEXT NOT NULL`: `live` or `regenerated`.
- `source_snapshot_at INTEGER NOT NULL`: Unix timestamp for the DB snapshot used to render.
- `source_step_result_id TEXT`: review step result that supplied the report.
- `source_round_ids_json TEXT NOT NULL`: JSON array of included review round IDs.
- `latest_review_round_id TEXT`: latest parsed review round included in the report.
- `latest_fix_round_id TEXT`: latest fix round included in the report.
- `generated_at INTEGER`: timestamp of first successful generation for this current artifact.
- `updated_at INTEGER NOT NULL`: timestamp of the latest generation attempt or metadata update.
- `stale INTEGER NOT NULL DEFAULT 0`: `1` when newer source evidence exists than the last successful artifact.
- `safe_error TEXT`: short sanitized generation or consistency error, if status is `error` or `unavailable`.

Validation:

- At most one row exists per run.
- `report_path` is non-empty only when a successful artifact exists.
- `summary_counts_json` contains every canonical key, even when a count is zero.
- `status=current` requires `stale=0`, an existing report path, and no `safe_error`.
- `status=stale` requires a last successful report path and a safe reason.
- `status=unavailable` is used when no report artifact can be referenced.
- `status=error` is used when generation failed but captured review data remains available for regeneration.

## Entity: ReviewResolutionReport

The generated Markdown artifact for one run.

Fields:

- `contract_version`: fixed to `review-resolution-report/v1`.
- `run_id`, `branch`, `base_sha`, `head_sha`, optional `pr_url`.
- `report_path`, `generation_mode`, `generated_at`, `updated_at`.
- `review_status`, `run_status`, `latest_outcome`.
- `safe_intent_summary`: from the stored run intent only after sanitization, otherwise `unavailable`.
- `summary_counts`: the canonical counts persisted in metadata.
- `findings`: ordered list of reportable review findings.
- `fix_attempts`: ordered list of fix rounds.
- `source_evidence`: step result ID, round IDs, source snapshot timestamp, and safe consistency notes.

Validation:

- The artifact uses the exact heading order defined in `contracts/review-resolution-report-markdown.md`.
- It includes no raw logs, raw transcripts, raw diff hunks, or code excerpts.
- It may include only sanitized finding locations, safe finding context, decisions, summaries, risk, and safe run intent.

## Entity: ReportableReviewRound

Derived from existing `step_rounds` rows for the review step.

Fields:

- `round_id`, `round_number`, `trigger_type`, `created_at`.
- `findings_json`: parsed structured review output when available.
- `user_findings_json`: merged selected findings plus user instructions and user-authored findings when available.
- `selected_finding_ids`: parsed selected IDs when recorded.
- `selection_source`: `user`, `auto_fix`, or `not recorded`.
- `fix_summary`: one-line fix summary for fix rounds, or unavailable.
- `is_fix_round`: true for `auto_fix` and legacy `user_fix` triggers.

Validation:

- Malformed JSON makes the report fail closed with `review data inconsistent` or `final findings unreadable` depending on which record failed.
- Selected IDs must refer to findings from the source round or selected user-authored findings in `user_findings_json`.
- Legacy records missing selected IDs are labeled `decision not recorded`; decisions are not inferred from counts or summaries.

## Entity: ReviewFinding

Derived from existing `types.Finding` records.

Fields:

- `id`
- `severity`
- `location`: `file` plus optional one-indexed `line`
- `issue_title`: existing `description`
- `context`
- `recommendation`: existing `suggested_fix`, rendered as `Recommendation`
- `action_type`: `ask-user`, `auto-fix`, `no-op`, or `unavailable`
- `source`: `agent`, `user`, or `not recorded`
- `user_instructions`
- `decision`: one `ResolutionDecision`
- `evidence_reference`: source round/selection/fix/review identifier

Validation:

- Each reportable finding appears exactly once in the report.
- `suggested_fix` is never rendered as `Applied fix`.
- Missing fields are rendered as `not recorded` or `unavailable`.
- User-authored findings are distinguished from agent-produced findings.

## Entity: ResolutionDecision

The report-level decision for a finding.

Allowed states:

- `selected for fix`
- `accepted`
- `skipped`
- `informational`
- `deferred`
- `still open`
- `decision not recorded`
- `unavailable`

Decision mapping:

- `selected for fix`: finding ID appears in `selected_finding_ids` for a fix attempt, with `selection_source` recorded as actor/provenance.
- `accepted`: only when stored run data contains explicit human/user risk-acceptance evidence. Existing generic `approve` or `skip` actions do not imply this label.
- `informational`: finding has effective action `no-op` and does not require a fix.
- `still open`: the latest trustworthy parsed review evidence still contains the finding or an equivalent finding.
- `skipped`: a recorded decision excludes an actionable finding from a fix attempt without accepting risk. This must have an evidence reference; it is not inferred from missing selected IDs.
- `deferred`: reserved for explicit future stored decision data; not inferred in v1.
- `decision not recorded`: historical data is missing selected IDs, actor, or decision evidence.
- `unavailable`: source data required to classify the decision is unreadable or unavailable.

Precedence for a finding:

1. Source data inconsistent or unreadable -> `unavailable`.
2. Explicit accepted-risk evidence -> `accepted`.
3. Latest trustworthy review still contains the finding -> `still open`.
4. Recorded selected ID for a fix attempt -> `selected for fix`.
5. Effective action is `no-op` -> `informational`.
6. Recorded exclusion/skip evidence -> `skipped`.
7. Explicit deferred evidence -> `deferred`.
8. Otherwise -> `decision not recorded`.

## Entity: FixAttempt

Derived from each review fix round.

Fields:

- `round_id`, `round_number`, `selection_source`, `selected_finding_ids`.
- `user_instructions_present`: true when selected/user findings include instructions.
- `user_authored_finding_ids`.
- `agent_reported_fix_summary`: sanitized summary or `fix applied, no summary recorded`.
- `verification_status`: derived from follow-up review evidence, not the summary.
- `evidence_reference`: previous round ID plus follow-up review round ID when available.

Validation:

- Fix attempts render in chronological order.
- A summary alone never proves resolution.
- Missing summaries are visible and counted separately from unavailable records.

## Entity: SummaryCounts

Persisted as `summary_counts_json` and rendered in the report.

Required keys:

- `total_findings`
- `actionable_findings`
- `selected_for_fix`
- `fix_attempts`
- `applied_fix_summaries`
- `accepted`
- `skipped`
- `informational`
- `deferred`
- `still_open`
- `unavailable`
- `decision_not_recorded`

Validation:

- Counts are derived from the same report snapshot as the Markdown artifact.
- Aggregated `resolved` counts are not emitted. If a future surface adds such a count, it must explicitly exclude skipped, deferred, informational, accepted-risk, unavailable, decision-not-recorded, and still-open findings.
- If integrity checks fail, confident resolved/unresolved totals are omitted from prose and the report uses `review data inconsistent`.

## Latest Outcome Precedence

Allowed latest outcome labels:

- `no issues remain`
- `unresolved findings remain`
- `no reviewable changes`
- `awaiting user decision`
- `final findings unavailable`
- `final findings unreadable`
- `review data inconsistent`
- `review resolution incomplete`

Precedence table:

1. Any source integrity error involving findings, selected IDs, fix summaries, source parse results, or counts -> `review data inconsistent`.
2. Review step is awaiting approval or fix review and no terminal decision is recorded -> `awaiting user decision`.
3. A fix attempt exists and the run is failed, cancelled, or superseded before a parseable post-fix review for the latest fix attempt -> `review resolution incomplete`.
4. A fix attempt exists and no final/post-fix review findings are available -> `review resolution incomplete`.
5. No fix attempt exists and final review findings are missing -> `final findings unavailable`.
6. Final review findings are present but malformed/unreadable -> `final findings unreadable`.
7. Latest parsed review evidence indicates no reviewable changes -> `no reviewable changes`.
8. Latest parsed review evidence has zero findings -> `no issues remain`.
9. Latest parsed review evidence has one or more findings -> `unresolved findings remain`.

`no issues remain` is valid only from a successfully parsed latest review pass for the same run after the relevant fix attempt.

## State Transitions

Report metadata states:

```text
absent
  -> current       successful first generation
  -> unavailable   reportable review state exists but no artifact can be written

current
  -> current       successful update from newer or equal source evidence
  -> stale         newer evidence exists but generation failed
  -> error         generation failed and no current artifact can be trusted

stale
  -> current       regeneration succeeds with newer evidence
  -> stale         another generation attempt fails

error
  -> current       generation/regeneration succeeds
  -> unavailable   source data remains reportable but no artifact/reference is available
```

Regeneration must not overwrite an artifact generated from newer evidence. If the source snapshot is older than existing metadata, write no new artifact and mark the attempted output as stale/unavailable in metadata.

## Sanitization Rules

Allowed inputs:

- finding ID, severity, source, action, file path, and line;
- safe finding context and recommendation after sanitizer checks;
- selected IDs, selection source, and user instructions after sanitizer checks;
- agent-reported fix summaries after sanitizer checks;
- stored run intent after sanitizer checks;
- risk level and rationale after sanitizer checks.

Disallowed inputs:

- raw diff payloads, including IPC `diff`;
- raw logs;
- raw transcripts;
- code excerpts and diff hunks;
- secret-bearing, log-like, code-like, or transcript-derived values.

Unsafe values are replaced with `unavailable` or summarized without copying the unsafe content.
