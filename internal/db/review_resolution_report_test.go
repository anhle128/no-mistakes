package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenCreatesReviewResolutionReportsTable(t *testing.T) {
	d := openTestDB(t)

	var count int
	if err := d.sql.QueryRow("SELECT count(*) FROM review_resolution_reports").Scan(&count); err != nil {
		t.Fatalf("review_resolution_reports table missing: %v", err)
	}
}

func TestOpenMigratesReviewResolutionReportsTableIntoExistingDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")

	legacyDB, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if _, err := legacyDB.Exec(`
		CREATE TABLE repos (
			id TEXT PRIMARY KEY,
			working_path TEXT NOT NULL UNIQUE,
			upstream_url TEXT NOT NULL,
			default_branch TEXT NOT NULL DEFAULT 'main',
			created_at INTEGER NOT NULL
		);
		CREATE TABLE runs (
			id TEXT PRIMARY KEY,
			repo_id TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
			branch TEXT NOT NULL,
			head_sha TEXT NOT NULL,
			base_sha TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			pr_url TEXT,
			error TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);
	`); err != nil {
		legacyDB.Close()
		t.Fatalf("create legacy db: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open migrated db: %v", err)
	}
	defer d.Close()

	var count int
	if err := d.sql.QueryRow("SELECT count(*) FROM review_resolution_reports").Scan(&count); err != nil {
		t.Fatalf("review_resolution_reports table missing after migration: %v", err)
	}
}

func TestReviewResolutionReportMetadataRoundTripAndUpdate(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/repo", "git@example.com:repo.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}

	path := "/tmp/nm/reports/run1/review-resolution.md"
	stepID := "step-review"
	roundID := "round-2"
	generatedAt := int64(1700000001)
	errText := "newer review evidence exists"
	meta := ReviewResolutionReportMetadata{
		RunID:               run.ID,
		ReportPath:          &path,
		Status:              "stale",
		ContractVersion:     "review-resolution-report/v1",
		LatestOutcome:       "review resolution incomplete",
		SummaryCountsJSON:   `{"total_findings":1}`,
		GenerationMode:      "live",
		SourceSnapshotAt:    1700000000,
		SourceStepResultID:  &stepID,
		SourceRoundIDsJSON:  `["round-1","round-2"]`,
		LatestReviewRoundID: &roundID,
		GeneratedAt:         &generatedAt,
		UpdatedAt:           1700000002,
		Stale:               true,
		SafeError:           &errText,
	}
	if err := d.UpsertReviewResolutionReportMetadata(meta); err != nil {
		t.Fatalf("upsert metadata: %v", err)
	}

	got, err := d.GetReviewResolutionReportMetadata(run.ID)
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if got == nil {
		t.Fatal("expected metadata")
	}
	if got.ReportPath == nil || *got.ReportPath != path {
		t.Fatalf("report path = %v, want %q", got.ReportPath, path)
	}
	if got.Status != "stale" || !got.Stale {
		t.Fatalf("status/stale = %q/%v, want stale/true", got.Status, got.Stale)
	}
	if got.SafeError == nil || *got.SafeError != errText {
		t.Fatalf("safe error = %v, want %q", got.SafeError, errText)
	}

	meta.Status = "current"
	meta.Stale = false
	meta.SafeError = nil
	meta.UpdatedAt = 1700000003
	if err := d.UpsertReviewResolutionReportMetadata(meta); err != nil {
		t.Fatalf("update metadata: %v", err)
	}
	got, err = d.GetReviewResolutionReportMetadata(run.ID)
	if err != nil {
		t.Fatalf("get updated metadata: %v", err)
	}
	if got.Status != "current" || got.Stale || got.SafeError != nil || got.UpdatedAt != 1700000003 {
		t.Fatalf("updated metadata = %+v", got)
	}
}

func TestReviewResolutionReportMetadataCascadesWithRun(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/repo", "git@example.com:repo.git", "main")
	if err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "head", "base")
	if err != nil {
		t.Fatalf("insert run: %v", err)
	}
	meta := ReviewResolutionReportMetadata{
		RunID:              run.ID,
		Status:             "unavailable",
		ContractVersion:    "review-resolution-report/v1",
		LatestOutcome:      "final findings unavailable",
		SummaryCountsJSON:  `{"total_findings":0}`,
		GenerationMode:     "live",
		SourceSnapshotAt:   1700000000,
		SourceRoundIDsJSON: `[]`,
		UpdatedAt:          1700000000,
	}
	if err := d.UpsertReviewResolutionReportMetadata(meta); err != nil {
		t.Fatalf("upsert metadata: %v", err)
	}
	if _, err := d.sql.Exec(`DELETE FROM runs WHERE id = ?`, run.ID); err != nil {
		t.Fatalf("delete run: %v", err)
	}

	got, err := d.GetReviewResolutionReportMetadata(run.ID)
	if err != nil {
		t.Fatalf("get metadata after delete: %v", err)
	}
	if got != nil {
		t.Fatalf("expected metadata to cascade delete, got %+v", got)
	}
}
