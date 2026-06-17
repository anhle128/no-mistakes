# Data Model: No-Worktree YOLO Guard

## Run

Represents a branch-scoped gate execution.

Existing fields remain: `id`, `repo_id`, `branch`, `head_sha`, `base_sha`, `status`, `pr_url`, `error`, intent fields, timestamps, and step results.

New boundary fields:

- `boundary_status`: `safe`, `unsafe`, or `unknown`.
- `boundary_reason`: stable short reason for status and tests.
- `boundary_detail`: display-safe explanation.
- `boundary_worktree_path`: canonical realpath used by the verifier.
- `boundary_git_common_dir`: canonical Git common dir used by the verifier.
- `boundary_verified_at`: unix timestamp for the latest proof.
- `boundary_fingerprint`: stable hash of verifier inputs.
- `boundary_verifier_version`: version string for future verifier changes.

Validation:

- `boundary_status` must be one of `safe`, `unsafe`, `unknown`.
- Persisted `safe` status does not authorize action by itself. Any automatic action must refresh proof first.
- Missing, stale, inconsistent, or unverifiable proof degrades to `unknown`.
- A safe boundary must refer to the expected run worktree for the current run and repo.

Relationships:

- Has many `StepResult`.
- Has many `RunEvent`.
- Has one latest `ExecutionBoundary` projection through the persisted boundary fields.

## ExecutionBoundary

Represents the verifier's current classification of the run's source-changing boundary.

Fields:

- `status`: `safe`, `unsafe`, `unknown`.
- `reason`: stable reason code, such as `verified_run_worktree`, `primary_checkout`, `missing_worktree`, `git_metadata_mismatch`, `symlink_escape`, `stale_proof`, or `unknown`.
- `detail`: display-safe explanation.
- `expected_worktree_path`: canonical expected run worktree path.
- `actual_worktree_path`: canonical current working tree path.
- `git_toplevel`: canonical `git rev-parse --show-toplevel`.
- `git_common_dir`: canonical `git rev-parse --git-common-dir`.
- `gate_repo_path`: canonical gate bare repo path.
- `primary_checkout_path`: canonical user checkout path from the repo record.
- `verified_at`: unix timestamp.
- `fingerprint`: stable hash of the fields that prove this classification.
- `verifier_version`: version string.

Validation:

- `safe` requires exact current path, Git toplevel, Git common-dir, and gate worktree metadata agreement.
- `unsafe` requires proof that the source boundary is a primary or non-disposable checkout.
- `unknown` is the default for any incomplete, stale, or contradictory proof.
- Symlink, nested-worktree, and metadata-inconsistent cases cannot be safe.

State transitions:

- `unknown -> safe`: verifier obtains fresh matching proof.
- `safe -> unknown`: proof becomes stale, path moves, metadata becomes unavailable, or the daemon restarts without fresh verification.
- `safe -> unsafe`: verifier proves the current boundary is a primary/non-disposable checkout.
- `unsafe -> safe`: only after a fresh verifier pass proves the expected disposable run worktree.

## YOLOConsent

Represents a request to resolve gates without fresh per-gate human input.

Fields:

- `mode`: `yolo`, `yes`, or `agent-unattended`.
- `requested_by`: actor label where available.
- `surface`: `tui`, `axi`, `headless`, or `agent-skill`.
- `requested_at`: unix timestamp.
- `run_id`: associated run.

Validation:

- Consent does not override boundary status.
- Consent may be honored only when the refreshed boundary is `safe`.
- Existing consent pauses while a run is `unknown` and resumes only after fresh safe proof is restored for the same run.

## GateDecision

Represents an approve, fix, skip, or cancel decision at a gate.

Fields:

- `run_id`.
- `step_name` or `gate_id`.
- `gate_fingerprint`.
- `gate_status`: for example `awaiting_approval`, `fix_review`, or `remote_advance`.
- `action`: `approve`, `fix`, `skip`, or `cancel`.
- `decision_source`: `manual` or `unattended`.
- `actor_type`: `human`, `agent`, or `system`.
- `approval_surface`: `tui`, `axi`, `headless`, `agent-skill`, or `daemon`.
- `consent_mode`: `none`, `manual`, `yolo`, or `yes`.
- `finding_ids`: selected findings when fixing.
- `boundary_status_at_decision`.
- `created_at`.

Validation:

- Unattended decisions require refreshed `safe` boundary proof.
- Manual decisions are allowed on unsafe/unknown runs only as explicit responses to the current pending gate.
- Manual source-changing fix decisions must be auditable and distinct from unattended decisions.
- Duplicate unattended responses for the same run, gate, fingerprint, source, and action class are idempotent.

## RunEvent

Auditable event for gate automation and boundary decisions.

Fields:

- `id`.
- `run_id`.
- `event_type`: `boundary_refreshed`, `yolo_allowed`, `yolo_withheld`, `yolo_not_requested`, `manual_decision`, `remote_write_allowed`, `remote_write_withheld`.
- `step_name` or `gate_id`.
- `gate_fingerprint`.
- `requested_mode`: `none`, `yolo`, `yes`, or `agent-unattended`.
- `decision_source`.
- `action`.
- `boundary_status`.
- `boundary_reason`.
- `message`.
- `actor_type`.
- `approval_surface`.
- `created_at`.

Validation:

- Every gate observed for automation records one of allowed, withheld, or not-requested status.
- Events must be append-only.
- Event messages are display-safe and must not include raw transcript text or secrets.

## Finding

Existing review/test/document/lint/CI item used at gates.

Relevant fields:

- `id`.
- `severity`.
- `description`.
- `action`: `auto-fix`, `ask-user`, or `no-op`.
- Optional file, line, context, suggested fix, and user instructions.

Validation:

- Existing action semantics remain unchanged.
- Under safe unattended consent, actionable findings may be selected for one fix round.
- Under unsafe/unknown boundary, actionable findings remain available for explicit manual response only.

## OriginReference

Companion planning artifact that preserves feature origin and source anchors.

Fields:

- Original request.
- Inferred purpose.
- Source-scout findings.
- Scope decision.
- Ambiguity resolution.
- First implementation files to inspect.
- Non-goals.

Validation:

- Lives in `specs/002-no-worktree-yolo/no-worktree-yolo.md`.
- Main stakeholder spec remains implementation-neutral; source anchors stay in the companion reference.
