# No-Worktree YOLO Origin Reference

**Created**: 2026-06-18  
**Feature directory**: `specs/003-no-worktree-yolo`  
**Main spec**: `spec.md`  
**Source requirement**: `plans/grill-me/no-worktree-yolo.md`

## Original Request

The user asked to read the requirement in `no-worktree-yolo.md`, create a detailed Speckit spec, preserve a reference so the next phase knows the origin and purpose, spawn sub-agents for help, scout the source code for context, and create the detailed spec.

## Source Requirement Summary

The source requirement is `plans/grill-me/no-worktree-yolo.md`. It asks for an explicit current-working-tree execution mode:

- `no-mistakes --no-worktree --yolo`
- `no-mistakes axi run --intent "..." --no-worktree --yolo`

`--no-worktree` means no-mistakes must not create an additional no-mistakes-owned worktree under the no-mistakes worktree store. Instead, the pipeline runs in the root of the current git worktree, which may be an Archon-created worktree or a normal checkout.

The implemented contract keeps explicit intent on the start path: new current-worktree runs start through `no-mistakes axi run --intent "..." --no-worktree`. Root `no-mistakes --no-worktree --yolo` has no `--intent` flag, so it fails before new-run creation with guidance to use AXI rather than starting with empty, generic, or inferred intent.

`--yolo` is only an alias for existing `--yes` behavior. It does not grant new approval behavior.

## Purpose For The Next Phase

The next Speckit phase should plan and implement current-worktree execution while preserving the existing isolated default. The core purpose is not to make YOLO more permissive. It is to avoid nesting a no-mistakes-owned worktree inside workflows where another tool has already created the correct branch worktree.

Planning should treat these as the fixed product decisions:

- Default runs stay isolated.
- Current-worktree mode is opt-in via `--no-worktree`.
- New current-worktree starts use AXI with explicit intent.
- The root command fails before new current-worktree run creation when explicit intent would be required.
- Current mode starts directly through CLI/daemon IPC rather than a gate-remote push.
- Current mode reviews the full branch diff against the default branch base.
- Current mode persists and displays `worktree_mode` and `work_dir`.
- Current mode never deletes or cleans up the current work directory.
- Current-mode failure leaves generated commits in the user/tool-owned worktree.
- `--yolo` is exactly an alias for `--yes`.

## Older Remote Branch To Avoid Confusing

The remote branch `origin/002-no-worktree-yolo` already contains a Speckit directory named `specs/002-no-worktree-yolo`, but that spec describes a different "YOLO guard" interpretation: blocking unattended YOLO unless a disposable boundary can be proven.

This feature intentionally follows the local source requirement in `plans/grill-me/no-worktree-yolo.md` instead. Do not use the older remote guard interpretation as the source of truth for this feature.

## Sub-Agent Context

Two sub-agents were spawned for this specify phase:

- `Halley` (`019edb68-1740-7073-a36c-9d95918ec929`) scouted implementation context for worktree handling, YOLO/approval behavior, CLI/config surfaces, data model gaps, and likely tests.
- `Cicero` (`019edb68-2cbf-7191-8e47-2efc92281672`) scouted planning/spec conventions and recommended preserving origin-purpose details in a companion reference.

## Source-Scout Findings

Primary source facts from `plans/grill-me/no-worktree-yolo.md` and the implementation scout:

- Root CLI currently has `--yes` and `--skip`; it calls the shared attach path without worktree-mode selection.
- `no-mistakes axi run` currently has `--yes`, `--skip`, and `--intent`; it requires `--intent` for new AXI starts.
- The shared attach path is wizard-focused when no active run exists and does not branch on current-worktree mode.
- Existing repo lookup already handles linked worktrees by falling back to the registered main repo record when needed.
- Existing git helpers distinguish the current git worktree root from the main repo root.
- The daemon start path currently always creates a no-mistakes-owned worktree and loads repo config from that created directory.
- Worktree cleanup is currently tied to no-mistakes-owned worktree paths and must be mode-aware before current mode can ship.
- The pipeline executor already accepts an explicit work directory, so the run execution boundary can move without changing every step.
- The run database schema currently lacks `worktree_mode` and `work_dir`.
- IPC run metadata currently lacks worktree mode and path.
- AXI and TUI rendering currently do not expose worktree mode.
- Recovery currently removes managed stale run worktrees and must never remove current-mode work directories.

## First Files For Future Implementation Planning

- `internal/cli/root.go`
- `internal/cli/attach.go`
- `internal/cli/axi_drive.go`
- `internal/cli/axi_render.go`
- `internal/cli/root_test.go`
- `internal/cli/axi_drive_test.go`
- `internal/daemon/manager.go`
- `internal/daemon/manager_test.go`
- `internal/daemon/daemon.go`
- `internal/db/schema.go`
- `internal/db/run.go`
- `internal/db/run_test.go`
- `internal/ipc/protocol.go`
- `internal/ipc/protocol_test.go`
- `internal/git/git.go`
- `internal/git/git_branch_worktree_test.go`
- `internal/pipeline/executor.go`
- `internal/tui/view.go`
- `internal/tui/pipeline.go`
- `internal/tui/yolo_test.go`
- `internal/types/types.go`
- `internal/paths/paths.go`
- `skills/no-mistakes/SKILL.md`
- `docs/src/content/docs/reference/cli.md`
- `docs/src/content/docs/guides/agents.md`
- `docs/src/content/docs/concepts/pipeline.md`

## Implementation Planning Notes

Future planning should separate these concerns:

1. Worktree-mode model and persistence.
2. Direct current-mode run start through IPC/daemon.
3. CLI flag surfaces and intent behavior.
4. Base SHA and full branch diff resolution.
5. Active-run compatibility.
6. Rendering and user warnings.
7. Cleanup and recovery boundaries.
8. Docs and generated skill guidance.
9. Regression coverage for existing isolated behavior.

## Non-Goals

- Do not change `git push no-mistakes` behavior.
- Do not bypass review, test, document, lint, push, PR, or CI.
- Do not allow dirty current worktrees.
- Do not add a new approval or permission mode.
- Do not make `--yolo` more powerful than `--yes`.
- Do not auto-revert current-mode fix commits after failure.
- Do not add typo aliases such as `no-misstakes`.
