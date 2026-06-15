# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: Go 1.25 or NEEDS CLARIFICATION
**Primary Dependencies**: Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, SQLite, YAML, TOON, Git/provider CLIs as applicable
**Storage**: SQLite under `NM_HOME`, local bare gate repos, disposable worktrees, logs/evidence directories, or N/A
**Testing**: `go test -race ./...`, targeted `_test.go`, tagged `internal/e2e` tests for cross-process flows, docs build when docs change
**Target Platform**: macOS, Linux, Windows CLI/daemon
**Project Type**: Go CLI/daemon with docs site and generated agent skill
**Performance Goals**: `git push no-mistakes` remains fast by handing long work to the daemon; TUI/AXI surfaces stay responsive during active runs
**Constraints**: `origin` remains unmodified; pipeline order is fixed; run work happens in isolated worktrees; approval gates pause for human judgment; transcript and telemetry controls remain explicit
**Scale/Scope**: Local per-user daemon managing multiple initialized repos and branch-scoped runs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Explicit Gate Semantics**: Does the feature preserve the meaning of a passed gate and keep `origin` behavior unchanged?
- **Isolation and User Control**: Are all intentional writes bounded to the run worktree or configured evidence locations, and do `ask-user`/destructive/credential-gated paths pause for human approval?
- **Evidence-First Quality**: Are focused tests or reviewer-visible evidence identified before implementation, including `go test -race ./...`, `make lint`, and e2e coverage when gate/daemon/agent/provider boundaries are crossed?
- **Agent-Agnostic Contracts**: Are structured outputs, transcript-derived intent, AXI/TUI labels, and supported agent behavior kept consistent across backends?
- **Simplicity and Recovery**: Is the smallest reversible design used, with idempotent init/install/update/daemon flows and actionable recovery errors?
- **Docs and Generated Artifacts**: Are README/docs/config references and generated skills updated or explicitly marked N/A?

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output (/speckit-plan command)
├── data-model.md        # Phase 1 output (/speckit-plan command)
├── quickstart.md        # Phase 1 output (/speckit-plan command)
├── contracts/           # Phase 1 output (/speckit-plan command)
└── tasks.md             # Phase 2 output (/speckit-tasks command - NOT created by /speckit-plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
cmd/no-mistakes/        # CLI entry point
internal/               # Core packages: agent, cli, daemon, gate, git, ipc, pipeline, tui, etc.
internal/e2e/           # Tagged end-to-end gate/agent journey tests
skills/no-mistakes/     # Generated agent skill output
scripts/                # Install and local helper scripts
docs/                   # Astro documentation site
.specify/               # Spec Kit templates, scripts, and memory
```

**Structure Decision**: [Document the specific packages, commands, docs, scripts,
and tests touched by this feature]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
