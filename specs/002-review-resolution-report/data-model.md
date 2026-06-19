# Data Model: Review Resolution Report

## Existing Entities Reused

### Run

Source: `internal/db/run.go`, `internal/db/schema.go`

- `id`
- `repo_id`
- `branch`
- `head_sha`
- `base_sha`
- `status`
- `pr_url`
- `error`
- lifecycle timestamps

Used for Run Context and report path scoping.

### Step Result

Source: `internal/db/step.go`, `internal/db/schema.go`

- `id`
- `run_id`
- `step_name`
- `status`
- `findings_json`
- `error`
- lifecycle/duration/log fields

Only the Review step is in scope for this feature.

### Step Round

Source: `internal/db/round.go`, `internal/db/schema.go`

Existing fields reused:

- `id`
- `step_result_id`
- `round`
- `trigger_type`
- `findings_json`
- `user_findings_json`
- `selected_finding_ids`
- `selection_source`
- `fix_summary`
- `duration_ms`
- `created_at`

Additive fields introduced by this feature:

- `fix_commit_sha TEXT NULL`
- `no_commit_reason TEXT NULL`
- `fix_resolution_details_json TEXT NULL`

Rules:

- `fix_commit_sha` is set only when the fix round produced a commit tied to the Review fix attempt.
- `no_commit_reason` distinguishes no-op, no changes, failed commit, missing evidence, and legacy unavailable cases.
- `fix_resolution_details_json` stores validated and bounded structured fix-agent resolution details, plus degradation markers when validation is partial.

## New Entities

### Review Resolution Report Metadata

Table: `review_resolution_reports`

Primary key: `run_id`

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `run_id` | TEXT PRIMARY KEY | References `runs(id)` with cascade delete |
| `report_path` | TEXT NOT NULL | Absolute local `$NM_HOME` path |
| `status` | TEXT NOT NULL | Enum below |
| `resolved_count` | INTEGER NOT NULL | Classified resolved issues |
| `accepted_count` | INTEGER NOT NULL | Accepted without fix |
| `informational_count` | INTEGER NOT NULL | No-action informational entries |
| `still_open_count` | INTEGER NOT NULL | Unresolved or inconclusive entries |
| `report_version` | TEXT NOT NULL | Markdown format version |
| `entry_count` | INTEGER NOT NULL | Total entries rendered |
| `source_round_start` | INTEGER NULL | First Review round included |
| `source_round_end` | INTEGER NULL | Last Review round included |
| `source_watermark` | TEXT NOT NULL | Stable watermark of source rounds/decisions |
| `content_hash` | TEXT NOT NULL | Hash of rendered Markdown |
| `last_refresh_result` | TEXT NOT NULL | Short success/degraded/error message |
| `first_generated_at` | INTEGER NOT NULL | Unix timestamp |
| `last_refreshed_at` | INTEGER NOT NULL | Unix timestamp |
| `finalized_at` | INTEGER NULL | Set when terminal report is final |
| `created_at` | INTEGER NOT NULL | Row creation timestamp |
| `updated_at` | INTEGER NOT NULL | Row update timestamp |

Status enum:

- `in_progress`
- `final`
- `incomplete`
- `stale`
- `degraded`
- `evidence_unavailable`

Status rules:

- `in_progress`: Review findings exist and the run/Review step is not terminal.
- `final`: Review evidence is terminal and all counts are supported by current source watermark.
- `incomplete`: run stopped before all unresolved findings received terminal evidence.
- `stale`: persisted Markdown/metadata watermark no longer matches Review rounds or decisions.
- `degraded`: report exists but some structured fix or integrity evidence is partial/invalid.
- `evidence_unavailable`: required evidence cannot be reconstructed.

### Review Resolution Decision

Table: `review_resolution_decisions`

Purpose: Preserve terminal decision provenance per Review finding without reading transcripts.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `id` | TEXT PRIMARY KEY | Generated ID |
| `run_id` | TEXT NOT NULL | References `runs(id)` |
| `step_result_id` | TEXT NOT NULL | References Review step result |
| `round_id` | TEXT NULL | Step round whose findings were acted on |
| `finding_id` | TEXT NOT NULL | Normalized finding ID |
| `action` | TEXT NOT NULL | `fix`, `approve`, `skip`, `abort`, `policy_accept`, `no_op` |
| `actor_source` | TEXT NOT NULL | `user`, `axi`, `tui`, `yolo`, `auto_fix`, `pipeline_policy`, `system` |
| `reason` | TEXT NULL | Sanitized bounded reason when available |
| `created_at` | INTEGER NOT NULL | Decision timestamp |

Rules:

- Accepted without fix requires `approve`, `skip`, or `policy_accept` with actor/source and affected finding ID.
- Abort/failure/supersede decisions never produce accepted entries by themselves.
- `no_op` applies only to informational Review findings.

### Review Resolution Entry

In-memory domain object rendered into Markdown.

Fields:

- `finding_id`
- original finding details: severity, file, line, action, source, description, context/suggested fix when available, risk/rationale, user instructions
- `outcome`: `resolved`, `accepted`, `informational`, `still_open`
- `outcome_reason`
- `selection_source`
- `decision_action`, `decision_actor_source`, `decision_timestamp`, `decision_reason`
- `fix_round_id`
- `fix_commit_sha`
- `no_commit_reason`
- `applied_solution`
- `why_this_solution`
- `applied_solution_source`
- `changed_files`
- `verification_text`
- `followup_round_id`
- `scope_equivalence_note`
- `verifier_source`
- `evidence_quality`: `structured`, `inferred`, `round_level`, `degraded`, `unavailable`

Rules:

- One entry per normalized finding ID.
- Repeated same-ID findings update the same entry.
- Ambiguous changed IDs do not resolve prior findings.
- No raw hunks, raw logs, raw transcripts, code fences, or secret-like values are stored.

### Fix Resolution Detail

Validated optional fix-agent output item.

JSON shape:

```json
{
  "finding_id": "review-1",
  "applied_solution": "short bounded text",
  "why_this_solution": "short bounded text",
  "changed_files": ["internal/example.go"]
}
```

Validation:

- required non-empty `finding_id`, `applied_solution`, `why_this_solution`, and `changed_files`
- unique IDs
- IDs must match selected Review finding IDs to count as finding-specific evidence
- unknown/duplicate/missing selected IDs are retained only as degraded evidence
- all text is sanitized and bounded before persistence

## Relationships

- `runs 1 -> 1 review_resolution_reports`
- `runs 1 -> many step_results`
- `step_results 1 -> many step_rounds`
- `runs 1 -> many review_resolution_decisions`
- Review report generation reads one run, its Review step result, all Review rounds, fix evidence, and decisions.

## Integrity Invariants

- If no Review findings exist across all Review rounds, no report row and no Markdown file are created.
- If Review findings exist, `entry_count = resolved_count + accepted_count + informational_count + still_open_count`.
- Metadata counts and Markdown section counts must come from the same classified snapshot.
- `source_watermark` changes whenever Review rounds, decisions, or fix evidence used by the report change.
- Consumers treat missing Markdown, hash mismatch, or stale watermark as degraded/evidence-unavailable rather than success.
