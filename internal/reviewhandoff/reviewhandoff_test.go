package reviewhandoff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func testFindings() types.Findings {
	return types.Findings{
		Items: []types.Finding{
			{
				ID:           "review-1",
				Severity:     "warning",
				File:         "main.go",
				Line:         7,
				Description:  "missing validation",
				Context:      "Input reaches the parser unchecked.",
				SuggestedFix: "Option 1: Validate the input before parsing.\nOption 2: Return an explicit parser error.",
				Action:       types.ActionAutoFix,
			},
			{
				ID:           "review-2",
				Severity:     "info",
				Description:  "explain tradeoff",
				SuggestedFix: "Document why this branch is intentional.",
				Action:       types.ActionAskUser,
			},
		},
		Summary: "2 findings",
	}
}

func TestProcessUsesStoredDefaultRecommendation(t *testing.T) {
	root := t.TempDir()
	findings := types.Findings{Items: []types.Finding{{
		ID:           "review-1",
		Severity:     "warning",
		Description:  "missing validation",
		SuggestedFix: "Validate the input before parsing.",
		Action:       types.ActionAutoFix,
	}}}
	content, state, err := RenderPending(RenderInput{
		RunID:        "run123456789",
		Branch:       "feature/review-file",
		Status:       types.StepStatusAwaitingApproval,
		RelativePath: ".no-mistakes/issues/feature-review-file/review-issues-run12345.md",
		CycleID:      "cycle-1",
		Findings:     findings,
		Now:          100,
	})
	if err != nil {
		t.Fatalf("RenderPending: %v", err)
	}
	if err := WritePending(root, state.RelativePath, content); err != nil {
		t.Fatalf("WritePending: %v", err)
	}

	result, err := Process(ProcessInput{
		Root:     root,
		RunID:    "run123456789",
		Branch:   "feature/review-file",
		Status:   types.StepStatusAwaitingApproval,
		State:    state,
		Findings: findings,
		Now:      200,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if got := result.Instructions["review-1"]; got != "Validate the input before parsing." {
		t.Fatalf("default instruction = %q", got)
	}
	if got := result.State.Decisions[0].SolutionSource; got != SolutionSourceDefaultRecommendation {
		t.Fatalf("solution source = %q", got)
	}
}

func TestProcessRejectsEmptyFixWithoutOptionOne(t *testing.T) {
	root := t.TempDir()
	findings := types.Findings{Items: []types.Finding{{
		ID:           "review-1",
		Severity:     "warning",
		Description:  "missing validation",
		SuggestedFix: "Validate the input before parsing.",
		Action:       types.ActionAutoFix,
	}}}
	content, state, err := RenderPending(RenderInput{
		RunID:        "run123456789",
		Branch:       "feature/review-file",
		Status:       types.StepStatusAwaitingApproval,
		RelativePath: ".no-mistakes/issues/feature-review-file/review-issues-run12345.md",
		CycleID:      "cycle-1",
		Findings:     findings,
		Now:          100,
	})
	if err != nil {
		t.Fatalf("RenderPending: %v", err)
	}
	state.DefaultRecommendations = nil
	if err := WritePending(root, state.RelativePath, content); err != nil {
		t.Fatalf("WritePending: %v", err)
	}

	_, err = Process(ProcessInput{
		Root:     root,
		RunID:    "run123456789",
		Branch:   "feature/review-file",
		Status:   types.StepStatusAwaitingApproval,
		State:    state,
		Findings: findings,
		Now:      200,
	})
	if err == nil || !strings.Contains(err.Error(), "machine-detectable option 1") {
		t.Fatalf("Process error = %v, want machine-detectable option 1 rejection", err)
	}
}

func TestProcessNoFindingsRetainsPriorDecisions(t *testing.T) {
	root := t.TempDir()
	prior := []Decision{{
		FindingID:      "review-1",
		Action:         "fix",
		Solution:       "Use the strict validator.",
		SolutionSource: SolutionSourceUser,
		DecisionSource: DecisionSourceFile,
		ProcessedAt:    100,
	}}
	content, state, err := RenderPending(RenderInput{
		RunID:          "run123456789",
		Branch:         "feature/review-file",
		Status:         types.StepStatusFixReview,
		RelativePath:   ".no-mistakes/issues/feature-review-file/review-issues-run12345.md",
		CycleID:        "cycle-2",
		Findings:       types.Findings{Summary: "0 findings"},
		PriorDecisions: prior,
		Now:            150,
	})
	if err != nil {
		t.Fatalf("RenderPending: %v", err)
	}
	if err := WritePending(root, state.RelativePath, content); err != nil {
		t.Fatalf("WritePending: %v", err)
	}

	result, err := Process(ProcessInput{
		Root:     root,
		RunID:    "run123456789",
		Branch:   "feature/review-file",
		Status:   types.StepStatusFixReview,
		State:    state,
		Findings: types.Findings{Summary: "0 findings"},
		Now:      200,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Action != types.ActionApprove {
		t.Fatalf("action = %s, want approve", result.Action)
	}
	if len(result.State.Decisions) != 1 || result.State.Decisions[0].Solution != "Use the strict validator." {
		t.Fatalf("decisions = %+v", result.State.Decisions)
	}
	path := filepath.Join(root, filepath.FromSlash(state.RelativePath))
	stamped, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stamped: %v", err)
	}
	for _, want := range []string{"processed_action: approve", "`review-1`: fix", "Use the strict validator."} {
		if !strings.Contains(string(stamped), want) {
			t.Fatalf("stamped handoff missing %q:\n%s", want, stamped)
		}
	}
}

func TestRenderProcessValidHandoff(t *testing.T) {
	root := t.TempDir()
	findings := testFindings()
	content, state, err := RenderPending(RenderInput{
		RunID:        "run123456789",
		Branch:       "feature/review-file",
		Status:       types.StepStatusAwaitingApproval,
		RelativePath: ".no-mistakes/issues/feature-review-file/review-issues-run12345.md",
		CycleID:      "cycle-1",
		Findings:     findings,
		Now:          100,
	})
	if err != nil {
		t.Fatalf("RenderPending: %v", err)
	}
	if !strings.Contains(string(content), "```no-mistakes-review-response review-1") {
		t.Fatalf("rendered handoff missing response block:\n%s", content)
	}
	if err := WritePending(root, state.RelativePath, content); err != nil {
		t.Fatalf("WritePending: %v", err)
	}
	path := filepath.Join(root, filepath.FromSlash(state.RelativePath))
	edited := strings.Replace(string(content), "solution: \n", "solution: Use the strict validator.\n", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("edit handoff: %v", err)
	}

	result, err := Process(ProcessInput{
		Root:     root,
		RunID:    "run123456789",
		Branch:   "feature/review-file",
		Status:   types.StepStatusAwaitingApproval,
		State:    state,
		Findings: findings,
		Now:      200,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Action != types.ActionFix {
		t.Fatalf("action = %s, want fix", result.Action)
	}
	if got := result.Instructions["review-1"]; got != "Use the strict validator." {
		t.Fatalf("instruction = %q", got)
	}
	if result.State.ProcessedAction != ProcessedFix || result.State.ProcessedAt == nil || *result.State.ProcessedAt != 200 {
		t.Fatalf("processed state = %+v", result.State)
	}
	stamped, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stamped: %v", err)
	}
	for _, want := range []string{"processed_action: fix", "processed_at: 200", "Resolved Decision Summary", "`review-1`: fix"} {
		if !strings.Contains(string(stamped), want) {
			t.Fatalf("stamped handoff missing %q:\n%s", want, stamped)
		}
	}
}

func TestParseRejectsUppercaseAction(t *testing.T) {
	content := []byte(`---
no_mistakes_review_handoff: v1
run_id: run1
step: review
status: awaiting_approval
branch: feature
review_cycle_id: c1
finding_digest: digest
review_file: review.md
processed_action: pending
processed_at: 
---

` + "```no-mistakes-review-response review-1\n" + `action: FIX
solution: do it
` + "```\n")
	if _, _, err := Parse(content); err == nil {
		t.Fatal("expected uppercase action to be rejected")
	}
}

func TestSafeJoinAndBranchSlug(t *testing.T) {
	root := t.TempDir()
	if _, err := SafeJoin(root, "../escape.md"); err == nil {
		t.Fatal("expected escaping path to fail")
	}
	if got := BranchSlug("Feature/Review File!"); got != "feature-review-file" {
		t.Fatalf("BranchSlug = %q", got)
	}
}

func TestSafeJoinForWriteDoesNotCreateThroughSymlinkParent(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := SafeJoinForWrite(root, "link/nested/review.md")
	if err == nil || !strings.Contains(err.Error(), "path escapes root") {
		t.Fatalf("SafeJoinForWrite error = %v, want escape rejection", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "nested")); !os.IsNotExist(err) {
		t.Fatalf("SafeJoinForWrite created outside directory, stat err = %v", err)
	}
}

func TestResolvePathParsesStagedAndModifiedAnchor(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := os.MkdirAll(filepath.Join(root, "specs", "001-review-file-handoff"), 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	planPath := filepath.Join(root, "specs", "001-review-file-handoff", "plan.md")
	if err := os.WriteFile(planPath, []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	if err := os.WriteFile(planPath, []byte("staged\n"), 0o644); err != nil {
		t.Fatalf("write staged plan: %v", err)
	}
	runGit(t, root, "add", planPath)
	if err := os.WriteFile(planPath, []byte("staged\nunstaged\n"), 0o644); err != nil {
		t.Fatalf("write unstaged plan: %v", err)
	}

	got, err := ResolvePath(context.Background(), PathInput{WorkDir: root, Branch: "feature/review-file", RunID: "run123456789"})
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want := "specs/001-review-file-handoff/review-issues-run12345.md"
	if got != want {
		t.Fatalf("ResolvePath = %q, want %q", got, want)
	}
}

func TestResolvePathParsesQuotedAnchorWithSpaces(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	anchorDir := filepath.Join(root, "specs", "foo bar")
	if err := os.MkdirAll(anchorDir, 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	planPath := filepath.Join(anchorDir, "plan.md")
	if err := os.WriteFile(planPath, []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	if err := os.WriteFile(planPath, []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("modify plan: %v", err)
	}

	got, err := ResolvePath(context.Background(), PathInput{WorkDir: root, Branch: "feature/review-file", RunID: "run123456789"})
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	want := "specs/foo bar/review-issues-run12345.md"
	if got != want {
		t.Fatalf("ResolvePath = %q, want %q", got, want)
	}
}

func TestSingleAnchorRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "specs", "target.md"), []byte("target\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink("target.md", filepath.Join(root, "specs", "plan.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if got := singleAnchor(root, []string{"specs/plan.md"}); got != "" {
		t.Fatalf("singleAnchor accepted symlink %q", got)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
