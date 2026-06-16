# Feature Specification: Review File Handoff

**Feature Branch**: `archon/thread-17570d85`  
**Created**: 2026-06-16  
**Status**: Draft  
**Input**: User description: "read requirement in create detail plan for review-file-handoff, deeply understand this new feature, scout source for know the context, spawn sub-agents for help, save reference review-file-handoff.md to spec.md, let next step follow the handoff, scout source code to know the context, create detail spec"

**Reference Source**: Derived from `plans/grill-me/review-file-handoff.md`. When this spec and the grill handoff differ, this spec is the authoritative implementation and review handoff contract.

## Clarifications

### Session 2026-06-16

- Q: Which anchor filenames should be eligible when choosing the review file location? → A: Changed `plan.md`, `task.md`, or `tasks.md` files are eligible anchors, but only when exactly one total anchor exists.
- Q: What machine-readable format should the review handoff file use for run and processing metadata? → A: YAML front matter at the top of the Markdown file.
- Q: What initial processing metadata should be written before the user processes the file? → A: Write `processed_at: null` and `processed_action: pending` initially; before processing, validation must reject any incoming file whose processing metadata is already non-null or non-`pending`, and the gate must derive processed outcomes only from valid response blocks or the executed automation decision.
- Q: Should the metadata include a review-result revision or hash to reject stale editor buffers? → A: Include a deterministic review-result hash, computed by the gate from live authoritative review state including a review-cycle revision, and require it to match the current gate; validation MUST recompute the hash from current gate state and MUST NOT trust the file-supplied hash as authority.
- Q: What should be the canonical source and fallback for each response block's finding ID? → A: Use existing structured finding IDs and fail file generation if any latest finding lacks one.
- Q: How should CLI/AXI expose the review phase label and review file path while preserving raw statuses? → A: Add nullable structured fields such as `review_phase_label` and `review_file_path`, without changing raw status fields.
- Q: When automation uses the old approve/fix/skip response contract, should the review file be updated to reflect that decision for PR auditability? → A: Yes, update processing metadata and response blocks to reflect the automation decision.
- Q: When review file validation fails, how much terminal detail should be shown? → A: Show the file path, one-line failure summary, first actionable validation error, and keep `p process` / `c cancel`.
- Q: What size bounds should validation enforce for handoff files and per-finding solutions? → A: Reject files over 1 MiB or any one `solution:` over 16 KiB.
- Q: If fix review has no remaining findings, should the final PR audit file preserve prior issue decisions or only show the no-findings final state? → A: Preserve prior finding decisions and add a final `No remaining review findings.` state.


## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review Issues In A File (Priority: P1)

As a developer stopped at the review gate, I need all current review findings written to a focused Markdown handoff file so I can read the issues, context, and recommendations in an editor instead of watching detailed findings stream through the terminal.

**Why this priority**: This is the core value of the feature. The terminal becomes a compact control surface, while the file becomes the durable review surface.

**Independent Test**: Can be tested by running a review gate that produces findings and confirming that the user sees only a compact terminal summary with the review file path, while the file contains the latest findings and response blocks.

**Acceptance Scenarios**:

1. **Given** a review gate produces one or more findings, **When** the review preview completes, **Then** the system creates a Markdown review file, displays its path, and does not render full finding details in the terminal gate view.
2. **Given** a review file is inside the project checkout, **When** the terminal displays the file location, **Then** it shows a relative path that the user can open directly from the project.
3. **Given** the review file is generated from the latest review results, **When** the user opens it, **Then** each finding includes `Issue`, `Context`, `Recommendation`, and `User Answer` sections.

---

### User Story 2 - Process File Answers Into Review Decisions (Priority: P1)

As a developer reviewing the handoff file, I need to answer each finding with `fix`, `accept`, or `skip` and then process the file from the terminal so the gate can continue with the correct approval or fix behavior.

**Why this priority**: A review file is only useful if the user's saved decisions round-trip back into the existing gate flow without losing intent.

**Independent Test**: Can be tested by editing response blocks in the handoff file, pressing `p process`, and confirming that selected fixes, accepted risks, skipped findings, and per-finding solution text drive the next gate action.

**Acceptance Scenarios**:

1. **Given** at least one response block has `action: fix`, **When** the user processes the review file, **Then** only those findings are sent to the fixer with the user's solution text or the default recommended solution.
2. **Given** all response blocks are `accept` or `skip`, **When** the user processes the review file, **Then** the review gate is approved without an extra confirmation prompt.
3. **Given** the review file is missing a latest finding, contains an unknown finding, has an invalid action, or has invalid metadata, **When** the user processes it, **Then** the system blocks processing, keeps the gate open, and displays a short actionable error with the file path.

---

### User Story 3 - Understand Review Sub-Phases Consistently (Priority: P2)

As a user watching the pipeline through terminal, TUI, CLI/AXI, logs, or PR summaries, I need review sub-phases to have consistent human labels so I can tell whether the system is previewing issues, waiting for file answers, fixing issues, or showing fix results.

**Why this priority**: The handoff changes how users inspect review findings, so every surface must describe the same review state with the same language.

**Independent Test**: Can be tested by moving a review run through `running`, `awaiting_approval`, `fixing`, `fix_review`, and `completed` states and checking each user-facing surface for the expected labels while raw statuses remain unchanged.

**Acceptance Scenarios**:

1. **Given** the review step is running, **When** a user views a human-facing surface, **Then** the phase label is `Review preview`.
2. **Given** the review step is awaiting file processing, **When** a user views a human-facing surface, **Then** the phase label is `Review preview complete`.
3. **Given** review fixes are running, **When** a user views a human-facing surface, **Then** the phase label is `Fixing review issues`.
4. **Given** fix results are awaiting review, **When** a user views a human-facing surface, **Then** the phase label is `Review fix result`.
5. **Given** review is completed or a non-review step is displayed, **When** a user views the pipeline, **Then** no review sub-phase label is added and a completed review step still displays as `Review`.

---

### User Story 4 - Preserve Automation And PR Auditability (Priority: P2)

As an automation user or reviewer, I need existing agent-driven approval commands to keep working while the final review file is included in the PR branch commit so review decisions remain auditable.

**Why this priority**: The feature must improve human review without breaking AXI/yolo/auto-fix flows or losing the review record before PR creation.

**Independent Test**: Can be tested by driving the same approval actions through automation, then allowing the pipeline to reach push and confirming that the final review file is committed to the PR branch even when it is the only remaining change.

**Acceptance Scenarios**:

1. **Given** an automation client uses existing approve, fix, or skip responses, **When** it responds to a review gate, **Then** the raw action and status contract still behaves as before and the review file records the mirrored decision for auditability.
2. **Given** a review handoff file exists when the pipeline reaches push, **When** the PR branch commit is created, **Then** the review file is included at the same relative path.
3. **Given** the review file was placed next to an anchor file, **When** the PR commit is created, **Then** the anchor file is not committed merely because it was used to choose the review file location.

### Edge Cases

- The review run is interrupted, canceled, superseded, or reattached after the event that first announced the file path.
- The review file is deleted, unreadable, moved, or edited with malformed metadata before `p process`.
- A response block is missing, duplicated, has an unknown finding ID, or contains an unsupported action.
- The user leaves `solution:` empty for a `fix` action.
- Comment lines appear inside a multiline `solution:` block.
- The review cycle reaches `fix_review` with no remaining findings.
- There are zero, one, or multiple changed `plan.md`, `task.md`, or `tasks.md` anchor candidates.
- The review file is the only remaining change when the pipeline reaches PR commit creation.
- Large findings or long user answers must remain usable in the file while compact surfaces avoid flooding the terminal.
- Existing automation responds through the old approve/fix/skip contract instead of editing the handoff file.
- A stale editor buffer attempts to process a file whose review-result hash no longer matches the current gate.
- The handoff file exceeds 1 MiB or a single `solution:` exceeds 16 KiB.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: When the review step produces findings for human decision, the system MUST write the latest findings and recommended responses to exactly one current Markdown review handoff file for the run.
- **FR-002**: The review handoff file MUST be named `review-issues-<run-short-id>.md`.
- **FR-003**: The system MUST place the review handoff file next to a single changed `plan.md`, `task.md`, or `tasks.md` anchor when exactly one total eligible anchor is present in uncommitted changes, otherwise next to a single such anchor from the latest reviewed commit, otherwise under `.no-mistakes/issues/<branch-slug>/`. "Latest reviewed commit" MUST mean the changed-file source used by the current review round: for normal review, paths from the resolved base commit to the persisted run head SHA; for fix-review after a fix, paths from the same resolved base commit to the current run head SHA plus any uncommitted worktree changes being reviewed. Reattach and reruns MUST use the persisted run base/head SHAs and current review step status for this fallback source rather than arbitrary later worktree state. Anchor resolution MUST treat changed paths as repo-relative, clean and resolve the candidate directory, reject absolute paths, traversal, `.git`, symlink escapes, or any path outside the project checkout, and FR-034 anchor suppression MUST use the resolved anchor path rather than basename alone.
- **FR-004**: The system MUST display a relative review file path when the file is inside the project checkout.
- **FR-005**: The review handoff file MUST include YAML front matter with machine-readable metadata for the current run, review step, review status, branch, deterministic review-result hash, processed timestamp, processed action, and review cycle revision; initially `processed_at` MUST be `null` and `processed_action` MUST be `pending`. The review cycle revision MUST be derived from the current persisted `step_rounds` record as the pair of its stable round ID and monotonic round number. It MUST advance only when the executor inserts a new review round for initial review, automatic/user-triggered fix review, or final no-findings review state; automation mirroring and metadata-only file processing MUST NOT advance it. Reattach and validation MUST recompute the revision from live persisted gate state, not from file front matter. The deterministic review-result hash MUST cover the normalized current gate inputs that can affect processing: run ID, review step/status, review cycle revision, ordered canonical finding IDs, severity, issue text, context, full recommendation option text, default response action, and any applied fix summary used by a final no-findings state.
- **FR-006**: The review handoff file MUST include a summary of total findings and severity counts.
- **FR-007**: Each latest finding MUST appear once in the handoff file with `Issue`, `Context`, `Recommendation`, and `User Answer` sections. The normalized content of each latest finding, including issue text, context, recommendation options, severity, and generated default action, is part of the review-result hash domain.
- **FR-008**: `Recommendation` content MUST contain one or two concrete options, with option 1 treated as the preferred default.
- **FR-009**: The user-answer area MUST contain one fenced `no-mistakes-response` block per latest finding ID, using the existing structured finding ID as canonical identity; file generation MUST fail if any latest finding lacks an ID or if the latest finding set contains duplicate IDs.
- **FR-010**: The system MUST parse only fenced `no-mistakes-response` blocks for user decisions and MUST ignore prose outside those blocks.
- **FR-011**: Each response block MUST support exactly these actions: `fix`, `accept`, and `skip`.
- **FR-012**: For `fix`, an empty `solution:` MUST mean use recommendation option 1; a non-empty `solution:` MUST be used as the user's per-finding instruction after ignoring comment-only lines. Non-empty `solution:` text MUST be delivered to the fixer as clearly delimited, untrusted user data scoped to that one finding ID; the fixer prompt MUST treat it as data, not as system/developer instructions or authority to modify unrelated findings.
- **FR-013**: For v1, `accept` and `skip` MUST both mean the finding is not sent to the fixer; they remain separate labels only for audit readability.
- **FR-014**: Initial response blocks MUST default from the reviewer's finding action as follows: auto-fix becomes `fix`, ask-user becomes `accept`, and no-op becomes `skip`.
- **FR-015**: On `p process`, the system MUST validate that the file is readable, YAML front matter metadata exists, run ID matches, step is review, status matches the current review gate state, the deterministic review-result hash matches the current gate, every latest finding has one response block, no unknown finding IDs are present, every action is valid, solution text is parseable, the file is at most 1 MiB, and each `solution:` value is at most 16 KiB. Any automation-mirrored review file MUST satisfy the same validation invariants before it is eligible to become the PR audit file.
- **FR-016**: If review file validation fails, the system MUST block processing, keep the gate open, show the relevant file path, a one-line failure summary, and the first actionable validation error, while keeping only `p process` and `c cancel` available.
- **FR-017**: The system MUST NOT regenerate the review handoff file or overwrite findings, recommendations, or user answers during `p process`. Validation, decision extraction, and processing metadata updates MUST operate on a single consistent snapshot of the file bytes; if the file changes before the update is committed, processing MUST reject the file and keep the gate open.
- **FR-018**: After successful file processing, the system MUST preserve the user's edited answers and update only processing metadata, changing `processed_at` from `null` and `processed_action` from `pending` to the successful processing values. File processing MUST read, validate, derive the decision, and update processing metadata against one current file snapshot; if the review hash, gate status, or processing metadata no longer matches current gate state before the update, processing MUST be rejected without advancing the gate.
- **FR-019**: If one or more response blocks use `fix`, the system MUST continue with a fix decision containing only those finding IDs and their solution instructions.
- **FR-020**: If every response block uses `accept` or `skip`, the system MUST approve the review gate without requiring another confirmation.
- **FR-021**: The review gate terminal view MUST show only a compact summary, the review file path, and the available actions `p process` and `c cancel`.
- **FR-022**: The review gate terminal view MUST NOT render full finding details or expose the legacy per-finding terminal controls `a approve`, `f fix`, `s skip`, `e edit`, `+ add`, `A all`, and `N none`.
- **FR-023**: Non-review approval gates MUST keep their existing terminal behavior and controls.
- **FR-024**: `c cancel` from the review file gate MUST cancel the run through the existing abort behavior.
- **FR-025**: The system MUST expose these review phase labels on human-facing surfaces: `Review preview` for running review, `Review preview complete` for awaiting review-file processing, `Fixing review issues` for review fixes, and `Review fix result` for fix-review results.
- **FR-026**: The system MUST NOT add review phase labels to completed review steps or non-review steps; completed review steps MUST still render as `Review`.
- **FR-027**: CLI/AXI run and gate output MUST preserve raw status values while also exposing nullable structured fields such as `review_phase_label` and `review_file_path` when applicable. `review_phase_label` MUST be non-null for the review statuses that have labels in FR-025, and `review_file_path` MUST be non-null whenever a review handoff file exists or is recoverable for the run; otherwise each field MUST be null.
- **FR-028**: Existing automation responses using approve, fix, skip, selected finding IDs, instructions, and user-added findings MUST continue to work without editing the handoff file, and the system MUST mirror those automation decisions into the review file by updating processing metadata and response blocks for PR auditability. The mirror MUST be generated from the exact automation decision that the gate will execute, including selected IDs, instructions, and user-added findings, using the same writer/parser contract as hand-edited files. For automation decisions, the mirror write MUST succeed before the gate decision is finalized; if the mirror cannot be written, the gate MUST remain unprocessed and report an actionable error instead of recording a decision without its audit file.
- **FR-029**: Normal review and fix-review cycles MUST reuse the same review file path for the run.
- **FR-030**: When a new review result with findings is generated for the same run, the system MUST overwrite the review handoff file with the latest findings and MUST NOT preserve old user-answer blocks from a previous cycle.
- **FR-031**: If fix review has no remaining findings, the system MUST overwrite the review file with a distinct final state that preserves prior finding decisions, includes metadata, any available applied fix summary, and `No remaining review findings.`, and `p process` MUST approve directly. In this final state, preserved prior decisions are audit entries rather than latest findings, so FR-015's "every latest finding" and "no unknown finding IDs" checks apply only to the zero latest-finding set while metadata, hash, size, and parseability checks still apply.
- **FR-032**: The system MUST include the final review handoff file in the PR branch commit when the pipeline reaches push, preserving the same relative path. If processing succeeded but the final review handoff file is absent or unreadable when push prepares the PR branch commit, the system MUST regenerate the final processed-state file from persisted review decisions and fix summaries, or block the commit with an explicit audit-file error if regeneration is impossible.
- **FR-033**: The system MUST create the PR branch commit when the review handoff file is the only remaining change that needs to be included.
- **FR-034**: The system MUST NOT commit `plan.md`, `task.md`, or `tasks.md` merely because one of those files was used as a location anchor.
- **FR-035**: The system MUST keep `auto_fix.review` behavior unchanged.
- **FR-036**: The system MUST NOT introduce new pipeline step names, raw status values, database schema requirements, review history files, automatic editor opening, manual regenerate commands, or parsing of user-edited prose outside response blocks in v1.
- **FR-037**: User-facing documentation and examples MUST describe the review file handoff, phase labels, file path behavior, and preserved automation contract.

### Key Entities

- **Review Run**: A pipeline run with branch, run ID, current review step status, findings, fix rounds, and the final review handoff file.
- **Review Finding**: A review issue with an existing structured ID, severity, file context when available, issue description, explanatory context, recommended fix options, and default user-answer action.
- **Review Handoff File**: The Markdown file that contains current review findings, YAML front matter metadata, response blocks, and processing metadata for one run.
- **Response Block**: A fenced user-answer block tied to one canonical structured finding ID, containing an action and optional solution text.
- **Review Phase Label**: A human-facing label derived from the review step's current status for display in terminal, TUI, CLI/AXI, logs, docs, and PR summaries.
- **Processed Review Decision**: The outcome created from a valid handoff file or an existing automation response, either a fix decision with the exact selected finding IDs, per-finding instructions, and user-added findings that were dispatched, or an approval decision. The PR audit file mirror MUST be derived from this executed decision payload.
- **PR Audit File**: The final handoff file included in the PR branch commit as an auditable record of review issues, user decisions, mirrored automation decisions, and the final no-findings state when applicable.

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: The feature preserves one review pipeline step and existing raw status/action contracts. File processing maps back to the same approve, fix, skip, and abort outcomes already used by the gate, while `auto_fix.review` remains unchanged.
- **Isolation/User Control**: The user controls review decisions by editing the handoff file and explicitly pressing `p process`. Processing must never overwrite answers, and PR commit behavior must include only the review file plus normal pipeline changes, not anchor files chosen only for location.
- **Evidence Plan**: Verification must cover the path resolver, Markdown writer, response parser, terminal review gate controls, review phase labels, CLI/AXI structured output, stale-hash and size-bound validation, process-to-fix/approve mapping, automation-decision mirroring, fix-review final state, PR commit inclusion, documentation examples, and preservation of existing automation behavior. Full validation should include targeted package tests for changed surfaces plus the project lint/test/build checks normally required for this repository.
- **Agent/Interface Contracts**: The file handoff adds review-file and phase presentation to human and agent-facing surfaces without changing raw statuses or existing automation response commands. Review recommendations must be rendered as `Recommendation` for users while preserving existing structured finding meaning.
- **Docs/Generated Artifacts**: Documentation must be updated where users learn approval gates, review behavior, CLI/AXI output, and TUI controls. Generated skill or help content must be updated if it repeats old review-gate instructions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of review gates with findings, users see a compact terminal gate containing a findings summary, a review file path, `p process`, and `c cancel`, with no full finding details rendered in that gate view.
- **SC-002**: Users can complete the primary review-file flow by saving answers in the file and taking one terminal action to process it.
- **SC-003**: 100% of malformed handoff files covered by validation tests, including stale result hashes and size-bound violations, are rejected without advancing the gate and with an actionable error that identifies the file.
- **SC-004**: 100% of review states covered by display tests show the expected phase labels, while non-review steps and completed review steps show no extra phase label.
- **SC-005**: Existing approve, fix, skip, selected-finding, instruction, and user-added-finding automation flows continue to pass their current behavior tests.
- **SC-006**: In 100% of tested push-to-PR flows where a review handoff file exists, the final review file is included in the PR branch commit at the expected relative path.
- **SC-007**: Documentation examples for review gates, CLI/AXI output, and review phase labels match the implemented user-facing behavior.
- **SC-008**: The full repository validation suite required for release passes after the feature is implemented.

## Assumptions

- Target users are developers or coding agents operating a no-mistakes review gate from terminal, TUI, or CLI/AXI surfaces.
- The source requirement artifact is `plans/grill-me/review-file-handoff.md`; this specification converts that handoff plan into the Spec Kit `spec.md` for downstream planning.
- Version 1 supports only the review step for file-based handoff and review sub-phase labels.
- Users are comfortable editing Markdown in their editor of choice; automatic editor launching is out of scope.
- Only one current review handoff file is kept per run; separate previous review-cycle handoff history files are out of scope. The final PR audit file must still include a compact ledger of prior-cycle decisions derived from persisted review round data when those decisions would otherwise be overwritten from the current handoff file.
- `accept` and `skip` have the same operational effect in v1 and differ only for audit readability.
- Existing data persistence can continue to store review findings, selected IDs, user instructions, fix summaries, and the deterministic review-result hash without requiring a new schema for the review file path. Persisted review round data is the authoritative source for prior finding decisions in the final no-findings state and must span all review/fix-review cycles in the run.
- Reattach can recover or recompute the deterministic review file path when the original event was missed.
- Full end-to-end coverage is not required for the first pass if targeted package tests prove each changed surface and existing automation contracts remain stable.
