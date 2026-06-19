package reviewreport

import (
	"os"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestClassifierRepeatedFindingIDUpdatesSingleEntry(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	first := `{"findings":[{"id":"review-1","severity":"warning","file":"old.go","line":3,"description":"old description","action":"auto-fix"}],"summary":"1 finding"}`
	if _, err := d.InsertStepRound(step.ID, 1, "initial", &first, nil, 10); err != nil {
		t.Fatalf("insert first round: %v", err)
	}
	second := `{"findings":[{"id":"review-1","severity":"error","file":"new.go","line":7,"description":"updated description","action":"ask-user"}],"summary":"still present"}`
	if _, err := d.InsertStepRound(step.ID, 2, "follow_up", &second, nil, 10); err != nil {
		t.Fatalf("insert second round: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.EntryCount != 1 || meta.StillOpenCount != 1 {
		t.Fatalf("metadata = %+v, want one still-open entry", meta)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	for _, want := range []string{"Severity: error", "File and line: new.go:7", "Description: updated description", "Last seen Review round ID: 2"} {
		if !strings.Contains(md, want) {
			t.Fatalf("report missing %q:\n%s", want, md)
		}
	}
	for _, disallowed := range []string{"old description", "old.go:3"} {
		if strings.Contains(md, disallowed) {
			t.Fatalf("report retained stale value %q:\n%s", disallowed, md)
		}
	}
}

func TestClassifierStatusForIncompleteRunStates(t *testing.T) {
	open := Entry{Finding: types.Finding{ID: "review-1"}, Outcome: OutcomeStillOpen}
	resolved := Entry{Finding: types.Finding{ID: "review-1"}, Outcome: OutcomeResolved}
	cases := []struct {
		name         string
		runStatus    types.RunStatus
		reviewStatus types.StepStatus
		snap         Snapshot
		want         string
	}{
		{
			name:         "awaiting approval remains in progress",
			runStatus:    types.RunRunning,
			reviewStatus: types.StepStatusAwaitingApproval,
			snap:         Snapshot{Entries: []Entry{open}},
			want:         db.ReviewResolutionStatusInProgress,
		},
		{
			name:         "failed open issue is incomplete",
			runStatus:    types.RunFailed,
			reviewStatus: types.StepStatusFailed,
			snap:         Snapshot{Entries: []Entry{open}},
			want:         db.ReviewResolutionStatusIncomplete,
		},
		{
			name:         "completed open issue is incomplete",
			runStatus:    types.RunCompleted,
			reviewStatus: types.StepStatusCompleted,
			snap:         Snapshot{Entries: []Entry{open}},
			want:         db.ReviewResolutionStatusIncomplete,
		},
		{
			name:         "degraded overrides terminal status",
			runStatus:    types.RunCompleted,
			reviewStatus: types.StepStatusCompleted,
			snap:         Snapshot{Degraded: true, Entries: []Entry{resolved}},
			want:         db.ReviewResolutionStatusDegraded,
		},
		{
			name:         "completed resolved issue is final",
			runStatus:    types.RunCompleted,
			reviewStatus: types.StepStatusCompleted,
			snap:         Snapshot{Entries: []Entry{resolved}},
			want:         db.ReviewResolutionStatusFinal,
		},
		{
			name:         "completed skipped accepted issue is final",
			runStatus:    types.RunCompleted,
			reviewStatus: types.StepStatusSkipped,
			snap:         Snapshot{Entries: []Entry{{Finding: types.Finding{ID: "review-1"}, Outcome: OutcomeAccepted}}},
			want:         db.ReviewResolutionStatusFinal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyReportStatus(tc.runStatus, tc.reviewStatus, tc.snap); got != tc.want {
				t.Fatalf("classifyReportStatus() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestClassifierFollowupStillReportingSelectedIDKeepsFindingOpen(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"fix target","action":"auto-fix"}],"summary":"1 finding"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "attempted fix"
	followup := `{"findings":[{"id":"review-1","severity":"warning","description":"fix target still present","action":"auto-fix"}],"summary":"still present"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &followup, &summary, nil, nil, nil, 20); err != nil {
		t.Fatalf("insert fix round: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.ResolvedCount != 0 || meta.StillOpenCount != 1 || meta.Status != db.ReviewResolutionStatusIncomplete {
		t.Fatalf("metadata = %+v, want one incomplete still-open finding", meta)
	}
}
