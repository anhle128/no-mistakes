# Gotchas

## 2026-06-16 - Curate Ultragoal stories from feature directories

What happened: I fed the full feature spec, plan, contracts, and tasks into `omx ultragoal create-goals --from-stdin`. The generator expanded clarifications, acceptance scenarios, functional requirements, success criteria, and notes into more than 100 tiny goals instead of executable implementation stories.

Why it went wrong: I treated a feature directory as a raw brief blob. Ultragoal goal generation works better when supplied explicit story-level `--goal "title::objective"` entries derived from the feature phases.

Rule: For Spec Kit feature directories, read the artifacts first, then create a small curated Ultragoal plan with explicit goals aligned to implementation slices. Do not pipe the entire spec directory into `create-goals` unless the desired output is requirement-fragment tracking.

Relevant context: `.specify/feature.json`, `specs/001-review-file-handoff/tasks.md`, `.omx/ultragoal/goals.json`.

## 2026-06-16 - Use fixed-string grep for Markdown verification

What happened: I ran `grep -c -F '<Before>'` checks for Markdown list-item snippets whose first character was `-`; BSD grep interpreted the pattern as an option and failed before searching. I later used regex grep on Markdown bold text containing `**`, which failed with `repetition-operator operand invalid`.

Why it went wrong: I quoted the pattern correctly for backticks but forgot that a leading hyphen is still parsed as an option unless option parsing is terminated, and that Markdown markers are regex metacharacters in normal grep mode.

Rule: For literal grep checks on Markdown bullets, task lines, or bold markers, run `grep -F -- '<pattern>' <file>` or `rg --fixed-strings '<pattern>' <file>` so Markdown syntax is treated as data.

Relevant context: `specs/001-review-file-handoff/analyze-findings-2026-06-16.md`.

## 2026-06-16 - Quote shell search patterns containing backticks

What happened: I ran an `rg` verification command with the pattern inside double quotes while the pattern contained Markdown backticks around `solution:`. zsh attempted command substitution and emitted `command not found: solution:` before `rg` ran.

Why it went wrong: I treated a documentation search pattern like inert text, but shell backticks are active inside double quotes.

Rule: Use single quotes, `--fixed-strings`, or a here-doc-driven verifier for shell searches containing Markdown code spans/backticks.

Relevant context: `specs/001-review-file-handoff/spec.md`.

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
