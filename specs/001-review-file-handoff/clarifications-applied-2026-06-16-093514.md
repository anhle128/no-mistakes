# Clarifications — Review File Handoff

**Status:** ARCHIVED
**Applied:** 2026-06-16-093514
**Generated:** 2026-06-16T09:26:41+07:00
**Spec:** spec.md
**Mode:** batch

**Instructions:**
- Edit each `Your Answer:` line below.
- Type an option letter (A/B/C/...), or `recommended` / `yes` / `suggested` to accept the suggestion, or your own short answer (<=5 words).
- Leave the line blank to skip a question.
- Save the file, then re-run `/clarifybatch` (or `/clarifybatch --apply`) to apply all answers in one pass.

---

## Q1. Which anchor filenames should be eligible when choosing the review file location?

**Category:** Terminology & Consistency
**Why it matters:** The current spec says `task.md`, while Spec Kit commonly uses `tasks.md`; this affects path resolution, tests, and user-facing file placement.
**Recommended:** Option C - Supporting both names avoids missing the intended anchor while preserving the "exactly one anchor" rule.

| Option | Description |
|--------|-------------|
| A | Only changed `plan.md` or `task.md` files are eligible anchors. |
| B | Only changed `plan.md` or `tasks.md` files are eligible anchors. |
| C | Changed `plan.md`, `task.md`, or `tasks.md` files are eligible anchors, but only when exactly one total anchor exists. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** C
**Reason:** `spec.md:93` currently requires a single changed `plan.md` or `task.md` anchor, while `.specify/templates/tasks-template.md:47` names the generated file `tasks.md`. Supporting all three under the existing "exactly one" rule keeps both contracts without widening placement behavior.

---

## Q2. What machine-readable format should the review handoff file use for run and processing metadata?

**Category:** Domain & Data Model
**Why it matters:** The writer and parser need a stable metadata contract so validation can be deterministic and documented.
**Recommended:** Option A - YAML front matter is readable in Markdown, easy to parse, and keeps metadata separate from user-editable prose.

| Option | Description |
|--------|-------------|
| A | YAML front matter at the top of the Markdown file. |
| B | A fenced `no-mistakes-metadata` block containing JSON. |
| C | A Markdown table with strict key/value rows. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:95` requires machine-readable run/status/processing metadata, and `spec.md:100` says user prose outside response blocks is ignored. YAML front matter gives one unambiguous metadata area outside the editable response prose.

---

## Q3. What initial processing metadata should be written before the user processes the file?

**Category:** Domain & Data Model
**Why it matters:** Empty or absent processing fields change validation behavior, audit output, and the exact diff after `p process`.
**Recommended:** Option A - Explicit pending values make the lifecycle testable without implying the file was already processed.

| Option | Description |
|--------|-------------|
| A | Write `processed_at: null` and `processed_action: pending` initially. |
| B | Omit processing fields until successful `p process`. |
| C | Write empty strings for processing fields initially. |
| D | Write `processed: false` initially, then replace it with processed fields after processing. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:95` requires processed timestamp/action metadata, and `spec.md:108` says successful processing updates only processing metadata. Initial `null`/`pending` gives a concrete unprocessed lifecycle state without pretending processing already happened.

---

## Q4. Should the metadata include a review-result revision or hash to reject stale editor buffers?

**Category:** Edge Cases & Failure Handling
**Why it matters:** A user can keep an old handoff file open while a later review cycle overwrites the file; a revision token determines whether stale saves are blocked reliably.
**Recommended:** Option A - A deterministic result hash blocks stale copies even when finding IDs are reused across cycles.

| Option | Description |
|--------|-------------|
| A | Include a deterministic review-result hash and require it to match the current gate. |
| B | Include only a generated timestamp and reject older timestamps. |
| C | Do not add a revision token; validate only run ID, status, and finding IDs. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:120` says new review results overwrite the same file with latest findings, and `internal/types/findings.go:95` can assign order-based `review-N` IDs. A result hash is the minimal token that distinguishes current content from a stale save when run/status/IDs still match.

---

## Q5. What should be the canonical source and fallback for each response block's finding ID?

**Category:** Domain & Data Model
**Why it matters:** Response block identity drives fix selection, duplicate detection, stale-file validation, and audit traceability.
**Recommended:** Option A - Existing structured finding IDs should remain canonical; failing when absent avoids unstable response IDs.

| Option | Description |
|--------|-------------|
| A | Use existing structured finding IDs and fail file generation if any latest finding lacks one. |
| B | Use existing structured IDs when present, otherwise derive deterministic IDs from finding content. |
| C | Assign sequential IDs based on render order for each review cycle. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:99` requires one response block per latest finding ID, and `internal/types/findings.go:24` defines that ID as structured finding data. If a latest finding has no ID, generating fallback IDs at the handoff boundary would weaken duplicate and stale-file checks.

---

## Q6. How should CLI/AXI expose the review phase label and review file path while preserving raw statuses?

**Category:** Integration & External Dependencies
**Why it matters:** Automation clients and tests need to know whether these values are structured API fields or only human-readable text.
**Recommended:** Option A - Additive structured fields preserve compatibility while giving clients a stable integration contract.

| Option | Description |
|--------|-------------|
| A | Add nullable structured fields such as `review_phase_label` and `review_file_path`, without changing raw status fields. |
| B | Include the phase label and path only in human-readable message text. |
| C | Put both values inside an existing generic metadata map with documented keys. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:117` explicitly says CLI/AXI must preserve raw statuses while exposing the review phase label and review file path. Existing AXI rendering already emits structured `step` and `status` fields (`internal/cli/axi_render.go:248`), so additive nullable fields fit the current contract.

---

## Q7. When automation uses the old approve/fix/skip response contract, should the review file be updated to reflect that decision for PR auditability?

**Category:** Integration & External Dependencies
**Why it matters:** The spec requires old automation to keep working and the final review file to be auditable; the two paths need one source of truth.
**Recommended:** Option A - Mirroring automation decisions into the file preserves the audit artifact without forcing automation clients to edit Markdown.

| Option | Description |
|--------|-------------|
| A | Yes, update processing metadata and response blocks to reflect the automation decision. |
| B | Update only processing metadata; leave response blocks at their generated defaults. |
| C | No, leave the file unchanged when automation bypasses file processing. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:70` keeps old automation responses working, and `spec.md:137` defines the final file as an auditable record of review issues and decisions. Mirroring the old response into the same file is the only single-source audit path that does not force automation clients to edit Markdown.

---

## Q8. When review file validation fails, how much terminal detail should be shown?

**Category:** Interaction & UX Flow
**Why it matters:** The terminal gate must stay compact, but validation errors need enough detail for the user to fix the file without guessing.
**Recommended:** Option A - A concise summary plus the first actionable validation error keeps the terminal compact and points directly at the file repair.

| Option | Description |
|--------|-------------|
| A | Show the file path, one-line failure summary, first actionable validation error, and keep `p process` / `c cancel`. |
| B | Show all validation errors inline in the terminal. |
| C | Show only a generic failure message and the file path. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:106` requires a short actionable terminal error with file path, and `spec.md:111` keeps the gate compact with only file path plus `p process` / `c cancel`. Showing one first actionable error satisfies both; dumping all errors would break compactness.

---

## Q9. What size bounds should validation enforce for handoff files and per-finding solutions?

**Category:** Non-Functional Quality Attributes
**Why it matters:** Bounds prevent runaway parser memory usage and make tests for "large findings or long user answers" concrete.
**Recommended:** Option B - These caps are large enough for real review detail while keeping parser behavior predictable.

| Option | Description |
|--------|-------------|
| A | No explicit size bounds in v1. |
| B | Reject files over 1 MiB or any one `solution:` over 16 KiB. |
| C | Reject files over 5 MiB or any one `solution:` over 64 KiB. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** B
**Reason:** `spec.md:84` says large findings and long answers must remain usable, but `spec.md:153` requires malformed handoff validation to reject bad files deterministically. The 1 MiB / 16 KiB caps are large enough for review text while bounding parser memory and fixtures.

---

## Q10. If fix review has no remaining findings, should the final PR audit file preserve prior issue decisions or only show the no-findings final state?

**Category:** Functional Scope & Behavior
**Why it matters:** The current requirements say the file is overwritten with a no-findings state, but also define it as the auditable record of review issues and user decisions.
**Recommended:** Option A - Preserving prior decisions maintains PR auditability while still making the final no-findings state explicit.

| Option | Description |
|--------|-------------|
| A | Preserve prior finding decisions and add a final `No remaining review findings.` state. |
| B | Replace prior findings with only metadata, applied fix summary, and `No remaining review findings.` |
| C | Preserve only a summarized count of prior findings and decisions. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:137` defines the PR audit file as a record of review issues and user decisions, while `spec.md:121` requires the final no-findings state to be explicit. Preserving prior decisions in the same file satisfies auditability without adding review history files, which `spec.md:126` forbids.
