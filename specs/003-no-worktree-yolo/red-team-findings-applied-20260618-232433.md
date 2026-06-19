# Red Team Findings: Current Worktree YOLO Mode

Session ID: RT-003-no-worktree-yolo-2026-06-18  
Target: specs/003-no-worktree-yolo/spec.md  
Date: 2026-06-18  
Maintainer: Kevin Le  
Selection method: auto (`--yes` accepted default lens set)  
Matched triggers: multi_party, contracts  
Supporting context: specs/003-no-worktree-yolo/no-worktree-yolo.md, specs/003-no-worktree-yolo/clarifications-applied-2026-06-18-225855.md, .specify/memory/constitution.md  
Wall-clock: not recorded by host runtime; all five lens agents completed successfully

## 1. Session Summary

Applied 23 spec-fix resolutions to `specs/003-no-worktree-yolo/spec.md` with historical SpecKit edits explicitly allowed. Two findings were skipped because their proposed changes conflicted with the clarified product scope or exceeded the origin-reference extraction need.

Severity summary:

| Severity | Count |
|---|---:|
| CRITICAL | 2 |
| HIGH | 13 |
| MEDIUM | 9 |
| LOW | 1 |

Lens summary:

| Lens | Count |
|---|---:|
| Agent Contract Integrity Adversary | 5 |
| Partial Evidence Recovery Adversary | 5 |
| Privacy and Transcript Exposure Adversary | 5 |
| Review Trust-Boundary Adversary | 5 |
| User Surface Misrepresentation Adversary | 5 |

## 2. Findings

| ID | Lens | Severity | Location | Finding | Suggested Resolution | Status |
|---|---|---|---|---|---|---|
| F-RT-003-no-worktree-yolo-2026-06-18-001 | Partial Evidence Recovery Adversary | CRITICAL | spec.md:109-110, FR-017 spec.md:135, FR-020 spec.md:138 | The spec says migrated runs with missing worktree metadata default to `isolated`, while also listing setup failure and crash recovery after current work directory resolution as edge cases. If a current-mode run record is partially written before `worktree_mode` or `work_dir` are persisted, stale recovery can classify it as isolated and either clean the wrong directory or omit the current-worktree warning from final output. | Require atomic persistence of `worktree_mode=current` and resolved `work_dir` before any run becomes recoverable or cleanup-eligible. For records missing these fields after partial setup, require an explicit `worktree_mode_unavailable` or `not_recorded` state that disables cleanup and labels final evidence incomplete. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-002 | Review Trust-Boundary Adversary | CRITICAL | FR-005, FR-006, FR-019, Assumptions | The spec treats `--no-worktree --yolo` as enough to let automation mutate the current checkout, create commits, push, open PRs, and drive CI, while only saying the warning is non-blocking. In a headless or agent-driven context, this can imply human approval for current-worktree mutation and downstream external effects that may only have been granted by an automated caller. | Separate auto-resolution from authority to mutate the current checkout and perform external side effects. Require start requests to record caller type and provenance and require explicit human-approved current-worktree consent for automation or headless starts, or clearly constrain which steps `--yolo` may drive without fresh human approval. | skipped |
| F-RT-003-no-worktree-yolo-2026-06-18-003 | Agent Contract Integrity Adversary | HIGH | FR-018/FR-019, SC-007; specs/003-no-worktree-yolo/spec.md:136-137,174 | The spec requires status, AXI, and TUI output to make current-worktree runs visible, but it does not define stable machine fields, labels, or schema validation for those surfaces. A future implementation could satisfy this with prose like an `equivalent compact label`, while downstream agents extracting mode or work directory silently miss or misparse it and still produce confident reports. | Define required machine-consumable fields for each structured surface, at minimum `worktree_mode`, `work_dir`, and `current_worktree_warning`, with exact allowed values and fail-closed behavior when absent or malformed. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-004 | Agent Contract Integrity Adversary | HIGH | FR-015/FR-017, Key Entities; specs/003-no-worktree-yolo/spec.md:133-135,148-153 | Run metadata persistence is underspecified for corruption and migration cases: missing metadata defaults to `isolated`, but malformed, empty, stale, or non-canonical `work_dir` is not defined. That creates a path where bad stored state can be normalized into a confident isolated/current report instead of being rejected or flagged degraded. | Specify validation rules for persisted run metadata: allowed enum values, canonical absolute path requirements, when `work_dir` is required or forbidden, and how readers must handle invalid values. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-005 | Agent Contract Integrity Adversary | HIGH | Constitution Alignment: Agent/Interface Contracts; .specify/memory/constitution.md:67-73, specs/003-no-worktree-yolo/spec.md:161 | The constitution says structured outputs must be schema-validated before use, but this feature spec only says warnings should be machine-consumable and does not require schema validation for review findings, fixes, approval choices, or AXI report objects. Missing or malformed structured review data could therefore be summarized as a successful normal pipeline run without a hard contract violation. | Add a functional requirement that current-worktree reports and AXI/status objects are schema-validated before rendering or agent consumption, and that missing required review/fix fields make the report explicitly incomplete rather than successful. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-006 | Partial Evidence Recovery Adversary | HIGH | User Story 4 spec.md:80-82, FR-014 spec.md:132, FR-020 spec.md:138 | Failure recovery is scoped to preserving commits and directories, but not to preventing a misleading final report. A failed, crashed, cancelled, or recovered run can still expose completed earlier steps and a PR URL without a required label that final gate evidence is incomplete or untrustworthy. | Add a requirement that terminal reports for failed, cancelled, stale-recovered, or setup-failed current-mode runs must include an explicit incomplete-evidence outcome, terminal reason, last trustworthy completed step, and whether generated commits may remain in the current worktree. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-007 | Partial Evidence Recovery Adversary | HIGH | FR-022 spec.md:140, FR-023 spec.md:141, Key Entities spec.md:148-153 | Active-run compatibility rejects incompatible requests, but the spec does not define how cancellation or supersession history is preserved when a current-worktree run is aborted, superseded, or replaced after head movement. Without a required terminal reason and replacement-run linkage, status/history can make the older run look like an ordinary cancellation or failure rather than a partial evidence artifact superseded by another run. | Require persisted terminal reason values such as `aborted_by_user`, `superseded_by_run`, `stale_recovered`, and `setup_failed`, plus an optional successor run ID/head SHA. Render these reasons distinctly in status, AXI, TUI, and regenerated reports. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-008 | Privacy and Transcript Exposure Adversary | HIGH | FR-008 and Assumptions, spec.md:126,184; Constitution IV, constitution.md:70-72,105 | The root command may infer intent from existing transcript-derived sources, but the spec does not require the inferred intent to be sanitized, bounded, or stored as a summary rather than raw transcript text. This conflicts with the constitution's transcript-handling boundary and creates a path for secrets or unavailable private transcript details to enter run metadata, AXI output, docs, or PR context. | Add an explicit requirement that inferred intent is a redacted, bounded summary only, never raw transcript/log text, and that failed inference guidance must not echo transcript snippets. Include tests for secret-redaction and raw-transcript non-persistence. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-009 | Privacy and Transcript Exposure Adversary | HIGH | FR-022/FR-023 and SC-010, spec.md:140-141,177 | Incompatible active-run requests must report the exact active run plus guidance, but the spec does not define a safe field whitelist for that report. An implementation could dump run records containing intent, finding summaries, paths, step logs, branch names, PR details, or error output into CLI/AXI/TUI surfaces. | Define the exact active-run fields allowed in conflict messages, such as run ID, mode, branch, short head, and normalized work-dir label. Prohibit raw logs, transcript-derived intent, diff hunks, code excerpts, and secret-bearing metadata in conflict output. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-010 | Privacy and Transcript Exposure Adversary | HIGH | User Story 4, FR-018/FR-019, SC-007, spec.md:70-81,136-137,174 | Status, AXI, and TUI rendering are required to expose the resolved work directory or equivalent label, but the privacy boundary for absolute paths is unspecified. Absolute paths can reveal usernames, customer names, repository locations, or temporary agent workspace identifiers, and AXI is explicitly machine-consumable so downstream tools may persist or retransmit it. | Require path minimization per surface: prefer repo-relative labels or basename plus stable run ID, and only expose full absolute paths behind an explicit verbose/debug field. Add rendering tests that prevent duplicated sensitive path detail across status, AXI, and TUI. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-011 | Review Trust-Boundary Adversary | HIGH | FR-022, FR-023, User Story 2 Acceptance Scenario 3 | Compatible active runs may be resumed or driven based on repo, branch, head, mode, and work directory, but not on initiating actor, approval mode, intent, skip settings, or review base. A lower-authority actor could attach to a higher-authority run and continue it under different gate-driving assumptions while the UI presents it as the same run. | Include authority-affecting fields in resume compatibility or require explicit handoff approval when they differ. At minimum persist and compare actor/session identity, approval mode, intent identity, skip configuration, and review base before allowing AXI or CLI to drive an active run. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-012 | Review Trust-Boundary Adversary | HIGH | FR-013, User Story 3 Acceptance Scenario 4, Constitution Alignment / Gate Semantics | The phrase `subject only to existing supported skip behavior` leaves open whether a skipped review, test, document, lint, push, PR, or CI step can still be rendered as a normal passed no-mistakes gate. A later surface could treat skipped or deferred checks as resolved fixes, weakening the gate meaning without a clear trust boundary. | Define the allowed skip taxonomy and require every skipped, deferred, or informational gate decision to be persisted, attributed, and rendered separately from passed/fixed. Success and PR/CI summaries should distinguish clean passes from accepted risk or configured omissions. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-013 | User Surface Misrepresentation Adversary | HIGH | FR-018/FR-019/FR-024, spec.md:136-142; User Story 4, spec.md:70-82 | The spec covers status, AXI, TUI, CLI warnings, and docs, but omits the PR body/PR summary even though the normal pipeline includes a PR step and existing PR summaries render review status, fixes, and unresolved findings. A current-worktree run could therefore produce a PR that says passed or auto-fixed without disclosing that fixes were made in the user's current checkout or linking the audit surface for the run. | Add PR summary/generated report surfaces to the visibility requirements. Require current-worktree mode, resolved work directory or safe compact equivalent, fix count/commit references, unresolved-finding state, and the run/report reference in the PR summary when a PR is created or updated. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-014 | User Surface Misrepresentation Adversary | HIGH | FR-014/FR-019 and User Story 4, spec.md:80-81,132,137 | The spec requires explaining retained current-worktree commits only when a later pipeline step fails. Successful outcomes can still hide that automated review/test/lint/CI fixes created commits in the current checkout, especially if final output says passed or checks-passed. | Extend the disclosure requirement to all terminal and checks-passed outcomes, not just failures. Require CLI, AXI, TUI/status, and PR summaries to state when pipeline fixes occurred and where the resulting commits live. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-015 | User Surface Misrepresentation Adversary | HIGH | SC-007 and FR-015/FR-018, spec.md:133-137,174 | SC-007 only verifies that run status, AXI, and TUI expose mode and directory; it does not require findings, fixes, and unresolved counts to agree across metadata, AXI summaries, TUI finding boxes, status output, and PR summaries. Existing surfaces can compute counts from different data shapes, so a user could see findings none in one place while another reports auto-fixed or still-open findings. | Define a single count contract with separate fields for reported, fixed, unresolved, skipped/approved-as-is, and unavailable findings. Add cross-surface tests that compare run metadata, AXI, status/TUI, and PR summary for the same multi-round run. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-016 | Agent Contract Integrity Adversary | MEDIUM | FR-014, User Story 4 scenario 2; specs/003-no-worktree-yolo/spec.md:80-81,132 | The spec says committed fixes remain in the current worktree after later failure, but it does not require stored fields that distinguish proposed fixes, attempted fixes, committed fixes, and user-applied follow-up changes. Future reports may rely on agent summaries that omit material fix details or overstate what was actually applied. | Require persisted fix outcome records with stable states such as `proposed`, `attempted`, `committed`, and `failed`, and include commit SHAs when applicable; reports should derive applied-fix claims from those records, not prose summaries. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-017 | Agent Contract Integrity Adversary | MEDIUM | FR-026, SC-008, Origin Reference; specs/003-no-worktree-yolo/spec.md:144,175; specs/003-no-worktree-yolo/no-worktree-yolo.md:70-97 | The origin reference is required as a companion artifact, but its extraction contract is prose-heading based and has no required keys, counts, or checksum-like completeness criteria for the source files and scout findings. Stable headings or summary counts can drift, letting future agents hallucinate that they found the origin purpose or first implementation files from an incomplete reference. | Define a small structured manifest in the spec directory for origin reference data, including source requirement path, purpose, conflicting prior branch, scout agent IDs, and an explicit first-files array with a count. | skipped |
| F-RT-003-no-worktree-yolo-2026-06-18-018 | Partial Evidence Recovery Adversary | MEDIUM | FR-015 spec.md:133, FR-018 spec.md:136, SC-007 spec.md:174 | The spec requires persisting mode and work directory, but not an immutable evidence snapshot for regeneration. A regenerated report after daemon restart can diverge from live review events if it reconstructs from mutable current worktree state, updated head SHA, missing logs, or rewritten step findings instead of the evidence captured when each step ran. | Require persisted report-reconstruction inputs: start head, final head, base SHA/source, base-refresh result, fix commit SHAs, step terminal timestamps, findings/log availability, and report generation source. If any historical input is missing, render it as `unavailable/not_recorded` rather than silently recomputing. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-019 | Partial Evidence Recovery Adversary | MEDIUM | FR-012 spec.md:130, Review Base spec.md:153, SC-009 spec.md:176 | Missing-base handling rejects before execution, but the spec does not require recording whether the one refresh happened, which ref was refreshed, or why the base remained unproven. Later status or regenerated output can only say the run failed, not whether final review evidence is absent because no trustworthy full-diff base existed. | Persist base-resolution evidence fields including default branch ref, merge-base SHA when proven, refresh attempted, refresh result/error, and rejection reason. Surface missing base as a distinct latest outcome such as `rejected_no_trustworthy_base`. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-020 | Privacy and Transcript Exposure Adversary | MEDIUM | US4 failure reporting and FR-013/FR-019/FR-024, spec.md:66,81,131,137,142 | The spec says failures should explain that commits remain and docs/agent guidance must describe the mode, while the normal pipeline still includes review, document, push, PR, and CI. It does not say whether PR bodies, AXI reports, TUI panes, generated docs, or agent guidance should reference report artifacts instead of duplicating raw review logs, code excerpts, or diff hunks. | Add a cross-surface reporting rule: user-visible and PR/AXI/TUI summaries may include bounded redacted summaries and artifact references, but must not inline raw logs, raw transcripts, secrets, code excerpts, or diff hunks unless an existing sanitized finding schema explicitly allows it. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-021 | Privacy and Transcript Exposure Adversary | MEDIUM | FR-026 and origin reference, spec.md:144; no-worktree-yolo.md:46-68 | The companion origin reference is required to record source requirement, purpose, sub-agent scouting, and source-code context, but the spec does not constrain what scouting content may be preserved. Future generators could paste raw sub-agent transcripts, code excerpts, command output, or secret-bearing local paths into the spec directory under the banner of preserving origin context. | Define the origin reference as a sanitized provenance summary with file paths, symbols, and decisions only. Prohibit raw sub-agent transcripts, raw logs, diff hunks, secrets, and long code excerpts, and require any detailed evidence to be referenced by location rather than copied. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-022 | Review Trust-Boundary Adversary | MEDIUM | FR-014, User Story 4 Acceptance Scenario 2, Key Entities / Run | The spec says current-worktree fix commits remain after later failure, but does not require commit provenance, mapping from findings to commits, or whether a commit represents an automated fix, user edit, accepted risk, or informational report. Reviewers could misattribute agent-authored commits or reports as user-authored resolution decisions. | Require run metadata and output to preserve fix provenance: actor, source finding, decision type, commit SHA, and whether the change was automated or user-authored. Render unresolved, accepted, skipped, and fixed states distinctly in CLI, AXI, TUI, and PR-facing summaries. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-023 | User Surface Misrepresentation Adversary | MEDIUM | FR-016/FR-018 and Constitution Alignment, spec.md:134-137,158-162 | The spec mandates internal values `isolated` and `current` and says outputs should label a `current-worktree run`, but it does not separate machine values from user-facing labels. Users can confuse `current` with current branch/current run/current status, and `isolated` may not clearly mean a disposable no-mistakes-owned checkout. | Require plain user-facing labels such as `uses this checkout` and `disposable no-mistakes checkout`, while keeping `worktree_mode: current|isolated` only in structured metadata. Apply the label contract to CLI, AXI, TUI, status, PR, and docs. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-024 | User Surface Misrepresentation Adversary | MEDIUM | FR-019, SC-007, and User Story 4, spec.md:76,80,137,174 | The warning requirement lacks precise timing and conditions: it says current-worktree output must warn in CLI, AXI, status, and TUI, but not whether this must appear before start, on resume, while fixing, on checks-passed, after failure, or after daemon recovery. A compliant implementation could show the warning once at start and later success/failure surfaces could omit it when it matters most. | Specify lifecycle points for the warning: pre-start/start acknowledgement, active run render, fix-in-progress/fix-review, terminal success/checks-passed, failure, and stale-run recovery. Require the warning to include the run/report reference once the run exists. | spec-fix |
| F-RT-003-no-worktree-yolo-2026-06-18-025 | Review Trust-Boundary Adversary | LOW | FR-026, Origin Reference / Sub-Agent Context, no-worktree-yolo.md | The companion origin reference mixes the original user requirement with sub-agent scouting and older-branch warnings, but the spec does not require authority labels for those sources. Future phases could treat agent-scouted implementation notes or conflict interpretation as user-approved product requirements. | Require the origin reference to label each item as user requirement, clarification decision, agent-derived evidence, or non-authoritative context. Future planning should cite only user requirements and applied clarifications as product authority unless separately approved. | spec-fix |

## 3. Resolutions Log

### F-RT-003-no-worktree-yolo-2026-06-18-001

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec explicitly lists "Setup fails after the current work directory is resolved but before the executor starts" and "The process crashes or daemon recovery runs while a current-worktree run is active or stale", so the finding's premise about partial setup and recovery holds. It also says "Existing and migrated runs MUST default to `isolated` mode when no explicit worktree metadata exists", which is too broad for a newly created current-mode run whose metadata failed to persist. Evidence: `specs/003-no-worktree-yolo/spec.md:109-110`, `specs/003-no-worktree-yolo/spec.md:135`, and `specs/003-no-worktree-yolo/spec.md:138`; locally, `internal/db/schema.go:12-23` has no current worktree metadata today and `internal/daemon/daemon.go:264-292` removes orphaned worktrees during recovery. This is not `skipped` because the cleanup risk is real, and not `new-OQ` because the spec already chooses current-mode non-cleanup as the invariant. A band-aid would special-case one recovery path; the durable fix is to make new current-mode run metadata atomic enough that every cleanup reader can fail closed.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-017**: Existing and migrated runs MUST default to `isolated` mode when no explicit worktree metadata exists.
  After: |
    - **FR-017**: Existing and migrated runs created before worktree metadata existed MUST default to `isolated` mode when no explicit worktree metadata exists. Newly created current-worktree runs MUST persist `worktree_mode=current` and the canonical resolved work directory in the same durable create/update boundary before the run becomes recoverable or cleanup-eligible. If recovery observes a new-format run whose worktree metadata is missing or invalid, it MUST mark metadata availability as `not_recorded`, disable directory cleanup for that run, and render final evidence incomplete instead of inferring `isolated`.

### F-RT-003-no-worktree-yolo-2026-06-18-002

- category: skipped
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref:
- notes:
  Reasoning: |
    Verification: The spec says "Current-worktree mode warnings are visible but non-blocking" and "`--yolo` MUST NOT grant approval behavior beyond existing `--yes` auto-resolution behavior"; the origin reference also says "Do not add a new approval or permission mode" and "Do not make `--yolo` more powerful than `--yes`." The finding correctly observes that the feature allows automation to proceed after explicit `--no-worktree --yolo`, but its proposed human-consent gate conflicts with the applied clarification and feature scope. Evidence: `specs/003-no-worktree-yolo/spec.md:15`, `specs/003-no-worktree-yolo/spec.md:123-125`, `specs/003-no-worktree-yolo/spec.md:137`, `specs/003-no-worktree-yolo/no-worktree-yolo.md:118-119`. This is not `new-OQ` because the clarification already answers the product decision, and not `spec-fix` because a fresh consent model would redesign approval semantics outside this feature. A band-aid would add a headless-only prompt exception; the durable outcome is to preserve the explicit opt-in contract and keep the warning non-blocking as specified.
  Reason: The requested authority split conflicts with the verified clarification that current-worktree warnings are non-blocking and with the origin non-goal "Do not add a new approval or permission mode" (`specs/003-no-worktree-yolo/no-worktree-yolo.md:118`).

### F-RT-003-no-worktree-yolo-2026-06-18-003

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says "Status, AXI rendering, and TUI rendering MUST make current-worktree runs visibly distinct from isolated runs", but it does not define stable structured fields for agent consumers. Evidence: `internal/ipc/protocol.go:180-193` currently exposes `RunInfo` without worktree fields, while `internal/cli/axi_render.go:67-75` and `internal/cli/axi_render.go:227-236` render only ID, branch, status, head, PR, and findings. The finding is valid, but the fix should stay narrow: define the new current-worktree fields rather than redesign every output surface. This is not `new-OQ` because the spec and local IPC shape are enough to define the contract, and a prose-only warning would be a band-aid because agents would still parse unstable text.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-018**: Status, AXI rendering, and TUI rendering MUST make current-worktree runs visibly distinct from isolated runs.
  After: |
    - **FR-018**: Status, AXI rendering, and TUI rendering MUST make current-worktree runs visibly distinct from isolated runs and MUST expose stable structured fields where the surface is structured: `worktree_mode` (`current` or `isolated`), a safe `work_dir_label`, and `current_worktree_warning` for current-mode runs. Missing or malformed required current-worktree fields MUST render the run as incomplete/degraded rather than as a normal passed or isolated run.

### F-RT-003-no-worktree-yolo-2026-06-18-004

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires "Run metadata MUST persist the worktree mode and resolved work directory for each run", but it does not say how to validate malformed mode values or non-canonical paths. Evidence: `specs/003-no-worktree-yolo/spec.md:133-135`; locally, `internal/db/schema.go:12-23` and `internal/db/run.go:10-28` show the current run record has no worktree-mode or work-dir validation surface. The finding's premise holds because migration/default behavior alone cannot safely normalize corrupted new metadata. This is not `accepted-risk` because metadata validation is a small durable contract, and not `new-OQ` because the valid enum and canonical current work directory are already defined. A band-aid would default invalid values to isolated; the durable fix is to require fail-closed validation rules.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-015**: Run metadata MUST persist the worktree mode and resolved work directory for each run.
  After: |
    - **FR-015**: Run metadata MUST persist the worktree mode and resolved work directory for each run, validate worktree mode against the allowed values, and validate current-mode work directories as canonical absolute git worktree roots. Readers MUST treat malformed, empty, stale, or non-canonical current-mode work directory metadata as degraded/incomplete state, not as `isolated` and not as safe for cleanup.

### F-RT-003-no-worktree-yolo-2026-06-18-005

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The constitution says "Structured outputs MUST be schema-validated before use", while this feature's current agent contract only says warnings are machine-consumable and human-readable. Evidence: `.specify/memory/constitution.md:67-73` and `specs/003-no-worktree-yolo/spec.md:161`; locally, `internal/types/findings.go:77-89` parses legacy/current findings JSON but the new worktree fields do not yet exist in a schema. The finding is valid for the new current-worktree report fields, but broadening it to redesign all review findings and approval choices would be unnecessary scope expansion. This is not `skipped` because the constitution creates a real contract, and not `new-OQ` because schema-validation is already required. A band-aid would add optional fields and hope renderers use them; the durable fix is fail-closed validation for the new current-worktree structured fields.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **Agent/Interface Contracts**: AXI and generated agent guidance must distinguish `--yolo` as an alias for `--yes`, keep `--intent` requirements clear, and expose current-worktree warnings in machine-consumable and human-readable output.
  After: |
    - **Agent/Interface Contracts**: AXI and generated agent guidance must distinguish `--yolo` as an alias for `--yes`, keep `--intent` requirements clear, and expose current-worktree warnings in machine-consumable and human-readable output. Current-worktree structured report fields MUST be schema-validated before rendering or agent consumption; missing or malformed required fields MUST make the affected report explicitly incomplete/degraded rather than successful.

### F-RT-003-no-worktree-yolo-2026-06-18-006

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says failed current-worktree runs must explain that commits remain, and stale recovery must not remove the current directory, but it does not require final evidence to be labeled incomplete after failed, cancelled, crashed, or setup-failed runs. Evidence: `specs/003-no-worktree-yolo/spec.md:80-82`, `specs/003-no-worktree-yolo/spec.md:132`, and `specs/003-no-worktree-yolo/spec.md:138`; locally, `internal/cli/axi_drive.go:488-509` can render an error and PR help without an explicit incomplete-evidence state. The finding is real because preserving commits is not the same as preserving trust in the final report. This is not `new-OQ` because the gate semantics already require honest evidence, and a band-aid would add one failure string in CLI only. The durable fix is a terminal-report contract that every current-mode surface can derive from persisted run state.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    2. **Given** a current-worktree run fails after automated fixes or generated commits, **When** the failure is reported, **Then** no-mistakes explains that those commits remain in the current worktree for inspection, amend, revert, or rerun.
  After: |
    2. **Given** a current-worktree run fails after automated fixes or generated commits, **When** the failure is reported, **Then** no-mistakes explains that those commits remain in the current worktree for inspection, amend, revert, or rerun, and terminal CLI, AXI, status, TUI, and generated report output marks the gate evidence as incomplete with the terminal reason and last trustworthy completed step.

### F-RT-003-no-worktree-yolo-2026-06-18-007

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires incompatible requests to be rejected, but does not define terminal reason history for current-worktree aborts, stale recovery, or any replacement/supersession that may occur through existing daemon paths. Evidence: `specs/003-no-worktree-yolo/spec.md:140-141`; locally, `internal/types/types.go:20-23` has ad hoc cancellation reason strings and `internal/daemon/manager.go:516-575` uses them, but there is no structured successor/recovery contract. The finding is valid, with the scope trimmed to persisted terminal reasons and optional successor linkage only where a replacement run actually exists. This is not `accepted-risk` because terminal reason persistence is low-cost and improves recovery correctness. A band-aid would infer meaning from free-text errors; the durable fix is structured terminal reason state used by every renderer.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-022**: Active-run selection MUST prevent mixing current and isolated modes on the same repo and branch; incompatible requests MUST be rejected with the exact active run and resume or abort guidance.
  After: |
    - **FR-022**: Active-run selection MUST prevent mixing current and isolated modes on the same repo and branch; incompatible requests MUST be rejected with the exact active run and resume or abort guidance. Current-worktree runs that are aborted, cancelled, stale-recovered, setup-failed, or superseded by an actual replacement run MUST persist a structured terminal reason and, when applicable, the successor run ID and head SHA so status, AXI, TUI, and regenerated reports do not present partial evidence as an ordinary failure.

### F-RT-003-no-worktree-yolo-2026-06-18-008

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec allows the root command to use "existing intent inference behavior where available", and the constitution says transcript-derived intent must be untrusted, redacted, and not stored as raw transcript text. Evidence: `specs/003-no-worktree-yolo/spec.md:126`, `specs/003-no-worktree-yolo/spec.md:184`, `.specify/memory/constitution.md:70-72`, and `.specify/memory/constitution.md:103-106`; locally, `internal/intent/redact.go:21-30`, `internal/intent/summarizer.go:12-23`, and `internal/pipeline/steps/intent.go:133-155` show the intended summary/redaction shape. The finding is valid because the feature spec should carry that contract into current-worktree root starts. This is not `new-OQ` because the constitution and existing intent implementation answer the policy. A band-aid would redact only display output; the durable fix is to require redacted bounded summaries at persistence and guidance boundaries.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-008**: The root `no-mistakes --no-worktree --yolo` command MUST be able to start a new current-worktree run without requiring an explicit `--intent`, using existing intent inference behavior where available; if usable intent cannot be inferred in non-interactive or `--yolo` mode, the command MUST fail before starting with recovery guidance for providing or generating intent.
  After: |
    - **FR-008**: The root `no-mistakes --no-worktree --yolo` command MUST be able to start a new current-worktree run without requiring an explicit `--intent`, using existing intent inference behavior where available; inferred intent MUST be persisted and rendered only as a redacted, bounded summary, never as raw transcript or log text. If usable intent cannot be inferred in non-interactive or `--yolo` mode, the command MUST fail before starting with recovery guidance for providing or generating intent, and that guidance MUST NOT echo transcript snippets.

### F-RT-003-no-worktree-yolo-2026-06-18-009

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The clarification says incompatible requests are rejected and report "the exact active run plus resume or abort guidance", but it does not define a safe field whitelist for that report. Evidence: `specs/003-no-worktree-yolo/spec.md:14`, `specs/003-no-worktree-yolo/spec.md:140`, and `specs/003-no-worktree-yolo/spec.md:177`; local AXI rows in `internal/cli/axi_render.go:36-42` are already small, but the new current-worktree conflict path is not specified. The finding is valid because "exact active run" can be misread as a full record dump. This is not `new-OQ` because safe operational fields are straightforward from the active-run contract. A band-aid would rely on individual renderers to be discreet; the durable fix is a shared conflict-message whitelist.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - Q3: If an active run for the same repo and branch is incompatible by mode, head, or work directory, the new request is rejected and reports the exact active run plus resume or abort guidance.
  After: |
    - Q3: If an active run for the same repo and branch is incompatible by mode, head, or work directory, the new request is rejected and reports the exact active run plus resume or abort guidance using only a safe conflict-message field set: run ID, worktree mode, branch, short head, safe work-directory label, status, and resume/abort commands. Conflict output MUST NOT dump raw logs, transcript-derived intent, diff hunks, code excerpts, secret-bearing metadata, or full run records.

### F-RT-003-no-worktree-yolo-2026-06-18-010

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires output to expose the resolved work directory or an equivalent compact label, but it does not draw a privacy boundary for absolute paths. Evidence: `specs/003-no-worktree-yolo/spec.md:70-81`, `specs/003-no-worktree-yolo/spec.md:136-137`, and `specs/003-no-worktree-yolo/spec.md:174`; the current AXI model is machine-consumable, so downstream persistence is a real concern. The finding is valid, but the fix should not hide the execution boundary entirely because current-worktree mode needs users to know which checkout is affected. This is not `accepted-risk` because path minimization is simple and durable. A band-aid would truncate paths inconsistently in one renderer; the durable fix is a safe default label plus explicit full-path/debug field.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **SC-007**: In 100% of run status, AXI, and TUI rendering tests for current-mode runs, the output exposes worktree mode and resolved work directory or an equivalent compact label.
  After: |
    - **SC-007**: In 100% of run status, AXI, and TUI rendering tests for current-mode runs, the output exposes worktree mode and a safe compact work-directory label by default. Full canonical absolute paths may appear only in explicit verbose/debug fields or logs intended for local diagnostics, and tests MUST prevent repeated sensitive path detail across normal status, AXI, TUI, and PR-facing summaries.

### F-RT-003-no-worktree-yolo-2026-06-18-011

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says current-worktree resume compatibility requires repo, branch, head commit, mode, and resolved current work directory, while the Start Request entity also includes base, skip settings, and intent. Evidence: `specs/003-no-worktree-yolo/spec.md:43-49`, `specs/003-no-worktree-yolo/spec.md:141`, and `specs/003-no-worktree-yolo/spec.md:151`; locally, `internal/cli/axi_drive.go:166-170` currently matches only run status and head for AXI attachment. The finding's actor-permission framing goes beyond this feature, but the omitted start-shape fields are a real contract gap. This is not `skipped` because differing intent/skip/base can change what the gate means, and not `new-OQ` because the relevant fields are already named. A band-aid would display a warning after attach; the durable fix is to persist immutable start-shape fields and reject or explicitly preserve them on resume.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-023**: A current-worktree request MAY resume or drive an active current-worktree run only when the repo, branch, head commit, selected worktree mode, and resolved current work directory are compatible.
  After: |
    - **FR-023**: A current-worktree request MAY resume or drive an active current-worktree run only when the repo, branch, head commit, selected worktree mode, resolved current work directory, review base, and immutable start-shape fields are compatible. Resume MUST NOT replace the active run's persisted intent, skip configuration, approval mode, or review base with values from the new request; differing requested values MUST be rejected or rendered as ignored with explicit guidance.

### F-RT-003-no-worktree-yolo-2026-06-18-012

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says the normal pipeline still runs "subject only to existing supported skip behavior", but it does not say how skipped gates are persisted or rendered relative to passed/fixed gates. Evidence: `specs/003-no-worktree-yolo/spec.md:66`, `specs/003-no-worktree-yolo/spec.md:131`, and `.specify/memory/constitution.md:38-44`; locally, `internal/pipeline/executor.go:50-59` and `internal/pipeline/executor.go:144-150` show skipped steps are a first-class status. The finding is valid because a skipped step is not a passed gate. This is not `new-OQ` because existing skip behavior already exists; the spec only needs to preserve its meaning. A band-aid would add a sentence to success output, while the durable fix is to require persisted, attributed skip decisions across reports.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-013**: Current-worktree mode MUST still run the normal pipeline: review, test, document, lint, push, PR, and CI, subject only to existing supported skip behavior.
  After: |
    - **FR-013**: Current-worktree mode MUST still run the normal pipeline: review, test, document, lint, push, PR, and CI, subject only to existing supported skip behavior. Every skipped, deferred, or informational gate decision MUST be persisted with its source and rendered distinctly from passed, fixed, or clean outcomes in status, AXI, TUI, generated reports, and PR-facing summaries.

### F-RT-003-no-worktree-yolo-2026-06-18-013

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec names status, AXI, TUI, CLI warnings, docs, and generated guidance, while the normal pipeline explicitly includes PR. Evidence: `specs/003-no-worktree-yolo/spec.md:131`, `specs/003-no-worktree-yolo/spec.md:136-142`, and local PR summary generation in `internal/pipeline/steps/prsummary.go:38-71`. The finding is valid because a PR-facing report is where downstream reviewers may see the gate result without the CLI/TUI context. This is not `new-OQ` because the PR step is already part of the fixed pipeline. A band-aid would mention current mode only in CLI; the durable fix is to include PR summaries in the visibility contract.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-024**: User-facing docs and generated no-mistakes agent guidance MUST describe both command forms, the meaning of `--no-worktree`, and the fact that `--yolo` is an alias for `--yes`.
  After: |
    - **FR-024**: User-facing docs, generated no-mistakes agent guidance, and PR-facing generated summaries MUST describe both command forms, the meaning of `--no-worktree`, and the fact that `--yolo` is an alias for `--yes`. When a PR is created or updated from a current-worktree run, the PR summary MUST include current-worktree mode, a safe work-directory label, fix count or commit references when fixes occurred, unresolved/degraded evidence state, and the run/report reference.

### F-RT-003-no-worktree-yolo-2026-06-18-014

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec currently calls out retained current-worktree commits only when "a later test, push, PR, or CI step fails"; it does not require the same disclosure for successful terminal outcomes. Evidence: `specs/003-no-worktree-yolo/spec.md:80-81`, `specs/003-no-worktree-yolo/spec.md:111`, `specs/003-no-worktree-yolo/spec.md:132`, and `specs/003-no-worktree-yolo/spec.md:137`; locally, successful AXI output can include fixes but not where current-mode commits live (`internal/cli/axi_drive.go:493-501`). The finding is valid because success does not erase the fact that automation modified the user's checkout. This is not `new-OQ` because the worktree mutation disclosure is already part of the feature. A band-aid would add failure-only wording; the durable fix is to tie disclosure to actual current-worktree fix commits across all terminal outcomes.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - Current-worktree auto-fixes create commits and a later test, push, PR, or CI step fails.
  After: |
    - Current-worktree auto-fixes create commits and any later terminal outcome is reported, including checks-passed, passed, failed, cancelled, or stale-recovered.

### F-RT-003-no-worktree-yolo-2026-06-18-015

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: SC-007 checks only that mode and directory appear; it does not require finding or fix counts to agree across status, AXI, TUI, and PR summaries. Evidence: `specs/003-no-worktree-yolo/spec.md:174`; locally, IPC exposes `ReportedFindings` and `FixedFindings` at step level (`internal/ipc/protocol.go:195-214`) and daemon conversion fills only those stats (`internal/daemon/daemon.go:493-515`). The finding is valid because inconsistent counts weaken the current-worktree audit trail without needing a broad reporting rewrite. This is not `accepted-risk` because the product already stores step/round data. A band-aid would recompute counts per renderer; the durable fix is a single persisted-count contract for the touched surfaces.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    **Independent Test**: Start current and isolated runs, inspect status output, AXI rendering, and TUI labels, and confirm each surface reports the mode and current run directory where relevant.
  After: |
    **Independent Test**: Start current and isolated runs, inspect status output, AXI rendering, TUI labels, and PR-facing summaries, and confirm each surface reports the mode, safe current run directory label where relevant, and matching finding/fix/evidence counts derived from the same persisted run data.

### F-RT-003-no-worktree-yolo-2026-06-18-016

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says current-worktree fix commits remain after later failure, but it does not require persisted fix outcome records. Evidence: `specs/003-no-worktree-yolo/spec.md:80-81` and `specs/003-no-worktree-yolo/spec.md:132`; locally, `internal/db/schema.go:40-52` stores step rounds, selected IDs, and `fix_summary`, but not a commit SHA or fix outcome state. The finding is valid because current-worktree mode makes committed fixes part of the user's checkout history. This is not `new-OQ` because the required shape is a data contract, not a product policy call. A band-aid would trust agent prose summaries; the durable fix is persisted fix outcome state with commit references when available.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-014**: If current-worktree fixes are committed and a later pipeline step fails, the system MUST leave those commits in the current worktree and MUST NOT auto-revert them.
  After: |
    - **FR-014**: If current-worktree fixes are committed and a later pipeline step fails, the system MUST leave those commits in the current worktree and MUST NOT auto-revert them. Current-worktree fix attempts MUST persist fix outcome records that distinguish proposed, attempted, committed, and failed fixes, include commit SHAs when commits are created, and drive applied-fix claims from those records rather than from prose summaries.

### F-RT-003-no-worktree-yolo-2026-06-18-017

- category: skipped
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref:
- notes:
  Reasoning: |
    Verification: The spec requires a companion origin reference, and the existing reference already records the source requirement, purpose, conflicting prior branch, sub-agent IDs, source-scout facts, and first files to inspect. Evidence: `specs/003-no-worktree-yolo/spec.md:144`, `specs/003-no-worktree-yolo/spec.md:175`, `specs/003-no-worktree-yolo/no-worktree-yolo.md:6-10`, `specs/003-no-worktree-yolo/no-worktree-yolo.md:23-38`, `specs/003-no-worktree-yolo/no-worktree-yolo.md:46-68`, and `specs/003-no-worktree-yolo/no-worktree-yolo.md:70-97`. The finding's premise that future agents need checksum-like completeness or a structured manifest is not established by the feature contract. This is not `spec-fix` because adding a manifest/count schema is machinery beyond the stated "under one minute" planning requirement, and not `new-OQ` because the existing artifact answers the purpose. A band-aid would add a one-off count; the durable answer is to keep the prose origin reference simple unless a later tool truly needs machine ingestion.
  Reason: The existing origin reference already satisfies the spec's stated extraction need with source, purpose, sub-agent context, source facts, and first files (`specs/003-no-worktree-yolo/no-worktree-yolo.md:6-10`, `specs/003-no-worktree-yolo/no-worktree-yolo.md:23-38`, `specs/003-no-worktree-yolo/no-worktree-yolo.md:46-97`); a structured manifest/checksum contract expands beyond the feature scope.

### F-RT-003-no-worktree-yolo-2026-06-18-018

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires persisting mode and work directory and rendering them, but it does not require immutable report-reconstruction inputs after daemon restart or current-worktree mutation. Evidence: `specs/003-no-worktree-yolo/spec.md:133`, `specs/003-no-worktree-yolo/spec.md:136`, and `specs/003-no-worktree-yolo/spec.md:174`; local run rows store head/base/status (`internal/db/schema.go:12-23`) and step rounds store findings/timing (`internal/db/schema.go:40-52`), but the spec does not bind regeneration to those persisted facts. The finding is valid because current-mode worktree state can move after evidence was produced. This is not `new-OQ` because evidence-first reporting is already a constitutional requirement. A band-aid would recompute from the live checkout; the durable fix is to persist report inputs and render missing ones as unavailable.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - The process crashes or daemon recovery runs while a current-worktree run is active or stale.
  After: |
    - The process crashes or daemon recovery runs while a current-worktree run is active or stale, and regenerated reports must rely on persisted run evidence rather than recomputing from mutable current worktree state.

### F-RT-003-no-worktree-yolo-2026-06-18-019

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires one non-interactive default-branch ref refresh and rejection if the base remains unavailable, but it does not require recording which ref was refreshed or why the base was unproven. Evidence: `specs/003-no-worktree-yolo/spec.md:12`, `specs/003-no-worktree-yolo/spec.md:105`, `specs/003-no-worktree-yolo/spec.md:130`, and `specs/003-no-worktree-yolo/spec.md:176`; the current schema has `base_sha` but no base-refresh evidence fields (`internal/db/schema.go:12-23`). The finding is valid because missing-base rejection affects whether any full-diff review evidence exists. This is not `new-OQ` because the behavior was clarified already. A band-aid would put the fetch error only in logs; the durable fix is persisted base-resolution evidence and a distinct missing-base outcome.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-012**: Current-worktree review scope MUST cover the full current branch diff against the default branch base; if a trustworthy base cannot be resolved, the system MUST attempt one non-interactive default-branch ref refresh and then reject the run if the base remains unavailable.
  After: |
    - **FR-012**: Current-worktree review scope MUST cover the full current branch diff against the default branch base; if a trustworthy base cannot be resolved, the system MUST attempt one non-interactive default-branch ref refresh and then reject the run if the base remains unavailable. Base-resolution evidence MUST be persisted, including default branch ref, proven merge-base SHA when available, whether refresh was attempted, refresh result or error, and rejection reason; missing-base rejection MUST render as a distinct `rejected_no_trustworthy_base` outcome.

### F-RT-003-no-worktree-yolo-2026-06-18-020

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires docs and generated guidance to describe current-worktree mode and requires failure reporting, but it does not define how much raw evidence those summaries may inline. Evidence: `specs/003-no-worktree-yolo/spec.md:66`, `specs/003-no-worktree-yolo/spec.md:81`, `specs/003-no-worktree-yolo/spec.md:131`, `specs/003-no-worktree-yolo/spec.md:137`, and `specs/003-no-worktree-yolo/spec.md:142`; locally, PR summaries can embed artifacts under size limits (`internal/pipeline/steps/prsummary.go:19-21`), and explicit AXI logs remain a separate diagnostic surface (`internal/cli/axi_query.go:103-167`). The finding is valid if scoped to generated/user-facing summaries, not to explicit log or approval-detail views. This is not `skipped` because privacy/minimization is a real reporting contract, and not `new-OQ` because the constitution already sets transcript boundaries. A band-aid would trim one PR template; the durable fix is a cross-surface summary rule with explicit exceptions for existing diagnostic surfaces.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **Docs/Generated Artifacts**: CLI docs, agent skill guidance, and any generated help or reference text must be updated because this feature adds user-visible flags and a new execution mode.
  After: |
    - **Docs/Generated Artifacts**: CLI docs, agent skill guidance, and any generated help or reference text must be updated because this feature adds user-visible flags and a new execution mode. Generated summaries for CLI, AXI, TUI/status, PR bodies, docs, and agent guidance SHOULD use bounded redacted summaries plus artifact/run references, and MUST NOT inline raw transcripts, raw logs, secrets, long code excerpts, or diff hunks except in existing explicit diagnostic or approval surfaces designed to show that detail.

### F-RT-003-no-worktree-yolo-2026-06-18-021

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires the origin reference to record source requirement, purpose, sub-agent scouting, and source-code context, but it does not constrain copied scout content. Evidence: `specs/003-no-worktree-yolo/spec.md:144`; the current origin reference includes sub-agent context and source-scout facts at `specs/003-no-worktree-yolo/no-worktree-yolo.md:46-68`. The finding is valid because preserving provenance does not require raw transcripts, logs, or long code excerpts. This is not `new-OQ` because local evidence is enough to define a safe summary shape. A band-aid would manually clean this one artifact; the durable fix is to require sanitized provenance summaries for origin references.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-026**: The spec directory MUST preserve a companion origin reference that records the source requirement, purpose, sub-agent scouting, and source-code context for later Speckit phases.
  After: |
    - **FR-026**: The spec directory MUST preserve a sanitized companion origin reference that records the source requirement, purpose, sub-agent scouting, and source-code context for later Speckit phases. The origin reference MUST preserve file paths, symbols, decisions, and concise evidence summaries only; it MUST NOT copy raw sub-agent transcripts, raw logs, diff hunks, secrets, or long code excerpts when a location reference is sufficient.

### F-RT-003-no-worktree-yolo-2026-06-18-022

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says fix commits remain after current-worktree failure, but the Run entity only names status, steps, branch/head/base metadata, mode, and directory. Evidence: `specs/003-no-worktree-yolo/spec.md:132` and `specs/003-no-worktree-yolo/spec.md:148`; locally, step rounds store selected finding IDs and fix summaries (`internal/db/schema.go:40-52`) but do not record commit provenance, actor, or decision type. The finding is valid because current-worktree commits can otherwise be misattributed in later reports. This is not `new-OQ` because provenance fields are mechanical and tied to existing fix rounds. A band-aid would add prose in PR summaries; the durable fix is persisted provenance that every surface can render consistently.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **Run**: A branch-scoped no-mistakes pipeline execution with status, steps, branch/head/base metadata, worktree mode, and resolved work directory.
  After: |
    - **Run**: A branch-scoped no-mistakes pipeline execution with status, steps, branch/head/base metadata, worktree mode, resolved work directory, evidence reconstruction inputs, and fix provenance for current-worktree commits. Fix provenance includes actor/source, source finding or decision, decision type, commit SHA when applicable, and whether the change was automated or user-authored.

### F-RT-003-no-worktree-yolo-2026-06-18-023

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec defines machine values `isolated` and `current`, and separately says outputs should label current-worktree runs. Evidence: `specs/003-no-worktree-yolo/spec.md:134-137`, `specs/003-no-worktree-yolo/spec.md:149`, and `.specify/memory/constitution.md:72-74`, which requires user-facing surfaces to render plain labels that match the workflow rather than internal field names. The finding is valid because "current" is overloaded in CLI status contexts. This is not `new-OQ` because the label intent follows directly from the constitution. A band-aid would rename one UI label; the durable fix is to separate structured enum values from human labels across all relevant surfaces.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **Worktree Mode**: The execution-boundary classification for a run. `isolated` means no-mistakes owns a disposable worktree; `current` means the run executes in the already-current git worktree root.
  After: |
    - **Worktree Mode**: The execution-boundary classification for a run. Structured metadata uses `worktree_mode: isolated|current`; user-facing labels MUST avoid relying on the bare enum words alone and SHOULD use plain labels such as "disposable no-mistakes checkout" for isolated mode and "uses this checkout" for current mode.

### F-RT-003-no-worktree-yolo-2026-06-18-024

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec says current-worktree output must warn in CLI, AXI, status, and TUI surfaces, but does not say when the warning must appear across the run lifecycle. Evidence: `specs/003-no-worktree-yolo/spec.md:15`, `specs/003-no-worktree-yolo/spec.md:76`, `specs/003-no-worktree-yolo/spec.md:80`, `specs/003-no-worktree-yolo/spec.md:137`, and `specs/003-no-worktree-yolo/spec.md:174`. The finding is valid because a start-only warning would satisfy a loose reading while hiding the risk during later fix, success, failure, or recovery states. This is not `new-OQ` because the warning is already required and non-blocking; only lifecycle coverage is missing. A band-aid would print it once at start; the durable fix is to define the warning lifecycle points and include the run/report reference once known.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **FR-019**: Current-worktree user-facing output MUST warn or explain that pipeline fixes may modify the current checkout in CLI, AXI, status, and TUI surfaces; this warning MUST NOT add a blocking confirmation beyond existing `--yes` or `--yolo` behavior.
  After: |
    - **FR-019**: Current-worktree user-facing output MUST warn or explain that pipeline fixes may modify the current checkout in CLI, AXI, status, TUI, and PR-facing surfaces; this warning MUST NOT add a blocking confirmation beyond existing `--yes` or `--yolo` behavior. The warning MUST appear at start/pre-start acknowledgement, active run rendering, fix-in-progress or fix-review, checks-passed/passed terminal output, failure/cancellation output, and stale-run recovery output; once a run exists, the warning MUST include the run/report reference.

### F-RT-003-no-worktree-yolo-2026-06-18-025

- category: spec-fix
- applied_at: 2026-06-18T23:24:33+07:00
- applied_by: codex
- downstream_ref: specs/003-no-worktree-yolo/spec.md
- notes:
  Reasoning: |
    Verification: The spec requires an Origin Reference, and the current companion artifact contains original request, source requirement summary, sub-agent context, source-scout findings, and older-branch warnings. Evidence: `specs/003-no-worktree-yolo/spec.md:154` and `specs/003-no-worktree-yolo/no-worktree-yolo.md:8-68`. The finding is valid because agent-derived scouting and older-branch warnings are useful context but not product authority. This is not `new-OQ` because authority labels are a documentation contract, not a new product decision. A band-aid would rely on section headings; the durable fix is to require explicit authority labels so future planning cites requirements and clarifications appropriately.
  Target: specs/003-no-worktree-yolo/spec.md
  Before: |
    - **Origin Reference**: The companion planning artifact that preserves why this feature exists and where future implementation work should begin.
  After: |
    - **Origin Reference**: The companion planning artifact that preserves why this feature exists and where future implementation work should begin, with each preserved item labeled as user requirement, applied clarification, agent-derived evidence, or non-authoritative context. Future planning MUST treat only user requirements and applied clarifications as product authority unless another spec explicitly approves more.

## 4. Validation Decision

Not a designated dogfood validation session.

## 5. Session Metadata

```yaml
session_id: RT-003-no-worktree-yolo-2026-06-18
target: specs/003-no-worktree-yolo/spec.md
invocation:
  requested_argument: specs/003-no-worktree-yolo
  resolved_target_spec: specs/003-no-worktree-yolo/spec.md
  yes: true
  explicit_lenses: null
  dry_run: false
date: 2026-06-18
maintainer: Kevin Le
feature_id: 003-no-worktree-yolo
matched_triggers:
  - multi_party
  - contracts
selection_method: auto
selected_lenses:
  - Agent Contract Integrity Adversary
  - Partial Evidence Recovery Adversary
  - Privacy and Transcript Exposure Adversary
  - Review Trust-Boundary Adversary
  - User Surface Misrepresentation Adversary
lens_failures: []
dropped_findings: []
counts:
  total: 25
  by_severity:
    CRITICAL: 2
    HIGH: 13
    MEDIUM: 9
    LOW: 1
  by_lens:
    Agent Contract Integrity Adversary: 5
    Partial Evidence Recovery Adversary: 5
    Privacy and Transcript Exposure Adversary: 5
    Review Trust-Boundary Adversary: 5
    User Surface Misrepresentation Adversary: 5
resolution_counts:
  spec-fix: 23
  new-OQ: 0
  accepted-risk: 0
  out-of-scope: 0
  skipped: 2
unresolved: 0
apply:
  applied_at: 2026-06-18T23:24:33+07:00
  applied_by: codex
  command: /speckit.red-team.apply --allow-historical-edits
  archived_as: specs/003-no-worktree-yolo/red-team-findings-applied-20260618-232433.md
notes:
  - "No hooks.before_speckit_red_team_run entry was present in .specify/extensions.yml."
  - "Constitution did not include a ## Red Team Trigger Criteria section; run used default trigger categories."
  - "High plus critical findings count is 15, below the overwhelming threshold of 25."
```
