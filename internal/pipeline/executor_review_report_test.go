package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"

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
