<!--
Sync Impact Report
Version change: template (unversioned) -> 1.0.0
Modified principles:
- Placeholder Principle 1 -> I. Explicit Gate Semantics
- Placeholder Principle 2 -> II. Isolated Execution and User Control
- Placeholder Principle 3 -> III. Evidence-First Quality Gates
- Placeholder Principle 4 -> IV. Agent-Agnostic Structured Contracts
- Placeholder Principle 5 -> V. Simplicity, Idempotence, and Recovery
Added sections:
- Operational Constraints
- Development Workflow
Removed sections:
- Placeholder Section 2
- Placeholder Section 3
Templates requiring updates:
- ✅ updated .specify/templates/plan-template.md
- ✅ updated .specify/templates/spec-template.md
- ✅ updated .specify/templates/tasks-template.md
- ✅ updated .specify/templates/checklist-template.md
- ✅ no update needed .specify/extensions/git/commands/speckit.git.commit.md
- ✅ no update needed .specify/extensions/git/commands/speckit.git.feature.md
- ✅ no update needed .specify/extensions/git/commands/speckit.git.initialize.md
- ✅ no update needed .specify/extensions/git/commands/speckit.git.remote.md
- ✅ no update needed .specify/extensions/git/commands/speckit.git.validate.md
- ✅ no update needed README.md
- ✅ no update needed docs/src/content/docs/start-here/introduction.md
- ✅ no update needed docs/src/content/docs/start-here/quick-start.md
- ✅ no update needed docs/src/content/docs/concepts/gate-model.md
- ✅ no update needed docs/src/content/docs/concepts/pipeline.md
- ✅ no update needed docs/src/content/docs/guides/agents.md
Follow-up TODOs: None
-->
# no-mistakes Constitution

## Core Principles

### I. Explicit Gate Semantics
Every user-facing claim about a passed gate MUST preserve the documented meaning:
the branch was checked against fresh upstream, review/test/document/lint happened
before any upstream push, and push/PR/CI happened only after the local gate passed.
Normal `origin` behavior MUST remain unmodified; `git push no-mistakes` is the
explicit opt-in path. Rationale: trust in this tool depends on one named remote
having stable, reviewable semantics instead of silently changing normal Git.

### II. Isolated Execution and User Control
Pipeline work MUST run in a disposable worktree or an explicitly bounded test
directory, never by intentionally mutating the user's day-to-day working tree.
Agents and helper commands MUST keep intentional writes inside the run worktree,
except for configured evidence directories. Any `ask-user` finding, destructive
operation, credential-gated action, or production-side effect MUST pause for a
human decision through the TUI, AXI, or equivalent approval surface. Rationale:
the tool can automate cleanup only while the user retains final authority over
intent and external side effects.

### III. Evidence-First Quality Gates
Code changes MUST include focused tests or reviewer-visible evidence for the
changed behavior. Go changes MUST be formatted with `gofmt`, tested with
`go test -race ./...`, and linted with `make lint` unless the plan records a
specific, temporary reason. Changes to gate flow, daemon lifecycle, git hooks,
agent protocols, approval handling, or PR/CI automation MUST include targeted
unit coverage and SHOULD include tagged e2e coverage when the behavior crosses
process, git, or provider boundaries. Rationale: the product exists to prevent
unvalidated changes from reaching upstream, so its own changes need stronger
proof than prose.

### IV. Agent-Agnostic Structured Contracts
Agent integrations MUST preserve the same pipeline meaning regardless of
whether the backend is Claude, Codex, Rovo Dev, OpenCode, Pi, or ACP. Structured
outputs MUST be schema-validated before use, transcript-derived intent MUST be
treated as untrusted context, and transcript readers MUST redact secrets and
avoid storing raw transcript text. User-facing surfaces MUST render findings,
fixes, and approval choices with plain labels that match the user's workflow,
not internal field names. Rationale: the gate is a product contract, not a
property of any single agent implementation.

### V. Simplicity, Idempotence, and Recovery
Implementation changes MUST prefer the smallest reversible design that preserves
the gate contract. `init`, install, update, daemon startup, hook refresh, and
repair flows MUST be idempotent. State migrations, daemon recovery, and error
paths MUST preserve user data, surface actionable diagnostics, and avoid hidden
global configuration changes. New dependencies or abstractions require a clear
reduction in risk or complexity. Rationale: a local gate has high trust cost;
predictable repair and understandable code are part of the feature.

## Operational Constraints

The primary implementation language is Go 1.25, with the CLI under
`cmd/no-mistakes`, core behavior under `internal/`, generated agent skill
content under `skills/no-mistakes`, scripts under `scripts/`, and documentation
under `docs/src/content/docs`.

The product MUST remain a cross-platform CLI/daemon for macOS, Linux, and
Windows. Git operations MUST be non-interactive in daemon and agent contexts
where possible, using explicit environment controls to avoid hanging on editors
or credential prompts.

Configuration MUST keep the fixed pipeline order intact. Users may configure
agent choice, baseline test/lint/format commands, evidence storage, auto-fix
limits, ignored paths, and transcript intent extraction, but features MUST NOT
make permanent step removal or step reordering part of normal configuration.

Telemetry and transcript handling MUST stay transparent and controllable.
Telemetry MUST remain disableable with documented environment variables, and
transcript-derived intent MUST avoid retaining raw transcript text in the
database.

## Development Workflow

Substantial features MUST start from a spec and implementation plan that name
the affected gate semantics, worktree or state boundary, approval behavior,
agent contract, documentation impact, and validation evidence.

Before implementation, plans MUST identify the targeted tests or evidence that
prove the change. Bug fixes and cleanup/refactor work MUST lock the existing
behavior with regression tests before changing behavior when coverage is absent.

Implementation MUST preserve unrelated user changes. Generated files and
committed skills MUST be updated from their source of truth, and generated skill
drift MUST be checked through `make lint`.

Documentation MUST change with user-visible behavior, configuration, CLI flags,
approval semantics, telemetry behavior, or troubleshooting guidance. Runtime
guidance in README and docs MUST stay consistent with the fixed pipeline and
agent workflow.

Before completion, the smallest sufficient validation MUST run and be reported.
For ordinary Go changes this means `gofmt`, `go test -race ./...`, and
`make lint`; for docs-only changes this means docs build or a clear explanation
when the build is intentionally skipped.

## Governance

This constitution supersedes conflicting local practices for Spec Kit-driven
planning in this repository. Amendments require a documented rationale, a
semantic version decision, a Sync Impact Report in this file, and review of all
dependent Spec Kit templates plus relevant runtime docs.

Versioning follows semantic versioning:
MAJOR for incompatible governance or principle redefinitions, MINOR for new or
materially expanded principles or required sections, and PATCH for wording,
clarification, or non-semantic corrections.

Every feature plan MUST pass the Constitution Check before Phase 0 research and
MUST re-check after Phase 1 design. Any violation MUST be recorded in the plan's
Complexity Tracking table with the rejected simpler alternative and the evidence
that justifies the exception.

Compliance reviews MUST verify that tests, docs, user approval behavior, agent
contracts, worktree boundaries, and recovery paths align with this constitution
before claiming a feature is ready to ship.

**Version**: 1.0.0 | **Ratified**: 2026-06-15 | **Last Amended**: 2026-06-15
