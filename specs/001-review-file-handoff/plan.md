# Implementation Plan: Review File Handoff

**Branch**: `001-review-file-handoff` | **Date**: 2026-06-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-review-file-handoff/spec.md`

**Command**: `/speckit-plan create detail plan specs/001-review-file-handoff`

## Summary

Move review-gate detail and user decisions out of the constrained terminal view and into one durable Markdown handoff file per run. The implementation will add a shared review-handoff domain package, persist a typed additive review-handoff state on the review step result, generate and validate strict response blocks, expose additive `phase` and `review_file` fields to live events, reattached run state, and AXI status output, and copy the latest processed handoff file into the PR branch through an explicit publishable-artifact allowlist.

The existing review step, raw statuses, response commands, auto-fix ordering, and executor approval transition remain intact. The terminal review gate becomes a compact process/cancel surface, while AXI, yolo, and direct IPC responses continue to resolve through the same executor transition and record equivalent audit decisions.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**: Existing Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, SQLite via `modernc.org/sqlite`, `gopkg.in/yaml.v3`, TOON, existing git/provider helpers
**Storage**: SQLite under `NM_HOME`; additive `step_results.review_handoff_json` metadata; generated Markdown file inside the project checkout or isolated pipeline work area; disposable gate worktrees
**Testing**: Focused Go unit tests for parser/path/state/IPC/TUI/AXI/push behavior, tagged e2e coverage for process/cancel and PR audit inclusion, then `gofmt`, `go test -race ./...`, and `make lint`
**Target Platform**: macOS, Linux, Windows CLI/daemon
**Project Type**: Go CLI/daemon with terminal UI, AXI automation output, git gate pipeline, docs site, and generated agent skill content
**Performance Goals**: Correctly process a valid 20-finding handoff in one terminal action; no explicit latency SLA beyond keeping TUI/AXI responsive
**Constraints**: Preserve `origin`; preserve fixed pipeline order and raw statuses; preserve automatic review auto-fix behavior; do not require incompatible run-history migration; do not parse user decisions from prose outside response blocks; keep writes inside checkout/worktree boundaries with symlink-aware validation
**Scale/Scope**: One current review handoff per run across review and fix-review cycles, with durable processed-decision audit data surviving handoff regeneration

## Constitution Check

*GATE: Passed before Phase 0 research. Re-check after Phase 1 design below.*

- **Explicit Gate Semantics**: PASS. The design preserves the existing review step, step order, raw statuses (`awaiting_approval`, `fixing`, `fix_review`, `completed`), executor approval actions, and `origin` behavior. Terminal `p process`, reattached sessions, AXI responses, yolo auto-resolution, and direct IPC `respond` calls converge on the existing executor decision switch.
- **Isolation and User Control**: PASS. Handoff files are written only under the project checkout or isolated pipeline work area after canonical path validation. User-authored response blocks are validated before any approval or fix dispatch. Cancel remains bound to the original run/step gate identity and uses the existing abort path.
- **Evidence-First Quality**: PASS. The verification plan names parser, path resolver, DB migration/state, executor, TUI, AXI, PR audit, docs, and tagged e2e coverage, followed by `gofmt`, `go test -race ./...`, and `make lint`.
- **Agent-Agnostic Contracts**: PASS. Existing internal `suggested_fix` data remains stable while user-facing labels become `Recommendation` and `Solution`. Additive `phase` and `review_file` fields are specified for IPC and AXI without renaming raw fields or response commands.
- **Simplicity and Recovery**: PASS. The plan adds one shared `internal/reviewhandoff` package and one additive typed JSON state column rather than new pipeline steps or broad schema reshaping. Atomic file writes and persisted content digests protect pending user edits and recovery after missed live events.
- **Docs and Generated Artifacts**: PASS. TUI, AXI, pipeline-step docs, README or skill guidance as needed, and generated `skills/no-mistakes` output are included in scope because the workflow is user-visible.

## Project Structure

### Documentation (this feature)

```text
specs/001-review-file-handoff/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── automation-surfaces.md
│   ├── pr-audit-copy.md
│   ├── review-gate-actions.md
│   └── review-handoff-file.md
└── checklists/
    └── requirements.md
```

### Source Code (repository root)

```text
internal/reviewhandoff/      # Markdown generation, strict parser, validation, path resolver, state structs
internal/types/              # Shared review phase helper/constants if they must be public to CLI/TUI/IPC
internal/db/                 # Additive review_handoff_json column, typed get/set helpers, migration tests
internal/ipc/                # Add phase/review_file fields to events and StepResultInfo
internal/pipeline/           # Generate/persist handoff before approval events; process file into executor response
internal/pipeline/steps/     # Push/PR audit copy and publishable-artifact allowlist integration
internal/tui/                # Compact review gate rendering and process/cancel key handling
internal/cli/                # AXI status run/gate fields and compatible respond behavior
internal/e2e/                # Cross-process review handoff journey coverage
docs/src/content/docs/       # TUI, AXI, pipeline-step, and troubleshooting docs
skills/no-mistakes/          # Regenerated agent guidance when AXI workflow text changes
```

**Structure Decision**: Create a new `internal/reviewhandoff` package that imports only low-level packages such as `internal/types` and standard library/YAML helpers. Keep DB persistence in `internal/db`, IPC projection in `internal/ipc`/`internal/daemon`, terminal behavior in `internal/tui`, and publish/copy behavior in `internal/pipeline/steps`. This avoids import cycles and keeps the parser/generator independently testable.

## Phase 0 Research Output

See [research.md](./research.md).

Key decisions:

- Use `internal/reviewhandoff` for generation, parsing, validation, path resolution, digesting, and typed state structs.
- Persist a typed JSON state blob in an additive `step_results.review_handoff_json` column. Derive `phase` from review step status rather than persisting display text.
- Define the handoff file as YAML front matter plus strict `no-mistakes-review-response <finding-id>` fenced blocks.
- Treat response blocks as the only parsed user-decision source. Empty/comment-only fix solutions fall back to option 1 from the active/latest finding model or validated trusted metadata, not editable prose.
- Preserve existing automatic review auto-fix precedence. Generate handoffs only when the existing executor would otherwise pause for human review.
- Publish the PR audit copy through an explicit allowlist that includes intentional pipeline outputs and the persisted normalized review-file relative path.

## Phase 1 Design Output

See [data-model.md](./data-model.md) and contracts under [contracts/](./contracts/).

Generated contracts:

- [review-handoff-file.md](./contracts/review-handoff-file.md): front matter, response block grammar, validation, metadata, trust boundary.
- [automation-surfaces.md](./contracts/automation-surfaces.md): IPC event fields, reattached run state, AXI run rows, AXI gate output, omission rules.
- [review-gate-actions.md](./contracts/review-gate-actions.md): TUI/AXI/IPC/yolo/process compatibility matrix and transition mapping.
- [pr-audit-copy.md](./contracts/pr-audit-copy.md): normalized relative-path copy, publishable-artifact allowlist, pending-state guard.

## Implementation Strategy

1. Add `internal/reviewhandoff` with:
   - typed state, metadata, response, decision, and validation error structs
   - strict block parser and Markdown generator
   - finding-set digest and generated-content digest helpers
   - path resolver for changed `plan.md`/`tasks.md` anchors and fallback `.no-mistakes/issues/<branch-slug>/review-issues-<run-short-id>.md`
   - canonical boundary checks for anchors, destinations, and audit-copy paths
2. Add DB support:
   - additive `review_handoff_json` column on `step_results`
   - typed `SetStepReviewHandoff`/`StepReviewHandoff` helpers
   - migration tests proving older DBs remain readable
3. Wire review approval transitions:
   - after a review/fix-review outcome is known and before marking the step awaiting approval, generate/overwrite the handoff file, persist state, then emit events
   - process action validates the current file, atomically stamps processed metadata, records decisions, then dispatches `RespondWithOverrides`
   - automation/yolo/direct IPC responses keep working and update equivalent audit state with source `automation` or `auto_fix`
4. Update surfaces:
   - IPC `Event` and `StepResultInfo` additive `phase` and `review_file`
   - TUI compact review gate with file path, summary, validation error, and only `p process` plus `c cancel`
   - AXI run/gate output with additive columns/fields and unchanged `respond` help
   - docs and generated skill guidance
5. Update publish path:
   - copy the persisted review-file relative path into the isolated work area before no-op commit decisions
   - stage only explicit publishable artifacts and the review file, not anchor files or neighboring working-tree changes
   - fail before PR/push completion if a passed review gate still has a pending latest required handoff file, except documented no-handoff auto-fix paths

## Verification Plan

Targeted unit coverage:

- `internal/reviewhandoff`: parser accepts valid 20-finding handoff; rejects duplicate/unknown/missing IDs, duplicate fields, unknown fields, uppercase actions, nested fences, multiline continuations, missing metadata, stale cycle/digest, processed non-pending state, empty fix fallback without option 1, and ignores prose outside blocks.
- `internal/reviewhandoff`: generator emits `Issue`, `Context`, `Recommendation`, `User Answer`, strict response blocks, pending processed metadata, final no-remaining-findings state, and resolved-decision summaries.
- `internal/reviewhandoff`: path resolver handles one uncommitted anchor, one latest committed anchor, zero/multiple anchors fallback, symlink anchors, non-regular anchors, path traversal, branch slugging, and relative path display.
- `internal/db`: migration and typed JSON state round trips.
- `internal/ipc` and `internal/daemon`: live events and reattached `StepResultInfo` include additive `phase` and `review_file` when appropriate and omit them for non-review steps.
- `internal/pipeline`: review auto-fix still runs before handoff generation; handoff write/persist/event ordering; validation failure leaves gate open and preserves file; processed metadata update happens before dispatch; partial failure keeps last committed state.
- `internal/tui`: review gates hide approve/fix/skip/edit/add/toggle controls, show only process/cancel, show validation error plus path, and preserve non-review gate behavior.
- `internal/cli`: AXI status run rows and gate output include `phase`/`review_file`; `axi respond approve|fix|skip` remains accepted and records audit source.
- `internal/pipeline/steps`: audit-copy allowlist stages the review file without staging `plan.md`/`tasks.md` anchors; review-file-only changes create a commit; pending latest required handoff blocks PR/push completion.

Cross-process/e2e coverage:

- Generate a review gate with two findings, edit one `fix` and one `accept`, process through TUI/IPC, and verify only the fixed finding reaches remediation with custom solution.
- Malformed/stale handoff processing blocks, keeps compact gate open, and preserves file content.
- Reattached TUI/AXI status after missed live event shows the persisted review file and phase.
- Successful run includes the latest processed review file in the PR branch commit, including review-file-only commit scenario.
- Automatic review auto-fix remains no-handoff and no-process until the existing behavior reaches a human decision point.

Final validation:

```text
gofmt on changed Go files
go test -race ./...
make lint
docs build or documented skip if docs tooling is unavailable
```

## Post-Design Constitution Re-Check

- **Explicit Gate Semantics**: PASS. Contracts keep raw statuses and response commands stable and define one shared review-gate decision transition.
- **Isolation and User Control**: PASS. The design validates generated/edit paths and binds process/cancel to the original run/step identity.
- **Evidence-First Quality**: PASS. Verification includes targeted unit tests and e2e coverage for gate/daemon/approval/PR boundaries.
- **Agent-Agnostic Contracts**: PASS. IPC/AXI/TUI contracts are additive and preserve existing automation semantics.
- **Simplicity and Recovery**: PASS. One shared package and one additive JSON state column minimize schema churn while supporting recovery.
- **Docs and Generated Artifacts**: PASS. Docs and skill outputs are explicitly in scope.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Review handoff files may be written in the user's project checkout beside a changed `plan.md` or `tasks.md` anchor before the isolated PR-copy stage. | The handoff is intentionally user-edited evidence for the active review gate, and the spec requires the developer to open/edit it from the checkout while PR publishing still copies only the persisted review file through the isolated work area. | Forcing all handoffs into the disposable worktree would hide the file from the developer's day-to-day checkout; forcing every handoff into `.no-mistakes/issues/<branch-slug>/` would remove the requested anchor-local review context. The narrower boundary is symlink-aware canonicalization plus an explicit publishable-artifact allowlist that stages only the review file, not anchors or neighboring files. |
