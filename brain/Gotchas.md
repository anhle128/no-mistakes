# Gotchas

## 2026-06-14 - Trace user-visible data end to end

What happened: I treated the review `suggested_fix` fields as proof that the UI showed the solution, but the user was asking about the post-fix `Review - review fix:` screen. The live TUI showed the original finding details but did not surface the applied fix summary after the agent ran.

Why it went wrong: I checked the schema/prompt/rendering for findings, but initially missed the separate data path for fix summaries: DB round storage -> IPC event -> TUI model -> action bar render. The field existed in stored run info, but not in the live event path the user was looking at.

Rule: For user-visible pipeline data, verify the exact screen and complete producer-to-render path before saying the feature is present. Distinguish proposed fixes (`suggested_fix`) from applied fixes (`fix_summaries`) and test the boundary that feeds the live UI.

Relevant context: `internal/pipeline/steps/review.go`, `internal/ipc/protocol.go`, `internal/pipeline/executor.go`, `internal/tui/events.go`, `internal/tui/pipeline.go`.

## 2026-06-14 - Use user-facing labels for review surfaces

What happened: After adding `Applied fix:` to the fix-review action bar, the review findings still rendered the per-finding remediation as `Suggested fix:`. The data was present, but the user still could not quickly identify the solution in the review phase.

Why it went wrong: I preserved the internal JSON field name too closely in the UI. The implementation proved that `suggested_fix` existed, but did not optimize the rendered label for the user's mental model: issue, context, solution.

Rule: Keep internal schema names stable when needed, but render them with plain user-facing labels. For review findings, show remediation as `Solution:` and verify the exact terminal text the user will see.

Relevant context: `internal/tui/review.go`, `internal/tui/findings_test.go`, `docs/src/content/docs/guides/tui.md`.

## 2026-06-14 - Local install scripts must tolerate stopped daemons

What happened: `scripts/install-local-path.sh` delegated to `make install`, which builds and copies the binary but then fails if `no-mistakes daemon stop` cannot inspect a stale or missing daemon PID. That left installation looking failed even though the binary copy had completed.

Why it went wrong: I reused the Makefile target instead of making the helper script own the exact local replacement workflow. The Makefile target treats daemon-stop failure as fatal, but the script's job is to replace the PATH binary reliably for repeated local use.

Rule: Local reinstall helpers should be idempotent. Build and copy the binary directly, tolerate "daemon already stopped" or stale PID failures during stop, then start the daemon and verify `command -v` plus `--version`.

Relevant context: `scripts/install-local-path.sh`, `Makefile`.

## 2026-06-15 - Inspect existing tests before asking test-scope questions

What happened: During the review-file handoff design grill, I asked the user what minimum tests should cover even though the repo already has existing review/TUI/AXI tests that can answer the current baseline.

Why it went wrong: I stayed in interview mode for a question that was partly discoverable from source. The useful question was not "what tests should exist?" but "do we accept adding tests for the new file-based behavior that current tests do not cover?"

Rule: In source-backed grill-me sessions, inspect current tests before asking about test coverage. Only ask the user about product-level acceptance or risk appetite after separating existing coverage from new behavior gaps.

Relevant context: `internal/pipeline/steps/review_test.go`, `internal/tui/findings_test.go`, `internal/tui/action_bar_test.go`, `internal/cli/axi_drive_test.go`.

## 2026-06-17 - Fresh proof and daemon-owned audit beat cached boundary state

What happened: I initially let executor auto-fix skip boundary verification when `BoundaryVerifiedAt` was nil, and let AXI/TUI render withheld automation locally without making the daemon persist the withheld decision.

Why it went wrong: I treated cached boundary fields and client-side presentation as equivalent to authoritative gate decisions. That missed migrated or partially-initialized runs and made run history diverge from what the user saw.

Rule: For unattended source-changing actions, recompute boundary proof at the concrete write/fix boundary and fail closed on unknown or unsafe results. For unattended gate automation, send intent to the daemon so allowed/withheld decisions are recorded in run events before reporting status.

Relevant context: `internal/pipeline/executor.go`, `internal/pipeline/steps/ci_fix.go`, `internal/daemon/manager.go`, `internal/cli/axi_drive.go`, `internal/tui/commands.go`.

## 2026-06-17 - Do not use blocked steering to reconcile completed ultragoals

What happened: I used `omx ultragoal steer --kind mark_blocked_superseded` to clear a stale `review_blocked` story after the replacement blocker-resolution story had completed. That command wrote `steeringStatus: "blocked"` onto G001, so the goals were both complete but the artifact still reported a steering blocker.

Why it went wrong: I treated the steering command name as a generic "blocked story is superseded" repair. In this CLI implementation, that mutation marks the target as a steering blocker, which is wrong for a story that has already been checkpointed complete.

Rule: When reconciling an already completed ultragoal, prefer a normal completion checkpoint plus `annotate_ledger` audit evidence. Do not use `mark_blocked_superseded` unless the goal should remain schedule-skipped and completion-blocking semantics have been checked with `omx ultragoal status --json`.

Relevant context: `.omx/ultragoal/goals.json`, `.omx/ultragoal/ledger.jsonl`, `.codex/skills/ultragoal/SKILL.md`.
