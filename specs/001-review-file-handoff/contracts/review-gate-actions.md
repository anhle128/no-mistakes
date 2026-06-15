# Contract: Review Gate Actions

## Compatibility Matrix

| Surface | Review awaiting/fix-review behavior | Existing behavior preserved |
| --- | --- | --- |
| TUI keys | Shows compact summary, review file path, `p process`, `c cancel` | Non-review gates keep approve/fix/skip/edit/add/select controls |
| TUI `p process` | Parses and validates current file, stamps processed metadata, dispatches approve or fix through executor | Uses existing executor response transition |
| TUI `c cancel` | Aborts only if the original run/step gate identity still matches active gate | Uses existing user-abort path |
| AXI `respond approve` | Accepted; records equivalent review audit source `automation` and approves | Command shape unchanged |
| AXI `respond fix --findings ... --instructions ...` | Accepted; records equivalent decisions and dispatches selected findings | Command shape unchanged |
| AXI `respond skip` | Accepted; records equivalent review audit source `automation` and skips | Command shape unchanged |
| Direct IPC `respond` | Accepted with same semantics as AXI | `RespondParams` remains backward compatible |
| Yolo | Preserves current auto-resolution behavior and records source `automation` or `auto_fix` as appropriate | Existing yolo fix-once/approve-fix-review flow remains |
| Configured automatic review auto-fix | Runs before handoff generation whenever existing executor would auto-fix | No new manual process action required |

## TUI Key Contract

During review `awaiting_approval` and `fix_review` gates:

- `p`: process current review file
- `c`: cancel active matching run through existing abort path

The following old review-specific keys must not be rendered or active for review gates:

- `a approve`
- `f fix`
- `s skip`
- edit
- add
- select all
- select none
- per-finding toggle

Non-review gates keep existing controls.

## Process Mapping

After successful file validation:

- if any response action is `fix`, dispatch `RespondWithOverrides(review, fix, fixedIDs, instructions, nil)`
- if all responses are `accept` or `skip`, dispatch `RespondWithOverrides(review, approve, nil, nil, nil)`
- preserve distinct `accept` and `skip` in review handoff state and audit summary

## Stale Gate Protection

Process and cancel actions are bound to the gate's original:

- run ID
- step name
- raw status
- review cycle ID
- finding digest

If the active gate no longer matches, the action no-ops and reports a concise stale-gate error with the current valid review file path or regeneration state when available.

## Processing Transaction

Commit points:

1. Validate active gate identity, file metadata, response blocks, and pending processed state.
2. Atomically overwrite the file metadata with processed action/timestamp and resolved-decision summary.
3. Persist the same processed state to `review_handoff_json`.
4. Dispatch the existing executor response.
5. Expose live event, reattach, and AXI state from the committed state.

Partial failure rule:

- If commit point 1 fails, no state changes.
- If commit point 2 fails, no executor response is dispatched.
- If commit point 3 fails after the file was stamped, processing reports failure and does not dispatch response.
- If commit point 4 fails, state remains processed with an error surfaced so the user does not retry against a stale pending file.
