# Clarifications — Review File Handoff

**Status:** ARCHIVED
**Applied:** 2026-06-15-205640
**Generated:** 2026-06-15T20:47:52+07:00
**Spec:** spec.md
**Mode:** batch

**Instructions:**
- Edit each `Your Answer:` line below.
- Type an option letter (A/B/C/...), or `recommended` / `yes` / `suggested` to accept the suggestion, or your own short answer (<=5 words).
- Leave the line blank to skip a question.
- Save the file, then re-run `/clarifybatch` (or `/clarifybatch --apply`) to apply all answers in one pass.

---

## Q1. What exact fenced response block schema should the parser accept for each finding?

**Category:** Functional Scope & Behavior
**Why it matters:** The answer fixes the Markdown contract, parser shape, docs examples, and validation fixtures.
**Recommended:** Option A - Existing IPC response fields already use action plus optional per-finding instructions; a line-oriented fenced block keeps manual editing easy while avoiding JSON fragility.

| Option | Description |
|--------|-------------|
| A | A fenced block tagged `no-mistakes-review-response` with `action: fix|accept|skip` and `solution: <one-line text>` fields |
| B | A fenced block tagged `no-mistakes-review-response` containing a JSON object with `action` and `solution` fields |
| C | Markdown headings plus HTML comments delimit the editable `action` and `solution` values |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** The spec says user decisions are read only from "fenced response blocks marked for no-mistakes review responses" and supported actions are exactly `fix`, `accept`, and `skip`. `internal/ipc/protocol.go:104`-`117` already models responses as an action plus optional per-finding instructions, so a line-oriented block matches the existing contract.

---

## Q2. Which finding identifier should each response block use when matching edited answers to the latest findings?

**Category:** Domain & Data Model
**Why it matters:** Identifier choice determines duplicate detection, unknown-ID validation, fix dispatch, and stable test fixtures.
**Recommended:** Option A - The codebase already normalizes findings to `Finding.ID`, and existing response APIs select findings by those IDs.

| Option | Description |
|--------|-------------|
| A | Use the existing normalized `Finding.ID` value persisted with the latest finding |
| B | Use the finding's ordinal position in the rendered review file |
| C | Use a deterministic hash of finding title, location, and context |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** `internal/types/findings.go:22`-`24` defines `Finding.ID`, and `internal/types/findings.go:95`-`103` normalizes missing IDs before findings are persisted or emitted. `internal/types/findings.go:106`-`124` then filters selected findings by those IDs.

---

## Q3. How should processed metadata appear before and after the user processes the review file?

**Category:** Domain & Data Model
**Why it matters:** Processing metadata affects stale-file validation, audit readability, and whether generated-but-unprocessed files are parseable.
**Recommended:** Option A - Explicit pending metadata is easy to validate before processing and can be overwritten only after a successful process action.

| Option | Description |
|--------|-------------|
| A | Render `processed_action: pending` with an empty processed timestamp initially, then overwrite both after successful processing |
| B | Omit processed metadata until after successful processing |
| C | Stamp processed timestamp at file generation time and keep `processed_action: pending` until processing |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** FR-002 requires metadata for both "processed timestamp" and "processed action" in the handoff file. Rendering `pending` up front keeps generated files parseable against the same metadata contract before and after processing.

---

## Q4. After validation fails during `p process`, what should the terminal gate display next?

**Category:** Interaction & UX Flow
**Why it matters:** This determines the user's recovery loop and the exact TUI/automation assertions for malformed files.
**Recommended:** Option A - Keeping the compact gate visible with one concise blocking error preserves the new file-first workflow without reintroducing per-finding controls.

| Option | Description |
|--------|-------------|
| A | Keep the compact gate open, show one concise validation error plus the review file path, and keep only process/cancel actions |
| B | Keep the gate open and show a multi-error validation list in the terminal |
| C | Re-render the old per-finding terminal controls after validation failure |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** FR-012 says validation failure must "keep the review gate open, show a concise terminal error with the review file path, and preserve the user's file contents." FR-014 also limits the terminal view to the summary/path/process/cancel workflow.

---

## Q5. What privacy or filesystem protection should generated review handoff files require?

**Category:** Non-Functional Quality Attributes
**Why it matters:** The review file may contain code locations and remediation context, and the chosen posture affects generation, commits, docs, and security tests.
**Recommended:** Option A - The file is already an explicit PR audit artifact, so normal checkout permissions and existing finding content are consistent with the feature's audit goal.

| Option | Description |
|--------|-------------|
| A | Write only inside the project checkout with normal repository file permissions and no additional redaction |
| B | Redact absolute paths and secret-like values before writing the file |
| C | Force owner-only permissions such as `0600` for local handoff files |
| D | Do not commit review files that contain high-severity security findings |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** The handoff file is explicitly a PR audit artifact: the spec requires the latest review file to be copied into the isolated pipeline work area and included in the PR commit. The assumptions also place editing inside the project checkout, so extra local-only permissions or redaction would conflict with that audit goal.

---

## Q6. What performance target should processing a valid 20-finding handoff meet on a local checkout?

**Category:** Non-Functional Quality Attributes
**Why it matters:** The success criteria require 20 findings in one action, but a latency target controls whether this needs benchmark-style coverage or only functional tests.
**Recommended:** Option A - One second is a practical local target for parsing, validation, and response preparation without overfitting to machine-specific timing.

| Option | Description |
|--------|-------------|
| A | Complete parsing, validation, and response preparation within 1 second |
| B | Complete parsing, validation, and response preparation within 3 seconds |
| C | No explicit latency target beyond correctly processing 20 findings |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** C
**Reason:** SC-002 only requires that a valid handoff with at least 20 findings is "processed in one terminal action" with correct response mapping. The feature intent does not state a latency-sensitive workflow, so adding a timing SLA would create test burden without a product constraint.

---

## Q7. When a `fix` response has no usable solution text, what should processing do?

**Category:** Edge Cases & Failure Handling
**Why it matters:** The spec mentions default fix choices and malformed empty/comment-only solutions; this answer removes a direct validation ambiguity.
**Recommended:** Option A - This aligns with the stated requirement that recommendation option 1 is the default fix choice when the user leaves a fix solution empty.

| Option | Description |
|--------|-------------|
| A | Use recommendation option 1 as the fix instruction |
| B | Reject validation until the user supplies a solution |
| C | Send the fix request with no user instruction |
| D | Treat the response as `accept` |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** FR-004 states that "option 1" is the default fix choice when the user leaves a fix solution empty. That directly covers empty or comment-only fix solution text.

---

## Q8. Which anchor filenames should the review file path resolver recognize?

**Category:** Constraints & Tradeoffs
**Why it matters:** The active spec says `task.md`, while the spec-kit project path resolver reports `tasks.md`; this affects path placement and regression tests.
**Recommended:** Option A - Spec-kit uses `tasks.md`, and pairing it with `plan.md` matches the active feature directory layout.

| Option | Description |
|--------|-------------|
| A | Recognize `plan.md` and `tasks.md` |
| B | Recognize `plan.md` and literal `task.md` |
| C | Recognize `plan.md`, `task.md`, and `tasks.md` |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** Spec-kit resolves the task artifact as `tasks.md`: `.specify/scripts/bash/common.sh:297`-`298` sets `IMPL_PLAN` to `plan.md` and `TASKS` to `tasks.md`. `.specify/scripts/bash/check-prerequisites.sh:132`-`136` also requires `tasks.md` for implementation.

---

## Q9. Which automation surfaces must include the additive `phase` and `review_file` values in the first release?

**Category:** Integration & External Dependencies
**Why it matters:** This bounds compatibility work across live events, reattach flows, CLI status output, and machine-readable automation payloads.
**Recommended:** Option A - It satisfies live and reattached automation consumers without forcing unrelated response-command changes.

| Option | Description |
|--------|-------------|
| A | Live gate events, reattached run state, and `axi status` run/gate output |
| B | Only `axi status` run/gate output |
| C | Only machine-readable JSON output, not human-readable automation text |
| D | Every CLI command that mentions review status |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** FR-026 requires the review file path for reattached terminal sessions and automation output, and FR-027 requires additive `phase` and `review_file` values in automation run and gate output. Live gate events need the same additive values so the terminal can show the compact handoff without waiting for reattach.

---

## Q10. Which labels are canonical for remediation guidance across the file, terminal, docs, and review summaries?

**Category:** Terminology & Consistency
**Why it matters:** The codebase has existing `suggested_fix` fields, while the spec chooses user-facing `Recommendation`; consistent labels prevent UI/docs drift.
**Recommended:** Option A - It preserves existing internal schema naming while giving users distinct labels for agent guidance and their own fix instruction.

| Option | Description |
|--------|-------------|
| A | Use `Recommendation` for agent guidance and `Solution` for user-authored fix text |
| B | Use `Suggested fix` everywhere |
| C | Use `Recommendation` everywhere |
| D | Use `Solution` everywhere |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** A
**Reason:** FR-003 fixes the handoff sections as `Issue`, `Context`, `Recommendation`, and `User Answer`, while FR-008 describes non-empty solution text as the user's instruction for a fix. The constitution keeps `Recommendation` as the review issue wording while preserving existing structured review concepts.
