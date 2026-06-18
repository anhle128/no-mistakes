package db

import (
	"database/sql"
	"fmt"
)

const (
	ReviewResolutionStatusInProgress          = "in_progress"
	ReviewResolutionStatusFinal               = "final"
	ReviewResolutionStatusIncomplete          = "incomplete"
	ReviewResolutionStatusStale               = "stale"
	ReviewResolutionStatusDegraded            = "degraded"
	ReviewResolutionStatusEvidenceUnavailable = "evidence_unavailable"

	ReviewResolutionDecisionFix          = "fix"
	ReviewResolutionDecisionApprove      = "approve"
	ReviewResolutionDecisionSkip         = "skip"
	ReviewResolutionDecisionAbort        = "abort"
	ReviewResolutionDecisionPolicyAccept = "policy_accept"
	ReviewResolutionDecisionNoOp         = "no_op"
)

// ReviewResolutionReport is the compact per-run metadata row that local
// surfaces and PR summaries use instead of reparsing Markdown.
type ReviewResolutionReport struct {
	RunID              string
	ReportPath         string
	Status             string
	ResolvedCount      int
	AcceptedCount      int
	InformationalCount int
	StillOpenCount     int
	ReportVersion      string
	EntryCount         int
	SourceRoundStart   *int
	SourceRoundEnd     *int
	SourceWatermark    string
	ContentHash        string
	LastRefreshResult  string
	FirstGeneratedAt   int64
	LastRefreshedAt    int64
	FinalizedAt        *int64
	CreatedAt          int64
	UpdatedAt          int64
}

// UpsertReviewResolutionReport inserts or refreshes report metadata. The first
// generated timestamp is preserved across refreshes.
func (d *DB) UpsertReviewResolutionReport(r ReviewResolutionReport) error {
	ts := now()
	if r.CreatedAt == 0 {
		r.CreatedAt = ts
	}
	if r.UpdatedAt == 0 {
		r.UpdatedAt = ts
	}
	if r.FirstGeneratedAt == 0 {
		r.FirstGeneratedAt = ts
	}
	if r.LastRefreshedAt == 0 {
		r.LastRefreshedAt = ts
	}

	tx, err := d.sql.Begin()
	if err != nil {
		return fmt.Errorf("begin review resolution report upsert: %w", err)
	}
	defer tx.Rollback()

	var firstGeneratedAt int64
	err = tx.QueryRow(`SELECT first_generated_at, created_at FROM review_resolution_reports WHERE run_id = ?`, r.RunID).Scan(&firstGeneratedAt, &r.CreatedAt)
	if err == nil {
		r.FirstGeneratedAt = firstGeneratedAt
		r.UpdatedAt = ts
	} else if err != sql.ErrNoRows {
		return fmt.Errorf("read existing review resolution report: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO review_resolution_reports (
			run_id, report_path, status, resolved_count, accepted_count,
			informational_count, still_open_count, report_version, entry_count,
			source_round_start, source_round_end, source_watermark, content_hash,
			last_refresh_result, first_generated_at, last_refreshed_at,
			finalized_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(run_id) DO UPDATE SET
			report_path = excluded.report_path,
			status = excluded.status,
			resolved_count = excluded.resolved_count,
			accepted_count = excluded.accepted_count,
			informational_count = excluded.informational_count,
			still_open_count = excluded.still_open_count,
			report_version = excluded.report_version,
			entry_count = excluded.entry_count,
			source_round_start = excluded.source_round_start,
			source_round_end = excluded.source_round_end,
			source_watermark = excluded.source_watermark,
			content_hash = excluded.content_hash,
			last_refresh_result = excluded.last_refresh_result,
			last_refreshed_at = excluded.last_refreshed_at,
			finalized_at = excluded.finalized_at,
			updated_at = excluded.updated_at
	`,
		r.RunID, r.ReportPath, r.Status, r.ResolvedCount, r.AcceptedCount,
		r.InformationalCount, r.StillOpenCount, r.ReportVersion, r.EntryCount,
		r.SourceRoundStart, r.SourceRoundEnd, r.SourceWatermark, r.ContentHash,
		r.LastRefreshResult, r.FirstGeneratedAt, r.LastRefreshedAt,
		r.FinalizedAt, r.CreatedAt, r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert review resolution report: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit review resolution report upsert: %w", err)
	}
	return nil
}

func (d *DB) GetReviewResolutionReport(runID string) (*ReviewResolutionReport, error) {
	r := &ReviewResolutionReport{}
	err := d.sql.QueryRow(`
		SELECT run_id, report_path, status, resolved_count, accepted_count,
			informational_count, still_open_count, report_version, entry_count,
			source_round_start, source_round_end, source_watermark, content_hash,
			last_refresh_result, first_generated_at, last_refreshed_at,
			finalized_at, created_at, updated_at
		FROM review_resolution_reports WHERE run_id = ?
	`, runID).Scan(
		&r.RunID, &r.ReportPath, &r.Status, &r.ResolvedCount, &r.AcceptedCount,
		&r.InformationalCount, &r.StillOpenCount, &r.ReportVersion, &r.EntryCount,
		&r.SourceRoundStart, &r.SourceRoundEnd, &r.SourceWatermark, &r.ContentHash,
		&r.LastRefreshResult, &r.FirstGeneratedAt, &r.LastRefreshedAt,
		&r.FinalizedAt, &r.CreatedAt, &r.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get review resolution report: %w", err)
	}
	return r, nil
}

func (d *DB) DeleteReviewResolutionReport(runID string) error {
	if _, err := d.sql.Exec(`DELETE FROM review_resolution_reports WHERE run_id = ?`, runID); err != nil {
		return fmt.Errorf("delete review resolution report: %w", err)
	}
	return nil
}

// ReviewResolutionDecision records provenance for terminal Review decisions.
type ReviewResolutionDecision struct {
	ID           string
	RunID        string
	StepResultID string
	RoundID      *string
	FindingID    string
	Action       string
	ActorSource  string
	Reason       *string
	CreatedAt    int64
}

func (d *DB) InsertReviewResolutionDecision(decision ReviewResolutionDecision) (*ReviewResolutionDecision, error) {
	if decision.ID == "" {
		decision.ID = newID()
	}
	if decision.CreatedAt == 0 {
		decision.CreatedAt = now()
	}
	_, err := d.sql.Exec(`
		INSERT INTO review_resolution_decisions (
			id, run_id, step_result_id, round_id, finding_id, action,
			actor_source, reason, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, decision.ID, decision.RunID, decision.StepResultID, decision.RoundID, decision.FindingID, decision.Action, decision.ActorSource, decision.Reason, decision.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert review resolution decision: %w", err)
	}
	return &decision, nil
}

func (d *DB) GetReviewResolutionDecisions(runID string) ([]*ReviewResolutionDecision, error) {
	rows, err := d.sql.Query(`
		SELECT id, run_id, step_result_id, round_id, finding_id, action,
			actor_source, reason, created_at
		FROM review_resolution_decisions
		WHERE run_id = ?
		ORDER BY created_at, id
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("get review resolution decisions: %w", err)
	}
	defer rows.Close()

	var out []*ReviewResolutionDecision
	for rows.Next() {
		d := &ReviewResolutionDecision{}
		if err := rows.Scan(&d.ID, &d.RunID, &d.StepResultID, &d.RoundID, &d.FindingID, &d.Action, &d.ActorSource, &d.Reason, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review resolution decision: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
