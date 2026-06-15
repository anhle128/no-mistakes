# Review File Handoff And Presentation Labels Plan

Status: grill complete, merged with review sub-phase labels

## Target Result

Move review issue handling out of the terminal detail view and into a Markdown handoff file, while using consistent review sub-phase labels across user-facing surfaces.

The review phase should:
- Find all review findings first.
- Write the latest findings and recommendations to one Markdown file.
- Show only a compact terminal summary, the review file path, and actions.
- Let the user edit answers in the file.
- Let the user return to terminal and press `p process` or `c cancel`.
- Show clear review sub-phase labels in TUI, CLI/AXI, PR summaries, logs, docs, and tests.

`auto_fix.review` remains unchanged.

## Problem

The current review gate streams and renders findings directly in the terminal as soon as review finds them. This makes the user wait and watch the terminal to know whether more issues will appear, and the terminal is not a good review surface for issue details plus human answers.

## Source Facts

- `internal/pipeline/steps/review.go` currently runs review and, in fixing mode, applies fixes and reviews again inside the same `ReviewStep`.
- Review findings already include `description`, `context`, `suggested_fix`, and `action`.
- `internal/pipeline/executor.go` handles `ActionFix` by selecting finding IDs, merging user instructions, setting `sctx.Fixing = true`, and re-executing the step.
- `accept` and `skip` do not exist as backend per-finding actions. In v1, both mean "do not include this finding in ActionFix".
- `internal/tui/keys.go` currently uses review gate keys such as `a approve`, `f fix`, `s skip`, `e edit`, `+ add`, `A all`, and `N none`.
- `internal/pipeline/steps/push.go` commits in the isolated pipeline worktree before PR creation.
- `internal/pipeline/steps/pr.go` creates the PR after push; it does not create the commit.

## Presentation Label Decisions

- Do not add a new pipeline step.
- Do not change DB schema.
- Do not change raw IPC or CLI `status` values.
- Do not change step order.
- Treat review sub-phase names as presentation-layer labels.
- Add one shared helper for review sub-phase labels.
- Apply labels consistently across TUI, CLI/AXI, PR summaries, logs, docs, and tests.

## Review Sub-Phase Labels

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

Create a small shared display helper package:

```go
package stepdisplay

func ReviewPhaseLabel(step types.StepName, status types.StepStatus) string
```

The helper returns an empty string when no review sub-phase label applies.

Use the helper for:
- TUI pipeline row: use the phase label instead of `stepLabel` when present.
- TUI action bar: use `Review preview complete:` or `Review fix result:` when present.
- TUI terminal title and active-step header: use the phase label when present.
- AXI/CLI run output: keep raw `status`, add a `phase` field.
- AXI/CLI gate output: keep raw `status`, add a `phase` field.
- PR summary/status detail: use the review phase label in human-facing copy.
- Review logs: align log wording with phase labels.

## Review File Location

Create `review-issues-<run-short-id>.md`.

Location resolver:
1. Look only at changed files.
2. First check uncommitted changed files in the project checkout, including staged, modified, and untracked files.
3. If exactly one changed file has basename `plan.md` or `task.md`, place the review file next to it.
4. If none exists in uncommitted changes, check the latest commit being reviewed.
5. If exactly one changed committed file has basename `plan.md` or `task.md`, place the review file next to it.
6. If zero or multiple candidates exist, place the file in `.no-mistakes/issues/<branch-slug>/review-issues-<run-short-id>.md`.

`plan.md` or `task.md` is only an anchor. Do not automatically commit it.

## Markdown Format

Use Markdown. Do not use `Solution` or `Suggestion` labels. Use:
- `Issue`
- `Context`
- `Recommendation`
- `User Answer`

The reviewer still writes to the existing `suggested_fix` schema field, but prompt and rendering must treat it as `Recommendation`.

`Recommendation` must contain 1-2 concrete options. Option 1 is the preferred/best solution.

Example:

````markdown
<!-- no-mistakes-review
run_id: run_123
step: review
status: awaiting_approval
branch: feature/example
processed_at:
processed_action:
-->

# Review Issues

Review file: review-issues-run123.md

## Summary

Findings: 2 total, 1 high, 1 warning

## Finding review-1

### Issue

Hardcoded API key is committed in public HTML.

### Context

`admin-web-wallet/index.html` is served to every client, so the key is exposed.

### Recommendation

1. Move the key to a deployment-provided value loaded through Vite HTML env substitution.
2. If the key is no longer used, remove it and rotate the exposed key.

### User Answer

```no-mistakes-response id=review-1
action: fix
solution:
  # Leave empty to use Recommendation option 1, or write your chosen solution here.
```
````

## Response Block Rules

Backend parses only fenced blocks with language `no-mistakes-response`.

Supported actions:
- `fix`: send this issue to the fix agent.
- `accept`: accept the risk and do not fix.
- `skip`: ignore/not relevant and do not fix.

Default action mapping from reviewer finding action:
- `auto-fix` -> `fix`
- `ask-user` -> `accept`
- `no-op` -> `skip`

For `action: fix`:
- Empty `solution:` means use Recommendation option 1.
- Non-empty `solution:` overrides and is passed as backend instruction for that finding ID.
- Parser ignores comment lines beginning with `#` inside `solution:`.

Backend v1 treats `accept` and `skip` the same operationally. They are kept separate for audit readability only.

## Validation

On `p process`, validate:
- Review file exists and is readable.
- Header exists.
- `run_id` matches current run.
- `step` is `review`.
- `status` matches current review gate status: `awaiting_approval` or `fix_review`.
- Every latest finding ID has one response block.
- No unknown finding IDs are present.
- `action` is one of `fix`, `accept`, `skip`.
- `solution:` format is parseable.

If validation fails, block processing, show a short terminal error plus path, and keep the gate open.

Do not regenerate the file during `p process`; regeneration could overwrite user answers.

## Terminal Behavior

Review gate terminal should not render full issue details.

Show compact summary only:

```text
Review preview complete
Findings: 3 total, 1 high, 2 warning
Review file: plans/foo/review-issues-a1b2c3.md

p process   c cancel
```

Rules:
- Use the shared review phase label for the terminal title/header and compact summary heading.
- `p process` reads and validates the Markdown file.
- `c cancel` maps to backend `ActionAbort`.
- Remove review gate use of `a approve`, `f fix`, `s skip`, `e edit`, `+ add`, `A all`, and `N none`.
- Non-review gates keep their existing behavior.
- Display relative file paths when the file is inside the project checkout.
- On read/parse error, include a short quoted path hint if useful.

## Process Mapping

When `p process` succeeds:
- If one or more response blocks have `action: fix`, send backend `ActionFix` with only those finding IDs.
- Map each `solution:` to `instructions[findingID]`.
- If all response blocks are `accept` or `skip`, send backend `ActionApprove`.
- No extra confirmation is needed when all issues are accepted/skipped.

After successful process:
- Keep the user-edited file content.
- Update only small metadata such as `processed_at` and `processed_action`.
- Log `Processed review file: <path>`.

## Review Cycles

Use one current handoff file per run.

For normal review and `fix_review`:
- Use the same file path.
- Overwrite the file when a new review result is generated.
- Do not preserve previous `User Answer` blocks when a new review result arrives.

If `fix_review` has no remaining findings:
- Overwrite the file with final state.
- Include metadata, applied fix summary if available, and `No remaining review findings.`
- `p process` approves directly.

No manual regenerate command in v1.

No history files in v1.

## Commit And PR Behavior

The review issue file should be added when the pipeline commits and creates the PR.

Because source shows the commit happens in Push step before PR step:
- Copy the final review issue file from the project checkout into the isolated pipeline worktree during Push step.
- Preserve the same relative path.
- Let the existing Push step `git add -A` include it in the PR branch commit.
- If the review issue file is the only remaining change, still create the commit so the PR contains the audit file.
- Do not commit `plan.md` or `task.md` merely because it was used as an anchor.

## IPC And Reattach

Add an IPC field for review file path, for example `review_file`.

Backend should create the file and expose the path through IPC events/run info.

Reattach should compute the same deterministic path if the original event was missed. Do not add DB schema in v1.

## AXI/CLI Contract

Keep existing automation contract:
- `respond --action approve|fix|skip`
- `--findings`
- `--instructions`
- `--add-finding`

Expose review file path and phase label where useful, but do not break existing AXI automation.

Do not replace machine-readable status values.

Example target run shape:

```text
steps[4]{step,status,phase,review_file,findings,duration_ms}:
  intent,completed,,,0,420
  rebase,completed,,,0,1800
  review,awaiting_approval,Review preview complete,plans/foo/review-issues-a1b2c3.md,2,1234
  test,pending,,,0,0
```

Gate output should similarly include raw status, phase, and review file:

```text
gate:
  step: review
  status: awaiting_approval
  phase: Review preview complete
  review_file: plans/foo/review-issues-a1b2c3.md
```

## Review Log Wording

Update review log text for consistency:

```text
review preview: reviewing changes...
fixing review issues...
```

This changes only user-facing log wording, not events or behavior.

## Test Plan

Existing tests cover the old review prompt, terminal findings rendering, action bar, and AXI auto-resolution. New tests should cover only the new behavior:

- Path resolver: uncommitted anchor, committed anchor, multiple anchors fallback, no anchor fallback.
- Markdown writer: labels, header, response blocks, recommendation label, no findings final state.
- Parser: `fix`, `accept`, `skip`, empty solution default, comment stripping, invalid ID, missing block, invalid action.
- TUI review gate: shows summary/path only, `p process`, `c cancel`, old review keys hidden.
- `internal/stepdisplay`: exact phase label mapping tests.
- TUI pipeline row: `review` renders phase labels for `running`, `awaiting_approval`, `fixing`, and `fix_review`.
- TUI action bar: `Review preview complete:` and `Review fix result:` render correctly; `Applied fix:` remains visible for fix review.
- TUI terminal title/header: active review step uses the phase label.
- AXI/CLI run output: `steps` table includes `phase` and `review_file`; review rows get phase labels; non-review rows have empty phase.
- AXI/CLI gate output: includes `phase` and `review_file`.
- Pipeline process mapping: fix IDs and instructions are passed to `ActionFix`; all accept/skip maps to `ActionApprove`.
- Push integration: final review file is copied into isolated worktree and included in the commit before PR.
- PR summary/status detail: review statuses use phase wording.
- Review logs: updated wording is covered where log wording is asserted.
- Docs: examples match the new labels and review file path behavior.

Full e2e is not required for the first pass because backend behavior is unchanged, but package tests must cover each rendering surface.

## Non-Goals

- Do not change `auto_fix.review`.
- Do not add DB schema in v1.
- Do not split `review` into `review_preview`, `review_fix`, or `review_verify` pipeline steps.
- Do not add a live `review_verify` event in this pass.
- Do not distinguish `Auto-fixing review issues` from user-triggered `Fixing review issues`.
- Do not clear old findings while a fix is running.
- Do not add review phase labels to non-review steps.
- Do not change automation behavior for `axi drive`, `yolo`, or `auto_fix.review`.
- Do not auto-open an editor.
- Do not add manual regenerate in v1.
- Do not preserve old review cycle history in v1.
- Do not parse user-edited prose outside `no-mistakes-response` blocks.

## Stop Condition

The implementation is complete when:
- `auto_fix.review` behavior is unchanged.
- No new step names are introduced.
- DB schema and raw status contracts are unchanged.
- Review sub-phase labels render consistently across all agreed surfaces.
- Review gate terminal shows only summary, review file path, and `p process` / `c cancel`.
- Non-review steps are not relabeled.
- Completed review still renders as `Review`.
- Latest review findings and user responses round-trip through the Markdown handoff file.
- Review file is included in the PR branch commit when the pipeline reaches push.
- Tests verify each changed surface.
- Build/tests pass.
