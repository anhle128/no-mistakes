package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

// Run represents a pipeline run.
type Run struct {
	ID                         string
	RepoID                     string
	Branch                     string
	HeadSHA                    string
	BaseSHA                    string
	Status                     types.RunStatus
	PRURL                      *string
	Error                      *string
	WorktreeMode               types.WorktreeMode
	WorkDir                    *string
	WorkDirLabel               *string
	CurrentWorktreeWarning     *string
	MetadataAvailability       types.MetadataAvailability
	EvidenceState              types.EvidenceState
	TerminalReason             *string
	ReviewBaseRef              *string
	ReviewBaseRefreshAttempted bool
	ReviewBaseRefreshError     *string
	RejectionReason            *string
	SkipSteps                  []types.StepName
	Intent                     *string
	IntentSource               *string
	IntentSessionID            *string
	IntentScore                *float64
	CreatedAt                  int64
	UpdatedAt                  int64
}

const runColumns = `id, repo_id, branch, head_sha, base_sha, status, pr_url, error, worktree_mode, work_dir, work_dir_label, current_worktree_warning, metadata_availability, evidence_state, terminal_reason, review_base_ref, review_base_refresh_attempted, review_base_refresh_error, rejection_reason, skip_steps, intent, intent_source, intent_session_id, intent_score, created_at, updated_at`

func scanRun(row interface {
	Scan(...any) error
}, r *Run) error {
	var refreshAttempted int
	var skipStepsJSON *string
	if err := row.Scan(
		&r.ID, &r.RepoID, &r.Branch, &r.HeadSHA, &r.BaseSHA, &r.Status,
		&r.PRURL, &r.Error,
		&r.WorktreeMode, &r.WorkDir, &r.WorkDirLabel, &r.CurrentWorktreeWarning,
		&r.MetadataAvailability, &r.EvidenceState, &r.TerminalReason,
		&r.ReviewBaseRef, &refreshAttempted, &r.ReviewBaseRefreshError, &r.RejectionReason,
		&skipStepsJSON, &r.Intent, &r.IntentSource, &r.IntentSessionID, &r.IntentScore,
		&r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return err
	}
	if skipStepsJSON != nil && *skipStepsJSON != "" {
		if err := json.Unmarshal([]byte(*skipStepsJSON), &r.SkipSteps); err != nil {
			return fmt.Errorf("decode run skip steps: %w", err)
		}
	}
	r.WorktreeMode = types.NormalizeWorktreeMode(r.WorktreeMode)
	r.MetadataAvailability = types.NormalizeMetadataAvailability(r.MetadataAvailability)
	r.EvidenceState = types.NormalizeEvidenceState(r.EvidenceState)
	r.ReviewBaseRefreshAttempted = refreshAttempted != 0
	return nil
}

// RunInsertOptions carries optional run metadata. The zero value preserves the
// historical isolated-worktree run shape.
type RunInsertOptions struct {
	WorktreeMode               types.WorktreeMode
	WorkDir                    string
	WorkDirLabel               string
	CurrentWorktreeWarning     string
	MetadataAvailability       types.MetadataAvailability
	EvidenceState              types.EvidenceState
	ReviewBaseRef              string
	ReviewBaseRefreshAttempted bool
	ReviewBaseRefreshError     string
	RejectionReason            string
	SkipSteps                  []types.StepName
}

// InsertRun creates a new run record.
func (d *DB) InsertRun(repoID, branch, headSHA, baseSHA string) (*Run, error) {
	return d.InsertRunWithOptions(repoID, branch, headSHA, baseSHA, RunInsertOptions{})
}

// InsertRunWithOptions creates a run record with explicit execution metadata.
func (d *DB) InsertRunWithOptions(repoID, branch, headSHA, baseSHA string, opts RunInsertOptions) (*Run, error) {
	ts := now()
	mode := types.NormalizeWorktreeMode(opts.WorktreeMode)
	if mode == "" {
		mode = types.WorktreeModeIsolated
	}
	metadataAvailability := types.NormalizeMetadataAvailability(opts.MetadataAvailability)
	evidenceState := types.NormalizeEvidenceState(opts.EvidenceState)
	workDir := stringPtrOrNil(opts.WorkDir)
	workDirLabel := stringPtrOrNil(opts.WorkDirLabel)
	if workDirLabel == nil {
		label := mode.Label()
		workDirLabel = &label
	}
	warning := stringPtrOrNil(opts.CurrentWorktreeWarning)
	reviewBaseRef := stringPtrOrNil(opts.ReviewBaseRef)
	reviewBaseRefreshError := stringPtrOrNil(opts.ReviewBaseRefreshError)
	rejectionReason := stringPtrOrNil(opts.RejectionReason)
	skipStepsJSON, err := skipStepsJSONString(opts.SkipSteps)
	if err != nil {
		return nil, err
	}
	r := &Run{
		ID:                         newID(),
		RepoID:                     repoID,
		Branch:                     branch,
		HeadSHA:                    headSHA,
		BaseSHA:                    baseSHA,
		Status:                     types.RunPending,
		WorktreeMode:               mode,
		WorkDir:                    workDir,
		WorkDirLabel:               workDirLabel,
		CurrentWorktreeWarning:     warning,
		MetadataAvailability:       metadataAvailability,
		EvidenceState:              evidenceState,
		ReviewBaseRef:              reviewBaseRef,
		ReviewBaseRefreshAttempted: opts.ReviewBaseRefreshAttempted,
		ReviewBaseRefreshError:     reviewBaseRefreshError,
		RejectionReason:            rejectionReason,
		SkipSteps:                  append([]types.StepName(nil), opts.SkipSteps...),
		CreatedAt:                  ts,
		UpdatedAt:                  ts,
	}
	_, err = d.sql.Exec(
		`INSERT INTO runs (id, repo_id, branch, head_sha, base_sha, status, worktree_mode, work_dir, work_dir_label, current_worktree_warning, metadata_availability, evidence_state, review_base_ref, review_base_refresh_attempted, review_base_refresh_error, rejection_reason, skip_steps, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.RepoID, r.Branch, r.HeadSHA, r.BaseSHA, r.Status,
		r.WorktreeMode, r.WorkDir, r.WorkDirLabel, r.CurrentWorktreeWarning,
		r.MetadataAvailability, r.EvidenceState,
		r.ReviewBaseRef, boolToInt(r.ReviewBaseRefreshAttempted), r.ReviewBaseRefreshError, r.RejectionReason,
		skipStepsJSON,
		r.CreatedAt, r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert run: %w", err)
	}
	return r, nil
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func skipStepsJSONString(steps []types.StepName) (*string, error) {
	if len(steps) == 0 {
		return nil, nil
	}
	raw, err := json.Marshal(steps)
	if err != nil {
		return nil, fmt.Errorf("encode run skip steps: %w", err)
	}
	encoded := string(raw)
	return &encoded, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// UpdateRunWorkDir records the checkout where a run is executing. Older run
// records may not have this populated, but repo-local artifacts need it.
func (d *DB) UpdateRunWorkDir(id, workDir string) error {
	_, err := d.sql.Exec(`UPDATE runs SET work_dir = ?, updated_at = ? WHERE id = ?`, workDir, now(), id)
	if err != nil {
		return fmt.Errorf("update run work dir: %w", err)
	}
	return nil
}

// GetRun returns a run by ID.
func (d *DB) GetRun(id string) (*Run, error) {
	r := &Run{}
	err := scanRun(d.sql.QueryRow(`SELECT `+runColumns+` FROM runs WHERE id = ?`, id), r)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	return r, nil
}

// GetRunsByRepo returns all runs for a repo, newest first.
func (d *DB) GetRunsByRepo(repoID string) ([]*Run, error) {
	rows, err := d.sql.Query(`SELECT `+runColumns+` FROM runs WHERE repo_id = ? ORDER BY created_at DESC, id DESC`, repoID)
	if err != nil {
		return nil, fmt.Errorf("get runs by repo: %w", err)
	}
	defer rows.Close()
	var runs []*Run
	for rows.Next() {
		r := &Run{}
		if err := scanRun(rows, r); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// GetActiveRun returns the currently active run (pending or running) for a repo,
// if any. When branch is non-empty, only a run on that exact branch is returned
// - the setup wizard relies on this to decide whether a new run is needed for
// the current branch. When branch is empty, returns the most recently created
// active run across any branch.
func (d *DB) GetActiveRun(repoID, branch string) (*Run, error) {
	r := &Run{}
	var err error
	if branch == "" {
		err = scanRun(d.sql.QueryRow(
			`SELECT `+runColumns+` FROM runs WHERE repo_id = ? AND status IN ('pending', 'running') ORDER BY created_at DESC, id DESC LIMIT 1`, repoID,
		), r)
	} else {
		err = scanRun(d.sql.QueryRow(
			`SELECT `+runColumns+` FROM runs WHERE repo_id = ? AND branch = ? AND status IN ('pending', 'running') ORDER BY created_at DESC, id DESC LIMIT 1`, repoID, branch,
		), r)
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active run: %w", err)
	}
	return r, nil
}

// GetActiveRuns returns all pending or running runs across all repos, newest first.
func (d *DB) GetActiveRuns() ([]*Run, error) {
	rows, err := d.sql.Query(
		`SELECT `+runColumns+` FROM runs WHERE status IN (?, ?) ORDER BY created_at DESC, id DESC`,
		types.RunPending, types.RunRunning,
	)
	if err != nil {
		return nil, fmt.Errorf("get active runs: %w", err)
	}
	defer rows.Close()

	var runs []*Run
	for rows.Next() {
		r := &Run{}
		if err := scanRun(rows, r); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// UpdateRunStatus updates a run's status and updated_at timestamp.
func (d *DB) UpdateRunStatus(id string, status types.RunStatus) error {
	_, err := d.sql.Exec(`UPDATE runs SET status = ?, updated_at = ? WHERE id = ?`, status, now(), id)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	return nil
}

// UpdateRunPRURL sets the PR URL on a run.
func (d *DB) UpdateRunPRURL(id, prURL string) error {
	_, err := d.sql.Exec(`UPDATE runs SET pr_url = ?, updated_at = ? WHERE id = ?`, prURL, now(), id)
	if err != nil {
		return fmt.Errorf("update run pr url: %w", err)
	}
	return nil
}

// UpdateRunHeadSHA updates the run head SHA and timestamp.
func (d *DB) UpdateRunHeadSHA(id, headSHA string) error {
	_, err := d.sql.Exec(`UPDATE runs SET head_sha = ?, updated_at = ? WHERE id = ?`, headSHA, now(), id)
	if err != nil {
		return fmt.Errorf("update run head sha: %w", err)
	}
	return nil
}

// UpdateRunError sets the error message on a run.
func (d *DB) UpdateRunError(id, errMsg string) error {
	return d.UpdateRunErrorStatus(id, errMsg, types.RunFailed)
}

// UpdateRunErrorStatus sets the error message and terminal status on a run.
func (d *DB) UpdateRunErrorStatus(id, errMsg string, status types.RunStatus) error {
	_, err := d.sql.Exec(`UPDATE runs SET error = ?, status = ?, updated_at = ? WHERE id = ?`, errMsg, status, now(), id)
	if err != nil {
		return fmt.Errorf("update run error: %w", err)
	}
	return nil
}

// UpdateRunTerminalReason records a structured terminal reason and evidence
// state without changing the public status.
func (d *DB) UpdateRunTerminalReason(id string, reason string, evidenceState types.EvidenceState) error {
	_, err := d.sql.Exec(
		`UPDATE runs SET terminal_reason = ?, evidence_state = ?, updated_at = ? WHERE id = ?`,
		reason, types.NormalizeEvidenceState(evidenceState), now(), id,
	)
	if err != nil {
		return fmt.Errorf("update run terminal reason: %w", err)
	}
	return nil
}

// RunIntent carries the four intent-related columns persisted on a run.
type RunIntent struct {
	Summary   string
	Source    string
	SessionID string
	Score     float64
}

// UpdateRunIntent persists the inferred user intent for a run.
func (d *DB) UpdateRunIntent(id string, intent RunIntent) error {
	_, err := d.sql.Exec(
		`UPDATE runs SET intent = ?, intent_source = ?, intent_session_id = ?, intent_score = ?, updated_at = ? WHERE id = ?`,
		intent.Summary, intent.Source, intent.SessionID, intent.Score, now(), id,
	)
	if err != nil {
		return fmt.Errorf("update run intent: %w", err)
	}
	return nil
}

// RecoverStaleRuns marks any runs stuck in pending/running status as failed
// and fails any in-progress steps. This is called at daemon startup to clean
// up after a previous crash. Returns the number of recovered runs.
func (d *DB) RecoverStaleRuns(errMsg string) (int, error) {
	ts := now()

	tx, err := d.sql.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Fail stale steps first (running, awaiting_approval, fixing, fix_review).
	_, err = tx.Exec(
		`UPDATE step_results SET status = ?, error = ?, completed_at = ? WHERE status IN (?, ?, ?, ?)`,
		types.StepStatusFailed, errMsg, ts,
		types.StepStatusRunning, types.StepStatusAwaitingApproval, types.StepStatusFixing, types.StepStatusFixReview,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale steps: %w", err)
	}

	// Fail stale runs.
	result, err := tx.Exec(
		`UPDATE runs SET status = ?, error = ?, terminal_reason = ?, evidence_state = ?, updated_at = ? WHERE status IN (?, ?)`,
		types.RunFailed, errMsg, types.RunTerminalReasonDaemonCrashed, types.EvidenceIncomplete, ts,
		types.RunPending, types.RunRunning,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale runs: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}
	return int(count), nil
}
