package daemon

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/reviewreport"
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

	info := stepToInfo(d, step)
	if len(info.FixSummaries) != 1 || info.FixSummaries[0] != sum {
		t.Errorf("fix summaries = %v, want [%q]", info.FixSummaries, sum)
	}
}

func TestStepToInfoSanitizesUnsafeFixSummaries(t *testing.T) {
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

	unsafe := `fmt.Sprintf("x: %s", value)`
	if _, err := d.InsertStepRound(step.ID, 2, "auto_fix", nil, &unsafe, 100); err != nil {
		t.Fatalf("insert fix round: %v", err)
	}

	info := stepToInfo(d, step)
	if len(info.FixSummaries) != 1 {
		t.Fatalf("fix summaries = %v, want one", info.FixSummaries)
	}
	if got := info.FixSummaries[0]; got != reviewreport.AppliedFixSummaryDisplayOmitted {
		t.Fatalf("fix summary = %q, want %q", got, reviewreport.AppliedFixSummaryDisplayOmitted)
	}
	if strings.Contains(strings.Join(info.FixSummaries, " "), "fmt.Sprintf") {
		t.Fatalf("unsafe fix summary leaked: %v", info.FixSummaries)
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

	info := stepToInfo(d, step)
	if len(info.FixSummaries) != 0 {
		t.Errorf("fix summaries = %v, want none", info.FixSummaries)
	}
}

func TestRunToInfoIncludesReviewResolutionReport(t *testing.T) {
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
	path := "/tmp/nm/reports/run/review-resolution.md"
	generatedAt := int64(1700000001)
	safeErr := "newer review evidence exists"
	if err := d.UpsertReviewResolutionReportMetadata(db.ReviewResolutionReportMetadata{
		RunID:              run.ID,
		ReportPath:         &path,
		Status:             reviewreport.StatusStale,
		ContractVersion:    reviewreport.ContractVersion,
		LatestOutcome:      reviewreport.LatestOutcomeReviewResolutionIncomplete,
		SummaryCountsJSON:  `{"total_findings":2,"selected_for_fix":1}`,
		GenerationMode:     reviewreport.GenerationModeLive,
		SourceSnapshotAt:   1700000000,
		SourceRoundIDsJSON: `["round-1"]`,
		GeneratedAt:        &generatedAt,
		UpdatedAt:          1700000002,
		Stale:              true,
		SafeError:          &safeErr,
	}); err != nil {
		t.Fatalf("upsert report metadata: %v", err)
	}

	info := runToInfo(d, run, nil)
	if info.ReviewResolutionReport == nil {
		t.Fatal("expected review resolution report metadata")
	}
	report := info.ReviewResolutionReport
	if report.Path != path {
		t.Fatalf("report path = %q, want %q", report.Path, path)
	}
	if report.Status != reviewreport.StatusStale || !report.Stale {
		t.Fatalf("report status/stale = %q/%v", report.Status, report.Stale)
	}
	if report.LatestOutcome != reviewreport.LatestOutcomeReviewResolutionIncomplete {
		t.Fatalf("latest outcome = %q", report.LatestOutcome)
	}
	if report.SummaryCounts["total_findings"] != 2 || report.SummaryCounts["selected_for_fix"] != 1 {
		t.Fatalf("summary counts = %+v", report.SummaryCounts)
	}
	if report.GeneratedAt != generatedAt || report.UpdatedAt != 1700000002 {
		t.Fatalf("timestamps = generated %d updated %d", report.GeneratedAt, report.UpdatedAt)
	}
	if report.Error != safeErr {
		t.Fatalf("error = %q, want %q", report.Error, safeErr)
	}
}
