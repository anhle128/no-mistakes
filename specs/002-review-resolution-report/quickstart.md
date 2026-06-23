# Quickstart: Review Resolution Report

## Feature Goal

When a Review step records findings, no-mistakes creates a durable repo-local Markdown report plus compact SQLite metadata that explains each finding outcome: resolved, accepted without fix, informational, or still open.

## Developer Flow

1. Implement DB/repo-local path foundation.

```sh
go test ./internal/db ./internal/paths
```

2. Implement report classification and rendering.

```sh
go test ./internal/reviewreport
```

3. Wire Review lifecycle refresh and failure behavior.

```sh
go test ./internal/pipeline
```

4. Wire PR, IPC, AXI, and TUI compact surfaces.

```sh
go test ./internal/pipeline/steps ./internal/ipc ./internal/cli ./internal/tui
```

5. Run E2E scenarios for mixed, clean, aborted, structured-resolution, and legacy-summary flows.

```sh
go test -tags=e2e ./internal/e2e -run 'ReviewResolution|Axi'
```

6. Run full validation before landing.

```sh
go test -race ./...
make lint
make skill
```

## Manual Smoke Scenario

Use a temporary `NM_HOME` and a fake-agent or fixture-driven Review run.

Expected behavior for a run with two Review findings, one fixed and one explicitly approved:

- `no-mistakes/<branch-slug>/review-resolution.md` exists in the run checkout.
- SQLite has one `review_resolution_reports` row for the run.
- Markdown has exactly one entry per normalized finding ID.
- Counts agree across Markdown, metadata, AXI/TUI output, and PR summary.
- AXI/TUI show repo-local path and compact counts.
- PR body shows compact counts/status plus the repo-relative report path and does not include absolute checkout paths.

Expected behavior for a clean Review run:

- No report file exists.
- No `review_resolution_reports` row exists.
- AXI/TUI/PR omit review-resolution status.

Expected behavior for an aborted run with unresolved findings:

- Report exists if Review findings were recorded.
- Unresolved findings are under `Still Open Issues`.
- Metadata status is `incomplete` or `evidence_unavailable`, not `final`.
- No unresolved finding is shown as accepted solely because the run stopped.

## Acceptance Evidence

Before tasks are considered complete, collect:

- targeted test output for report package, DB, pipeline, PR summary, IPC, AXI, and TUI
- E2E output for at least one mixed resolved/accepted run and one aborted unresolved run
- full `go test -race ./...` output
- `make lint` output
- `make skill` output when generated skill text changes
