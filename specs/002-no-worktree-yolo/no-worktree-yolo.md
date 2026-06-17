# No-Worktree YOLO Origin Reference

**Created**: 2026-06-17  
**Feature directory**: `specs/002-no-worktree-yolo`  
**Main spec**: `spec.md`

## Original Request

The user asked to create a detailed Speckit spec for `no-worktree-yolo.md`, save a reference so future tasks understand the origin and purpose, spawn a dedicated sub-agent for help, scout source code for context, and create the detailed spec.

## Inferred Purpose

Source scouting found no existing `no-worktree-yolo.md` file. The closest product need is to connect two existing product contracts:

- YOLO mode can auto-resolve paused approval gates when the user gives broad unattended consent.
- no-mistakes relies on disposable or bounded execution so automated fixes do not mutate the user's day-to-day checkout.

This spec therefore defines the requirement conservatively: unattended YOLO must run only when the system can prove a safe execution boundary. If the run is not in a disposable or bounded context, or the boundary is unknown, YOLO must fail closed and leave the gate for manual action.

## Source-Scout Findings

Dedicated source scout: `Turing` (`019ed3bf-aea6-78e1-9cc8-14c3661a67f0`).

Key findings:

- `README.md:33` and `README.md:50` describe the product as a local git proxy that runs the pipeline in a disposable worktree.
- `README.md:95` and `README.md:101` through `README.md:103` describe the user-facing gate paths and headless agent workflow.
- `.specify/memory/constitution.md:46` through `.specify/memory/constitution.md:54` require disposable-worktree or bounded-directory execution and human approval for intent-sensitive or side-effectful actions.
- `docs/src/content/docs/reference/pipeline-steps.md:12` through `docs/src/content/docs/reference/pipeline-steps.md:16` document YOLO behavior and the worktree steering boundary.
- `docs/src/content/docs/concepts/auto-fix.md:70` through `docs/src/content/docs/concepts/auto-fix.md:93` document `ask-user`, YOLO, and one-fix-round behavior.
- `skills/no-mistakes/SKILL.md:172` through `skills/no-mistakes/SKILL.md:179` describe `--yes` as broad consent for unattended gate driving.
- `internal/tui/commands.go:76` through `internal/tui/commands.go:99` implements current YOLO resolution: fix actionable findings once, approve fix review, approve no-op-only gates, and avoid duplicate terminal actions.
- `internal/tui/yolo_test.go:18` through `internal/tui/yolo_test.go:248` covers current YOLO toggle, auto-approve, auto-fix, no-op, and fix-review behavior.
- `internal/agent/steering.go:10` through `internal/agent/steering.go:24` steers agents to confine intentional writes to the worktree, with narrow evidence/temp exceptions.
- `internal/pipeline/steps/execution_context.go:3` through `internal/pipeline/steps/execution_context.go:23` explains that pipeline agents run inside an isolated git worktree and must treat it as the source of truth.

## Scope Decision

The spec intentionally does not define a new pipeline order, finding taxonomy, or PR behavior. It defines a safety gate around unattended YOLO consent. Safe isolated runs should keep today's YOLO behavior; unsafe or unknown-boundary runs should stay manual.

## Ambiguity Resolved

The phrase "no-worktree-yolo" could mean "allow YOLO without a worktree" or "block YOLO when no safe worktree is available." The selected interpretation is the safer one because it aligns with the constitution, README product promise, and existing worktree steering model.

## First Files For Future Implementation Planning

- `internal/tui/commands.go`
- `internal/tui/yolo_test.go`
- `internal/tui/keys.go`
- `internal/cli/axi_drive.go`
- `internal/cli/axi_render.go`
- `internal/boundary/policy.go`
- `internal/boundary/verifier.go`
- `internal/db/event.go`
- `internal/db/run.go`
- `internal/types/automation.go`
- `internal/daemon/manager.go`
- `internal/pipeline/executor.go`
- `internal/pipeline/pipeline.go`
- `internal/pipeline/steps/push.go`
- `internal/pipeline/steps/pr.go`
- `internal/pipeline/steps/ci_fix.go`
- `skills/no-mistakes/SKILL.md`
- `internal/skill/skill.go`
- `docs/src/content/docs/reference/pipeline-steps.md`
- `docs/src/content/docs/concepts/auto-fix.md`
- `docs/src/content/docs/reference/cli.md`
- `docs/src/content/docs/guides/troubleshooting.md`
- `internal/agent/steering.go`
- `internal/pipeline/steps/execution_context.go`
- `.specify/memory/constitution.md`

## Non-Goals

- Do not remove YOLO mode.
- Do not change normal `origin` behavior or the explicit `no-mistakes` gate remote.
- Do not reorder the fixed pipeline.
- Do not redefine `auto-fix`, `ask-user`, or `no-op`.
- Do not treat broad `--yes` consent as permission to ignore the isolation boundary.
