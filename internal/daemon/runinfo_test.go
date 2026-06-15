package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/reviewhandoff"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestStepToInfoIncludesFixSummaries(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc", "def")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := d.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}

	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"x"}],"summary":"1"}`
	if _, err := d.InsertStepRound(step.ID, 1, "initial", &findings, nil, 100); err != nil {
		t.Fatalf("insert round 1: %v", err)
	}
	sum := "handle nil pointer in executor"
	if _, err := d.InsertStepRound(step.ID, 2, "auto_fix", nil, &sum, 100); err != nil {
		t.Fatalf("insert round 2: %v", err)
	}

	info := stepToInfo(d, nil, nil, step)
	if len(info.FixSummaries) != 1 || info.FixSummaries[0] != sum {
		t.Errorf("fix summaries = %v, want [%q]", info.FixSummaries, sum)
	}
}

func TestStepToInfoNoFixSummariesWithoutFixRounds(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc", "def")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := d.InsertStepResult(run.ID, types.StepLint)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	if _, err := d.InsertStepRound(step.ID, 1, "initial", nil, nil, 100); err != nil {
		t.Fatalf("insert round: %v", err)
	}

	info := stepToInfo(d, nil, nil, step)
	if len(info.FixSummaries) != 0 {
		t.Errorf("fix summaries = %v, want none", info.FixSummaries)
	}
}

func TestRecordReviewAutomationDecisionStampsHandoffFile(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := d.InsertRun(repo.ID, "feature/review-file", "abc", "def")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := d.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	findings := types.Findings{Items: []types.Finding{{
		ID:           "review-1",
		Severity:     "warning",
		Description:  "missing validation",
		SuggestedFix: "Option 1: Add validation before parsing.",
		Action:       types.ActionAutoFix,
	}}}
	findingsJSON := `{"findings":[{"id":"review-1","severity":"warning","description":"missing validation","suggested_fix":"Option 1: Add validation before parsing.","action":"auto-fix"}]}`
	if err := d.SetStepFindings(step.ID, findingsJSON); err != nil {
		t.Fatalf("set findings: %v", err)
	}
	rel := ".no-mistakes/issues/feature-review-file/review-issues-run.md"
	content, state, err := reviewhandoff.RenderPending(reviewhandoff.RenderInput{
		RunID:        run.ID,
		Branch:       run.Branch,
		Status:       types.StepStatusAwaitingApproval,
		RelativePath: rel,
		CycleID:      "cycle-1",
		Findings:     findings,
		Now:          100,
	})
	if err != nil {
		t.Fatalf("render handoff: %v", err)
	}
	if err := d.SetStepReviewHandoff(step.ID, state); err != nil {
		t.Fatalf("set handoff: %v", err)
	}
	p := paths.WithRoot(t.TempDir())
	worktree := p.WorktreeDir(repo.ID, run.ID)
	if err := reviewhandoff.WritePending(worktree, rel, content); err != nil {
		t.Fatalf("write pending: %v", err)
	}

	mgr := NewRunManager(d, p, nil)
	mgr.recordReviewAutomationDecision(run.ID, types.ActionFix, []string{"review-1"}, map[string]string{"review-1": "Use a strict parser."})

	updated, err := d.StepReviewHandoff(step.ID)
	if err != nil {
		t.Fatalf("load handoff: %v", err)
	}
	if updated == nil || updated.ProcessedAction != reviewhandoff.ProcessedFix || updated.DecisionSource != reviewhandoff.DecisionSourceAutomation {
		t.Fatalf("updated state = %+v", updated)
	}
	path := filepath.Join(worktree, filepath.FromSlash(rel))
	stamped, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stamped file: %v", err)
	}
	for _, want := range []string{"processed_action: fix", "Resolved Decision Summary", "`review-1`: fix", "Use a strict parser."} {
		if !strings.Contains(string(stamped), want) {
			t.Fatalf("stamped handoff missing %q:\n%s", want, stamped)
		}
	}
}

func TestStepToInfoIncludesOpenableReviewFilePath(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer d.Close()

	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := d.InsertRun(repo.ID, "feature/review-file", "abc", "def")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := d.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	rel := ".no-mistakes/issues/feature-review-file/review-issues-run.md"
	state := reviewhandoff.NewState(rel, "cycle-1", "digest-1", "generated-1", 100)
	if err := d.SetStepReviewHandoff(step.ID, state); err != nil {
		t.Fatalf("set handoff: %v", err)
	}
	step, err = d.GetStepResult(step.ID)
	if err != nil {
		t.Fatalf("reload step: %v", err)
	}
	p := paths.WithRoot(t.TempDir())

	info := stepToInfo(d, p, run, step)
	wantPath := filepath.Join(p.WorktreeDir(repo.ID, run.ID), filepath.FromSlash(rel))
	if info.ReviewFile == nil || *info.ReviewFile != rel {
		t.Fatalf("review_file = %v, want %q", info.ReviewFile, rel)
	}
	if info.ReviewFilePath == nil || *info.ReviewFilePath != wantPath {
		t.Fatalf("review_file_path = %v, want %q", info.ReviewFilePath, wantPath)
	}
}
