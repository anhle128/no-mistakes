# Implementation Plan: Review File Handoff

**Branch**: `archon/thread-17570d85` | **Date**: 2026-06-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/001-review-file-handoff/spec.md`

## Summary

Move review-gate human decision-making from inline terminal finding controls to a durable Markdown handoff file while keeping the existing review step, raw step statuses, approval actions, and automation response contract intact. The implementation adds a focused internal review handoff package for deterministic path resolution, Markdown/YAML writing, response-block parsing, canonical review-result hashing, validation, metadata updates, automation mirroring, and final audit-file regeneration. Pipeline, TUI, IPC, CLI/AXI, push, PR summary, docs, and generated skill surfaces consume this contract additively.

The preferred implementation shape is conservative: no database schema migration, no new pipeline step names, no new raw status values, no automatic editor opening, and no parsing of prose outside fenced `no-mistakes-response` blocks. The existing `step_rounds` data is sufficient to recover prior finding decisions, selected IDs, user instructions, and fix summaries for the final PR audit file.

## Current Source Status

The current worktree already contains substantial in-flight implementation for this feature. Treat this plan as a source-grounded handoff for finishing and verification, not as a blank-slate design exercise.

- Core file contract code lives in `internal/reviewhandoff/` and already covers path resolution, hashing, parsing, validation, rendering, audit entries, and review phase labels.
- Executor integration lives in `internal/pipeline/review_handoff.go` and `internal/pipeline/executor.go`, including review file generation, `p process` handling, validation-error persistence, automation mirroring, and final no-findings audit rendering.
- Review-gate presentation lives in `internal/tui/review_file_gate.go`, `internal/tui/keys.go`, `internal/tui/events.go`, `internal/tui/pipeline.go`, `internal/tui/view.go`, and `internal/tui/review.go`.
- Structured surface changes live in `internal/ipc/protocol.go`, `internal/daemon/daemon.go`, `internal/cli/axi.go`, `internal/cli/axi_query.go`, and `internal/cli/axi_render.go`.
- PR-audit inclusion and recovery logic lives in `internal/pipeline/steps/push.go`; PR summary behavior remains in `internal/pipeline/steps/pr.go` and `internal/pipeline/steps/prsummary.go`.
- User and agent guidance has already been updated in `docs/src/content/docs/guides/tui.md`, `docs/src/content/docs/reference/cli.md`, `docs/src/content/docs/reference/pipeline-steps.md`, and `skills/no-mistakes/SKILL.md`.

## Next-Step Handoff

For the next implementation or review pass:

1. Treat `spec.md` as authoritative over `plans/grill-me/review-file-handoff.md`, especially for YAML front matter, pending/processed metadata, deterministic review-result hash plus review-cycle revision, automation mirroring, safe anchor resolution, and final PR-audit regeneration rules.
2. Review the current dirty worktree before adding new code. The main feature surface is already present; the likely remaining work is gap-fixing, test hardening, and full validation.
3. Start verification with focused package tests: `go test ./internal/reviewhandoff`, `go test ./internal/pipeline ./internal/pipeline/steps`, `go test ./internal/ipc ./internal/cli`, and `go test ./internal/tui`.
4. Then run repository-level validation: `go test -race ./...`, `make lint`, and `make docs-build`.
5. If cross-process behavior still feels under-proven after focused tests, add tagged e2e coverage for the review-file path from daemon review gate through TUI or AXI processing into push/PR audit staging.

The most likely review focus is no longer feature design. It is contract-faithfulness against `spec.md`, validation completeness, and ensuring the final committed audit file matches the processed review state.

## Likely Verification Gaps

The current source scan suggests these are the highest-value checks for the next pass:

- No single end-to-end journey test obviously proves `write handoff -> edit/process -> fix/fix-review -> push/PR audit inclusion`.
- Validator edge coverage may still need explicit assertions for duplicate response blocks, ignored prose outside fenced blocks, 16 KiB `solution:` enforcement, and empty or comment-only `solution:` fallback behavior.
- Phase-label tests appear strongest around awaiting states; verify direct coverage for `running -> Review preview` and `fixing -> Fixing review issues`.
- Reattach and recovery behavior deserves special attention: the daemon and TUI file-gate path currently depends on live `review_file_path` propagation, so confirm that reattach paths recover the review file gate instead of silently falling back to the legacy inline review UI.

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**: Existing Cobra CLI, Bubble Tea/Bubbles/Lip Gloss TUI, SQLite, `gopkg.in/yaml.v3`, TOON, Git/provider CLIs. No new third-party dependency is planned.
**Storage**: Existing SQLite run/step/round tables, the run worktree, deterministic Markdown review handoff file path, logs/evidence directories, and final PR branch commit.
**Testing**: Targeted Go unit tests for the new handoff package and touched packages, tagged e2e coverage where the flow crosses daemon/TUI/AXI/git boundaries, then `go test -race ./...` and `make lint`. Docs changes should run `make docs-build` or record why it is skipped.
**Target Platform**: macOS, Linux, Windows CLI/daemon.
**Project Type**: Go CLI/daemon with TUI, agent-facing AXI surface, generated agent skill, and Astro documentation.
**Performance Goals**: Review handoff file generation and validation remain local and bounded by 1 MiB. TUI and AXI must remain responsive while the daemon waits at the review gate.
**Constraints**: Preserve fixed pipeline order and raw review statuses (`running`, `awaiting_approval`, `fixing`, `fix_review`, `completed`). Preserve old approve/fix/skip automation. Do not trust file-supplied hashes, response prose, or `solution:` text as instructions. Reject absolute paths, traversal, `.git`, symlink escapes, stale hashes, processed files, files over 1 MiB, and single `solution:` values over 16 KiB.
**Scale/Scope**: One current review handoff file per run, reused across normal review and fix-review cycles, with prior-cycle decisions preserved in the final audit state.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Explicit Gate Semantics**: PASS. The feature preserves the single review step and maps file decisions back into existing approve/fix/skip executor outcomes. `origin` behavior and upstream push semantics remain unchanged.
- **Isolation and User Control**: PASS. Review files are written inside the isolated run worktree at a deterministic safe path, and the user must explicitly press `p process` or an automation client must send an existing approval response before the gate advances.
- **Evidence-First Quality**: PASS. The validation plan names package-level unit tests for resolver/writer/parser/hash, executor process/mirror behavior, TUI/AXI contracts, push inclusion, docs, plus repository validation.
- **Agent-Agnostic Contracts**: PASS. IPC/AXI additions are nullable and additive, raw statuses remain unchanged, and the fixer receives `solution:` values as delimited untrusted per-finding data only.
- **Simplicity and Recovery**: PASS. No new DB schema or review history files. File path and final audit state are recoverable from existing run, step, and round data.
- **Docs and Generated Artifacts**: PASS. Docs and `skills/no-mistakes/SKILL.md` are in scope because review gate controls and AXI guidance change.

## Project Structure

### Documentation (this feature)

```text
specs/001-review-file-handoff/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── ipc-axi-review-fields.md
│   ├── push-audit-file.md
│   ├── review-handoff-file.md
│   └── review-processing.md
└── tasks.md                  # Created later by /speckit-tasks
```

### Source Code (repository root)

```text
cmd/no-mistakes/              # CLI entry point
internal/reviewhandoff/       # New bounded package: path, hash, writer, parser, validation, metadata update
internal/pipeline/            # Executor gate integration and automation mirroring
internal/pipeline/steps/      # Review step prompt/fix integration and push/PR inclusion behavior
internal/ipc/                 # Additive review_file_path / review_phase_label fields and process-review RPC
internal/tui/                 # Compact review gate, p process / c cancel controls, phase labels
internal/cli/                 # AXI TOON fields, legacy respond behavior, docs-aligned help text
internal/db/                  # Reuse step_rounds data; no schema migration planned
internal/types/               # Existing finding/action/status types remain authoritative
internal/e2e/                 # Tagged daemon/AXI journey coverage if required
skills/no-mistakes/           # Generated agent skill output if command guidance changes
docs/src/content/docs/        # TUI, CLI, pipeline, auto-fix, troubleshooting docs
```

**Structure Decision**: Implement the file contract in `internal/reviewhandoff` instead of spreading Markdown parsing and hash logic across TUI, CLI, and pipeline packages. Pipeline owns when a handoff file is written, processed, mirrored, and included in the PR branch. TUI and AXI only display the structured fields and send explicit user/automation decisions to the daemon.

## Phase 0 Research Output

See [research.md](research.md). The design decisions are:

- Reuse `gopkg.in/yaml.v3` for front matter and response block parsing.
- Use a line-scanned Markdown parser that only recognizes fenced `no-mistakes-response` blocks.
- Compute a canonical SHA-256 hash from live gate state with explicit ordered fields.
- Add a review-only process RPC instead of adding a new `ApprovalAction`.
- Resolve the review file path deterministically without adding DB columns: first reuse the existing safe `review-issues-<run-short-id>.md` for the run when exactly one exists inside the checkout; otherwise compute it from FR-003 anchor rules using the same changed-file source as the current review round. Once a file is written, later review and fix-review cycles reuse that path instead of re-evaluating mutable anchors.
- Change push staging so the final audit file is explicitly included while anchor files are not staged solely by location.

## Phase 1 Design Output

See [data-model.md](data-model.md) and the contracts under [contracts](contracts/). The core model includes `ReviewHandoffFile`, `ReviewHandoffMetadata`, `ReviewFindingEntry`, `ResponseBlock`, `ProcessedReviewDecision`, `ReviewPhaseLabel`, `ValidationError`, and `ReviewAuditFile`.

## Implementation Notes

1. Generate or overwrite the review handoff after the review step has produced the current round and before the executor emits the review gate event.
2. Reuse the same file path across initial review and fix-review cycles. New findings overwrite the file, while final no-findings state writes a processed audit view preserving prior decisions from `step_rounds`.
3. Add `review_file_path` and `review_phase_label` to IPC step/event data as nullable fields. AXI should expose them while preserving raw `status` and existing `respond` behavior.
4. TUI review gates show a compact summary, relative file path, validation error when present, `p process`, and `c cancel`. Non-review gates keep current inline findings and approve/fix/skip controls.
5. Processing validates one byte snapshot, derives the existing approval outcome, then updates only `processed_at` and `processed_action`. If the bytes change before the metadata update, processing is rejected and the gate stays open.
6. Existing automation responses mirror the exact executed decision into the file before the executor advances. Mirror failure blocks the review gate with an actionable error.
7. Push step explicitly stages the final review audit file and normal pipeline changes. It must not stage anchor files merely because they were used to choose the review file directory.

## Validation Plan

Targeted tests before full-suite validation:

- `go test ./internal/reviewhandoff`
- `go test ./internal/pipeline ./internal/pipeline/steps`
- `go test ./internal/ipc ./internal/cli`
- `go test ./internal/tui`
- `go test -tags=e2e -count=1 ./internal/e2e/...` for the review-file journey if implementation crosses daemon/TUI/AXI/git boundaries in a way unit tests cannot prove.
- `go test -race ./...`
- `make lint`
- `make docs-build` when docs are updated.

Required behavioral coverage:

- Path resolver anchor precedence, fallback, branch slug, traversal, absolute paths, `.git`, symlink escape, and anchor suppression by resolved path.
- Writer/parser/hash front matter, response block grammar, missing/duplicate/unknown IDs, invalid actions, comment-only solutions, size bounds, stale hash, processed metadata rejection, and byte-snapshot race rejection.
- Executor file processing for fix subset and all accept/skip approval; validation failure keeps the gate open.
- Automation approve/fix/skip, selected IDs, per-finding instructions, and user-added findings still work and mirror into the audit file before gate advancement.
- TUI review gate compact controls/path/error and non-review gate behavior unchanged.
- AXI raw statuses unchanged with nullable `review_phase_label` and `review_file_path`.
- Fix-review no-findings final state preserves prior decisions and approves directly.
- Push includes the review audit file when it is the only remaining PR change and does not include an anchor file solely because it was an anchor.

## Complexity Tracking

No constitution violations are currently justified.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
