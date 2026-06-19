package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestExecutorFinalizesReviewResolutionReportOnRunCompletion(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()

	findings := `{"findings":[{"id":"review-1","severity":"info","description":"documented tradeoff","action":"no-op"}],"summary":"1 informational"}`
	review := &mockStep{
		name: types.StepReview,
		outcome: &StepOutcome{
			Findings: findings,
			ExitCode: 0,
		},
	}
	exec := NewExecutor(database, p, nil, nil, []Step{review}, nil)

	if err := exec.Execute(context.Background(), run, repo, workDir); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	report, err := database.GetReviewResolutionReport(run.ID)
	if err != nil {
		t.Fatalf("get review report: %v", err)
	}
	if report == nil {
		t.Fatal("expected review resolution report metadata")
	}
	if report.Status != db.ReviewResolutionStatusFinal {
		t.Fatalf("status = %s, want final", report.Status)
	}
	if report.InformationalCount != 1 || report.EntryCount != 1 {
		t.Fatalf("unexpected report counts: %+v", report)
	}
	raw, err := os.ReadFile(report.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(raw), "## Informational / No Action Required") {
		t.Fatalf("report missing informational section:\n%s", string(raw))
	}
}

func TestExecutorDoesNotCreateReviewResolutionReportForNonReviewFindings(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()

	findings := `{"findings":[{"id":"test-1","severity":"info","description":"test note","action":"no-op"}],"summary":"1 note"}`
	testStep := &mockStep{
		name: types.StepTest,
		outcome: &StepOutcome{
			Findings: findings,
			ExitCode: 0,
		},
	}
	exec := NewExecutor(database, p, nil, nil, []Step{testStep}, nil)

	if err := exec.Execute(context.Background(), run, repo, workDir); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	report, err := database.GetReviewResolutionReport(run.ID)
	if err != nil {
		t.Fatalf("get review report: %v", err)
	}
	if report != nil {
		t.Fatalf("non-review findings created review resolution report: %+v", report)
	}
	if _, err := os.Stat(p.ReviewResolutionReportPath(run.ID)); !os.IsNotExist(err) {
		t.Fatalf("expected no review resolution report file, stat err=%v", err)
	}
}

func TestExecutorRefreshesReviewResolutionReportOnReviewAbort(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()

	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"requires decision","action":"ask-user"}],"summary":"1 warning"}`
	review := &mockStep{
		name: types.StepReview,
		outcome: &StepOutcome{
			NeedsApproval: true,
			Findings:      findings,
			ExitCode:      1,
		},
	}
	exec := NewExecutor(database, p, nil, nil, []Step{review}, nil)

	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()
	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	if err := exec.Respond(types.StepReview, types.ActionAbort, nil); err != nil {
		t.Fatalf("respond abort: %v", err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected abort to fail run")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}

	report, err := database.GetReviewResolutionReport(run.ID)
	if err != nil {
		t.Fatalf("get review report: %v", err)
	}
	if report == nil {
		t.Fatal("expected review resolution report metadata")
	}
	if report.Status != db.ReviewResolutionStatusIncomplete || report.StillOpenCount != 1 || report.EntryCount != 1 {
		t.Fatalf("unexpected report metadata after abort: %+v", report)
	}
	raw, err := os.ReadFile(report.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(raw), "## Still Open Issues") || !strings.Contains(string(raw), "### review-1") {
		t.Fatalf("aborted report did not preserve open issue:\n%s", string(raw))
	}
}
