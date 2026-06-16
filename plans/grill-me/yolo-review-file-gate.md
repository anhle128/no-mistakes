# Yolo Review File Gate Decisions

Status: grill complete, implementation pending

## Source Issue

The current `001-review-file-handoff` implementation writes review issues to the handoff file, but TUI yolo mode does not auto-resolve review-file gates. Source currently short-circuits in `internal/tui/commands.go` when `m.isReviewFileGate(step.StepName)` is true.

This conflicts with the intended automation/audit contract:
- Yolo mode means the user has opted into auto-resolving paused gates.
- Existing automation responses must keep working without editing Markdown.
- Automation decisions must be mirrored into the same review handoff file before the gate advances.
- The review file remains the audit surface in both manual and yolo/automation flows.

## Decisions

1. Treat TUI yolo on a review-file gate as an automation decision, not as manual file processing.

   Recommended and accepted: TUI yolo should use the existing legacy response path, the same conceptual path as AXI `--yes`. It should not call `p process`.

2. Preserve the current yolo decision mapping.

   Recommended and accepted:
   - Initial review gate with actionable findings sends `ActionFix` with all finding IDs.
   - Initial review gate with only `no-op` findings sends `ActionApprove`.
   - `fix_review` sends `ActionApprove` to avoid an unbounded fix loop.

3. Do not parse the review file in yolo mode.

   Recommended and accepted: TUI yolo should derive IDs from live gate state (`FindingsJSON`), just like current yolo and AXI `--yes`. The file is the audit surface. Human-edited decisions remain owned by `p process`.

4. Do not invent per-finding solution text in TUI yolo.

   Recommended and accepted: yolo should send selected finding IDs only. The executor/fixer already has the structured finding data and recommendations. The mirrored audit file records which findings were fixed; custom solution text is only supplied by explicit user/file/automation instructions.

5. Implement by removing the TUI review-file-gate yolo guard and reusing `respondCmd`.

   Recommended and accepted: remove the early return that skips review-file gates in `maybeAutoApproveCmd`. Do not add a new `process_review --auto` RPC or special TUI-only code path.

6. Do not add new review-file gate controls for yolo.

   Recommended and accepted: manual UI remains compact with `p process` and `c cancel`. When yolo is on, the existing footer state (`y end yolo`) is enough; yolo will normally advance quickly after the gate event.

7. Add focused test coverage.

   Recommended and accepted:
   - Add or update `internal/tui/yolo_test.go` so a review-file gate still emits a `Respond` call under yolo.
   - Assert actionable review-file gates send `ActionFix` with all finding IDs.
   - Assert fix-review review-file gates send `ActionApprove`.
   - Rely on existing pipeline mirror tests for executor audit-file behavior, extending only if a decision is not asserted.

8. Preserve yolo convergence behavior.

   Recommended and accepted: even if a `fix_review` gate still has actionable findings, TUI yolo approves it after the first fix attempt, matching current behavior and avoiding infinite loops.

9. Mirror failures must block the gate.

   Recommended and accepted: if executor automation mirroring fails, yolo should surface the existing error and leave the review gate open. Do not add fallback behavior that advances without an audit record.

10. Treat this as a bug fix against the current implementation.

    Recommended and accepted: the spec already requires preserved yolo/automation behavior and audit mirroring. The current TUI guard is the mismatch; implementation and docs should be aligned with the spec.

## Documentation Impact

Update docs that currently state TUI yolo does not answer review-file gates automatically. The target wording should say TUI yolo auto-resolves review-file gates through the preserved automation response contract, and no-mistakes mirrors the executed decision into the same handoff file before continuing.

## Implementation Sketch

In `internal/tui/commands.go`, remove this branch from `maybeAutoApproveCmd`:

```go
if m.isReviewFileGate(step.StepName) {
	return nil
}
```

No new backend action is needed. The existing `respondCmd(types.ActionFix)` and `respondCmd(types.ActionApprove)` calls should continue to send the normal `RespondParams`; executor-side automation mirroring owns writing decisions into the handoff file.
