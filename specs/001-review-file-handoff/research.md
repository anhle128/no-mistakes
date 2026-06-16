# Research: Review File Handoff

## Decision: Use the existing YAML dependency for metadata and response blocks

Use `gopkg.in/yaml.v3`, already present in `go.mod`, to parse the front matter and the YAML payload inside each fenced `no-mistakes-response` block.

**Rationale**: The feature requires YAML front matter, the dependency already exists, and using one parser for both metadata and response blocks avoids ad hoc string parsing for structured fields.

**Alternatives considered**:

- Hand-rolled key/value parsing. Rejected because multiline `solution:` values, null metadata, and actionable validation errors are easier and safer with a real YAML parser.
- Add another Markdown/front-matter package. Rejected because no new dependency is needed and the grammar is intentionally small.

## Decision: Parse only fenced `no-mistakes-response` blocks with a small Markdown scanner

The parser should scan lines for fences whose info string is exactly `no-mistakes-response` after trimming, parse only the fenced body as YAML, and ignore all prose outside those blocks.

**Rationale**: This satisfies the spec's injection boundary. Users can edit explanatory Markdown freely, but only response blocks influence decisions.

**Alternatives considered**:

- Parse the whole Markdown document. Rejected because it would make prose semantically active.
- Use a full CommonMark parser. Rejected because the required grammar is one fenced block type and front matter, not arbitrary Markdown structure.

## Decision: Keep response-block actions separate from executor approval actions

Response blocks support `fix`, `accept`, and `skip`. Processing maps any fix response to existing `ActionFix`; all accept/skip responses map to existing `ActionApprove`. Legacy automation still sends `approve`, `fix`, or `skip`.

**Rationale**: This preserves current executor and automation semantics while allowing the audit file to distinguish `accept` and `skip` for readability.

**Alternatives considered**:

- Add `accept` as a new `ApprovalAction`. Rejected because the existing gate action contract is approve/fix/skip/abort.
- Treat file `skip` as executor `ActionSkip`. Rejected because the spec defines `accept` and `skip` as the same operational outcome in v1.

## Decision: Add a review-only process RPC instead of a new raw action

Add a daemon IPC method for processing the current review handoff file. The method validates the current file and current gate state, derives an existing approval decision, and feeds that decision into the executor. It does not add a new `types.ApprovalAction`.

**Rationale**: TUI needs a daemon-side operation because validation depends on authoritative live gate state and persisted round data. A dedicated method avoids overloading `RespondParams.Action` with a new raw action.

**Alternatives considered**:

- Add `ActionProcess`. Rejected because it expands the approval action domain when processing should be an implementation path to existing decisions.
- Validate in the TUI process. Rejected because the TUI does not own DB-backed gate state or the executor's current round.

## Decision: Compute the review-result hash from live canonical gate state

Hash a canonical JSON payload with explicit field names and order:

- run ID
- review step name and current status
- review cycle revision from the persisted step round identity and round number
- ordered finding IDs
- severity
- issue text
- context
- full recommendation option text
- default response action
- applied fix summary used by final no-findings state

Validation recomputes this hash from live gate state and treats the file-supplied hash only as a value to compare.

**Rationale**: This rejects stale editor buffers and prevents users from changing the hash to authorize outdated decisions.

**Alternatives considered**:

- Hash the generated file bytes. Rejected because user answers and prose edits should not change whether the file corresponds to the current review result.
- Trust the front matter hash. Rejected because the file is user-editable.

## Decision: Use byte-snapshot validation and atomic metadata update

Processing should read the file bytes once, validate, derive the decision, then re-read or otherwise compare the same file bytes immediately before writing updated metadata through a temp-file-and-rename path. If bytes changed, reject processing and keep the gate open.

**Rationale**: The spec requires validation, decision extraction, and metadata update to operate on one consistent snapshot and reject editor-save races.

**Alternatives considered**:

- Validate then overwrite metadata without checking for intervening writes. Rejected because it can erase newer user answers.
- Lock the file for editing. Rejected because cross-platform editor lock behavior is inconsistent.

## Decision: Recover the review file path deterministically instead of adding DB schema

Resolve `review-issues-<run-short-id>.md` from the current run, branch, and anchor rules. Expose the path when the file exists or can be recomputed. Reuse existing `step_rounds` fields for selected IDs, user findings, and fix summaries.

**Rationale**: The spec explicitly avoids new DB schema requirements. Existing round data is enough to regenerate the final processed audit file if the file is missing at push time.

**Alternatives considered**:

- Add a `review_file_path` DB column. Rejected because deterministic recovery is enough and a schema migration increases risk.
- Store a separate review history file per cycle. Rejected because v1 keeps one current handoff file per run.

## Decision: Create `internal/reviewhandoff` as the artifact boundary

Place path resolution, safe path checks, Markdown writing, response parsing, hash computation, validation, metadata update, and final audit rendering in one internal package.

**Rationale**: Pipeline, TUI, CLI, and IPC all need the same contract. Centralizing the artifact prevents divergent parsers or hash domains.

**Alternatives considered**:

- Put the code in `internal/pipeline`. Rejected because CLI/IPC/TUI tests need to reason about the same file contract without depending on executor internals.
- Put rendering in TUI. Rejected because AXI and automation mirroring also need the file.

## Decision: Keep AXI legacy finding details while adding review file fields

AXI should preserve existing raw statuses and `respond --action approve|fix|skip` behavior. It should add nullable `review_phase_label` and `review_file_path` fields. For compatibility, AXI may continue to expose structured findings needed by existing automation, while TUI review gates become compact and file-focused.

**Rationale**: The spec requires old automation responses to keep working. Removing AXI findings would break agents that select IDs from `axi status` or `axi run`.

**Alternatives considered**:

- Suppress all AXI finding rows for review gates. Rejected because old automation needs finding IDs.
- Require agents to edit the handoff file. Rejected because existing approve/fix/skip automation remains supported.

## Decision: Explicitly stage the review audit file in push behavior

Before the push step commits agent changes, ensure the final processed review handoff file is present or regenerated from persisted decisions. Stage that file explicitly. Do not stage any anchor file solely because it was used to choose the review file location.

**Rationale**: Current push uses broad `git add -A`; the feature needs the review audit file included even when it is the only remaining change, while preventing anchor file leakage.

**Alternatives considered**:

- Rely on existing broad staging. Rejected because it does not distinguish audit file inclusion from anchor-file side effects.
- Store the audit file outside the repo. Rejected because the final PR branch commit must contain it.
