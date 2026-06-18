---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests or reviewer-visible evidence are REQUIRED for code changes. Include targeted `_test.go` tasks before implementation, and include e2e/docs validation tasks when the plan touches gate, daemon, agent, provider, docs, or generated-skill behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **CLI entry point**: `cmd/no-mistakes/`
- **Core packages**: `internal/<package>/`
- **Unit/integration tests**: co-located `*_test.go`
- **End-to-end tests**: `internal/e2e/` with the `e2e` build tag
- **Generated skill**: `skills/no-mistakes/SKILL.md` from `internal/skill` via `make skill`
- **Docs**: `docs/src/content/docs/`
- **Scripts**: `scripts/` and platform-specific install/update helpers
- Tasks MUST include exact file paths from `plan.md`

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit-tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Confirm affected packages, commands, docs, scripts, generated skill files, and test files from plan.md
- [ ] T002 Confirm feature branch, git status, and unrelated user changes before editing
- [ ] T003 [P] Identify exact validation commands: `go test -race ./...`, `make lint`, `make e2e` or docs build as applicable

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T004 Define or update shared types/contracts in `internal/types/` or affected package paths
- [ ] T005 [P] Add or update persistence/recovery helpers in `internal/db/`, `internal/daemon/`, or affected package paths
- [ ] T006 [P] Add or update CLI/AXI/TUI command plumbing in `internal/cli/`, `internal/ipc/`, or `internal/tui/`
- [ ] T007 Add or update agent/pipeline boundary handling in `internal/agent/` or `internal/pipeline/`
- [ ] T008 Configure actionable error handling and structured logging for new failure paths
- [ ] T009 Update configuration defaults or parsing in `internal/config/` only if required by the spec

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T010 [P] [US1] Unit test for [package behavior] in internal/[package]/[file]_test.go
- [ ] T011 [P] [US1] Integration/e2e test for [gate or agent journey] in internal/e2e/[file]_test.go, if the story crosses process/git/provider boundaries
- [ ] T012 [US1] Evidence check for [manual or visual reviewer artifact] when automated tests cannot fully prove user-visible behavior

### Implementation for User Story 1

- [ ] T013 [P] [US1] Implement package changes in internal/[package]/[file].go
- [ ] T014 [P] [US1] Implement CLI/AXI/TUI surface changes in internal/[package]/[file].go, if applicable
- [ ] T015 [US1] Wire pipeline, daemon, git, or provider behavior through the affected boundary
- [ ] T016 [US1] Add validation, approval-gate handling, and actionable error messages
- [ ] T017 [US1] Update docs or generated skill content for user-visible behavior

**Checkpoint**: At this point, User Story 1 MUST be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2

- [ ] T018 [P] [US2] Unit test for [package behavior] in internal/[package]/[file]_test.go
- [ ] T019 [P] [US2] Integration/e2e test for [gate or agent journey] in internal/e2e/[file]_test.go, if required by plan.md

### Implementation for User Story 2

- [ ] T020 [P] [US2] Implement package changes in internal/[package]/[file].go
- [ ] T021 [US2] Implement command, daemon, pipeline, TUI, or AXI behavior in the affected package
- [ ] T022 [US2] Integrate with User Story 1 components only where the spec requires it
- [ ] T023 [US2] Update docs, config reference, or generated skill content for user-visible behavior

**Checkpoint**: At this point, User Stories 1 AND 2 MUST both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 3

- [ ] T024 [P] [US3] Unit test for [package behavior] in internal/[package]/[file]_test.go
- [ ] T025 [P] [US3] Integration/e2e test for [gate or agent journey] in internal/e2e/[file]_test.go, if required by plan.md

### Implementation for User Story 3

- [ ] T026 [P] [US3] Implement package changes in internal/[package]/[file].go
- [ ] T027 [US3] Implement command, daemon, pipeline, TUI, or AXI behavior in the affected package
- [ ] T028 [US3] Update docs, config reference, or generated skill content for user-visible behavior

**Checkpoint**: All user stories MUST now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in docs/
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX Performance optimization across all stories
- [ ] TXXX [P] Additional unit tests in internal/[package]/[file]_test.go
- [ ] TXXX Security hardening
- [ ] TXXX Regenerate `skills/no-mistakes/SKILL.md` with `make skill`, if agent guidance changed
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but MUST remain independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but MUST remain independently testable

### Within Each User Story

- Tests or evidence tasks MUST be defined before implementation; new or changed tests MUST fail before implementation where feasible
- Shared contracts before package implementation
- Package implementation before CLI/TUI/AXI wiring
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Independent package, docs, and generated-artifact tasks marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Unit test for [package behavior] in internal/[package]/[file]_test.go"
Task: "Integration/e2e test for [gate or agent journey] in internal/e2e/[file]_test.go"

# Launch independent implementation slices for User Story 1 together:
Task: "Implement package changes in internal/[package]/[file].go"
Task: "Update docs in docs/src/content/docs/[area]/[file].md"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story MUST be independently completable and testable
- Verify tests fail before implementing when a test file is added or changed
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
