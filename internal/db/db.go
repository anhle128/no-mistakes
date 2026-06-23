package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

// DB wraps a SQLite database connection.
type DB struct {
	sql *sql.DB
}

// Open opens (or creates) the SQLite database at path and runs migrations.
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	for _, stmt := range migrationStatements {
		if _, err := sqlDB.Exec(stmt); err != nil && !isDuplicateColumnErr(err) {
			sqlDB.Close()
			return nil, fmt.Errorf("migrate db: %w", err)
		}
	}
	if err := runStructuredMigrations(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return &DB{sql: sqlDB}, nil
}

// isDuplicateColumnErr reports whether err is SQLite's "duplicate column name"
// error, which ALTER TABLE ADD COLUMN emits when the column already exists.
// Treating this as a no-op keeps migrations idempotent without a version table.
func isDuplicateColumnErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate column name")
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.sql.Close()
}

// newID generates a new ULID with monotonic ordering.
func newID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), entropy).String()
}

// now returns the current unix timestamp in seconds.
func now() int64 {
	return time.Now().Unix()
}

func runStructuredMigrations(sqlDB *sql.DB) error {
	if err := migrateLegacyReviewResolutionReports(sqlDB); err != nil {
		return err
	}
	return nil
}

func migrateLegacyReviewResolutionReports(sqlDB *sql.DB) error {
	columns, err := tableColumns(sqlDB, "review_resolution_reports")
	if err != nil {
		return fmt.Errorf("inspect review_resolution_reports schema: %w", err)
	}
	if len(columns) == 0 || reviewResolutionReportsSchemaCurrent(columns) {
		return nil
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("begin review_resolution_reports migration: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DROP TABLE IF EXISTS review_resolution_reports_legacy_migrating`); err != nil {
		return fmt.Errorf("remove stale review_resolution_reports migration table: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE review_resolution_reports RENAME TO review_resolution_reports_legacy_migrating`); err != nil {
		return fmt.Errorf("rename legacy review_resolution_reports: %w", err)
	}
	if _, err := tx.Exec(createReviewResolutionReportsSQL); err != nil {
		return fmt.Errorf("create migrated review_resolution_reports: %w", err)
	}

	if columns["run_id"] {
		ts := now()
		reportPathExpr := "''"
		if columns["report_path"] {
			reportPathExpr = "COALESCE(l.report_path, '')"
		}
		firstGeneratedExpr := legacyReportTimestampExpr(columns, ts, "generated_at", "source_snapshot_at", "updated_at")
		lastRefreshedExpr := legacyReportTimestampExpr(columns, ts, "updated_at", "generated_at", "source_snapshot_at")
		createdExpr := legacyReportTimestampExpr(columns, ts, "generated_at", "source_snapshot_at", "updated_at")
		updatedExpr := legacyReportTimestampExpr(columns, ts, "updated_at", "generated_at", "source_snapshot_at")

		_, err = tx.Exec(fmt.Sprintf(`
			INSERT INTO review_resolution_reports (
				run_id, report_path, status, resolved_count, accepted_count,
				informational_count, still_open_count, report_version, entry_count,
				source_round_start, source_round_end, source_watermark, content_hash,
				last_refresh_result, first_generated_at, last_refreshed_at,
				finalized_at, created_at, updated_at
			)
			SELECT
				l.run_id, %s, ?, 0, 0,
				0, 0, ?, 0,
				NULL, NULL, ?, ?,
				?, %s, %s,
				NULL, %s, %s
			FROM review_resolution_reports_legacy_migrating l
			WHERE l.run_id IS NOT NULL
				AND l.run_id <> ''
				AND EXISTS (SELECT 1 FROM runs r WHERE r.id = l.run_id)
		`, reportPathExpr, firstGeneratedExpr, lastRefreshedExpr, createdExpr, updatedExpr),
			ReviewResolutionStatusEvidenceUnavailable,
			"1",
			"legacy-schema-migration",
			"",
			"legacy_schema_migrated",
		)
		if err != nil {
			return fmt.Errorf("copy legacy review_resolution_reports rows: %w", err)
		}
	}

	if _, err := tx.Exec(`DROP TABLE review_resolution_reports_legacy_migrating`); err != nil {
		return fmt.Errorf("drop legacy review_resolution_reports: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit review_resolution_reports migration: %w", err)
	}
	return nil
}

func tableColumns(sqlDB *sql.DB, table string) (map[string]bool, error) {
	rows, err := sqlDB.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = true
	}
	return columns, rows.Err()
}

func reviewResolutionReportsSchemaCurrent(columns map[string]bool) bool {
	for _, name := range reviewResolutionReportCurrentColumns {
		if !columns[name] {
			return false
		}
	}
	for _, name := range reviewResolutionReportLegacyColumns {
		if columns[name] {
			return false
		}
	}
	return true
}

func legacyReportTimestampExpr(columns map[string]bool, fallback int64, names ...string) string {
	for _, name := range names {
		if columns[name] {
			return fmt.Sprintf("COALESCE(l.%s, %d)", name, fallback)
		}
	}
	return fmt.Sprintf("%d", fallback)
}
