# Implementation Plan: No-Worktree YOLO Guard

**Branch**: `002-no-worktree-yolo` | **Date**: 2026-06-17 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-no-worktree-yolo/spec.md`

**Plan command**: `/speckit-plan create detail plan specs/002-no-worktree-yolo`

## Summary

Unattended YOLO gate handling must be allowed only when the controller can prove the run is operating inside the disposable source worktree created for that run. The implementation will add a controller-owned execution-boundary verifier, persist the latest boundary classification and gate automation events on the run, extend IPC/CLI/TUI/agent guidance with boundary-aware status, and enforce the guard at daemon/pipeline decision points so UI or agent-side consent cannot bypass it.

Safe isolated runs keep the existing YOLO behavior: actionable findings are fixed once, fix-review gates are approved, no-op-only gates are approved, and duplicate automatic responses are suppressed. Unsafe or unknown-boundary runs fail closed for unattended fix/approve/skip, git push, PR create/update, and provider review-advancing writes while preserving explicit per-gate manual actions.

## Technical Context

**Language/Version**: Go 1.25.
**Primary Dependencies**: Existing dependencies only: Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, SQLite via `modernc.org/sqlite`, TOON output, Git CLI, and provider CLIs through the existing `internal/scm` adapters.
**Storage**: SQLite under `NM_HOME`, local bare gate repositories under `repos/`, disposable run worktrees under `worktrees/<repo>/<run>`, run logs/evidence directories, and docs/generated skill files.
**Testing**: Targeted `_test.go` coverage plus `go test -race ./...`, `make lint`, tagged e2e for cross-process gate behavior, and docs/skill checks when user guidance changes.
**Target Platform**: macOS, Linux, and Windows CLI/daemon.
**Project Type**: Go CLI/daemon with TUI, AXI/headless CLI, provider adapters, docs site, and generated agent skill.
**Performance Goals**: Boundary verification is local Git/path inspection and must not add provider/network latency to gate polling. TUI and AXI polling remain responsive; long provider work stays in existing pipeline steps.
**Constraints**: Normal `origin` behavior remains unchanged; `git push no-mistakes` stays the explicit gate entrypoint; pipeline order remains fixed; source writes are confined to the verified run worktree except configured evidence/temp allowances; unsafe/unknown unattended actions must fail closed; no new dependencies.
**Scale/Scope**: Local per-user daemon managing multiple initialized repos and branch-scoped runs. The feature affects run lifecycle, approval/gate automation, remote-advancing steps, persisted run status, TUI, AXI/headless output, generated agent guidance, and docs.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Initial Gate

- **Explicit Gate Semantics**: PASS. The feature blocks unsafe unattended advancement but does not change `origin`, the `no-mistakes` gate remote, or pipeline order.
- **Isolation and User Control**: PASS. Safe unattended source-changing work requires a verified disposable run worktree; unsafe or unknown boundaries leave the gate/manual decision path intact.
- **Evidence-First Quality**: PASS. The plan requires targeted boundary verifier, DB, IPC, TUI, AXI, daemon/pipeline, and e2e coverage before implementation is complete.
- **Agent-Agnostic Contracts**: PASS. IPC carries boundary and gate automation status so TUI, AXI/headless, terminal status, and generated agent guidance render the same state.
- **Simplicity and Recovery**: PASS. The design reuses existing run/step orchestration and adds narrow verifier/policy helpers plus additive schema fields/events.
- **Docs and Generated Artifacts**: PASS. Generated agent skill and docs must describe allowed, withheld, and manual recovery paths.

### Post-Design Gate

- **Explicit Gate Semantics**: PASS. Remote-advancing steps remain in the same order; unsafe/unknown boundaries create/retain manual decision points instead of silently advancing provider state.
- **Isolation and User Control**: PASS. Automatic paths re-check boundary proof immediately before action. Manual responses are recorded with actor/surface/source metadata.
- **Evidence-First Quality**: PASS. See [quickstart.md](quickstart.md) for targeted and full validation commands.
- **Agent-Agnostic Contracts**: PASS. See [contracts/yolo-boundary.md](contracts/yolo-boundary.md) for persisted, IPC, CLI, TUI, and guidance contracts.
- **Simplicity and Recovery**: PASS. Additive migrations and idempotent event recording preserve recovery across daemon restart/reattach.
- **Docs and Generated Artifacts**: PASS. User-facing behavior changes require docs and `skills/no-mistakes/SKILL.md` updates during implementation.

No constitution violations require justification.

## Project Structure

### Documentation (this feature)

```text
specs/002-no-worktree-yolo/
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   `-- yolo-boundary.md
|-- checklists/
|   `-- requirements.md
|-- no-worktree-yolo.md
`-- spec.md
```

### Source Code (repository root)

```text
internal/db/                 # Add run boundary fields, run/gate event persistence, migration tests
internal/git/                # Reuse Git root/common-dir/worktree helpers for verifier inputs
internal/boundary/           # New narrow verifier/policy package for execution-boundary proof
internal/ipc/                # Carry boundary and gate automation status through RunInfo/events/respond params
internal/daemon/             # Classify runs at creation, refresh before unattended responses, record events
internal/pipeline/           # Central approval and auto-fix enforcement; StepContext boundary status
internal/pipeline/steps/     # Guard push, PR create/update, CI auto-fix push, and provider write paths
internal/tui/                # TUI YOLO withholding, manual action rendering, regression tests
internal/cli/                # AXI --yes withholding, status/run output, regression tests
internal/e2e/                # Tagged safe and unsafe/unknown YOLO journeys
skills/no-mistakes/          # Generated agent guidance for boundary-aware unattended consent
docs/src/content/docs/       # User docs for YOLO allowed/withheld/manual recovery behavior
```

**Structure Decision**: Add one small internal boundary/policy surface and keep enforcement tied to existing daemon, executor, step, IPC, TUI, and AXI paths. Do not introduce a new pipeline step, reorder existing steps, or add a dependency. Provider write suppression belongs at both policy entry points and concrete write callsites because UI-only checks cannot satisfy the safety contract.

## Complexity Tracking

No constitution violations or complexity exceptions.

## Phase 0 Research Summary

Resolved in [research.md](research.md):

- Controller-owned verifier inputs and failure modes.
- Persisted Run boundary state and gate event shape.
- Authoritative enforcement placement across daemon, executor, and remote write callsites.
- Stable gate identity for duplicate unattended-response prevention across reattach/restart.
- User-facing withheld-output and recovery text contract.
- Practical verification method for the 95% comprehension success criterion.

## Phase 1 Design Summary

Generated artifacts:

- [data-model.md](data-model.md): Run boundary fields, execution boundary, YOLO consent, gate decision, run event, finding, and origin reference entities with validation/state transitions.
- [contracts/yolo-boundary.md](contracts/yolo-boundary.md): IPC, CLI/AXI, TUI, generated agent guidance, persistence, and provider-write contracts.
- [quickstart.md](quickstart.md): Implementation sequence, smoke checks, and validation commands.

Implementation readiness:

- All planning clarifications are resolved in `research.md`.
- Contracts are additive/backward-compatible where possible. Legacy respond calls default to explicit manual source unless a caller marks the response as unattended.
- Full implementation should proceed through `/speckit-tasks` before code changes.
