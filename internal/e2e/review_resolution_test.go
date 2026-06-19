//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestReviewResolutionMix(t *testing.T) {
	scenario := writeReviewResolutionScenario(t, `actions:
  - match: "Investigate previous review findings and address legitimate ones."
    text: "fixed selected review finding"
    edits:
      - path: "rr.txt"
        old: "before structured target\n"
        new: "after structured fix\n"
    structured:
      summary: "resolve review-1"
      resolutions:
        - finding_id: "review-1"
          applied_solution: "Changed rr.txt to remove the selected review issue."
          why_this_solution: "It preserves the branch while addressing the reported warning."
          changed_files:
            - "rr.txt"
  - match: "review scope: current worktree and HEAD changes relative to base commit"
    text: "review still needs human acceptance"
    structured:
      findings:
        - id: "review-2"
          severity: error
          file: "rr.txt"
          line: 1
          description: "approval finding remains"
          action: ask-user
      summary: "1 finding remains"
      risk_level: medium
      risk_rationale: "approval remains"
  - match: "Review the code changes and return structured findings"
    text: "review found two issues"
    structured:
      findings:
        - id: "review-1"
          severity: warning
          file: "rr.txt"
          line: 1
          description: "structured finding can be fixed"
          action: auto-fix
        - id: "review-2"
          severity: error
          file: "rr.txt"
          line: 1
          description: "approval finding remains"
          action: ask-user
      summary: "2 findings"
      risk_level: medium
      risk_rationale: "one fix and one approval"
  - text: "no issues found"
    structured:
      findings: []
      summary: "no issues found"
      risk_level: low
      risk_rationale: "no risks detected"
      tested:
        - "fakeagent: simulated test run"
      testing_summary: "simulated tests passed"
      title: "feat: review resolution"
      body: "## Summary\nfakeagent canned PR body"
`)
	h, initWorktree := newReviewResolutionHarness(t, scenario)
	_ = initWorktree

	h.CommitChange("review-resolution-mixed", "rr.txt", "before structured target\n", "add review resolution target")
	fw := h.AddWorktree("review-resolution-mixed")
	gateOut, err := h.RunInDir(fw, "axi", "run", "--intent", "review resolution mixed")
	if err != nil {
		t.Fatalf("axi run: %v\n%s", err, gateOut)
	}
	for _, want := range []string{"gate:", "review-1", "review-2"} {
		if !strings.Contains(gateOut, want) {
			t.Fatalf("initial gate missing %q in:\n%s", want, gateOut)
		}
	}

	fixOut, err := h.RunInDir(fw, "axi", "respond", "--action", "fix", "--findings", "review-1")
	if err != nil {
		t.Fatalf("axi respond fix: %v\n%s", err, fixOut)
	}
	for _, want := range []string{"gate:", "review-2", "approval finding remains"} {
		if !strings.Contains(fixOut, want) {
			t.Fatalf("fix response missing %q in:\n%s", want, fixOut)
		}
	}

	doneOut, err := h.RunInDir(fw, "axi", "respond", "--action", "approve")
	if err != nil {
		t.Fatalf("axi respond approve: %v\n%s", err, doneOut)
	}
	if !strings.Contains(doneOut, "outcome: passed") {
		t.Fatalf("approve output did not pass:\n%s", doneOut)
	}
	info := refreshedRunInfo(t, h, "review-resolution-mixed")
	rr := requireReviewResolution(t, info)
	if rr.Status != db.ReviewResolutionStatusFinal || rr.ResolvedCount != 1 || rr.AcceptedCount != 1 || rr.StillOpenCount != 0 {
		dumpReviewResolutionPrompts(t, h)
		t.Fatalf("review resolution info = %+v, want final 1 resolved 1 accepted", rr)
	}
	md := readReviewResolutionReport(t, rr)
	for _, want := range []string{
		"### review-1",
		"Applied Solution Source: fix agent structured output",
		"Changed files: rr.txt",
		"### review-2",
		"Outcome: Accepted Without Fix",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report missing %q:\n%s", want, md)
		}
	}
	statusOut, err := h.RunInDir(fw, "axi", "status")
	if err != nil {
		t.Fatalf("axi status: %v\n%s", err, statusOut)
	}
	for _, want := range []string{"review_resolution:", "status: final", "resolved: 1", "accepted_without_fix: 1"} {
		if !strings.Contains(statusOut, want) {
			t.Fatalf("status output missing %q:\n%s", want, statusOut)
		}
	}
}

func TestReviewResolutionClean(t *testing.T) {
	scenario := writeReviewResolutionScenario(t, `actions:
  - match: "Review the code changes and return structured findings"
    text: "review clean"
    structured:
      findings: []
      summary: "no review findings"
      risk_level: low
      risk_rationale: "review found no issues"
  - text: "no issues found"
    structured:
      findings: []
      summary: "no issues found"
      risk_level: low
      risk_rationale: "no risks detected"
      tested:
        - "fakeagent: simulated test run"
      testing_summary: "simulated tests passed"
      title: "feat: clean"
      body: "## Summary\nfakeagent canned PR body"
`)
	h, _ := newReviewResolutionHarness(t, scenario)
	h.CommitChange("review-resolution-clean", "clean.txt", "clean\n", "add clean target")
	fw := h.AddWorktree("review-resolution-clean")

	out, err := h.RunInDir(fw, "axi", "run", "--intent", "review resolution clean")
	if err != nil {
		t.Fatalf("axi run clean: %v\n%s", err, out)
	}
	if !strings.Contains(out, "outcome: passed") {
		t.Fatalf("clean run did not pass:\n%s", out)
	}
	info := refreshedRunInfo(t, h, "review-resolution-clean")
	if info.ReviewResolution != nil {
		t.Fatalf("clean review should not attach report metadata: %+v", info.ReviewResolution)
	}
	statusOut, err := h.RunInDir(fw, "axi", "status")
	if err != nil {
		t.Fatalf("axi status clean: %v\n%s", err, statusOut)
	}
	if strings.Contains(statusOut, "review_resolution:") {
		t.Fatalf("clean status should omit review_resolution:\n%s", statusOut)
	}
}

func TestReviewResolutionLegacy(t *testing.T) {
	scenario := writeReviewResolutionScenario(t, `actions:
  - match: "Investigate previous review findings and address legitimate ones."
    text: "legacy summary only"
    edits:
      - path: "legacy.txt"
        old: "before legacy target\n"
        new: "after legacy fix\n"
    structured:
      summary: "legacy summary fix"
  - match: "review scope: current worktree and HEAD changes relative to base commit"
    text: "review clean after legacy fix"
    structured:
      findings: []
      summary: "clean after legacy fix"
      risk_level: low
      risk_rationale: "issue absent"
  - match: "Review the code changes and return structured findings"
    text: "review found legacy issue"
    structured:
      findings:
        - id: "review-1"
          severity: warning
          file: "legacy.txt"
          line: 1
          description: "legacy finding can be fixed"
          action: auto-fix
      summary: "1 finding"
      risk_level: medium
      risk_rationale: "legacy fix needed"
  - text: "no issues found"
    structured:
      findings: []
      summary: "no issues found"
      risk_level: low
      risk_rationale: "no risks detected"
      tested:
        - "fakeagent: simulated test run"
      testing_summary: "simulated tests passed"
      title: "feat: legacy"
      body: "## Summary\nfakeagent canned PR body"
`)
	h, _ := newReviewResolutionHarness(t, scenario)
	h.CommitChange("review-resolution-legacy", "legacy.txt", "before legacy target\n", "add legacy target")
	fw := h.AddWorktree("review-resolution-legacy")

	gateOut, err := h.RunInDir(fw, "axi", "run", "--intent", "review resolution legacy")
	if err != nil {
		t.Fatalf("axi run legacy: %v\n%s", err, gateOut)
	}
	if !strings.Contains(gateOut, "review-1") {
		t.Fatalf("legacy gate missing review-1:\n%s", gateOut)
	}
	doneOut, err := h.RunInDir(fw, "axi", "respond", "--action", "fix", "--findings", "review-1")
	if err != nil {
		t.Fatalf("axi respond legacy fix: %v\n%s", err, doneOut)
	}
	if !strings.Contains(doneOut, "outcome: passed") {
		t.Fatalf("legacy fix did not complete run:\n%s", doneOut)
	}
	info := refreshedRunInfo(t, h, "review-resolution-legacy")
	rr := requireReviewResolution(t, info)
	if rr.Status != db.ReviewResolutionStatusFinal || rr.ResolvedCount != 1 || rr.EntryCount != 1 {
		t.Fatalf("review resolution info = %+v, want one final resolved entry", rr)
	}
	md := readReviewResolutionReport(t, rr)
	for _, want := range []string{
		"Applied Solution Source: inferred from fix round summary and commit changed-file diff because structured resolution details were unavailable",
		"Changed files: legacy.txt",
		"Evidence quality: round\\_level",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("legacy report missing %q:\n%s", want, md)
		}
	}
}

func TestReviewResolutionAbort(t *testing.T) {
	scenario := writeReviewResolutionScenario(t, `actions:
  - match: "Review the code changes and return structured findings"
    text: "review found abort issue"
    structured:
      findings:
        - id: "review-1"
          severity: warning
          file: "abort.txt"
          line: 1
          description: "issue remains when aborted"
          action: ask-user
      summary: "1 finding"
      risk_level: medium
      risk_rationale: "human decision needed"
  - text: "no issues found"
    structured:
      findings: []
      summary: "no issues found"
      risk_level: low
      risk_rationale: "no risks detected"
      tested:
        - "fakeagent: simulated test run"
      testing_summary: "simulated tests passed"
      title: "feat: abort"
      body: "## Summary\nfakeagent canned PR body"
`)
	h, _ := newReviewResolutionHarness(t, scenario)
	h.CommitChange("review-resolution-abort", "abort.txt", "abort target\n", "add abort target")
	fw := h.AddWorktree("review-resolution-abort")

	gateOut, err := h.RunInDir(fw, "axi", "run", "--intent", "review resolution abort")
	if err != nil {
		t.Fatalf("axi run abort: %v\n%s", err, gateOut)
	}
	if !strings.Contains(gateOut, "review-1") {
		t.Fatalf("abort gate missing review-1:\n%s", gateOut)
	}
	run := h.ActiveRun("review-resolution-abort")
	if run == nil {
		t.Fatal("expected active run")
	}
	if err := h.RespondError(run.ID, types.StepReview, types.ActionAbort); err != nil {
		t.Fatalf("abort response: %v", err)
	}
	info := h.WaitForRun("review-resolution-abort", 60*time.Second)
	info = h.RunInfo(info.ID)
	if info.Status != types.RunFailed {
		t.Fatalf("aborted run status = %s, want failed", info.Status)
	}
	rr := requireReviewResolution(t, info)
	if rr.Status != db.ReviewResolutionStatusIncomplete || rr.StillOpenCount != 1 || rr.EntryCount != 1 {
		t.Fatalf("review resolution info = %+v, want incomplete one still-open entry", rr)
	}
	md := readReviewResolutionReport(t, rr)
	for _, want := range []string{"## Still Open Issues", "### review-1", "Outcome: Still Open"} {
		if !strings.Contains(md, want) {
			t.Fatalf("abort report missing %q:\n%s", want, md)
		}
	}
}

func writeReviewResolutionScenario(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "review-resolution-scenario.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write scenario: %v", err)
	}
	return path
}

func newReviewResolutionHarness(t *testing.T, scenario string) (*Harness, string) {
	t.Helper()
	h := NewHarness(t, SetupOpts{Agent: "claude", Scenario: scenario})
	h.CommitChange("init-review-resolution", "seed-review-resolution.txt", "seed\n", "seed review resolution init")
	initWorktree := h.AddWorktree("init-review-resolution")
	out, err := h.RunInDir(initWorktree, "init")
	if err != nil {
		h.dumpDebugState()
		t.Fatalf("nm init: %v\n%s", err, out)
	}
	return h, initWorktree
}

func refreshedRunInfo(t *testing.T, h *Harness, branch string) *ipc.RunInfo {
	t.Helper()
	info := h.WaitForRun(branch, 60*time.Second)
	return h.RunInfo(info.ID)
}

func requireReviewResolution(t *testing.T, info *ipc.RunInfo) *ipc.ReviewResolutionReportInfo {
	t.Helper()
	if info == nil {
		t.Fatal("expected run info")
	}
	if info.ReviewResolution == nil || !info.ReviewResolution.Exists {
		t.Fatalf("expected review resolution metadata on run %+v", info)
	}
	return info.ReviewResolution
}

func readReviewResolutionReport(t *testing.T, rr *ipc.ReviewResolutionReportInfo) string {
	t.Helper()
	data, err := os.ReadFile(rr.Path)
	if err != nil {
		t.Fatalf("read review resolution report %s: %v", rr.Path, err)
	}
	return string(data)
}

func dumpReviewResolutionPrompts(t *testing.T, h *Harness) {
	t.Helper()
	for i, inv := range h.AgentInvocations() {
		prompt := inv.Prompt
		if len(prompt) > 1600 {
			prompt = prompt[:1600] + "...[truncated]"
		}
		t.Logf("agent invocation %d cwd=%s prompt:\n%s", i+1, inv.CWD, prompt)
	}
}
