# Tasks: No-Worktree YOLO Guard

**Input**: Design documents from `/specs/002-no-worktree-yolo/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/yolo-boundary.md`, `quickstart.md`
**Tests**: Required. The feature changes gate automation, source writes, remote/provider writes, user output, docs, and generated agent guidance.
**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently after the shared foundation is in place.

## Phase 1: Setup (Shared Context)

**Purpose**: Confirm scope, affected paths, and baseline validation before editing.

- [x] T001 Review feature requirements and acceptance criteria in `specs/002-no-worktree-yolo/spec.md`, `specs/002-no-worktree-yolo/plan.md`, and `specs/002-no-worktree-yolo/contracts/yolo-boundary.md`
- [x] T002 Map existing unattended gate entry points in `internal/tui/commands.go`, `internal/cli/axi_drive.go`, `internal/daemon/daemon.go`, and `internal/daemon/manager.go`
- [x] T003 [P] Capture the current YOLO and AXI baseline by running focused tests for `internal/tui/yolo_test.go` and `internal/cli/axi_drive_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared boundary, policy, persistence, and IPC contracts required by all user stories.

**Critical**: No user story implementation should begin until this phase is complete.

- [x] T004 Add boundary status, decision source, actor type, approval surface, consent mode, and gate automation enums with validation tests in `internal/types/types.go` and `internal/types/types_test.go`
- [x] T005 [P] Add execution boundary verifier tests for safe worktree, primary checkout, configured evidence/temp allowances as non-source-only boundaries, rejection of source paths outside the disposable run worktree, missing worktree, symlink escape, nested worktree, stale proof, and Git metadata mismatch in `internal/boundary/verifier_test.go`
- [x] T006 Implement execution boundary result types and verifier logic using canonical paths, `paths.WorktreeDir`, `paths.RepoDir`, and trusted Git metadata in `internal/boundary/boundary.go` and `internal/boundary/verifier.go`
- [x] T007 [P] Add gate automation policy and fingerprint tests for allowed, withheld, not-requested, duplicate unattended response, and manual response cases in `internal/boundary/policy_test.go`
- [x] T008 Implement gate automation policy, recovery messages, and stable gate fingerprint helpers in `internal/boundary/policy.go`
- [x] T009 [P] Add database migration and round-trip tests for run boundary fields and run events in `internal/db/run_test.go` and `internal/db/event_test.go`
- [x] T010 Add run boundary columns, `run_events` schema, idempotent event insertion, and query helpers in `internal/db/schema.go`, `internal/db/run.go`, and `internal/db/event.go`
- [x] T011 [P] Add IPC serialization tests for boundary status, gate automation status, response decision metadata, and event fields in `internal/ipc/protocol_test.go`
- [x] T012 Extend `RunInfo`, `Event`, and `RespondParams` with boundary, gate automation, gate identity, and decision metadata in `internal/ipc/protocol.go`

**Checkpoint**: Shared contracts, persistence, and policy helpers compile and have focused failing-then-passing tests.

---

## Phase 3: User Story 1 - Block Unattended YOLO Without Isolation (Priority: P1) MVP

**Goal**: Unattended fix, approve, skip, git push, PR create/update, merge, and provider write actions are allowed only after fresh safe disposable-worktree proof.

**Independent Test**: Run a safe isolated run and an unsafe/unknown-boundary run with YOLO or `--yes`; only the safe run sends automatic responses or remote/provider writes.

### Tests for User Story 1

- [x] T013 [P] [US1] Add daemon response tests for unattended safe acceptance, unsafe/unknown withholding, and explicit manual response acceptance in `internal/daemon/manager_test.go`
- [x] T014 [P] [US1] Add daemon IPC handler tests for legacy manual defaults and unattended decision metadata rejection paths in `internal/daemon/daemon_test.go`
- [x] T015 [P] [US1] Add executor auto-fix guard tests for unsafe/unknown boundaries pausing automatic source-changing fixes in `internal/pipeline/executor_autofix_test.go`
- [x] T016 [P] [US1] Add remote/provider write guard tests for git push, PR create/update/merge, PR body updates, comments, statuses/check-runs, labels, metadata writes, future provider-write hooks through the shared policy, and CI auto-fix push paths in `internal/pipeline/steps/push_test.go`, `internal/pipeline/steps/pr_test.go`, and `internal/pipeline/steps/ci_autofix_test.go`
- [x] T017 [P] [US1] Add tagged safe and unsafe YOLO journey tests for cross-process behavior in `internal/e2e/no_worktree_yolo_test.go`

### Implementation for User Story 1

- [x] T018 [US1] Classify and persist the initial execution boundary immediately after worktree creation in `internal/daemon/manager.go`
- [x] T019 [US1] Refresh boundary proof before every unattended response and record withheld events when proof is unsafe or unknown in `internal/daemon/manager.go`
- [x] T020 [US1] Pass response decision metadata from IPC handlers into the run manager and executor in `internal/daemon/daemon.go` and `internal/daemon/manager.go`
- [x] T021 [US1] Mark TUI YOLO responses and AXI `--yes` responses as unattended with actor, surface, consent mode, gate ID, and fingerprint in `internal/tui/commands.go` and `internal/cli/axi_drive.go`
- [x] T022 [US1] Gate executor follow-up auto-fix loops on fresh safe proof and record withheld automation in `internal/pipeline/executor.go`
- [x] T023 [US1] Add boundary status to step execution context for concrete write callsites in `internal/pipeline/pipeline.go`
- [x] T024 [US1] Withhold unsafe or unknown unattended git push and CI auto-fix push writes in `internal/pipeline/steps/push.go` and `internal/pipeline/steps/ci_fix.go`
- [x] T025 [US1] Withhold unsafe or unknown unattended PR create/update and provider review-advancing writes in `internal/pipeline/steps/pr.go`, `internal/scm/host.go`, `internal/scm/github/github.go`, and `internal/scm/gitlab/gitlab.go`
- [x] T026 [US1] Persist allowed, withheld, and not-requested gate automation events with idempotent gate identity in `internal/db/event.go` and `internal/daemon/manager.go`

**Checkpoint**: US1 is independently functional when focused daemon, executor, step, and e2e tests prove unsafe/unknown unattended automation performs zero source or external advancing actions.

---

## Phase 4: User Story 2 - Explain Why Automation Was Withheld (Priority: P2)

**Goal**: TUI, AXI, headless CLI output, terminal status, and generated agent guidance explain requested mode, current gate, boundary status, reason, and recovery options when YOLO is withheld.

**Independent Test**: Trigger YOLO on unsafe and unknown runs, then verify every gate-driving or gate-observing surface reports the same structured withheld status and leaves manual actions available.

### Tests for User Story 2

- [x] T027 [P] [US2] Add run-info projection tests for boundary and gate automation status in `internal/daemon/runinfo_test.go`
- [x] T028 [P] [US2] Add AXI render tests for `automation.status: withheld`, boundary reason, gate, and help output in `internal/cli/axi_test.go` and `internal/cli/axi_drive_test.go`
- [x] T029 [P] [US2] Add TUI rendering tests for withheld YOLO footer or gate status copy in `internal/tui/yolo_test.go` and `internal/tui/action_bar_test.go`
- [x] T030 [P] [US2] Add generated skill guidance tests for withheld automation and manual-response restrictions in `internal/skill/skill_test.go`

### Implementation for User Story 2

- [x] T031 [US2] Project boundary and gate automation status into `RunInfo` for get-run, get-runs, active-run, and subscription consumers in `internal/daemon/daemon.go`
- [x] T032 [US2] Render structured AXI `automation` output for `axi run --yes`, `axi respond --yes`, and `axi status` in `internal/cli/axi_drive.go`, `internal/cli/axi.go`, and `internal/cli/axi_render.go`
- [x] T033 [US2] Render terminal status and run list boundary/gate automation summaries in `internal/cli/status.go` and `internal/cli/runs.go`
- [x] T034 [US2] Render withheld YOLO explanation and manual/restart recovery affordance in `internal/tui/view.go`, `internal/tui/pipeline.go`, and `internal/tui/action_bar_test.go`
- [x] T035 [US2] Keep manual approve, fix, skip, and `abort` controls available on withheld unsafe/unknown runs, mapping the TUI `abort` label to the canonical `cancel` GateDecision action, in `internal/tui/keys.go` and `internal/tui/commands.go`
- [x] T036 [US2] Update canonical generated-agent guidance source for `--yes`, withheld automation, and explicit per-gate human decisions in `internal/skill/skill.go`
- [x] T037 [US2] Regenerate and verify committed agent guidance in `skills/no-mistakes/SKILL.md` using `cmd/genskill/main.go`
- [x] T038 [US2] Update user docs for allowed, withheld, and manual recovery behavior in `docs/src/content/docs/concepts/auto-fix.md`, `docs/src/content/docs/reference/pipeline-steps.md`, `docs/src/content/docs/reference/cli.md`, and `docs/src/content/docs/guides/troubleshooting.md`

**Checkpoint**: US2 is independently functional when unsafe/unknown output across TUI, AXI, status, docs, and generated guidance gives the same reason and recovery path without requiring logs.

---

## Phase 5: User Story 3 - Preserve Existing Isolated YOLO Behavior (Priority: P3)

**Goal**: Safe isolated runs preserve today's YOLO behavior: actionable findings get one fix round, fix-review gates are approved, no-op-only gates are approved, and duplicate automatic responses are suppressed.

**Independent Test**: Run existing safe YOLO unit tests plus a safe isolated e2e journey and compare outcomes to current documented behavior.

### Tests for User Story 3

- [x] T039 [P] [US3] Extend safe-boundary TUI YOLO tests for actionable fix, fix-review approval, no-op approval, and duplicate suppression in `internal/tui/yolo_test.go`
- [x] T040 [P] [US3] Extend safe-boundary AXI `--yes` tests for action selection, fix-review approval, no-op approval, and wait-for-gate behavior in `internal/cli/axi_drive_test.go`
- [x] T041 [P] [US3] Add daemon restart or reattach duplicate-response tests using persisted gate identity in `internal/daemon/subscribe_recover_test.go`

### Implementation for User Story 3

- [x] T042 [US3] Report `gate_automation.status=allowed` for safe unattended runs without adding warning copy in `internal/daemon/daemon.go`
- [x] T043 [US3] Preserve in-memory and persisted duplicate suppression for safe TUI YOLO responses in `internal/tui/commands.go`
- [x] T044 [US3] Preserve AXI `--yes` one-fix-round and fix-review convergence while adding persisted gate identity checks in `internal/cli/axi_drive.go`
- [x] T045 [US3] Keep executor and CI auto-fix behavior unchanged for refreshed safe boundaries in `internal/pipeline/executor.go` and `internal/pipeline/steps/ci.go`

**Checkpoint**: US3 is independently functional when the existing safe-run YOLO behavior remains unchanged except for structured non-warning boundary fields.

---

## Phase 6: User Story 4 - Preserve Requirement Origin for Future Tasks (Priority: P4)

**Goal**: The feature directory keeps the original request, inferred purpose, source-scout findings, and future implementation anchors easy to find.

**Independent Test**: Open the origin reference and confirm it states the request, purpose, source-scout findings, files to inspect first, and non-goals without polluting the stakeholder spec with implementation detail.

### Tests for User Story 4

- [x] T046 [P] [US4] Verify the companion origin reference covers original request, inferred purpose, source-scout findings, first files, and non-goals in `specs/002-no-worktree-yolo/no-worktree-yolo.md`

### Implementation for User Story 4

- [x] T047 [US4] Update newly discovered implementation anchors or changed source paths in `specs/002-no-worktree-yolo/no-worktree-yolo.md`
- [x] T048 [US4] Confirm the stakeholder specification remains implementation-neutral after task generation in `specs/002-no-worktree-yolo/spec.md`

**Checkpoint**: US4 is complete when future implementers can recover origin and source-context anchors from the spec directory in under one minute.

---

## Phase 7: Polish & Cross-Cutting Verification

**Purpose**: Final consistency, validation, and reviewer-visible evidence across the full feature.

- [x] T049 [P] Run `gofmt` on changed Go files under `internal/boundary/`, `internal/db/`, `internal/ipc/`, `internal/daemon/`, `internal/pipeline/`, `internal/cli/`, `internal/tui/`, and `internal/skill/`
- [x] T050 Run targeted package tests from `specs/002-no-worktree-yolo/quickstart.md` for `internal/db`, `internal/ipc`, `internal/daemon`, `internal/pipeline`, `internal/pipeline/steps`, `internal/tui`, and `internal/cli`
- [x] T051 Run boundary and git package tests from `specs/002-no-worktree-yolo/quickstart.md` for `internal/git` and `internal/boundary`
- [x] T052 Run tagged cross-process smoke tests from `specs/002-no-worktree-yolo/quickstart.md` for `internal/e2e/...`
- [x] T053 Run full Go validation from `specs/002-no-worktree-yolo/quickstart.md` with `go test -race ./...`
- [x] T054 Run lint validation from `specs/002-no-worktree-yolo/quickstart.md` with `make lint`
- [x] T055 [P] Run docs and generated skill checks from `specs/002-no-worktree-yolo/quickstart.md` with `make docs-build` and `make skill-check`
- [x] T056 Record reviewer-visible evidence for withheld output and safe-run unchanged behavior in `specs/002-no-worktree-yolo/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies.
- **Phase 2 Foundational**: Depends on Phase 1 and blocks all user story phases.
- **Phase 3 US1**: Depends on Phase 2. This is the MVP and safety-critical path.
- **Phase 4 US2**: Depends on Phase 2 and uses US1's persisted/policy status when integrated.
- **Phase 5 US3**: Depends on Phase 2 and should be validated after US1 guard behavior exists.
- **Phase 6 US4**: Depends only on Phase 1 and can run in parallel with code work after task generation.
- **Phase 7 Polish**: Depends on all desired user story phases.

### User Story Dependencies

- **US1 (P1)**: First deliverable after foundation; no dependency on US2 or US3.
- **US2 (P2)**: Can start after foundation once `RunInfo` boundary/gate automation fields exist; integrates with US1 event statuses.
- **US3 (P3)**: Can start after foundation; validates that safe path changes from US1 do not regress current behavior.
- **US4 (P4)**: Documentation/reference validation; can run independently of code implementation.

### Parallel Opportunities

- Foundation tests in T005, T007, T009, and T011 target different packages and can run in parallel.
- US1 tests in T013 through T017 target daemon, pipeline, steps, and e2e boundaries and can be drafted in parallel.
- US2 render and guidance tests in T027 through T030 target different packages and can run in parallel.
- US3 safe-path tests in T039 through T041 target independent surfaces and can run in parallel.
- US4 reference validation in T046 can run while code implementation proceeds.
- Final format and docs/skill checks in T049 and T055 can run in parallel after edits settle.

---

## Parallel Example: User Story 1

```bash
# Independent tests before implementation:
Task: "T013 daemon response tests in internal/daemon/manager_test.go"
Task: "T015 executor auto-fix guard tests in internal/pipeline/executor_autofix_test.go"
Task: "T016 remote/provider write guard tests in internal/pipeline/steps/push_test.go, internal/pipeline/steps/pr_test.go, and internal/pipeline/steps/ci_autofix_test.go"

# Independent implementation slices after foundation:
Task: "T019 refresh boundary proof before unattended responses in internal/daemon/manager.go"
Task: "T024 withhold unsafe/unknown git push and CI auto-fix push writes in internal/pipeline/steps/push.go and internal/pipeline/steps/ci_fix.go"
Task: "T025 withhold unsafe/unknown PR writes in internal/pipeline/steps/pr.go and internal/scm/"
```

## Parallel Example: User Story 2

```bash
# Independent render/guidance tests:
Task: "T028 AXI render tests in internal/cli/axi_test.go and internal/cli/axi_drive_test.go"
Task: "T029 TUI rendering tests in internal/tui/yolo_test.go and internal/tui/action_bar_test.go"
Task: "T030 generated skill guidance tests in internal/skill/skill_test.go"

# Independent output updates:
Task: "T032 structured AXI automation output in internal/cli/axi_drive.go, internal/cli/axi.go, and internal/cli/axi_render.go"
Task: "T034 TUI withheld explanation in internal/tui/view.go, internal/tui/pipeline.go, and internal/tui/action_bar_test.go"
Task: "T038 documentation updates in docs/src/content/docs/"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1) end to end.
3. Validate with targeted boundary, daemon, pipeline, step, CLI/TUI, and e2e tests.
4. Stop and confirm unsafe/unknown unattended automation performs zero automatic source or external advancing actions.

### Incremental Delivery

1. Deliver US1 safety enforcement first.
2. Add US2 user-facing explanation and generated guidance.
3. Add US3 safe-run regression hardening.
4. Confirm US4 origin reference remains current.
5. Run Phase 7 full validation.

### Notes

- [P] tasks touch different files or can be drafted without depending on incomplete implementation.
- Tests should fail before implementation where the behavior does not exist yet.
- Persisted `safe` status is display state only; every unattended action still needs a fresh verifier pass.
- Manual responses on unsafe or unknown boundaries must remain explicit per-gate actions, not reuse of broad unattended consent.
- Do not add new dependencies for this feature.
