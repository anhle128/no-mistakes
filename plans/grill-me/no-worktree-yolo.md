# No-Worktree Yolo Plan

Status: grill complete

## Target Result

Add a current-working-tree execution mode for users who are already inside a
git working tree created by another tool, such as an Archon-created worktree.

The user-facing commands are:

```sh
no-mistakes --no-worktree --yolo
no-mistakes axi run --intent "..." --no-worktree --yolo
```

`--no-worktree` means no-mistakes must not create an additional
no-mistakes-owned worktree under `~/.no-mistakes/worktrees`. Instead, the
pipeline runs in the root of the current git working tree. The current working
tree can be an Archon-created worktree or a normal checkout.

`--yolo` is only an alias for existing `--yes` behavior. It does not grant any
new approval behavior beyond auto-resolving gates the way `--yes` already does.

## Decisions

- Keep the default mode unchanged: without `--no-worktree`, no-mistakes still
  creates an isolated no-mistakes-owned worktree.
- Add `--no-worktree` to bare `no-mistakes` and `no-mistakes axi run`.
- Add `--yolo` as an alias of `--yes` on both surfaces.
- In current mode, resolve `work_dir` to `git rev-parse --show-toplevel` from
  the process current directory, not the shell subdirectory.
- Preserve existing preflight rules: repo must be initialized, branch must not
  be detached, branch must not be the default branch, and the working tree must
  be clean and committed before starting.
- Bare `no-mistakes --no-worktree --yolo` does not require `--intent`; it uses
  existing intent inference behavior when no explicit intent is available.
- `no-mistakes axi run --no-worktree --yolo` still requires `--intent` when it
  starts a new run.
- Current mode starts directly through CLI/daemon IPC. It does not rely on
  `git push no-mistakes` to trigger the post-receive hook.
- Current mode reviews the full current branch diff against the default branch
  base, not only the last commit.
- Current mode still runs the normal pipeline: review, test, document, lint,
  push, PR, and CI.
- If current-mode fixes are committed and the pipeline later fails, leave those
  commits in the user/tool-owned working tree. Do not auto-revert.
- Store run metadata as `worktree_mode` and `work_dir`.
- Internal mode names are `isolated` for current default behavior and `current`
  for `--no-worktree`.
- Never remove or cleanup `work_dir` for `worktree_mode=current`.
- Recovery may mark stale current-mode runs failed, but must not delete the
  current working tree.
- Support only correctly spelled `no-mistakes`; do not ship a typo alias for
  `no-misstakes`.

## Source Facts

- The root command currently has `--yes` and `--skip`; it calls `attachRun`
  without any worktree-mode parameter (`internal/cli/root.go:53`,
  `internal/cli/root.go:77`, `internal/cli/root.go:82`).
- `attachRun` is the shared root/attach path and currently routes no-active-run
  cases through the wizard (`internal/cli/attach.go:25`,
  `internal/cli/attach.go:91`).
- `axi run` currently has `--yes`, `--skip`, and `--intent`, and calls
  `runAxiRun(cmd, autoYes, skipSteps, intent)` (`internal/cli/axi_drive.go:57`,
  `internal/cli/axi_drive.go:90`, `internal/cli/axi_drive.go:96`).
- `axi run` requires `--intent` before starting a new run
  (`internal/cli/axi_drive.go:123`).
- The existing preflight rejects default-branch and dirty working-tree starts
  (`internal/cli/axi_drive.go:173`, `internal/cli/axi_drive.go:179`,
  `internal/cli/axi_drive.go:185`, `internal/cli/axi_drive.go:192`).
- `axi run` currently starts a fresh run by pushing to the `no-mistakes` gate
  remote (`internal/cli/axi_drive.go:206`, `internal/cli/axi_drive.go:211`).
- `findRepo` already handles being invoked inside a git worktree by falling
  back to the main repo record when needed (`internal/cli/root.go:104`,
  `internal/cli/root.go:120`).
- `git.FindGitRoot` returns the current working tree root
  (`internal/git/git.go:78`), while `git.FindMainRepoRoot` resolves a worktree
  back to the main repository root (`internal/git/git.go:99`).
- The daemon `startRun` currently always creates a no-mistakes-owned worktree
  from the bare gate repo (`internal/daemon/manager.go:217`,
  `internal/daemon/manager.go:270`, `internal/daemon/manager.go:272`,
  `internal/daemon/manager.go:273`).
- Repo config is currently loaded from that created worktree
  (`internal/daemon/manager.go:300`, `internal/daemon/manager.go:306`).
- Worktree cleanup is currently unconditional in setup failure and goroutine
  cleanup paths (`internal/daemon/manager.go:291`,
  `internal/daemon/manager.go:399`).
- The executor already takes an explicit `workDir`, so the pipeline execution
  boundary can be moved without changing every step (`internal/pipeline/executor.go:111`,
  `internal/pipeline/executor.go:215`).
- The `runs` schema currently has no worktree-mode/work-dir metadata
  (`internal/db/schema.go:12`), and migrations are additive statements
  (`internal/db/schema.go:63`).
- IPC `RunInfo` currently exposes branch/head/base/status but no worktree mode
  or path (`internal/ipc/protocol.go:180`).
- AXI rendering has a `run:` object and active gate rendering where run metadata
  can be exposed (`internal/cli/axi_render.go:223`,
  `internal/cli/axi_render.go:244`).
- The no-mistakes worktree directory is owned by `paths.WorktreeDir`
  (`internal/paths/paths.go:46`).

## Implementation Plan

### 1. Add Worktree Mode Types

Create a shared type, likely under `internal/types`:

```go
type WorktreeMode string

const (
    WorktreeModeIsolated WorktreeMode = "isolated"
    WorktreeModeCurrent  WorktreeMode = "current"
)
```

Use these names in DB, IPC, telemetry, and daemon options. Avoid a core
`NoWorktree bool` because current mode still uses a git working tree; it only
skips creating an additional no-mistakes-owned worktree.

### 2. Persist Run Metadata

Add columns to `runs`:

```sql
worktree_mode TEXT NOT NULL DEFAULT 'isolated'
work_dir TEXT
```

Update:

- `internal/db/schema.go`
- `internal/db/run.go`
- `internal/db/run_test.go`

`InsertRun` should either accept a run options struct or gain a new helper so
new current-mode runs persist:

```text
worktree_mode = current
work_dir = resolved current git working tree root
```

Existing rows migrate to `isolated`; their `work_dir` can be empty.

### 3. Extend IPC

Add a direct start method, for example:

```go
MethodStartRun = "start_run"
```

Params:

```go
type StartRunParams struct {
    RepoID       string
    Branch       string
    HeadSHA      string
    BaseSHA      string
    WorktreeMode types.WorktreeMode
    WorkDir      string
    SkipSteps    []types.StepName
    Intent       string
}
```

Result can reuse `RunID`.

Also add `WorktreeMode` and `WorkDir` to `ipc.RunInfo`, so status, AXI, TUI,
and reattach can show whether the run is isolated or current.

Keep `PushReceivedParams` and `RerunParams` defaulting to isolated mode unless
explicitly extended later. Do not change `git push no-mistakes` behavior in
this pass.

### 4. Refactor RunManager Start Options

Replace the long `startRun` signature with an options struct:

```go
type StartRunOptions struct {
    Repo         *db.Repo
    Branch       string
    HeadSHA      string
    BaseSHA      string
    Trigger      string
    SkipSteps    []types.StepName
    Intent       string
    WorktreeMode types.WorktreeMode
    WorkDir      string
}
```

Behavior:

- `isolated`: preserve current flow: create `wtDir := paths.WorktreeDir(...)`,
  `git.WorktreeAdd`, copy local identity, load repo config from `wtDir`, run
  executor in `wtDir`, and cleanup with `git.WorktreeRemove`.
- `current`: do not call `git.WorktreeAdd`, do not call `git.WorktreeRemove`,
  use the resolved current git root as `workDir`, copy no identity, load repo
  config from `workDir`, run executor in `workDir`, and leave the directory
  untouched on success/failure/panic.

Telemetry should include `worktree_mode`. Failure telemetry should distinguish
`create_worktree` from current-mode validation/config failures.

### 5. Add CLI Flags

Root command:

```sh
no-mistakes --no-worktree --yolo
```

AXI command:

```sh
no-mistakes axi run --intent "..." --no-worktree --yolo
```

Flag mapping:

- `--yolo` sets the same boolean as `--yes`.
- If both `--yes` and `--yolo` are passed, treat it as true, not an error.
- `--no-worktree` selects `WorktreeModeCurrent`.

Root behavior:

- When `--no-worktree` is present, do not route through the wizard.
- Resolve repo, current branch, current head, clean preflight, base SHA, then
  call the new start-run IPC method.
- Attach to TUI after the run is registered, preserving root command behavior.
- Do not require `--intent`.

AXI behavior:

- Preserve `--intent` requirement for starting a new run.
- If `--no-worktree` is present and no compatible active run exists, call the
  new start-run IPC method instead of `triggerRun`/gate push.
- Drive the run with existing `driveRun`.

### 6. Resolve Current Workdir And Base SHA

For current mode:

- `workDir = git.FindGitRoot(".")`, so running from a subdirectory uses the
  root of the current Archon-created worktree.
- `repo` lookup may still need `findRepo`, because installed repo records may
  be keyed to the main checkout path for attached git worktrees.
- `headSHA = git rev-parse HEAD` in `workDir`.
- `baseSHA` should represent the full branch diff against default branch:
  fetch or inspect `origin/<defaultBranch>`, then use merge base with `HEAD`.

Preferred base algorithm:

```text
fetch origin <defaultBranch> best effort
baseSHA = git merge-base HEAD origin/<defaultBranch>
fallback = existing base resolution behavior if remote/default is unavailable
```

Do not use only the previous gate old SHA for current mode; there may be no
gate push.

### 7. Active Run Compatibility

When `--no-worktree` is requested:

- Active current-mode run for same repo+branch+head: resume/drive it.
- Active isolated-mode run for same repo+branch: fail with a clear error asking
  the user to finish or abort the existing run first.
- No active run: start a current-mode run.

When isolated/default mode is requested:

- Existing behavior should remain unchanged.
- If a current-mode run is active on the same branch, fail rather than mixing
  modes.

This requires active-run lookup to compare `WorktreeMode` and probably `HeadSHA`
the way `activeRunInfoForHead` does today.

### 8. Render Mode Clearly

AXI:

- Add `worktree_mode` and `work_dir` to the `run:` object.
- For current mode, include a short help/warning line:

```text
This run is using the current working tree; pipeline fixes may modify this checkout.
```

TUI:

- Show a compact label such as `current worktree` or `isolated worktree`.
- For current mode, show the resolved `work_dir` somewhere visible but not noisy.

Docs/skill:

- Update `skills/no-mistakes/SKILL.md` so agents know to pass
  `--no-worktree --yolo` when the user asks for that mode.
- Update CLI docs with both command forms and explain that `--yolo` is `--yes`.

### 9. Preserve Cleanup Boundaries

In current mode:

- Do not remove `workDir` during setup failure.
- Do not remove `workDir` after the goroutine finishes.
- Do not remove `workDir` after panic.
- Do not remove `workDir` during daemon recovery.

In isolated mode:

- Keep existing cleanup behavior.
- Continue cleanup of orphaned directories under `~/.no-mistakes/worktrees`.

### 10. Failure Behavior

If current mode fails after auto-fixes:

- Leave committed fixes in the current working tree.
- Report that the run failed and that any generated commits remain in the
  current working tree for inspection/amend/revert/rerun.
- Do not auto-revert commits.

## Acceptance Criteria

- `no-mistakes --no-worktree --yolo` starts a pipeline in the current git
  working tree root without creating `~/.no-mistakes/worktrees/<repo>/<run>`.
- `no-mistakes axi run --intent "..." --no-worktree --yolo` starts and drives
  the same current-mode pipeline.
- `--yolo` has the same auto-resolution behavior as `--yes`.
- Existing `--yes` behavior remains unchanged.
- Existing default isolated mode still creates and cleans up a no-mistakes-owned
  worktree.
- Current mode rejects dirty worktrees using the existing clean-committed
  preflight.
- Current mode rejects default-branch and detached-HEAD starts using existing
  preflight behavior.
- Current mode requires `no-mistakes init` to have already registered the repo.
- Current mode computes review scope as full branch diff versus default branch.
- Current mode still runs push, PR, and CI steps.
- Current mode never removes the current working tree directory.
- Active runs cannot mix current and isolated modes on the same branch.
- Run metadata persists `worktree_mode` and `work_dir`.
- AXI and TUI make current-mode runs visibly distinct.
- Existing tests for push-triggered isolated runs continue to pass.

## Test Strategy

Unit tests:

- DB schema/migration inserts old runs as `isolated` and can persist/read
  `current` plus `work_dir`.
- IPC protocol round trips the new start-run params and run metadata.
- CLI flag parsing treats `--yolo` as `--yes` on root and AXI.
- AXI current-mode start requires `--intent`; bare root current-mode start does
  not require intent.
- Preflight tests cover dirty, detached, and default branch through existing
  logic with current-mode callers.
- Active-run selection rejects mode mismatches.
- Base SHA resolver covers normal merge-base, missing remote/default fallback,
  and invocation from a subdirectory.

Daemon/manager tests:

- Isolated start still calls the worktree creation path and cleans up.
- Current-mode start does not create a no-mistakes worktree.
- Current-mode executor receives the current git root as `WorkDir`.
- Current-mode config loads from the current git root.
- Current-mode cleanup paths do not call `git.WorktreeRemove` for `workDir`.
- Current-mode failure leaves the directory intact.

Render tests:

- AXI `run:` output includes `worktree_mode` and `work_dir`.
- Current-mode AXI output includes the warning/help text.
- TUI model/view exposes a compact current-worktree indicator.

Regression tests:

- `internal/cli/root_test.go` existing `-y` wizard tests still pass.
- `internal/cli/axi_drive_test.go` existing `--yes` drive behavior still passes.
- `internal/daemon/manager_test.go` existing push/rerun isolated tests still
  pass.

Suggested verification commands:

```sh
go test ./internal/db ./internal/ipc ./internal/cli ./internal/daemon ./internal/git
go test ./...
```

## Non-Goals

- Do not change `git push no-mistakes` behavior.
- Do not make current mode bypass review/test/lint/push/PR/CI.
- Do not allow dirty working trees for current mode.
- Do not support typo command names such as `no-misstakes`.
- Do not add a new permission mode beyond `--yes`; `--yolo` is only an alias.
- Do not auto-revert current-mode fix commits after failure.

## Stop Condition

Implementation is complete when both CLI surfaces support the new flags,
current-mode runs execute in the resolved current git working tree root, default
isolated behavior is unchanged, run metadata is visible, and the targeted plus
full Go test suites pass.
