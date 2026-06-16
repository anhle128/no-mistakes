# Contract: IPC and AXI Review Fields

## IPC Additions

Add nullable fields to review-capable step/event payloads:

```go
ReviewPhaseLabel *string `json:"review_phase_label,omitempty"`
ReviewFilePath   *string `json:"review_file_path,omitempty"`
```

These fields are additive. Existing `status`, `step_name`, `findings_json`, `fix_summaries`, and response fields remain unchanged.

## Phase Label Mapping

| Step | Raw status | `review_phase_label` |
|------|------------|----------------------|
| review | running | `Review preview` |
| review | awaiting_approval | `Review preview complete` |
| review | fixing | `Fixing review issues` |
| review | fix_review | `Review fix result` |
| review | completed/skipped/failed | null |
| non-review | any | null |

## Review File Path Field

`review_file_path` is non-null when a review handoff file exists or can be deterministically recovered for the run. When the path is inside the project checkout, human-facing surfaces display it relative to the checkout.

## AXI Rendering

AXI run/gate output must:

- Preserve raw `status` values.
- Add `review_phase_label` when applicable.
- Add `review_file_path` when applicable.
- Preserve old `no-mistakes axi respond --action approve|fix|skip` behavior.
- Keep enough structured finding data for existing automation to select IDs for legacy `fix` responses.

## TUI Rendering

For review gates with a handoff file, TUI must display:

- compact finding summary
- review file path
- validation error when present
- `p process`
- `c cancel`

TUI must not display full inline finding details or legacy review controls for that review gate:

- `a approve`
- `f fix`
- `s skip`
- `e edit`
- `+ add`
- `A all`
- `N none`

Non-review approval gates keep existing controls.
