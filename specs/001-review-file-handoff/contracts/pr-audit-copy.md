# Contract: PR Audit Copy

## Source Of Truth

The PR/push path uses only `Review Handoff State.relative_path` from the review step. It must not re-run anchor discovery in the isolated work area.

## Copy Rules

Before a PR branch commit or push completion decision:

1. Load the latest review handoff state for the run's review step.
2. If no handoff exists because the run used documented no-handoff auto-fix behavior, continue.
3. Validate `processed_action` is not `pending`.
4. Validate `relative_path` is repository-relative and canonicalizes inside the source checkout.
5. Copy the file to the same normalized relative path inside the isolated work area.
6. Validate the destination canonicalizes inside the isolated work area immediately before writing.

## Publishable Artifact Allowlist

The commit builder may stage:

- intentional pipeline outputs
- configured in-repo evidence artifacts already allowed by existing evidence settings
- the normalized review handoff relative path

The commit builder must not stage:

- anchor `plan.md` or `tasks.md` merely because they were used for placement
- neighboring files in the anchor directory
- unrelated staged, modified, or untracked user files

## Review-File-Only Commit

If the copied review handoff file is the only remaining publishable change, the system still creates the commit so the PR includes the audit file.

## Failure Rules

Fail before PR/push completion when:

- a passed review gate has a latest required handoff file with `processed_action: pending`
- the persisted relative path escapes source checkout or work area
- the file is missing/unreadable and the path is required
- copy succeeds but staging would include files outside the allowlist

## Tests Required

- review file copied at same relative path
- review-file-only commit is created
- unrelated changed anchor is not staged
- neighboring files are not staged
- path traversal and symlink escape are rejected
- no-handoff automatic auto-fix path is exempted explicitly
