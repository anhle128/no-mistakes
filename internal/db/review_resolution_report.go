package db

import (
	"database/sql"
	"fmt"
)

// ReviewResolutionReportMetadata is the persisted run-scoped report reference
// and compact summary projected to IPC surfaces.
type ReviewResolutionReportMetadata struct {
	RunID               string
	ReportPath          *string
	Status              string
	ContractVersion     string
	LatestOutcome       string
	SummaryCountsJSON   string
	GenerationMode      string
	SourceSnapshotAt    int64
	SourceStepResultID  *string
	SourceRoundIDsJSON  string
	LatestReviewRoundID *string
	LatestFixRoundID    *string
	GeneratedAt         *int64
	UpdatedAt           int64
	Stale               bool
	SafeError           *string
}

const reviewResolutionReportColumns = `run_id, report_path, status, contract_version, latest_outcome, summary_counts_json, generation_mode, source_snapshot_at, source_step_result_id, source_round_ids_json, latest_review_round_id, latest_fix_round_id, generated_at, updated_at, stale, safe_error`

func scanReviewResolutionReport(row interface {
	Scan(...any) error
}, m *ReviewResolutionReportMetadata) error {
	var stale int
	if err := row.Scan(
		&m.RunID,
		&m.ReportPath,
		&m.Status,
		&m.ContractVersion,
		&m.LatestOutcome,
		&m.SummaryCountsJSON,
		&m.GenerationMode,
		&m.SourceSnapshotAt,
		&m.SourceStepResultID,
		&m.SourceRoundIDsJSON,
		&m.LatestReviewRoundID,
		&m.LatestFixRoundID,
		&m.GeneratedAt,
		&m.UpdatedAt,
		&stale,
		&m.SafeError,
	); err != nil {
		return err
	}
	m.Stale = stale != 0
	return nil
}

// UpsertReviewResolutionReportMetadata stores the current metadata row for a run.
func (d *DB) UpsertReviewResolutionReportMetadata(m ReviewResolutionReportMetadata) error {
	stale := 0
	if m.Stale {
		stale = 1
	}
	_, err := d.sql.Exec(
		`INSERT INTO review_resolution_reports (
			run_id, report_path, status, contract_version, latest_outcome, summary_counts_json,
			generation_mode, source_snapshot_at, source_step_result_id, source_round_ids_json,
			latest_review_round_id, latest_fix_round_id, generated_at, updated_at, stale, safe_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			report_path = excluded.report_path,
			status = excluded.status,
			contract_version = excluded.contract_version,
			latest_outcome = excluded.latest_outcome,
			summary_counts_json = excluded.summary_counts_json,
			generation_mode = excluded.generation_mode,
			source_snapshot_at = excluded.source_snapshot_at,
			source_step_result_id = excluded.source_step_result_id,
			source_round_ids_json = excluded.source_round_ids_json,
			latest_review_round_id = excluded.latest_review_round_id,
			latest_fix_round_id = excluded.latest_fix_round_id,
			generated_at = excluded.generated_at,
			updated_at = excluded.updated_at,
			stale = excluded.stale,
			safe_error = excluded.safe_error`,
		m.RunID,
		m.ReportPath,
		m.Status,
		m.ContractVersion,
		m.LatestOutcome,
		m.SummaryCountsJSON,
		m.GenerationMode,
		m.SourceSnapshotAt,
		m.SourceStepResultID,
		m.SourceRoundIDsJSON,
		m.LatestReviewRoundID,
		m.LatestFixRoundID,
		m.GeneratedAt,
		m.UpdatedAt,
		stale,
		m.SafeError,
	)
	if err != nil {
		return fmt.Errorf("upsert review resolution report metadata: %w", err)
	}
	return nil
}

// GetReviewResolutionReportMetadata returns the report metadata for a run.
func (d *DB) GetReviewResolutionReportMetadata(runID string) (*ReviewResolutionReportMetadata, error) {
	m := &ReviewResolutionReportMetadata{}
	err := scanReviewResolutionReport(
		d.sql.QueryRow(`SELECT `+reviewResolutionReportColumns+` FROM review_resolution_reports WHERE run_id = ?`, runID),
		m,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get review resolution report metadata: %w", err)
	}
	return m, nil
}
