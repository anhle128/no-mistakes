# Review File Handoff Plan

Status: grill complete

## Target Result

Move review issue handling out of the terminal detail view and into a Markdown handoff file.

The review phase should:
- Find all review findings first.
- Write the latest findings and recommendations to one Markdown file.
- Show only a compact terminal summary, the review file path, and actions.
- Let the user edit answers in the file.
- Let the user return to terminal and press `p process` or `c cancel`.

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

Expose review file path where useful, but do not break existing AXI automation.

## Test Plan

Existing tests cover the old review prompt, terminal findings rendering, action bar, and AXI auto-resolution. New tests should cover only the new behavior:

- Path resolver: uncommitted anchor, committed anchor, multiple anchors fallback, no anchor fallback.
- Markdown writer: labels, header, response blocks, recommendation label, no findings final state.
- Parser: `fix`, `accept`, `skip`, empty solution default, comment stripping, invalid ID, missing block, invalid action.
- TUI review gate: shows summary/path only, `p process`, `c cancel`, old review keys hidden.
- Pipeline process mapping: fix IDs and instructions are passed to `ActionFix`; all accept/skip maps to `ActionApprove`.
- Push integration: final review file is copied into isolated worktree and included in the commit before PR.

## Non-Goals

- Do not change `auto_fix.review`.
- Do not add DB schema in v1.
- Do not auto-open an editor.
- Do not add manual regenerate in v1.
- Do not preserve old review cycle history in v1.
- Do not parse user-edited prose outside `no-mistakes-response` blocks.
