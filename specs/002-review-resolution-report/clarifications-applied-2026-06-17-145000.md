# Clarifications — Review Resolution Report

**Status:** ARCHIVED
**Applied:** 2026-06-17-145000
**Generated:** 2026-06-17T14:42:47+07:00
**Spec:** spec.md
**Mode:** batch

**Instructions:**
- Edit each `Your Answer:` line below.
- Type an option letter (A/B/C/...), or `recommended` / `yes` / `suggested` to accept the suggestion, or your own short answer (<=5 words).
- Leave the line blank to skip a question.
- Save the file, then re-run `/clarifybatch` (or `/clarifybatch --apply`) to apply all answers in one pass.

---

## Q1. Where should the durable Markdown report live relative to run data?

**Category:** Integration & External Dependencies
**Why it matters:** Storage ownership determines how AXI, TUI, PR summaries, regeneration, and future agents locate the same report without relying on terminal output.
**Recommended:** Option A - A run-scoped artifact with its path recorded in run metadata gives durable file access while keeping the report tied to the source run.

| Option | Description |
|--------|-------------|
| A | Store one run-scoped Markdown artifact and persist its path/reference with the run metadata. |
| B | Store report content only in the run database and render Markdown on demand. |
| C | Write the report into the working tree as a user-managed project file. |
| D | Do not store a separate report; only include summaries in existing output surfaces. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:100` requires a durable report for qualifying runs, and `spec.md:122` defines it as a record "for one run." A run-scoped artifact with a persisted reference is the smallest storage contract that preserves that invariant.

---

## Q2. When review-resolution state changes during one run, how should the report lifecycle behave?

**Category:** Functional Scope & Behavior
**Why it matters:** The lifecycle determines whether users see a single current truth, per-round snapshots, or only a final report, and it affects cancellation/failure handling.
**Recommended:** Option A - A single report updated in place can preserve chronological fix attempts while keeping discovery simple and avoiding stale competing artifacts.

| Option | Description |
|--------|-------------|
| A | Maintain one report per run, updating it as review decisions, fixes, and final outcomes become available. |
| B | Create a separate immutable report for each review or fix round. |
| C | Generate the report only after the full pipeline reaches a terminal state. |
| D | Generate only on explicit user request. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:107` requires applied fix summaries "in chronological order," while `spec.md:111` says the report remains available after failure or cancellation. One updated report per run keeps that history current without multiplying artifacts.

---

## Q3. Which user-facing surfaces must expose a direct report reference?

**Category:** Interaction & UX Flow
**Why it matters:** The answer defines where users and agents can discover the report, and prevents a durable artifact from existing without a reliable workflow entry point.
**Recommended:** Option A - AXI success/status, TUI review gate details, and PR summaries cover the active human, headless agent, and code-review surfaces named by the spec.

| Option | Description |
|--------|-------------|
| A | AXI success/status output, TUI review gate details, and PR summaries when review resolution occurred. |
| B | AXI command output only. |
| C | PR summaries only. |
| D | TUI surfaces only. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `source-context.md:20-28` names TUI, AXI, and PR summaries as the existing surfaces where review-resolution details are spread today. `spec.md:113-114` also explicitly require agent-facing command output and PR summaries to include or reference review-resolution information.

---

## Q4. May the report include code excerpts or diff hunks from reviewed changes?

**Category:** Non-Functional Quality Attributes
**Why it matters:** The content boundary affects privacy, report size, and whether the report can safely be reused in PR and agent-facing contexts.
**Recommended:** Option A - Existing sanitized finding context, locations, decisions, and fix summaries are enough for traceability while honoring the no-raw-logs/no-raw-transcripts boundary.

| Option | Description |
|--------|-------------|
| A | No; include finding locations, safe finding context, decisions, and summaries, but no code excerpts or diff hunks. |
| B | Include bounded code excerpts only when already present in structured finding context. |
| C | Include bounded diff hunks for findings selected for fix. |
| D | Include full relevant diffs when available. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:115` says the report must avoid raw transcripts, raw logs, and secret-bearing data, allowing only safe intent or summarized context. Locations, decisions, and summaries satisfy traceability without pulling raw code or diffs into the artifact.

---

## Q5. How should the report label a run that is cancelled or fails after review fixes but before trustworthy final review evidence exists?

**Category:** Edge Cases & Failure Handling
**Why it matters:** This prevents the report from overstating success when the latest review outcome is missing, stale, or unverified.
**Recommended:** Option A - "Review resolution incomplete" communicates that some fix work happened while explicitly avoiding a false all-clear claim.

| Option | Description |
|--------|-------------|
| A | Mark the latest outcome as "review resolution incomplete" and show the latest trustworthy evidence. |
| B | Mark all previously fixed findings as resolved and leave only later failures outside the report. |
| C | Mark the entire review resolution as failed even if some findings were fixed. |
| D | Suppress the report until final review evidence exists. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:110` says the report must never claim all review issues are resolved when final evidence is missing or unreadable. `source-context.md:86` repeats the same rule: do not claim resolution unless latest review evidence proves it.

---

## Q6. How should regenerated reports handle older or partial run data that lacks selected finding IDs, user instructions, or fix summaries?

**Category:** Domain & Data Model
**Why it matters:** Legacy and partial data handling determines whether regeneration is trustworthy or silently invents missing resolution facts.
**Recommended:** Option A - Explicit "not recorded" labels preserve durability and truthfulness without inferring decisions from incomplete historical data.

| Option | Description |
|--------|-------------|
| A | Generate the report and label missing fields as "not recorded" or "unavailable" without inferring decisions. |
| B | Skip report generation for runs missing any resolution metadata. |
| C | Infer missing decisions from finding counts, latest findings, and fix summaries. |
| D | Generate only a minimal run-level summary with no per-finding details. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:92-94` lists missed live events and older run data with missing metadata as explicit edge cases. `spec.md:156` requires regeneration from stored state, so missing fields need honest "not recorded" labels rather than inferred facts.

---

## Q7. What is the canonical label for actionable findings the user chose not to fix but deliberately accepted as risk?

**Category:** Terminology & Consistency
**Why it matters:** Consistent labels keep accepted risk, skipped review work, deferred work, and unresolved findings from being collapsed into the same state.
**Recommended:** Option A - "Accepted" should mean deliberate risk acceptance, while "Skipped" and "Deferred" remain available for different workflows.

| Option | Description |
|--------|-------------|
| A | Accepted |
| B | Skipped |
| C | Deferred |
| D | Still open |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:112` lists `Accepted`, `Skipped`, and `Risk` as plain workflow labels, and `spec.md:131` separately names accepted risks and skipped findings. That split makes "Accepted" the canonical label for deliberate risk acceptance.

---

## Q8. What stable report contract should future agents and tests rely on?

**Category:** Completion Signals
**Why it matters:** A stable contract makes acceptance tests meaningful and gives future agents a predictable way to extract resolution facts without requiring a separate API.
**Recommended:** Option A - Stable Markdown headings, labels, and summary counts keep the report human-readable while making it testable and agent-friendly.

| Option | Description |
|--------|-------------|
| A | Stable Markdown headings, user-facing labels, and summary counts. |
| B | Human-readable prose only, with no stable structure beyond Markdown. |
| C | Markdown plus a separate machine-readable JSON sidecar. |
| D | A formal embedded JSON block inside the Markdown report. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:133` requires the format to be human-readable and stable enough for future agents. Stable headings, labels, and counts meet that contract without adding a sidecar API the spec does not require.

---

## Q9. When a PR summary references a durable report, what should it include directly?

**Category:** Integration & External Dependencies
**Why it matters:** PR reviewers need enough context in the PR body to assess risk even if the detailed artifact is opened separately.
**Recommended:** Option A - Counts, latest outcome, and the material applied-fix summaries give reviewers a bounded summary while avoiding duplication of the full report.

| Option | Description |
|--------|-------------|
| A | Summary counts, latest review outcome, material applied-fix summaries, and a report reference. |
| B | Only a report reference. |
| C | The full report body. |
| D | Only unresolved findings and risk level. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:77-79` requires PR summaries to include review fixes, latest outcome, and a report reference without duplicating the full report body. `spec.md:114` keeps that bounded by allowing either a concise summary or durable report reference.

---

## Q10. What should happen if report generation itself fails after review data has been captured?

**Category:** Non-Functional Quality Attributes
**Why it matters:** The failure policy determines whether reporting can block the pipeline or whether the run continues with an explicit reporting error.
**Recommended:** Option A - Continue the run, surface a reporting error, and keep review semantics unchanged; the spec already treats reporting as non-semantic.

| Option | Description |
|--------|-------------|
| A | Continue the pipeline, surface a report-generation error, and preserve captured review data for later regeneration. |
| B | Fail the pipeline immediately whenever report generation fails. |
| C | Retry indefinitely until the report can be written. |
| D | Silently skip the report and rely on existing output. |
| Short | Provide a different short answer (<=5 words) |

**Your Answer:** recommended
**Reason:** `spec.md:12` defines this as reporting/reference capability that must not change review, PR, or CI behavior. If generation fails after review data is captured, preserving the pipeline semantics while surfacing the reporting error follows that boundary.
