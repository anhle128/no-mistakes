# Quickstart: Review File Handoff

This quickstart is for the implementation pass after planning.

## 1. Confirm the Current Plan

```bash
sed -n '1,220p' specs/001-review-file-handoff/plan.md
sed -n '1,220p' specs/001-review-file-handoff/research.md
sed -n '1,220p' specs/001-review-file-handoff/data-model.md
```

## 2. Implement in Bounded Slices

Recommended order:

1. `internal/reviewhandoff`: path resolver, safe path checks, metadata model, writer, parser, canonical hash, validation, and metadata update.
2. `internal/pipeline`: generate handoff files before review gates, process files through existing approval decisions, mirror automation decisions, and preserve validation errors.
3. `internal/ipc`: add nullable review fields and review-only process request.
4. `internal/tui`: compact review gate, `p process`, `c cancel`, review phase labels, and validation error rendering.
5. `internal/cli`: AXI fields and legacy automation compatibility.
6. `internal/pipeline/steps`: final no-findings audit state and push inclusion.
7. `docs` and `skills/no-mistakes`: update user and agent guidance.

## 3. Targeted Validation

Run focused tests as each slice lands:

```bash
go test ./internal/reviewhandoff
go test ./internal/pipeline ./internal/pipeline/steps
go test ./internal/ipc ./internal/cli
go test ./internal/tui
```

Add or update tests for:

- safe anchor resolution and fallback path
- front matter and response block parsing
- deterministic hash mismatch rejection
- size limits
- byte-snapshot race rejection
- process-to-fix and process-to-approve behavior
- automation mirror before gate advancement
- compact review TUI controls
- AXI additive fields with raw status preservation
- final no-findings audit file
- push inclusion and anchor suppression

## 4. Full Validation

```bash
go test -race ./...
make lint
make docs-build
```

If e2e coverage is needed for daemon/TUI/AXI/git behavior:

```bash
go test -tags=e2e -count=1 -timeout 300s ./internal/e2e/...
```

## 5. Manual Smoke Scenario

1. Trigger a review gate that produces at least two findings.
2. Confirm TUI shows a compact review summary, relative review file path, `p process`, and `c cancel`.
3. Edit one response block to `fix` with non-empty `solution:` and another to `accept` or `skip`.
4. Press `p process`.
5. Confirm only selected fix findings are sent to the fixer.
6. Let fix review produce no remaining findings.
7. Confirm the final audit file preserves prior decisions and says `No remaining review findings.`
8. Let push prepare the PR branch commit.
9. Confirm the audit file is included and anchor files are not included unless they were normal pipeline changes.
