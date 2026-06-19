# Phase 0 Research: Current Worktree YOLO Mode

## Decision: Represent execution boundary as `worktree_mode`

Use a shared enum with persisted values `isolated` and `current`. Keep these as
machine values only; user-facing surfaces should render plain labels such as
"disposable no-mistakes checkout" and "uses this checkout".

**Rationale**: The DB, IPC, AXI, TUI, status, PR summary, and recovery code all
need the same source of truth. Separating machine values from labels avoids
ambiguous UI text such as "current".

**Alternatives considered**:

- Infer mode from the work directory path. Rejected because invalid or missing
  paths would be normalized into unsafe cleanup decisions.
- Use a boolean `no_worktree`. Rejected because `isolated|current` is clearer in
  persisted state, contracts, and future migrations.

## Decision: Add a direct IPC start path for current mode

Current-worktree mode should start through a daemon IPC request that carries the
resolved repo, branch, head, base evidence, mode, work directory, trigger,
approval mode, skip settings, and intent metadata.

**Rationale**: Existing AXI start pushes to the `no-mistakes` gate remote and the
daemon then creates a disposable worktree. The feature requires no extra
worktree and no dependency on `git push no-mistakes`.

**Alternatives considered**:

- Encode current mode as a git push option. Rejected because it still routes
  through gate push semantics and does not naturally carry the current work
  directory identity.
- Use rerun for current starts. Rejected because rerun uses the latest gate head
  and does not prove the caller's current checkout boundary.

## Decision: Resolve and persist the canonical current git worktree root

Resolve the execution directory with `git rev-parse --show-toplevel` behavior
through existing git helpers, normalize symlinks, require an absolute canonical
path, and persist it before the run is recoverable or cleanup-eligible.

**Rationale**: Current-mode fixes mutate a concrete directory. Resume
compatibility, warnings, cleanup exclusion, and report reconstruction all depend
on that path being stable and validated.

**Alternatives considered**:

- Use the shell working directory. Rejected because subdirectory invocation must
  execute at the git worktree root.
- Use the registered main repo path. Rejected because linked worktrees must run
  in the current linked checkout, not the main checkout.

## Decision: Prove the default-branch merge base before current-mode execution

Current mode must calculate the full branch review base before the pipeline
starts. If the merge base cannot be proven locally, attempt one non-interactive
default-branch ref refresh, persist the attempt/result, and reject with
`rejected_no_trustworthy_base` if still unproven.

**Rationale**: Current mode changes the execution boundary, so it cannot fall
back to a narrow or empty-tree diff without weakening the meaning of a passed
gate.

**Alternatives considered**:

- Reuse `resolveBranchBaseSHA` fallback behavior. Rejected for current mode
  because it can silently degrade review scope.
- Continue with a degraded review. Rejected by the clarified spec.

## Decision: Make cleanup data-driven and fail closed

Only remove managed worktrees for runs with validated `worktree_mode=isolated`.
For `worktree_mode=current`, malformed current metadata, missing new-format
metadata, setup failure, panic, cancellation, stale recovery, and normal
completion must not remove the recorded current work directory.

**Rationale**: The riskiest failure is deleting a user/tool-owned checkout. Data
validation must decide cleanup eligibility before path operations run.

**Alternatives considered**:

- Skip cleanup only when the path is outside `NM_HOME/worktrees`. Rejected
  because path inference is weaker than explicit mode metadata and fails under
  malformed data.

## Decision: Preserve active-run compatibility instead of cancelling on conflict

For the same repo and branch, current-mode requests may only drive a compatible
active run. Compatibility includes repo, branch, head commit, worktree mode,
current-mode work directory, review base, approval mode, skip configuration,
and immutable start-shape fields. Incompatible requests are rejected with a
whitelisted conflict message.

**Rationale**: Existing daemon behavior cancels active same-branch runs before
starting a new one. That is unsafe when one run may mutate the current checkout.

**Alternatives considered**:

- Auto-abort the old run. Rejected because it can hide partial evidence and
  mutate another actor's active workflow.
- Start a second run. Rejected because branch/run evidence would race.

## Decision: Add structured rendering fields and path minimization

Expose `worktree_mode`, `work_dir_label`, `current_worktree_warning`, metadata
state, degraded/incomplete evidence state, findings/fix counts, and run/report
references in structured surfaces. Full absolute paths may appear only in
explicit verbose/debug diagnostics.

**Rationale**: AXI and generated summaries are consumed by agents and may be
persisted. They need stable fields without repeatedly leaking local usernames,
customer names, or temporary workspace paths.

**Alternatives considered**:

- Add prose warnings only. Rejected because downstream agents cannot reliably
  parse prose and the constitution requires structured contracts.

## Decision: Persist fix outcome and provenance records

Record current-worktree fix states as proposed, attempted, committed, or failed.
Include actor/source, source finding or decision, decision type, commit SHA when
created, and whether the change was automated or user-authored.

**Rationale**: In current mode, fix commits stay in the user's checkout after
success or failure. Reports must not rely on agent prose summaries to claim what
was applied.

**Alternatives considered**:

- Use existing step round `fix_summary` only. Rejected because it lacks commit
  identity and provenance.

## Decision: Treat `--yolo` as an alias for `--yes`

Register `--yolo` on root and AXI run commands and combine it with `--yes` by
logical OR. Do not create a separate approval mode.

**Rationale**: The spec requires exact equivalence with existing `--yes`
auto-resolution behavior and acceptance of both flags together.

**Alternatives considered**:

- Add a new "yolo" permission level. Rejected because it would weaken the
  approval contract and exceed the feature scope.
