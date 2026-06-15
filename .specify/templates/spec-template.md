# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`  
**Created**: [DATE]  
**Status**: Draft  
**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
-->

### User Story 1 - [Brief Title] (Priority: P1)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently - e.g., "Can be fully tested by [specific action] and delivers [specific value]"]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]
2. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 2 - [Brief Title] (Priority: P2)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 3 - [Brief Title] (Priority: P3)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

[Add more user stories as needed, each with an assigned priority]

### Edge Cases

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right edge cases.
-->

- What happens when the gate run is interrupted, canceled, or superseded by a newer push?
- How does the system handle missing credentials, missing agent/provider binaries, or non-interactive Git failures?
- What approval path is used for findings that require human judgment?
- What evidence is produced when user intent is available but automated validation is incomplete?
- How are cross-platform path, shell, daemon, and service-manager differences handled?

## Requirements *(mandatory)*

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right functional requirements.
-->

### Functional Requirements

- **FR-001**: System MUST [specific capability tied to the no-mistakes gate]
- **FR-002**: System MUST preserve explicit `git push no-mistakes` gate semantics and MUST NOT alter normal `origin` behavior unless the feature explicitly concerns Git remote setup
- **FR-003**: Users MUST be able to [key TUI, AXI, CLI, or git workflow interaction]
- **FR-004**: System MUST keep intentional writes inside the disposable worktree or configured evidence directory
- **FR-005**: System MUST provide actionable findings, logs, or recovery messages for failure paths
- **FR-006**: System MUST update docs, config reference, or generated skill content when user-visible behavior changes

*Example of marking unclear requirements:*

- **FR-007**: System MUST authenticate users via [NEEDS CLARIFICATION: auth method not specified - email/password, SSO, OAuth?]
- **FR-008**: System MUST retain user data for [NEEDS CLARIFICATION: retention period not specified]

### Key Entities *(include if feature involves data)*

- **Run**: [Branch-scoped pipeline execution, status, steps, findings, approvals, and fix rounds]
- **Step Result**: [Per-step status, findings, tested evidence, artifacts, duration, and logs]
- **Approval Gate**: [Finding IDs, actions, user instructions, added findings, and selected resolution]
- **Agent Invocation**: [Prompt, worktree CWD, environment, structured schema, output, and retry behavior]

## Constitution Alignment *(mandatory)*

- **Gate Semantics**: [How this feature preserves the meaning of a passed gate and normal `origin` behavior]
- **Isolation/User Control**: [Where writes happen, which paths require approval, and how destructive or credential-gated actions pause]
- **Evidence Plan**: [Targeted tests, `go test -race ./...`, `make lint`, e2e/docs validation, and reviewer-visible artifacts]
- **Agent/Interface Contracts**: [Structured output, transcript intent, AXI/TUI labels, and supported-agent impact]
- **Docs/Generated Artifacts**: [README/docs/config/generated skill updates or N/A with rationale]

## Success Criteria *(mandatory)*

<!--
  ACTION REQUIRED: Define measurable success criteria.
  These must be technology-agnostic and measurable.
-->

### Measurable Outcomes

- **SC-001**: [Measurable metric, e.g., "Users can complete account creation in under 2 minutes"]
- **SC-002**: [Measurable metric, e.g., "System handles 1000 concurrent users without degradation"]
- **SC-003**: [User satisfaction metric, e.g., "90% of users successfully complete primary task on first attempt"]
- **SC-004**: [Business metric, e.g., "Reduce support tickets related to [X] by 50%"]

## Assumptions

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right assumptions based on reasonable defaults
  chosen when the feature description did not specify certain details.
-->

- [Assumption about target users, e.g., "Users have stable internet connectivity"]
- [Assumption about scope boundaries, e.g., "Mobile support is out of scope for v1"]
- [Assumption about data/environment, e.g., "Existing authentication system will be reused"]
- [Dependency on existing system/service, e.g., "Requires access to the existing user profile API"]
