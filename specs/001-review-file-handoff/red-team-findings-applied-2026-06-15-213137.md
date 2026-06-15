# Red Team Findings: Review File Handoff

- **Session ID**: `RT-001-review-file-handoff-2026-06-15`
- **Target**: `specs/001-review-file-handoff/spec.md`
- **Date**: 2026-06-15
- **Maintainer**: Kevin Le
- **Matched triggers**: `contracts`, `multi_party`, `immutability_audit`
- **Lenses**: Automation Contract Adversary; Gate Semantics Adversary; Handoff File Integrity Adversary; PR Audit Trail Adversary; User Authority Adversary
- **Selection method**: auto
- **Supporting context**: `.specify/memory/constitution.md`; `specs/001-review-file-handoff/checklists/requirements.md`; `specs/001-review-file-handoff/clarifications-applied-2026-06-15-205640.md`
- **Wall-clock**: native parallel subagent batch completed in-session
- Status: ARCHIVED
- **Applied:** 2026-06-15-213137

## 1. Session Summary

Pending maintainer review.

## 2. Findings

| ID | Lens | Severity | Location | Finding | Suggested Resolution | Status |
|---|---|---|---|---|---|---|
| F-RT-001-review-file-handoff-2026-06-15-001 | Automation Contract Adversary | HIGH | FR-026, FR-027, FR-034 | The spec requires `review_file` and `phase` to be available from live events, reattached run state, and `axi status` even when the original event was missed, but also says the first version must not require a persisted run-history format change. That leaves no authoritative source for `review_file` after reattach, daemon restart, file overwrite, or anchor-path drift, so surfaces can legitimately reconstruct different paths or omit the field. | Define the durable source of truth for `review_file` and `phase`, including whether an additive step metadata field, deterministic path derivation, or review-file scan owns reattach reconstruction. If persistence is required, narrow FR-034 to forbid incompatible migrations while allowing additive fields. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-002 | Automation Contract Adversary | HIGH | FR-027, FR-031, FR-032 | The spec names additive `phase` and `review_file` values but does not define the exact schema placement for live JSON events, reattached run state, `axi status` run output, or `axi status` gate output. Existing automation consumers may parse run step rows, gate objects, and help text differently, so adding fields in only one object or renaming or reshaping gate findings would create drift while still appearing to satisfy the prose. | Add a compatibility contract table for each automation surface with exact field names, nesting, cardinality, and omission rules. Explicitly state that existing raw status fields, finding IDs, and response command affordances remain unchanged. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-003 | Automation Contract Adversary | HIGH | FR-015, FR-026, FR-032, Assumptions | Review-specific terminal controls are hidden, but existing automation response commands are required to remain compatible. The spec does not say whether `axi respond --action approve|fix|skip`, yolo auto-approval, or direct IPC respond calls should still work on review gates, be rejected with review-file guidance, or be translated through file processing, which can split terminal behavior from automation behavior. | Define a command compatibility matrix for TUI keys, AXI commands, and IPC responses during review and fix-review gates. If direct automation responses remain supported, specify how they coexist with `review_file`; if not, define the replacement process command and machine-readable error shape. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-004 | Gate Semantics Adversary | HIGH | FR-009, FR-026, FR-032, Assumptions | The spec says existing automation response commands remain compatible while the terminal path moves to file processing, but it does not require automation approve, fix, or skip responses to pass through the same validation and per-finding decision recording as `p process`. This leaves a path where terminal users cannot approve without a valid handoff file, but AXI or daemon automation could still approve using legacy commands without processing the latest file decisions, changing what a passed review gate means depending on surface. | Define a single review-gate decision transition shared by terminal `p process`, reattached sessions, AXI, and daemon automation. Either require legacy automation commands to validate and process equivalent finding decisions, or explicitly document and test their bypass semantics while ensuring the PR audit file records that path. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-005 | Gate Semantics Adversary | HIGH | FR-002, FR-011, FR-017, FR-018 | Validation requires run, status, branch, and latest finding IDs, but not a review-cycle generation, finding-set hash, or step attempt identifier. A stale file from an earlier review or fix-review cycle with the same run and status and repeated normalized finding IDs could be accepted even though its recommendations and user answers do not correspond to the current gate state. | Add a required review-cycle token or deterministic finding-set checksum to the handoff metadata and require processing to match it against the active in-memory or latest persisted review result. Include stale same-ID review-cycle tests. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-006 | Gate Semantics Adversary | HIGH | FR-001, FR-007, FR-033, SC-006 | The spec says a handoff file is created whenever review produces a human decision point and that automatic review auto-fix behavior must not change, but it does not state precedence between existing automatic auto-fix and the new manual handoff defaults. An implementation could generate a handoff and pause on findings that would previously have auto-fixed, silently weakening the existing gate's automatic remediation path across terminal, automation, or daemon runs. | State that existing configured automatic review auto-fix takes precedence and must not require a handoff or process action until the existing behavior would have reached a human approval point. Add explicit terminal, daemon, and automation regression scenarios for auto-fix-enabled review runs. | skipped |
| F-RT-001-review-file-handoff-2026-06-15-007 | Handoff File Integrity Adversary | HIGH | FR-004, FR-005, FR-008 | The spec says prose outside fenced response blocks MUST NOT affect processing, but also says an empty or comment-only fix solution falls back to recommendation option 1. Because recommendations are rendered as editable Markdown prose outside the response block, an implementation could accidentally use a user-edited or stale recommendation as the fix instruction despite the trust-boundary rule. | Specify that fallback recommendation option 1 is read from the active in-memory or latest finding model, not from Markdown prose. If the file must carry fallback text, place it in validated structured metadata inside the fenced block or another explicitly trusted, non-user-editable contract. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-008 | Handoff File Integrity Adversary | HIGH | FR-021, FR-022, FR-023, FR-039 | The path placement rules require files to be written inside the checkout or isolated work area, but do not require canonical path resolution, symlink rejection, regular-file checks, or post-resolution boundary validation. A changed `plan.md` or `tasks.md` symlink, nested checkout, or path traversal-like relative path could cause the handoff write or PR audit copy to target outside the intended boundary while still appearing checkout-relative. | Require resolving anchors and destination paths with symlink-aware canonicalization, reject non-regular anchor files for placement, and validate the final write or copy target remains under the intended checkout or isolated worktree root immediately before writing. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-009 | Handoff File Integrity Adversary | HIGH | FR-002, FR-011, FR-017, FR-018 | Stale-file validation relies on expected metadata matching the active run and review status, but the spec does not require a review-cycle generation, finding-set hash, or processed and pending freshness guard. A file from an earlier review cycle in the same run and status could contain valid-looking IDs and metadata, causing stale answers to be processed or previously processed files to be replayed. | Add a required review-cycle identifier or finding-set digest to the file metadata and require `processed_action: pending` plus empty processed timestamp before processing. Processing should reject files whose cycle or digest does not match the latest active review result. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-010 | PR Audit Trail Adversary | HIGH | FR-018, FR-019, User Story 4 | The spec says each new review or fix-review result overwrites the current handoff file and does not preserve old answer blocks, but the PR audit trail is supposed to show the decisions that shaped approval or fix actions. In a successful fix-review with no remaining findings, the final PR artifact could contain only "no remaining review findings" plus an optional applied-fix summary, omitting the original finding IDs, user actions, and solution or default recommendation choices that caused the fixes. | Require the final no-remaining-findings or fix-review file to include a compact resolved-decision summary: prior finding IDs, action taken, user solution or default recommendation used, processed action, and processed timestamp. If full prior answers are intentionally excluded, require an explicit audit section with enough equivalent data for PR reviewers. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-011 | PR Audit Trail Adversary | HIGH | FR-021, FR-023, FR-037 | Anchor selection can use any single uncommitted `plan.md` or `tasks.md`, including staged, modified, or untracked files unrelated to the pipeline's intended publishable changes. That can place the review file beside an unrelated anchor; a later implementation that stages by directory or broad path could drag the anchor or neighboring working-tree changes into the PR despite FR-037. | Constrain anchors to the active feature or pipeline-owned change set, or require anchor use to be placement-only with an explicit commit allowlist containing the review file path and intentional pipeline outputs. Add a negative acceptance scenario with an unrelated changed `plan.md` or `tasks.md` proving neither the anchor nor nearby changes are committed. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-012 | PR Audit Trail Adversary | HIGH | FR-035, FR-036 | FR-035 and FR-036 state the desired outcome, but they do not require the PR branch commit builder to treat the review file as an explicit publishable artifact independent of normal code or doc diffs. If commit creation first checks for remediated file changes and returns early on an empty diff, the audit file can be omitted exactly when it is the only remaining change. | Add a requirement that the stored latest review-file relative path is always staged or copied before any no-op commit decision, and that a review-file-only diff is sufficient to create the PR branch commit. Include a test where all findings are accepted or fixes produce no code diff and the review file is still committed. | skipped |
| F-RT-001-review-file-handoff-2026-06-15-013 | User Authority Adversary | HIGH | FR-007/FR-008/FR-014 | Generated response blocks contain default actions, including `fix` for automatically fixable findings, and processing treats any valid `fix` block as user authorization. Because the compact terminal only shows a summary, path, and actions, and the spec does not require explicit user-edited acknowledgement for fix defaults, a user can press `p process` on an unreviewed generated file and trigger remediation as if they had made an affirmative decision. | Require explicit user authority for any `fix` action, such as a non-default confirmation marker in the response block, or require the terminal or review file to clearly surface default-fix counts and block processing until generated defaults have been intentionally confirmed. | skipped |
| F-RT-001-review-file-handoff-2026-06-15-014 | User Authority Adversary | HIGH | FR-010/FR-018/FR-035 | The spec requires `accept` and `skip` to remain distinct in the review file, but also requires the current handoff file to be overwritten on each new review or fix-review result and only the latest file to be copied into the PR audit trail. This can erase earlier accept/skip distinctions that shaped the run, especially after a fix cycle, undermining the stated auditability goal. | Persist processed per-finding decisions, including accept versus skip, in a durable audit section or separate run-level record that survives handoff regeneration and is included in the PR audit copy. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-015 | Automation Contract Adversary | MEDIUM | FR-017, FR-018, FR-026, FR-028 | Phase labels are derived from status, while the handoff file is overwritten per review cycle, but the spec omits transition timing between file write, stored run-state update, and live event emission. A reattached session during or after a fix-review transition could show `Review fix result` with the previous review file, or `Review preview complete` without the newly generated file, especially when the live event was missed. | Specify the atomic ordering for each review-cycle transition: generate or overwrite file, persist current review-file reference and phase inputs, then emit live events and expose status. Add stale-file invalidation rules for reattach if any step in that sequence fails. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-016 | Automation Contract Adversary | MEDIUM | FR-002, FR-008, FR-009, FR-012, FR-026 | Processed metadata is updated after successful processing, but the spec does not define whether that update happens before the approval or fix response is sent, after the daemon accepts it, or after the next phase transition. This can leave automation seeing a `Fixing review issues` phase while the referenced file still says `processed_action: pending`, or a validation failure preserving user contents while another surface reports a processed state. | Define processing as a transaction with explicit commit points for validation, processed metadata update, response dispatch, and event/state emission. State which side wins on partial failure and require `axi status` and reattach to expose the same committed processed state as the file. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-017 | Gate Semantics Adversary | MEDIUM | FR-002, FR-035, FR-036, SC-005 | The PR audit requirement only says the latest review file is included in the commit; it does not require that the file show a non-pending processed action after the gate passes. A bug or alternate approval path could produce a PR containing a `processed_action: pending` review file while the pipeline claims the review gate passed, making reviewers believe findings were processed when the audit artifact says otherwise. | Require the push or PR preparation phase to block or fail if a review gate has passed but the latest handoff file still has pending processing metadata, except for explicitly documented no-handoff auto-fix paths. Add a pre-commit audit consistency check to the success criteria. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-018 | Gate Semantics Adversary | MEDIUM | FR-008, FR-011, FR-032 | For `fix` responses with empty or comment-only solution text, the spec falls back to recommendation option 1, but it does not require validation that option 1 exists and is parseable for every fixable finding. If recommendation generation is incomplete, processing could send an empty or unintended fix instruction while still treating the file as valid, changing remediation behavior without changing raw statuses or commands. | Require generated fixable findings to have a machine-detectable option 1 before they can default to `fix`, and require processing to reject empty-solution `fix` responses when no valid option 1 exists. Cover this in parser and validation tests. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-019 | Handoff File Integrity Adversary | MEDIUM | FR-005, FR-011 | The response block contract does not define the exact key syntax or parsing grammar for line-oriented fields. Ambiguities such as duplicate `action` lines, duplicate `solution` lines, whitespace or case variants, extra fields, nested fences, multi-line continuations, or multiple IDs in one block could lead different parsers to accept malformed edits differently. | Define a strict block grammar: one ID field with exact name, exactly one action line, exactly one solution line, permitted whitespace rules, unknown-field handling, duplicate-line rejection, and fence termination behavior. Add validation requirements for each malformed variant. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-020 | Handoff File Integrity Adversary | MEDIUM | FR-012, FR-017, FR-018, Assumptions | The spec mandates one current file per run and overwriting it on every new review or fix-review result, but does not protect meaningful unprocessed user edits during that overwrite. If a new review result arrives while a developer is editing or after validation failed, their answer text can be lost even though the feature promises confidence and preservation on validation failure. | Require atomic writes with conflict detection before overwrite, such as checking whether the existing pending file changed since generation. Preserve superseded pending files with a timestamped backup or block regeneration with a concise stale-edit diagnostic before replacing user-authored answers. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-021 | PR Audit Trail Adversary | MEDIUM | FR-022, FR-035, FR-039 | The spec does not pin whether anchor resolution and review-file copying use the project checkout root, the isolated pipeline work-area root, or a persisted repository-relative path. Re-running anchor discovery in the isolated work area can disagree with the original checkout state, while copying by an absolute checkout path can fail or target the wrong location once the pipeline moves into its disposable work area. | Require the system to persist a normalized repository-relative review file path at generation time, validate it stays inside the checkout, and copy that exact relative path into the isolated work area. State that the push phase must not re-run anchor selection. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-022 | User Authority Adversary | MEDIUM | FR-011/FR-012 | Validation failure keeps the compact gate open with only `process` and `cancel`, but the required error is only one concise message plus the file path. For stale or wrong-run files, this can leave the user repeatedly retrying `process` against a dangerous stale artifact without seeing the expected run/status or how to recover. | For stale metadata failures, require the error to show actual versus expected run/status and the current valid review file path or regeneration state, while still preserving the user's file contents. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-15-023 | User Authority Adversary | MEDIUM | FR-013 and Edge Cases | Cancel is specified as aborting the active run through the existing user-abort path, but the edge cases include interrupted, canceled, or superseded runs. Without a gate-bound run identity requirement, a stale terminal gate could route `cancel` to the wrong currently active run or leave an old run half-processed. | Specify that process and cancel actions must be bound to the gate's original run/step identity, must no-op or show a stale-gate error if that run is no longer active, and must only call the existing user-abort path for the matching active run. | spec-fix |

## 3. Resolutions Log

### F-RT-001-review-file-handoff-2026-06-15-001

- Category: spec-fix
- Reasoning: Verification: FR-026 says the system must expose the review file path to "live gate events, reattached terminal sessions, reattached run state, and `axi status` run/gate output even when the original live event was missed," so the finding's premise holds. Evidence: FR-034 also says the first release must not "require a persisted run-history format change," while `internal/db/schema.go:25`-`38` shows the current step row has status and findings fields but no review-file field. This is not `new-OQ` because the required behavior is already fixed by FR-026; the missing piece is the durable source contract. Rejected band-aid: deriving the path opportunistically from whatever file happens to exist after reattach, because that would make surfaces disagree. The durable fix is to allow backward-compatible additive metadata or deterministic derivation and define it as the source of truth without an incompatible run-history migration.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-026**: The system MUST expose the review file path to live gate events, reattached terminal sessions, reattached run state, and `axi status` run/gate output even when the original live event was missed.
  After: - **FR-026**: The system MUST expose the review file path to live gate events, reattached terminal sessions, reattached run state, and `axi status` run/gate output even when the original live event was missed; the authoritative source MUST be the current review cycle's persisted review-file reference and phase inputs, or a deterministic derivation from persisted run/step state, and the first version MAY add backward-compatible metadata fields to existing run or step records but MUST NOT require an incompatible run-history migration.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:126

### F-RT-001-review-file-handoff-2026-06-15-002

- Category: spec-fix
- Reasoning: Verification: FR-027 requires additive `phase` and `review_file` values but does not say exactly where they live on each automation surface, so the finding's premise holds. Evidence: `internal/ipc/protocol.go:195`-`214` defines the current step-result JSON shape and `internal/cli/axi_render.go:244`-`280` defines the existing gate object/help shape; the spec must preserve those raw fields while adding new ones. This is not `new-OQ` because the clarification already chooses the surfaces at `specs/001-review-file-handoff/spec.md:20`. Rejected band-aid: adding the values only to the terminal event payload, because `axi status` and reattach would still drift. The durable fix is a compact compatibility contract that pins field names, placement, omission rules, and raw-status compatibility.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-027**: The system MUST expose additive human-readable `phase` and `review_file` values in live gate events, reattached run state, and `axi status` run/gate output without renaming raw statuses.
  After: - **FR-027**: The system MUST expose additive human-readable `phase` and `review_file` values without renaming raw statuses, using this compatibility contract: live review gate events carry top-level `phase` and `review_file` values when a review file is known; reattached run state carries the same additive values on the review step result; `axi status` run step rows expose additive `phase` and `review_file` columns for review steps; `axi status` gate output exposes additive `phase` and `review_file` fields beside the existing `step`, `status`, `summary`, `risk`, `findings`, and help fields; non-review steps omit both fields; existing raw step names, raw statuses, finding IDs, response commands, and finding row fields remain unchanged.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:127

### F-RT-001-review-file-handoff-2026-06-15-003

- Category: spec-fix
- Reasoning: Verification: FR-015 hides terminal-specific controls, while FR-032 says existing automation response commands remain compatible, so the spec leaves a real automation/terminal split. Evidence: `internal/cli/axi_drive.go:560`-`565` exposes `axi respond --action approve|fix|skip`, and `internal/ipc/protocol.go:104`-`118` shows IPC responses already carry action, finding IDs, and optional instructions. This is not `new-OQ` because the spec and existing command contract establish that automation stays supported. Rejected band-aid: returning a generic "use the file" error for all automation responses, because that would violate FR-032 compatibility. The durable fix is to define the compatibility matrix and make direct automation responses coexist with file processing through the same review-gate decision semantics.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-032**: Raw step names, raw step statuses, step order, existing automation response commands, and existing approval actions MUST remain compatible with current users and automation.
  After: - **FR-032**: Raw step names, raw step statuses, step order, existing automation response commands, and existing approval actions MUST remain compatible with current users and automation; during review and fix-review gates, terminal users MUST use the file handoff `process`/`cancel` path, while `axi respond --action approve|fix|skip`, yolo auto-resolution, and direct IPC `respond` calls MUST remain accepted and MUST map to the same review-gate decision transition, using explicit command payloads rather than parsing unvalidated Markdown prose.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:132

### F-RT-001-review-file-handoff-2026-06-15-004

- Category: spec-fix
- Reasoning: Verification: FR-009 says a valid file with all `accept` or `skip` responses approves the gate, and FR-032 preserves automation response commands, but the spec does not yet define a single transition behind both paths. Evidence: `internal/pipeline/executor.go:404`-`454` shows approve, skip, abort, and fix already converge in one executor response switch, so the durable contract should reuse that invariant. This is not `skipped` because the concern is not asking for a broader redesign; it protects the existing gate meaning described at `specs/001-review-file-handoff/spec.md:152`. Rejected band-aid: documenting automation as an intentional bypass, because that would make "passed review" mean different things by surface. The durable fix is to state that every surface reaches one review-gate decision transition and records an equivalent decision source for audit.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **Gate Semantics**: The feature preserves `git push no-mistakes` gate meaning by keeping the same review step, step order, raw statuses, approval actions, and pass/fail semantics. The handoff file changes how review decisions are presented and collected, not what a passed gate means.
  After: - **Gate Semantics**: The feature preserves `git push no-mistakes` gate meaning by keeping the same review step, step order, raw statuses, approval actions, and pass/fail semantics. The handoff file changes how review decisions are presented and collected, not what a passed gate means. Terminal `p process`, reattached sessions, AXI responses, yolo auto-resolution, and direct IPC responses MUST converge on one review-gate decision transition; the PR audit data MUST identify whether that transition came from a processed handoff file, an explicit automation response, or an existing no-handoff auto-fix path.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:152

### F-RT-001-review-file-handoff-2026-06-15-005

- Category: spec-fix
- Reasoning: Verification: FR-002 lists run, step, status, branch, processed timestamp, and processed action metadata, but it does not include a review-cycle identity or digest, so the stale-cycle premise holds. Evidence: FR-017 and FR-018 define one current file that is overwritten per normal review/fix-review cycle, while the key entity at `specs/001-review-file-handoff/spec.md:147` names "Review Cycle" as a real domain object. This is not `new-OQ` because the spec already chooses latest-cycle semantics; it just lacks an identifier for that invariant. Rejected band-aid: relying only on run/status/finding IDs, because normalized IDs can repeat across cycles. The durable fix is to include a generated review-cycle ID and finding-set digest in handoff metadata and validation.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-002**: The review handoff file MUST contain metadata identifying the run, review step, current review-gate status, branch, processed timestamp, and processed action; generated files MUST render `processed_action: pending` with an empty processed timestamp, then overwrite both after successful processing.
  After: - **FR-002**: The review handoff file MUST contain metadata identifying the run, review step, current review-gate status, branch, review cycle identifier, deterministic finding-set digest, processed timestamp, and processed action; generated files MUST render `processed_action: pending` with an empty processed timestamp, then overwrite both after successful processing.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:102

### F-RT-001-review-file-handoff-2026-06-15-006

- Category: skipped
- Reasoning: Verification: the finding says precedence is unstated, but FR-033 says "The review handoff behavior MUST NOT change automatic review auto-fix behavior configured for review" and SC-006 says "no new manual handoff required for automatic review fixes." Evidence: `internal/pipeline/executor.go:313`-`340` confirms the existing executor attempts configured auto-fix before entering approval, matching that spec invariant. This is not `spec-fix` because the cited spec text already states the long-term contract strongly enough. Rejected band-aid: adding another special-case precedence flag, because that would create a new configuration branch for behavior the spec already preserves. The durable resolution is to keep the existing auto-fix invariant and reject this accusation as already covered.
- Payload:
  Reason: Skipped because the premise is contradicted by `specs/001-review-file-handoff/spec.md:133` ("MUST NOT change automatic review auto-fix behavior") and `specs/001-review-file-handoff/spec.md:167` ("no new manual handoff required for automatic review fixes").
- Status: skipped
- Reason: Skipped because the premise is contradicted by `specs/001-review-file-handoff/spec.md:133` ("MUST NOT change automatic review auto-fix behavior") and `specs/001-review-file-handoff/spec.md:167` ("no new manual handoff required for automatic review fixes").
- Applied-at: 2026-06-15T21:31:37+07:00

### F-RT-001-review-file-handoff-2026-06-15-007

- Category: spec-fix
- Reasoning: Verification: FR-005 says "prose outside those blocks MUST NOT affect processing," while FR-008 says empty or comment-only `solution` falls back to recommendation option 1, so the trust-boundary ambiguity is real. Evidence: the existing finding model has an internal `SuggestedFix` field at `internal/types/findings.go:22`-`34`, which gives the implementation a source other than editable Markdown prose. This is not `new-OQ` because the spec already establishes response blocks as the only parsed user source. Rejected band-aid: copying option 1 into a hidden editable comment in the Markdown file, because that still trusts file text. The durable fix is to say fallback option 1 is resolved from the active/latest finding model or validated metadata, never from human-readable prose.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-008**: When processing a valid file, any response block marked `fix` MUST send only that finding for remediation, along with any non-empty `solution` text as the user's instruction for that finding; if the `solution` is empty or comment-only, the system MUST use recommendation option 1 as the fix instruction.
  After: - **FR-008**: When processing a valid file, any response block marked `fix` MUST send only that finding for remediation, along with any non-empty `solution` text as the user's instruction for that finding; if the `solution` is empty or comment-only, the system MUST use recommendation option 1 from the active latest finding model or validated handoff metadata as the fix instruction, and MUST NOT read fallback instructions from editable human-readable Markdown prose outside the response block.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:108

### F-RT-001-review-file-handoff-2026-06-15-008

- Category: spec-fix
- Reasoning: Verification: FR-039 only says generated files are written "inside the project checkout or isolated pipeline work area," so the symlink and canonical-boundary premise holds. Evidence: the existing project already uses symlink-aware boundary checks for artifact paths at `internal/pipeline/steps/prsummary.go:644`-`669`, showing the local durable pattern. This is not `out-of-scope` because path placement is central to FR-021 through FR-023 and FR-039. Rejected band-aid: checking that the raw string starts with the checkout path, because symlinks and `..` can defeat string-prefix checks. The durable fix is canonical path resolution, regular-file anchor checks, and final boundary validation immediately before writing or copying.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-039**: Generated review handoff files MUST be written only inside the project checkout or isolated pipeline work area, use normal repository file permissions, and perform no additional redaction in the first version.
  After: - **FR-039**: Generated review handoff files MUST be written only inside the project checkout or isolated pipeline work area, use normal repository file permissions, and perform no additional redaction in the first version; anchor files and destination paths MUST be resolved with symlink-aware canonicalization, anchor candidates MUST be regular files, and the final write or copy target MUST be revalidated as inside the intended checkout or isolated work-area root immediately before filesystem mutation.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:139

### F-RT-001-review-file-handoff-2026-06-15-009

- Category: spec-fix
- Reasoning: Verification: FR-011 validates expected metadata and latest finding IDs, but it does not require processed-state freshness or cycle/digest matching, so the replay premise holds. Evidence: FR-002 now needs a review cycle identifier and digest, and `internal/db/round.go:23`-`29` shows the codebase already treats selected finding IDs and selection source as round-level state. This is not `new-OQ` because latest-active-review semantics are already specified by FR-017 and FR-018. Rejected band-aid: accepting any file whose run/status/finding IDs match, because that misses same-ID stale cycles. The durable fix is to validate cycle/digest and require pending processed metadata before processing.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-011**: Processing MUST validate that the file exists, is readable, has the expected metadata, matches the active run and review status, includes exactly one response keyed by every latest normalized `Finding.ID`, includes no unknown finding IDs, uses only supported actions, and has parseable solution fields.
  After: - **FR-011**: Processing MUST validate that the file exists, is readable, has the expected metadata, matches the active run, review step, review status, review cycle identifier, and deterministic finding-set digest, includes exactly one response keyed by every latest normalized `Finding.ID`, includes no unknown finding IDs, uses only supported actions, has parseable solution fields, and still shows `processed_action: pending` with an empty processed timestamp before any approval or fix response is sent.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:111

### F-RT-001-review-file-handoff-2026-06-15-010

- Category: spec-fix
- Reasoning: Verification: FR-018 says new review results overwrite the current file and "MUST NOT preserve old answer blocks," while User Story 4 says the PR reviewer should see the latest file that shaped approval or fixes; the gap is real. Evidence: `internal/db/round.go:23`-`32` already persists selected finding IDs, selection source, and fix summaries at round level, and `internal/pipeline/steps/prsummary.go:1034`-`1090` builds PR narratives from rounds. This is not `out-of-scope` because the audit copy is the core P4 story at `specs/001-review-file-handoff/spec.md:73`-`84`. Rejected band-aid: preserving every old answer block in the active file, because FR-018 explicitly rejects that. The durable fix is a compact resolved-decision summary in the latest/final file sourced from durable round data.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-019**: When a fix-review result has no remaining findings, the handoff file MUST show a final state including metadata, any available applied-fix summary, and a clear "no remaining review findings" message.
  After: - **FR-019**: When a fix-review result has no remaining findings, the handoff file MUST show a final state including metadata, any available applied-fix summary, a clear "no remaining review findings" message, and a compact resolved-decision summary for the prior cycle that lists prior finding IDs, action taken, user-authored solution or default recommendation used, processed action, processed timestamp, and whether the decision came from the file handoff, automation response, or no-handoff auto-fix path.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:119

### F-RT-001-review-file-handoff-2026-06-15-011

- Category: spec-fix
- Reasoning: Verification: FR-037 says anchor use must not cause the anchor file to be committed, but FR-021 allows staged, modified, or untracked `plan.md` or `tasks.md` anchors, so the accidental commit premise holds. Evidence: `internal/pipeline/steps/push.go:38`-`44` currently commits with `git add -A`, which would stage unrelated uncommitted paths unless the feature constrains commit selection. This is not `skipped` because the finding points at a real conflict with FR-037's outcome. Rejected band-aid: warning users not to edit unrelated anchors, because the system chooses anchors automatically. The durable fix is to make anchor use placement-only and require an explicit publishable artifact allowlist.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-037**: Using `plan.md` or `tasks.md` as a placement anchor MUST NOT by itself cause that anchor file to be committed.
  After: - **FR-037**: Using `plan.md` or `tasks.md` as a placement anchor MUST NOT by itself cause that anchor file, neighboring files, or unrelated working-tree changes to be committed; anchor use is placement-only, and PR preparation MUST stage or copy only intentional pipeline outputs plus the normalized review-file relative path through an explicit publishable-artifact allowlist.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:137

### F-RT-001-review-file-handoff-2026-06-15-012

- Category: skipped
- Reasoning: Verification: the finding claims the review-file-only case is not required, but FR-036 says "If the review handoff file is the only remaining change to publish, the system MUST still create the commit so the PR includes the audit file." Evidence: FR-035 also requires the latest review file to be copied into the isolated work area and included in the commit. This is not `spec-fix` because the desired long-term outcome is already expressed as a MUST-level requirement and testable by User Story 4's independent test. Rejected band-aid: adding implementation-order prose about a no-op commit branch, because that overfits the spec to one code path. The durable resolution is to keep the existing outcome contract and let planning/tests enforce the builder details.
- Payload:
  Reason: Skipped because `specs/001-review-file-handoff/spec.md:135` says the latest review file "MUST be copied into the isolated pipeline work area at the same relative path and included in the commit," and `specs/001-review-file-handoff/spec.md:136` says the system "MUST still create the commit" when the review file is the only remaining change.
- Status: skipped
- Reason: Skipped because `specs/001-review-file-handoff/spec.md:135` says the latest review file "MUST be copied into the isolated pipeline work area at the same relative path and included in the commit," and `specs/001-review-file-handoff/spec.md:136` says the system "MUST still create the commit" when the review file is the only remaining change.
- Applied-at: 2026-06-15T21:31:37+07:00

### F-RT-001-review-file-handoff-2026-06-15-013

- Category: skipped
- Reasoning: Verification: the finding is true that generated blocks can default to `fix`, but FR-007 intentionally says automatically fixable findings default to `fix` and FR-008 says a valid `fix` block sends remediation. Evidence: FR-014 keeps the terminal compact, so the intended user authority moment is pressing `p process` after reviewing the file, not editing every generated default. This is not `spec-fix` because requiring a separate confirmation marker conflicts with the verified default-action requirement. Rejected band-aid: adding a generated `confirm_fix: true` field, because it adds another parser contract without changing the meaningful action. The durable resolution is to keep the default mapping and rely on the review file plus `process` action as the explicit workflow contract.
- Payload:
  Reason: Skipped because the recommended fix conflicts with `specs/001-review-file-handoff/spec.md:107` ("automatically fixable findings default to `fix`") and `specs/001-review-file-handoff/spec.md:108` ("any response block marked `fix` MUST send only that finding for remediation").
- Status: skipped
- Reason: Skipped because the recommended fix conflicts with `specs/001-review-file-handoff/spec.md:107` ("automatically fixable findings default to `fix`") and `specs/001-review-file-handoff/spec.md:108` ("any response block marked `fix` MUST send only that finding for remediation").
- Applied-at: 2026-06-15T21:31:37+07:00

### F-RT-001-review-file-handoff-2026-06-15-014

- Category: spec-fix
- Reasoning: Verification: FR-010 requires `accept` and `skip` to remain distinct for audit readability, while FR-018 overwrites prior answer blocks and FR-035 copies only the latest handoff file, so the erasure premise holds. Evidence: the local DB already has durable round selection fields at `internal/db/round.go:23`-`29`, which can preserve decisions without keeping historical files. This is not `new-OQ` because the business decision is already made: `accept` and `skip` are distinct in the audit surface. Rejected band-aid: keeping all prior answer blocks in the current handoff file, because that violates FR-018 and makes the active file harder to process. The durable fix is to carry processed per-finding decisions into a resolved-decision summary that survives regeneration and PR copying.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-010**: `accept` and `skip` MUST remain distinct in the review file for audit readability, even though both mean "do not fix this finding" in the first version.
  After: - **FR-010**: `accept` and `skip` MUST remain distinct in the review file for audit readability, even though both mean "do not fix this finding" in the first version; processed per-finding decisions MUST preserve the distinct `accept` versus `skip` action in the latest handoff file's resolved-decision summary and in the PR audit copy, even after later review or fix-review regeneration overwrites old answer blocks.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:110

### F-RT-001-review-file-handoff-2026-06-15-015

- Category: spec-fix
- Reasoning: Verification: FR-017 and FR-018 define one current file and overwrite behavior, while FR-026 requires reattach and `axi status` to see the file even if the live event was missed; the ordering gap is real. Evidence: `internal/pipeline/executor.go:376`-`380` currently updates step status and emits events around approval gates, so the review-file write and persisted reference must happen before those surfaces are exposed. This is not `new-OQ` because the desired invariant is mechanical: every surface sees the same committed review cycle. Rejected band-aid: refreshing the path on every render by scanning the filesystem, because that creates races and drift. The durable fix is an atomic transition order for each review cycle.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-017**: The system MUST use one current review handoff file per run across normal review and fix-review cycles.
  After: - **FR-017**: The system MUST use one current review handoff file per run across normal review and fix-review cycles; each review-cycle transition MUST generate or atomically overwrite the file, persist the current review-file reference plus phase inputs, and only then emit live events or expose reattached/`axi status` state, with the prior committed file reference retained if any step in that sequence fails.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:117

### F-RT-001-review-file-handoff-2026-06-15-016

- Category: spec-fix
- Reasoning: Verification: FR-002 says processed metadata is overwritten after successful processing, but it does not define commit points relative to dispatch and event/state exposure, so the transaction premise holds. Evidence: `internal/pipeline/executor.go:404`-`452` sends approve/fix state transitions after a response, and FR-026 requires automation surfaces to agree with the file state. This is not `new-OQ` because no product judgment is missing; it is an ordering contract. Rejected band-aid: updating the file after the daemon accepts the response without guarding partial failure, because observers could see a passed gate with stale pending metadata. The durable fix is a transaction rule for validation, metadata write, response dispatch, and event/state emission.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-012**: If processing validation fails, the system MUST block the action, keep the compact review gate open with only process/cancel actions, show one concise terminal error with the review file path, and preserve the user's file contents.
  After: - **FR-012**: If processing validation fails, the system MUST block the action, keep the compact review gate open with only process/cancel actions, show one concise terminal error with the review file path, and preserve the user's file contents; successful processing MUST be committed as a transaction whose ordered commit points are validation, atomic processed-metadata update in the handoff file, response dispatch, and then live event/reattach/`axi status` state exposure, and any partial failure MUST leave the file and automation state on the same last committed processed state.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:112

### F-RT-001-review-file-handoff-2026-06-15-017

- Category: spec-fix
- Reasoning: Verification: SC-005 requires PR commits to include the latest review file, but it does not require the audit file to prove the gate was processed, so the pending-metadata premise holds. Evidence: FR-002 defines `processed_action: pending`, and FR-035/FR-036 define PR inclusion; those contracts must compose before publishing. This is not `accepted-risk` because the consistency check is cheap and directly guards audit correctness in this milestone. Rejected band-aid: letting reviewers infer processing from the run being completed, because the file is the audit artifact. The durable fix is to block PR preparation when a required review handoff still says pending, except documented no-handoff auto-fix paths.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **SC-005**: 100% of PR branch commits created after a review handoff include the latest review file whenever the file exists and is inside the project checkout.
  After: - **SC-005**: 100% of PR branch commits created after a review handoff include the latest review file whenever the file exists and is inside the project checkout, and PR preparation MUST fail before commit if a review gate has passed but the latest required handoff file still has `processed_action: pending`, except for explicitly documented no-handoff automatic auto-fix paths.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:166

### F-RT-001-review-file-handoff-2026-06-15-018

- Category: spec-fix
- Reasoning: Verification: FR-008 falls back to recommendation option 1 for empty/comment-only solutions, but FR-004 only says recommendations present one or two options and does not require option 1 to be machine-detectable, so the premise holds. Evidence: the clarification at `specs/001-review-file-handoff/clarifications-applied-2026-06-15-205640.md:126`-`141` confirms option 1 is the chosen default behavior. This is not `new-OQ` because the human already chose fallback-to-option-1. Rejected band-aid: sending an empty fix instruction when option 1 cannot be parsed, because that changes remediation behavior silently. The durable fix is to require a parseable option 1 for any generated default-fix fallback and reject the file otherwise.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-004**: Recommendations MUST present one or two concrete options, with option 1 treated as the default fix choice when the user leaves a fix solution empty.
  After: - **FR-004**: Recommendations MUST present one or two concrete options, with option 1 treated as the default fix choice when the user leaves a fix solution empty; for every finding whose generated response can default to `fix`, option 1 MUST be machine-detectable and non-empty, and processing MUST reject an empty-solution `fix` response when no valid option 1 exists.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:104

### F-RT-001-review-file-handoff-2026-06-15-019

- Category: spec-fix
- Reasoning: Verification: FR-005 chooses line-oriented `action` and `solution` fields, but it does not specify duplicate handling, exact key case, unknown fields, or fence termination behavior, so the parser ambiguity premise holds. Evidence: the clarification answer at `specs/001-review-file-handoff/spec.md:12` fixes the block tag and two fields but not the grammar edge cases. This is not `new-OQ` because strict parsing is a local engineering contract, not a product decision. Rejected band-aid: accepting parser-dependent whitespace and duplicate variants, because different surfaces would process the same file differently. The durable fix is a strict grammar with exact field names and deterministic rejection rules.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-005**: User answers MUST be read only from fenced `no-mistakes-review-response` blocks keyed by the latest finding's normalized `Finding.ID` and containing line-oriented `action: fix|accept|skip` and `solution: <one-line text>` fields; prose outside those blocks MUST NOT affect processing.
  After: - **FR-005**: User answers MUST be read only from fenced `no-mistakes-review-response` blocks keyed by the latest finding's normalized `Finding.ID` and containing line-oriented `action: fix|accept|skip` and `solution: <one-line text>` fields; the parser MUST require exact lowercase field names, exactly one `action` line, exactly one `solution` line, one finding ID per block, deterministic whitespace trimming around field values, rejection of duplicate or unknown fields, rejection of nested response fences or multi-line continuations, and prose outside those blocks MUST NOT affect processing.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:105

### F-RT-001-review-file-handoff-2026-06-15-020

- Category: spec-fix
- Reasoning: Verification: FR-018 requires overwrite on each new review/fix-review result, while FR-012 promises to preserve user contents after validation failure, so the unprocessed-edit loss premise holds. Evidence: the assumption at `specs/001-review-file-handoff/spec.md:172` says historical review-cycle files are out of scope, which means any protection must be narrow and not become a full archive. This is not `accepted-risk` because conflict detection before overwrite is small and protects the core file-handoff trust model. Rejected band-aid: always keeping timestamped historical copies of every cycle as the main audit model, because that expands scope beyond one current file. The durable fix is atomic overwrite with pending-edit conflict detection and either backup or block behavior for user-authored pending changes.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-018**: When a new review or fix-review result is generated, the system MUST overwrite the current handoff file with the latest findings and MUST NOT preserve old answer blocks from a previous review result.
  After: - **FR-018**: When a new review or fix-review result is generated, the system MUST overwrite the current handoff file with the latest findings and MUST NOT preserve old answer blocks from a previous review result in the current file; before overwriting a pending file, the system MUST detect whether user-editable response content changed since generation and MUST either preserve the superseded pending file as a timestamped backup or block regeneration with a concise stale-edit diagnostic rather than silently discarding user-authored answers.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:118

### F-RT-001-review-file-handoff-2026-06-15-021

- Category: spec-fix
- Reasoning: Verification: FR-035 says the latest file is copied into the isolated work area at the same relative path, but it does not say that path is persisted at generation time or that anchor discovery must not run again, so the premise holds. Evidence: FR-022 can resolve anchors from committed changes under review, while FR-039 allows both checkout and isolated work area roots, so root ambiguity is real. This is not `new-OQ` because the feature already chooses the relative-path outcome. Rejected band-aid: re-running anchor discovery during push, because the isolated work area can have different changed files from the checkout. The durable fix is to persist a normalized repository-relative path at generation and copy exactly that path into the isolated work area.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-035**: When the pipeline prepares the PR branch commit, the latest review handoff file MUST be copied into the isolated pipeline work area at the same relative path and included in the commit.
  After: - **FR-035**: When the pipeline prepares the PR branch commit, the latest review handoff file MUST be copied into the isolated pipeline work area at the same normalized repository-relative path recorded when the file was generated and included in the commit; the push or PR preparation phase MUST NOT re-run anchor discovery in the isolated work area and MUST validate that the persisted relative path stays inside both the checkout root and isolated work-area root before copying.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:135

### F-RT-001-review-file-handoff-2026-06-15-022

- Category: spec-fix
- Reasoning: Verification: FR-012 requires one concise error with the file path, but for wrong-run/stale metadata it does not require the expected and actual values, so the user-recovery premise holds. Evidence: User Story 2 acceptance scenario 1 at `specs/001-review-file-handoff/spec.md:51` covers wrong run identifiers specifically. This is not `new-OQ` because the UX choice remains concise error handling; the fix only defines required contents for stale metadata errors. Rejected band-aid: dumping a full validation report into the terminal, because FR-012 deliberately keeps the gate compact. The durable fix is to include actual versus expected metadata and the current valid path or regeneration state inside the concise error.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: 1. **Given** a review file whose run identifier does not match the active run, **When** the developer presses `p process`, **Then** processing is blocked, the terminal shows a concise error and the file path, and the gate remains open.
  After: 1. **Given** a review file whose run identifier or review-cycle metadata does not match the active gate, **When** the developer presses `p process`, **Then** processing is blocked, the terminal shows a concise error containing the actual versus expected run/status/cycle values plus the current valid review file path or regeneration state, and the gate remains open.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:51

### F-RT-001-review-file-handoff-2026-06-15-023

- Category: spec-fix
- Reasoning: Verification: FR-013 says cancel aborts the active run, and the edge cases include interrupted, canceled, or superseded runs, but the spec does not bind cancel/process to the original gate identity, so the premise holds. Evidence: `internal/tui/commands.go:148`-`155` cancels by model run ID, while AXI resolves the active run by branch before responding at `internal/cli/axi_drive.go:602`-`610`; the spec should prevent stale gates from targeting a different active run. This is not `new-OQ` because the desired behavior follows from the existing user-abort path and stale-file validation model. Rejected band-aid: canceling whichever run is active on the branch, because a superseded terminal gate could abort the wrong run. The durable fix is to bind process and cancel actions to the gate's run/step identity and no-op with a stale-gate error when it no longer matches.
- Payload:
  Target: specs/001-review-file-handoff/spec.md
  Before: - **FR-013**: Pressing cancel from the review handoff gate MUST abort the active run through the existing user-abort path.
  After: - **FR-013**: Pressing cancel from the review handoff gate MUST abort the active run through the existing user-abort path only when the gate's original run and review step identity still match the currently active gate; process and cancel actions MUST no-op with a stale-gate error if that run is interrupted, canceled, superseded, or no longer awaiting the same review decision.
- Status: applied
- Applied-at: 2026-06-15T21:31:37+07:00
- Downstream-ref: specs/001-review-file-handoff/spec.md:113

## 4. Session Metadata

```yaml
session_id: RT-001-review-file-handoff-2026-06-15
target: specs/001-review-file-handoff/spec.md
feature_id: 001-review-file-handoff
date: "2026-06-15"
maintainer: Kevin Le
matched_triggers:
  - contracts
  - multi_party
  - immutability_audit
selected_lenses:
  - Automation Contract Adversary
  - Gate Semantics Adversary
  - Handoff File Integrity Adversary
  - PR Audit Trail Adversary
  - User Authority Adversary
selection_method: auto
supporting_context:
  - .specify/memory/constitution.md
  - specs/001-review-file-handoff/checklists/requirements.md
  - specs/001-review-file-handoff/clarifications-applied-2026-06-15-205640.md
finding_counts_by_lens:
  Automation Contract Adversary: 5
  Gate Semantics Adversary: 5
  Handoff File Integrity Adversary: 5
  PR Audit Trail Adversary: 4
  User Authority Adversary: 4
finding_counts_by_severity:
  CRITICAL: 0
  HIGH: 14
  MEDIUM: 9
  LOW: 0
dropped_findings_by_lens: {}
lens_failures: []
resolution_counts:
  spec-fix: 20
  new-OQ: 0
  accepted-risk: 0
  out-of-scope: 0
  skipped: 3
unresolved: 0
apply:
  applied_at: "2026-06-15T21:31:37+07:00"
  applied_by: Kevin Le
  resolutions:
    spec_fix: 20
    new_OQ: 0
    accepted_risk: 0
    out_of_scope: 0
    skipped: 3
  unresolved: 0
  allow_historical_edits: true
  historical_edits_applied:
    - "F-RT-001-review-file-handoff-2026-06-15-001: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-002: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-003: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-004: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-005: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-007: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-008: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-009: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-010: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-011: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-014: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-015: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-016: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-017: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-018: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-019: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-020: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-021: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-022: specs/001-review-file-handoff/spec.md"
    - "F-RT-001-review-file-handoff-2026-06-15-023: specs/001-review-file-handoff/spec.md"
warnings:
  - constitution does not yet declare red team trigger criteria; default trigger categories were used in bootstrap mode
  - red-team extension registry is corrupted locally; project-specific lens catalog was created at .specify/extensions/red-team/red-team-lenses.yml so this run could proceed
```
