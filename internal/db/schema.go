package db

const schemaSQL = `
CREATE TABLE IF NOT EXISTS repos (
    id             TEXT PRIMARY KEY,
    working_path   TEXT NOT NULL UNIQUE,
    upstream_url   TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    created_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id                        TEXT PRIMARY KEY,
    repo_id                   TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    branch                    TEXT NOT NULL,
    head_sha                  TEXT NOT NULL,
    base_sha                  TEXT NOT NULL,
    status                    TEXT NOT NULL DEFAULT 'pending',
    pr_url                    TEXT,
    error                     TEXT,
    boundary_status           TEXT NOT NULL DEFAULT 'unknown',
    boundary_reason           TEXT NOT NULL DEFAULT 'unknown',
    boundary_detail           TEXT NOT NULL DEFAULT '',
    boundary_expected_worktree_path TEXT NOT NULL DEFAULT '',
    boundary_actual_worktree_path   TEXT NOT NULL DEFAULT '',
    boundary_git_common_dir         TEXT NOT NULL DEFAULT '',
    boundary_gate_repo_path         TEXT NOT NULL DEFAULT '',
    boundary_fingerprint            TEXT NOT NULL DEFAULT '',
    boundary_verified_at      INTEGER,
    boundary_verifier_version TEXT NOT NULL DEFAULT 'yolo-boundary-v1',
    created_at                INTEGER NOT NULL,
    updated_at                INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS run_events (
    id               TEXT PRIMARY KEY,
    run_id           TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    event_type       TEXT NOT NULL,
    step_name        TEXT,
    action           TEXT,
    gate_id          TEXT,
    gate_fingerprint TEXT,
    status           TEXT,
    requested_mode   TEXT,
    reason           TEXT,
    message          TEXT,
    decision_source  TEXT,
    actor_type       TEXT,
    approval_surface TEXT,
    consent_mode     TEXT,
    created_at       INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS run_events_gate_automation_identity
ON run_events(run_id, event_type, step_name, action, gate_id, gate_fingerprint, decision_source, approval_surface, consent_mode)
WHERE event_type IN ('gate_automation_allowed', 'gate_automation_withheld', 'gate_automation_not_requested');

CREATE TABLE IF NOT EXISTS step_results (
    id            TEXT PRIMARY KEY,
    run_id        TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    step_name     TEXT NOT NULL,
    step_order    INTEGER NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    exit_code     INTEGER,
    duration_ms   INTEGER,
    log_path      TEXT,
    findings_json TEXT,
    error         TEXT,
    started_at    INTEGER,
    completed_at  INTEGER
);

CREATE TABLE IF NOT EXISTS step_rounds (
    id                   TEXT PRIMARY KEY,
    step_result_id       TEXT NOT NULL REFERENCES step_results(id) ON DELETE CASCADE,
    round                INTEGER NOT NULL,
    trigger_type         TEXT NOT NULL,
    findings_json        TEXT,
    user_findings_json   TEXT,
    selected_finding_ids TEXT,
    selection_source     TEXT,
    fix_summary          TEXT,
    duration_ms          INTEGER NOT NULL,
    created_at           INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS intent_cache (
    cache_key   TEXT PRIMARY KEY,
    summary     TEXT NOT NULL,
    agent_name  TEXT NOT NULL,
    session_id  TEXT NOT NULL,
    created_at  INTEGER NOT NULL
);
`

// migrationStatements hold additive schema changes applied to databases that
// were created before the referenced columns existed. Each statement must be
// idempotent via its error being tolerated when the column already exists.
var migrationStatements = []string{
	`ALTER TABLE step_rounds ADD COLUMN selected_finding_ids TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN selection_source TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN fix_summary TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN user_findings_json TEXT`,
	`ALTER TABLE runs ADD COLUMN intent TEXT`,
	`ALTER TABLE runs ADD COLUMN intent_source TEXT`,
	`ALTER TABLE runs ADD COLUMN intent_session_id TEXT`,
	`ALTER TABLE runs ADD COLUMN intent_score REAL`,
	`ALTER TABLE runs ADD COLUMN boundary_status TEXT NOT NULL DEFAULT 'unknown'`,
	`ALTER TABLE runs ADD COLUMN boundary_reason TEXT NOT NULL DEFAULT 'unknown'`,
	`ALTER TABLE runs ADD COLUMN boundary_detail TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE runs ADD COLUMN boundary_expected_worktree_path TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE runs ADD COLUMN boundary_actual_worktree_path TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE runs ADD COLUMN boundary_git_common_dir TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE runs ADD COLUMN boundary_gate_repo_path TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE runs ADD COLUMN boundary_fingerprint TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE runs ADD COLUMN boundary_verified_at INTEGER`,
	`ALTER TABLE runs ADD COLUMN boundary_verifier_version TEXT NOT NULL DEFAULT 'yolo-boundary-v1'`,
	`CREATE TABLE IF NOT EXISTS run_events (
		id               TEXT PRIMARY KEY,
		run_id           TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
		event_type       TEXT NOT NULL,
		step_name        TEXT,
		action           TEXT,
		gate_id          TEXT,
		gate_fingerprint TEXT,
		status           TEXT,
		requested_mode   TEXT,
		reason           TEXT,
		message          TEXT,
		decision_source  TEXT,
		actor_type       TEXT,
		approval_surface TEXT,
		consent_mode     TEXT,
		created_at       INTEGER NOT NULL
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS run_events_gate_automation_identity
		ON run_events(run_id, event_type, step_name, action, gate_id, gate_fingerprint, decision_source, approval_surface, consent_mode)
		WHERE event_type IN ('gate_automation_allowed', 'gate_automation_withheld', 'gate_automation_not_requested')`,
}
