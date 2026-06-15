# Contract: Automation Surfaces

This feature adds fields only. Existing raw statuses, step names, finding IDs, response commands, and finding row fields remain unchanged.

## Review Phase Values

| Step | Raw status | `phase` |
| --- | --- | --- |
| `review` | `running` | `Review preview` |
| `review` | `awaiting_approval` | `Review preview complete` |
| `review` | `fixing` | `Fixing review issues` |
| `review` | `fix_review` | `Review fix result` |
| `review` | `completed` | omitted |
| non-review | any | omitted |

`review_file` is the persisted repository-relative path from review handoff state. It is omitted when unknown or not applicable.

## Live IPC Event

For review gate events, `ipc.Event` adds:

```json
{
  "phase": "Review preview complete",
  "review_file": ".no-mistakes/issues/feature/review-issues-run123.md"
}
```

Rules:

- Fields are top-level and additive.
- Fields are included for review `awaiting_approval`, `fixing`, and `fix_review` when the handoff state is known.
- `phase` may appear for review `running` even before a handoff file exists.
- Non-review events omit both fields.

## Reattached Run State

`ipc.StepResultInfo` adds:

```json
{
  "phase": "Review preview complete",
  "review_file": ".no-mistakes/issues/feature/review-issues-run123.md"
}
```

Rules:

- Values are loaded from raw step status plus durable review handoff state.
- Reattach must not require a live event to have been observed.
- Missing legacy state omits `review_file` without failing run loading.

## AXI Status Run Output

The `steps` rows add optional columns for review steps:

```toon
steps[1]{step,status,phase,review_file,findings,duration_ms}:
  review,awaiting_approval,Review preview complete,.no-mistakes/issues/feature/review-issues-run123.md,2,1234
```

Rules:

- Existing `step`, `status`, `findings`, and `duration_ms` columns retain meaning.
- Non-review rows leave `phase` and `review_file` empty or omit them according to TOON row compatibility chosen by implementation tests.

## AXI Status Gate Output

The `gate` object adds:

```toon
gate:
  step: review
  status: awaiting_approval
  phase: Review preview complete
  review_file: .no-mistakes/issues/feature/review-issues-run123.md
  summary: 2 findings
  risk: medium
```

Rules:

- Existing `summary`, `risk`, `findings`, and help fields remain.
- For review gates, help keeps `axi respond` compatibility and may also mention file processing for humans.
- Non-review gates omit `phase` and `review_file`.

## TUI Reattach

The TUI uses `StepResultInfo.phase` and `StepResultInfo.review_file` when present. If the live gate event was missed, reattached state must still show the compact review file path from durable state.
