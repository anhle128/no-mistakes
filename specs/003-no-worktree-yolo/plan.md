# Implementation Plan: Current Worktree YOLO Mode

**Branch**: `003-no-worktree-yolo` | **Date**: 2026-06-18 | **Spec**: `specs/003-no-worktree-yolo/spec.md`
**Input**: Feature specification from `specs/003-no-worktree-yolo/spec.md`

**Command**: `/speckit-plan create detail plan specs/003-no-worktree-yolo`
**Origin reference**: `specs/003-no-worktree-yolo/no-worktree-yolo.md`
**Red team gate**: Satisfied by `specs/003-no-worktree-yolo/red-team-findings-applied-20260618-232433.md`

## Summary

Add an explicit current-worktree execution mode for no-mistakes. Users opt in with
`--no-worktree` on both the root `no-mistakes` command and `no-mistakes axi run`;
default behavior remains the existing disposable no-mistakes-owned worktree flow.
The new mode starts runs directly through CLI/daemon IPC, executes in the
canonical current git worktree root, preserves the full branch review scope, and
persists enough metadata for status, AXI, TUI, generated reports, PR summaries,
cleanup, and recovery to distinguish "uses this checkout" from "disposable
no-mistakes checkout".

`--yolo` is only an alias for existing `--yes` auto-resolution behavior. It does
not add approval authority and must be accepted alongside `--yes`.

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, SQLite, YAML, TOON, Git, provider CLIs already supported by no-mistakes
**Storage**: SQLite under `NM_HOME`, existing gate bare repos, existing disposable worktree store for isolated runs, current git worktree for current-mode runs, run logs/evidence directories
**Testing**: Focused `_test.go`, `go test -race ./...`, `make lint`, tagged `internal/e2e` coverage for cross-process AXI/daemon/git flows
**Target Platform**: macOS, Linux, Windows CLI/daemon
**Project Type**: Go CLI/daemon with docs site and generated agent skill
**Performance Goals**: Current-mode start should avoid the gate-remote push trigger and avoid creating an extra worktree; normal `git push no-mistakes` and isolated AXI/root flows remain unchanged
**Constraints**: `origin` behavior remains unmodified; pipeline order remains intent, rebase, review, test, document, lint, push, PR, CI; current mode must reject unsafe starts before pipeline execution; warnings are visible but non-blocking under `--yes`/`--yolo`; current work directories are never removed by no-mistakes cleanup
**Scale/Scope**: Local per-user daemon managing multiple initialized repos and branch-scoped runs, with possible linked worktrees for the same repo

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Explicit Gate Semantics**: PASS. Current-worktree mode keeps the fixed pipeline and preserves normal `origin` behavior. Review scope is strengthened by requiring a trustworthy default-branch merge base for the full branch diff before execution.
- **Isolation and User Control**: PASS WITH EXPLICIT EXCEPTION. The default remains disposable isolated execution. Current mode is opt-in through `--no-worktree`, rejects dirty starts, labels that the run uses this checkout, and never cleans up the user's current work directory.
- **Evidence-First Quality**: PASS. Plan requires targeted unit coverage for flags, preflight, IPC, DB migration, compatibility, rendering, cleanup, and recovery, plus `go test -race ./...`, `make lint`, and e2e coverage for daemon/AXI/git behavior.
- **Agent-Agnostic Contracts**: PASS. AXI/status/TUI/PR output gets stable structured fields, path minimization, schema validation, and fail-closed degraded states for missing or malformed current-mode metadata.
- **Simplicity and Recovery**: PASS. The smallest design is a mode-aware extension of existing run start, DB, IPC, and rendering contracts. Current-mode cleanup is disabled by data contract, not by best-effort path heuristics.
- **Docs and Generated Artifacts**: PASS. CLI docs, agent guide, generated `skills/no-mistakes/SKILL.md`, and PR-facing summary text are in scope because the feature adds flags and a new execution boundary.

No unjustified constitution violations are present.

## Project Structure

### Documentation (this feature)

```text
specs/003-no-worktree-yolo/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── current-worktree-run.md
│   └── current-worktree-rendering.md
├── no-worktree-yolo.md
├── clarifications-applied-2026-06-18-225855.md
├── red-team-findings-applied-20260618-232433.md
└── tasks.md              # Produced later by /speckit-tasks, not this command
```

### Source Code (repository root)

```text
cmd/no-mistakes/          # Cobra entrypoint tests for flag wiring
cmd/genskill/             # Generated agent skill source if skill output is regenerated from code
internal/cli/             # root, attach, AXI start/drive/render/status/runs surfaces
internal/daemon/          # direct start, run manager lifecycle, stale recovery, IPC handlers
internal/db/              # run metadata, migrations, fix provenance, compatibility data
internal/git/             # current worktree root, dirty preflight, base refresh helpers
internal/ipc/             # start-run and run-info wire contracts
internal/pipeline/        # executor workDir boundary, skip/fix evidence, run events
internal/pipeline/steps/  # full branch diff/base usage, PR summary, fix provenance
internal/tui/             # current-mode labels, warnings, degraded evidence rendering
internal/e2e/             # cross-process current-worktree AXI and regression journeys
skills/no-mistakes/       # generated agent guidance
docs/src/content/docs/    # CLI, daemon/pipeline, TUI, agents, troubleshooting docs
```

**Structure Decision**: Keep the implementation inside existing CLI, daemon,
DB, IPC, git, pipeline, TUI, docs, and skill packages. Add no new dependency.
Add new types in `internal/types` only for shared enums/state names. Do not
create a parallel pipeline for current mode; make run start and cleanup
mode-aware while reusing `pipeline.Executor.Execute(workDir)`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| None | N/A | N/A |

## Phase 0 Research Summary

See `research.md` for decisions and alternatives. The key decisions are:

- Persist `worktree_mode` as `isolated|current` and keep user-facing labels separate.
- Add a direct daemon IPC start path for current mode instead of using a gate-remote push.
- Resolve and persist the canonical current git worktree root before run creation becomes recoverable.
- Resolve and persist trustworthy review-base evidence before current-mode execution starts.
- Make cleanup mode-aware and fail closed when new-format metadata is malformed or unavailable.
- Add structured rendering fields and path-minimized labels across AXI, status, TUI, PR summaries, and docs.
- Persist fix outcome/provenance records for current-worktree committed fixes.

## Phase 1 Design Summary

See `data-model.md` and `contracts/`.

Primary model changes:

- `Run` gains `worktree_mode`, `work_dir`, `work_dir_label`, review-base evidence, metadata availability, terminal reason, optional successor fields, and count/provenance inputs for report reconstruction.
- `StartRequest` captures branch, head, review base, worktree mode, current work directory, trigger, approval mode, skip steps, intent source, and caller provenance.
- `ActiveRunCompatibility` compares repo, branch, head, selected worktree mode, current-mode work directory, review base, and immutable start-shape fields before allowing resume or drive.
- `FixOutcome` records proposed, attempted, committed, and failed fix states with source finding, actor/source, decision type, and commit SHA when available.

Primary contract changes:

- CLI: root and AXI accept `--no-worktree`; root and AXI accept `--yolo` as alias for `--yes`; both flags together are valid.
- IPC: add a direct start/run request for current mode and extend `RunInfo` with mode, safe work-dir label, warning, metadata/degraded state, review-base evidence, terminal reason, and fix/count summaries.
- Rendering: status, AXI, TUI, generated reports, and PR summaries expose stable structured fields and plain labels while avoiding repeated full absolute path disclosure.

## Implementation Approach

1. Add shared types and persistence.
   - Add `types.WorktreeMode` with `isolated` and `current`.
   - Add run metadata columns through additive migrations.
   - Keep pre-existing runs with no metadata as `isolated`.
   - Treat new-format missing/malformed current metadata as degraded/incomplete and cleanup-disabled.

2. Add current-worktree preflight helpers.
   - Resolve canonical git worktree root with symlink normalization.
   - Reject uninitialized repos, detached/unborn/default branch, tracked changes, and untracked non-ignored files.
   - Allow ignored-only local files.
   - Resolve review base using default branch merge base; attempt one non-interactive default-branch ref refresh and reject with `rejected_no_trustworthy_base` if still unproven.

3. Add direct start and compatibility flow.
   - Extend daemon IPC with a start request that can choose `worktree_mode`.
   - For isolated mode, retain existing push/rerun/wizard flows.
   - For current mode, create the run with mode/workdir/base metadata atomically before recoverability, load repo config from the current root, and call `Executor.Execute` with that root.
   - Replace automatic cancellation of active same-branch runs with compatibility validation and whitelisted conflict output.

4. Make cleanup and recovery mode-aware.
   - Remove no-mistakes-owned worktrees only for isolated runs.
   - Never remove `worktree_mode=current` directories.
   - On setup failure, panic, cancellation, normal completion, and stale recovery, persist terminal reason and incomplete/degraded evidence state when appropriate.

5. Update rendering and report contracts.
   - Add structured fields to IPC and AXI/status output.
   - Add TUI and CLI warning placements for start/pre-start, active rendering, fix in progress/review, checks-passed/passed, failure/cancellation, and stale recovery.
   - Update PR summaries and generated reports with current mode, safe label, fix commit references, unresolved/degraded evidence state, and run/report reference.

6. Update docs and generated skill guidance.
   - Document both command forms, `--no-worktree`, and `--yolo` as an alias only.
   - Update agent guidance to use `axi run --intent "..." --no-worktree --yolo` only when the user explicitly wants current-checkout execution.

## Validation Plan

Targeted tests before broad verification:

- CLI parsing tests for root and AXI `--no-worktree`, `--yolo`, and `--yes --yolo`.
- Git/preflight tests for clean feature branch, subdirectory invocation, dirty tracked files, untracked non-ignored files, ignored-only files, detached HEAD, default branch, unborn/uninitialized repo, and missing default-branch base.
- DB migration tests for old runs defaulting to isolated, current runs atomically persisting mode/workdir/base metadata, invalid metadata rendering degraded, and cleanup disabled for degraded current-mode rows.
- IPC protocol tests for direct current-mode start request, extended `RunInfo`, active-run conflict errors, and schema round trips.
- Daemon manager tests for current-mode execution directory, no managed worktree creation, compatibility/resume/reject behavior, setup failure, panic, cancellation, stale recovery, and isolated regression cleanup.
- Rendering tests for AXI/status/TUI/PR summaries: required fields present, safe label used, warning lifecycle present, full paths minimized, and incomplete/degraded evidence distinct from passed.
- Fix provenance tests for committed fix SHA capture and applied-fix claims derived from records instead of prose summaries.
- E2E tests for `no-mistakes axi run --intent "..." --no-worktree --yolo` from a clean feature branch and for existing isolated root/wizard/push/AXI flows.

Required final commands after implementation:

```sh
gofmt -w <changed-go-files>
go test -race ./...
make lint
make e2e
```

If docs-only or generated-skill-only changes are isolated in later phases, run
the docs build or record the reason it is intentionally skipped.

## Post-Design Constitution Re-Check

- **Explicit Gate Semantics**: PASS. Current mode still runs the fixed pipeline and persists skip/degraded evidence distinctly.
- **Isolation and User Control**: PASS. The exception is explicit, labeled, clean-start-only, and cleanup-disabled for the current checkout.
- **Evidence-First Quality**: PASS. Cross-boundary tests and e2e coverage are identified before implementation.
- **Agent-Agnostic Contracts**: PASS. Stable fields and fail-closed schema handling are defined in contracts.
- **Simplicity and Recovery**: PASS. Existing executor/pipeline is reused; mode-aware run start and cleanup carry the new behavior.
- **Docs and Generated Artifacts**: PASS. User-visible docs and generated guidance are planned.

## Extension Hooks

**Optional Pre-Hook**: git
Command: `/speckit-git-commit`
Description: Auto-commit before implementation planning

Prompt: Commit outstanding changes before planning?
To execute: `/speckit-git-commit`

**Optional Hook**: git
Command: `/speckit-git-commit`
Description: Auto-commit after implementation planning

Prompt: Commit plan changes?
To execute: `/speckit-git-commit`
