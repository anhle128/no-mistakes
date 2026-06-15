# Quickstart: Review File Handoff

## Goal

Implement the review-file handoff without changing raw pipeline semantics:

- review details live in a Markdown file
- terminal review gate shows compact process/cancel controls
- automation responses remain compatible
- live/reattach/AXI surfaces expose additive `phase` and `review_file`
- latest processed handoff file is included in the PR audit commit

## Suggested Implementation Order

1. Add `internal/reviewhandoff` domain package.
2. Add `step_results.review_handoff_json` migration and typed DB helpers.
3. Generate and persist handoff state before review/fix-review approval events.
4. Add process-file validation and executor response dispatch.
5. Update IPC projection and AXI rendering.
6. Update TUI compact review gate and keys.
7. Add PR audit copy and publishable-artifact allowlist.
8. Update docs and generated skill guidance.

## Targeted Test Commands During Development

Start narrow:

```bash
go test ./internal/reviewhandoff
go test ./internal/db ./internal/ipc ./internal/daemon
go test ./internal/pipeline
go test ./internal/tui ./internal/cli
go test ./internal/pipeline/steps
```

Then run full validation:

```bash
gofmt -w <changed-go-files>
go test -race ./...
make lint
```

If docs change and docs tooling is available:

```bash
npm --prefix docs install
npm --prefix docs run build
```

## Manual Smoke Flow

1. Create a branch with changes that trigger review findings.
2. Push through `git push no-mistakes`.
3. Wait for the review gate.
4. Confirm the terminal shows:
   - review phase label
   - finding count/summary
   - review file path
   - only process/cancel review actions
5. Open the review file.
6. Edit one block:

````text
```no-mistakes-review-response review-1
action: fix
solution: Apply the parser validation described in option 1.
```
````

7. Leave another block as `accept` or `skip`.
8. Press `p process`.
9. Confirm only fixed IDs are sent to remediation and accept/skip decisions remain in the audit summary.
10. Let the run finish and confirm the PR branch includes the review file at the persisted relative path.

## Malformed File Smoke Flow

1. Reach a review gate.
2. Delete one response block or change `run_id`.
3. Press `p process`.
4. Confirm no fix/approval dispatch happens.
5. Confirm the compact gate remains open and shows one concise validation error plus the current review file path.
6. Confirm the edited file content is preserved.

## AXI Compatibility Smoke Flow

1. Reach a review gate.
2. Run `no-mistakes axi status`.
3. Confirm review step/gate output includes additive `phase` and `review_file`.
4. Run existing `no-mistakes axi respond --action approve` or `--action fix --findings <ids>`.
5. Confirm the response is accepted and review audit state records source `automation`.

## PR Audit Smoke Flow

1. Use a review file placed next to a changed `plan.md` or `tasks.md` anchor.
2. Leave the anchor changed but unrelated.
3. Complete the review gate.
4. Confirm the PR branch commit includes the review file and does not include the anchor solely because it was used for placement.
5. Repeat with the review file as the only publishable change and confirm a commit is still created.

## Completion Criteria

- No unresolved clarification markers remain in planning artifacts.
- Contracts cover file grammar, automation fields, action compatibility, and PR audit copy.
- Targeted tests cover parser, validation, path resolution, DB state, IPC/AXI/TUI behavior, process/cancel, auto-fix precedence, and PR audit inclusion.
- Full validation command results are recorded in the implementation final report.
