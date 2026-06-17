# Quickstart: Review Resolution Report

This quickstart is for implementers and reviewers after `/speckit-tasks` generates task-level work.

## 1. Read The Contracts

Start with:

```sh
sed -n '1,260p' specs/002-review-resolution-report/contracts/review-resolution-report-markdown.md
sed -n '1,220p' specs/002-review-resolution-report/contracts/review-resolution-report-ipc.md
```

The Markdown contract defines exact headings, count keys, labels, and extraction rules. The IPC contract defines the compact metadata that AXI, TUI, and PR surfaces may display directly.

## 2. Implement In Testable Slices

Recommended order:

1. Add path helpers and DB metadata persistence.
2. Add pure `internal/reviewreport` derivation and Markdown rendering.
3. Wire executor report updates around review findings, decisions, fix attempts, final review evidence, and generation errors.
4. Project metadata through daemon/IPC.
5. Update AXI, TUI, PR summaries, docs, and generated skill if needed.

## 3. Required Scenario Coverage

Targeted tests should cover:

- one review finding selected for fix, with applied fix summary and clean follow-up review;
- partial selection where unselected actionable findings are not mislabeled as `Accepted`;
- selected finding with user instructions;
- user-authored finding selected for fix;
- fix round with no summary rendered as `fix applied, no summary recorded`;
- multiple fix rounds in chronological order;
- informational-only review findings;
- no reviewable changes after ignore patterns;
- invalid or unreadable final findings JSON;
- selected finding IDs that do not match source findings;
- report-generation failure after review data has been captured;
- cancelled/failed/superseded run after fix but before trustworthy final review evidence;
- regeneration from legacy data missing selected IDs, user instructions, or fix summaries;
- stale regeneration attempt older than current report metadata;
- sanitizer redacts or replaces diff-like, code-like, log-like, transcript-like, and secret-like content;
- AXI status/run/respond includes report reference/counts/outcome when a report exists or review resolution state is non-empty;
- TUI review gate details expose report reference/counts/latest outcome without duplicating unsafe details;
- PR summary includes counts, latest outcome, material sanitized fix summaries, omitted-summary count, and report reference;
- cross-surface counts match persisted report metadata.

## 4. Targeted Test Commands

Use package-level tests while iterating:

```sh
go test ./internal/reviewreport ./internal/db -run 'ReviewResolution|Report' -count=1
go test ./internal/ipc ./internal/daemon -run 'ReviewResolution|Report' -count=1
go test ./internal/cli ./internal/tui ./internal/pipeline/steps -run 'ReviewResolution|Report|PipelineSummary|ApplyEvent|Axi' -count=1
```

If package names or test names differ after implementation, keep the intent: run the smallest tests that prove the changed report derivation, persistence, projection, and rendering paths.

## 5. Full Validation

Before claiming implementation complete:

```sh
go test -race ./...
make lint
make docs-build
```

If AXI guidance changes, also run:

```sh
make skill
make lint
```

`make lint` includes generated skill drift checks.

## 6. Manual Smoke Expectations

For a run with review fixes, user-facing surfaces should show:

- report path/reference;
- status (`current`, `stale`, `unavailable`, or `error`);
- latest outcome;
- stable summary counts;
- applied fix summary count and material summaries in PR output;
- explicit stale/error warning when generation failed.

The durable report should let a user identify the original issue, selected resolution, applied fix summary, and latest review outcome in under 60 seconds without reading raw logs.
