# Contract: Boundary-Aware YOLO and Gate Automation

## Boundary Status Values

All surfaces use the same status values:

- `safe`: refreshed controller proof shows source-changing work is bounded to the disposable run worktree.
- `unsafe`: refreshed controller proof shows the current source boundary is a primary/non-disposable checkout or other disallowed source location.
- `unknown`: proof is missing, stale, contradictory, or unavailable.

`safe` is required for unattended source-changing and remote-advancing automation.

## IPC RunInfo Additions

`RunInfo` adds a boundary object:

```json
{
  "boundary": {
    "status": "safe|unsafe|unknown",
    "reason": "verified_run_worktree|primary_checkout|missing_worktree|git_metadata_mismatch|symlink_escape|stale_proof|unknown",
    "detail": "display-safe explanation",
    "verified_at": 1781683200,
    "verifier_version": "yolo-boundary-v1"
  }
}
```

When a gate is awaiting action, `RunInfo` also exposes gate automation status:

```json
{
  "gate_automation": {
    "gate_id": "review",
    "gate_fingerprint": "stable-current-decision-fingerprint",
    "status": "allowed|withheld|not_requested",
    "requested_mode": "none|yolo|yes|agent-unattended",
    "reason": "safe|unsafe|unknown|not_requested",
    "message": "display-safe summary",
    "recovery_options": [
      "Respond manually to this gate",
      "Restart through no-mistakes so the run uses a disposable worktree"
    ]
  }
}
```

Rules:

- `gate_automation.status=allowed` may be reported only after a fresh safe boundary check.
- `withheld` means no automatic response was sent and the gate remains available for manual action.
- `not_requested` means no unattended automation was active for the current gate.

## IPC RespondParams Additions

`RespondParams` adds explicit decision metadata:

```json
{
  "decision_source": "manual|unattended",
  "actor_type": "human|agent|system",
  "approval_surface": "tui|axi|headless|agent-skill|daemon|unknown",
  "consent_mode": "none|manual|yolo|yes",
  "gate_id": "review",
  "gate_fingerprint": "stable-current-decision-fingerprint"
}
```

Backward compatibility:

- Missing `decision_source` defaults to `manual`.
- Missing `approval_surface` defaults to `unknown`.
- Missing `consent_mode` defaults to `manual` for explicit `respond` calls and `none` for non-response status checks.

Enforcement:

- `decision_source=unattended` requires a fresh `safe` boundary before the daemon accepts or forwards the response.
- Manual `fix` on unsafe/unknown boundaries is accepted only for the current pending gate and is recorded as a manual source-changing action.

## CLI and AXI Output Contract

When unattended automation is withheld, TOON output includes an `automation` object:

```toon
automation:
  requested_mode: yes
  status: withheld
  boundary: unknown
  reason: stale_proof
  gate: review
  help[2]:
    Respond manually to this gate with `no-mistakes axi respond --action ...`
    Restart validation through the no-mistakes gate to create a disposable worktree
```

Rules:

- `no-mistakes axi run --yes` must return normally at the gate with `automation.status: withheld`; unattended intent may be sent to the daemon for proof refresh and audit, but no response may be forwarded to the executor or advance the gate when automation is withheld.
- `no-mistakes axi status` shows current gate automation status for gate-observing consumers.
- Safe runs keep existing output shape except for optional structured boundary fields that do not create warnings.

## TUI Contract

When YOLO is active on a safe run:

- Existing behavior remains: fix actionable findings once, approve fix-review gates, approve no-op-only gates, and suppress duplicate automatic responses.

When YOLO is active on unsafe/unknown run:

- The unattended intent is recorded by the daemon as withheld, and no response is forwarded to the executor.
- The current gate stays visible.
- The footer or gate status area displays the requested mode, boundary status, reason, and a manual/restart recovery option.
- Manual `approve`, `fix`, `skip`, and `abort` remain available as explicit per-gate actions.

## Generated Agent Guidance Contract

`skills/no-mistakes/SKILL.md` must say:

- `--yes` and YOLO are unattended consent and are honored only when `automation.status` is `allowed`.
- If `automation.status` is `withheld`, the agent must not submit a manual response unless the user explicitly gives a per-gate decision.
- The agent must report the boundary reason and recovery options exactly enough for the user to decide.

## Persistence Contract

Run boundary fields store the latest verifier result for display and diagnostics, but do not authorize future automatic action without refresh.

`run_events` records:

- Boundary refreshes.
- Unattended automation allowed, withheld, or not requested.
- Manual decisions on unsafe/unknown boundaries.
- Remote/provider write allowed or withheld states.

Events are append-only and idempotent for duplicate unattended responses with the same run ID, gate ID, fingerprint, source, and action class.

## Provider and Remote Write Contract

The following writes require fresh `safe` boundary proof when unattended:

- Git push to upstream.
- PR create/update.
- PR merge or provider status/comment/check-run/label/metadata writes when added by current or future provider adapters.
- CI auto-fix commit and push.

Read-only provider queries may continue on unsafe/unknown boundaries.

If a remote/provider write is needed while boundary is unsafe/unknown, the step must withhold automatic write, record status, and surface a manual recovery path without reordering the pipeline.
