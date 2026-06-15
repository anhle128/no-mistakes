# Contract: Review Handoff File

## File Location

The generator writes one current file per run at the persisted normalized repository-relative path:

```text
<anchor-dir>/review-issues-<run-short-id>.md
```

If no single `plan.md` or `tasks.md` anchor is found, use:

```text
.no-mistakes/issues/<branch-slug>/review-issues-<run-short-id>.md
```

The path must be validated with symlink-aware canonicalization before write and before PR audit copy.

## Front Matter

The file starts with YAML front matter:

```yaml
---
no_mistakes_review_handoff: v1
run_id: run_123
step: review
status: awaiting_approval
branch: feature/review-handoff
review_cycle_id: review-1
finding_digest: sha256:012345
review_file: .no-mistakes/issues/feature-review-handoff/review-issues-run_123.md
processed_action: pending
processed_at:
---
```

Rules:

- `processed_action` is `pending` and `processed_at` is empty on generated files.
- Successful processing overwrites both fields before dispatching approval/fix.
- The parser rejects a file whose metadata does not match the active gate state and durable review handoff state.

## Finding Section Shape

Each latest finding renders exactly once:

````markdown
## review-1

### Issue

Unsafe parser fallback accepts malformed response blocks.

### Context

The current parser accepts duplicate fields.

### Recommendation

1. Reject duplicate fields before dispatch.
2. Add parser fixtures for duplicate action and solution lines.

### User Answer

```no-mistakes-review-response review-1
action: fix
solution: Reject duplicate fields in the response block parser.
```
````

Use `Recommendation` for agent guidance and `Solution` for user-authored fix text. Internal JSON field names such as `suggested_fix` remain unchanged.

## Response Fence Grammar

Opening fence:

````text
```no-mistakes-review-response <finding-id>
````

Body:

```text
action: fix|accept|skip
solution: <one-line text>
```

Closing fence:

````text
```
````

Parser rules:

- The tag is exact lowercase `no-mistakes-review-response`.
- There must be exactly one finding ID token after the tag.
- There must be exactly one `action:` line and exactly one `solution:` line.
- Field names are exact lowercase.
- Field value whitespace is trimmed.
- `solution:` may be empty.
- Unknown fields, duplicate fields, nested response fences, multiple IDs, missing IDs, multi-line continuations, and unsupported actions are validation errors.
- Prose outside response fences must not affect processing.

## Default Actions

Generated defaults map from existing finding action:

| Finding action | Response action |
| --- | --- |
| `auto-fix` | `fix` |
| `ask-user` | `accept` |
| `no-op` | `skip` |

For default or edited `fix` responses with empty or comment-only `solution`, processing uses trusted recommendation option 1 from the active/latest finding model or generated trusted metadata. It must not read editable recommendation prose from the Markdown body.

## Validation Failure Contract

On validation failure:

- no approval, skip, abort, or fix response is dispatched
- the user's file content is preserved
- the compact gate remains open
- terminal shows one concise error plus the review file path
- stale metadata errors include actual versus expected run/status/cycle/digest values when known

## No-Remaining-Findings File

When fix-review has no remaining findings, the file still contains front matter and a final state body:

- applied fix summary when available
- clear "no remaining review findings" message
- resolved-decision summary for the prior cycle
- response processing path that approves directly
