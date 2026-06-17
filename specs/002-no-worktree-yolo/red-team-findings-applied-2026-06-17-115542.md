# Red Team Findings: No-Worktree YOLO Guard

Session: `RT-002-no-worktree-yolo-2026-06-17`
Target: `specs/002-no-worktree-yolo/spec.md`
Date: 2026-06-17
Maintainer: unspecified
Status: ARCHIVED
**Applied:** 2026-06-17-115542
Lenses: Trust-Boundary Adversary
Selection method: auto
Supporting context:
- `specs/002-no-worktree-yolo/no-worktree-yolo.md`
- `.specify/memory/constitution.md`

Warnings:
- `.specify/memory/constitution.md` does not declare `## Red Team Trigger Criteria`; ran in bootstrap mode using default trigger categories.
- Matched triggers included `immutability_audit`, `multi_party`, and `contracts`, but the scaffolded catalog only provided a matching lens for `multi_party` and `contracts`.
- Selected lens count was 1, below the recommended 3-lens diversity.

## 1. Session Summary

Pending maintainer review.

Findings by severity:
- CRITICAL: 0
- HIGH: 2
- MEDIUM: 3
- LOW: 0

Findings by lens:
- Trust-Boundary Adversary: 5

## 2. Findings

| ID | Lens | Severity | Location | Finding | Suggested Resolution | Status |
|---|---|---|---|---|---|---|
| F-RT-002-no-worktree-yolo-2026-06-17-001 | Trust-Boundary Adversary | HIGH | FR-003 | The spec requires a "verified disposable source workspace" but does not define the trusted verifier or the proof properties. A low-privilege agent could mark a primary checkout as disposable by supplying a misleading path, symlinked directory, nested worktree, bind mount, or repo metadata that appears isolated while writes still affect trusted source. | Define the boundary proof as independently computed from trusted local facts, including canonical realpaths, git worktree/common-dir checks, workspace creator/owner metadata, symlink escape rejection, and source-write path enforcement. | spec-fix |
| F-RT-002-no-worktree-yolo-2026-06-17-002 | Trust-Boundary Adversary | HIGH | FR-002, FR-015 | The Run record is treated as the authoritative store for safe/unsafe/unknown classification, but the spec does not protect against stale, replayed, or tampered Run state. If an agent can edit or replay a prior safe classification after reattach, restart, or workspace movement, it can resume unattended source-changing actions without current isolation proof. | Require each automatic action to recompute or cryptographically bind the classification to current workspace identity, gate fingerprint, and freshness metadata; stale or unverifiable persisted classifications should degrade to unknown. | spec-fix |
| F-RT-002-no-worktree-yolo-2026-06-17-003 | Trust-Boundary Adversary | MEDIUM | FR-005 | Manual approve, fix, skip, and cancel remain available on unsafe or unknown-boundary runs, but the spec does not define what counts as a fresh human decision. A headless agent, AXI caller, or scripted TUI input could relabel an unattended action as manual and bypass the higher-privilege human gate. | Bind manual decisions to an authenticated interactive user/session, record actor type and surface, require a current gate fingerprint, and reject agent-originated or replayed decisions as manual proof. | spec-fix |
| F-RT-002-no-worktree-yolo-2026-06-17-004 | Trust-Boundary Adversary | MEDIUM | FR-004 | Remote/provider side effects are blocked only when they "advance external review state," which leaves a trust gap around comments, statuses, labels, check runs, or metadata writes that can influence reviewers or merge automation. An adversary could choose a provider action framed as informational while still changing PR outcome or authority signals. | Use a default-deny policy for all remote provider writes on unsafe or unknown boundaries, with an explicit allowlist for truly read-only operations and separately authorized manual remote actions. | spec-fix |
| F-RT-002-no-worktree-yolo-2026-06-17-005 | Trust-Boundary Adversary | MEDIUM | FR-006, FR-014 | Duplicate automatic-response prevention depends on run ID plus gate ID or step plus a gate version/fingerprint, but the spec does not say who computes that fingerprint or what fields it covers. If the fingerprint is derived from mutable status text or agent-controlled gate metadata, repeated events can be made to look like new gates and trigger multiple automatic responses. | Define gate identity as a server-side or controller-owned value over immutable gate inputs, decision state, and run identity; changing user-facing status text or agent guidance must not create a new actionable gate version. | spec-fix |

## 3. Resolutions Log

### F-RT-002-no-worktree-yolo-2026-06-17-001

Category: spec-fix
Payload:
Reasoning:
Verification: FR-003 says, "A run MUST be considered safe for unattended source-changing YOLO only when intentional source-changing work is bounded to a verified disposable source workspace; explicitly allowed evidence/test boundaries may authorize non-source artifacts only and MUST NOT authorize source writes outside the disposable run workspace." The finding's premise holds: "verified" states the outcome but does not say who computes it or which local facts are trusted. Evidence: `.specify/memory/constitution.md:47` says pipeline work must run in "a disposable worktree or an explicitly bounded test directory," `internal/pipeline/steps/execution_context.go:19` says agents run inside "an isolated git worktree," and `internal/pipeline/steps/evidence.go:81` already validates evidence paths as relative paths "that stays inside the repo worktree." This is not `new-OQ` because the local contract already establishes the shape: controller-owned verification from filesystem/Git facts, with evidence boundaries non-source only. This is not `skipped` because the cited spec text does omit proof properties. Band-aid rejected: accepting a user-supplied safe flag or relying on prompt steering would preserve the same trust-boundary hole; the durable fix is to make controller-owned boundary verification part of the functional contract.
Target: specs/002-no-worktree-yolo/spec.md
Before:
```text
- **FR-003**: A run MUST be considered safe for unattended source-changing YOLO only when intentional source-changing work is bounded to a verified disposable source workspace; explicitly allowed evidence/test boundaries may authorize non-source artifacts only and MUST NOT authorize source writes outside the disposable run workspace.
```
After:
```text
- **FR-003**: A run MUST be considered safe for unattended source-changing YOLO only when a controller-owned verifier independently proves that intentional source-changing work is bounded to a disposable source workspace for the current Run, using canonical realpaths and trusted Git workspace metadata rather than user-supplied or agent-supplied path claims; the verifier MUST reject unknown, symlink-escaping, nested, or metadata-inconsistent boundaries. Explicitly allowed evidence/test boundaries may authorize non-source artifacts only and MUST NOT authorize source writes outside the disposable run workspace.
```
Status: applied
Applied-at: 2026-06-17T11:55:42+07:00
Downstream-ref: specs/002-no-worktree-yolo/spec.md:101

### F-RT-002-no-worktree-yolo-2026-06-17-002

Category: spec-fix
Payload:
Reasoning:
Verification: FR-002 says, "System MUST persist the safe, unsafe, or unknown execution-boundary classification on the Run and refresh it before the first automatic gate action and before any later automatic action after reattach, rerun, daemon restart, or status refresh," and FR-015 says unattended actions pause while unknown and resume only after safe proof is restored. The finding overstates the premise slightly because the spec already requires refreshes, but it is still real: the spec does not say persisted classification is only a cache/audit record and cannot authorize an action when its verifier inputs are stale or unavailable. Evidence: `internal/db/schema.go:12` through `internal/db/schema.go:23` show the current `runs` table has no boundary-classification or freshness fields, and `internal/db/run.go:10` through `internal/db/run.go:28` show the Run model currently persists status, SHAs, PR URL, error, and intent only. This is not `new-OQ` because the clarification at `specs/002-no-worktree-yolo/spec.md:13` already chooses Run persistence plus refresh before each automatic gate action. This is not `accepted-risk` because the fix is a narrow contract clarification, not a costly redesign. Band-aid rejected: adding a timestamp-only expiry or trusting a prior safe row after restart would still allow stale state to drive automation; the durable fix is to require recomputation against current workspace identity and downgrade unverifiable state to unknown.
Target: specs/002-no-worktree-yolo/spec.md
Before:
```text
- **FR-002**: System MUST persist the safe, unsafe, or unknown execution-boundary classification on the Run and refresh it before the first automatic gate action and before any later automatic action after reattach, rerun, daemon restart, or status refresh.
```
After:
```text
- **FR-002**: System MUST persist the safe, unsafe, or unknown execution-boundary classification on the Run with the verifier inputs and freshness metadata needed to explain the decision, but persisted classification MUST NOT by itself authorize unattended action. Before each automatic gate action, and before any later automatic action after reattach, rerun, daemon restart, status refresh, or workspace movement, the system MUST recompute the classification against the current workspace identity and degrade to unknown when current proof is stale, missing, or inconsistent.
```
Status: applied
Applied-at: 2026-06-17T11:55:42+07:00
Downstream-ref: specs/002-no-worktree-yolo/spec.md:100

### F-RT-002-no-worktree-yolo-2026-06-17-003

Category: spec-fix
Payload:
Reasoning:
Verification: FR-005 says, "Manual approve, fix, skip, and cancel actions MUST remain available on unsafe or unknown-boundary runs, and manual fix MUST be recorded as explicit per-gate manual source-changing intent distinguishable from unattended automatic decisions." The finding's core premise holds because the spec distinguishes manual from unattended but does not define the response proof beyond "explicit per-gate." Evidence: `docs/src/content/docs/reference/cli.md:76` states that `--yes` is standing consent that fixes actionable findings and accepts fix review, while `docs/src/content/docs/reference/cli.md:84` through `docs/src/content/docs/reference/cli.md:104` show `axi respond` can answer a gate and then optionally continue with `--yes`; `internal/ipc/protocol.go:111` through `internal/ipc/protocol.go:118` show the current response payload has run, step, action, findings, and fix data but no actor, surface, or gate fingerprint. This is not `new-OQ` because the constitution at `.specify/memory/constitution.md:50` through `.specify/memory/constitution.md:52` already permits human decisions through TUI, AXI, or an equivalent approval surface, so the spec should preserve those surfaces rather than invent an auth policy. This is not `skipped` because treating any non-`--yes` response as manual would be too weak for unsafe/unknown boundaries. Band-aid rejected: blocking AXI or requiring only an authenticated interactive TUI would conflict with the documented agent-facing workflow; the durable fix is to record the response actor/surface and bind manual intent to the current gate identity while keeping broad unattended consent distinct.
Target: specs/002-no-worktree-yolo/spec.md
Before:
```text
- **FR-005**: Manual approve, fix, skip, and cancel actions MUST remain available on unsafe or unknown-boundary runs, and manual fix MUST be recorded as explicit per-gate manual source-changing intent distinguishable from unattended automatic decisions.
```
After:
```text
- **FR-005**: Manual approve, fix, skip, and cancel actions MUST remain available on unsafe or unknown-boundary runs only as explicit responses to the current pending gate, not as reuse of broad unattended consent. Manual fix MUST be recorded as explicit per-gate manual source-changing intent distinguishable from unattended automatic decisions, including the response actor type, approval surface, gate identity, and whether any subsequent `--yes` or YOLO continuation was requested; generated agent guidance MUST require a human decision before an agent submits a manual response on an unsafe or unknown-boundary run.
```
Status: applied
Applied-at: 2026-06-17T11:55:42+07:00
Downstream-ref: specs/002-no-worktree-yolo/spec.md:103

### F-RT-002-no-worktree-yolo-2026-06-17-004

Category: spec-fix
Payload:
Reasoning:
Verification: FR-004 says, "When the run boundary is unsafe or unknown, the system MUST fail closed by withholding automatic fix, approve, skip, git push, PR create/update/merge, and provider status/comment actions that advance external review state while leaving the gate available for manual user action." The finding's premise mostly holds: the spec is already conservative for current PR advancement, but the phrase "that advance external review state" leaves avoidable ambiguity for future remote writes framed as informational. Evidence: `internal/scm/host.go:116` through `internal/scm/host.go:130` show current provider writes are PR create/update while PR state, checks, mergeability, and failed-log fetches are read operations; `docs/src/content/docs/reference/pipeline-steps.md:123` through `docs/src/content/docs/reference/pipeline-steps.md:158` document automatic push and PR create/update as external side effects after local checks pass. This is not `out-of-scope` because FR-004 already owns unsafe/unknown remote side effects; it should state the invariant clearly. This is not `skipped` because the ambiguity could matter as soon as a provider integration adds labels, comments, statuses, or metadata writes. Band-aid rejected: enumerating only today's `CreatePR` and `UpdatePR` calls would let the next provider write reopen the same hole; the durable fix is default-deny for automatic remote provider writes while still allowing read-only provider queries.
Target: specs/002-no-worktree-yolo/spec.md
Before:
```text
- **FR-004**: When the run boundary is unsafe or unknown, the system MUST fail closed by withholding automatic fix, approve, skip, git push, PR create/update/merge, and provider status/comment actions that advance external review state while leaving the gate available for manual user action.
```
After:
```text
- **FR-004**: When the run boundary is unsafe or unknown, the system MUST fail closed by withholding automatic fix, approve, skip, git push, PR create/update/merge, and all automatic remote provider write actions, including PR body updates, comments, statuses or check-run writes, labels, and metadata changes; read-only provider queries may continue, and the gate MUST remain available for manual user action.
```
Status: applied
Applied-at: 2026-06-17T11:55:42+07:00
Downstream-ref: specs/002-no-worktree-yolo/spec.md:102

### F-RT-002-no-worktree-yolo-2026-06-17-005

Category: spec-fix
Payload:
Reasoning:
Verification: FR-006 says duplicate automatic responses are prevented "using run ID plus gate ID or step plus a gate version or fingerprint," and FR-014 says repeated unsafe or unknown YOLO attempts must not enqueue duplicate responses for "the same run ID plus gate ID or step plus gate version or fingerprint." The finding's premise holds because neither requirement names the owner of that identity nor excludes mutable text or agent metadata from the fingerprint. Evidence: `internal/tui/commands.go:76` through `internal/tui/commands.go:83` document process-local duplicate suppression, `internal/tui/commands.go:88` through `internal/tui/commands.go:98` key it by step name in in-memory maps, and `internal/cli/axi_drive.go:387` through `internal/cli/axi_drive.go:390` only prevents a same-process double-approve race by waiting for the step to leave the observed gate. This is not `new-OQ` because the spec already chose the identity shape at `specs/002-no-worktree-yolo/spec.md:19`; the missing piece is a controller-owned computation rule. This is not `accepted-risk` because a one-line contract addition can guide implementation without large design work. Band-aid rejected: keeping only TUI/AXI in-memory maps would fail across restart or reattach; the durable fix is a persisted, controller-owned gate identity over stable decision inputs.
Target: specs/002-no-worktree-yolo/spec.md
Before:
```text
- **FR-014**: Error and recovery behavior MUST be idempotent: repeated attempts to enable YOLO on an unsafe or unknown-boundary run must not enqueue duplicate responses or change the paused gate state for the same run ID plus gate ID or step plus gate version or fingerprint.
```
After:
```text
- **FR-014**: Error and recovery behavior MUST be idempotent: repeated attempts to enable YOLO on an unsafe or unknown-boundary run must not enqueue duplicate responses or change the paused gate state for the same controller-owned gate identity. The gate identity MUST be computed from immutable current decision inputs for the run, including run ID, gate ID or step, gate status, current step round or gate version, and the decision fingerprint; user-facing status text, generated agent guidance, or mutable agent-provided metadata MUST NOT create a new actionable gate identity.
```
Status: applied
Applied-at: 2026-06-17T11:55:42+07:00
Downstream-ref: specs/002-no-worktree-yolo/spec.md:112

## 5. Session Metadata

```yaml
schema_version: red-team-findings/v1
session_id: RT-002-no-worktree-yolo-2026-06-17
target: specs/002-no-worktree-yolo/spec.md
feature_id: 002-no-worktree-yolo
date: 2026-06-17
arguments:
  resolved_feature_directory: specs/002-no-worktree-yolo
  target_spec_path: specs/002-no-worktree-yolo/spec.md
  yes: true
matched_triggers:
  - immutability_audit
  - multi_party
  - contracts
selected_lenses:
  - Trust-Boundary Adversary
selection_method: auto
warnings:
  - constitution lacks a Red Team Trigger Criteria section; bootstrap trigger matching used
  - selected lens count below recommended diversity
lens_failures: []
dropped_findings: 0
counts:
  by_severity:
    CRITICAL: 0
    HIGH: 2
    MEDIUM: 3
    LOW: 0
  by_lens:
    Trust-Boundary Adversary: 5
resolution_counts:
  spec-fix: 5
  new-OQ: 0
  accepted-risk: 0
  out-of-scope: 0
  skipped: 0
unresolved: 0
resolution_state: applied
apply:
  applied_at: 2026-06-17T11:55:42+07:00
  applied_by: unspecified
  resolutions:
    spec_fix: 5
    new_OQ: 0
    accepted_risk: 0
    out_of_scope: 0
    skipped: 0
  unresolved: 0
  allow_historical_edits: true
  historical_edits_applied:
    - F-RT-002-no-worktree-yolo-2026-06-17-001:specs/002-no-worktree-yolo/spec.md
    - F-RT-002-no-worktree-yolo-2026-06-17-002:specs/002-no-worktree-yolo/spec.md
    - F-RT-002-no-worktree-yolo-2026-06-17-003:specs/002-no-worktree-yolo/spec.md
    - F-RT-002-no-worktree-yolo-2026-06-17-004:specs/002-no-worktree-yolo/spec.md
    - F-RT-002-no-worktree-yolo-2026-06-17-005:specs/002-no-worktree-yolo/spec.md
```
