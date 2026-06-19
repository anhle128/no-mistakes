# Contract: Current Worktree Rendering

## Required Structured Fields

Structured surfaces must expose these fields for every run object where the
surface has structured output:

```json
{
  "worktree_mode": "current",
  "worktree_label": "uses this checkout",
  "work_dir_label": "no-mistakes/thread-21a29cc0",
  "current_worktree_warning": "This run uses this checkout; automated fixes may create commits here.",
  "metadata_availability": "recorded",
  "evidence_state": "complete",
  "terminal_reason": "",
  "run_report_ref": "run_123",
  "reported_findings": 2,
  "fixed_findings": 1,
  "unresolved_findings": 0,
  "skipped_findings": 0,
  "fix_commits": ["0123456789abcdef..."]
}
```

Allowed `worktree_mode` values:

- `isolated`
- `current`

Allowed `metadata_availability` values:

- `recorded`
- `not_recorded`
- `malformed`
- `stale`

Allowed `evidence_state` values:

- `complete`
- `incomplete`
- `degraded`

## Surface Requirements

### CLI start and terminal output

- On current-mode start/pre-start, print a warning that the run uses this
  checkout and automated fixes may create commits here.
- On checks-passed, passed, failed, cancelled, and stale recovery, include the
  current-mode warning and run/report reference.
- If fix commits occurred, include safe commit references and state that they
  remain in this checkout.

### AXI output

- Include all required structured fields in run objects.
- Use `work_dir_label` by default, not full absolute path.
- Full path may appear only in explicit verbose/debug fields.
- Missing or malformed current-mode fields must render `evidence_state` as
  `incomplete` or `degraded`, never as a normal passed isolated run.

### Status and runs output

- Label current-mode runs as "uses this checkout".
- Label isolated runs as "disposable no-mistakes checkout".
- Include the safe work-dir label once per run row/detail.
- Include terminal reason for setup-failed, stale-recovered, superseded,
  cancelled, and missing-base outcomes.

### TUI output

- Show current-mode warning in active run rendering.
- Preserve the warning around fix-in-progress, fix-review, checks-passed/passed,
  failure/cancellation, and stale recovery states.
- Avoid crowding the pipeline list with repeated full paths.

### Generated reports and PR summaries

- Include worktree mode and plain label.
- Include safe work-dir label.
- Include fix count and commit references when fixes occurred.
- Include unresolved or degraded evidence state.
- Include run/report reference.
- Do not inline raw logs, raw transcripts, secrets, long code excerpts, or diff
  hunks except in existing explicit diagnostic or approval-detail surfaces.

## Count Consistency

The same run must render consistent counts across AXI, status, TUI, generated
reports, and PR summaries:

- `reported_findings`
- `fixed_findings`
- `unresolved_findings`
- `skipped_findings`
- `approved_as_is_findings`
- `unavailable_findings`

Renderers should consume persisted count/provenance data rather than recompute
from mutable current worktree state.

## Path Privacy

Default rendering must minimize path detail:

- Prefer repo basename plus run ID or another stable compact label.
- Avoid repeated absolute paths across normal AXI, status, TUI, and PR-facing
  surfaces.
- Full canonical absolute paths are allowed only in verbose/debug diagnostics
  intended for local troubleshooting.

## Warning Text Requirements

Warning text can vary by surface, but must convey:

- The run uses the current checkout.
- Automated fixes may modify this checkout and create commits here.
- no-mistakes will not remove this checkout during cleanup or recovery.
- The run/report reference identifies the evidence trail once a run exists.
