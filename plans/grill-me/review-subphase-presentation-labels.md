# Review Sub-Phase Presentation Labels

## Goal

Make the review phase easier to understand by showing clear sub-phase labels across user-facing surfaces while keeping the current pipeline behavior unchanged.

## Decisions

- Keep `auto_fix.review` behavior unchanged.
- Do not add a new pipeline step.
- Do not change DB schema.
- Do not change raw IPC or CLI `status` values.
- Do not change step order.
- Treat this as a presentation-layer change.
- Add one shared helper for review sub-phase labels.
- Apply the labels consistently across TUI, CLI/AXI, PR summaries, logs, docs, and tests.

## Phase Mapping

Only the `review` step has sub-phase labels in this first pass.

| Step | Status | Phase label |
| --- | --- | --- |
| `review` | `running` | `Review preview` |
| `review` | `awaiting_approval` | `Review preview complete` |
| `review` | `fixing` | `Fixing review issues` |
| `review` | `fix_review` | `Review fix result` |
| `review` | `completed` | empty |
| non-review steps | any status | empty |

Completed review steps should still render as `Review`.

## Implementation Shape

Create a small shared display helper package:

```go
package stepdisplay

func ReviewPhaseLabel(step types.StepName, status types.StepStatus) string
```

The helper should return an empty string when no review sub-phase label applies.

Use the helper as follows:

- TUI pipeline row: use the phase label instead of `stepLabel` when present.
- TUI action bar: use `Review preview complete:` or `Review fix result:` when present.
- TUI terminal title and active-step header: use the phase label when present.
- AXI/CLI run output: keep raw `status`, add a `phase` field.
- AXI/CLI gate output: keep raw `status`, add a `phase` field.
- PR summary/status detail: use the review phase label in human-facing copy.
- Review logs: align log wording with phase labels.

## AXI/CLI Contract

Do not replace machine-readable status values.

Example target shape:

```text
steps[4]{step,status,phase,findings,duration_ms}:
  intent,completed,,0,420
  rebase,completed,,0,1800
  review,awaiting_approval,Review preview complete,2,1234
  test,pending,,0,0
```

Gate output should similarly include both fields:

```text
gate:
  step: review
  status: awaiting_approval
  phase: Review preview complete
```

## Review Log Wording

Update review log text for consistency:

```text
review preview: reviewing changes...
fixing review issues...
```

This changes only user-facing log wording, not events or behavior.

## Non-Goals

- Do not split `review` into `review_preview`, `review_fix`, or `review_verify` pipeline steps.
- Do not add a live `review_verify` event in this pass.
- Do not distinguish `Auto-fixing review issues` from user-triggered `Fixing review issues`.
- Do not clear old findings while a fix is running.
- Do not add review phase labels to non-review steps.
- Do not change automation behavior for `axi drive`, `yolo`, or `auto_fix.review`.

## Test Strategy

Cover each changed surface, not only the helper:

- `internal/stepdisplay`: exact mapping tests.
- TUI pipeline row: `review` renders the four phase labels for `running`, `awaiting_approval`, `fixing`, and `fix_review`.
- TUI action bar: `Review preview complete:` and `Review fix result:` render correctly; `Applied fix:` remains visible for fix review.
- TUI terminal title/header: active review step uses the phase label.
- AXI/CLI run output: `steps` table includes `phase`; review rows get phase labels; non-review rows have empty phase.
- AXI/CLI gate output: includes `phase`.
- PR summary/status detail: review statuses use phase wording.
- Review logs: updated wording is covered where log wording is asserted.
- Docs: examples match the new labels.

Full e2e is not required for the first pass because backend behavior is unchanged, but package tests must cover each rendering surface.

## Stop Condition

The implementation is complete when:

- `auto_fix.review` behavior is unchanged.
- No new step names are introduced.
- DB schema and raw status contracts are unchanged.
- Review sub-phase labels render consistently across all agreed surfaces.
- Non-review steps are not relabeled.
- Completed review still renders as `Review`.
- Tests verify each surface.
- Build/tests pass.
