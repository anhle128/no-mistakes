package db

import (
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestReviewResolutionReportAccessors(t *testing.T) {
	d := openTestDB(t)
	repo, _ := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "abc", "def")

	start, end, finalized := 1, 3, int64(123)
	report := ReviewResolutionReport{
		RunID:              run.ID,
		ReportPath:         "/tmp/review-resolution.md",
		Status:             ReviewResolutionStatusFinal,
		ResolvedCount:      1,
		AcceptedCount:      2,
		InformationalCount: 3,
		StillOpenCount:     4,
		ReportVersion:      "1",
		EntryCount:         10,
		SourceRoundStart:   &start,
		SourceRoundEnd:     &end,
		SourceWatermark:    "watermark",
		ContentHash:        "hash",
		LastRefreshResult:  "ok",
		FirstGeneratedAt:   100,
		LastRefreshedAt:    200,
		FinalizedAt:        &finalized,
	}
	if err := d.UpsertReviewResolutionReport(report); err != nil {
		t.Fatalf("upsert report: %v", err)
	}

	got, err := d.GetReviewResolutionReport(run.ID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if got == nil {
		t.Fatal("expected report")
	}
	if got.Status != ReviewResolutionStatusFinal || got.ResolvedCount != 1 || got.AcceptedCount != 2 || got.InformationalCount != 3 || got.StillOpenCount != 4 {
		t.Fatalf("unexpected report counts/status: %+v", got)
	}

	report.Status = ReviewResolutionStatusIncomplete
	report.ResolvedCount = 9
	report.FirstGeneratedAt = 999
	if err := d.UpsertReviewResolutionReport(report); err != nil {
		t.Fatalf("second upsert report: %v", err)
	}
	got, err = d.GetReviewResolutionReport(run.ID)
	if err != nil {
		t.Fatalf("get report after update: %v", err)
	}
	if got.Status != ReviewResolutionStatusIncomplete || got.ResolvedCount != 9 {
		t.Fatalf("report not updated: %+v", got)
	}
	if got.FirstGeneratedAt != 100 {
		t.Fatalf("first_generated_at = %d, want preserved 100", got.FirstGeneratedAt)
	}
}

func TestReviewResolutionDecisionAccessors(t *testing.T) {
	d := openTestDB(t)
	repo, _ := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "abc", "def")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)
	round, _ := d.InsertStepRound(step.ID, 1, "initial", nil, nil, 10)
	reason := "approved by test"

	decision, err := d.InsertReviewResolutionDecision(ReviewResolutionDecision{
		RunID:        run.ID,
		StepResultID: step.ID,
		RoundID:      &round.ID,
		FindingID:    "review-1",
		Action:       ReviewResolutionDecisionApprove,
		ActorSource:  "user",
		Reason:       &reason,
	})
	if err != nil {
		t.Fatalf("insert decision: %v", err)
	}
	if decision.ID == "" || decision.CreatedAt == 0 {
		t.Fatalf("decision missing generated fields: %+v", decision)
	}

	got, err := d.GetReviewResolutionDecisions(run.ID)
	if err != nil {
		t.Fatalf("get decisions: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d decisions, want 1", len(got))
	}
	if got[0].FindingID != "review-1" || got[0].Action != ReviewResolutionDecisionApprove || got[0].Reason == nil || *got[0].Reason != reason {
		t.Fatalf("unexpected decision: %+v", got[0])
	}
}

func TestStepRoundFixEvidence(t *testing.T) {
	d := openTestDB(t)
	repo, _ := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "abc", "def")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	summary := "fix review issue"
	sha := "abc123"
	details := `{"resolutions":[{"finding_id":"review-1","applied_solution":"fixed","why_this_solution":"correct","changed_files":["a.go"]}]}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 1, "auto_fix", nil, &summary, &sha, nil, &details, 20); err != nil {
		t.Fatalf("insert round evidence: %v", err)
	}
	rounds, err := d.GetRoundsByStep(step.ID)
	if err != nil {
		t.Fatalf("get rounds: %v", err)
	}
	if len(rounds) != 1 {
		t.Fatalf("got %d rounds, want 1", len(rounds))
	}
	if rounds[0].FixCommitSHA == nil || *rounds[0].FixCommitSHA != sha {
		t.Fatalf("fix commit sha = %v, want %q", rounds[0].FixCommitSHA, sha)
	}
	if rounds[0].FixResolutionDetailsJSON == nil || *rounds[0].FixResolutionDetailsJSON != details {
		t.Fatalf("fix details = %v, want %q", rounds[0].FixResolutionDetailsJSON, details)
	}
}
