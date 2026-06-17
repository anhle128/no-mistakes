# Quickstart: No-Worktree YOLO Guard

## Starting Context

Read these first:

1. [spec.md](spec.md)
2. [no-worktree-yolo.md](no-worktree-yolo.md)
3. [research.md](research.md)
4. [data-model.md](data-model.md)
5. [contracts/yolo-boundary.md](contracts/yolo-boundary.md)

## Implementation Sequence

1. Add the boundary verifier.
   - Create a narrow verifier around canonical paths and trusted Git metadata.
   - Cover safe disposable worktree, primary checkout, missing metadata, symlink escape, nested worktree, stale proof, and Git metadata mismatch.

2. Add persistence.
   - Add additive run boundary columns.
   - Add run/gate event persistence for boundary refreshes and allowed/withheld/not-requested automation states.
   - Add DB migration and round-trip tests.

3. Extend IPC and render models.
   - Add boundary and gate automation status to `RunInfo` and events.
   - Add decision metadata to `RespondParams`.
   - Default legacy responses to manual/unknown-surface for compatibility.

4. Enforce the guard.
   - Classify at run creation.
   - Refresh proof before every unattended response.
   - Gate executor auto-fix loops on fresh safe proof.
   - Guard push, PR create/update, CI auto-fix push, and future provider write paths.
   - Record allowed, withheld, and not-requested events.

5. Update user surfaces.
   - TUI: show withheld YOLO reason and recovery while leaving manual actions available.
   - AXI/headless: include structured `automation` output and treat daemon-withheld unattended intent as a gate state, not as a client failure.
   - Terminal/status output: expose current boundary/gate automation status.
   - Generated agent skill: require explicit per-gate user decision before an agent submits manual responses on unsafe/unknown runs.

6. Update docs and generated artifacts.
   - Update YOLO/auto-fix docs and troubleshooting.
   - Update `skills/no-mistakes/SKILL.md` from its source of truth if applicable.

## Focused Verification

Run targeted tests while implementing:

```bash
go test ./internal/db ./internal/ipc ./internal/daemon ./internal/pipeline ./internal/pipeline/steps ./internal/tui ./internal/cli
```

Boundary verifier and Git/worktree tests:

```bash
go test ./internal/git ./internal/boundary ./internal/gate
```

Tagged e2e smoke tests for cross-process behavior:

```bash
go test -tags=e2e -count=1 -timeout 300s ./internal/e2e/...
```

Full required validation for ordinary Go changes:

```bash
gofmt -w <changed-go-files>
go test -race ./...
make lint
```

If docs or generated skill guidance changes:

```bash
make docs-build
make skill-check
```

## Manual Smoke Scenarios

Safe isolated run:

1. Trigger a normal `git push no-mistakes` run that creates a disposable worktree.
2. Enable YOLO in the TUI or run AXI with `--yes`.
3. Confirm actionable findings get one fix round, fix-review is approved, no-op-only gates are approved, and no extra warnings appear.

Unsafe or unknown run:

1. Simulate a run whose work directory is the primary checkout, missing, symlink-escaped, or metadata-inconsistent.
2. Enable YOLO or run AXI with `--yes`.
3. Confirm unattended intent is recorded as withheld, no response is forwarded to the executor, no source or provider write occurs, the gate remains awaiting manual action, and output names the requested mode, gate, boundary status, reason, and recovery option.

Reattach/restart:

1. Reach an awaiting gate with unattended consent active.
2. Reattach the TUI or restart the daemon.
3. Confirm the same gate identity prevents duplicate automatic responses and stale/missing proof degrades to `unknown` until refreshed.

Provider/remote write:

1. Reach push, PR, or CI auto-fix write paths.
2. Confirm unsafe/unknown boundaries withhold writes and read-only provider queries can still report status.

## Reviewer-Visible Evidence

Recorded on 2026-06-17:

- Focused package validation: `go test ./internal/db ./internal/ipc ./internal/daemon ./internal/pipeline ./internal/pipeline/steps ./internal/tui ./internal/cli ./internal/git ./internal/boundary ./internal/gate ./internal/skill`
- Boundary/guidance additions: `go test ./internal/boundary ./internal/skill`
- Cross-process withheld automation: `go test -tags=e2e -run TestYoloNoWorktree -count=1 -timeout 180s ./internal/e2e`
- Full tagged journey suite: `go test -tags=e2e -count=1 -timeout 300s ./internal/e2e/...`
- Full race validation: `go test -race ./...`
- Lint validation: `make lint`
- Docs and generated skill checks: `make docs-build` and `make skill-check`

The focused e2e removes the daemon-managed run worktree while a review gate is awaiting approval, then runs `axi run --yes`. The observed AXI output includes `automation.status: withheld`, `requested_mode: yes`, `reason: unknown`, `gate: review`, and recovery options to respond manually or restart validation through no-mistakes. The run remains at the review approval gate and the upstream feature branch is not pushed.

Safe-run unchanged behavior is covered by the existing AXI e2e `TestAxiAgentJourney`, which drives `axi run --yes` through an isolated run to `outcome: passed`, and by the full tagged e2e suite above.
