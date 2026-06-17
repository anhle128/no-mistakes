# Feature Specification: No-Worktree YOLO Guard

**Feature Branch**: `002-no-worktree-yolo`  
**Created**: 2026-06-17  
**Status**: Draft  
**Input**: User description: "read requirement in create detail spec for no-worktree-yolo.md, save reference for next tasks know the origin purpose, spawn sub-agent for help, use dedicate agent, scout source code to know the context, create detail spec"

## Clarifications

### Session 2026-06-17

- Q: What proof should be required before a run is classified safe for unattended source-changing YOLO actions? → A: Require a verified disposable source workspace for unattended source-changing actions; allow evidence/test boundaries only for non-source artifacts.
- Q: Where should the safe/unsafe/unknown execution-boundary classification be recorded as the authoritative state? → A: Persist boundary classification on the Run record and refresh it before each automatic gate action.
- Q: If a run was safe when YOLO consent was given but later becomes unknown before another automatic action, what should happen to the existing consent? → A: Pause unattended actions while unknown; resume for the same run only after safe proof is restored.
- Q: Which user-facing surfaces must explain that YOLO was withheld because the boundary is unsafe or unknown? → A: All gate-driving or gate-observing surfaces: TUI, AXI, headless CLI output, terminal status, and generated agent guidance.
- Q: What audit record is required when unattended YOLO is allowed, withheld, or not requested for a gate? → A: Persist a run event and expose current gate status for allowed, withheld, and not-requested YOLO states.
- Q: On unsafe or unknown-boundary runs, how should an explicit manual "fix" action behave? → A: Allow manual fix after explicit per-gate user decision and record it as a manual source-changing action.
- Q: Which external side effects count as PR-advancing or remote-advancing actions that unattended YOLO must withhold on unsafe or unknown boundaries? → A: Git push plus PR create/update/merge and provider status/comment actions that advance external review state.
- Q: What identity should define "the same gate" for duplicate automatic-response prevention across reattach, restart, and repeated status events? → A: Run ID plus gate ID or step plus a gate version/fingerprint for the current pending decision.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Block Unattended YOLO Without Isolation (Priority: P1)

As a developer or coding agent using YOLO-style automation, I need the system to verify that unattended source-changing gate resolution is running inside a disposable source workspace before it fixes or approves anything, so my day-to-day checkout and external state are not changed by an unattended decision.

**Why this priority**: This is the core safety contract. If the system cannot prove the run boundary, unattended approval must not proceed.

**Independent Test**: Can be fully tested by attempting to enable unattended YOLO on both a safe isolated run and an unsafe or unknown-boundary run, then confirming that only the safe run performs automatic gate actions.

**Acceptance Scenarios**:

1. **Given** a gate run with a verified disposable execution boundary, **When** the user enables YOLO or equivalent unattended consent, **Then** the system may auto-resolve paused gates according to the existing YOLO rules.
2. **Given** a gate run whose execution boundary is the user's primary checkout or cannot be verified, **When** YOLO or equivalent unattended consent is requested, **Then** the system must not automatically fix, approve, skip, push, create or update PR state, or otherwise advance the gate.
3. **Given** a non-isolated or unknown-boundary run paused at a gate, **When** the user manually chooses approve, fix, skip, or cancel, **Then** the system accepts the explicit per-gate human decision and records manual fix as manual source-changing intent.

---

### User Story 2 - Explain Why Automation Was Withheld (Priority: P2)

As a developer or agent operator, I need a clear explanation when YOLO is withheld because the run is not safely isolated, so I know whether to continue manually, restart through the gate, or correct setup.

**Why this priority**: A fail-closed guard is useful only if the user can understand and recover from it without reading logs or source code.

**Independent Test**: Can be tested by triggering YOLO on an unsafe or unknown-boundary run and checking that every user-facing surface reports the reason, current gate, and available recovery path.

**Acceptance Scenarios**:

1. **Given** an unsafe or unknown-boundary run, **When** YOLO is requested from a gate-driving or gate-observing surface, **Then** the TUI, AXI, headless CLI output, terminal status, and generated agent guidance show that unattended YOLO is disabled for this run and leave the gate awaiting manual action.
2. **Given** an unsafe or unknown-boundary run, **When** an agent uses headless unattended consent, **Then** the command output clearly states that consent was not used because the run boundary is not safe for unattended resolution.
3. **Given** a run that is safe for unattended YOLO, **When** YOLO proceeds, **Then** the normal status output does not add unnecessary warnings or require extra user steps.

---

### User Story 3 - Preserve Existing Isolated YOLO Behavior (Priority: P3)

As an existing no-mistakes user, I need YOLO to behave exactly as it does today when the run is safely isolated, so the safety guard does not make the normal disposable-worktree workflow slower or more surprising.

**Why this priority**: The feature should narrow unsafe automation, not redefine the established YOLO experience.

**Independent Test**: Can be tested with an isolated run that contains actionable findings, no-op findings, and fix-review gates, then comparing the observed outcomes to the documented YOLO behavior.

**Acceptance Scenarios**:

1. **Given** an isolated run with actionable findings, **When** YOLO is enabled, **Then** the system selects every current actionable finding and runs one fix round.
2. **Given** an isolated run in fix review after an automatic fix round, **When** YOLO observes the gate, **Then** the system approves the fix-review result instead of starting an unbounded loop.
3. **Given** an isolated run with only informational no-op findings, **When** YOLO is enabled, **Then** the system approves the gate without attempting a fix.

---

### User Story 4 - Preserve Requirement Origin for Future Tasks (Priority: P4)

As a future maintainer planning or implementing this feature, I need the spec directory to preserve the origin, purpose, and source-context references for the no-worktree-yolo requirement, so later planning does not have to rediscover the same background.

**Why this priority**: The user explicitly asked for a saved reference so follow-up tasks understand the requirement's origin and purpose.

**Independent Test**: Can be tested by opening the companion origin reference and confirming it states the request, the inferred purpose, the source-scout findings, and the files future tasks should inspect first.

**Acceptance Scenarios**:

1. **Given** the completed spec directory, **When** a future task starts, **Then** it can find a companion no-worktree-yolo reference with the original request and source-context anchors.
2. **Given** the stakeholder spec, **When** product planning continues, **Then** implementation-specific source anchors remain outside the main spec while the requirement purpose remains clear.

### Edge Cases

- YOLO is toggled after a gate is already awaiting approval on a run whose boundary is unknown.
- A daemon restart, TUI reattach, or headless command loses in-memory state about prior automatic actions.
- A run starts safe but later cannot prove its boundary before a subsequent automatic gate action; unattended actions pause while unknown and resume for the same run only after safe proof is restored.
- A gate contains only no-op findings, but approving it would still advance toward push or PR side effects.
- A user provides broad unattended consent through an agent command, but the run is not safely isolated.
- A configured evidence directory is allowed, but source writes outside the disposable run boundary are not.
- A manual user action is taken on an unsafe or unknown-boundary run after unattended automation was withheld.
- Multiple UI or command events report the same paused gate while a response is in flight; duplicate prevention uses run ID plus gate ID or step plus a gate version or fingerprint.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST define unattended YOLO as any mode that resolves approval gates without a fresh per-gate human decision, including terminal YOLO mode, headless `--yes` consent, and agent-skill consent that drives gates unattended.
- **FR-002**: System MUST persist the safe, unsafe, or unknown execution-boundary classification on the Run with the verifier inputs and freshness metadata needed to explain the decision, but persisted classification MUST NOT by itself authorize unattended action. Before each automatic gate action, and before any later automatic action after reattach, rerun, daemon restart, status refresh, or workspace movement, the system MUST recompute the classification against the current workspace identity and degrade to unknown when current proof is stale, missing, or inconsistent.
- **FR-003**: A run MUST be considered safe for unattended source-changing YOLO only when a controller-owned verifier independently proves that intentional source-changing work is bounded to a disposable source workspace for the current Run, using canonical realpaths and trusted Git workspace metadata rather than user-supplied or agent-supplied path claims; the verifier MUST reject unknown, symlink-escaping, nested, or metadata-inconsistent boundaries. Explicitly allowed evidence/test boundaries may authorize non-source artifacts only and MUST NOT authorize source writes outside the disposable run workspace.
- **FR-004**: When the run boundary is unsafe or unknown, the system MUST fail closed by withholding automatic fix, approve, skip, git push, PR create/update/merge, and all automatic remote provider write actions, including PR body updates, comments, statuses or check-run writes, labels, and metadata changes; read-only provider queries may continue, and the gate MUST remain available for manual user action.
- **FR-005**: Manual approve, fix, skip, and cancel actions MUST remain available on unsafe or unknown-boundary runs only as explicit responses to the current pending gate, not as reuse of broad unattended consent. Manual fix MUST be recorded as explicit per-gate manual source-changing intent distinguishable from unattended automatic decisions, including the response actor type, approval surface, gate identity, and whether any subsequent `--yes` or YOLO continuation was requested; generated agent guidance MUST require a human decision before an agent submits a manual response on an unsafe or unknown-boundary run.
- **FR-006**: On safe runs, existing YOLO behavior MUST be preserved: actionable findings are selected for one fix round, fix-review gates are approved, no-op-only gates are approved, and duplicate automatic responses for the same gate are prevented using run ID plus gate ID or step plus a gate version or fingerprint.
- **FR-007**: Broad unattended consent MUST NOT override the safety boundary. A user or agent may consent to automatic gate handling only after the run is safe for unattended YOLO.
- **FR-008**: TUI, AXI, headless CLI output, terminal status, and generated agent guidance MUST explain withheld YOLO with the requested mode, the current gate, the boundary status, and at least one recovery option.
- **FR-009**: Run history or status output MUST persist a run event and expose current gate status showing whether unattended YOLO was allowed, withheld, or not requested for the current gate, so users and agents can report accurately.
- **FR-010**: System MUST preserve explicit gate semantics: normal `origin` behavior remains unchanged, upstream push still happens only after the local gate passes, and this feature MUST NOT reorder or remove pipeline steps.
- **FR-011**: System MUST preserve the existing finding action model: `auto-fix`, `ask-user`, and `no-op` keep their current meaning, with `ask-user` still requiring human judgment unless safe unattended consent is active.
- **FR-012**: Documentation and generated agent guidance MUST describe when unattended YOLO is allowed, when it is withheld, and how users should recover.
- **FR-013**: The spec directory MUST include a companion origin reference for future tasks that records the original request, inferred purpose, source-scout findings, and relevant code/docs anchors.
- **FR-014**: Error and recovery behavior MUST be idempotent: repeated attempts to enable YOLO on an unsafe or unknown-boundary run must not enqueue duplicate responses or change the paused gate state for the same controller-owned gate identity. The gate identity MUST be computed from immutable current decision inputs for the run, including run ID, gate ID or step, gate status, current step round or gate version, and the decision fingerprint; user-facing status text, generated agent guidance, or mutable agent-provided metadata MUST NOT create a new actionable gate identity.
- **FR-015**: If a run with existing unattended consent becomes unknown before a later automatic action, the system MUST pause unattended actions while the boundary is unknown and may resume for the same run only after safe proof is restored.

### Key Entities

- **Run**: A branch-scoped gate execution with status, steps, findings, approval state, persisted execution-boundary classification, audit events, and user-facing outputs.
- **Execution Boundary**: The declared write boundary for a run, classified as safe, unsafe, or unknown for unattended YOLO and refreshed before automatic gate actions.
- **YOLO Consent**: A user or agent request to resolve gates without a fresh per-gate human decision.
- **Gate Decision**: A fix, approve, skip, or cancel action, including whether it came from a manual decision or safe unattended automation, plus the gate identity made from run ID, gate ID or step, and gate version or fingerprint.
- **Finding**: A review, test, documentation, lint, or CI item with an action classification that determines whether it can be fixed automatically, requires user judgment, or is informational.
- **Origin Reference**: The companion planning artifact that preserves why this feature exists and where future implementation work should start.

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: The feature keeps `git push no-mistakes` as the explicit opt-in path, leaves normal `origin` behavior unchanged, and blocks unattended git push, PR creation or update, merge, and provider review advancement when the run boundary is not safe enough to preserve the meaning of a passed gate.
- **Isolation/User Control**: The feature directly enforces the principle that automated source-changing cleanup may proceed only inside a verified disposable source workspace. Unsafe or unknown boundaries fall back to manual approval, keeping the user in charge of intent and side effects.
- **Evidence Plan**: Planning and implementation should include targeted regression coverage for safe YOLO, unsafe/unknown-boundary withholding, reattach/restart behavior, duplicate-response prevention, and user-facing TUI/AXI/agent guidance. Cross-process or git-boundary behavior should include reviewer-visible evidence if unit coverage cannot fully prove it.
- **Agent/Interface Contracts**: TUI, AXI, terminal, headless, and agent-skill outputs must use plain labels that distinguish allowed unattended automation from withheld automation. `ask-user` findings must remain visible and accurately reported.
- **Docs/Generated Artifacts**: User-visible docs and generated agent guidance need updates because the feature changes when broad unattended consent is honored. The companion origin reference is required for future planning continuity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of unsafe or unknown-boundary runs tested, unattended YOLO performs zero automatic fix, approve, skip, git push, PR create/update/merge, or provider review-advancing actions and leaves the gate awaiting manual action.
- **SC-002**: In 100% of safe isolated runs tested, existing YOLO acceptance behavior is preserved for actionable findings, no-op findings, fix-review gates, and duplicate gate events across reattach, restart, and repeated status observations.
- **SC-003**: At least 95% of users or agent operators reviewing withheld YOLO output across TUI, AXI, terminal status, headless output, or generated agent guidance can identify the reason and next manual or recovery action without opening logs.
- **SC-004**: In boundary-audit tests, zero intentional source changes are written outside the verified disposable source workspace during unattended YOLO; allowed evidence/test boundaries produce only non-source artifacts.
- **SC-005**: Safe isolated YOLO runs that previously required no manual intervention still complete without extra user steps in at least 95% of representative acceptance cases.
- **SC-006**: Future planning tasks can locate the origin reference, original request, inferred purpose, and first source anchors from the spec directory in under one minute.

## Assumptions

- "No-worktree-yolo" is interpreted conservatively as a guard against unattended YOLO when the system cannot prove an isolated or bounded run workspace.
- The feature concerns unattended automation only; explicit per-gate manual user actions remain valid even when unattended YOLO is withheld.
- Existing pipeline order, finding action names, and normal gate remote semantics are out of scope for redesign; this feature only constrains unattended advancement around PR and remote side effects.
- The safety boundary is a product contract and user-control guarantee; it is not assumed to be an operating-system sandbox.
- The primary gate-driving or gate-observing surfaces are TUI, AXI, terminal status, headless command output, and generated agent guidance that drives the same gate workflow.
