# Data Model: Review File Handoff

## Review Handoff State

Durable state stored on the review step result as typed JSON in `step_results.review_handoff_json`.

Fields:

- `version`: schema version, initially `1`
- `relative_path`: normalized repository-relative path, for example `.no-mistakes/issues/feature-x/review-issues-abc12345.md`
- `cycle_id`: stable identifier for the current review or fix-review cycle
- `finding_digest`: deterministic digest of the latest normalized finding set for this cycle
- `generated_content_digest`: digest of the generated pending file content used to detect user edits before overwrite
- `processed_action`: `pending`, `approve`, `fix`, `skip`, `automation`, or `auto_fix`
- `processed_at`: Unix timestamp or null when pending
- `decision_source`: `file`, `automation`, `auto_fix`, or empty when pending
- `decisions`: processed per-finding decisions for audit, empty until processing or automation response
- `updated_at`: Unix timestamp for the persisted state

Validation rules:

- `relative_path` must be repository-relative, slash-normalized, non-empty, and must not escape the checkout or isolated work area when joined and canonicalized.
- `cycle_id` and `finding_digest` must match the active latest review result before file processing.
- `processed_action` must be `pending` with empty `processed_at` before a file process action can dispatch approval or fix.
- Existing rows without `review_handoff_json` are valid legacy rows and omit `phase`/`review_file`.

## Review Handoff File

The Markdown file shown to the developer for a review decision point.

Fields in front matter:

- `no_mistakes_review_handoff`: `v1`
- `run_id`
- `step`: `review`
- `status`: `awaiting_approval` or `fix_review`
- `branch`
- `review_cycle_id`
- `finding_digest`
- `review_file`
- `processed_action`
- `processed_at`

Sections per finding:

- `Issue`
- `Context`
- `Recommendation`
- `User Answer`
- one fenced response block keyed by normalized `Finding.ID`

Validation rules:

- Metadata must match the active review gate and durable handoff state.
- Each latest finding must appear exactly once as a response block.
- No duplicate, missing, or unknown finding IDs are accepted.
- Prose outside the fenced response block is ignored for processing.

## Review Finding

Existing `types.Finding` enriched only by generated handoff rendering.

Relevant fields:

- `ID`: normalized stable ID such as `review-1`
- `Severity`
- `File`
- `Line`
- `Description`
- `Context`
- `SuggestedFix`: rendered as `Recommendation`
- `Action`: `auto-fix`, `ask-user`, or `no-op`
- `UserInstructions`: used when a fix response supplies solution text

Validation rules:

- Every generated response block must use the persisted normalized `ID`.
- Default response action mapping:
  - `auto-fix` -> `fix`
  - `ask-user` -> `accept`
  - `no-op` -> `skip`
- A generated finding whose default action is `fix` must have a machine-detectable non-empty recommendation option 1.

## Response Block

The only editable user-decision input parsed from the file.

Fields:

- `finding_id`: carried in the fence info string after the tag
- `action`: `fix`, `accept`, or `skip`
- `solution`: one-line text, possibly empty

Validation rules:

- Opening fence must be exactly ` ```no-mistakes-review-response <finding-id>`.
- The body must contain exactly one `action:` line and exactly one `solution:` line.
- Field names are exact lowercase.
- Unknown fields, duplicate fields, nested fences, multiline continuations, and multiple IDs are rejected.
- Whitespace around field values is trimmed deterministically.
- `fix` with empty/comment-only solution uses trusted recommendation option 1. If none exists, validation fails.

## Processed Decision

Durable per-finding audit data kept in `Review Handoff State.decisions` and rendered in the latest/final handoff file as a resolved-decision summary.

Fields:

- `finding_id`
- `action`: `fix`, `accept`, or `skip`
- `solution`: user-authored solution, default recommendation option 1, or empty
- `solution_source`: `user`, `default_recommendation`, or `none`
- `decision_source`: `file`, `automation`, or `auto_fix`
- `processed_at`

Validation rules:

- `accept` and `skip` remain distinct even though both mean no remediation in the first version.
- A later handoff overwrite must not erase processed decisions from the latest audit summary.

## Review Phase Label

Human-readable label derived from review step status.

States:

- review `running`: `Review preview`
- review `awaiting_approval`: `Review preview complete`
- review `fixing`: `Fixing review issues`
- review `fix_review`: `Review fix result`
- review `completed`: no sub-phase label; render as `Review`
- non-review steps: no phase label

Validation rules:

- Raw statuses and step names remain unchanged.
- `phase` is omitted for non-review steps and completed review steps.

## Review Cycle

One generated review result or fix-review result that replaces the current handoff file.

Fields:

- `cycle_id`
- `status`
- `finding_digest`
- `generated_content_digest`
- `created_at`
- `findings`
- optional `applied_fix_summary`

State transitions:

```text
review running
  -> generate latest findings
  -> write handoff file atomically
  -> persist handoff state
  -> emit awaiting_approval or fix_review event

pending handoff
  -> validate file
  -> stamp processed metadata
  -> persist processed decisions
  -> dispatch approve/fix/skip response
```

Overwrite protection:

- If a pending file changed from the generated content digest before regeneration, preserve it as a timestamped backup or block regeneration with a stale-edit diagnostic.

## PR Audit Copy

The review handoff file included in the PR branch commit.

Fields:

- `source_relative_path`: from durable handoff state
- `work_area_relative_path`: same normalized path in the isolated work area
- `processed_action`
- `processed_at`

Validation rules:

- PR preparation must not re-run anchor discovery.
- Copy target must remain inside the isolated work area after canonicalization.
- Review-file-only changes are sufficient to create a commit.
- A passed review gate with pending latest required handoff metadata must fail before PR/push completion, except documented no-handoff automatic auto-fix paths.
