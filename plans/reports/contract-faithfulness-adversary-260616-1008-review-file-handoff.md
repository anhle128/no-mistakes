# Red-Team: Contract & Interface Faithfulness — Review File Handoff

**Spec:** specs/001-review-file-handoff/spec.md
**Lens:** wire-contract faithfulness (round-trip loss, schema drift, file-vs-automation divergence)
**Date:** 2026-06-16

## Findings (ranked)

### CRITICAL — FR-005 / FR-012 / FR-015: review-result hash coverage undefined
The "deterministic review-result hash" is required to be written (FR-005) and matched (FR-015)
but its input domain is never specified. FR-012 resolves an empty `solution:` to "recommendation
option 1" *at process time* from the current gate. If the hash covers finding IDs/issue text but
NOT recommendation option text, the gate can overwrite recommendation content within the same
review result (same IDs, same hash) and an empty `fix` now means a different instruction than the
file's author saw — yet validation passes. Wrong fix instruction on a "valid" file.
**Fix:** Specify the hash domain explicitly: it MUST cover every byte the parser relies on for a
decision — finding IDs, issue text, AND full `Recommendation` option text — so any recommendation
change invalidates stale buffers.

### CRITICAL — FR-028 vs FR-015: mirrored automation file has no validation/hash contract
FR-028 mirrors automation approve/fix/skip decisions into the file's response blocks + processing
metadata, and FR-032/FR-137 make that file the PR audit artifact. But nothing requires the mirrored
file to satisfy FR-015 (valid actions, one block per finding, matching hash, size bounds) or to
carry the current hash. The automation path can advance the gate while writing a file that would
FAIL `p process`, so the committed audit record can contradict the real gate outcome — two
outcomes for one logical decision.
**Fix:** Require mirrored files to be written through the same writer and to satisfy FR-015
invariants (matching hash, one valid block per finding); the mirror MUST be a file that would
re-validate.

### HIGH — FR-011: no normalization rule for action tokens (case/whitespace)
FR-011 says blocks support "exactly these actions: fix, accept, skip" but gives no case-sensitivity,
trimming, or alias rule. A user/editor writing `Fix`, `FIX`, or `fix ` is accepted by a lenient
parser (advances gate) and rejected by a strict one (blocks gate). Two conforming implementations
produce opposite gate decisions for the identical file.
**Fix:** State exact matching semantics — e.g. case-insensitive, trimmed, no aliases — so the
accept/reject boundary is deterministic across implementations.

### HIGH — FR-031: no-findings final file vs FR-015 unknown-ID rejection
FR-031 overwrites the file with a final state that PRESERVES prior finding response blocks plus
`No remaining review findings.`, and `p process` MUST approve directly. But FR-015 validation
checks "every latest finding has one response block, no unknown finding IDs." With zero latest
findings, every preserved prior block is now an "unknown finding ID" — a literal FR-015 reading
rejects the very file FR-031 says must approve. Contract self-conflict → gate stuck or
implementation-divergent.
**Fix:** Carve out the final no-findings state from FR-015's finding-ID checks (validate metadata +
hash only) or define a distinct `final` schema/status the parser branches on.

### MEDIUM — FR-027: nullable AXI fields never specified as to when non-null → surface divergence
`review_phase_label` / `review_file_path` are nullable (FR-027) but the spec never states the
exact run/gate states in which they MUST be non-null. Terminal/TUI always render a path (FR-021,
FR-025); a conforming AXI client may emit `null` for the same `awaiting_approval` state. The same
run then looks different across surfaces (US3 goal violated) even though all are "spec-compliant."
**Fix:** Tie non-null population to the same status set that drives the phase labels (FR-025): both
fields non-null exactly when a review handoff file exists for the run.

## Unresolved questions
- Is the front-matter schema versioned? FR-036 forbids new schema but a file edited by an
  older/unknown tool dropping a key (e.g. the hash) — is a missing key rejected (FR-015 "metadata
  exists") or treated as mismatch? Reject-vs-mismatch path is unspecified.
- FR-010 ignores prose outside response blocks; does it also ignore a SECOND `no-mistakes-response`
  block for an ID already answered (duplicate), or is duplicate-block an FR-015 failure? Edge Cases
  line 94 lists "duplicated" but no FR maps the outcome.
