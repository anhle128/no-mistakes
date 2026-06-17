package reviewreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestUpdateWritesReportAndMetadata(t *testing.T) {
	root := t.TempDir()
	p := paths.WithRoot(root)
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(filepath.Join(root, "state.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	repo, err := database.InsertRepo("/repo", "git@example.com:repo.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := database.InsertRun(repo.ID, "feature", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := database.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"bad input","action":"auto-fix"}],"summary":"1 finding"}`
	selected := `["review-1"]`
	fixSummary := "validate input"
	final := `{"findings":[],"summary":"clean","risk_level":"low","risk_rationale":"clean"}`
	round1, err := database.InsertStepRound(step.ID, 1, "initial", &findings, nil, 10)
	if err != nil {
		t.Fatalf("insert round 1: %v", err)
	}
	if err := database.SetStepRoundSelection(round1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatalf("set selection: %v", err)
	}
	if _, err := database.InsertStepRound(step.ID, 2, "auto_fix", &final, &fixSummary, 10); err != nil {
		t.Fatalf("insert round 2: %v", err)
	}
	if err := database.CompleteStep(step.ID, 0, 30, "/tmp/review.log"); err != nil {
		t.Fatalf("complete step: %v", err)
	}

	meta, err := Update(database, p, run.ID, GenerationModeLive)
	if err != nil {
		t.Fatalf("update report: %v", err)
	}
	if meta.Status != StatusCurrent {
		t.Fatalf("status = %q", meta.Status)
	}
	if meta.ReportPath == nil || *meta.ReportPath != p.RunReviewResolutionReportPath(run.ID) {
		t.Fatalf("report path = %v", meta.ReportPath)
	}
	body, err := os.ReadFile(*meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(body), "# Review Resolution Report") || !strings.Contains(string(body), "`Applied fix`: validate input") {
		t.Fatalf("unexpected report body:\n%s", body)
	}
	var counts map[string]int
	if err := json.Unmarshal([]byte(meta.SummaryCountsJSON), &counts); err != nil {
		t.Fatalf("parse counts: %v", err)
	}
	if counts[CountSelectedForFix] != 1 || counts[CountAppliedFixSummaries] != 1 {
		t.Fatalf("counts = %+v", counts)
	}

	stored, err := database.GetReviewResolutionReportMetadata(run.ID)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if stored == nil || stored.LatestOutcome != LatestOutcomeNoIssuesRemain {
		t.Fatalf("stored metadata = %+v", stored)
	}
}

func TestUpdateWithoutReviewRoundsDoesNotWriteReport(t *testing.T) {
	root := t.TempDir()
	p := paths.WithRoot(root)
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(filepath.Join(root, "state.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	repo, err := database.InsertRepo("/repo", "git@example.com:repo.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := database.InsertRun(repo.ID, "feature", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	if _, err := database.InsertStepResult(run.ID, types.StepReview); err != nil {
		t.Fatalf("insert review step: %v", err)
	}

	meta, err := Update(database, p, run.ID, GenerationModeLive)
	if err != nil {
		t.Fatalf("update report: %v", err)
	}
	if meta.Status != StatusUnavailable {
		t.Fatalf("status = %q, want unavailable", meta.Status)
	}
	if meta.ReportPath != nil {
		t.Fatalf("report path = %v, want nil", *meta.ReportPath)
	}
	if _, err := os.Stat(p.RunReviewResolutionReportPath(run.ID)); !os.IsNotExist(err) {
		t.Fatalf("report artifact stat err = %v, want not exist", err)
	}
}

func TestUpdateFirstWriteFailureDoesNotPersistPlannedPath(t *testing.T) {
	root := t.TempDir()
	p := paths.WithRoot(root)
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	database, run := createReportableReviewRun(t, root)
	defer database.Close()

	if err := os.WriteFile(p.RunReportDir(run.ID), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("block report dir: %v", err)
	}

	meta, err := Update(database, p, run.ID, GenerationModeLive)
	if err == nil {
		t.Fatal("expected update report to fail")
	}
	if meta.ReportPath != nil {
		t.Fatalf("report path = %v, want nil on first write failure", *meta.ReportPath)
	}
	if meta.GeneratedAt != nil {
		t.Fatalf("generated_at = %v, want nil on first write failure", *meta.GeneratedAt)
	}
	if meta.Status != StatusUnavailable {
		t.Fatalf("status = %q, want unavailable", meta.Status)
	}
	if meta.SafeError == nil || !strings.Contains(*meta.SafeError, "create report directory") {
		t.Fatalf("safe error = %v, want create report directory", meta.SafeError)
	}
}

func TestUpdateWriteFailurePreservesExistingReportAsStale(t *testing.T) {
	root := t.TempDir()
	p := paths.WithRoot(root)
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	database, run := createReportableReviewRun(t, root)
	defer database.Close()

	meta, err := Update(database, p, run.ID, GenerationModeLive)
	if err != nil {
		t.Fatalf("initial update report: %v", err)
	}
	if meta.ReportPath == nil || meta.GeneratedAt == nil {
		t.Fatalf("initial metadata missing path/generated_at: %+v", meta)
	}
	oldPath := *meta.ReportPath
	oldGeneratedAt := *meta.GeneratedAt
	oldLatestOutcome := meta.LatestOutcome
	oldSummaryCounts := meta.SummaryCountsJSON
	oldSourceRounds := meta.SourceRoundIDsJSON
	oldLatestReviewRoundID := meta.LatestReviewRoundID

	steps, err := database.GetStepsByRun(run.ID)
	if err != nil {
		t.Fatalf("get steps: %v", err)
	}
	step := findReviewStep(steps)
	if step == nil {
		t.Fatal("review step not found")
	}
	nextFindings := `{"findings":[{"id":"review-2","severity":"warning","description":"new issue","action":"auto-fix"}],"summary":"1 finding","risk_level":"medium","risk_rationale":"new issue remains"}`
	if _, err := database.InsertStepRound(step.ID, 2, "initial", &nextFindings, nil, 10); err != nil {
		t.Fatalf("insert new round: %v", err)
	}

	if err := os.RemoveAll(p.RunReportDir(run.ID)); err != nil {
		t.Fatalf("remove report dir: %v", err)
	}
	if err := os.WriteFile(p.RunReportDir(run.ID), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("block report dir: %v", err)
	}

	meta, err = Update(database, p, run.ID, GenerationModeLive)
	if err == nil {
		t.Fatal("expected update report to fail")
	}
	if meta.Status != StatusStale || !meta.Stale {
		t.Fatalf("metadata = %+v, want stale", meta)
	}
	if meta.ReportPath == nil || *meta.ReportPath != oldPath {
		t.Fatalf("report path = %v, want existing %q", meta.ReportPath, oldPath)
	}
	if meta.GeneratedAt == nil || *meta.GeneratedAt != oldGeneratedAt {
		t.Fatalf("generated_at = %v, want existing %d", meta.GeneratedAt, oldGeneratedAt)
	}
	if meta.LatestOutcome != oldLatestOutcome {
		t.Fatalf("latest_outcome = %q, want existing %q", meta.LatestOutcome, oldLatestOutcome)
	}
	if meta.SummaryCountsJSON != oldSummaryCounts {
		t.Fatalf("summary_counts_json = %q, want existing %q", meta.SummaryCountsJSON, oldSummaryCounts)
	}
	if meta.SourceRoundIDsJSON != oldSourceRounds {
		t.Fatalf("source_round_ids_json = %q, want existing %q", meta.SourceRoundIDsJSON, oldSourceRounds)
	}
	if (meta.LatestReviewRoundID == nil) != (oldLatestReviewRoundID == nil) {
		t.Fatalf("latest_review_round_id = %v, want existing %v", meta.LatestReviewRoundID, oldLatestReviewRoundID)
	}
	if meta.LatestReviewRoundID != nil && oldLatestReviewRoundID != nil && *meta.LatestReviewRoundID != *oldLatestReviewRoundID {
		t.Fatalf("latest_review_round_id = %q, want existing %q", *meta.LatestReviewRoundID, *oldLatestReviewRoundID)
	}
}

func createReportableReviewRun(t *testing.T, root string) (*db.DB, *db.Run) {
	t.Helper()
	database, err := db.Open(filepath.Join(root, "state.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	repo, err := database.InsertRepo("/repo", "git@example.com:repo.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := database.InsertRun(repo.ID, "feature", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := database.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatalf("insert step: %v", err)
	}
	final := `{"findings":[],"summary":"clean","risk_level":"low","risk_rationale":"clean"}`
	if _, err := database.InsertStepRound(step.ID, 1, "initial", &final, nil, 10); err != nil {
		t.Fatalf("insert round: %v", err)
	}
	if err := database.CompleteStep(step.ID, 0, 10, "/tmp/review.log"); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	return database, run
}
