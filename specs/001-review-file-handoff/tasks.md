# Tasks: Review File Handoff

**Input**: Design documents from `/specs/001-review-file-handoff/`
**Prerequisites**: `specs/001-review-file-handoff/plan.md`, `specs/001-review-file-handoff/spec.md`, `specs/001-review-file-handoff/research.md`, `specs/001-review-file-handoff/data-model.md`, `specs/001-review-file-handoff/contracts/`

**Tests**: Tests or reviewer-visible evidence are REQUIRED for code changes. Add or update targeted `_test.go` coverage before implementation, then run the focused package checks and full validation commands listed below.

**Organization**: Tasks are grouped by user story to keep each story independently implementable and testable.

**Current Worktree Note**: This branch already contains substantial review-file-handoff implementation across `internal/reviewhandoff`, `internal/pipeline`, `internal/tui`, `internal/ipc`, `internal/cli`, `internal/pipeline/steps`, docs, and `skills/no-mistakes`. Use this file as a coverage and verification checklist for the next pass, not as proof that all listed items are still unstarted.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files or has no dependency on incomplete tasks
- **[Story]**: User-story task label such as `[US1]`, `[US2]`, `[US3]`, or `[US4]`
- Every task includes exact file paths or package paths

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm the local implementation surface and validation commands before edits.

- [ ] T001 Review feature requirements in `specs/001-review-file-handoff/spec.md`, `specs/001-review-file-handoff/plan.md`, and `specs/001-review-file-handoff/contracts/review-handoff-file.md`
- [ ] T002 Confirm current worktree state and unrelated edits before implementation using `AGENTS.md`, `brain/Gotchas.md`, and `specs/001-review-file-handoff/tasks.md`
- [ ] T003 [P] Inventory existing review, approval, and gate tests in `internal/pipeline/executor_approval_test.go`, `internal/pipeline/steps/review_test.go`, `internal/tui/findings_test.go`, and `internal/tui/action_bar_test.go`
- [ ] T004 [P] Inventory existing AXI, IPC, and push tests in `internal/cli/axi_drive_test.go`, `internal/ipc/protocol_test.go`, `internal/ipc/rpc_test.go`, and `internal/pipeline/steps/push_test.go`
- [ ] T005 [P] Confirm `gopkg.in/yaml.v3` is already available and no new dependency is needed in `go.mod`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared review handoff package contract used by every story.

**Critical**: No user story implementation should begin until this phase is complete.

- [ ] T006 [P] Add shared model contract tests for metadata, finding entries, response blocks, processed decisions, and audit file data in `internal/reviewhandoff/model_test.go`
- [ ] T007 [P] Add bounded file I/O and atomic rename tests for review handoff writes in `internal/reviewhandoff/io_test.go`
- [ ] T008 Implement shared model structs and constants in `internal/reviewhandoff/model.go`
- [ ] T009 Implement bounded read/write and temp-file rename helpers in `internal/reviewhandoff/io.go`
- [ ] T010 Add package overview and invariants for the review handoff boundary in `internal/reviewhandoff/doc.go`
- [ ] T011 Run foundational package tests with `go test ./internal/reviewhandoff`

**Checkpoint**: The shared package compiles, and user-story tasks can start.

---

## Phase 3: User Story 1 - Review Issues In A File (Priority: P1, MVP)

**Goal**: Generate one current Markdown review handoff file with latest findings and show a compact terminal review gate with the file path.

**Independent Test**: Run a review gate with findings and confirm the TUI shows a compact summary, relative review file path, `p process`, and `c cancel`, while the file contains each finding with `Issue`, `Context`, `Recommendation`, and `User Answer`.

### Tests for User Story 1

- [ ] T012 [P] [US1] Add path resolver tests for anchor precedence, fallback path, branch slug, traversal, absolute paths, `.git`, symlink escape, and anchor suppression by resolved path in `internal/reviewhandoff/path_test.go`
- [ ] T013 [P] [US1] Add writer and hash tests for YAML front matter, severity summary, finding sections, default response actions, missing IDs, duplicate IDs, and deterministic hash inputs in `internal/reviewhandoff/writer_test.go`
- [ ] T014 [P] [US1] Add executor test proving a review handoff file is written before the review gate event is emitted in `internal/pipeline/executor_review_handoff_test.go`
- [ ] T015 [P] [US1] Add TUI review-file gate rendering tests for compact summary, relative file path, and hidden legacy controls in `internal/tui/review_file_gate_test.go`

### Implementation for User Story 1

- [ ] T016 [P] [US1] Implement safe review file path resolution and anchor selection in `internal/reviewhandoff/path.go`
- [ ] T017 [P] [US1] Implement canonical review-result hash computation in `internal/reviewhandoff/hash.go`
- [ ] T018 [US1] Implement Markdown handoff writer for front matter, summary, finding sections, recommendation options, and response blocks in `internal/reviewhandoff/writer.go`
- [ ] T019 [US1] Adapt current review findings into handoff entries and default response actions in `internal/pipeline/findings.go`
- [ ] T020 [US1] Write or overwrite the review handoff file after review findings are produced and before gate emission in `internal/pipeline/executor.go`
- [ ] T021 [US1] Add nullable review file path to review-capable IPC payloads in `internal/ipc/protocol.go`
- [ ] T022 [US1] Populate the review file path on review gate events and recoverable review run snapshots in `internal/ipc/server.go`
- [ ] T023 [US1] Store review file path from IPC events in the TUI model in `internal/tui/events.go`
- [ ] T024 [US1] Render compact review gate summary and relative review file path in `internal/tui/review.go`
- [ ] T025 [US1] Restrict review-file gate key bindings to `p process` and `c cancel` in `internal/tui/keys.go`
- [ ] T026 [US1] Update action bar assertions for review-file gates and non-review gates in `internal/tui/action_bar_test.go`
- [ ] T027 [US1] Run US1 checks with `go test ./internal/reviewhandoff ./internal/pipeline ./internal/ipc ./internal/tui`

**Checkpoint**: User Story 1 is functional and independently testable.

---

## Phase 4: User Story 2 - Process File Answers Into Review Decisions (Priority: P1)

**Goal**: Parse saved response blocks, validate them against live review gate state, and advance the existing gate with the correct approve or fix decision.

**Independent Test**: Edit response blocks in the handoff file, press `p process`, and confirm fix responses select only those findings while all accept/skip responses approve the review gate; malformed files keep the gate open with an actionable error.

### Tests for User Story 2

- [ ] T028 [P] [US2] Add response parser tests for fenced `no-mistakes-response` blocks, ignored prose, valid actions, malformed YAML, multiline `solution:`, comment-only solution lines, and 16 KiB solution bounds in `internal/reviewhandoff/parser_test.go`
- [ ] T029 [P] [US2] Add validator tests for missing blocks, duplicate IDs, unknown IDs, invalid actions, stale hashes, status mismatch, processed metadata rejection, file size limit, and zero-latest-finding final state in `internal/reviewhandoff/validator_test.go`
- [ ] T030 [P] [US2] Add byte-snapshot race and metadata-only update tests in `internal/reviewhandoff/metadata_update_test.go`
- [ ] T031 [P] [US2] Add executor processing tests for fix subset, all accept/skip approval, empty fix solution defaulting, validation failure staying open, and actionable error preservation in `internal/pipeline/executor_review_process_test.go`
- [ ] T032 [P] [US2] Add review-only IPC process request tests in `internal/ipc/review_process_test.go`
- [ ] T033 [P] [US2] Add TUI `p process` and validation error rendering tests in `internal/tui/review_file_gate_test.go`

### Implementation for User Story 2

- [ ] T034 [P] [US2] Implement response block scanner and YAML parser in `internal/reviewhandoff/parser.go`
- [ ] T035 [US2] Implement live-state validation for metadata, hash, finding coverage, actions, parseability, and size limits in `internal/reviewhandoff/validator.go`
- [ ] T036 [US2] Implement byte-snapshot processing metadata update in `internal/reviewhandoff/metadata_update.go`
- [ ] T037 [US2] Implement file-to-executor decision derivation for approve and fix outcomes in `internal/reviewhandoff/decision.go`
- [ ] T038 [US2] Add review-only process request and response payloads to `internal/ipc/protocol.go`
- [ ] T039 [US2] Implement process-review RPC client and server handling in `internal/ipc/client.go` and `internal/ipc/server.go`
- [ ] T040 [US2] Wire process-review RPC into the waiting review gate and existing approval action flow in `internal/pipeline/executor.go`
- [ ] T041 [US2] Preserve review file validation errors on review gate state for subsequent rendering in `internal/pipeline/executor.go`
- [ ] T042 [US2] Delimit non-empty file `solution:` text as untrusted per-finding data in the fixer prompt in `internal/pipeline/steps/review.go`
- [ ] T043 [US2] Send `p process` and `c cancel` actions from the TUI to the daemon in `internal/tui/keys.go` and `internal/tui/events.go`
- [ ] T044 [US2] Render review file validation failure summary and first actionable error in `internal/tui/review.go`
- [ ] T045 [US2] Run US2 checks with `go test ./internal/reviewhandoff ./internal/pipeline ./internal/ipc ./internal/tui ./internal/pipeline/steps`

**Checkpoint**: User Story 2 works independently and preserves the existing review gate action domain.

---

## Phase 5: User Story 3 - Understand Review Sub-Phases Consistently (Priority: P2)

**Goal**: Expose consistent human review phase labels across TUI, IPC, AXI, logs, and summaries while preserving raw statuses.

**Independent Test**: Move review through `running`, `awaiting_approval`, `fixing`, `fix_review`, and `completed`, then confirm labels match the spec while raw status fields remain unchanged.

### Tests for User Story 3

- [ ] T046 [P] [US3] Add review phase label mapping tests for review and non-review statuses in `internal/reviewhandoff/phase_label_test.go`, and add review log wording regression coverage in `internal/pipeline/executor_logging_test.go`
- [ ] T047 [P] [US3] Add IPC serialization tests for nullable `review_phase_label` and `review_file_path` fields in `internal/ipc/protocol_test.go`
- [ ] T048 [P] [US3] Add TUI pipeline label tests for running, awaiting approval, fixing, fix review, completed review, and non-review steps in `internal/tui/pipeline_test.go`
- [ ] T049 [P] [US3] Add AXI output tests for raw status preservation plus nullable review fields in `internal/cli/axi_review_fields_test.go`

### Implementation for User Story 3

- [ ] T050 [US3] Implement review phase label mapper in `internal/reviewhandoff/phase_label.go`
- [ ] T051 [US3] Add nullable review phase label and review file path fields to IPC structs in `internal/ipc/protocol.go`
- [ ] T052 [US3] Populate review phase labels in IPC server snapshots and streams in `internal/ipc/server.go`
- [ ] T053 [US3] Store review phase labels in the TUI model from daemon events in `internal/tui/events.go`
- [ ] T054 [US3] Render review phase labels without changing completed review labels in `internal/tui/pipeline.go` and `internal/tui/review.go`
- [ ] T055 [US3] Expose review phase label and review file path through AXI query data in `internal/cli/axi_query.go` and `internal/cli/axi.go`
- [ ] T056 [US3] Render review phase label and review file path in AXI text and structured output in `internal/cli/axi_render.go`
- [ ] T057 [US3] Run US3 checks with `go test ./internal/reviewhandoff ./internal/ipc ./internal/tui ./internal/cli`

**Checkpoint**: User Story 3 works independently without changing raw statuses.

---

## Phase 6: User Story 4 - Preserve Automation And PR Auditability (Priority: P2)

**Goal**: Keep legacy approve/fix/skip automation working while mirroring executed decisions into the review file and committing the final audit file to the PR branch.

**Independent Test**: Drive review decisions through existing automation responses, let the run reach push, and confirm the final review audit file is committed at the deterministic path even when it is the only remaining change and anchor files are not staged only because they selected the directory.

### Tests for User Story 4

- [ ] T058 [P] [US4] Add automation mirror validation tests for approve, skip, fix selected IDs, per-finding instructions, user-added findings, and same-parser/validator rejection before gate advancement or PR audit eligibility in `internal/pipeline/executor_review_mirror_test.go`
- [ ] T059 [P] [US4] Add audit renderer and regeneration tests for prior decisions, applied fix summaries, and `No remaining review findings.` in `internal/reviewhandoff/audit_test.go`
- [ ] T060 [P] [US4] Add final no-findings review-step tests in `internal/pipeline/steps/review_test.go`
- [ ] T061 [P] [US4] Add push tests for audit-file-only commits, deterministic audit path inclusion, ignored path behavior, missing audit regeneration, and anchor suppression in `internal/pipeline/steps/push_test.go`
- [ ] T062 [P] [US4] Add AXI legacy response regression tests for approve, fix, skip, selected IDs, instructions, and user-added findings in `internal/cli/axi_drive_test.go`
- [ ] T063 [P] [US4] Add or update `auto_fix.review` regression coverage in `internal/pipeline/executor_autofix_test.go`, and add tagged daemon/TUI/AXI/git review-file journey coverage in `internal/e2e/review_file_handoff_test.go` if unit tests cannot prove the cross-process flow

### Implementation for User Story 4

- [ ] T064 [US4] Implement automation decision mirroring helpers in `internal/reviewhandoff/mirror.go`
- [ ] T065 [US4] Mirror exact automation decisions into the review handoff file before review gate advancement in `internal/pipeline/executor.go`
- [ ] T066 [US4] Add persisted review decision ledger recovery helpers for file-processed and automation-processed cycles in `internal/db/round.go`, covering selected IDs, accept/skip labels, per-finding solution/instruction text, user findings, selection source, and fix summaries without adding a new schema requirement
- [ ] T067 [US4] Implement final audit renderer and no-findings state in `internal/reviewhandoff/audit.go`
- [ ] T068 [US4] Write final no-findings audit state after fix review succeeds with no remaining findings in `internal/pipeline/steps/review.go`
- [ ] T069 [US4] Regenerate or block missing final audit files before PR branch commits in `internal/pipeline/steps/push.go`
- [ ] T070 [US4] Explicitly stage the review audit file and avoid staging anchor files solely by anchor selection in `internal/pipeline/steps/push.go`
- [ ] T071 [US4] Preserve legacy AXI response handling while using the mirrored audit file path in `internal/cli/axi_drive.go`
- [ ] T072 [US4] Include review audit and phase label context in PR summaries where review state is surfaced in `internal/pipeline/steps/prsummary.go`
- [ ] T073 [US4] Run US4 checks with `go test ./internal/reviewhandoff ./internal/pipeline ./internal/pipeline/steps ./internal/db ./internal/cli`

**Checkpoint**: User Story 4 works independently and legacy automation remains compatible.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated guidance, formatting, and full validation.

- [ ] T074 [P] Update TUI review gate documentation for file handoff, `p process`, `c cancel`, and validation errors in `docs/src/content/docs/guides/tui.md`
- [ ] T075 [P] Update CLI/AXI documentation for `review_phase_label`, `review_file_path`, and legacy `respond --action approve|fix|skip` behavior in `docs/src/content/docs/reference/cli.md`
- [ ] T076 [P] Update gate and pipeline documentation for phase labels, review-file processing, and PR audit inclusion in `docs/src/content/docs/concepts/gate-model.md`, `docs/src/content/docs/concepts/pipeline.md`, and `docs/src/content/docs/reference/pipeline-steps.md`
- [ ] T077 [P] Update troubleshooting and auto-fix docs for stale hash, malformed handoff file, missing audit file, and automation mirror failures in `docs/src/content/docs/guides/troubleshooting.md` and `docs/src/content/docs/concepts/auto-fix.md`
- [ ] T078 [P] Update generated-agent guidance source and regenerate `skills/no-mistakes/SKILL.md` with `make skill`
- [ ] T079 [P] Update docs and generated-file guard tests for changed documentation and skill output in `workflow_docs_test.go` and `workflow_guard_generated_files_test.go`
- [ ] T080 Run `gofmt` on Go changes in `internal/reviewhandoff`, `internal/pipeline`, `internal/pipeline/steps`, `internal/ipc`, `internal/tui`, `internal/cli`, and `internal/db`
- [ ] T081 Run focused validation from `specs/001-review-file-handoff/quickstart.md`: `go test ./internal/reviewhandoff`, `go test ./internal/pipeline ./internal/pipeline/steps`, `go test ./internal/ipc ./internal/cli`, and `go test ./internal/tui`
- [ ] T082 Run tagged e2e review-file coverage with `go test -tags=e2e -count=1 -timeout 300s ./internal/e2e/...` when `internal/e2e/review_file_handoff_test.go` is added
- [ ] T083 Run full repository validation with `go test -race ./...`
- [ ] T084 Run final lint and docs validation with `make lint` and `make docs-build`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational and is the MVP.
- **User Story 2 (Phase 4)**: Depends on Foundational and the US1 handoff file path/writer contract.
- **User Story 3 (Phase 5)**: Depends on Foundational; can run after US1 IPC path fields are known.
- **User Story 4 (Phase 6)**: Depends on US1 writer/path contracts and US2 processing decision contracts.
- **Polish (Phase 7)**: Depends on all desired user stories.

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational; no dependency on other stories.
- **US2 (P1)**: Requires US1 handoff file generation and hash contract.
- **US3 (P2)**: Can start after Foundational, but AXI/TUI path rendering should align with US1 fields.
- **US4 (P2)**: Requires US1 file generation and US2 processing/mirroring invariants.

### Within Each User Story

- Add or update tests before implementation where feasible.
- Implement shared `internal/reviewhandoff` behavior before pipeline, IPC, TUI, CLI, or push integration.
- Keep raw review statuses and existing automation actions unchanged.
- Validate each story independently before starting the next dependent story.

---

## Parallel Opportunities

- Setup inventory tasks T003, T004, and T005 can run in parallel.
- Foundational tests T006 and T007 can run in parallel before model/I/O implementation.
- US1 tests T012 through T015 can run in parallel before implementation.
- US2 parser, validator, metadata update, executor, IPC, and TUI tests T028 through T033 can be drafted in parallel.
- US3 label, IPC, TUI, and AXI tests T046 through T049 can be drafted in parallel.
- US4 automation, audit, push, AXI, and e2e tests T058 through T063 can be drafted in parallel.
- Documentation updates T074 through T077 can run in parallel once behavior names and fields are stable.

---

## Parallel Example: User Story 1

```bash
# Launch independent US1 test work:
Task: "T012 [US1] Add path resolver tests in internal/reviewhandoff/path_test.go"
Task: "T013 [US1] Add writer and hash tests in internal/reviewhandoff/writer_test.go"
Task: "T014 [US1] Add executor handoff generation test in internal/pipeline/executor_review_handoff_test.go"
Task: "T015 [US1] Add TUI compact gate tests in internal/tui/review_file_gate_test.go"

# Launch independent US1 implementation after tests exist:
Task: "T016 [US1] Implement safe review file path resolution in internal/reviewhandoff/path.go"
Task: "T017 [US1] Implement canonical review-result hash computation in internal/reviewhandoff/hash.go"
```

## Parallel Example: User Story 2

```bash
Task: "T028 [US2] Add parser tests in internal/reviewhandoff/parser_test.go"
Task: "T029 [US2] Add validator tests in internal/reviewhandoff/validator_test.go"
Task: "T031 [US2] Add executor processing tests in internal/pipeline/executor_review_process_test.go"
Task: "T032 [US2] Add review-only IPC process request tests in internal/ipc/review_process_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "T046 [US3] Add phase label mapping tests in internal/reviewhandoff/phase_label_test.go"
Task: "T047 [US3] Add IPC serialization tests in internal/ipc/protocol_test.go"
Task: "T048 [US3] Add TUI pipeline label tests in internal/tui/pipeline_test.go"
Task: "T049 [US3] Add AXI output tests in internal/cli/axi_review_fields_test.go"
```

## Parallel Example: User Story 4

```bash
Task: "T058 [US4] Add automation mirror tests in internal/pipeline/executor_review_mirror_test.go"
Task: "T059 [US4] Add audit renderer tests in internal/reviewhandoff/audit_test.go"
Task: "T061 [US4] Add push audit inclusion tests in internal/pipeline/steps/push_test.go"
Task: "T062 [US4] Add AXI legacy response regression tests in internal/cli/axi_drive_test.go"
```

---

## Implementation Strategy

### MVP First (US1)

1. Complete Phase 1 and Phase 2.
2. Complete US1 tests and implementation.
3. Run `go test ./internal/reviewhandoff ./internal/pipeline ./internal/ipc ./internal/tui`.
4. Validate the compact review gate and generated handoff file before processing behavior is added.

### Incremental Delivery

1. Deliver US1 to move findings into a durable file.
2. Deliver US2 to process edited file answers into existing gate decisions.
3. Deliver US3 to normalize labels and structured fields across user-facing surfaces.
4. Deliver US4 to preserve automation compatibility and PR auditability.
5. Complete documentation, generated skill output, and full validation.

### Final Validation

1. Run focused package tests from `specs/001-review-file-handoff/quickstart.md`.
2. Run tagged e2e coverage when cross-process behavior is not fully proven by unit tests.
3. Run `go test -race ./...`.
4. Run `make lint`.
5. Run `make docs-build`.

---

## Summary

- Total tasks: 84
- Setup tasks: 5
- Foundational tasks: 6
- US1 tasks: 16
- US2 tasks: 18
- US3 tasks: 12
- US4 tasks: 16
- Polish tasks: 11
- Suggested MVP scope: Phase 1, Phase 2, and User Story 1
- Format rule: all task rows use `- [ ] T###`, optional `[P]`, optional story label only inside user-story phases, and exact file or package paths
