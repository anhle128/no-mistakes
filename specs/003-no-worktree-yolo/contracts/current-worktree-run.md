# Contract: Current Worktree Run Start

## CLI Surface

### Root command

```text
no-mistakes [--no-worktree] [--yes|-y] [--yolo] [--skip=<steps>]
```

Rules:

- `--no-worktree` selects current-worktree mode.
- Absence of `--no-worktree` preserves existing isolated behavior.
- `--yolo` is an alias for `--yes`.
- Passing both `--yes` and `--yolo` is valid and means auto-resolution enabled.
- In current mode, the root command may use existing intent inference.
- In non-interactive or `--yolo` current mode, missing usable inferred intent
  fails before run creation with recovery guidance that does not echo transcript
  snippets.

### AXI command

```text
no-mistakes axi run --intent "<goal>" [--no-worktree] [--yes|-y] [--yolo] [--skip=<steps>]
```

Rules:

- `--intent` remains required when AXI starts a new run.
- AXI may drive a compatible active run without restating intent.
- Current mode starts directly through daemon IPC and must not push to the
  `no-mistakes` gate remote to trigger the run.

## IPC Method

Add a direct start method for current mode. The exact Go constant name may be
chosen during implementation, but the wire method should be stable and tested.

```json
{
  "method": "start_run",
  "params": {
    "repo_id": "repo_123",
    "branch": "feature/current-worktree",
    "head_sha": "0123456789abcdef...",
    "worktree_mode": "current",
    "work_dir": "/abs/path/to/repo-worktree",
    "review_base_ref": "origin/main",
    "review_base_sha": "abcdef0123456789...",
    "review_base_refresh_attempted": false,
    "trigger": "axi",
    "approval_mode": "auto_yes",
    "skip_steps": ["lint"],
    "intent": "User goal summary",
    "intent_source": "agent",
    "caller_kind": "agent_axi",
    "start_shape_hash": "stable-hash"
  }
}
```

### Required validation

- `worktree_mode` must be `isolated` or `current`.
- Current mode requires canonical absolute `work_dir`.
- Current mode requires proven `review_base_sha`.
- Current mode rejects unsafe preflight before creating a recoverable run.
- Start request must not overwrite compatible active run intent, skip settings,
  approval mode, or review base.

## IPC Result

```json
{
  "run_id": "run_123",
  "resumed": false,
  "worktree_mode": "current",
  "work_dir_label": "no-mistakes/thread-21a29cc0",
  "current_worktree_warning": "This run uses this checkout; automated fixes may create commits here.",
  "run_report_ref": "run_123"
}
```

## Rejection: Preflight

```json
{
  "error": {
    "code": -32602,
    "message": "current worktree is not clean",
    "details": {
      "reason": "dirty_worktree",
      "recovery": "Commit tracked changes and untracked non-ignored files, then retry.",
      "worktree_mode": "current"
    }
  }
}
```

Preflight reason values:

- `repo_not_initialized`
- `detached_head`
- `unborn_head`
- `default_branch`
- `dirty_worktree`
- `untrusted_review_base`
- `missing_intent`

## Rejection: Missing Base

```json
{
  "error": {
    "code": -32602,
    "message": "cannot prove default-branch merge base",
    "details": {
      "reason": "rejected_no_trustworthy_base",
      "default_branch_ref": "origin/main",
      "refresh_attempted": true,
      "refresh_result": "failed",
      "recovery": "Fetch the default branch or fix remote access, then retry."
    }
  }
}
```

## Rejection: Incompatible Active Run

```json
{
  "error": {
    "code": -32602,
    "message": "active run is incompatible with this request",
    "details": {
      "run_id": "run_existing",
      "worktree_mode": "isolated",
      "branch": "feature/current-worktree",
      "short_head": "01234567",
      "work_dir_label": "disposable no-mistakes checkout",
      "status": "running",
      "resume_command": "no-mistakes attach --run run_existing",
      "abort_command": "no-mistakes axi abort --run run_existing"
    }
  }
}
```

The conflict response must not include raw logs, raw intent, diff hunks, code
excerpts, secret-bearing metadata, or full run records.
