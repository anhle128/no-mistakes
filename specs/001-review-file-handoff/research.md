# Research: Review File Handoff

## Decision: Create a shared `internal/reviewhandoff` package

**Rationale**: The feature spans pipeline generation, TUI processing, AXI output, DB state, and PR publishing. A shared package can own Markdown generation, strict parsing, validation, digesting, path resolution, and typed state structs without making TUI or CLI code depend on pipeline internals. Existing shared finding behavior already lives in `internal/types/findings.go`, so a small domain package follows the repo's current separation.

**Alternatives considered**:

- Put parser/generator in `internal/tui`: rejected because AXI, daemon reattach, and PR publishing also need the same contracts.
- Put parser/generator in `internal/pipeline`: rejected because TUI process handling and CLI status would either duplicate logic or import pipeline code.
- Add a new external Markdown parser: rejected because the contract is a strict line-oriented fenced-block grammar and no dependency is needed.

## Decision: Persist review handoff state as additive typed JSON on `step_results`

**Rationale**: The current durable step state is `step_results` plus `step_rounds`. The feature needs one current handoff reference per review step, not independent queryable records. Adding `step_results.review_handoff_json` is an additive migration consistent with existing `findings_json`, avoids incompatible history changes, and gives reattach/AXI/PR copy one authoritative source after missed live events or daemon restarts.

The JSON value must be marshaled through a typed Go struct, not arbitrary maps. It should include the normalized repository-relative path, review cycle ID, finding-set digest, generated content digest, processed action/timestamp, decision source, and per-finding processed decisions.

**Alternatives considered**:

- Deterministically derive the path from run and branch only: rejected because anchor-based placement and fallback can differ after working-tree changes.
- Add many nullable columns: rejected because the state is cohesive, not separately queried, and would create unnecessary migration noise.
- Add a separate `review_handoffs` table: rejected for the first version because only current-state lookup is required and round history already exists for execution history.

## Decision: Derive `phase` from review step status

**Rationale**: The raw status is already authoritative. Deriving labels avoids persisting display text and keeps the compatibility contract stable:

- `running` review step -> `Review preview`
- `awaiting_approval` review step -> `Review preview complete`
- `fixing` review step -> `Fixing review issues`
- `fix_review` review step -> `Review fix result`
- `completed` review step -> omit sub-phase and render as `Review`
- non-review steps -> omit `phase`

**Alternatives considered**:

- Persist a phase string: rejected because it can drift from raw status.
- Rename raw statuses: rejected by the spec and by automation compatibility requirements.

## Decision: Use YAML front matter plus strict response fences

**Rationale**: The repo already depends on `gopkg.in/yaml.v3`. YAML front matter keeps metadata readable and parseable. Response blocks remain simple to edit:

````text
```no-mistakes-review-response review-1
action: fix
solution: Tighten validation in parser.go
```
````

The parser only trusts front matter and fenced response blocks. Human-readable `Issue`, `Context`, and `Recommendation` prose is for review only and must not affect processing.

**Alternatives considered**:

- JSON response blocks: rejected because manual edits are more fragile and clarifications chose line-oriented fields.
- HTML comments: rejected because they are harder to inspect and edit safely.
- Parse Markdown headings/prose: rejected because the spec requires prose outside response blocks to be ignored.

## Decision: Resolve empty fix solution fallback from trusted finding data

**Rationale**: If `action: fix` has an empty or comment-only solution, processing uses recommendation option 1 from the active/latest finding model or trusted generated metadata, never from editable human prose. This preserves the trust boundary while supporting default fix actions.

**Alternatives considered**:

- Read the rendered `Recommendation` prose from the file: rejected because users can edit prose and FR-005 forbids prose affecting processing.
- Reject all empty fix solutions: rejected because clarifications and FR-004 choose option 1 as the default.

## Decision: Existing automatic review auto-fix takes precedence

**Rationale**: The executor currently attempts configured auto-fix before entering approval. The handoff must be generated only when review reaches a human decision point under existing behavior. This preserves SC-006 and avoids adding a manual gate to existing automatic remediation.

**Alternatives considered**:

- Always generate a handoff before auto-fix: rejected because it changes existing auto-fix behavior.
- Disable handoff when auto-fix is configured at all: rejected because a human gate may still be reached after auto-fix attempts are exhausted.

## Decision: Process/cancel are TUI-only review gate controls, but automation responses stay compatible

**Rationale**: The terminal review gate should guide humans through the file handoff. Automation users already rely on `axi respond --action approve|fix|skip`, direct IPC `respond`, and yolo resolution. Those commands remain accepted and route to the same executor transition, with audit source recorded as `automation` or `auto_fix`.

**Alternatives considered**:

- Force AXI/direct IPC to process Markdown files: rejected because it would break existing automation and is unnecessary for structured automation payloads.
- Keep old TUI per-finding controls: rejected because FR-014 and FR-015 move terminal detail out to the file.

## Decision: Use a publishable-artifact allowlist for PR audit inclusion

**Rationale**: The current push step stages broad dirty state after agent fixes. The feature requires anchor files to affect placement only and never be committed merely because they are nearby or changed. The push/PR preparation path must copy/stage only intentional pipeline outputs and the persisted review-file relative path.

**Alternatives considered**:

- Continue broad `git add -A`: rejected because it can stage anchor files and unrelated working-tree changes.
- Re-run anchor discovery in the isolated work area: rejected because it can disagree with the original checkout and path chosen at generation time.

## Decision: Atomic ordering for review-cycle transitions

**Rationale**: A review/fix-review cycle must generate or overwrite the file, persist the current handoff state, then emit live events or expose reattach/AXI state. Processing must validate, atomically stamp processed metadata in the file, persist processed state, dispatch the executor response, then emit state. On failure, the previous committed state remains authoritative.

**Alternatives considered**:

- Emit events before file write: rejected because live or reattached clients could see a phase without a valid file.
- Persist state after response dispatch: rejected because a crash could let the gate advance while the file still says pending.

## Decision: No new runtime dependencies

**Rationale**: Existing Go, YAML, SQLite, TOON, and git helpers are sufficient. The feature is parsing and state coordination, not a dependency-selection problem.

**Alternatives considered**:

- Add a Markdown AST library: rejected because strict fenced blocks are easier to parse directly and the human-readable sections are not semantic input.
