# Red Team Findings — Review File Handoff

Status: ARCHIVED
**Applied:** 2026-06-16-104115

| Field | Value |
|---|---|
| **Session ID** | `RT-001-review-file-handoff-2026-06-16` |
| **Target spec** | `specs/001-review-file-handoff/spec.md` |
| **Date** | 2026-06-16 |
| **Maintainer** | Kevin Le |
| **Matched triggers** | `contracts`, `multi_party`, `immutability_audit`, `ai_llm` |
| **Lenses** | Trust-Boundary Adversary; Contract & Interface Faithfulness Adversary; Immutability & Audit-Trail Adversary; Agent/LLM Faithfulness Adversary |
| **Selection method** | auto (4 trigger-matched lenses; catalog extended with 3 domain lenses before this run) |
| **Supporting context** | `specs/001-review-file-handoff/clarifications-applied-2026-06-16-093514.md` |
| **Wall-clock** | dispatch 10:08:03 → all findings in 10:13:12 (+07); adversarial pass ~1.5 min, well under 30-min target |
| **Lens failures** | none (4/4 returned, 5 findings each) |
| **Dropped (per-lens bound)** | 0 (each lens returned ≤ finding_bound=5) |

---

## §1 Session Summary

*Draft auto-summary — maintainer to confirm/replace after resolution.*

20 findings: **8 CRITICAL, 7 HIGH, 5 MEDIUM, 0 LOW**. Four lenses converge on **three under-specified root gaps**, each surfaced independently from multiple angles (high confidence):

- **Cluster A — Deterministic review-result hash input domain undefined** (FR-005/FR-015). Raised by 3 lenses: `F-…-003` (recommendation text excluded), `F-…-006` (input domain/collision via reused IDs), `F-…-008` (content vs IDs). The hash is the sole stale-buffer guard yet its inputs are unspecified, so a same-cycle content change can pass validation and drive the fixer with the wrong instruction.
- **Cluster B — Automation-decision mirroring (FR-028) is under-specified.** Four CRITICAL/HIGH findings: atomicity vs gate decision (`F-…-005`), mirrored file must satisfy FR-015 (`F-…-004`), two-writer race/precedence (`F-…-007`), exact-executed-decision fidelity + user-added findings (`F-…-009`). The committed PR audit file can diverge from what the gate actually executed.
- **Cluster C — Finding identity & preservation across cycles** (FR-009/FR-030/FR-031). Order-based `review-N` IDs (confirmed in code at `findings.go:95`) are not stable per logical finding (`F-…-002`), duplicate IDs aren't rejected (`F-…-010`), overwrite loses prior-cycle audit decisions (`F-…-013`), and the FR-031 no-findings final state both conflicts with FR-015's ID checks (`F-…-012`) and has an undefined source for "preserved" decisions (`F-…-014`).

Standalone notables: prompt-injection of free-text `solution:` into the fixer (`F-…-001`, CRITICAL); forged `processed_action` trusted (`F-…-015`); all-defaults rubber-stamp approval (`F-…-019`); anchor path-escape (`F-…-020`).

---

## §2 Findings

Sorted by severity (CRITICAL → HIGH → MEDIUM → LOW); within band, by lens name (alphabetical), then adversary return order. `status` filled during §7 resolution.

| ID | Sev | Lens | Location | Finding | Suggested Resolution | Status |
|---|---|---|---|---|---|---|
| F-RT-001-review-file-handoff-2026-06-16-001 | CRITICAL | Agent/LLM Faithfulness | FR-012, FR-019 | Free-text user `solution:` is passed to the fixer agent as a per-finding instruction with only comment-line stripping (FR-012). The spec never constrains this text as data rather than executable instruction, so adversarial directives in `solution:` (e.g. "ignore the finding and instead delete X", or injected reviewer/system framing) reach the fixer's prompt verbatim, letting untrusted file content redirect the fixer beyond the named finding. | Add an FR requiring `solution:` to be delivered as clearly delimited, untrusted user-data scoped to exactly one finding ID, with explicit instruction-injection containment, and bound the fixer's authority to that finding. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-002 | CRITICAL | Agent/LLM Faithfulness | FR-009, FR-030 (order-based `review-N` IDs) | Finding IDs may be order-based `review-N` (`findings.go:95`) and FR-030 overwrites the file each cycle without preserving prior answers. Across cycles the same `review-N` ID can denote a different logical finding while the result-hash only blocks stale buffers, not identity drift. A regenerated same-logical finding is treated as new (prior fix instruction silently discarded), and a different finding can inherit the same ID slot, mis-attributing audit history. | Require finding IDs to be stable per logical finding across cycles (content/anchor-derived, not render-order), and specify whether surviving findings carry forward prior user answers or are deliberately reset. | skipped |
| F-RT-001-review-file-handoff-2026-06-16-003 | CRITICAL | Contract & Interface Faithfulness | FR-005, FR-012, FR-015 | The deterministic review-result hash has no defined input domain. FR-012 resolves an empty `solution:` to "recommendation option 1" at process time from the current gate. If the hash covers IDs/issue text but not the `Recommendation` option text, the gate can overwrite recommendation content within the same review result (same IDs, same hash), so an empty `fix` sends a different instruction than the file's author saw — yet validation passes. | Specify the hash domain explicitly: it MUST cover every byte the parser relies on for a decision — finding IDs, issue text, and full `Recommendation` option text — so any recommendation change invalidates stale buffers. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-004 | CRITICAL | Contract & Interface Faithfulness | FR-028 vs FR-015, FR-032 | FR-028 mirrors automation approve/fix/skip decisions into response blocks and processing metadata, and FR-032 makes that file the committed PR audit artifact. Nothing requires the mirrored file to satisfy FR-015 (valid actions, one block per finding, matching hash, size bounds) or carry the current hash. The automation path can advance the gate while writing a file that would FAIL `p process`, so the committed audit record can contradict the real gate outcome. | Require mirrored files to be written through the same writer and satisfy all FR-015 invariants (matching hash, one valid block per finding, bounds); the mirror MUST be a file that would re-validate. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-005 | CRITICAL | Immutability & Audit-Trail | FR-028 / §Edge Cases (interrupted/canceled/superseded) | Automation-decision mirroring updates the file's response blocks and processing metadata, but nothing requires this mirror write to be atomic with or transactionally coupled to the gate decision. If the gate decision is applied (approval recorded, fixer dispatched) and the run is then interrupted/canceled before the mirror lands — or the mirror fails — the committed PR audit file misrepresents or omits a real decision, with no FR mandating reconciliation or an inconsistency marker. | Require the mirrored write to succeed before the gate decision is final (write-then-act), or add an FR forcing the committed file to carry a "decision-pending/unreconciled" marker when mirror and gate state diverge. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-006 | CRITICAL | Immutability & Audit-Trail | FR-005, FR-015, Q4 | The deterministic hash is the sole defense against stale buffers, but the spec never defines its input domain or that it is recomputed from authoritative gate state at validation. If the implementation hashes only finding IDs/text, two cycles reusing order-based `review-N` IDs with identical finding text (`findings.go:95`) produce a colliding hash, so a stale prior-cycle file passes FR-015 and a wrong-but-matching answer set is accepted. Nothing forbids hashing over a non-unique field or trusting a file-supplied hash. | Specify the hash MUST be computed by the gate over a cycle-unique, collision-resistant input (ordered finding IDs + full content + per-cycle nonce/revision) and MUST be recomputed from live gate state at validation, never read back from the file. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-007 | CRITICAL | Trust-Boundary | FR-028 vs FR-018/FR-015 | Two actors hold authority over the same gate file: a human editing answers + `p process`, and automation that mirrors its decision by rewriting blocks/metadata (FR-028). The spec defines no precedence, locking, or re-validation when both touch the same run; FR-028 does not state that mirrored writes re-check the hash or invalidate a human's open buffer, and FR-018 lets `p process` succeed while the hash still matches. A last-writer-wins race lets automation silently flip a human's decision (or vice versa). | Add an FR defining single-writer authority per gate state with a precedence rule: any write re-validates the current hash, and mirrored automation writes bump the hash so a concurrently-open human buffer fails `p process`. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-008 | CRITICAL | Trust-Boundary | FR-005, FR-015 | The hash is the sole guard against stale edits yet its inputs are unspecified. FR-030 reuses finding IDs across cycles and Q4 notes order-based `review-N` IDs are reused. If the hash covers only run ID, status, and finding IDs, a re-review changing only finding text/recommendations (FR-007/FR-008) yields an identical hash, so a stale buffer's answers process against silently-changed findings, driving the fixer with wrong instructions. | Require the hash to cover the full normalized finding content (issue, context, recommendation options, severity, default action) for every latest finding, not just IDs/status, and state this in FR-005. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-009 | HIGH | Agent/LLM Faithfulness | FR-028 | FR-028 mirrors automation decisions (selected IDs, instructions, user-added findings) into response blocks/metadata, but does not require the mirror be derived from the identical decision the gate executed, nor define how user-added findings (no pre-generated block per FR-009) are represented. The PR audit file can record a decision set diverging from what the gate ran. | Specify mirroring writes the exact executed decision (same IDs, actions, instructions) atomically after the gate acts, and define how user-added findings appear in the file. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-010 | HIGH | Agent/LLM Faithfulness | FR-007, FR-009 | FR-007 says each latest finding appears once and FR-009 keys blocks by structured finding ID, but neither requires the latest finding set to have unique IDs. If two latest findings share an ID (collision in order-based or reused structured IDs), the writer emits a duplicate `no-mistakes-response` block or drops one, and the parser cannot map an answer to a single finding — a finding is silently dropped or its answer mis-applied. | Require generation to fail (like the missing-ID case in FR-009) when the latest finding set contains duplicate IDs, guaranteeing one-to-one finding-to-block mapping. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-011 | HIGH | Contract & Interface Faithfulness | FR-011 | FR-011 says blocks support "exactly these actions: fix, accept, skip" but gives no case-sensitivity, whitespace-trimming, or alias rule. `Fix`, `FIX`, or `fix ` is accepted by a lenient parser (advances the gate) and rejected by a strict one (blocks the gate). Two spec-conforming implementations produce opposite gate decisions for the identical file. | State exact matching semantics — e.g. case-insensitive, trimmed, no aliases — so the accept/reject boundary is deterministic across implementations. | skipped |
| F-RT-001-review-file-handoff-2026-06-16-012 | HIGH | Contract & Interface Faithfulness | FR-031 vs FR-015 | FR-031 overwrites the file with a final state preserving prior response blocks plus `No remaining review findings.`, and `p process` MUST approve directly. But FR-015 validates "every latest finding has one response block, no unknown finding IDs." With zero latest findings, every preserved prior block is now an "unknown finding ID", so a literal FR-015 reading rejects the very file FR-031 says must approve — a self-conflict that strands the gate or forces divergent implementations. | Carve the final no-findings state out of FR-015's finding-ID checks (validate metadata + hash only) or define a distinct `final` status/schema the parser branches on. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-013 | HIGH | Immutability & Audit-Trail | FR-030, FR-029, §Assumptions | FR-030 overwrites the single per-run file each cycle and forbids preserving old answer blocks, so the PR audit file (FR-032) records only the last cycle's decisions. A finding the user explicitly accepted/skipped in cycle 1 that doesn't recur in cycle 2 vanishes from the auditable record, contradicting the "auditable record of user decisions" claim (§Key Entities PR Audit File) and FR-033's audit intent. | Either narrow the claim (file audits only the final cycle) or require the overwrite to carry forward a compact immutable ledger of prior-cycle decided finding IDs + actions. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-014 | HIGH | Immutability & Audit-Trail | FR-031, FR-030 | FR-031's final state MUST "preserve prior finding decisions," but FR-030 already destroyed prior-cycle answer blocks on every intervening overwrite. The spec never says where the preserved decisions come from on the no-findings path, so an implementation reading them from the now-overwritten file preserves only the most recent cycle, producing a final audit file that looks complete but understates what was decided. | Specify the authoritative source for "prior finding decisions" in FR-031 (persisted decision store, not the overwritten file) and require it to span all cycles of the run. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-015 | HIGH | Trust-Boundary | FR-015, FR-017, FR-018 | `processed_at`/`processed_action` are user-editable front matter (FR-005) and the file is the PR audit record. FR-017/FR-018 forbid regeneration and only update metadata on success, but no requirement says the gate ignores a hand-forged `processed_action: approved`/`processed_at` written before `p process`. FR-015 only checks the fields are present/valid, so the gate or a downstream auditor may trust a self-asserted outcome. | Require the gate to derive the processed outcome solely from response-block actions, never trust an incoming `processed_action`/`processed_at`, and reject any non-`pending`/non-null processing values on a not-yet-processed file during FR-015. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-016 | MEDIUM | Agent/LLM Faithfulness | FR-015, FR-017, FR-030 | FR-015 validates "every latest finding has one block" and the hash matches, but FR-030 can overwrite the file between path announcement and `p process`, and FR-017 forbids regeneration during processing without requiring an atomic read-validate-apply snapshot. A regeneration landing mid-process could let the gate validate one file state then apply decisions against a different on-disk state. | Require validation and decision extraction to operate on a single consistent file snapshot and re-check the result-hash immediately before applying, rejecting if the file changed underneath. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-017 | MEDIUM | Contract & Interface Faithfulness | FR-027 | `review_phase_label` and `review_file_path` are declared nullable but the spec never states the run/gate states in which they MUST be non-null. Terminal/TUI always render a path (FR-021, FR-025), yet a conforming CLI/AXI client may emit `null` for the same `awaiting_approval` state, so the same run looks different across surfaces (violating US3 consistency) while every surface stays "spec-compliant". | Tie non-null population to the same status set that drives the phase labels (FR-025): both fields are non-null exactly when a review handoff file exists for the run. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-018 | MEDIUM | Immutability & Audit-Trail | FR-018, FR-017, FR-032 / §Edge Cases | No FR governs the audit record when the file is deleted, moved, or made unreadable after processing succeeds yet before PR commit (FR-032). FR-032 says "when a review handoff file exists"; if it no longer exists at push, the PR is created (FR-033 path) with no audit file and no recorded failure, leaving a committed PR whose review decisions are unreconstructable from the branch. | Add an FR requiring that if processing succeeded but the audit file is absent/unreadable at PR-commit time, the system regenerates the final processed-state file from persisted decisions or blocks the commit with an explicit error. | spec-fix |
| F-RT-001-review-file-handoff-2026-06-16-019 | MEDIUM | Trust-Boundary | FR-014, FR-020 | FR-014 defaults ask-user findings to `accept` and no-op to `skip`, and FR-020 approves with no extra confirmation when every block is accept/skip. A findings set of only ask-user/no-op therefore generates a file that, processed entirely unedited, silently approves the gate with zero human decision — degrading human review to a rubber-stamp keypress. No edit-detection or untouched-defaults guard exists. | Require `p process` to detect when all blocks remain at generated defaults and either warn before approving or distinguish "reviewed" from "defaulted", or require explicit acknowledgement when approval comes entirely from unedited defaults. | skipped |
| F-RT-001-review-file-handoff-2026-06-16-020 | MEDIUM | Trust-Boundary | FR-003, FR-034 | The committed audit file's location is chosen from a single changed `plan.md`/`task.md`/`tasks.md` anchor in uncommitted working-tree changes (agent-influenceable state), and FR-034 must suppress committing that anchor. The spec does not constrain the anchor to a path inside the repo checkout or guard against a directory escaping the project (symlink/`..`), so the audit file could be written/committed to an unexpected location, and FR-034 suppression depends on correctly identifying the adversary-chosen anchor. | Require anchor resolution to reject anchors outside the repo checkout and resolve symlinks, and require FR-034 suppression to key off the resolved anchor path rather than filename, with a test for an out-of-directory anchor. | spec-fix |

---

## §3 Resolutions Log

Filled during the §7 finding-by-finding walk. Each entry records: resolution category (spec-fix / new-OQ / accepted-risk / out-of-scope / skipped), reasoning, apply-critical fields when needed, and notes.

- **F-RT-001-review-file-handoff-2026-06-16-001** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:120` says "a non-empty `solution:` MUST be used as the user's per-finding instruction after ignoring comment-only lines"; the premise holds because the spec only strips comments and never marks the text as untrusted data. Evidence: `specs/001-review-file-handoff/spec.md:127` says fix decisions contain "only those finding IDs and their solution instructions", and `internal/pipeline/executor.go:434-436` passes merged findings into `sctx.PreviousFindings`, so solution text reaches the fix path as instructions. Category choice: spec-fix, not new-OQ, because the existing contract already says the text is per-finding instruction and the missing invariant is prompt-containment, not a product policy question. Long-term vs band-aid: rejected a band-aid that only strips more strings from `solution:`; the durable fix is to specify the trust boundary and authority scope of user text before it reaches any fixer prompt.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-012**: For `fix`, an empty `solution:` MUST mean use recommendation option 1; a non-empty `solution:` MUST be used as the user's per-finding instruction after ignoring comment-only lines.
  ```
  After:
  ```md
  - **FR-012**: For `fix`, an empty `solution:` MUST mean use recommendation option 1; a non-empty `solution:` MUST be used as the user's per-finding instruction after ignoring comment-only lines. Non-empty `solution:` text MUST be delivered to the fixer as clearly delimited, untrusted user data scoped to that one finding ID; the fixer prompt MUST treat it as data, not as system/developer instructions or authority to modify unrelated findings.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:120


- **F-RT-001-review-file-handoff-2026-06-16-002** — skipped
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:117` says response blocks use "the existing structured finding ID as canonical identity" and fail only when a latest finding "lacks an ID"; `specs/001-review-file-handoff/spec.md:138` says new review results "MUST NOT preserve old user-answer blocks from a previous cycle." Evidence: `internal/types/findings.go:95-102` confirms the current local normalizer can assign order-based IDs, but the feature spec deliberately does not redesign upstream finding identity and `specs/001-review-file-handoff/spec.md:184` says "previous review-cycle handoff history is out of scope." Category choice: skipped, not spec-fix, because requiring stable logical IDs and answer carry-forward conflicts with the verified v1 contract; not new-OQ because the spec already answers reset vs preserve for normal cycles. Long-term vs band-aid: rejected a band-aid that carries old answer blocks forward by matching `review-N`; the durable v1 shape is strong per-cycle hashing and final audit preservation, not pretending order IDs are logical identities.
  Reason: Skipped because the requested stable-logical-ID/carry-forward fix expands beyond this feature's v1 scope and conflicts with `specs/001-review-file-handoff/spec.md:138` ("MUST NOT preserve old user-answer blocks from a previous cycle") plus `specs/001-review-file-handoff/spec.md:184` ("previous review-cycle handoff history is out of scope").
  Status: skipped
  Reason: Skipped because the requested stable-logical-ID/carry-forward fix expands beyond this feature's v1 scope and conflicts with `specs/001-review-file-handoff/spec.md:138` ("MUST NOT preserve old user-answer blocks from a previous cycle") plus `specs/001-review-file-handoff/spec.md:184` ("previous review-cycle handoff history is out of scope").
  Applied-at: 2026-06-16T10:41:15+07:00


- **F-RT-001-review-file-handoff-2026-06-16-003** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:113` requires a "deterministic review-result hash" but does not define the bytes included; the premise holds because `specs/001-review-file-handoff/spec.md:120` makes empty `solution:` resolve to recommendation option 1. Evidence: `specs/001-review-file-handoff/spec.md:115-120` lists `Issue`, `Context`, `Recommendation`, and answer semantics as decision-driving content, so the hash must cover those fields. Category choice: spec-fix, not new-OQ, because the spec already establishes what data drives processing and only omits the invariant that the hash covers it. Long-term vs band-aid: rejected a band-aid that invalidates only timestamps; the durable fix is a deterministic content-domain contract for the hash.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-005**: The review handoff file MUST include YAML front matter with machine-readable metadata for the current run, review step, review status, branch, deterministic review-result hash, processed timestamp, and processed action; initially `processed_at` MUST be `null` and `processed_action` MUST be `pending`.
  ```
  After:
  ```md
  - **FR-005**: The review handoff file MUST include YAML front matter with machine-readable metadata for the current run, review step, review status, branch, deterministic review-result hash, processed timestamp, and processed action; initially `processed_at` MUST be `null` and `processed_action` MUST be `pending`. The deterministic review-result hash MUST cover the normalized current gate inputs that can affect processing: run ID, review step/status, review cycle revision, ordered canonical finding IDs, severity, issue text, context, full recommendation option text, default response action, and any applied fix summary used by a final no-findings state.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:113


- **F-RT-001-review-file-handoff-2026-06-16-004** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:136` says automation decisions are mirrored into the review file, and `specs/001-review-file-handoff/spec.md:123` defines the validation invariants only for `p process`; the premise holds because no text says automation mirrors must satisfy those same invariants. Evidence: `specs/001-review-file-handoff/spec.md:140` makes the final review file the PR commit artifact, so a malformed mirror would become the audit record. Category choice: spec-fix, not accepted-risk, because making the mirror validate through the same contract is narrow and required for audit correctness in this milestone. Long-term vs band-aid: rejected a band-aid that writes a separate automation note; the durable fix is one file contract that both human and automation paths satisfy.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-015**: On `p process`, the system MUST validate that the file is readable, YAML front matter metadata exists, run ID matches, step is review, status matches the current review gate state, the deterministic review-result hash matches the current gate, every latest finding has one response block, no unknown finding IDs are present, every action is valid, solution text is parseable, the file is at most 1 MiB, and each `solution:` value is at most 16 KiB.
  ```
  After:
  ```md
  - **FR-015**: On `p process`, the system MUST validate that the file is readable, YAML front matter metadata exists, run ID matches, step is review, status matches the current review gate state, the deterministic review-result hash matches the current gate, every latest finding has one response block, no unknown finding IDs are present, every action is valid, solution text is parseable, the file is at most 1 MiB, and each `solution:` value is at most 16 KiB. Any automation-mirrored review file MUST satisfy the same validation invariants before it is eligible to become the PR audit file.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:123


- **F-RT-001-review-file-handoff-2026-06-16-005** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:136` requires automation mirroring, but only says to update metadata and response blocks; the premise holds because it does not order that write relative to the gate decision. Evidence: `specs/001-review-file-handoff/spec.md:86-88` makes automation compatibility and PR auditability part of the same user story, so the mirror must be the decision record, not an afterthought. Category choice: spec-fix, not accepted-risk, because write-before-finalize is the smallest durable invariant and fits the existing automation contract in `internal/ipc/protocol.go:104-118`. Long-term vs band-aid: rejected a band-aid "unreconciled" marker as normal behavior; the long-term fix is that a decision is not final until its audit mirror exists.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-028**: Existing automation responses using approve, fix, skip, selected finding IDs, instructions, and user-added findings MUST continue to work without editing the handoff file, and the system MUST mirror those automation decisions into the review file by updating processing metadata and response blocks for PR auditability.
  ```
  After:
  ```md
  - **FR-028**: Existing automation responses using approve, fix, skip, selected finding IDs, instructions, and user-added findings MUST continue to work without editing the handoff file, and the system MUST mirror those automation decisions into the review file by updating processing metadata and response blocks for PR auditability. The mirror MUST be generated from the exact automation decision that the gate will execute, including selected IDs, instructions, and user-added findings, using the same writer/parser contract as hand-edited files. For automation decisions, the mirror write MUST succeed before the gate decision is finalized; if the mirror cannot be written, the gate MUST remain unprocessed and report an actionable error instead of recording a decision without its audit file.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:136


- **F-RT-001-review-file-handoff-2026-06-16-006** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:15` says the hash must match the current gate, but does not say it is recomputed from live gate state or cycle-unique; the premise holds. Evidence: `specs/001-review-file-handoff/clarifications-applied-2026-06-16-093514.md:86` explicitly cites order-based `review-N` IDs as the reason a result hash is needed, and `internal/types/findings.go:95-102` confirms that local fallback. Category choice: spec-fix, not new-OQ, because the local contract shows exactly why a live, cycle-scoped hash is required. Long-term vs band-aid: rejected trusting a file-supplied hash or timestamp; the durable fix is recomputation from authoritative gate state.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - Q: Should the metadata include a review-result revision or hash to reject stale editor buffers? → A: Include a deterministic review-result hash and require it to match the current gate.
  ```
  After:
  ```md
  - Q: Should the metadata include a review-result revision or hash to reject stale editor buffers? → A: Include a deterministic review-result hash, computed by the gate from live authoritative review state including a review-cycle revision, and require it to match the current gate; validation MUST recompute the hash from current gate state and MUST NOT trust the file-supplied hash as authority.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:15


- **F-RT-001-review-file-handoff-2026-06-16-007** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:126` says successful file processing updates only processing metadata, while `specs/001-review-file-handoff/spec.md:136` allows automation to mirror into the same file; the premise holds because precedence is not specified. Evidence: `internal/pipeline/executor.go:88-107` accepts an approval response asynchronously through the existing gate channel, so automation and human paths share the same gate authority. Category choice: spec-fix, not new-OQ, because the contract can be resolved by a single current-state validation rule without a product decision. Long-term vs band-aid: rejected a feature flag or lock file; the durable fix is atomic read/validate/update against the current gate state.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-018**: After successful file processing, the system MUST preserve the user's edited answers and update only processing metadata, changing `processed_at` from `null` and `processed_action` from `pending` to the successful processing values.
  ```
  After:
  ```md
  - **FR-018**: After successful file processing, the system MUST preserve the user's edited answers and update only processing metadata, changing `processed_at` from `null` and `processed_action` from `pending` to the successful processing values. File processing MUST read, validate, derive the decision, and update processing metadata against one current file snapshot; if the review hash, gate status, or processing metadata no longer matches current gate state before the update, processing MUST be rejected without advancing the gate.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:126


- **F-RT-001-review-file-handoff-2026-06-16-008** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:113` names a deterministic hash but does not define whether finding content is included; the premise holds because `specs/001-review-file-handoff/spec.md:115-120` makes finding text and recommendation text drive user decisions. Evidence: `specs/001-review-file-handoff/spec.md:138` overwrites the file on new review results, making content changes across cycles expected. Category choice: spec-fix, not accepted-risk, because covering normalized finding content is a small contract addition and is central to the stale-buffer invariant. Long-term vs band-aid: rejected hashing only IDs/status; the durable fix is content-based invalidation of changed review results.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-007**: Each latest finding MUST appear once in the handoff file with `Issue`, `Context`, `Recommendation`, and `User Answer` sections.
  ```
  After:
  ```md
  - **FR-007**: Each latest finding MUST appear once in the handoff file with `Issue`, `Context`, `Recommendation`, and `User Answer` sections. The normalized content of each latest finding, including issue text, context, recommendation options, severity, and generated default action, is part of the review-result hash domain.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:115


- **F-RT-001-review-file-handoff-2026-06-16-009** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:136` includes selected IDs, instructions, and user-added findings in the automation path, but does not define how the audit file proves those exact values were executed; the premise holds. Evidence: `internal/ipc/protocol.go:104-118` defines `RespondParams` with `FindingIDs`, `Instructions`, and `AddedFindings`, so the local automation contract has a concrete data shape to mirror. Category choice: spec-fix, not new-OQ, because no human policy call is needed; the mirror should render the exact local automation payload. Long-term vs band-aid: rejected summarizing automation as "approved" or "fixed"; the durable fix is exact decision fidelity in the audit file.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **Processed Review Decision**: The outcome created from a valid handoff file, either a fix decision with selected finding IDs and instructions or an approval decision.
  ```
  After:
  ```md
  - **Processed Review Decision**: The outcome created from a valid handoff file or an existing automation response, either a fix decision with the exact selected finding IDs, per-finding instructions, and user-added findings that were dispatched, or an approval decision. The PR audit file mirror MUST be derived from this executed decision payload.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:154


- **F-RT-001-review-file-handoff-2026-06-16-010** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:117` fails generation when a latest finding lacks an ID, but says nothing about duplicate IDs; the premise holds. Evidence: `internal/types/findings.go:95-102` can assign order-based fallback IDs, and `specs/001-review-file-handoff/spec.md:123` depends on one response block per latest finding, so duplicate latest IDs break the one-to-one mapping. Category choice: spec-fix, not new-OQ, because uniqueness is a structural parser/writer invariant already implied by the response-block contract. Long-term vs band-aid: rejected dropping duplicate findings or suffixing IDs at render time; the durable fix is to fail generation when canonical identity is not unique.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-009**: The user-answer area MUST contain one fenced `no-mistakes-response` block per latest finding ID, using the existing structured finding ID as canonical identity; file generation MUST fail if any latest finding lacks an ID.
  ```
  After:
  ```md
  - **FR-009**: The user-answer area MUST contain one fenced `no-mistakes-response` block per latest finding ID, using the existing structured finding ID as canonical identity; file generation MUST fail if any latest finding lacks an ID or if the latest finding set contains duplicate IDs.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:117


- **F-RT-001-review-file-handoff-2026-06-16-011** — skipped
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:119` says "exactly these actions: `fix`, `accept`, and `skip`"; those are lowercase literal tokens, not aliases. Evidence: the source plan repeats the same exact list at `plans/grill-me/review-file-handoff.md:151-154`, and validation says `action` is one of those values at `plans/grill-me/review-file-handoff.md:176-179`. Category choice: skipped, not spec-fix, because the finding's premise that `Fix`/`FIX` can be spec-conforming is wrong under the exact literal-token contract. Long-term vs band-aid: rejected a lenient alias rule as a compatibility band-aid; strict lower-case action literals are the simpler durable boundary.
  Reason: Skipped because the spec already disproves the ambiguity: `specs/001-review-file-handoff/spec.md:119` requires "exactly these actions: `fix`, `accept`, and `skip`."
  Status: skipped
  Reason: Skipped because the spec already disproves the ambiguity: `specs/001-review-file-handoff/spec.md:119` requires "exactly these actions: `fix`, `accept`, and `skip`."
  Applied-at: 2026-06-16T10:41:15+07:00


- **F-RT-001-review-file-handoff-2026-06-16-012** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:139` requires a final state preserving prior decisions and approving directly, while `specs/001-review-file-handoff/spec.md:123` rejects unknown finding IDs; the premise holds for a final state with zero latest findings. Evidence: `specs/001-review-file-handoff/clarifications-applied-2026-06-16-093514.md:193-194` confirms the intended final file should both preserve decisions and show `No remaining review findings.` Category choice: spec-fix, not new-OQ, because the product decision was already made and the parser branch needs to be specified. Long-term vs band-aid: rejected deleting prior blocks to satisfy FR-015; the durable fix is a distinct final-state validation rule that preserves audit content.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-031**: If fix review has no remaining findings, the system MUST overwrite the review file with a final state that preserves prior finding decisions, includes metadata, any available applied fix summary, and `No remaining review findings.`, and `p process` MUST approve directly.
  ```
  After:
  ```md
  - **FR-031**: If fix review has no remaining findings, the system MUST overwrite the review file with a distinct final state that preserves prior finding decisions, includes metadata, any available applied fix summary, and `No remaining review findings.`, and `p process` MUST approve directly. In this final state, preserved prior decisions are audit entries rather than latest findings, so FR-015's "every latest finding" and "no unknown finding IDs" checks apply only to the zero latest-finding set while metadata, hash, size, and parseability checks still apply.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:139


- **F-RT-001-review-file-handoff-2026-06-16-013** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:138` says normal cycles overwrite the file and do not preserve old answer blocks, while `specs/001-review-file-handoff/spec.md:155` defines the PR audit file as a record of user decisions; the premise holds that final-cycle-only content would undercut auditability. Evidence: `internal/db/round.go:23-28` already persists selected finding IDs per round, and `internal/pipeline/steps/round_history.go:63-99` shows prior decisions are locally available without adding history files. Category choice: spec-fix, not accepted-risk, because the structural source of truth already exists and the fix stays within the current feature. Long-term vs band-aid: rejected preserving full old handoff files; the durable fix is a compact final audit ledger derived from persisted rounds.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - Only one current review handoff file is kept per run; previous review-cycle handoff history is out of scope.
  ```
  After:
  ```md
  - Only one current review handoff file is kept per run; separate previous review-cycle handoff history files are out of scope. The final PR audit file must still include a compact ledger of prior-cycle decisions derived from persisted review round data when those decisions would otherwise be overwritten from the current handoff file.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:184


- **F-RT-001-review-file-handoff-2026-06-16-014** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:139` says the final state preserves prior finding decisions, but does not name the source after `specs/001-review-file-handoff/spec.md:138` overwrites old answer blocks; the premise holds. Evidence: `internal/db/round.go:16-33` stores findings, user findings, selected IDs, selection source, and fix summaries across rounds, which is the narrow local contract needed for preservation. Category choice: spec-fix, not new-OQ, because the repository already has a persistence contract that answers where prior decisions come from. Long-term vs band-aid: rejected reading from the overwritten Markdown file; the durable fix is to declare persisted round data authoritative.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - Existing data persistence can continue to store review findings, selected IDs, user instructions, fix summaries, and the deterministic review-result hash without requiring a new schema for the review file path.
  ```
  After:
  ```md
  - Existing data persistence can continue to store review findings, selected IDs, user instructions, fix summaries, and the deterministic review-result hash without requiring a new schema for the review file path. Persisted review round data is the authoritative source for prior finding decisions in the final no-findings state and must span all review/fix-review cycles in the run.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:186


- **F-RT-001-review-file-handoff-2026-06-16-015** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:14` says initial metadata is `processed_at: null` and `processed_action: pending`, but validation in `specs/001-review-file-handoff/spec.md:123` only says metadata is valid; the premise holds because forged non-pending values are not rejected. Evidence: `specs/001-review-file-handoff/spec.md:126` says successful processing changes those fields, proving non-pending values are outputs, not inputs. Category choice: spec-fix, not new-OQ, because this is a lifecycle invariant already implied by the metadata contract. Long-term vs band-aid: rejected hiding processed fields from the file; the durable fix is to reject self-asserted processed state and derive outcomes from parsed decisions.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - Q: What initial processing metadata should be written before the user processes the file? → A: Write `processed_at: null` and `processed_action: pending` initially.
  ```
  After:
  ```md
  - Q: What initial processing metadata should be written before the user processes the file? → A: Write `processed_at: null` and `processed_action: pending` initially; before processing, validation must reject any incoming file whose processing metadata is already non-null or non-`pending`, and the gate must derive processed outcomes only from valid response blocks or the executed automation decision.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:14


- **F-RT-001-review-file-handoff-2026-06-16-016** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:125` forbids regenerating during `p process`, but does not require the read/validate/apply sequence to operate on one snapshot; the premise holds. Evidence: `specs/001-review-file-handoff/spec.md:138` explicitly allows later review results to overwrite the same path, so concurrent file changes are part of the feature's normal lifecycle. Category choice: spec-fix, not accepted-risk, because snapshot validation is narrow and required to make FR-015 meaningful. Long-term vs band-aid: rejected a retry loop that silently re-reads changed content; the durable fix is to reject when the processed bytes are not the validated bytes.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-017**: The system MUST NOT regenerate the review handoff file or overwrite findings, recommendations, or user answers during `p process`.
  ```
  After:
  ```md
  - **FR-017**: The system MUST NOT regenerate the review handoff file or overwrite findings, recommendations, or user answers during `p process`. Validation, decision extraction, and processing metadata updates MUST operate on a single consistent snapshot of the file bytes; if the file changes before the update is committed, processing MUST reject the file and keep the gate open.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:125


- **F-RT-001-review-file-handoff-2026-06-16-017** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:135` exposes nullable `review_phase_label` and `review_file_path`, but does not define when they are populated; the premise holds for CLI/AXI consistency. Evidence: `specs/001-review-file-handoff/spec.md:133-134` defines the phase-label statuses, and `plans/grill-me/review-file-handoff.md:280-288` shows the intended gate output carries both raw status and review file. Category choice: spec-fix, not new-OQ, because the phase/status mapping already exists. Long-term vs band-aid: rejected surface-specific fallback text; the durable fix is a structured non-null contract tied to handoff existence and phase status.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-027**: CLI/AXI run and gate output MUST preserve raw status values while also exposing nullable structured fields such as `review_phase_label` and `review_file_path` when applicable.
  ```
  After:
  ```md
  - **FR-027**: CLI/AXI run and gate output MUST preserve raw status values while also exposing nullable structured fields such as `review_phase_label` and `review_file_path` when applicable. `review_phase_label` MUST be non-null for the review statuses that have labels in FR-025, and `review_file_path` MUST be non-null whenever a review handoff file exists or is recoverable for the run; otherwise each field MUST be null.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:135


- **F-RT-001-review-file-handoff-2026-06-16-018** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:140` says include the final handoff file in the PR commit, but does not say what happens if it is absent or unreadable at push; the premise holds. Evidence: `specs/001-review-file-handoff/spec.md:186` says existing persistence stores findings, selected IDs, user instructions, fix summaries, and hash, so regeneration can use local state without a new schema. Category choice: spec-fix, not accepted-risk, because an audit file missing at commit time directly violates the feature's PR audit outcome and has a bounded structural fix. Long-term vs band-aid: rejected committing an empty placeholder; the durable fix is regenerate from persisted decisions or block the PR commit.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-032**: The system MUST include the final review handoff file in the PR branch commit when the pipeline reaches push, preserving the same relative path.
  ```
  After:
  ```md
  - **FR-032**: The system MUST include the final review handoff file in the PR branch commit when the pipeline reaches push, preserving the same relative path. If processing succeeded but the final review handoff file is absent or unreadable when push prepares the PR branch commit, the system MUST regenerate the final processed-state file from persisted review decisions and fix summaries, or block the commit with an explicit audit-file error if regeneration is impossible.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:140


- **F-RT-001-review-file-handoff-2026-06-16-019** — skipped
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:122` intentionally defaults ask-user findings to `accept` and no-op findings to `skip`, and `specs/001-review-file-handoff/spec.md:128` says all accept/skip blocks approve "without requiring another confirmation." Evidence: the source plan repeats "No extra confirmation is needed when all issues are accepted/skipped" at `plans/grill-me/review-file-handoff.md:214`, so the suggested extra warning conflicts with the verified feature intent. Category choice: skipped, not spec-fix, because the finding asks to add a confirmation/acknowledgement gate the spec explicitly excludes; not new-OQ because the user already answered this through the source plan and spec. Long-term vs band-aid: rejected edit-detection as a band-aid that creates a second hidden confirmation model; the durable contract is that pressing `p process` is the explicit review action for generated defaults.
  Reason: Skipped because the suggested acknowledgement conflicts with `specs/001-review-file-handoff/spec.md:128` ("without requiring another confirmation") and `plans/grill-me/review-file-handoff.md:214` ("No extra confirmation is needed when all issues are accepted/skipped").
  Status: skipped
  Reason: Skipped because the suggested acknowledgement conflicts with `specs/001-review-file-handoff/spec.md:128` ("without requiring another confirmation") and `plans/grill-me/review-file-handoff.md:214` ("No extra confirmation is needed when all issues are accepted/skipped").
  Applied-at: 2026-06-16T10:41:15+07:00


- **F-RT-001-review-file-handoff-2026-06-16-020** — spec-fix
  Reasoning:
  Verification: `specs/001-review-file-handoff/spec.md:111` chooses anchors by changed `plan.md`, `task.md`, or `tasks.md` files, but does not state repo-bound path validation; the premise holds. Evidence: `internal/pipeline/steps/evidence.go:81-98` has an existing local `safeRepoSubdir` pattern rejecting absolute paths, traversal, and `.git`, which is the narrowest relevant contract for repo-contained writes. Category choice: spec-fix, not new-OQ, because local code already establishes the safety shape for repo-relative write paths. Long-term vs band-aid: rejected suppressing anchor commits by basename only; the durable fix is resolving and validating anchors inside the checkout before placement and suppression.
  Target: specs/001-review-file-handoff/spec.md
  Before:
  ```md
  - **FR-003**: The system MUST place the review handoff file next to a single changed `plan.md`, `task.md`, or `tasks.md` anchor when exactly one total eligible anchor is present in uncommitted changes, otherwise next to a single such anchor from the latest reviewed commit, otherwise under `.no-mistakes/issues/<branch-slug>/`.
  ```
  After:
  ```md
  - **FR-003**: The system MUST place the review handoff file next to a single changed `plan.md`, `task.md`, or `tasks.md` anchor when exactly one total eligible anchor is present in uncommitted changes, otherwise next to a single such anchor from the latest reviewed commit, otherwise under `.no-mistakes/issues/<branch-slug>/`. Anchor resolution MUST treat changed paths as repo-relative, clean and resolve the candidate directory, reject absolute paths, traversal, `.git`, symlink escapes, or any path outside the project checkout, and FR-034 anchor suppression MUST use the resolved anchor path rather than basename alone.
  ```
  Status: applied
  Applied-at: 2026-06-16T10:41:15+07:00
  Downstream-ref: specs/001-review-file-handoff/spec.md:111


---

## §4 Validation Decision

Not applicable — this is **not a designated dogfood session** (no dogfood target declared in `.specify/memory/constitution.md` or extension-adoption docs). Section intentionally omitted.

---

## §5 Session Metadata

```yaml
session_id: RT-001-review-file-handoff-2026-06-16
target_spec: specs/001-review-file-handoff/spec.md
date: 2026-06-16
maintainer: Kevin Le
matched_triggers: [contracts, multi_party, immutability_audit, ai_llm]
selection_method: auto
catalog_extended: true   # 3 domain lenses added to red-team-lenses.yml before this run
lenses:
  - name: Trust-Boundary Adversary
    findings: 5
    status: ok
  - name: Contract & Interface Faithfulness Adversary
    findings: 5
    status: ok
  - name: Immutability & Audit-Trail Adversary
    findings: 5
    status: ok
  - name: Agent/LLM Faithfulness Adversary
    findings: 5
    status: ok
lens_failures: []
findings_total: 20
severity_counts:
  CRITICAL: 8
  HIGH: 7
  MEDIUM: 5
  LOW: 0
dropped_by_finding_bound: 0
wall_clock:
  dispatch: 2026-06-16T10:08:03+07:00
  findings_complete: 2026-06-16T10:13:12+07:00
resolution_counts:
  spec-fix: 17
  new-OQ: 0
  accepted-risk: 0
  out-of-scope: 0
  skipped: 3
unresolved: 0
apply:
  applied_at: 2026-06-16T10:41:15+07:00
  applied_by: Kevin Le
  resolutions:
    spec_fix: 17
    new_OQ: 0
    accepted_risk: 0
    out_of_scope: 0
    skipped: 3
  unresolved: 0
  allow_historical_edits: true
  historical_edits_applied: 
    - F-RT-001-review-file-handoff-2026-06-16-001:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-003:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-004:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-005:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-006:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-007:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-008:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-009:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-010:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-012:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-013:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-014:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-015:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-016:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-017:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-018:specs/001-review-file-handoff/spec.md
    - F-RT-001-review-file-handoff-2026-06-16-020:specs/001-review-file-handoff/spec.md

```
