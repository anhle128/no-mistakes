# Clarifications — Current Worktree YOLO Mode

**Status:** ARCHIVED
**Applied:** 2026-06-18T22:58:55+07:00
**Generated:** 2026-06-18T22:53:25+07:00
**Spec:** spec.md
**Mode:** batch

**Instructions:**
- Edit each `Your Answer:` line below.
- Type an option letter (A/B/C/...), or `recommended` / `yes` / `suggested` to accept the suggestion, or your own short answer (<=5 words).
- Leave the line blank to skip a question.
- Save the file, then re-run `/clarifybatch` (or `/clarifybatch --apply`) to apply all answers in one pass.

---

## Q1. When current-worktree mode cannot resolve a trustworthy default-branch merge base, what should happen before pipeline execution starts?

**Category:** Edge Cases & Failure Handling
**Why it matters:** The review scope requirement depends on a correct base; a weak fallback could silently narrow the gate.
**Recommended:** Option B - A single non-interactive refresh reduces avoidable false failures while still rejecting when the full branch base cannot be proven.

| Option | Description |
|--------|-------------|
| A | Fail fast with a recovery message; never fetch automatically. |
| B | Attempt one non-interactive default-branch ref refresh, then reject if the base is still unavailable. |
| C | Use the best locally cached default-branch ref even if it may be stale. |
| D | Continue with a narrower diff and mark the review scope degraded. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** B
**Reason:** Current mode must review "the full branch diff against the default-branch merge base" (`specs/003-no-worktree-yolo/spec.md:54`), and the existing git helper supports a non-interactive remote-branch refresh (`internal/git/git.go:227`). If that still cannot prove the base, starting would weaken the gate.

---

## Q2. For the root `no-mistakes --no-worktree --yolo` command, what should happen if existing intent inference cannot produce usable intent?

**Category:** Functional Scope & Behavior
**Why it matters:** AXI explicitly requires `--intent`, but the root command is required to start without it where inference is available; the fallback affects CLI behavior and review quality.
**Recommended:** Option B - This preserves the root command's convenient path when inference works, but avoids sending an intent-free run into review under unattended `--yolo`.

| Option | Description |
|--------|-------------|
| A | Start anyway with an empty or generic inferred intent. |
| B | In non-interactive or `--yolo` mode, fail with a recovery message explaining how to provide or generate intent. |
| C | Open the existing wizard/interactive prompt even when `--yolo` was passed. |
| D | Fall back to branch name and commit subjects as the intent without failing. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** B
**Reason:** FR-008 allows the root command to start without explicit intent only "using existing intent inference behavior where available" (`specs/003-no-worktree-yolo/spec.md:114`), while inference can return "no intent attached" when no transcript matches (`internal/intent/intent.go:55`). Under `--yolo` or non-interactive execution, failing with recovery is the simple path that does not fabricate intent.

---

## Q3. How should no-mistakes handle an active run for the same repo and branch when the requested worktree mode or head commit is incompatible?

**Category:** Edge Cases & Failure Handling
**Why it matters:** Auto-fixes and status rendering become unsafe if an isolated run and a current-worktree run can race over the same branch.
**Recommended:** Option A - Blocking preserves user control and makes the recovery path explicit without silently aborting or mutating another run.

| Option | Description |
|--------|-------------|
| A | Reject the new request and show the exact active run plus resume or abort guidance. |
| B | Auto-abort the incompatible active run, then start the requested run. |
| C | Start a second run and let the daemon serialize work internally. |
| D | Convert the active run to the requested mode when branch matches. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** FR-022 requires active-run selection to prevent mixing current and isolated modes on the same repo and branch (`specs/003-no-worktree-yolo/spec.md:128`), and FR-023 only permits current-mode resume when repo, branch, and head are compatible (`specs/003-no-worktree-yolo/spec.md:129`). Rejecting preserves the existing active run instead of silently aborting or converting it.

---

## Q4. In current-worktree mode, should the warning that automated fixes may modify the current checkout ever become a blocking confirmation?

**Category:** Interaction & UX Flow
**Why it matters:** A blocking confirmation changes unattended `--yolo` semantics, while a warning-only surface may be too easy to miss.
**Recommended:** Option A - `--no-worktree` is already the explicit opt-in boundary and `--yolo` should remain equivalent to `--yes`; the warning should be visible but not a new approval gate.

| Option | Description |
|--------|-------------|
| A | Warning-only in CLI, AXI, status, and TUI; `--yes`/`--yolo` does not require extra confirmation. |
| B | Require an extra confirmation for current-worktree mode unless `--yes` or `--yolo` is present. |
| C | Require an extra confirmation even when `--yes` or `--yolo` is present. |
| D | Do not emit a warning; the flag name is sufficient. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** FR-019 requires output to warn or explain that current-worktree fixes may modify the checkout (`specs/003-no-worktree-yolo/spec.md:125`), while FR-006 says `--yolo` must not grant approval behavior beyond existing `--yes` (`specs/003-no-worktree-yolo/spec.md:112`). A blocking confirmation would make `--no-worktree --yolo` no longer equivalent to `--yes`.

---

## Q5. For active-run compatibility in current-worktree mode, should the resolved current work directory be part of the compatibility key?

**Category:** Domain & Data Model
**Why it matters:** A resumed current-mode run mutates a concrete directory, so compatibility keyed only by repo, branch, and head could resume a run tied to a different checkout path.
**Recommended:** Option A - Current-mode runs should match the same canonical repo, branch, head, mode, and resolved work directory; isolated runs can keep their existing repo/branch/head behavior.

| Option | Description |
|--------|-------------|
| A | Yes; current-mode resume requires the same resolved work directory in addition to repo, branch, head, and mode. |
| B | No; repo, branch, head, and mode are sufficient. |
| C | Only require the same directory when multiple linked worktrees are detected. |
| D | Use remote URL plus branch and head instead of local path identity. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** FR-015 requires persisting both worktree mode and resolved work directory (`specs/003-no-worktree-yolo/spec.md:121`), and the Current Work Directory is the canonical git worktree root (`specs/003-no-worktree-yolo/spec.md:138`). Since current-mode fixes mutate that concrete directory, resuming against a different path is incompatible even with the same repo, branch, and head.

---

## Q6. What should current-worktree mode do with untracked or ignored evidence/config files that appear before the run starts?

**Category:** Edge Cases & Failure Handling
**Why it matters:** The spec requires a clean committed worktree, but existing tooling may create local ignored files; the exact dirty check affects preflight rejection and test fixtures.
**Recommended:** Option B - Rejecting tracked modifications and untracked non-ignored files preserves safety while allowing normal ignored local state that Git already excludes.

| Option | Description |
|--------|-------------|
| A | Reject any tracked, untracked, or ignored file difference from a pristine checkout. |
| B | Reject tracked changes and untracked non-ignored files; allow ignored files. |
| C | Reject tracked changes only; allow all untracked files. |
| D | Allow dirty starts and rely on later review to catch conflicts. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** B
**Reason:** FR-011 preserves clean committed worktree preflight (`specs/003-no-worktree-yolo/spec.md:117`), and the existing dirty check is `git status --porcelain`, which reports tracked changes and untracked files without listing ignored files by default (`internal/git/git.go:278`). That rejects real uncommitted inputs without breaking ignored local state.
