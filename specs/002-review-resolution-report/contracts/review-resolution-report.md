# Contract: Review Resolution Report

## Markdown Artifact Contract

Path:

```text
no-mistakes/<branch-slug>/review-resolution.md
```

The file is repo-local PR evidence. The push step must stage or force-add exactly this current-run artifact when it exists, even if `no-mistakes/` is ignored, and must not stage unrelated `no-mistakes/` files.

Required top-level shape:

```markdown
# Review Resolution Report

Report Format Version: 1

## Run Context

...

## Counts

...

## Resolved Issues

...

## Accepted Without Fix

...

## Informational / No Action Required

...

## Still Open Issues

...
```

Run Context fields:

- Run ID
- Repository identifier or path
- Branch
- Base commit
- Current/final head commit at generation time
- Review step status
- Report lifecycle state
- First generated timestamp
- Last refreshed timestamp
- Finalized timestamp, when applicable
- Repo-local report path

Per-issue required fields:

- Finding ID
- Severity
- File and line, or explicit unavailable marker
- Action
- Source
- Review round ID
- Selection source
- Description
- Context, suggested/proposed fix, risk level/rationale, and user instructions when available
- Outcome
- Outcome evidence and provenance
- Decision action, actor/source, timestamp, reason, and decision round ID when available
- Fix round ID
- Applied solution source
- Applied solution or attempted solution
- Rationale
- Changed files
- Fix commit SHA or no-commit reason
- Verification text
- Follow-up round ID and scope-equivalence note
- Verifier source
- Evidence reference
- Evidence quality

Content constraints:

- Escape Markdown controls in untrusted fields.
- Strip or summarize raw diffs, code fences, logs, transcripts, and code snippets.
- Redact common secret-like values.
- Truncate bounded fields with explicit markers.
- Treat the report byte limit as a full-detail budget: entries beyond the budget render as compact stubs that retain finding ID, outcome, and classification provenance.
- Preserve structured fields such as IDs, severity, action, source, file, and line.

## Metadata Contract

Local consumers read compact metadata rather than parsing Markdown.

JSON-equivalent shape:

```json
{
  "exists": true,
  "path": "/home/user/project/no-mistakes/feature/review-resolution.md",
  "status": "in_progress",
  "resolved_count": 1,
  "accepted_count": 1,
  "informational_count": 0,
  "still_open_count": 0,
  "report_version": "1",
  "entry_count": 2,
  "last_refreshed_at": 1792345678,
  "finalized_at": null,
  "last_refresh_result": "ok"
}
```

Allowed statuses:

- `in_progress`
- `final`
- `incomplete`
- `stale`
- `degraded`
- `evidence_unavailable`

Surface rules:

- AXI/TUI: may show `status`, counts, and repo-local `path`.
- PR body: may show `status`, counts, and the repo-relative `no-mistakes/<branch-slug>/review-resolution.md` path.
- PR body: must not show absolute local paths, report excerpts, or private filesystem details.
- Consumers must validate local Markdown content hash and source watermark before surfacing confident counts; mismatches render `stale`.
- Any nonzero `still_open_count`, `degraded`, `incomplete`, `stale`, or `evidence_unavailable` must use non-success wording.

## Review Fix Agent Output Contract

Existing responses remain valid:

```json
{
  "summary": "address review findings"
}
```

Extended responses:

```json
{
  "summary": "address review findings",
  "resolutions": [
    {
      "finding_id": "review-1",
      "applied_solution": "Added validation before using the configured path.",
      "why_this_solution": "Validation keeps the original flow while preventing invalid input from reaching Git.",
      "changed_files": ["internal/pipeline/steps/review.go"]
    }
  ]
}
```

Validation rules:

- `summary` remains required for backward compatibility.
- `resolutions` is optional.
- Each resolution requires non-empty `finding_id`, `applied_solution`, `why_this_solution`, and at least one `changed_files` entry.
- Duplicate IDs are degraded evidence.
- Unknown IDs are degraded evidence.
- Missing selected finding IDs are degraded evidence.
- Structured details are descriptive evidence only. They become applied-solution language only when tied to persisted fix-round evidence and a resolved or verified-attempt outcome.

## PR Summary Contract

When report metadata exists, the generated PR `## Pipeline` section may include one compact line such as:

```markdown
- Review resolution: final; 1 resolved, 1 accepted without fix, 0 informational, 0 still open. Report: `no-mistakes/feature/review-resolution.md`.
```

When no report metadata exists, omit review-resolution status.

Forbidden in PR body:

- absolute local paths
- report excerpts
- raw finding descriptions beyond existing pipeline summary behavior
- GitHub blob links to absolute local paths or uncommitted reports
