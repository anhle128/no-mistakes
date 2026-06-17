# Contract: Review Resolution Report IPC Metadata v1

This contract defines the compact metadata projected from SQLite to daemon IPC, AXI, TUI, and PR summary code. It intentionally does not expose the full report body.

## Type: ReviewResolutionReportInfo

```go
type ReviewResolutionReportInfo struct {
    Path          string         `json:"path,omitempty"`
    Status        string         `json:"status"`
    LatestOutcome string         `json:"latest_outcome"`
    SummaryCounts map[string]int `json:"summary_counts,omitempty"`
    UpdatedAt     int64          `json:"updated_at,omitempty"`
    GeneratedAt   int64          `json:"generated_at,omitempty"`
    Stale         bool           `json:"stale,omitempty"`
    Error         string         `json:"error,omitempty"`
}
```

## Projection Points

Add this field to `ipc.RunInfo`:

```go
ReviewResolutionReport *ReviewResolutionReportInfo `json:"review_resolution_report,omitempty"`
```

Add this field to `ipc.Event` for run and review-step updates:

```go
ReviewResolutionReport *ReviewResolutionReportInfo `json:"review_resolution_report,omitempty"`
```

`StepResultInfo` does not need to own the report in v1 because the report is run-scoped. Review-step TUI details may render the run-level metadata when the active or selected step is `review`.

## Allowed Status Values

- `current`
- `stale`
- `unavailable`
- `error`

## Allowed Latest Outcome Values

- `no issues remain`
- `unresolved findings remain`
- `no reviewable changes`
- `awaiting user decision`
- `final findings unavailable`
- `final findings unreadable`
- `review data inconsistent`
- `review resolution incomplete`

## Summary Counts

When `SummaryCounts` is present, it MUST include every canonical key:

- `total_findings`
- `actionable_findings`
- `selected_for_fix`
- `fix_attempts`
- `applied_fix_summaries`
- `accepted`
- `skipped`
- `informational`
- `deferred`
- `still_open`
- `unavailable`
- `decision_not_recorded`

## Direct Display Limits

AXI and TUI may directly display:

- `Path`
- `Status`
- `LatestOutcome`
- `SummaryCounts`
- `UpdatedAt`
- `Stale`
- `Error`

AXI and TUI MUST NOT directly display full finding context, user instructions, raw fix-summary chains, raw logs, raw transcripts, or raw diff content from this metadata path. Detailed review-resolution content belongs in the sanitized Markdown report.

PR summaries may directly include:

- report reference;
- summary counts;
- latest outcome;
- sanitized material applied-fix summaries;
- omitted lower-severity or unavailable summary counts;
- stale/unavailable/error labels.

## Backward Compatibility

The field is optional. Existing clients that ignore unknown JSON fields must continue to work. New renderers must handle missing metadata as `unavailable` without failing the run.
