package cli

import (
	"path/filepath"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestAttachReviewResolutionFromDBOmitsCleanRun(t *testing.T) {
	database := openTestDB(t)
	repo, err := database.InsertRepo(t.TempDir(), "origin", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := database.InsertRun(repo.ID, "feature/clean-review", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}

	rv := runView{ID: run.ID}
	attachReviewResolutionFromDB(database, &rv)
	if rv.ReviewResolution != nil {
		t.Fatalf("clean run attached review resolution metadata: %+v", rv.ReviewResolution)
	}
}

func TestAttachReviewResolutionFromDBMarksMissingReportStale(t *testing.T) {
	database := openTestDB(t)
	repo, err := database.InsertRepo(t.TempDir(), "origin", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := database.InsertRun(repo.ID, "feature/stale-review-report", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	step, err := database.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatalf("insert review step: %v", err)
	}
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"x","action":"auto-fix"}],"summary":"1"}`
	if _, err := database.InsertStepRound(step.ID, 1, "initial", &findings, nil, 10); err != nil {
		t.Fatalf("insert review round: %v", err)
	}
	if err := database.UpsertReviewResolutionReport(db.ReviewResolutionReport{
		RunID:             run.ID,
		ReportPath:        filepath.Join(t.TempDir(), "missing.md"),
		Status:            db.ReviewResolutionStatusFinal,
		ResolvedCount:     1,
		ReportVersion:     "1",
		EntryCount:        1,
		SourceWatermark:   "old-watermark",
		ContentHash:       "not-the-file-hash",
		LastRefreshResult: "ok",
		FirstGeneratedAt:  1,
		LastRefreshedAt:   2,
	}); err != nil {
		t.Fatalf("upsert report: %v", err)
	}

	rv := runView{ID: run.ID}
	attachReviewResolutionFromDB(database, &rv)
	if rv.ReviewResolution == nil {
		t.Fatal("expected attached review resolution metadata")
	}
	if rv.ReviewResolution.Status != db.ReviewResolutionStatusStale {
		t.Fatalf("status = %s, want stale", rv.ReviewResolution.Status)
	}
	if !rv.ReviewResolution.Exists || rv.ReviewResolution.ResolvedCount != 1 || rv.ReviewResolution.Path == "" {
		t.Fatalf("unexpected review resolution info: %+v", rv.ReviewResolution)
	}
}
