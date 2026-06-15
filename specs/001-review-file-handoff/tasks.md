# Tasks: Review File Handoff

**Input**: Design documents from `/specs/001-review-file-handoff/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Required. The feature changes approval-gate behavior, file parsing, DB state, IPC/AXI/TUI surfaces, PR publishing, docs, and generated agent guidance.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested as an independently reviewable increment.

## Phase 1: Setup

**Purpose**: Establish the implementation surfaces and local fixtures before changing behavior.

- [ ] T001 Create the review handoff package scaffold and package documentation in `internal/reviewhandoff/doc.go`
- [ ] T002 [P] Add valid and invalid Markdown handoff fixtures for parser/generator tests in `internal/reviewhandoff/testdata/review-handoff-fixtures.md`
- [ ] T003 [P] Add a source map of existing approval, TUI, AXI, IPC, DB, and push touch points in `specs/001-review-file-handoff/tasks.md`
- [ ] T004 [P] Confirm development validation commands from the feature quickstart in `specs/001-review-file-handoff/quickstart.md`

---

## Phase 2: Foundational

**Purpose**: Shared types, state, persistence, and IPC contracts that block all user stories.

- [ ] T005 [P] Add review phase label unit tests for running, awaiting_approval, fixing, fix_review, completed, and non-review steps in `internal/types/types_test.go`
- [ ] T006 Implement shared review phase label helpers without changing raw statuses in `internal/types/types.go`
- [ ] T007 [P] Add review handoff state validation and JSON round-trip tests in `internal/reviewhandoff/state_test.go`
- [ ] T008 Implement review handoff state structs, constants, decision structs, and validation helpers in `internal/reviewhandoff/state.go`
- [ ] T009 [P] Add DB migration and typed state helper tests for legacy and new databases in `internal/db/step_test.go`
- [ ] T010 Add the additive `review_handoff_json` column migration in `internal/db/schema.go`
- [ ] T011 Add typed `SetStepReviewHandoff`, `StepReviewHandoff`, and clear helpers in `internal/db/step.go`
- [ ] T012 [P] Add IPC JSON round-trip coverage for `phase`, `review_file`, and process-review-handoff request/response fields in `internal/ipc/protocol_test.go`
- [ ] T013 Add additive IPC fields and a process-review-handoff RPC contract in `internal/ipc/protocol.go`

**Checkpoint**: The repo has stable shared contracts for review phase labels, durable handoff state, and IPC payloads.

---

## Phase 3: User Story 1 - Review Issues In A File (Priority: P1)

**Goal**: A developer reaches a review gate, sees a compact terminal prompt with a Markdown review file path, edits the response blocks, and processes valid responses through the existing review transition.

**Independent Test**: Produce a review gate with two findings, confirm the compact terminal output points to the file and hides old review controls, edit one `fix` and one `accept`, process the file, and verify only the fixed finding reaches remediation with the custom solution.

### Tests for User Story 1

- [ ] T014 [P] [US1] Add generator tests for front matter, Issue/Context/Recommendation/User Answer sections, default actions, option 1 metadata, and no-remaining-findings output in `internal/reviewhandoff/render_test.go`
- [ ] T015 [P] [US1] Add happy-path parser tests for valid response blocks, whitespace trimming, 20 findings, and distinct accept/skip decisions in `internal/reviewhandoff/parse_test.go`
- [ ] T016 [P] [US1] Add path resolver tests for single uncommitted anchors in staged, unstaged modified, and untracked states; single latest committed anchor; fallback directory; branch slug; and relative path rendering in `internal/reviewhandoff/path_test.go`
- [ ] T017 [P] [US1] Add executor tests proving configured review auto-fix runs before handoff generation and that handoff generation plus DB persistence happen before approval events in `internal/pipeline/executor_test.go`
- [ ] T018 [P] [US1] Add TUI tests for compact review gate rendering, review file path display, and process/cancel-only action bar in `internal/tui/action_bar_test.go`
- [ ] T019 [P] [US1] Add daemon process-review-handoff happy-path tests that dispatch `RespondWithOverrides` with selected fix IDs and instructions in `internal/daemon/runinfo_test.go`
- [ ] T020 [P] [US1] Add cross-process happy-path coverage for edit-one-fix/edit-one-accept/process flow in `internal/e2e/review_handoff_test.go`

### Implementation for User Story 1

- [ ] T021 [P] [US1] Implement Markdown rendering, default action selection, recommendation option extraction, finding digesting, and generated content digesting in `internal/reviewhandoff/render.go`
- [ ] T022 [P] [US1] Implement strict happy-path response block parsing and response-to-decision conversion in `internal/reviewhandoff/parse.go`
- [ ] T023 [P] [US1] Implement anchor discovery, branch slugging, canonical path validation, and relative path display in `internal/reviewhandoff/path.go`
- [ ] T024 [US1] Generate or atomically overwrite the review handoff file after review/fix-review findings and before approval event emission in `internal/pipeline/executor.go`
- [ ] T025 [US1] Persist the current review handoff state, review cycle ID, finding digest, and generated content digest while entering review approval in `internal/pipeline/executor.go`
- [ ] T026 [US1] Add daemon-side process-review-handoff handling that validates the active run/step and dispatches existing executor responses in `internal/daemon/manager.go`
- [ ] T027 [US1] Register the process-review-handoff RPC and wire it to the run manager in `internal/daemon/daemon.go`
- [ ] T028 [US1] Store and update review file paths and phase values from live events and reattached state in `internal/tui/events.go`
- [ ] T029 [US1] Replace review-gate TUI controls with compact summary, review file path, `p process`, and `c cancel` behavior in `internal/tui/pipeline.go`
- [ ] T030 [US1] Route `p` to process-review-handoff and `c` to stale-gate-aware review cancel handling in `internal/tui/commands.go`, with tests proving cancel checks the original run/review-step identity and returns a stale-gate error without aborting a superseded run in `internal/tui/commands_test.go` or `internal/daemon/runinfo_test.go`
- [ ] T031 [US1] Keep non-review approval gates on the existing approve/fix/skip/edit/add/select controls in `internal/tui/commands.go`
- [ ] T032 [US1] Document the review file workflow, response block format, and process/cancel terminal controls in `docs/src/content/docs/guides/tui.md`

**Checkpoint**: User Story 1 is functional through the compact TUI and process path with valid handoff files.

---

## Phase 4: User Story 2 - Safely Process Edited Review Responses (Priority: P2)

**Goal**: Malformed, stale, or unsafe review files are rejected before any approval or fix action, while user edits remain intact and the compact gate stays open.

**Independent Test**: Change run metadata, delete a response block, add an unknown finding ID, or leave an unsupported action, then process and verify no executor response is dispatched and the concise validation error includes the current review file path.

### Tests for User Story 2

- [ ] T033 [P] [US2] Add parser rejection tests for duplicate fields, unknown fields, uppercase actions, nested fences, multiline continuations, duplicate IDs, missing IDs, unknown IDs, and unsupported actions in `internal/reviewhandoff/parse_test.go`
- [ ] T034 [P] [US2] Add validation tests for stale run/status/cycle/digest metadata, non-pending processed metadata, missing files, unreadable files, empty fix fallback without option 1, and prose ignored outside response blocks in `internal/reviewhandoff/validate_test.go`
- [ ] T035 [P] [US2] Add overwrite-protection tests for edited pending files and timestamped backup or stale-edit diagnostic behavior in `internal/reviewhandoff/render_test.go`
- [ ] T036 [P] [US2] Add daemon tests proving validation failure leaves the gate open, preserves file contents, and dispatches no response in `internal/daemon/runinfo_test.go`
- [ ] T037 [P] [US2] Add TUI tests for one concise validation error plus review file path after failed processing in `internal/tui/action_bar_test.go`
- [ ] T038 [P] [US2] Add malformed/stale handoff e2e coverage in `internal/e2e/review_handoff_test.go`

### Implementation for User Story 2

- [ ] T039 [US2] Implement full response block validation, metadata matching, pending-state checks, and trusted default recommendation fallback in `internal/reviewhandoff/validate.go`
- [ ] T040 [US2] Preserve edited pending files by creating a timestamped backup or returning a stale-edit diagnostic before regeneration in `internal/reviewhandoff/render.go`
- [ ] T041 [US2] Implement atomic processed metadata stamping and resolved-decision summary writes in `internal/reviewhandoff/process.go`
- [ ] T042 [US2] Persist processed decisions, processed action, processed timestamp, and decision source transactionally in `internal/db/step.go`
- [ ] T043 [US2] Enforce validation before response dispatch and keep the executor waiting after process failures in `internal/daemon/manager.go`
- [ ] T044 [US2] Surface stale-gate and validation errors without clearing user edits in `internal/tui/events.go`
- [ ] T045 [US2] Map valid no-remaining-findings handoffs to direct approval, with daemon/validator tests covering that no response blocks are required, and map valid all-accept/all-skip files to approval and valid fix files to selected finding fixes in `internal/daemon/manager.go` and `internal/daemon/runinfo_test.go`

**Checkpoint**: User Story 2 blocks unsafe processing before state changes or executor responses.

---

## Phase 5: User Story 3 - Understand Review Sub-Phases Consistently (Priority: P3)

**Goal**: Human-readable phase labels and additive review file paths appear consistently across terminal, live IPC, reattached state, AXI output, logs, docs, and tests without renaming raw statuses.

**Independent Test**: Move a review step through running, awaiting approval, fixing, fix review, and completed states, then compare `phase` and `review_file` values across IPC events, reattached run state, TUI, and `no-mistakes axi status`.

### Tests for User Story 3

- [ ] T046 [P] [US3] Add IPC event round-trip tests for additive `phase` and `review_file` fields on review events and omission on non-review events in `internal/ipc/protocol_test.go`
- [ ] T047 [P] [US3] Add daemon reattach tests proving `StepResultInfo` derives phase from raw status and loads review file from durable handoff state in `internal/daemon/runinfo_test.go`
- [ ] T048 [P] [US3] Add AXI status tests for step rows and gate output with additive `phase` and `review_file` fields in `internal/cli/axi_test.go`
- [ ] T049 [P] [US3] Add TUI rendering and log-surface audit tests for exact labels `Review preview`, `Review preview complete`, `Fixing review issues`, and `Review fix result` in `internal/tui/pipeline_test.go` and any log-emitter tests touched by review phase wording

### Implementation for User Story 3

- [ ] T050 [US3] Populate additive `phase` and `review_file` fields on live review events from committed handoff state in `internal/pipeline/executor.go`
- [ ] T051 [US3] Populate additive `phase` and `review_file` fields on reattached run state in `internal/daemon/daemon.go`
- [ ] T052 [US3] Add phase and review file fields to AXI step rows and gate output in `internal/cli/axi_render.go`
- [ ] T053 [US3] Preserve existing `no-mistakes axi respond approve|fix|skip` behavior while recording automation decision source in `internal/cli/axi_drive.go`
- [ ] T054 [US3] Render shared phase labels in pipeline rows and active review gate headers in `internal/tui/pipeline.go`
- [ ] T055 [US3] Document AXI `phase` and `review_file` output additions in `docs/src/content/docs/reference/cli.md`
- [ ] T056 [US3] Document stable raw statuses and new review phase wording in `docs/src/content/docs/reference/pipeline-steps.md`

**Checkpoint**: User Story 3 exposes consistent review phase and review file context while keeping raw automation contracts stable.

---

## Phase 6: User Story 4 - Carry Review Handoff Into The PR Audit Trail (Priority: P4)

**Goal**: The latest processed review handoff file is copied into the PR branch commit through an explicit allowlist without staging anchor files or unrelated working-tree changes.

**Independent Test**: Complete a run with a processed review file placed beside a changed `plan.md` or `tasks.md` anchor and confirm the PR branch includes only intentional pipeline artifacts plus the review file, including the review-file-only commit case.

### Tests for User Story 4

- [ ] T057 [P] [US4] Add push-step tests for copying the review file at the same relative path into the isolated work area in `internal/pipeline/steps/push_test.go`
- [ ] T058 [P] [US4] Add push-step tests for review-file-only commits, pending handoff failure, missing file failure, and no-handoff auto-fix exemption in `internal/pipeline/steps/push_test.go`
- [ ] T059 [P] [US4] Add allowlist tests proving changed anchor files, neighboring files, path traversal, and symlink escapes are not staged in `internal/pipeline/steps/push_test.go`
- [ ] T060 [P] [US4] Add PR summary tests proving processed review decisions appear in the audit trail when available in `internal/pipeline/steps/prsummary_test.go`

### Implementation for User Story 4

- [ ] T061 [US4] Implement review handoff source and destination canonicalization helpers for PR copy in `internal/reviewhandoff/path.go`
- [ ] T062 [US4] Copy the latest processed review handoff file into the isolated work area before commit decisions in `internal/pipeline/steps/push.go`
- [ ] T063 [US4] Replace broad staging with a publishable-artifact allowlist that includes intentional outputs, configured evidence, and the review file path in `internal/pipeline/steps/push.go`
- [ ] T064 [US4] Fail push/PR preparation when the latest required handoff is pending, missing, unreadable, or escapes source/work-area roots in `internal/pipeline/steps/push.go`
- [ ] T065 [US4] Include processed review decisions and decision source in PR summary generation in `internal/pipeline/steps/prsummary.go`
- [ ] T066 [US4] Document PR audit copy behavior and allowlist constraints in `docs/src/content/docs/concepts/pipeline.md`

**Checkpoint**: User Story 4 preserves review handoff audit evidence in PR commits without leaking unrelated files.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated guidance, regression coverage, and final validation across all user stories.

- [ ] T067 [P] Update troubleshooting guidance for stale, malformed, deleted, unreadable, or superseded handoff files in `docs/src/content/docs/guides/troubleshooting.md`
- [ ] T068 [P] Update the gate model documentation with file handoff process/cancel semantics in `docs/src/content/docs/concepts/gate-model.md`
- [ ] T069 Regenerate agent-facing no-mistakes skill guidance after AXI workflow text changes in `skills/no-mistakes/SKILL.md`
- [ ] T070 Run `gofmt` on changed Go files listed by `git diff --name-only` from repository root `.`
- [ ] T071 Run targeted Go tests from `specs/001-review-file-handoff/quickstart.md`
- [ ] T072 Run `go test -race ./...` from repository root `.`
- [ ] T073 Run `make lint` from repository root `.`
- [ ] T074 Run docs build or record the tooling skip reason for `docs/src/content/docs/guides/tui.md`
- [ ] T075 Run manual smoke flows for valid, malformed, AXI compatibility, and PR audit scenarios from `specs/001-review-file-handoff/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on User Story 1 parser/generator/process scaffolding.
- **User Story 3 (Phase 5)**: Depends on Foundational and can proceed after handoff state is persisted.
- **User Story 4 (Phase 6)**: Depends on processed handoff state from User Stories 1 and 2.
- **Polish (Final Phase)**: Depends on all implemented user stories.

### User Story Dependencies

- **US1 (P1)**: No dependency on later stories; provides the usable review file handoff MVP.
- **US2 (P2)**: Builds on US1 processing and hardens validation/failure behavior.
- **US3 (P3)**: Can run partly in parallel with US2 after durable state exists, but final verification needs US1-generated review files.
- **US4 (P4)**: Requires processed handoff state from US1/US2 and should run after push-step behavior is mapped.

### Parallel Opportunities

- Setup fixture and documentation checks can run in parallel after T001.
- Foundational tests T005, T007, T009, and T012 can be written in parallel because they touch different packages.
- US1 test tasks T014 through T020 can run in parallel after Foundational.
- US2 test tasks T033 through T038 can run in parallel after US1 parser/generator scaffolding.
- US3 IPC, daemon, AXI, and TUI tests T046 through T049 can run in parallel.
- US4 push and PR summary tests T057 through T060 can run in parallel, then converge in `internal/pipeline/steps/push.go`.
- Docs tasks T067 and T068 can run in parallel with final validation after behavior is stable.

---

## Parallel Example: User Story 1

```bash
# Independent test-writing slices:
Task: "T014 generator tests in internal/reviewhandoff/render_test.go"
Task: "T015 parser tests in internal/reviewhandoff/parse_test.go"
Task: "T016 path resolver tests in internal/reviewhandoff/path_test.go"
Task: "T018 TUI compact gate tests in internal/tui/action_bar_test.go"

# Independent implementation slices after tests are in place:
Task: "T021 renderer in internal/reviewhandoff/render.go"
Task: "T022 parser in internal/reviewhandoff/parse.go"
Task: "T023 path resolver in internal/reviewhandoff/path.go"
```

## Parallel Example: User Story 3

```bash
Task: "T050 live IPC event fields in internal/pipeline/executor.go"
Task: "T051 reattached run state in internal/daemon/daemon.go"
Task: "T052 AXI status rendering in internal/cli/axi_render.go"
Task: "T054 TUI phase labels in internal/tui/pipeline.go"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete User Story 1.
3. Validate with `go test ./internal/reviewhandoff ./internal/pipeline ./internal/tui ./internal/daemon` and the valid handoff smoke flow from `specs/001-review-file-handoff/quickstart.md`.
4. Stop and review the MVP before adding US2 validation hardening, US3 surface consistency, and US4 PR audit publishing.

### Incremental Delivery

1. US1 delivers the file handoff and process path.
2. US2 makes processing safe against malformed and stale edits.
3. US3 makes all human and automation surfaces consistent.
4. US4 makes the handoff durable in PR audit history.
5. Final phase validates docs, generated skill guidance, lint, race tests, and smoke flows.

### Risk Notes

- Keep raw step statuses, raw step names, and existing `axi respond approve|fix|skip` commands unchanged.
- Do not parse user decisions from Markdown prose outside `no-mistakes-review-response` fences.
- Do not stage `plan.md`, `tasks.md`, neighboring files, or unrelated user changes merely because they are near the review file.
- Preserve automatic review auto-fix precedence; generate handoffs only when the existing executor reaches a human decision point.
