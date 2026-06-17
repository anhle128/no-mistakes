# Clarifications — No-Worktree YOLO Guard

**Status:** ARCHIVED
**Applied:** 2026-06-17-112104
**Generated:** 2026-06-17T11:15:47+07:00
**Spec:** spec.md
**Mode:** batch

**Instructions:**
- Edit each `Your Answer:` line below.
- Type an option letter (A/B/C/...), or `recommended` / `yes` / `suggested` to accept the suggestion, or your own short answer (<=5 words).
- Leave the line blank to skip a question.
- Save the file, then re-run `/clarifybatch` (or `/clarifybatch --apply`) to apply all answers in one pass.

---

## Q1. What proof should be required before a run is classified safe for unattended source-changing YOLO actions?

**Category:** Functional Scope & Behavior
**Why it matters:** This defines the core fail-closed boundary and determines which runs can auto-fix, approve, or advance without per-gate input.
**Recommended:** Option A - Source-changing unattended actions should require a verified disposable source workspace, while allowed evidence/test paths remain narrow exceptions. This matches the isolation contract without turning evidence output into permission to mutate a primary checkout.

| Option | Description |
|--------|-------------|
| A | Require a verified disposable source workspace for unattended source-changing actions; allow evidence/test boundaries only for non-source artifacts. |
| B | Treat an explicitly configured evidence/test boundary as sufficient for all unattended YOLO actions, including source changes. |
| C | Treat a user-set "safe" flag as sufficient even if the workspace boundary cannot be independently verified. |
| D | Require an operating-system sandbox in addition to a disposable workspace before any unattended action. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:88-89` requires boundary checks before automatic actions and fail-closed withholding for unsafe or unknown runs; `spec.md:78` separately allows evidence directories while still excluding source writes outside the disposable run boundary.

---

## Q2. Where should the safe/unsafe/unknown execution-boundary classification be recorded as the authoritative state?

**Category:** Domain & Data Model
**Why it matters:** Reattach, rerun, daemon restart, and status refresh behavior all depend on whether boundary state survives process-local memory.
**Recommended:** Option A - Persisting the classification on the run keeps all UI, headless, and agent surfaces aligned after restarts. It also gives tests a stable state boundary to assert.

| Option | Description |
|--------|-------------|
| A | Persist boundary classification on the Run record and refresh it before each automatic gate action. |
| B | Recompute classification only from live workspace probes and do not persist the result. |
| C | Store classification only on the current Approval Gate and discard it after the gate resolves. |
| D | Derive classification only from static configuration and never from runtime run state. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** The spec defines a Run as the entity carrying "status, steps, findings, approval state, and user-facing outputs" at `spec.md:103`, and `spec.md:87` requires refreshing safety before later automatic actions after reattach, rerun, daemon restart, or status refresh.

---

## Q3. If a run was safe when YOLO consent was given but later becomes unknown before another automatic action, what should happen to the existing consent?

**Category:** Edge Cases & Failure Handling
**Why it matters:** This determines whether transient boundary loss pauses safely, resumes automatically, or invalidates the automation contract.
**Recommended:** Option B - Withhold automation while the boundary is unknown, then resume only if the same run is proven safe again. This preserves safe-run YOLO ergonomics while still failing closed during uncertainty.

| Option | Description |
|--------|-------------|
| A | Invalidate consent permanently for that run; require a fresh manual action or fresh YOLO consent after proof is restored. |
| B | Pause unattended actions while unknown; resume for the same run only after safe proof is restored. |
| C | Continue using the prior consent until the current gate completes, even if proof is temporarily unavailable. |
| D | Cancel the run when a previously safe boundary becomes unknown. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** B
**Reason:** `spec.md:89` says unsafe or unknown boundaries fail closed by withholding automatic actions, while `spec.md:44-46` requires preserving existing YOLO behavior once the run is safely isolated. That supports pausing during uncertainty, not permanently discarding same-run consent after proof is restored.

---

## Q4. Which user-facing surfaces must explain that YOLO was withheld because the boundary is unsafe or unknown?

**Category:** Interaction & UX Flow
**Why it matters:** The spec names several interfaces; inconsistent messaging would leave some users or agents unable to recover without reading logs.
**Recommended:** Option A - Every surface that can request or observe unattended gate handling should show the reason, current gate, boundary status, and recovery path. This keeps terminal, TUI, AXI, headless, and generated-agent workflows consistent.

| Option | Description |
|--------|-------------|
| A | All gate-driving or gate-observing surfaces: TUI, AXI, headless CLI output, terminal status, and generated agent guidance. |
| B | Only terminal and headless CLI output; TUI/AXI may rely on existing paused-gate UI. |
| C | Only logs and run history; interactive surfaces do not need inline withheld-YOLO messaging. |
| D | Only documentation and generated guidance; runtime surfaces do not need special text. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:32` requires every user-facing surface to report the reason, current gate, and recovery path, and `spec.md:114-116` explicitly calls out TUI/AXI, terminal, headless, agent guidance, and docs/generated artifacts.

---

## Q5. What audit record is required when unattended YOLO is allowed, withheld, or not requested for a gate?

**Category:** Non-Functional Quality Attributes
**Why it matters:** Operators and agents need accurate reporting after the fact, especially across daemon restart, reattach, and headless workflows.
**Recommended:** Option A - A persisted run event plus current gate status provides durable observability without requiring users to scrape logs. It also supports regression tests for allowed, withheld, and not-requested states.

| Option | Description |
|--------|-------------|
| A | Persist a run event and expose current gate status for allowed, withheld, and not-requested YOLO states. |
| B | Show the state only in the current interactive UI; no durable record is required. |
| C | Write log lines only; structured run history/status does not need the YOLO state. |
| D | Record only withheld YOLO; allowed and not-requested states can remain implicit. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:94` requires run history or status output to expose whether unattended YOLO was allowed, withheld, or not requested for the current gate. `spec.md:74` names daemon restart and reattach as edge cases, so logs or transient UI state are not enough.

---

## Q6. On unsafe or unknown-boundary runs, how should an explicit manual "fix" action behave?

**Category:** Functional Scope & Behavior
**Why it matters:** The spec preserves manual fix, approve, skip, and cancel, but manual fix may still mutate source if not constrained or clearly confirmed.
**Recommended:** Option A - Allow manual fix only as an explicit per-gate decision and record it as manual source-changing intent. This preserves user control while keeping unattended automation blocked.

| Option | Description |
|--------|-------------|
| A | Allow manual fix after explicit per-gate user decision and record it as a manual source-changing action. |
| B | Disallow manual fix on unsafe/unknown runs; only approve, skip, or cancel remain available. |
| C | Allow manual fix with no additional distinction from safe-run fixes. |
| D | Require the user to restart in a disposable workspace before any fix action is available. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:22` says unsafe or unknown-boundary runs still accept manual approve, fix, skip, or cancel, and `spec.md:90` requires those decisions to be distinguishable from unattended automatic decisions.

---

## Q7. Which external side effects count as PR-advancing or remote-advancing actions that unattended YOLO must withhold on unsafe or unknown boundaries?

**Category:** Integration & External Dependencies
**Why it matters:** Git remotes and provider integrations can change external state even if local source writes are blocked.
**Recommended:** Option A - Treat push, PR creation/update, merge, and provider status/comment actions as external advancement. This makes the guard conservative across current and future remote integrations.

| Option | Description |
|--------|-------------|
| A | Git push plus PR create/update/merge and provider status/comment actions that advance external review state. |
| B | Only git push; PR provider actions are outside this feature. |
| C | Only PR creation; updates, comments, and statuses can proceed unattended. |
| D | Defer the list to each provider integration with no shared product rule. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:89` withholds automatic "push, and PR-advancing decisions" on unsafe or unknown boundaries, while current local code mutates external state through upstream push in `internal/pipeline/steps/push.go:55-77` and PR create/update in `internal/pipeline/steps/pr.go:66-99`. Provider status or comment actions that advance review state fall under the same external-side-effect rule.

---

## Q8. What identity should define "the same gate" for duplicate automatic-response prevention across reattach, restart, and repeated status events?

**Category:** Domain & Data Model
**Why it matters:** Duplicate suppression can be too broad or too narrow unless the gate identity and versioning model are explicit.
**Recommended:** Option A - Run ID plus gate ID/step and gate version gives enough precision to suppress duplicate responses without blocking a genuinely new gate. It also supports restart-safe tests.

| Option | Description |
|--------|-------------|
| A | Run ID plus gate ID or step plus a gate version/fingerprint for the current pending decision. |
| B | Branch name only; any later gate on the same branch is treated as duplicate. |
| C | Finding text hash only; identical text across gates is treated as duplicate. |
| D | Process-local flag only; duplicate prevention does not need to survive restart. |
| Short | Provide a different short answer (<=5 words). |

**Your Answer:** A
**Reason:** `spec.md:99` requires repeated unsafe/unknown YOLO attempts not to enqueue duplicate responses or change the paused gate state. Existing YOLO duplicate prevention is currently process-local and step-keyed in `internal/tui/commands.go:88-98`, so restart-safe suppression needs run, gate, and gate-version identity.

---
