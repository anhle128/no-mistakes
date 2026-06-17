package pipeline

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/reviewreport"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestExecutor_ReviewFixLoopWritesResolutionReport(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()
	callCount := 0
	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"bad input","suggested_fix":"validate it","action":"auto-fix"}],"summary":"1 finding"}`
	final := `{"findings":[],"summary":"clean","risk_level":"low","risk_rationale":"clean"}`
	step := &adaptiveCallStep{
		name: types.StepReview,
		fn: func(sctx *StepContext) (*StepOutcome, error) {
			callCount++
			if callCount == 1 {
				return &StepOutcome{NeedsApproval: true, Findings: initial}, nil
			}
			return &StepOutcome{Findings: final, FixSummary: "validate input"}, nil
		},
	}
	exec := NewExecutor(database, p, nil, nil, []Step{step}, nil)
	events := collectEvents(exec)
	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()

	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	if err := exec.Respond(types.StepReview, types.ActionFix, []string{"review-1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("execute: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}

	meta, err := database.GetReviewResolutionReportMetadata(run.ID)
	if err != nil {
		t.Fatalf("get report metadata: %v", err)
	}
	if meta == nil {
		t.Fatal("expected review report metadata")
	}
	if meta.Status != reviewreport.StatusCurrent || meta.LatestOutcome != reviewreport.LatestOutcomeNoIssuesRemain {
		t.Fatalf("metadata = %+v", meta)
	}
	if meta.ReportPath == nil {
		t.Fatal("expected report path")
	}
	body, err := os.ReadFile(*meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "`Recommendation`: validate it") {
		t.Fatalf("report did not render recommendation:\n%s", body)
	}
	if !strings.Contains(string(body), "`Applied fix`: validate input") {
		t.Fatalf("report did not render applied fix:\n%s", body)
	}
	completed := events.findRunEvent(ipc.EventRunCompleted)
	if completed == nil || completed.ReviewResolutionReport == nil {
		t.Fatalf("expected run_completed event to include report metadata, got %+v", completed)
	}
}
