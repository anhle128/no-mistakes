# Data Model: Current Worktree YOLO Mode

## Run

Branch-scoped pipeline execution.

### Existing fields

- `id`
- `repo_id`
- `branch`
- `head_sha`
- `base_sha`
- `status`
- `pr_url`
- `error`
- `intent`
- `intent_source`
- `intent_session_id`
- `intent_score`
- `created_at`
- `updated_at`

### New or expanded fields

- `worktree_mode`: `isolated|current`
- `work_dir`: canonical absolute git worktree root for current mode; managed
  no-mistakes worktree path for isolated mode when useful for diagnostics
- `work_dir_label`: safe display label, for example repo basename plus run ID
- `metadata_availability`: `recorded|not_recorded|malformed|stale`
- `evidence_state`: `complete|incomplete|degraded`
- `terminal_reason`: structured reason for failed, cancelled, setup-failed,
  stale-recovered, rejected, or superseded outcomes
- `successor_run_id`: optional replacement run identifier
- `successor_head_sha`: optional replacement head SHA
- `review_base_ref`: default branch ref used for base resolution
- `review_base_sha`: proven merge-base SHA when available
- `review_base_refresh_attempted`: boolean
- `review_base_refresh_result`: `not_attempted|succeeded|failed`
- `review_base_refresh_error`: bounded diagnostic string
- `review_base_rejection_reason`: structured rejection reason when base cannot
  be proven
- `approval_mode`: `manual|auto_yes`
- `start_shape_hash`: stable identity for immutable start fields
- `skip_config_hash`: stable identity for skip settings used at start

### Validation rules

- Missing `worktree_mode` on pre-migration runs defaults to `isolated`.
- Newly created runs must always persist `worktree_mode`.
- `worktree_mode=current` requires a canonical absolute `work_dir`.
- `worktree_mode=current` with missing, empty, relative, non-canonical, or stale
  `work_dir` is degraded/incomplete and cleanup-disabled.
- Readers must not reinterpret malformed current-mode data as isolated.
- Current-mode run creation must persist mode, work directory, head, base
  evidence, and approval/skip start shape before the run can be recovered or
  cleanup-eligible.

## Worktree Mode

Execution-boundary classification.

### Values

- `isolated`: existing default disposable no-mistakes checkout.
- `current`: opt-in current git worktree selected by `--no-worktree`.

### Relationships

- A run has one worktree mode.
- Current-mode runs have one current work directory.
- Isolated runs own one managed worktree during execution.

## Start Request

CLI/AXI-to-daemon request to start or resume a pipeline run.

### Fields

- `repo_id`
- `branch`
- `head_sha`
- `worktree_mode`
- `work_dir`
- `review_base_ref`
- `review_base_sha`
- `review_base_refresh_attempted`
- `trigger`: `root|axi|push|rerun|wizard`
- `approval_mode`: `manual|auto_yes`
- `skip_steps`
- `intent_summary`
- `intent_source`
- `caller_kind`: `human_cli|agent_axi|daemon_hook|unknown`
- `start_shape_hash`

### Validation rules

- `axi` starts require non-empty `intent_summary` when no compatible active run
  exists.
- Root current-mode starts may infer intent, but non-interactive or `--yolo`
  starts fail before run creation if usable intent cannot be inferred.
- Current-mode starts reject detached HEAD, default branch, dirty tracked files,
  untracked non-ignored files, uninitialized repo, and untrusted base.
- Ignored files alone do not block current-mode starts.

## Review Base Evidence

Proof that current mode reviews the full branch diff.

### Fields

- `default_branch`
- `default_branch_ref`
- `merge_base_sha`
- `refresh_attempted`
- `refresh_remote`
- `refresh_refspec`
- `refresh_result`
- `refresh_error`
- `rejection_reason`

### Validation rules

- `merge_base_sha` must be a concrete commit SHA before pipeline execution.
- If local base is missing, exactly one non-interactive default-branch ref
  refresh may be attempted.
- Missing base after refresh rejects the start with
  `rejected_no_trustworthy_base`.

## Active Run Compatibility

Rules for deciding whether a request may drive an existing active run.

### Compatible when all apply

- Same repo.
- Same branch.
- Same head commit.
- Same `worktree_mode`.
- Same canonical `work_dir` for current mode.
- Same review base.
- Same immutable start-shape fields.
- No conflicting approval mode, skip configuration, or intent identity.

### Incompatible response fields

- `run_id`
- `worktree_mode`
- `branch`
- `short_head`
- `work_dir_label`
- `status`
- `resume_command`
- `abort_command`

### Prohibited conflict output

- Raw logs.
- Transcript-derived intent.
- Diff hunks.
- Code excerpts.
- Secret-bearing metadata.
- Full run records.

## Fix Outcome

Durable record of a fix attempt.

### Fields

- `id`
- `run_id`
- `step_result_id`
- `round_id`
- `state`: `proposed|attempted|committed|failed`
- `source_finding_id`
- `decision_type`
- `actor_source`
- `automated`
- `commit_sha`
- `summary`
- `created_at`

### Validation rules

- Current-worktree committed fixes must record commit SHA when a commit is
  created.
- Applied-fix claims in AXI/status/TUI/PR summaries must be derived from these
  records, not from prose summaries.

## Gate Decision Evidence

Persisted record for skipped, deferred, informational, fixed, approved, and
passed outcomes.

### Fields

- `step_name`
- `decision_state`
- `source`
- `reported_count`
- `fixed_count`
- `unresolved_count`
- `skipped_count`
- `approved_as_is_count`
- `unavailable_count`

### Validation rules

- Skipped, deferred, and informational decisions render distinctly from passed,
  fixed, or clean outcomes.
- Cross-surface counts must agree for the same run.

## Rendering Envelope

Structured data used by CLI status, AXI, TUI, generated reports, and PR
summaries.

### Fields

- `worktree_mode`
- `worktree_label`
- `work_dir_label`
- `current_worktree_warning`
- `metadata_availability`
- `evidence_state`
- `terminal_reason`
- `run_report_ref`
- `reported_findings`
- `fixed_findings`
- `unresolved_findings`
- `skipped_findings`
- `fix_commits`

### Validation rules

- Structured renderers fail closed when required current-mode fields are
  missing or malformed.
- Full canonical paths appear only in explicit verbose/debug diagnostics.
- PR-facing summaries use safe labels and artifact/run references.

## State Transitions

```text
requested
  -> rejected_no_trustworthy_base
  -> rejected_preflight
  -> rejected_incompatible_active_run
  -> pending
  -> running
  -> checks_passed
  -> completed
  -> failed
  -> cancelled
  -> stale_recovered
  -> superseded_by_run
```

Current-mode terminal states must preserve incomplete/degraded evidence when
the full gate did not complete.
