# Feature Specification: Current Worktree YOLO Mode

**Feature Branch**: `003-no-worktree-yolo`  
**Created**: 2026-06-18  
**Status**: Draft  
**Input**: User description: "read requirement in create detail spec for no-worktree-yolo.md, save reference so that the next phase can know the origin purpose, spawn sub-agents for help, scout source code to know the context, create detail spec"

## Clarifications

### Session 2026-06-18

- Q1: If current-worktree mode cannot resolve a trustworthy default-branch merge base, the system attempts one non-interactive default-branch ref refresh, then rejects the run before pipeline execution if the base is still unavailable.
- Q2: If the root `no-mistakes --no-worktree --yolo` command cannot infer usable intent, non-interactive or `--yolo` execution fails with recovery guidance instead of starting with empty or generic intent.
- Q3: If an active run for the same repo and branch is incompatible by mode, head, or work directory, the new request is rejected and reports the exact active run plus resume or abort guidance using only a safe conflict-message field set: run ID, worktree mode, branch, short head, safe work-directory label, status, and resume/abort commands. Conflict output MUST NOT dump raw logs, transcript-derived intent, diff hunks, code excerpts, secret-bearing metadata, or full run records.
- Q4: Current-worktree mode warnings are visible but non-blocking in CLI, AXI, status, and TUI output; `--yes` and `--yolo` do not require an extra confirmation.
- Q5: Current-mode resume compatibility requires the same resolved work directory in addition to compatible repo, branch, head commit, and worktree mode.
- Q6: Current-worktree preflight rejects tracked changes and untracked non-ignored files, while ignored files do not block the run by themselves.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run From The Current Git Worktree (Priority: P1)

As a developer or coding agent already working inside a tool-created git worktree, I need to start no-mistakes without creating another nested no-mistakes-owned worktree, so the gate validates and fixes the branch exactly where the current tool expects the files and commits to live.

**Why this priority**: This is the primary feature. Without it, users in Archon-created or other tool-created worktrees must pay the cost and confusion of an extra execution checkout.

**Independent Test**: Start a run from a clean non-default branch using `no-mistakes --no-worktree --yolo` and confirm the run executes in the current git worktree root without creating a new directory under the no-mistakes worktree store.

**Acceptance Scenarios**:

1. **Given** the user is inside a clean initialized git worktree on a non-default branch, **When** they run `no-mistakes --no-worktree --yolo`, **Then** no-mistakes starts a pipeline using the current git worktree root as the execution directory.
2. **Given** the user invokes the command from a subdirectory of that worktree, **When** the run starts in current-worktree mode, **Then** the execution directory is the git worktree root rather than the shell subdirectory.
3. **Given** the user passes `--yolo`, **When** the run reaches approval gates, **Then** the behavior is identical to the existing `--yes` behavior and grants no additional approval power.

---

### User Story 2 - Drive Current-Worktree Runs Through AXI (Priority: P1)

As a headless agent using AXI, I need to start and drive the same current-worktree pipeline with an explicit intent, so agent workflows can validate the branch without relying on a gate-remote push.

**Why this priority**: The agent-facing workflow is the expected automation path and must support the same execution mode as the root command.

**Independent Test**: Run `no-mistakes axi run --intent "..." --no-worktree --yolo` from a clean feature branch and confirm it starts or resumes a compatible current-worktree run and drives it using the existing AXI gate behavior.

**Acceptance Scenarios**:

1. **Given** no compatible active run exists, **When** AXI is invoked with `--intent`, `--no-worktree`, and `--yolo`, **Then** no-mistakes starts a new current-worktree run directly and drives it with the same rules as `--yes`.
2. **Given** AXI is invoked with `--no-worktree` but no intent while starting a new run, **When** no compatible run exists, **Then** the command fails with the existing explicit-intent requirement for AXI run starts.
3. **Given** a compatible current-worktree run already exists for the same branch and head, **When** AXI is invoked again with current-worktree mode, **Then** it resumes or drives that run instead of starting a duplicate run.

---

### User Story 3 - Preserve Gate Safety And Review Scope (Priority: P2)

As a maintainer, I need current-worktree mode to keep the same branch safety rules and review the full branch diff, so opting out of an extra worktree does not weaken what a passed no-mistakes gate means.

**Why this priority**: Current-worktree mode intentionally changes the execution boundary, but it must not bypass branch hygiene, review scope, or the fixed pipeline.

**Independent Test**: Attempt current-worktree starts from dirty, detached, default-branch, and valid feature-branch states, then confirm only the valid clean feature branch starts and that review scope uses the full branch diff against the default branch base.

**Acceptance Scenarios**:

1. **Given** the current worktree is dirty or has uncommitted changes, **When** current-worktree mode is requested, **Then** the run is rejected before pipeline execution starts.
2. **Given** the current worktree is on a detached head or the default branch, **When** current-worktree mode is requested, **Then** the run is rejected with a clear recovery message.
3. **Given** a valid current-worktree run starts, **When** review runs, **Then** it reviews the full branch diff against the default-branch merge base rather than only the most recent commit.
4. **Given** current-worktree mode is used, **When** the pipeline executes, **Then** the normal review, test, document, lint, push, PR, and CI steps still run unless the user explicitly configured supported skip behavior.

---

### User Story 4 - Make Current-Worktree Runs Visible And Recoverable (Priority: P3)

As a user watching the TUI, AXI output, or daemon status, I need to know whether a run is isolated or using my current worktree, so I understand where automated fixes and commits may appear and how to recover after failures.

**Why this priority**: Current-worktree mode is safe only if the execution boundary is explicit and cleanup behavior is predictable.

**Independent Test**: Start current and isolated runs, inspect status output, AXI rendering, TUI labels, and PR-facing summaries, and confirm each surface reports the mode, safe current run directory label where relevant, and matching finding/fix/evidence counts derived from the same persisted run data.

**Acceptance Scenarios**:

1. **Given** a current-worktree run is active, **When** status, AXI, or the TUI renders the run, **Then** the output labels it as a current-worktree run and exposes the resolved work directory without noisy repetition.
2. **Given** a current-worktree run fails after automated fixes or generated commits, **When** the failure is reported, **Then** no-mistakes explains that those commits remain in the current worktree for inspection, amend, revert, or rerun, and terminal CLI, AXI, status, TUI, and generated report output marks the gate evidence as incomplete with the terminal reason and last trustworthy completed step.
3. **Given** daemon recovery or cleanup runs after a current-worktree run, **When** stale run state is handled, **Then** no-mistakes may mark the run failed but must not remove the current worktree directory.

---

### User Story 5 - Preserve Existing Isolated Defaults (Priority: P4)

As an existing no-mistakes user, I need all default commands and existing `--yes` behavior to remain unchanged, so the normal disposable-worktree workflow stays predictable.

**Why this priority**: The new mode is opt-in. Existing users should not see behavior changes unless they request current-worktree mode or use the new alias.

**Independent Test**: Run existing root, wizard, push-triggered, rerun, and AXI `--yes` flows without `--no-worktree`, and confirm they still create and clean up no-mistakes-owned worktrees as before.

**Acceptance Scenarios**:

1. **Given** the user does not pass `--no-worktree`, **When** a new run starts, **Then** no-mistakes uses the existing isolated no-mistakes-owned worktree behavior.
2. **Given** the user passes `--yes`, **When** gates are reached, **Then** the existing auto-resolution behavior is unchanged.
3. **Given** the user passes both `--yes` and `--yolo`, **When** the command starts, **Then** no-mistakes treats this as auto-resolution enabled and does not fail due to duplicate aliases.

### Edge Cases

- The command is invoked from a nested subdirectory in an attached git worktree whose registered no-mistakes repo record points at the main checkout.
- The repo has not been initialized with no-mistakes before current-worktree mode is requested.
- The current branch is dirty, detached, unborn, or equal to the default branch; dirty means tracked changes or untracked non-ignored files are present, while ignored files alone are allowed.
- The default branch remote ref is missing, stale, or temporarily unavailable while computing the full branch base; current-worktree mode attempts one non-interactive default-branch ref refresh and then rejects the run if the base still cannot be proven.
- The root command is invoked without explicit intent and existing intent inference cannot produce usable intent in non-interactive or `--yolo` mode.
- An active run exists for the same branch but a different head commit, selected worktree mode, or current-mode resolved work directory.
- An active isolated run exists for the same repo and branch when current-worktree mode is requested, or an active current-worktree run exists when default isolated mode is requested.
- Setup fails after the current work directory is resolved but before the executor starts.
- The process crashes or daemon recovery runs while a current-worktree run is active or stale, and regenerated reports must rely on persisted run evidence rather than recomputing from mutable current worktree state.
- Current-worktree auto-fixes create commits and any later terminal outcome is reported, including checks-passed, passed, failed, cancelled, or stale-recovered.
- Both `--yes` and `--yolo` are present.
- The user misspells the binary name or requests a typo alias.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST add a current-worktree mode selected by `--no-worktree` on both the root `no-mistakes` command and `no-mistakes axi run`.
- **FR-002**: In current-worktree mode, the system MUST NOT create an additional no-mistakes-owned worktree under the no-mistakes worktree store.
- **FR-003**: In current-worktree mode, the system MUST resolve the execution directory to the current git worktree root, including when the command is invoked from a subdirectory.
- **FR-004**: The system MUST keep default behavior unchanged: runs without `--no-worktree` continue to use the existing isolated no-mistakes-owned worktree lifecycle.
- **FR-005**: The system MUST add `--yolo` as an alias for existing `--yes` behavior on the root command and AXI run command.
- **FR-006**: `--yolo` MUST NOT grant approval behavior beyond existing `--yes` auto-resolution behavior.
- **FR-007**: Passing both `--yes` and `--yolo` MUST be accepted as auto-resolution enabled, not treated as a conflict.
- **FR-008**: The root `no-mistakes --no-worktree --yolo` command MUST be able to start a new current-worktree run without requiring an explicit `--intent`, using existing intent inference behavior where available; inferred intent MUST be persisted and rendered only as a redacted, bounded summary, never as raw transcript or log text. If usable intent cannot be inferred in non-interactive or `--yolo` mode, the command MUST fail before starting with recovery guidance for providing or generating intent, and that guidance MUST NOT echo transcript snippets.
- **FR-009**: `no-mistakes axi run --no-worktree --yolo` MUST retain the existing `--intent` requirement when starting a new run.
- **FR-010**: Current-worktree starts MUST use a direct CLI-to-daemon start path and MUST NOT depend on `git push no-mistakes` to trigger a post-receive hook.
- **FR-011**: Current-worktree starts MUST preserve existing preflight requirements: initialized repo, attached branch, non-default branch, and clean committed worktree; clean committed worktree means tracked changes and untracked non-ignored files are rejected, while ignored files alone do not block the run.
- **FR-012**: Current-worktree review scope MUST cover the full current branch diff against the default branch base; if a trustworthy base cannot be resolved, the system MUST attempt one non-interactive default-branch ref refresh and then reject the run if the base remains unavailable. Base-resolution evidence MUST be persisted, including default branch ref, proven merge-base SHA when available, whether refresh was attempted, refresh result or error, and rejection reason; missing-base rejection MUST render as a distinct `rejected_no_trustworthy_base` outcome.
- **FR-013**: Current-worktree mode MUST still run the normal pipeline: review, test, document, lint, push, PR, and CI, subject only to existing supported skip behavior. Every skipped, deferred, or informational gate decision MUST be persisted with its source and rendered distinctly from passed, fixed, or clean outcomes in status, AXI, TUI, generated reports, and PR-facing summaries.
- **FR-014**: If current-worktree fixes are committed and a later pipeline step fails, the system MUST leave those commits in the current worktree and MUST NOT auto-revert them. Current-worktree fix attempts MUST persist fix outcome records that distinguish proposed, attempted, committed, and failed fixes, include commit SHAs when commits are created, and drive applied-fix claims from those records rather than from prose summaries.
- **FR-015**: Run metadata MUST persist the worktree mode and resolved work directory for each run, validate worktree mode against the allowed values, and validate current-mode work directories as canonical absolute git worktree roots. Readers MUST treat malformed, empty, stale, or non-canonical current-mode work directory metadata as degraded/incomplete state, not as `isolated` and not as safe for cleanup.
- **FR-016**: Valid worktree mode values MUST include `isolated` for the existing default behavior and `current` for `--no-worktree`.
- **FR-017**: Existing and migrated runs created before worktree metadata existed MUST default to `isolated` mode when no explicit worktree metadata exists. Newly created current-worktree runs MUST persist `worktree_mode=current` and the canonical resolved work directory in the same durable create/update boundary before the run becomes recoverable or cleanup-eligible. If recovery observes a new-format run whose worktree metadata is missing or invalid, it MUST mark metadata availability as `not_recorded`, disable directory cleanup for that run, and render final evidence incomplete instead of inferring `isolated`.
- **FR-018**: Status, AXI rendering, and TUI rendering MUST make current-worktree runs visibly distinct from isolated runs and MUST expose stable structured fields where the surface is structured: `worktree_mode` (`current` or `isolated`), a safe `work_dir_label`, and `current_worktree_warning` for current-mode runs. Missing or malformed required current-worktree fields MUST render the run as incomplete/degraded rather than as a normal passed or isolated run.
- **FR-019**: Current-worktree user-facing output MUST warn or explain that pipeline fixes may modify the current checkout in CLI, AXI, status, TUI, and PR-facing surfaces; this warning MUST NOT add a blocking confirmation beyond existing `--yes` or `--yolo` behavior. The warning MUST appear at start/pre-start acknowledgement, active run rendering, fix-in-progress or fix-review, checks-passed/passed terminal output, failure/cancellation output, and stale-run recovery output; once a run exists, the warning MUST include the run/report reference.
- **FR-020**: The system MUST never remove or clean up the resolved current work directory for runs whose worktree mode is `current`, including setup failure, normal completion, panic handling, and stale-run recovery.
- **FR-021**: Isolated runs MUST retain existing worktree cleanup behavior.
- **FR-022**: Active-run selection MUST prevent mixing current and isolated modes on the same repo and branch; incompatible requests MUST be rejected with the exact active run and resume or abort guidance. Current-worktree runs that are aborted, cancelled, stale-recovered, setup-failed, or superseded by an actual replacement run MUST persist a structured terminal reason and, when applicable, the successor run ID and head SHA so status, AXI, TUI, and regenerated reports do not present partial evidence as an ordinary failure.
- **FR-023**: A current-worktree request MAY resume or drive an active current-worktree run only when the repo, branch, head commit, selected worktree mode, resolved current work directory, review base, and immutable start-shape fields are compatible. Resume MUST NOT replace the active run's persisted intent, skip configuration, approval mode, or review base with values from the new request; differing requested values MUST be rejected or rendered as ignored with explicit guidance.
- **FR-024**: User-facing docs, generated no-mistakes agent guidance, and PR-facing generated summaries MUST describe both command forms, the meaning of `--no-worktree`, and the fact that `--yolo` is an alias for `--yes`. When a PR is created or updated from a current-worktree run, the PR summary MUST include current-worktree mode, a safe work-directory label, fix count or commit references when fixes occurred, unresolved/degraded evidence state, and the run/report reference.
- **FR-025**: The implementation MUST support only the correctly spelled `no-mistakes` command and MUST NOT add typo aliases.
- **FR-026**: The spec directory MUST preserve a sanitized companion origin reference that records the source requirement, purpose, sub-agent scouting, and source-code context for later Speckit phases. The origin reference MUST preserve file paths, symbols, decisions, and concise evidence summaries only; it MUST NOT copy raw sub-agent transcripts, raw logs, diff hunks, secrets, or long code excerpts when a location reference is sufficient.

### Key Entities

- **Run**: A branch-scoped no-mistakes pipeline execution with status, steps, branch/head/base metadata, worktree mode, resolved work directory, evidence reconstruction inputs, and fix provenance for current-worktree commits. Fix provenance includes actor/source, source finding or decision, decision type, commit SHA when applicable, and whether the change was automated or user-authored.
- **Worktree Mode**: The execution-boundary classification for a run. Structured metadata uses `worktree_mode: isolated|current`; user-facing labels MUST avoid relying on the bare enum words alone and SHOULD use plain labels such as "disposable no-mistakes checkout" for isolated mode and "uses this checkout" for current mode.
- **Current Work Directory**: The canonical git worktree root selected for a current-mode run and included in current-mode resume compatibility.
- **Start Request**: A CLI or AXI request to start or resume a run, including branch, head, base, skip settings, intent where required, and selected worktree mode.
- **Active Run Compatibility**: The rules that decide whether an existing active run can be resumed or must block a new request because its mode, branch, head commit, or current-mode work directory differs.
- **Review Base**: The default-branch merge base used to ensure current-mode review covers the full branch diff, with one non-interactive default-branch ref refresh before rejecting an unavailable base.
- **Origin Reference**: The companion planning artifact that preserves why this feature exists and where future implementation work should begin, with each preserved item labeled as user requirement, applied clarification, agent-derived evidence, or non-authoritative context. Future planning MUST treat only user requirements and applied clarifications as product authority unless another spec explicitly approves more.

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: Current-worktree mode does not change normal `origin` behavior or the meaning of a passed no-mistakes gate. The fixed pipeline still gates upstream push, PR, and CI after local validation.
- **Isolation/User Control**: The default remains disposable isolated execution. Current-worktree mode is an explicit opt-in for users already operating inside a tool-owned or otherwise intentional git worktree; the mode must be visibly labeled, clean before start, and never silently cleaned up by no-mistakes.
- **Evidence Plan**: Implementation planning must include targeted tests for flag parsing, preflight rejection, direct run start, active-run compatibility, metadata persistence, rendering, cleanup boundaries, and full default-mode regression. Cross-process daemon behavior needs focused daemon tests or reviewer-visible evidence.
- **Agent/Interface Contracts**: AXI and generated agent guidance must distinguish `--yolo` as an alias for `--yes`, keep `--intent` requirements clear, and expose current-worktree warnings in machine-consumable and human-readable output. Current-worktree structured report fields MUST be schema-validated before rendering or agent consumption; missing or malformed required fields MUST make the affected report explicitly incomplete/degraded rather than successful.
- **Docs/Generated Artifacts**: CLI docs, agent skill guidance, and any generated help or reference text must be updated because this feature adds user-visible flags and a new execution mode. Generated summaries for CLI, AXI, TUI/status, PR bodies, docs, and agent guidance SHOULD use bounded redacted summaries plus artifact/run references, and MUST NOT inline raw transcripts, raw logs, secrets, long code excerpts, or diff hunks except in existing explicit diagnostic or approval surfaces designed to show that detail.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of current-worktree acceptance tests, no-mistakes starts without creating a no-mistakes-owned worktree directory for the run.
- **SC-002**: In 100% of tested current-worktree starts from subdirectories, the recorded execution directory equals the git worktree root.
- **SC-003**: In 100% of tested dirty, detached-head, default-branch, or uninitialized current-worktree starts, the command rejects before pipeline execution and reports a recovery path.
- **SC-004**: In 100% of tested `--yolo` and `--yes` equivalence cases, the selected approval behavior matches existing `--yes` behavior.
- **SC-005**: In 100% of default-mode regression tests, isolated runs still create and clean up no-mistakes-owned worktrees as before.
- **SC-006**: In 100% of current-worktree cleanup and recovery tests, no-mistakes never removes the recorded current work directory.
- **SC-007**: In 100% of run status, AXI, and TUI rendering tests for current-mode runs, the output exposes worktree mode and a safe compact work-directory label by default. Full canonical absolute paths may appear only in explicit verbose/debug fields or logs intended for local diagnostics, and tests MUST prevent repeated sensitive path detail across normal status, AXI, TUI, and PR-facing summaries.
- **SC-008**: A future planner can identify the original requirement, the implementation purpose, the conflicting older remote interpretation, and first source files to inspect from the spec directory in under one minute.
- **SC-009**: In 100% of tested missing-base cases, current-worktree mode attempts one non-interactive default-branch ref refresh and rejects before pipeline execution if the base still cannot be proven.
- **SC-010**: In 100% of active-run compatibility tests, incompatible mode, head, or current-mode work directory requests are rejected with actionable resume or abort guidance.
- **SC-011**: In 100% of clean-worktree preflight tests, tracked changes and untracked non-ignored files block current-worktree starts, while ignored files alone do not block the run.

## Assumptions

- Users requesting `--no-worktree` understand that the current git worktree is the intended place for pipeline fixes and commits.
- Archon-created worktrees are a primary use case, but normal clean non-default checkouts are also supported when the user explicitly selects current-worktree mode.
- Existing intent inference is the root command fallback when no explicit intent is supplied, but non-interactive or `--yolo` starts fail with recovery guidance if inference cannot produce usable intent.
- Existing AXI behavior continues to require explicit intent for new headless starts.
- The existing pipeline can execute from an explicit work directory once the daemon chooses that directory.
- The feature does not require a new permission model, a new finding taxonomy, or changes to supported pipeline step ordering.
- The older remote `002-no-worktree-yolo` branch represents a different safety-guard interpretation and is not the source of truth for this feature.
