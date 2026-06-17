# Research: Review Resolution Report

## Decision: Store a run-scoped Markdown artifact plus SQLite metadata

Use `$NM_HOME/reports/<runID>/review-resolution.md` as the durable report path and persist one row per run in a new `review_resolution_reports` table. The metadata row stores path, generation status, latest outcome, stable summary counts, source round identifiers, timestamps, stale/error state, and report contract version.

**Rationale**: The spec requires one current report per run, discoverable by AXI, TUI, PR summaries, regeneration, and future agents. A file artifact is readable and PR-friendly, while SQLite metadata lets hot surfaces display a compact reference without parsing Markdown.

**Alternatives considered**:

- Store full report content only in SQLite: rejected because it weakens the durable human-readable artifact requirement and makes PR/user review harder.
- Write the report into the user's working tree: rejected because reporting must not intentionally mutate the user's day-to-day tree or affect push semantics.
- Only add report summaries to existing output: rejected because future agents and users need one durable reference.

## Decision: Derive truth from persisted step results and rounds

Build reports from `step_results` and `step_rounds`: `findings_json`, `user_findings_json`, `selected_finding_ids`, `selection_source`, `fix_summary`, round order, step status, run status, branch, base/head SHAs, PR URL, and safe run intent.

**Rationale**: Existing review resolution state is already captured in the review round model. Stored rounds survive missed live events and are the only durable source that ties findings, user selections, user-authored findings, instructions, and fix summaries together.

**Alternatives considered**:

- Derive from live IPC events: rejected because missed events would make regeneration impossible.
- Derive from terminal logs or raw transcripts: rejected by privacy and constitution constraints.
- Re-run review to reconstruct history: rejected because it can change results and would not preserve the original decisions.

## Decision: Add `internal/reviewreport` as the central derivation/rendering package

Create a package that owns report snapshots, data-integrity checks, decision mapping, latest-outcome precedence, summary counts, sanitization, and Markdown rendering.

**Rationale**: AXI, TUI, daemon, executor, PR, and DB code all need consistent labels and counts. A shared package prevents drift and keeps the stable report contract testable.

**Alternatives considered**:

- Render directly inside the executor: rejected because PR/AXI/TUI would need duplicate logic.
- Extend `internal/pipeline/steps/prsummary.go` to own report rendering: rejected because the report is not PR-specific and must be available even when PR is skipped.

## Decision: Treat fix summaries as fix attempts, not verification

Every fix round appears in chronological order, but its `fix_summary` is an agent-reported summary of an attempted fix. The report may label it `Applied fix` only as a recorded fix attempt summary and must use follow-up review evidence to decide whether findings remain open.

**Rationale**: Prior gotchas and the spec both warn not to conflate proposed recommendations, agent summaries, and verified review outcomes.

**Alternatives considered**:

- Count each fix summary as resolved: rejected because an agent summary can be incomplete, wrong, or followed by unresolved findings.
- Hide missing summaries: rejected because the report must show `fix applied, no summary recorded`.

## Decision: Preserve `Accepted` for explicit human risk acceptance only

The current approval actions are `approve`, `fix`, `skip`, and `abort`. `Accepted` in the report is used only when stored data contains an explicit human/user risk-acceptance decision. Existing approve/skip behavior is not automatically translated to `Accepted`.

**Rationale**: `Accepted` is an approval-like label. Inferring it from a generic step approval or an unselected finding would overstate user intent and violate the trust boundary.

**Alternatives considered**:

- Map every unselected finding to `Accepted`: rejected because unselected can also mean skipped, deferred, still open, or not recorded.
- Add a new approval action in this feature: rejected because the spec forbids changing approval semantics.

## Decision: Fail closed on data inconsistencies

If stored review data is malformed or inconsistent, the report uses `review data inconsistent`, surfaces safe diagnostics, and omits confident resolved/unresolved totals until the bad records are identified.

Integrity checks include:

- malformed `findings_json`, `user_findings_json`, or `selected_finding_ids`;
- selected IDs that are absent from the source round and not present as user-authored findings;
- duplicate finding IDs within the same reportable round;
- fix summaries without a corresponding fix round;
- final step findings disagreeing with the latest stored round;
- summary counts that cannot be derived from the per-finding decisions;
- regenerated output older than a newer recorded source snapshot.

**Rationale**: A successful-looking report is worse than no report when the evidence chain is broken.

**Alternatives considered**:

- Best-effort counts with warnings: rejected because it can still imply resolution.
- Drop invalid records silently: rejected because missing records are themselves material evidence.

## Decision: Use deterministic latest-outcome precedence

The report chooses exactly one latest outcome using the precedence defined in [data-model.md](data-model.md). `no issues remain` is valid only from a successfully parsed latest review pass for the same run after the relevant fix attempt.

**Rationale**: Cancelled, failed, stale, unreadable, and awaiting-decision states can overlap. A deterministic table prevents reassuring labels from winning accidentally.

**Alternatives considered**:

- Let renderers choose wording per surface: rejected because AXI/TUI/PR would drift.
- Collapse unavailable/incomplete outcomes into one warning: rejected because tests and future agents need precise evidence-state labels.

## Decision: Direct surfaces show metadata only

AXI, TUI, and PR summaries may directly show report path/reference, generation status, latest outcome, stable summary counts, material sanitized fix summaries for PR, stale/error labels, and updated timestamp. They must not directly duplicate full finding context, user instructions, raw fix-summary text beyond sanitized one-line summaries, diffs, logs, transcripts, or code excerpts.

**Rationale**: The report is the detailed sanitized surface. Other outputs should stay compact and reduce accidental sensitive duplication.

**Alternatives considered**:

- Copy the full report into PR and AXI output: rejected because it duplicates sensitive-prone content and makes outputs noisy.
- Only print the path without counts/outcome: rejected because users and agents need enough state to notice unresolved/incomplete outcomes.

## Decision: No new dependency

Use existing Go standard library, SQLite helpers, Markdown string rendering, and current sanitizer patterns.

**Rationale**: The report format is deliberately stable and simple. New Markdown or sanitization dependencies would add maintenance and security surface without reducing meaningful complexity.

**Alternatives considered**:

- Markdown AST renderer: rejected for v1 because exact headings/labels and table-like sections are simple enough to render directly with tests.
- Separate JSON sidecar: rejected because the spec chose a stable Markdown contract and persisted metadata rather than a second artifact contract.
