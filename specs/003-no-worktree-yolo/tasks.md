# Tasks: Current Worktree YOLO Mode

**Input**: Design documents from `specs/003-no-worktree-yolo/`
**Prerequisites**: `specs/003-no-worktree-yolo/plan.md`, `specs/003-no-worktree-yolo/spec.md`, `specs/003-no-worktree-yolo/research.md`, `specs/003-no-worktree-yolo/data-model.md`, `specs/003-no-worktree-yolo/contracts/`, `specs/003-no-worktree-yolo/quickstart.md`

**Tests**: Required for all code changes. Add or update focused `_test.go` coverage before implementation, then run broad race, lint, e2e, docs, and generated-skill validation in the final phase.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested as an independent increment after shared foundations are complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and does not depend on incomplete tasks.
- **[Story]**: User story label from `specs/003-no-worktree-yolo/spec.md`.
- Every task names exact repository file paths.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Confirm implementation boundaries and validation commands before changing code.

- [ ] T001 Review source-of-truth requirements in `specs/003-no-worktree-yolo/spec.md`, `specs/003-no-worktree-yolo/plan.md`, and `specs/003-no-worktree-yolo/no-worktree-yolo.md`
- [ ] T002 Confirm no unrelated user edits will be modified by checking `AGENTS.md`, `.specify/feature.json`, and `specs/003-no-worktree-yolo/tasks.md`
- [ ] T003 [P] Confirm final validation commands from `Makefile` and `specs/003-no-worktree-yolo/quickstart.md`: `go test -race ./...`, `make lint`, `make e2e`, and docs build when docs change

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared contracts, persistence, and helper seams required by every user story.

**Critical**: No user story implementation should begin until this phase is complete.

- [ ] T004 [P] Add enum and validation tests for worktree mode, metadata availability, evidence state, terminal reasons, and rejection reasons in `internal/types/types_test.go`
- [ ] T005 Define `WorktreeMode`, metadata/evidence state values, current-mode terminal/rejection constants, and validation helpers in `internal/types/types.go`
- [ ] T006 [P] Add run metadata migration/default/degraded-state tests in `internal/db/run_test.go`
- [ ] T007 Extend run schema, migrations, `db.Run`, scan columns, insert/update helpers, and cleanup-eligibility accessors in `internal/db/schema.go` and `internal/db/run.go`
- [ ] T008 [P] Add fix outcome/provenance persistence tests for proposed, attempted, committed, and failed states in `internal/db/round_test.go`
- [ ] T009 Add fix outcome/provenance storage helpers and count aggregation support in `internal/db/schema.go` and `internal/db/round.go`
- [ ] T010 [P] Add IPC protocol round-trip tests for direct start params, start result, extended `RunInfo`, structured conflict details, and JSON-RPC error details in `internal/ipc/protocol_test.go`
- [ ] T011 Extend IPC constants, direct start request/result structs, `RunInfo` fields, rejection detail structs, and JSON-RPC error detail marshaling in `internal/ipc/protocol.go` and `internal/ipc/server.go`
- [ ] T012 [P] Add git helper tests for canonical worktree root, clean/dirty classification, ignored-only files, default branch detection, and review-base refresh evidence in `internal/git/git_branch_worktree_test.go`
- [ ] T013 Add git helpers for canonical current worktree root, tracked/untracked non-ignored dirty checks, ignored-only allowance, and trustworthy default-branch merge-base resolution in `internal/git/git.go`

**Checkpoint**: Shared types, DB, IPC, and git helpers are ready for story work.

---

## Phase 3: User Story 1 - Run From The Current Git Worktree (Priority: P1)

**Goal**: `no-mistakes --no-worktree --yolo` starts a run in the current git worktree root without creating a no-mistakes-owned worktree.

**Independent Test**: From a clean non-default branch, run the root command from a subdirectory and verify the persisted execution directory is the git worktree root and no directory is created under the managed worktree store.

### Tests for User Story 1

- [ ] T014 [P] [US1] Add root command flag parsing tests for `--no-worktree`, `--yolo`, and `--yes --yolo` in `internal/cli/root_test.go`
- [ ] T015 [P] [US1] Add root current-mode start tests for subdirectory invocation, no managed worktree creation, and `--yolo` equivalence in `internal/daemon/manager_test.go`
- [ ] T016 [P] [US1] Add inferred-intent tests for non-interactive or `--yolo` root current-mode starts, covering missing-intent rejection, redacted bounded persistence/rendering, no raw transcript or log storage, and recovery guidance that does not echo transcript snippets in `internal/cli/root_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Add root command `--no-worktree` and `--yolo` flag wiring with `--yes` logical OR behavior in `internal/cli/root.go`
- [ ] T018 [US1] Add root current-worktree start flow that resolves branch, head, intent, skip settings, worktree mode, and direct daemon start inputs in `internal/cli/root.go` and `internal/cli/attach.go`
- [ ] T019 [US1] Add daemon current-mode start support that bypasses `git.WorktreeAdd` and executes in the resolved current root in `internal/daemon/manager.go`
- [ ] T020 [US1] Persist root current-mode run metadata before recoverability and use the canonical current worktree root as `Executor.Execute` workDir in `internal/daemon/manager.go`

**Checkpoint**: Root current-worktree mode works independently.

---

## Phase 4: User Story 2 - Drive Current-Worktree Runs Through AXI (Priority: P1)

**Goal**: `no-mistakes axi run --intent "..." --no-worktree --yolo` starts or drives a compatible current-worktree run through AXI without gate-remote push triggering.

**Independent Test**: From a clean feature branch, run AXI current mode with intent and confirm it starts directly through IPC, resumes only compatible active current-mode runs, and keeps the existing AXI gate behavior.

### Tests for User Story 2

- [ ] T021 [P] [US2] Add AXI flag parsing and explicit-intent requirement tests for `--no-worktree`, `--yolo`, and compatible-drive cases in `internal/cli/axi_drive_test.go`
- [ ] T022 [P] [US2] Add IPC client/server tests for the direct current-mode start method and structured invalid-params responses in `internal/ipc/rpc_test.go`
- [ ] T023 [P] [US2] Add tagged e2e coverage for `no-mistakes axi run --intent "validate current-worktree execution" --no-worktree --yolo` in `internal/e2e/axi_journey_test.go`

### Implementation for User Story 2

- [ ] T024 [US2] Add AXI `run` command `--no-worktree` and `--yolo` flags and telemetry fields in `internal/cli/axi_drive.go`
- [ ] T025 [US2] Route AXI current-mode starts through direct IPC instead of `git.PushWithOptions` to `no-mistakes` in `internal/cli/axi_drive.go`
- [ ] T026 [US2] Register and implement the daemon direct start handler for current-mode and isolated-compatible params in `internal/daemon/daemon.go` and `internal/daemon/manager.go`
- [ ] T027 [US2] Preserve AXI drive/auto-resolution behavior after direct start or compatible resume in `internal/cli/axi_drive.go`

**Checkpoint**: AXI current-worktree mode works independently.

---

## Phase 5: User Story 3 - Preserve Gate Safety And Review Scope (Priority: P2)

**Goal**: Current-worktree mode keeps branch hygiene, active-run safety, and full branch review scope against a proven default-branch base.

**Independent Test**: Dirty, detached, default-branch, missing-base, and incompatible-active-run starts reject before pipeline execution; valid starts review the full branch diff.

### Tests for User Story 3

- [ ] T028 [P] [US3] Add current-mode preflight rejection tests for dirty tracked files, untracked non-ignored files, ignored-only files, detached HEAD, unborn HEAD, default branch, and uninitialized repo in `internal/git/git_branch_worktree_test.go`
- [ ] T029 [P] [US3] Add review-base and current-mode rebase/fresh-upstream evidence tests for local merge base, one non-interactive default-branch ref refresh, preserved rebase-before-review execution, unsafe freshness/rebase failure rendering, and `rejected_no_trustworthy_base` in `internal/git/git_diff_log_test.go` and `internal/pipeline/steps/rebase_test.go`
- [ ] T030 [P] [US3] Add review step tests proving current mode uses the persisted full branch base and does not silently fall back to a narrow diff in `internal/pipeline/steps/review_test.go`
- [ ] T031 [P] [US3] Add active-run compatibility tests for mode, head, work directory, review base, approval mode, skip config, intent identity or `start_shape_hash`, immutable start-shape rejection/ignored guidance, and whitelisted conflict output fields in `internal/daemon/manager_test.go`

### Implementation for User Story 3

- [ ] T032 [US3] Enforce current-mode preflight rejection before recoverable run creation in `internal/cli/axi_drive.go`, `internal/cli/root.go`, and `internal/git/git.go`
- [ ] T033 [US3] Persist review-base evidence and missing-base rejection details on current-mode start in `internal/daemon/manager.go` and `internal/db/run.go`
- [ ] T034 [US3] Update review diff selection to use proven current-mode review base without empty-tree or narrow fallback in `internal/pipeline/steps/review.go`
- [ ] T035 [US3] Replace same-branch auto-cancellation with active-run compatibility validation and safe conflict responses in `internal/daemon/manager.go`
- [ ] T036 [US3] Persist skipped, deferred, informational, fixed, approved, and passed gate decision counts distinctly in `internal/pipeline/executor.go` and `internal/db/step.go`

**Checkpoint**: Current mode is safety-equivalent to isolated mode with stronger review-base proof.

---

## Phase 6: User Story 4 - Make Current-Worktree Runs Visible And Recoverable (Priority: P3)

**Goal**: CLI, AXI, status, TUI, reports, PR summaries, cleanup, and recovery clearly distinguish "uses this checkout" from disposable isolated execution.

**Independent Test**: Start current and isolated runs, inspect all rendering surfaces, crash/recover stale current runs, and confirm current work directories are never removed.

### Tests for User Story 4

- [ ] T037 [P] [US4] Add AXI/status/runs rendering tests for `worktree_mode`, `worktree_label`, `work_dir_label`, `current_worktree_warning`, metadata state, evidence state, terminal reason, full finding count fields (`reported_findings`, `fixed_findings`, `unresolved_findings`, `skipped_findings`, `approved_as_is_findings`, `unavailable_findings`), cross-surface consistency for multi-round runs, and path minimization in `internal/cli/axi_render_test.go`
- [ ] T038 [P] [US4] Add TUI rendering tests for current-mode warning lifecycle, safe labels, degraded metadata, and no repeated full path output in `internal/tui/rendering_test.go`
- [ ] T039 [P] [US4] Add cleanup and stale-recovery tests proving `worktree_mode=current` and malformed current metadata never remove the current directory in `internal/daemon/manager_test.go` and `internal/db/run_test.go`
- [ ] T040 [P] [US4] Add PR summary and generated report tests for current-mode label, safe work-dir label, fix count, commit references, degraded evidence, and run/report reference in `internal/pipeline/steps/prsummary_test.go`
- [ ] T041 [P] [US4] Add fix provenance tests proving applied-fix claims come from persisted records instead of prose summaries in `internal/pipeline/executor_fix_test.go`

### Implementation for User Story 4

- [ ] T042 [US4] Populate extended run info, rendering envelope fields, terminal reasons, metadata availability, evidence state, and fix/count summaries in `internal/daemon/daemon.go` and `internal/ipc/protocol.go`
- [ ] T043 [US4] Update root/attach CLI terminal output plus AXI home/run/status/runs rendering to show current-mode warnings, run/report references, structured fields, and safe labels at start/pre-start, fix-review, checks-passed/passed, failure/cancellation, and stale recovery in `internal/cli/root.go`, `internal/cli/attach.go`, `internal/cli/axi.go`, `internal/cli/axi_render.go`, `internal/cli/status.go`, and `internal/cli/runs.go`
- [ ] T044 [US4] Update TUI pipeline, footer, outcome, and error rendering for current-mode warnings and degraded evidence in `internal/tui/pipeline.go`, `internal/tui/view.go`, and `internal/tui/messages.go`
- [ ] T045 [US4] Make setup failure, panic handling, cancellation, normal completion, and stale recovery cleanup mode-aware in `internal/daemon/manager.go` and `internal/daemon/daemon.go`
- [ ] T046 [US4] Add current-mode metadata, safe labels, fix counts, commit references, unresolved/degraded evidence, and run/report references to PR summaries and generated reports in `internal/pipeline/steps/prsummary.go`
- [ ] T047 [US4] Record fix outcome state transitions and commit SHA provenance during auto-fix and manual fix rounds in `internal/pipeline/executor.go` and `internal/db/round.go`
- [ ] T048 [US4] Fail closed when current-mode metadata is missing, malformed, stale, or non-canonical instead of rendering as passed or isolated in `internal/db/run.go`, `internal/cli/axi_render.go`, and `internal/tui/pipeline.go`

**Checkpoint**: Current-mode runs are visible, recoverable, and cleanup-safe across all surfaces.

---

## Phase 7: User Story 5 - Preserve Existing Isolated Defaults (Priority: P4)

**Goal**: Existing default root, wizard, push-triggered, rerun, AXI `--yes`, cleanup, and generated guidance behavior remains unchanged unless current mode is explicitly requested.

**Independent Test**: Run existing flows without `--no-worktree` and verify they still create and clean up disposable no-mistakes-owned worktrees with unchanged `--yes` behavior.

### Tests for User Story 5

- [ ] T049 [P] [US5] Add isolated default regression tests for root, wizard, push-triggered, rerun, and AXI start paths in `internal/cli/axi_drive_test.go` and `internal/daemon/manager_test.go`
- [ ] T050 [P] [US5] Add `--yes` behavior regression tests and `--yes --yolo` acceptance tests in `internal/cli/root_test.go` and `internal/cli/axi_drive_test.go`
- [ ] T051 [P] [US5] Add tagged e2e regression coverage for isolated AXI/root/push flows without `--no-worktree` in `internal/e2e/journey_test.go` and `internal/e2e/daemon_run_test.go`

### Implementation for User Story 5

- [ ] T052 [US5] Preserve the existing gate-push `triggerRun` and rerun paths when `--no-worktree` is absent in `internal/cli/axi_drive.go` and `internal/daemon/manager.go`
- [ ] T053 [US5] Keep isolated worktree creation, config loading, executor workDir, and cleanup behavior unchanged for default runs in `internal/daemon/manager.go`
- [ ] T054 [US5] Ensure `--yolo` only aliases `--yes` and does not create a new approval mode in `internal/cli/root.go`, `internal/cli/axi_drive.go`, and `internal/types/types.go`

**Checkpoint**: Default isolated behavior is regression-covered and unchanged.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated guidance, formatting, and full verification.

- [ ] T055 [P] Update CLI reference docs for `--no-worktree`, `--yolo`, root command behavior, AXI current mode, and explicit intent requirements in `docs/src/content/docs/reference/cli.md`
- [ ] T056 [P] Update conceptual, agent-workflow, and troubleshooting docs for current-worktree execution, warnings, cleanup boundaries, review-base rejection, recovery, and AXI current-worktree usage in `docs/src/content/docs/concepts/gate-model.md`, `docs/src/content/docs/concepts/pipeline.md`, `docs/src/content/docs/guides/agents.md`, and `docs/src/content/docs/guides/troubleshooting.md`
- [ ] T057 Update generated agent guidance source and committed output for current-worktree AXI usage by editing `internal/skill/skill.go` and regenerating `skills/no-mistakes/SKILL.md` with `make skill`
- [ ] T058 [P] Update implementer validation guidance for current-mode and isolated-regression smoke checks in `specs/003-no-worktree-yolo/quickstart.md`
- [ ] T059 Run focused package tests from `specs/003-no-worktree-yolo/quickstart.md`: `go test ./internal/cli ./internal/daemon ./internal/db ./internal/git ./internal/ipc ./internal/tui`
- [ ] T060 Run pipeline-focused package tests from `specs/003-no-worktree-yolo/quickstart.md`: `go test ./internal/pipeline ./internal/pipeline/steps`
- [ ] T061 Run repository race test from `Makefile`: `go test -race ./...`
- [ ] T062 Run lint and generated-skill drift check from `Makefile`: `make lint`
- [ ] T063 Run tagged end-to-end suite from `Makefile`: `make e2e`
- [ ] T064 Run docs build after documentation changes using `make docs-build`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 and blocks all story phases.
- **Phase 3 US1**: Depends on Phase 2; can be implemented as the MVP.
- **Phase 4 US2**: Depends on Phase 2; can proceed in parallel with US1 once shared direct-start contracts exist.
- **Phase 5 US3**: Depends on Phase 2 and must be complete before declaring current mode safe.
- **Phase 6 US4**: Depends on Phase 2 and uses metadata from US1/US2/US3.
- **Phase 7 US5**: Depends on Phase 2 and should run continuously as a regression guard while US1-US4 land.
- **Phase 8 Polish**: Depends on desired story scope being complete.

### User Story Dependencies

- **US1 (P1)**: Independent after Phase 2. Provides root current-mode MVP.
- **US2 (P1)**: Independent after Phase 2. Shares direct-start and metadata foundations but must remain AXI-testable on its own.
- **US3 (P2)**: Independent safety layer after Phase 2; strengthens valid starts and rejection behavior for both US1 and US2.
- **US4 (P3)**: Independent visibility/recovery layer after Phase 2; consumes persisted fields from all current-mode runs.
- **US5 (P4)**: Independent regression layer after Phase 2; protects existing isolated behavior throughout.

### Within Each User Story

- Write or update tests before implementation.
- Add shared data and wire contracts before CLI/daemon behavior.
- Implement CLI/AXI input handling before daemon execution paths.
- Implement persistence before rendering and cleanup decisions.
- Validate each story independently before moving to the next priority.

### Parallel Opportunities

- T003 can run alongside T001-T002.
- T004, T006, T008, T010, and T012 can run in parallel because they target different packages.
- After Phase 2, US1 and US2 tests can be written in parallel.
- US3 safety tests, US4 rendering tests, and US5 regression tests can be written in parallel after Phase 2.
- T055, T056, and T058 can run in parallel once user-visible behavior text stabilizes.

---

## Parallel Example: User Story 2

```bash
# Independent tests for AXI current mode:
Task: "T021 [US2] Add AXI flag parsing and intent requirement tests in internal/cli/axi_drive_test.go"
Task: "T022 [US2] Add direct start IPC tests in internal/ipc/rpc_test.go"
Task: "T023 [US2] Add tagged e2e coverage in internal/e2e/axi_journey_test.go"

# Independent implementation slices after tests exist:
Task: "T024 [US2] Add AXI flags in internal/cli/axi_drive.go"
Task: "T026 [US2] Register daemon direct start handler in internal/daemon/daemon.go and internal/daemon/manager.go"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 for root current-worktree mode.
3. Stop and validate US1 independently with focused CLI/daemon tests.

### Incremental Delivery

1. Complete US1 root current mode.
2. Complete US2 AXI current mode.
3. Complete US3 safety and review-base hardening.
4. Complete US4 visibility and recovery surfaces.
5. Complete US5 isolated regression guard.
6. Complete Phase 8 verification.

### Parallel Team Strategy

1. One lane owns foundational DB/types/IPC.
2. One lane owns git/preflight/review-base helpers.
3. One lane owns root/AXI command surfaces.
4. One lane owns rendering/docs/generated skill after metadata contracts stabilize.

---

## Notes

- `[P]` tasks touch different files or are independent test-writing tasks.
- Tests are intentionally placed before implementation tasks in every story.
- Current-mode cleanup must be data-driven from persisted mode metadata, not inferred from path shape.
- Conflict responses must use only the whitelisted fields from `specs/003-no-worktree-yolo/contracts/current-worktree-run.md`.
- Do not add typo aliases for `no-mistakes`; only add `--yolo` as a `--yes` alias.
