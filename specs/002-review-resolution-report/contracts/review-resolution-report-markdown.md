# Contract: Review Resolution Report Markdown v1

**Contract version**: `review-resolution-report/v1`

This contract defines the stable Markdown artifact generated at:

```text
$NM_HOME/reports/<runID>/review-resolution.md
```

## Required Heading Order

The report MUST use these exact headings in this exact order:

```markdown
# Review Resolution Report
## Report Metadata
## Purpose
## Run Context
## Summary Counts
## Latest Review Outcome
## Review Findings
## Fix Attempts
## Remaining Risks
## Source Evidence
## Generation Notes
```

No required heading may be renamed or removed in v1. Optional subsections may be added under these headings only when tests continue to validate the required anchors.

## Report Metadata

Required labels:

- `Contract version`
- `Report path`
- `Report status`
- `Generation mode`
- `Generated at`
- `Updated at`
- `Source snapshot at`

Allowed `Report status` values:

- `current`
- `stale`
- `unavailable`
- `error`

Allowed `Generation mode` values:

- `live`
- `regenerated`

## Purpose

The section MUST state that the report records review findings, resolution decisions, fix attempts, applied fix summaries, and remaining risks for later human and agent review. It MUST state that it does not change review, approval, auto-fix, push, PR, or CI behavior.

## Run Context

Required labels:

- `Run`
- `Branch`
- `Base commit`
- `Head commit`
- `Run status`
- `Review status`
- `Safe intent summary`
- `PR`

Missing values render as `unavailable`.

## Summary Counts

The report MUST include every canonical count key exactly once:

```text
total_findings
actionable_findings
selected_for_fix
fix_attempts
applied_fix_summaries
accepted
skipped
informational
deferred
still_open
unavailable
decision_not_recorded
```

Counts MUST match the persisted `summary_counts_json` metadata for the same source snapshot. The report MUST NOT emit an aggregate `resolved` count in v1.

## Latest Review Outcome

Allowed latest-outcome labels:

- `no issues remain`
- `unresolved findings remain`
- `no reviewable changes`
- `awaiting user decision`
- `final findings unavailable`
- `final findings unreadable`
- `review data inconsistent`
- `review resolution incomplete`

The section MUST include:

- `Latest outcome`
- `Evidence`
- `Risk`
- `Rationale`

`no issues remain` MUST cite a successfully parsed latest review pass for the same run after the relevant fix attempt.

## Review Findings

Each finding MUST render exactly once with these labels:

- `Issue`
- `Severity`
- `Location`
- `Source`
- `Action type`
- `Context`
- `Recommendation`
- `Selected for fix`
- `Resolution decision`
- `Decision actor`
- `Decision evidence`
- `User instructions`

Allowed user-facing decision labels:

- `Selected for fix`
- `Accepted`
- `Skipped`
- `Informational`
- `Deferred`
- `Still open`
- `Decision not recorded`
- `Unavailable`

`Accepted` may appear only when stored data contains explicit human/user risk-acceptance evidence. `Recommendation` and `Applied fix` are distinct labels; a recommendation MUST NOT be rendered as an applied fix.

Missing historical fields render as `not recorded` or `unavailable`.

## Fix Attempts

Each fix attempt MUST render in chronological order with these labels:

- `Fix attempt`
- `Selected findings`
- `Selection source`
- `User instructions`
- `User-authored findings`
- `Applied fix`
- `Verification`
- `Evidence`

When a fix round has no summary, `Applied fix` MUST be:

```text
fix applied, no summary recorded
```

The `Verification` label is derived from follow-up review evidence, never from the fix summary alone.

## Remaining Risks

The section MUST list still-open, unavailable, decision-not-recorded, incomplete, or inconsistent review states when any are present. If none are present, it may state `None recorded`.

## Source Evidence

Required labels:

- `Review step result`
- `Included rounds`
- `Latest review round`
- `Latest fix round`
- `Source snapshot at`
- `Integrity status`

Integrity status values:

- `consistent`
- `inconsistent`
- `partial`
- `unavailable`

Safe diagnostics may identify record IDs and field names but MUST NOT include raw invalid payloads.

## Generation Notes

This section MUST include safe generation errors, stale/unavailable warnings, redaction notes, or `No generation warnings`.

## Sanitization And Privacy

The report may include only allowlisted, sanitized fields:

- finding locations;
- safe finding context;
- recommendations;
- decisions and actors;
- selected IDs;
- user instructions after sanitization;
- fix summaries after sanitization;
- safe run intent;
- risk level and rationale.

The report MUST NOT include:

- raw agent transcripts;
- raw logs;
- raw diff hunks;
- raw code excerpts;
- secret-bearing values;
- adjacent IPC diff payloads.

Unsafe values render as `unavailable` or a non-sensitive summary.

## Extraction Rules

Future agents and tests may rely on:

- the exact heading order;
- the contract version string;
- the canonical count keys;
- the allowed latest-outcome labels;
- the allowed decision labels;
- `Recommendation` never meaning `Applied fix`;
- `Applied fix` never proving `no issues remain` without cited review evidence.
