# Contract: Review Handoff File

## File Location

- Name: `review-issues-<run-short-id>.md`
- Preferred directory: next to exactly one eligible changed anchor file named `plan.md`, `task.md`, or `tasks.md`.
- Fallback directory: next to exactly one eligible anchor from the latest reviewed commit.
- Final fallback: `.no-mistakes/issues/<branch-slug>/`

Path resolution must reject absolute paths, traversal, `.git`, symlink escapes, and any path outside the checkout.

## Front Matter

```yaml
---
run_id: 01JZEXAMPLEFULL
run_short_id: 01JZEXAM
branch: feature/review-file-handoff
step: review
status: awaiting_approval
review_cycle_revision: review-round-2
review_result_hash: sha256:...
processed_at: null
processed_action: pending
---
```

Required initial values:

- `processed_at: null`
- `processed_action: pending`

Successful processing updates only these two fields.

## Finding Section

Each latest finding is rendered once and contains:

- `Issue`
- `Context`
- `Recommendation`
- `User Answer`

`Recommendation` includes one or two concrete options. Option 1 is the default when a `fix` response leaves `solution:` empty.

## Response Block

````text
```no-mistakes-response
id: review-1
action: fix
solution: |
  Use the existing parser helper and add stale-hash coverage.
```
````

Rules:

- `id` must match one latest finding.
- `action` is exactly `fix`, `accept`, or `skip`.
- `solution` is optional.
- Prose outside response blocks is ignored.
- Comment-only `solution` lines are ignored before deriving fixer instructions.

## Size Limits

- File: at most 1 MiB.
- One `solution:` value: at most 16 KiB.

## Final No-Findings State

When fix review has no remaining findings, overwrite the file with metadata, preserved prior decisions, applied fix summaries when available, and the literal final state:

```text
No remaining review findings.
```

In this state, latest-finding ID validation applies to an empty latest-finding set while metadata, hash, size, and parseability checks still apply.
