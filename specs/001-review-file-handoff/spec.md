# Feature Specification: Review File Handoff

**Feature Branch**: `001-review-file-handoff`  
**Created**: 2026-06-15  
**Status**: Draft  
**Input**: User description: "read requirement in create detail spec for review-file-handoff, deeply understand this new feature, scout source for know the context, spawn sub-agents for help, scout source code to know the context, create detail spec"

## Clarifications

### Session 2026-06-15

- Q: What exact fenced response block schema should the parser accept for each finding? → A: A fenced block tagged `no-mistakes-review-response` with `action: fix|accept|skip` and `solution: <one-line text>` fields.
- Q: Which finding identifier should each response block use when matching edited answers to the latest findings? → A: Use the existing normalized `Finding.ID` value persisted with the latest finding.
- Q: How should processed metadata appear before and after the user processes the review file? → A: Render `processed_action: pending` with an empty processed timestamp initially, then overwrite both after successful processing.
- Q: After validation fails during `p process`, what should the terminal gate display next? → A: Keep the compact gate open, show one concise validation error plus the review file path, and keep only process/cancel actions.
- Q: What privacy or filesystem protection should generated review handoff files require? → A: Write only inside the project checkout with normal repository file permissions and no additional redaction.
- Q: What performance target should processing a valid 20-finding handoff meet on a local checkout? → A: No explicit latency target beyond correctly processing 20 findings.
- Q: When a `fix` response has no usable solution text, what should processing do? → A: Use recommendation option 1 as the fix instruction.
- Q: Which anchor filenames should the review file path resolver recognize? → A: Recognize `plan.md` and `tasks.md`.
- Q: Which automation surfaces must include the additive `phase` and `review_file` values in the first release? → A: Live gate events, reattached run state, and `axi status` run/gate output.
- Q: Which labels are canonical for remediation guidance across the file, terminal, docs, and review summaries? → A: Use `Recommendation` for agent guidance and `Solution` for user-authored fix text.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review Issues In A File (Priority: P1)

A developer reaches the review gate and receives a compact terminal summary that points to one Markdown review file. The developer opens that file, reads all issues and recommendations in a review-friendly format, edits the response blocks, and returns to the terminal to process or cancel the handoff.

**Why this priority**: This is the core user value: review detail and user answers move out of the constrained terminal view into a durable, editable handoff surface.

**Independent Test**: Can be fully tested by producing a review gate with multiple findings, confirming the terminal shows only the summary/path/actions, editing the file, and processing the responses without using the old per-finding terminal controls.

**Acceptance Scenarios**:

1. **Given** a review gate with two findings, **When** the review phase pauses for user input, **Then** the terminal shows the review phase label, finding counts, a review file path, and only `p process` plus `c cancel` as review-gate actions.
2. **Given** the generated review file, **When** the developer opens it, **Then** every latest finding is represented with `Issue`, `Context`, `Recommendation`, and `User Answer` sections plus one editable `no-mistakes-review-response` block keyed by the finding's normalized `Finding.ID`.
3. **Given** one response block marked `fix` with a custom solution and one marked `accept`, **When** the developer presses `p process`, **Then** only the fixed finding is sent for remediation with the custom solution, and the accepted finding is not sent for remediation.

---

### User Story 2 - Safely Process Edited Review Responses (Priority: P2)

A developer can edit the Markdown review file confidently because processing validates the file before acting and preserves the developer's answers when validation fails.

**Why this priority**: A file handoff is only trustworthy if malformed edits cannot silently produce the wrong approval or fix request.

**Independent Test**: Can be tested by changing the review file metadata, deleting a response block, adding an unknown finding ID, and confirming processing blocks with a short error while the review gate remains open.

**Acceptance Scenarios**:

1. **Given** a review file whose run identifier or review-cycle metadata does not match the active gate, **When** the developer presses `p process`, **Then** processing is blocked, the terminal shows a concise error containing the actual versus expected run/status/cycle values plus the current valid review file path or regeneration state, and the gate remains open.
2. **Given** a review file missing a response block for a latest finding, **When** the developer presses `p process`, **Then** processing is blocked and no fix or approval action is sent.
3. **Given** a valid review file where all findings are marked `accept` or `skip`, **When** the developer presses `p process`, **Then** the review gate approves directly without an extra confirmation.

---

### User Story 3 - Understand Review Sub-Phases Consistently (Priority: P3)

A developer and automation consumer see the same human-readable review sub-phase labels across terminal, automation output, logs, PR summaries, and documentation, while machine-readable step statuses remain stable.

**Why this priority**: The feature changes how review is presented; inconsistent wording would make the handoff hard to trust and would break downstream automation if raw statuses were renamed.

**Independent Test**: Can be tested by moving a review step through preview, paused preview, fixing, fix review, and completed states, then comparing labels across all user-facing surfaces.

**Acceptance Scenarios**:

1. **Given** the review step is running, **When** any human-facing surface renders the active phase, **Then** it shows `Review preview`.
2. **Given** the review step is awaiting approval, **When** terminal and automation surfaces render the gate, **Then** they show `Review preview complete` while the raw status remains `awaiting_approval`.
3. **Given** the review step is in fix review, **When** terminal and automation surfaces render the gate, **Then** they show `Review fix result` while the raw status remains `fix_review`.

---

### User Story 4 - Carry Review Handoff Into The PR Audit Trail (Priority: P4)

A reviewer of the final PR can see the latest review issue file that shaped the pipeline's approval or fix actions.

**Why this priority**: The handoff file is the durable audit artifact for human review decisions, so it must survive beyond the local terminal session.

**Independent Test**: Can be tested by completing a run with a review file and confirming the PR branch includes the latest review file at the expected relative path, even when it is the only new file left to commit.

**Acceptance Scenarios**:

1. **Given** a review file exists when the pipeline reaches the push phase, **When** the PR branch commit is created, **Then** the latest review file is included at the same relative path.
2. **Given** the review file was placed next to a `plan.md` or `tasks.md` anchor, **When** the PR branch commit is created, **Then** only intentional pipeline changes and the review file are included; the anchor file is not committed merely because it was used for placement.

### Edge Cases

- Review processing is attempted after the run is interrupted, canceled, or superseded by a newer push.
- The review file is deleted, unreadable, moved, or no longer matches the active review gate.
- The review file header is missing, duplicated, malformed, or contains the wrong run, step, branch, or status.
- A response block is missing, duplicated, has an unknown finding ID, or uses an unsupported action.
- A `fix` response has an empty solution, comment-only solution, multi-line solution, or malformed solution field.
- No `plan.md` or `tasks.md` anchor exists, or multiple anchors exist, so the system must use the fallback issue directory.
- A fix-review pass has no remaining findings and still needs a final review file state plus a direct process-to-approve path.
- Automation consumers continue using existing approve/fix/skip commands and raw statuses while also seeing the added phase and review-file data.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create or update one Markdown review handoff file whenever the review step produces a human decision point.
- **FR-002**: The review handoff file MUST contain metadata identifying the run, review step, current review-gate status, branch, review cycle identifier, deterministic finding-set digest, processed timestamp, and processed action; generated files MUST render `processed_action: pending` with an empty processed timestamp, then overwrite both after successful processing.
- **FR-003**: The review handoff file MUST list every latest review finding exactly once with `Issue`, `Context`, `Recommendation`, and `User Answer` sections; `Recommendation` is the canonical label for agent guidance and `Solution` is the canonical label for user-authored fix text.
- **FR-004**: Recommendations MUST present one or two concrete options, with option 1 treated as the default fix choice when the user leaves a fix solution empty; for every finding whose generated response can default to `fix`, option 1 MUST be machine-detectable and non-empty, and processing MUST reject an empty-solution `fix` response when no valid option 1 exists.
- **FR-005**: User answers MUST be read only from fenced response blocks whose opening info string is exactly `no-mistakes-review-response <Finding.ID>`, where `<Finding.ID>` is the latest finding's normalized persisted ID; each block MUST contain line-oriented `action: fix|accept|skip` and `solution: <one-line text>` fields, and the parser MUST require the exact lowercase tag and field names, exactly one finding ID token after the tag, exactly one `action` line, exactly one `solution` line, deterministic whitespace trimming around field values, rejection of duplicate or unknown fields, rejection of nested response fences or multi-line continuations, and prose outside those blocks MUST NOT affect processing.
- **FR-006**: Supported user-answer actions MUST be `fix`, `accept`, and `skip`.
- **FR-007**: Default response actions MUST map existing review intent as follows: automatically fixable findings default to `fix`, human-judgment findings default to `accept`, and informational findings default to `skip`.
- **FR-008**: When processing a valid file, any response block marked `fix` MUST send only that finding for remediation, along with any non-empty `solution` text as the user's instruction for that finding; if the `solution` is empty or comment-only, the system MUST use recommendation option 1 from the active latest finding model or validated handoff metadata as the fix instruction, and MUST NOT read fallback instructions from editable human-readable Markdown prose outside the response block.
- **FR-009**: When processing a valid file where all response blocks are `accept` or `skip`, the system MUST approve the review gate without requiring an additional confirmation.
- **FR-010**: `accept` and `skip` MUST remain distinct in the review file for audit readability, even though both mean "do not fix this finding" in the first version; processed per-finding decisions MUST preserve the distinct `accept` versus `skip` action in the latest handoff file's resolved-decision summary and in the PR audit copy, even after later review or fix-review regeneration overwrites old answer blocks.
- **FR-011**: Processing MUST validate that the file exists, is readable, has the expected metadata, matches the active run, review step, review status, review cycle identifier, and deterministic finding-set digest, includes exactly one response keyed by every latest normalized `Finding.ID`, includes no unknown finding IDs, uses only supported actions, has parseable solution fields, and still shows `processed_action: pending` with an empty processed timestamp before any approval or fix response is sent.
- **FR-012**: If processing validation fails, the system MUST block the action, keep the compact review gate open with only process/cancel actions, show one concise terminal error with the review file path, and preserve the user's file contents; successful processing MUST be committed as a transaction whose ordered commit points are validation, atomic processed-metadata update in the handoff file, response dispatch, and then live event/reattach/`axi status` state exposure, and any partial failure MUST leave the file and automation state on the same last committed processed state.
- **FR-013**: Pressing cancel from the review handoff gate MUST abort the active run through the existing user-abort path only when the gate's original run and review step identity still match the currently active gate; process and cancel actions MUST no-op with a stale-gate error if that run is interrupted, canceled, superseded, or no longer awaiting the same review decision.
- **FR-014**: The review gate terminal view MUST NOT render full review issue details; it MUST show only the review phase label, finding summary, review file path, and the process/cancel actions.
- **FR-015**: The review gate MUST hide the old review-specific terminal controls for approve, fix, skip, edit, add, select all, select none, and per-finding toggling.
- **FR-016**: Non-review approval gates MUST keep their existing terminal behavior and controls unless they explicitly adopt the file handoff in a later feature.
- **FR-017**: The system MUST use one current review handoff file per run across normal review and fix-review cycles; each review-cycle transition MUST generate or atomically overwrite the file, persist the current review-file reference plus phase inputs, and only then emit live events or expose reattached/`axi status` state, with the prior committed file reference retained if any step in that sequence fails.
- **FR-018**: When a new review or fix-review result is generated, the system MUST overwrite the current handoff file with the latest findings and MUST NOT preserve old answer blocks from a previous review result in the current file; before overwriting a pending file, the system MUST detect whether user-editable response content changed since generation and MUST either preserve the superseded pending file as a timestamped backup or block regeneration with a concise stale-edit diagnostic rather than silently discarding user-authored answers.
- **FR-019**: When a fix-review result has no remaining findings, the handoff file MUST show a final state including metadata, any available applied-fix summary, a clear "no remaining review findings" message, and a compact resolved-decision summary for the prior cycle that lists prior finding IDs, action taken, user-authored solution or default recommendation used, processed action, processed timestamp, and whether the decision came from the file handoff, automation response, or no-handoff auto-fix path.
- **FR-020**: Processing a valid no-remaining-findings handoff file MUST approve the review gate directly.
- **FR-021**: The review file path resolver MUST first look for exactly one changed `plan.md` or `tasks.md` file among uncommitted changed files in the project checkout, including staged, modified, and untracked files.
- **FR-022**: If no uncommitted anchor exists, the resolver MUST look for exactly one changed `plan.md` or `tasks.md` file in the latest committed change under review.
- **FR-023**: If zero or multiple anchors exist, the resolver MUST place the review file in `.no-mistakes/issues/<branch-slug>/`.
- **FR-024**: Review file names MUST follow the format `review-issues-<run-short-id>.md`.
- **FR-025**: User-facing paths SHOULD be shown relative to the project checkout when the path is inside that checkout.
- **FR-026**: The system MUST expose the review file path to live gate events, reattached terminal sessions, reattached run state, and `axi status` run/gate output even when the original live event was missed; the authoritative source MUST be the current review cycle's persisted review-file reference and phase inputs, or a deterministic derivation from persisted run/step state, and the first version MAY add backward-compatible metadata fields to existing run or step records but MUST NOT require an incompatible run-history migration.
- **FR-027**: The system MUST expose additive human-readable `phase` and `review_file` values without renaming raw statuses, using this compatibility contract: live review gate events carry top-level `phase` and `review_file` values when a review file is known; reattached run state carries the same additive values on the review step result; `axi status` run step rows expose additive `phase` and `review_file` columns for review steps; `axi status` gate output exposes additive `phase` and `review_file` fields beside the existing `step`, `status`, `summary`, `risk`, `findings`, and help fields; non-review steps omit both fields; existing raw step names, raw statuses, finding IDs, response commands, and finding row fields remain unchanged.
- **FR-028**: Review phase labels MUST be exactly: `Review preview` while review is running, `Review preview complete` while review awaits approval, `Fixing review issues` while review is fixing, and `Review fix result` during fix review.
- **FR-029**: Completed review steps MUST continue to render as `Review` without a sub-phase label.
- **FR-030**: Non-review steps MUST NOT receive review phase labels.
- **FR-031**: Human-facing terminal titles, active-step headers, action-bar prompts, automation output, PR summaries, logs, docs, and tests MUST use the shared review phase wording where applicable.
- **FR-032**: Raw step names, raw step statuses, step order, existing automation response commands, and existing approval actions MUST remain compatible with current users and automation; during review and fix-review gates, terminal users MUST use the file handoff `process`/`cancel` path, while `axi respond --action approve|fix|skip`, yolo auto-resolution, and direct IPC `respond` calls MUST remain accepted and MUST map to the same review-gate decision transition, using explicit command payloads rather than parsing unvalidated Markdown prose.
- **FR-033**: The review handoff behavior MUST NOT change automatic review auto-fix behavior configured for review.
- **FR-034**: The first version MUST NOT add new pipeline steps, rename existing step statuses, or require a persisted run-history format change.
- **FR-035**: When the pipeline prepares the PR branch commit, the latest review handoff file MUST be copied into the isolated pipeline work area at the same normalized repository-relative path recorded when the file was generated and included in the commit; the push or PR preparation phase MUST NOT re-run anchor discovery in the isolated work area and MUST validate that the persisted relative path stays inside both the checkout root and isolated work-area root before copying.
- **FR-036**: If the review handoff file is the only remaining change to publish, the system MUST still create the commit so the PR includes the audit file.
- **FR-037**: Using `plan.md` or `tasks.md` as a placement anchor MUST NOT by itself cause that anchor file, neighboring files, or unrelated working-tree changes to be committed; anchor use is placement-only, and PR preparation MUST stage or copy only intentional pipeline outputs plus the normalized review-file relative path through an explicit publishable-artifact allowlist.
- **FR-038**: Documentation and user-facing examples MUST be updated to describe the review handoff file, process/cancel actions, response block format, `Recommendation` and `Solution` labels, phase labels, and automation output additions.
- **FR-039**: Generated review handoff files MUST be written only inside the project checkout or isolated pipeline work area, use normal repository file permissions, and perform no additional redaction in the first version; anchor files and destination paths MUST be resolved with symlink-aware canonicalization, anchor candidates MUST be regular files, and the final write or copy target MUST be revalidated as inside the intended checkout or isolated work-area root immediately before filesystem mutation.

### Key Entities *(include if feature involves data)*

- **Review Handoff File**: The current Markdown file for a run's review decision point, containing metadata, finding summaries, recommendations, response blocks, and processed metadata that starts as pending and is overwritten after processing.
- **Review Finding**: A single review issue with severity, optional location, issue title, detailed context, recommendation text, action intent, and existing normalized `Finding.ID` stable identifier.
- **Response Block**: The editable fenced `no-mistakes-review-response` block for one finding, keyed by normalized `Finding.ID`, that records the user's selected `action` and optional `solution` guidance.
- **Review Phase Label**: A human-readable presentation label derived from the review step's existing status without changing the underlying status.
- **Review Cycle**: One normal review or fix-review result that replaces the current handoff file with the latest finding state.
- **PR Audit Copy**: The review handoff file as included in the PR branch commit so reviewers can inspect the decisions that informed the run.

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: The feature preserves `git push no-mistakes` gate meaning by keeping the same review step, step order, raw statuses, approval actions, and pass/fail semantics. The handoff file changes how review decisions are presented and collected, not what a passed gate means. Terminal `p process`, reattached sessions, AXI responses, yolo auto-resolution, and direct IPC responses MUST converge on one review-gate decision transition; the PR audit data MUST identify whether that transition came from a processed handoff file, an explicit automation response, or an existing no-handoff auto-fix path.
- **Isolation/User Control**: User edits happen in the review handoff file. Processing validates those edits before acting, preserves user answers on failure, and uses the existing abort path for cancellation. The PR audit copy is included through the isolated pipeline work area without causing unrelated anchor files to be committed.
- **Evidence Plan**: Verification must cover path resolution, Markdown generation, response parsing, validation failures, terminal summary behavior, process/cancel actions, phase labels, reattach behavior, automation output, PR audit inclusion, docs examples, and unchanged automatic review auto-fix behavior.
- **Agent/Interface Contracts**: Existing raw statuses and response commands remain stable. New `phase` and `review_file` fields are additive on live gate events, reattached run state, and `axi status` run/gate output. Review issue wording uses `Recommendation` for agent guidance and `Solution` for user-authored fix text while preserving the existing structured review concept of an actionable fix recommendation.
- **Docs/Generated Artifacts**: TUI, automation, and pipeline-step documentation must be updated because the review decision workflow, labels, and examples are user-visible.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of review gates with findings, the developer can identify the review file path and available review actions from a compact terminal summary without scrolling through issue details.
- **SC-002**: A valid handoff file containing at least 20 findings can be processed in one terminal action, with every `fix`, `accept`, and `skip` response mapped correctly; no separate latency SLA is required for the first version.
- **SC-003**: 100% of malformed or stale handoff files are rejected before any approval or fix action is sent, and user-written answer text remains intact after rejection.
- **SC-004**: Review phase labels appear consistently across all agreed human-facing surfaces while raw automation statuses remain unchanged.
- **SC-005**: 100% of PR branch commits created after a review handoff include the latest review file whenever the file exists and is inside the project checkout, and PR preparation MUST fail before commit if a review gate has passed but the latest required handoff file still has `processed_action: pending`, except for explicitly documented no-handoff automatic auto-fix paths.
- **SC-006**: Existing automated review auto-fix scenarios continue to behave as before, with no new manual handoff required for automatic review fixes.

## Assumptions

- The first version uses a Markdown file handoff only; it does not auto-open an editor or add an in-terminal editor replacement.
- Only the latest review result matters in the active handoff file; preserving historical review-cycle files is out of scope for this version.
- The project checkout is the user-visible location for handoff editing, while the pipeline may use an isolated work area for later commit and PR publishing.
- Response blocks are the only parsed source of user decisions; surrounding prose is for human readability.
- Automation users may continue using existing response commands, and the new review file path is additional context rather than a required replacement for all automation flows.
- Processing correctness for 20 findings is the release target; benchmark-style latency coverage is out of scope for the first version.
