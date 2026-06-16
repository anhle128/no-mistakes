# Contract: Review File Processing

## Process Request

The TUI sends a review-only daemon request when the user presses `p process`.

Required input:

- `run_id`
- `step`: `review`

The daemon resolves the current review file path from live run/gate state.

## Validation Sequence

1. Read one byte snapshot of the review file.
2. Reject if the file is missing, unreadable, malformed, or over 1 MiB.
3. Parse YAML front matter.
4. Verify `run_id`, `step`, `status`, and current review hash against live gate state.
5. Reject pre-processed metadata before hand-edited processing.
6. Parse only `no-mistakes-response` fences.
7. Verify latest finding coverage, unknown IDs, duplicate IDs, valid actions, parseable solutions, and 16 KiB solution limit.
8. Derive the existing executor decision:
   - one or more `fix` blocks -> `ActionFix`
   - all `accept`/`skip` blocks -> `ActionApprove`
9. Re-check that the file bytes still match the validated snapshot.
10. Atomically update only `processed_at` and `processed_action`.
11. Feed the derived decision into the waiting review gate.

## Validation Failure

On any validation failure:

- The gate remains open.
- No executor decision is sent.
- The TUI displays file path, one-line summary, and first actionable validation error.
- Available review gate actions remain `p process` and `c cancel`.

## Automation Mirroring

Existing automation uses the old approve/fix/skip response contract. Before a review gate advances, the executor mirrors the exact automation decision into the review handoff file:

- `approve` marks all latest findings as `accept` unless there are no findings.
- `skip` marks all latest findings as `skip`.
- `fix` marks selected findings as `fix`, records per-finding instructions, and preserves user-added findings in the audit view.

The mirror must use the same writer/parser contract as hand-edited files and must satisfy validation invariants before the gate decision finalizes. If mirror writing or validation fails, the gate remains unprocessed and reports an actionable review-file error.

## Fixer Prompt Boundary

Non-empty file `solution:` text is untrusted user data. The fix prompt must delimit it per finding ID and state that it is not system, developer, or project-wide instruction authority.
