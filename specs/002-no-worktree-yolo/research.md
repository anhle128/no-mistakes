# Research: No-Worktree YOLO Guard

## Decision: Verify execution boundary from controller-owned Git and path state

The verifier will classify a run as `safe`, `unsafe`, or `unknown` using only controller-owned inputs: run ID, repo ID, expected worktree path from `paths.WorktreeDir(repo.ID, run.ID)`, gate bare repo path from `paths.RepoDir(repo.ID)`, repo working path from the persisted repo record, canonical realpaths, and trusted Git metadata from commands such as `rev-parse --show-toplevel`, `rev-parse --git-common-dir`, and `git worktree list --porcelain` against the gate repo.

Safe proof requires all of the following:

- The current step work directory resolves to the expected run worktree path.
- The Git toplevel resolves to that same expected worktree path.
- The Git common dir resolves under the gate bare repository for the repo ID.
- The expected worktree is registered in the gate repo worktree metadata.
- The worktree path is not the user's primary checkout and is not nested inside it.
- Canonical realpath checks reject symlink escape, missing path, nested boundary ambiguity, and metadata mismatch.

Unsafe proof is used when the verifier can prove the current source boundary is the user's primary checkout or another non-disposable source checkout. Unknown is used when proof is stale, missing, inconsistent, unavailable on the host, or cannot distinguish the source boundary safely.

**Rationale**: The spec requires independent controller proof and forbids trusting user/agent path claims. The daemon already owns run creation and knows the intended gate/worktree locations in `internal/daemon/manager.go`, while `internal/git/git.go` already contains Git root/common-dir helpers that can be reused or extended.

**Alternatives considered**:

- Trust `workDir` passed into the executor: rejected because it is a process-local claim and cannot detect moved/symlinked/metadata-inconsistent workspaces after restart or reattach.
- Trust agent prompt steering: rejected because `internal/agent/steering.go` explicitly says it is soft steering, not enforcement.
- Require an OS sandbox: rejected as out of scope and unnecessary for this product contract; the requirement is proof of disposable source workspace, not general filesystem confinement.

## Decision: Persist boundary classification on Run and gate automation events separately

Add additive run columns for latest boundary state:

- `boundary_status`: `safe`, `unsafe`, or `unknown`.
- `boundary_reason`: short stable reason code or label for display.
- `boundary_detail`: human-readable explanation safe to show in TUI/AXI.
- `boundary_worktree_path`: canonical worktree path used as proof.
- `boundary_git_common_dir`: canonical Git common dir used as proof.
- `boundary_verified_at`: unix timestamp for freshness checks.
- `boundary_fingerprint`: stable hash of verifier inputs.
- `boundary_verifier_version`: verifier contract version.

Add a `run_events` table for allowed, withheld, and not-requested gate automation states. Events record run ID, step, gate ID/fingerprint, decision source, consent mode, action, boundary status/reason, actor type, approval surface, and created time.

**Rationale**: FR-002 requires the Run to carry the authoritative current classification, while FR-009 requires auditable allowed/withheld/not-requested status per gate. Current `runs` rows have no boundary fields, and `step_rounds` cannot represent not-requested or withheld responses cleanly.

**Alternatives considered**:

- Store all data inside `step_rounds`: rejected because withheld/not-requested states may occur before a fix round and must also cover remote/provider write gates.
- Store only in logs: rejected because TUI, AXI, status, and agents need structured status after restart/reattach.

## Decision: Enforce in the daemon/pipeline, not only in TUI or AXI

The daemon and executor will be the authoritative guard:

- `RunManager.startRun` classifies the initial boundary after creating the run worktree.
- Before every unattended response, daemon response handling refreshes boundary proof and rejects/records withheld automation if status is not `safe`.
- Executor auto-fix loops check the boundary before automatic source-changing fixes.
- Push, PR create/update, CI auto-fix push, and provider review-advancing writes check boundary immediately before the concrete write.
- TUI `maybeAutoApproveCmd` and AXI `driveRun` may render cached withheld feedback immediately, but they still send unattended intent to the daemon so proof refresh, allowed/withheld audit, and executor forwarding decisions stay daemon-owned.

**Rationale**: Existing automatic paths are split across `internal/tui/commands.go`, `internal/cli/axi_drive.go`, `internal/pipeline/executor.go`, `internal/pipeline/steps/push.go`, `internal/pipeline/steps/pr.go`, and `internal/pipeline/steps/ci_fix.go`. UI-only enforcement would miss headless, daemon, restart, and provider-write paths.

**Alternatives considered**:

- Disable the `y` key and `--yes` only: rejected because pipeline auto-fix and provider writes can still advance unattended after a gate.
- Make every provider adapter responsible for policy: rejected because it duplicates the same boundary rule and risks inconsistent behavior.

## Decision: Extend response metadata to distinguish manual and unattended decisions

Extend IPC `RespondParams` with decision metadata:

- `decision_source`: `manual` or `unattended`.
- `actor_type`: `human`, `agent`, or `system`.
- `approval_surface`: `tui`, `axi`, `headless`, `agent-skill`, or `daemon`.
- `consent_mode`: `none`, `manual`, `yolo`, or `yes`.
- `gate_id` and `gate_fingerprint` for duplicate prevention.

Legacy callers that do not set these fields default to `manual` with an unknown surface. TUI YOLO and AXI `--yes` set `unattended`. Explicit `axi respond --action ...` remains manual unless `--yes` is used to continue subsequent gates unattended.

**Rationale**: FR-005 requires manual source-changing actions to be distinguishable from unattended automatic decisions, including actor and surface. Current IPC response parameters do not carry that information.

**Alternatives considered**:

- Infer manual/unattended from command names or TUI state: rejected because after IPC serialization the daemon needs an explicit contract, and inference breaks for agents and future surfaces.

## Decision: Persist stable gate identity and use it for duplicate automatic-response prevention

Gate identity is computed from immutable current decision inputs:

- Run ID.
- Step name or gate ID.
- Current step status (`awaiting_approval`, `fix_review`, remote-advance gate status).
- Current round number or gate version.
- Decision fingerprint from current findings JSON, selected diff/head SHA when applicable, and remote/provider write target.

User-facing status text, generated guidance, and mutable agent metadata are excluded. Add a unique constraint or idempotent insert path for unattended gate decisions keyed by run ID, gate ID, gate fingerprint, source, and action class.

**Rationale**: The spec requires duplicate prevention across reattach, restart, and repeated status events. Current TUI suppression is in memory, and AXI suppression waits for status changes, so neither survives every restart/reattach case.

**Alternatives considered**:

- Keep existing in-memory maps only: rejected because daemon restart and TUI reattach lose state.
- Use step name only: rejected because a step can have multiple rounds/gates and fix-review state.

## Decision: Withheld output is structured status plus concise recovery text

All gate-driving or gate-observing surfaces render the same structured state:

- Requested mode: `yolo`, `--yes`, or agent unattended consent.
- Current gate: step/gate ID and status.
- Boundary status and reason.
- Automation status: `allowed`, `withheld`, or `not_requested`.
- Recovery options: continue manually for this gate, restart through the normal no-mistakes gate to create a disposable worktree, or inspect setup/status.

Safe runs do not show extra warnings. Unsafe/unknown runs do not enqueue a response and leave the current gate available for manual action.

**Rationale**: FR-008 requires clear explanation on TUI, AXI, headless CLI output, terminal status, and generated agent guidance. Rendering from a shared IPC contract keeps labels consistent.

**Alternatives considered**:

- Log-only warnings: rejected because the user must not need logs to recover.
- Warnings only when toggling YOLO: rejected because status/reattach/headless observers also need to report current automation state.

## Decision: Validate SC-003 through snapshot and reviewer-visible copy checks

SC-003's "95% of users or agent operators" target will be approximated for implementation with:

- Snapshot/unit tests for withheld copy in TUI rendering, AXI TOON output, terminal/status output, and generated agent guidance.
- Contract checks that each output includes requested mode, current gate, boundary status, reason, and at least one recovery option.
- Manual reviewer-visible evidence in the PR or test output for the generated docs/skill copy.

**Rationale**: A formal user study is too heavy for this repository's normal validation loop, but the implementation can make the comprehension criteria mechanically testable.

**Alternatives considered**:

- Ignore SC-003 until later: rejected because user-facing explanation is a required part of fail-closed behavior.

## Decision: No new dependency

Use standard library path handling plus existing Git helpers and SQLite migrations. No package or SDK adoption is needed.

**Rationale**: The constitution requires new dependencies only when they reduce risk or complexity. Existing code already wraps Git, path, IPC, DB, and provider surfaces.

**Alternatives considered**:

- Add a Git library: rejected because current code already shells out to Git consistently and tests use Git command fixtures.
