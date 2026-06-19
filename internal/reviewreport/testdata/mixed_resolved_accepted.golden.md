# Review Resolution Report

Report Format Version: 1

## Run Context

- Run ID: run-123
- Repository identifier or path: /repo/project
- Branch: feature/review-report
- Base commit: base123
- Current/final head commit: head456
- Review step status: completed
- Report lifecycle state: final
- First generated timestamp: 2023-11-14T22:13:20Z
- Last refreshed timestamp: 2023-11-14T22:14:20Z
- Finalized timestamp: 2023-11-14T22:15:20Z
- Local report path: /tmp/nm/reports/run-123/review-resolution.md

## Counts

- Resolved: 1
- Accepted Without Fix: 1
- Informational / No Action Required: 1
- Still Open: 1
- Total Entries: 4

## Resolved Issues

### review-1

- Finding ID: review-1
- Severity: warning
- File and line: a.go:10
- Action: auto-fix
- Source: agent
- Review round ID: 1
- Description: fixed issue
- Context: unavailable in historical data
- Suggested/proposed fix: unavailable in historical data
- Risk level: unavailable in historical data
- Risk rationale: unavailable in historical data
- User instructions: not recorded
- Outcome: Resolved
- Outcome evidence and provenance: Comparable follow-up Review round no longer reported this normalized finding ID.
- Selection source: user
- Decision action: not recorded
- Decision actor/source: not recorded
- Decision timestamp: not recorded
- Decision round ID: not recorded
- Decision reason: not recorded
- Fix round ID: 2
- Applied Solution Source: fix agent structured output
- Applied solution or attempted solution: Changed a.go to handle nil input.
- Rationale: This preserves behavior and removes the warning.
- Changed files: a.go
- Fix commit SHA: abc123
- No-commit reason: not recorded
- Verification text: finding absent from follow-up Review output
- Follow-up round ID: 3
- Scope-equivalence note: same Review step run
- Verifier source: follow-up review
- Evidence reference: fix round 2 and follow-up Review round 3
- Evidence quality: structured


## Accepted Without Fix

### review-2

- Finding ID: review-2
- Severity: error
- File and line: b.go:5
- Action: ask-user
- Source: agent
- Review round ID: 1
- Description: accepted issue
- Context: unavailable in historical data
- Suggested/proposed fix: unavailable in historical data
- Risk level: unavailable in historical data
- Risk rationale: unavailable in historical data
- User instructions: not recorded
- Outcome: Accepted Without Fix
- Outcome evidence and provenance: Persisted Review terminal decision accepted the finding without a fix.
- Selection source: not recorded
- Decision action: approve
- Decision actor/source: user
- Decision timestamp: 2023-11-14T22:13:50Z
- Decision round ID: round-1
- Decision reason: approved tradeoff
- Fix round ID: not recorded
- Applied Solution Source: not applicable
- Applied solution or attempted solution: not recorded
- Rationale: not recorded
- Changed files: not recorded
- Fix commit SHA: not recorded
- No-commit reason: not recorded
- Verification text: accepted without fix by user
- Follow-up round ID: not recorded
- Scope-equivalence note: not recorded
- Verifier source: review terminal decision
- Evidence reference: persisted review resolution decision round-1
- Evidence quality: structured


## Informational / No Action Required

### review-3

- Finding ID: review-3
- Severity: info
- File and line: unavailable in historical data
- Action: no-op
- Source: agent
- Review round ID: 1
- Description: FYI only
- Context: unavailable in historical data
- Suggested/proposed fix: unavailable in historical data
- Risk level: unavailable in historical data
- Risk rationale: unavailable in historical data
- User instructions: not recorded
- Outcome: Informational / No Action Required
- Outcome evidence and provenance: Review marked this finding as no action required.
- Selection source: not recorded
- Decision action: not recorded
- Decision actor/source: not recorded
- Decision timestamp: not recorded
- Decision round ID: not recorded
- Decision reason: not recorded
- Fix round ID: not recorded
- Applied Solution Source: not applicable
- Applied solution or attempted solution: not recorded
- Rationale: not recorded
- Changed files: not recorded
- Fix commit SHA: not recorded
- No-commit reason: not recorded
- Verification text: no action required
- Follow-up round ID: not recorded
- Scope-equivalence note: not recorded
- Verifier source: review finding action
- Evidence reference: Review round 1 finding action
- Evidence quality: structured


## Still Open Issues

### review-4

- Finding ID: review-4
- Severity: warning
- File and line: c.go:8
- Action: auto-fix
- Source: agent
- Review round ID: 2
- Description: still open issue
- Context: unavailable in historical data
- Suggested/proposed fix: unavailable in historical data
- Risk level: unavailable in historical data
- Risk rationale: unavailable in historical data
- User instructions: not recorded
- Outcome: Still Open
- Outcome evidence and provenance: No persisted acceptance or comparable resolved evidence was recorded.
- Selection source: not recorded
- Decision action: not recorded
- Decision actor/source: not recorded
- Decision timestamp: not recorded
- Decision round ID: not recorded
- Decision reason: not recorded
- Fix round ID: not recorded
- Applied Solution Source: not applicable
- Applied solution or attempted solution: not recorded
- Rationale: not recorded
- Changed files: not recorded
- Fix commit SHA: not recorded
- No-commit reason: not recorded
- Verification text: verification inconclusive
- Follow-up round ID: not recorded
- Scope-equivalence note: no comparable parsed follow-up evidence
- Verifier source: report classifier
- Evidence reference: latest Review evidence round 2
- Evidence quality: unavailable
