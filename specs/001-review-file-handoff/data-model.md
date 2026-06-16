# Data Model: Review File Handoff

## ReviewHandoffFile

One Markdown file representing the current review decision surface for a pipeline run.

**Identity**

- `path`: deterministic safe path inside the run worktree.
- `file_name`: `review-issues-<run-short-id>.md`.
- `run_id`: owning pipeline run.

**Fields**

- `metadata`: `ReviewHandoffMetadata`
- `summary`: finding count and severity counts.
- `findings`: ordered `ReviewFindingEntry` list for the current review result.
- `responses`: ordered `ResponseBlock` list, one per latest finding.
- `audit_entries`: optional prior-cycle decisions shown in the final no-findings state.
- `final_state`: optional final state text, including `No remaining review findings.`

**Validation**

- Exactly one current file per run.
- File size is at most 1 MiB before parsing.
- Path must remain inside the checkout after cleaning/resolution.
- Current-finding state must match the live review gate hash.

## ReviewHandoffMetadata

YAML front matter at the top of the file.

**Fields**

- `run_id`: full run ID.
- `run_short_id`: short stable run ID used in the filename.
- `branch`: run branch name.
- `step`: always `review`.
- `status`: current review step status, normally `awaiting_approval` or `fix_review`.
- `review_cycle_revision`: deterministic revision from the current persisted review round.
- `review_result_hash`: canonical hash computed from live authoritative gate state.
- `processed_at`: `null` initially; RFC3339 timestamp after successful processing or automation mirroring.
- `processed_action`: `pending` initially; set to `fix`, `approve`, or `skip` according to the executed gate decision.

**Validation**

- Front matter must exist and parse as YAML.
- `run_id`, `step`, `status`, and `review_result_hash` must match live gate state.
- Initial hand-edited processing requires `processed_at: null` and `processed_action: pending`.
- File-supplied `review_result_hash` is never authoritative; it is only compared to a recomputed live value.

## ReviewFindingEntry

A rendered finding from the latest review result.

**Fields**

- `id`: existing structured finding ID from `types.Finding.ID`.
- `severity`: `error`, `warning`, `info`, or an unknown value rendered as plain text.
- `file`: optional file path.
- `line`: optional one-indexed line.
- `issue`: short issue title from `description`.
- `context`: explanatory context.
- `recommendations`: one or two concrete recommendation option strings.
- `default_response_action`: `fix`, `accept`, or `skip`.

**Validation**

- Every latest finding must have an ID.
- IDs must be unique.
- Normalized latest finding content participates in `review_result_hash`.

## ResponseBlock

The machine-readable user decision block embedded in Markdown.

**Fence**

````text
```no-mistakes-response
id: review-1
action: fix
solution: |
  Use the existing recommendation.
```
````

**Fields**

- `id`: canonical latest finding ID.
- `action`: exactly `fix`, `accept`, or `skip`.
- `solution`: optional multiline string.

**Validation**

- Parse only fenced `no-mistakes-response` blocks.
- Ignore all prose outside the blocks.
- Every latest finding has exactly one block.
- No unknown IDs.
- No duplicated IDs.
- `solution` is parseable YAML and at most 16 KiB.
- Comment-only lines inside `solution` are ignored before deriving fixer instructions.

## ProcessedReviewDecision

The authoritative decision derived from a valid handoff file or an existing automation response.

**Fields**

- `source`: `file` or `automation`.
- `executed_action`: `approve`, `fix`, or `skip`.
- `selected_finding_ids`: ordered finding IDs selected for fixing.
- `instructions`: per-finding untrusted solution text.
- `added_findings`: user-authored findings from existing automation response, if any.
- `processed_at`: timestamp written into metadata.

**Rules**

- Any `fix` response produces `executed_action: fix` with only those finding IDs.
- Empty `solution` for `fix` uses recommendation option 1.
- Non-empty `solution` is delivered to the fixer as delimited untrusted data scoped to that one finding ID.
- All `accept` or `skip` responses produce `executed_action: approve`.
- Existing automation `approve`, `fix`, and `skip` are mirrored into the file before the executor advances.

## ReviewPhaseLabel

Human-facing label derived from the review step status.

| Step | Raw status | Label |
|------|------------|-------|
| review | running | Review preview |
| review | awaiting_approval | Review preview complete |
| review | fixing | Fixing review issues |
| review | fix_review | Review fix result |
| review | completed/skipped/failed | null |
| non-review | any | null |

Raw status values remain unchanged.

## ValidationError

Short actionable error displayed when processing fails.

**Fields**

- `path`: review file path.
- `summary`: one-line failure summary.
- `first_error`: first actionable validation error.

**Behavior**

- Processing stays blocked.
- Gate remains open.
- TUI review gate keeps only `p process` and `c cancel`.
- AXI/IPC returns a structured error without changing raw statuses.

## ReviewAuditFile

The final handoff file committed to the PR branch.

**Fields**

- Current metadata and hash.
- Preserved prior finding decisions from `step_rounds`.
- Applied fix summaries when available.
- Final `No remaining review findings.` state when fix review has no findings.

**Regeneration source**

- Current run and review step state.
- `step_rounds.findings_json`
- `step_rounds.user_findings_json`
- `step_rounds.selected_finding_ids`
- `step_rounds.selection_source`
- `step_rounds.fix_summary`

**Validation**

- If the processed audit file is missing or unreadable when push prepares the PR branch commit, regenerate it from persisted decisions.
- If regeneration is impossible, block push with an explicit audit-file error.
