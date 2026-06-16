# Contract: Push Audit File Inclusion

## Goal

The final review handoff file is committed to the PR branch as the review audit file.

## Push Step Requirements

Before committing agent changes, the push step must:

1. Detect whether the run has a processed review handoff file or persisted review decisions.
2. Ensure the final audit file exists at the deterministic review file path.
3. Regenerate the final processed-state file from persisted round data if the file is missing or unreadable and regeneration is possible.
4. Stage the review audit file explicitly.
5. Stage normal pipeline changes.
6. Avoid staging any anchor file merely because it was used to choose the review file location.

## Blocking Error

If processing succeeded but the audit file is absent and regeneration is impossible, the push step fails with an explicit audit-file error. The error should identify the expected path and the missing persistence data.

## Test Obligations

- Review audit file is committed when it is the only remaining change.
- Review audit file is committed at the expected relative path.
- Anchor file is not committed solely because it selected the audit file directory.
- Ignored audit path behavior is explicit: either force-add the audit file or block with a clear error.
