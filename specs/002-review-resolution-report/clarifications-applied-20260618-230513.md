# Clarifications - Review Resolution Report

**Status:** APPLIED
**Generated:** 2026-06-18T22:59:30+07:00
**Applied:** 2026-06-18T23:05:13+07:00
**Spec:** spec.md
**Mode:** batch

**Instructions:**
- Edit each `Your Answer:` line below.
- Type an option letter (A/B/C/...), or `recommended` / `yes` / `suggested` to accept the suggestion, or your own short answer (<=5 words).
- Leave the line blank to skip a question.
- Save the file, then re-run `/clarifybatch` (or `/clarifybatch --apply`) to apply all answers in one pass.

---

## Q1. What SQLite shape should store review-resolution report metadata?

**Category:** Domain & Data Model
**Why it matters:** This determines the migration boundary, query API, uniqueness rules, and how AXI/TUI/PR code finds report status without reparsing Markdown.
**Recommended:** Option A - A dedicated run-keyed table keeps report metadata additive, queryable, and isolated from the already-busy `runs` and `step_rounds` tables.

| Option | Description |
|--------|-------------|
| A | Add a `review_resolution_reports` table keyed by `run_id`, with path, generated timestamp, counts, and status fields. |
| B | Add nullable report metadata columns directly to `runs`. |
| C | Store no report metadata table; infer status from `step_rounds` and the filesystem when needed. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** The spec says compact report metadata belongs in SQLite so AXI, TUI, and PR code can reference status without reparsing logs (`specs/002-review-resolution-report/spec.md:21-22`). A run-keyed additive table fits that contract while keeping report-specific fields out of the existing `runs` and `step_rounds` shapes (`internal/db/schema.go:12`, `internal/db/schema.go:40`).

---

## Q2. How should duplicate or repeated Review findings be represented in the report?

**Category:** Domain & Data Model
**Why it matters:** The spec requires every Review issue to appear exactly once, but follow-up reviews may repeat, rename, or slightly alter a finding.
**Recommended:** Option A - Normalized finding ID should be the primary identity; repeated sightings update the same entry, while ambiguous ID changes remain still open instead of being coalesced heuristically.

| Option | Description |
|--------|-------------|
| A | One entry per normalized finding ID; repeated same-ID findings update the entry, and ambiguous ID changes stay still open. |
| B | One entry per Review sighting, even if the same issue appears across multiple rounds. |
| C | Coalesce by file, line, and description when IDs differ. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** The spec assumes normalized Review finding IDs are the primary identity for matching selections, resolutions, and follow-up Review output (`specs/002-review-resolution-report/spec.md:182`). It also says uncertain identity must preserve honesty instead of marking an issue resolved (`specs/002-review-resolution-report/spec.md:183`).

---

## Q3. When should the report file first be created during a Review step with findings?

**Category:** Functional Scope & Behavior
**Why it matters:** This affects interrupted-run behavior and whether users can inspect a partial report while a Review gate is waiting for approval.
**Recommended:** Option A - Create it as soon as Review findings are recorded, with unresolved items marked still open, then refresh after fixes and terminal decisions.

| Option | Description |
|--------|-------------|
| A | Create immediately after first Review findings are recorded and refresh as the step progresses. |
| B | Create only after the Review step reaches a terminal approve, skip, abort, failure, or clean outcome. |
| C | Create only after at least one fix round runs. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-001 requires a report whenever the Review step produces findings, and FR-020 requires updating it when Review findings are first recorded (`specs/002-review-resolution-report/spec.md:124`, `specs/002-review-resolution-report/spec.md:143`). Creating it immediately is the simplest way to satisfy interrupted-run visibility.

---

## Q4. What should happen if report writing or metadata persistence fails for a run with Review findings?

**Category:** Edge Cases & Failure Handling
**Why it matters:** The spec says failures must be loud or actionable, but the exact gate behavior changes pipeline control flow and tests.
**Recommended:** Option A - Fail the Review step/run when required report persistence fails; this avoids claiming a passed gate without the required audit artifact.

| Option | Description |
|--------|-------------|
| A | Fail the Review step/run with an actionable error. |
| B | Add an actionable Review finding and pause for approval. |
| C | Log the error and continue without report metadata. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-025 requires report generation failures to be loud or actionable for runs with Review findings (`specs/002-review-resolution-report/spec.md:148`). Since FR-001 makes the report mandatory once findings exist, continuing as if the gate passed would violate the artifact contract (`specs/002-review-resolution-report/spec.md:124`).

---

## Q5. How much review-resolution detail should AXI and TUI display directly?

**Category:** Interaction & UX Flow
**Why it matters:** This defines the IPC payload, rendering tests, and whether UI surfaces need report-content parsing or only metadata.
**Recommended:** Option A - Show compact status, counts, and local path only; leave the full narrative in the Markdown report.

| Option | Description |
|--------|-------------|
| A | Show compact status, counts, and local report path; do not inline full report content. |
| B | Show the full report content inline in AXI/TUI run detail views. |
| C | Show only a boolean "report exists" indicator with no path or counts. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** The AXI/TUI scenario requires those surfaces to expose that a report exists and provide the local path or compact status (`specs/002-review-resolution-report/spec.md:56`). The Report Reference entity reserves full detail for the report while AXI/TUI show a local path, and PR summaries show compact counts/status (`specs/002-review-resolution-report/spec.md:158`).

---

## Q6. What exact review-resolution content should generated PR bodies include?

**Category:** Integration & External Dependencies
**Why it matters:** PR summaries are public-ish remote text, so this decides what can leave the local `$NM_HOME` boundary.
**Recommended:** Option A - Include compact counts/status only and never include local paths, report excerpts, or private filesystem details.

| Option | Description |
|--------|-------------|
| A | Include compact resolved/accepted/still-open counts or status only; omit local paths and full report excerpts. |
| B | Include local report path as plain text but not as a link. |
| C | Include a sanitized excerpt from each report section. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-023 requires generated PR content to reference review-resolution status without publishing local-only filesystem paths as public links (`specs/002-review-resolution-report/spec.md:146`). The PR story also says the Pipeline section should include compact counts or status from metadata (`specs/002-review-resolution-report/spec.md:102`).

---

## Q7. Should the report include raw diff hunks or transcript snippets as resolution evidence?

**Category:** Non-Functional Quality Attributes
**Why it matters:** This controls privacy, Markdown-safety, report size, and whether generated artifacts risk leaking raw agent transcripts.
**Recommended:** Option A - Avoid raw diffs and transcripts; use structured resolution text, changed-file lists, commit SHA, and sanitized summaries instead.

| Option | Description |
|--------|-------------|
| A | No raw diffs or transcripts; include structured summaries, changed files, and commit SHA when available. |
| B | Include bounded sanitized diff excerpts, but never transcripts. |
| C | Include bounded sanitized diff excerpts and relevant transcript snippets. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-024 requires sanitizing report content from agent output, findings, diffs, and commit metadata to avoid malformed Markdown and accidental raw transcript storage (`specs/002-review-resolution-report/spec.md:147`). The Resolution Entry entity already captures structured summaries, changed files, and fix commit evidence without needing raw hunks or transcripts (`specs/002-review-resolution-report/spec.md:155`).

---

## Q8. What size limits should apply to generated report content sourced from findings, agent output, and diffs?

**Category:** Non-Functional Quality Attributes
**Why it matters:** Without limits, malformed or verbose agent output can create unusable Markdown and expensive UI/PR rendering paths.
**Recommended:** Option A - Enforce per-field and total-report caps with explicit truncation markers so the report stays readable while preserving honesty.

| Option | Description |
|--------|-------------|
| A | Enforce per-field and total-report caps, with explicit truncation markers. |
| B | Allow unbounded report content after Markdown sanitization. |
| C | Fail report generation when content exceeds limits. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-024 forces sanitization of agent output, findings, diffs, and commit metadata, so bounded fields with explicit truncation markers are part of preserving readable Markdown (`specs/002-review-resolution-report/spec.md:147`). Hard failure on length would conflict with FR-005's requirement to include every Review issue known for the run (`specs/002-review-resolution-report/spec.md:128`).

---

## Q9. How should fix-commit evidence be recorded when a fix round produces no commit?

**Category:** Edge Cases & Failure Handling
**Why it matters:** The report must avoid guessing later commits, but some fix rounds may produce no changes or fail before a commit exists.
**Recommended:** Option A - Store nullable `fix_commit_sha` plus an explicit `no_commit_reason` so report generation can distinguish no-op, failed, and missing evidence cases.

| Option | Description |
|--------|-------------|
| A | Store nullable `fix_commit_sha` and a short `no_commit_reason` for fix rounds without commits. |
| B | Leave commit SHA empty and infer from run head or later git history when needed. |
| C | Require every Review fix round to create a commit. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-018 requires associating fix commit SHA only for Review fix rounds that create commits, so later commits are not guessed as Review fixes (`specs/002-review-resolution-report/spec.md:141`). The edge cases require no-commit situations to preserve the issue and label missing evidence honestly (`specs/002-review-resolution-report/spec.md:114`).

---

## Q10. Which documentation and generated guidance updates are in scope for the first implementation?

**Category:** Constraints & Tradeoffs
**Why it matters:** The spec names several docs and generated skill surfaces; this determines task scope and verification for user-visible behavior.
**Recommended:** Option A - Update the core Review/Auto-Fix/Pipeline/TUI/AXI/PR docs plus generated `/no-mistakes` skill text affected by the new report.

| Option | Description |
|--------|-------------|
| A | Update core Review, Auto-Fix, Pipeline, TUI, AXI, PR docs, and affected generated `/no-mistakes` skill text. |
| B | Update every docs page mentioning findings, PRs, reports, artifacts, logs, or local paths. |
| C | Defer docs and generated guidance until after implementation. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** recommended
**Reason:** FR-026 requires documentation and generated agent-facing guidance updates when report behavior, AXI/TUI output, or PR summary content changes (`specs/002-review-resolution-report/spec.md:149`). The constitution names the exact doc families in scope: Review, Auto-Fix Loop, Pipeline, AXI, TUI, PR summaries, local state paths, and affected `/no-mistakes` skill text (`specs/002-review-resolution-report/spec.md:166`).

## Correction Note - 2026-06-21

These clarifications remain useful for metadata, sanitization, status, and
surface behavior, but they must be read under the corrected grill-me authority:
the full `## Decisions` section in `plans/grill-me/review-resolution-report.md`
is binding. The prior interpretation that moved the report to
`$NM_HOME/reports/<runID>/review-resolution.md` was based on an invalid
`superseded` header and is not current product direction.

Any local-only path, no-force-add, or PR-no-path reading derived from this
clarification batch is superseded. The current contract is repo-local committed
evidence at `no-mistakes/<branch-slug>/review-resolution.md`, exact force-add
staging for that artifact, and PR surfacing of the repo-relative path without
absolute local filesystem details.
