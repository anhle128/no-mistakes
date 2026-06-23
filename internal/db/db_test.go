package db

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenAndClose(t *testing.T) {
	d := openTestDB(t)
	if d == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestOpenCreatesSchema(t *testing.T) {
	d := openTestDB(t)
	// verify tables exist by querying them
	var count int
	if err := d.sql.QueryRow("SELECT count(*) FROM repos").Scan(&count); err != nil {
		t.Fatalf("repos table missing: %v", err)
	}
	if err := d.sql.QueryRow("SELECT count(*) FROM runs").Scan(&count); err != nil {
		t.Fatalf("runs table missing: %v", err)
	}
	if err := d.sql.QueryRow("SELECT count(*) FROM step_results").Scan(&count); err != nil {
		t.Fatalf("step_results table missing: %v", err)
	}
}

func TestOpenCreatesStepRoundsTable(t *testing.T) {
	d := openTestDB(t)
	var count int
	if err := d.sql.QueryRow("SELECT count(*) FROM step_rounds").Scan(&count); err != nil {
		t.Fatalf("step_rounds table missing: %v", err)
	}
}

func TestOpenMigratesExistingStepRoundsColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")

	legacyDB, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if _, err := legacyDB.Exec(`
		CREATE TABLE step_rounds (
			id TEXT PRIMARY KEY,
			step_result_id TEXT NOT NULL,
			round INTEGER NOT NULL,
			trigger_type TEXT NOT NULL,
			findings_json TEXT,
			duration_ms INTEGER NOT NULL,
			created_at INTEGER NOT NULL
		);
	`); err != nil {
		legacyDB.Close()
		t.Fatalf("create legacy step_rounds table: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open migrated db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	rows, err := d.sql.Query(`PRAGMA table_info(step_rounds)`)
	if err != nil {
		t.Fatalf("pragma table_info(step_rounds): %v", err)
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dfltValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}

	for _, name := range []string{"selected_finding_ids", "selection_source", "fix_summary", "user_findings_json", "fix_commit_sha", "no_commit_reason", "fix_resolution_details_json"} {
		if !columns[name] {
			t.Fatalf("expected migrated column %q to exist", name)
		}
	}

	for _, table := range []string{"review_resolution_reports", "review_resolution_decisions"} {
		var name string
		if err := d.sql.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name); err != nil {
			t.Fatalf("expected migrated table %q to exist: %v", table, err)
		}
	}
}

func TestOpenMigratesLegacyReviewResolutionReportsTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")

	legacyDB, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	if _, err := legacyDB.Exec(schemaSQL); err != nil {
		legacyDB.Close()
		t.Fatalf("create current base schema: %v", err)
	}
	if _, err := legacyDB.Exec(`DROP TABLE review_resolution_reports`); err != nil {
		legacyDB.Close()
		t.Fatalf("drop current review_resolution_reports table: %v", err)
	}
	if _, err := legacyDB.Exec(`
		CREATE TABLE review_resolution_reports (
			run_id                 TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
			report_path            TEXT,
			status                 TEXT NOT NULL,
			contract_version       TEXT NOT NULL,
			latest_outcome         TEXT NOT NULL,
			summary_counts_json    TEXT NOT NULL,
			generation_mode        TEXT NOT NULL,
			source_snapshot_at     INTEGER NOT NULL,
			source_step_result_id  TEXT,
			source_round_ids_json  TEXT NOT NULL,
			latest_review_round_id TEXT,
			latest_fix_round_id    TEXT,
			generated_at           INTEGER,
			updated_at             INTEGER NOT NULL,
			stale                  INTEGER NOT NULL DEFAULT 0,
			safe_error             TEXT
		)
	`); err != nil {
		legacyDB.Close()
		t.Fatalf("create legacy review_resolution_reports table: %v", err)
	}
	if _, err := legacyDB.Exec(`INSERT INTO repos (id, working_path, upstream_url, default_branch, created_at) VALUES ('repo1', '/tmp/project', 'git@example.com:project.git', 'main', 100)`); err != nil {
		legacyDB.Close()
		t.Fatalf("insert legacy repo: %v", err)
	}
	if _, err := legacyDB.Exec(`INSERT INTO runs (id, repo_id, branch, head_sha, base_sha, created_at, updated_at) VALUES ('run1', 'repo1', 'feature', 'head', 'base', 100, 100)`); err != nil {
		legacyDB.Close()
		t.Fatalf("insert legacy run: %v", err)
	}
	if _, err := legacyDB.Exec(`
		INSERT INTO review_resolution_reports (
			run_id, report_path, status, contract_version, latest_outcome,
			summary_counts_json, generation_mode, source_snapshot_at,
			source_round_ids_json, generated_at, updated_at
		) VALUES (
			'run1', '/tmp/legacy-review-resolution.md', 'final', 'legacy',
			'resolved', '{"resolved":1}', 'legacy', 111, '[]', 123, 222
		)
	`); err != nil {
		legacyDB.Close()
		t.Fatalf("insert legacy review resolution report: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	d, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open migrated db: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	columns, err := tableColumns(d.sql, "review_resolution_reports")
	if err != nil {
		t.Fatalf("read migrated columns: %v", err)
	}
	for _, name := range reviewResolutionReportCurrentColumns {
		if !columns[name] {
			t.Fatalf("expected migrated column %q to exist", name)
		}
	}
	for _, name := range reviewResolutionReportLegacyColumns {
		if columns[name] {
			t.Fatalf("legacy column %q should not remain after migration", name)
		}
	}

	report, err := d.GetReviewResolutionReport("run1")
	if err != nil {
		t.Fatalf("get migrated review resolution report: %v", err)
	}
	if report == nil {
		t.Fatal("expected migrated review resolution report row")
	}
	if report.Status != ReviewResolutionStatusEvidenceUnavailable {
		t.Fatalf("migrated status = %q, want %q", report.Status, ReviewResolutionStatusEvidenceUnavailable)
	}
	if report.ReportPath != "/tmp/legacy-review-resolution.md" {
		t.Fatalf("migrated report path = %q", report.ReportPath)
	}
	if report.ResolvedCount != 0 || report.AcceptedCount != 0 || report.InformationalCount != 0 || report.StillOpenCount != 0 || report.EntryCount != 0 {
		t.Fatalf("migrated counts should be zeroed, got %+v", report)
	}
	if report.LastRefreshResult != "legacy_schema_migrated" {
		t.Fatalf("last refresh result = %q", report.LastRefreshResult)
	}

	run2, err := d.InsertRun("repo1", "feature-2", "head2", "base")
	if err != nil {
		t.Fatalf("insert post-migration run: %v", err)
	}
	if err := d.UpsertReviewResolutionReport(ReviewResolutionReport{
		RunID:              run2.ID,
		ReportPath:         "/tmp/current-review-resolution.md",
		Status:             ReviewResolutionStatusFinal,
		ResolvedCount:      1,
		AcceptedCount:      2,
		InformationalCount: 3,
		StillOpenCount:     4,
		ReportVersion:      "1",
		EntryCount:         10,
		SourceWatermark:    "watermark",
		ContentHash:        "hash",
		LastRefreshResult:  "ok",
	}); err != nil {
		t.Fatalf("upsert current report after migration: %v", err)
	}
	current, err := d.GetReviewResolutionReport(run2.ID)
	if err != nil {
		t.Fatalf("get current report after migration: %v", err)
	}
	if current == nil || current.ResolvedCount != 1 || current.AcceptedCount != 2 || current.InformationalCount != 3 || current.StillOpenCount != 4 {
		t.Fatalf("unexpected current report after migration: %+v", current)
	}
}

func TestOpenWaitsForTransientMigrationLock(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	locker, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		t.Fatalf("open locker db: %v", err)
	}
	defer locker.Close()
	if _, err := locker.Exec("BEGIN EXCLUSIVE"); err != nil {
		t.Fatalf("begin exclusive lock: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		d, err := Open(dbPath)
		if err == nil {
			err = d.Close()
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Open returned before the migration lock was released")
		}
		t.Fatalf("Open should wait for a transient migration lock, got: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if _, err := locker.Exec("COMMIT"); err != nil {
		t.Fatalf("commit exclusive lock: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Open after lock release: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Open did not finish after the migration lock was released")
	}
}
