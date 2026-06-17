# Review Resolution Report Plan

## Goal

After the review step finds issues, no-mistakes must produce a committed
evidence report that explains every review issue and what happened to it. The
report must be detailed enough for a reviewer who does not know the project to
understand:

- what the problem was
- what solution was applied, if any
- why that solution was chosen
- whether the issue was verified as resolved, accepted without a fix, or still
  open

The report is a PR evidence artifact, not local daemon state.

## Current Problem

The current review flow captures rich review findings before a fix:

- `description`
- `context`
- `suggested_fix`
- `action`
- risk fields

After a fix, the fix agent currently returns only a short commit-oriented
`summary`. The DB and PR body can show that a fix happened, but they cannot
explain the actual applied solution or why that solution was chosen.

This creates a reviewability gap: users can see issues in the PR request, but
cannot see a detailed issue-to-solution explanation after no-mistakes resolves
them.

## Decisions

### 1. Scope

Version 1 covers only the **Review step**.

Do not include test, document, lint, rebase, push, PR, or CI findings in this
first version. Those steps have different schemas and evidence models.

### 2. Report Path

Write the report to:

```text
no-mistakes/<branch-slug>/review-resolution.md
```

Do not use `~/.no-mistakes`; that is local runtime state and cannot be committed
to the PR.

Do not use `.no-mistakes` in the target repo. The target repo may not already
have that directory, and this report should be visible PR evidence rather than
private-looking runtime state.

### 3. Branch Slug

Sanitize the branch slug using the same safety model as existing evidence branch
slug handling:

- split branch names on `/` to keep readable nested folders
- keep only `A-Z`, `a-z`, `0-9`, `-`, `_`, and `.`
- replace other characters with `-`
- collapse repeated `-`
- trim leading and trailing `-`
- drop empty, `.`, and `..` segments
- fall back to the run ID if no safe segment remains

Examples:

```text
feature/review-report -> no-mistakes/feature/review-report/review-resolution.md
../../main -> no-mistakes/main/review-resolution.md
feat:fix login -> no-mistakes/feat-fix-login/review-resolution.md
```

### 4. Report Structure

Use a stable Markdown format:

```markdown
# Review Resolution Report

## Run Context
- Branch:
- Base commit:
- Final commit:
- Review step:
- Generated at:

## Resolved Issues

### 1. <finding id>: <short issue title>
**Severity:**
**Location:**
**Action:**
**Fix Commit:**
**Applied Solution Source:**

#### Problem

#### Proposed Solution Before Fix

#### Applied Solution

#### Why This Solution

#### Verification

## Not Fixed / Accepted Issues

## Still Open Issues
```

Each review issue gets its own section. Do not collapse multiple issues into a
single broad summary.

### 5. Include All Review Issues

The report must include every review issue, not only resolved ones.

Use separate sections:

- `Resolved Issues`: findings fixed and verified by a follow-up review pass
- `Not Fixed / Accepted Issues`: findings the user approved as-is, skipped, or
  intentionally did not select for fix
- `Still Open Issues`: findings that remain unresolved after attempted fixes or
  when the pipeline aborts/fails before final acceptance

### 6. One Issue, One Commit

Each review issue must map to one fix commit.

If the user selects multiple review issues at once, keep the UX but internally
queue the selected findings and process them sequentially:

```text
round 1: review finds review-1, review-2, review-3
selection: [review-1, review-2, review-3]

round 2: fix only review-1 -> commit abc123 -> review again
round 3: fix only review-2 -> commit def456 -> review again
round 4: fix only review-3 -> commit ghi789 -> review again
```

This preserves traceability:

```text
review-1 -> fix commit abc123
review-2 -> fix commit def456
```

### 7. Retry Uses Amend

If a fix attempt for an issue does not resolve that issue, retry the same issue
by amending the existing issue fix commit instead of creating another commit.

Example:

```text
review-1 selected
fix attempt 1 -> commit abc123
follow-up review: review-1 still open
fix attempt 2 -> amend abc123
follow-up review: clean
report review-1 = resolved by abc123
```

If the fix limit is exhausted, keep the report honest:

```markdown
**Resolution Status:** Still open after fix attempts
**Fix Commit:** abc123
**Verification:** Follow-up review still reported this issue.
```

### 8. Auto-Fix Limit

For review resolution, interpret `auto_fix.review` as a per-issue attempt limit.

Example:

```text
auto_fix.review: 3

review-1: up to 3 attempts, amend the same commit
review-2: up to 3 attempts, amend the same commit
```

Do not allow the first issue to consume the entire step's fix budget.

### 9. Fix Agent Output

Extend the review fix agent structured response from:

```json
{
  "summary": "address review findings"
}
```

to:

```json
{
  "summary": "address review finding",
  "resolutions": [
    {
      "finding_id": "review-1",
      "applied_solution": "Added validation before ...",
      "why_this_solution": "This fixes the root invariant instead of only guarding the failing call site because ...",
      "changed_files": ["internal/foo.go", "internal/foo_test.go"]
    }
  ]
}
```

`summary` remains short and suitable for the commit subject. `resolutions[]`
feeds the detailed report.

### 10. Resolution Source Priority

Prefer fix-agent structured resolution details when they are present and map to
the finding ID.

Render:

```markdown
**Applied Solution Source:** fix agent structured output
```

If the agent omits `resolutions[]` or fails to include the selected
`finding_id`, no-mistakes must infer the applied solution from the fix commit
diff and surrounding source context.

Render the fallback clearly:

```markdown
**Applied Solution Source:** inferred by no-mistakes from fix commit diff because the fix agent did not return structured resolution details.
```

The fallback must not pretend to be the agent's own reported solution.

### 11. Fix Commit SHA

Persist the fix commit SHA on the corresponding fix round.

Add DB metadata equivalent to:

```text
step_rounds.fix_commit_sha
```

This is required because later document, lint, push, or other commits can appear
after the review fix. The report must not guess the fix commit from git log.

### 12. Report Commit Behavior

The report update for an issue must be included in the same commit as that
issue's fix.

Example:

```text
commit abc123: no-mistakes(review): fix review-1 missing nil guard
  - code fix
  - no-mistakes/<branch-slug>/review-resolution.md section for review-1
```

If verification details are only known after the follow-up review pass, amend
the same fix commit to update the report before moving to the next issue.

Do not create a separate report-only commit for each issue; that would violate
the one issue, one commit decision.

### 13. Incremental Report Updates

Update the report after each issue fix commit and follow-up review pass.

This means a partially completed or interrupted run still leaves evidence for
the issues already processed.

At Review step completion, finalize the report with:

- resolved issues
- accepted/unfixed issues
- still-open issues, if any
- final run context and final HEAD SHA at that point

### 14. Accepted / Unfixed Issues

Write `Not Fixed / Accepted Issues` only when the Review step reaches a user or
pipeline decision that accepts not fixing the issue.

Rules:

- user selects Fix for `review-1` but not `review-2`: `review-2` is accepted as
  `not selected for fix` when the step is ultimately approved or exits without
  fixing it
- user presses Approve while findings remain: remaining findings are
  `approved as-is`
- user skips the Review step: remaining findings are `review step skipped`
- pipeline abort/fail: do not call remaining findings accepted; place them in
  `Still Open Issues`

### 15. Force-Add Report File

If the target repo ignores `no-mistakes/`, still commit the report by force
adding only the exact report file:

```text
git add -f -- no-mistakes/<branch-slug>/review-resolution.md
```

Do not force-add the whole `no-mistakes/` folder.

### 16. Default Behavior

Enable report generation by default when review findings exist.

Behavior:

- clean review: no report file
- review findings approved as-is: report file records accepted issues
- review fixes occur: report file records issue-to-fix evidence

No config gate is required for v1.

### 17. PR Body Link

When a review resolution report exists, generated PR content should link to it.

Prefer a GitHub blob link when the remote and ref are known:

```markdown
[Review resolution report](https://github.com/org/repo/blob/<headSHA>/no-mistakes/feature/foo/review-resolution.md)
```

Otherwise use the repo-relative path:

```markdown
Review resolution report: `no-mistakes/feature/foo/review-resolution.md`
```

Do not show a report link when no report exists.

## Implementation Notes

The current implementation already commits after each fix through
`commitAgentFixes`. That behavior should become the foundation of the report
traceability model, but it needs to return or persist the resulting commit SHA.

The current executor sends all selected findings into one fix run. For this
feature, the Review step needs a per-issue queue so one selected finding is sent
to the fix agent at a time.

The report generator should build from persisted round data plus the git commit
diff for each fix commit. It should not rely only on live in-memory state.

## Minimum Test Coverage

### Review Queue

- Multi-selected review findings are processed one at a time.
- Each selected finding produces one fix commit.
- Retry for the same finding amends the existing fix commit instead of creating
  another commit.

### DB Metadata

- `fix_commit_sha` is persisted for review fix rounds.
- The schema migration is additive and does not break existing DBs.

### Report Generation

- Resolved issues include Problem, Proposed Solution Before Fix, Applied
  Solution, Why This Solution, Fix Commit, and Verification.
- Missing `resolutions[]` triggers the inferred-from-diff source label.
- Accepted/unfixed findings render under `Not Fixed / Accepted Issues`.
- Failed or aborted unresolved findings render under `Still Open Issues`.

### Git Staging

- The report path is force-added even when `no-mistakes/` is ignored.
- Only the report file is force-added, not the whole folder.

### PR Summary

- The PR body links to the report when it exists.
- The PR body omits the report link when no review findings/report exist.

## Non-Goals

- Do not build all-step pipeline evidence reporting in v1.
- Do not write report artifacts into `~/.no-mistakes`.
- Do not create a separate report-only commit for every issue.
- Do not infer accepted/unfixed status before the user or pipeline has actually
  accepted that outcome.
