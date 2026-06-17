package reviewreport_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/pipeline/steps"
	"github.com/kunchenguid/no-mistakes/internal/reviewreport"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestCrossSurfaceSummaryCountsStayConsistent(t *testing.T) {
	t.Parallel()

	reportPath := "/tmp/no-mistakes/reports/run-1/review-resolution.md"
	meta := &db.ReviewResolutionReportMetadata{
		ReportPath:        &reportPath,
		Status:            reviewreport.StatusCurrent,
		LatestOutcome:     reviewreport.LatestOutcomeNoIssuesRemain,
		SummaryCountsJSON: `{"selected_for_fix":2,"fix_attempts":2,"applied_fix_summaries":1,"still_open":0,"decision_not_recorded":1}`,
	}

	info := reviewreport.IPCInfoFromMetadata(meta)
	gotCounts := reviewreport.CompactSummaryCountsFromMap(info.SummaryCounts)
	wantCounts := reviewreport.CompactSummaryCounts{
		SelectedForFix:      2,
		FixAttempts:         2,
		AppliedFixSummaries: 1,
		StillOpen:           0,
		DecisionNotRecorded: 1,
	}
	if gotCounts != wantCounts {
		t.Fatalf("IPC compact counts = %+v, want %+v", gotCounts, wantCounts)
	}

	wantKeys := []string{
		reviewreport.CountSelectedForFix,
		reviewreport.CountFixAttempts,
		reviewreport.CountAppliedFixSummaries,
		reviewreport.CountStillOpen,
		reviewreport.CountDecisionNotRecorded,
	}
	if got := reviewreport.CompactSummaryCountKeys(); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("compact count keys = %#v, want %#v", got, wantKeys)
	}

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"issue one"},{"id":"review-2","severity":"warning","description":"issue two"}]}`
	final := `{"findings":[]}`
	fixSummary := "tighten report metadata projection"
	stepResults := []*db.StepResult{
		{ID: "s1", StepName: types.StepReview, Status: types.StepStatusCompleted, FindingsJSON: &final},
	}
	rounds := map[string][]*db.StepRound{
		"s1": {
			{Round: 1, Trigger: "initial", FindingsJSON: &initial},
			{Round: 2, Trigger: "auto_fix", FindingsJSON: &final, FixSummary: &fixSummary},
			{Round: 3, Trigger: "auto_fix", FindingsJSON: &final},
		},
	}
	prSummary, _ := steps.BuildPipelineSummaryWithReviewReport(stepResults, rounds, meta)

	for _, want := range []string{
		"selected_for_fix=2, fix_attempts=2, applied_fix_summaries=1, still_open=0, decision_not_recorded=1",
		"latest outcome: no issues remain",
		"Applied fix: tighten report metadata projection",
		"Applied fix summaries omitted: 1",
	} {
		if !strings.Contains(prSummary, want) {
			t.Fatalf("PR summary missing cross-surface value %q in:\n%s", want, prSummary)
		}
	}
}
