package db

const schemaSQL = `
CREATE TABLE IF NOT EXISTS repos (
    id             TEXT PRIMARY KEY,
    working_path   TEXT NOT NULL UNIQUE,
    upstream_url   TEXT NOT NULL,
    fork_url       TEXT,
    default_branch TEXT NOT NULL DEFAULT 'main',
    created_at     INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS runs (
    id                            TEXT PRIMARY KEY,
    repo_id                       TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    branch                        TEXT NOT NULL,
    head_sha                      TEXT NOT NULL,
    base_sha                      TEXT NOT NULL,
    status                        TEXT NOT NULL DEFAULT 'pending',
    pr_url                        TEXT,
    error                         TEXT,
    worktree_mode                 TEXT NOT NULL DEFAULT 'isolated',
    work_dir                      TEXT,
    work_dir_label                TEXT,
    current_worktree_warning      TEXT,
    metadata_availability         TEXT NOT NULL DEFAULT 'available',
    evidence_state                TEXT NOT NULL DEFAULT 'complete',
    terminal_reason               TEXT,
    review_base_ref               TEXT,
    review_base_refresh_attempted INTEGER NOT NULL DEFAULT 0,
    review_base_refresh_error     TEXT,
    rejection_reason              TEXT,
    skip_steps                    TEXT,
    created_at                    INTEGER NOT NULL,
    updated_at                    INTEGER NOT NULL
);

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
    fix_commit_sha       TEXT,
    no_commit_reason     TEXT,
    fix_resolution_details_json TEXT,
    duration_ms          INTEGER NOT NULL,
    created_at           INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS review_resolution_reports (
    run_id              TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
    report_path         TEXT NOT NULL,
    status              TEXT NOT NULL,
    resolved_count      INTEGER NOT NULL,
    accepted_count      INTEGER NOT NULL,
    informational_count INTEGER NOT NULL,
    still_open_count    INTEGER NOT NULL,
    report_version      TEXT NOT NULL,
    entry_count         INTEGER NOT NULL,
    source_round_start  INTEGER,
    source_round_end    INTEGER,
    source_watermark    TEXT NOT NULL,
    content_hash        TEXT NOT NULL,
    last_refresh_result TEXT NOT NULL,
    first_generated_at  INTEGER NOT NULL,
    last_refreshed_at   INTEGER NOT NULL,
    finalized_at        INTEGER,
    created_at          INTEGER NOT NULL,
    updated_at          INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS review_resolution_decisions (
    id             TEXT PRIMARY KEY,
    run_id         TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    step_result_id TEXT NOT NULL REFERENCES step_results(id) ON DELETE CASCADE,
    round_id       TEXT REFERENCES step_rounds(id) ON DELETE SET NULL,
    finding_id     TEXT NOT NULL,
    action         TEXT NOT NULL,
    actor_source   TEXT NOT NULL,
    reason         TEXT,
    created_at     INTEGER NOT NULL
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
	`ALTER TABLE repos ADD COLUMN fork_url TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN selected_finding_ids TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN selection_source TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN fix_summary TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN user_findings_json TEXT`,
	`ALTER TABLE runs ADD COLUMN intent TEXT`,
	`ALTER TABLE runs ADD COLUMN intent_source TEXT`,
	`ALTER TABLE runs ADD COLUMN intent_session_id TEXT`,
	`ALTER TABLE runs ADD COLUMN intent_score REAL`,
	`ALTER TABLE step_rounds ADD COLUMN fix_commit_sha TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN no_commit_reason TEXT`,
	`ALTER TABLE step_rounds ADD COLUMN fix_resolution_details_json TEXT`,
	`CREATE TABLE IF NOT EXISTS review_resolution_reports (
		run_id              TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
		report_path         TEXT NOT NULL,
		status              TEXT NOT NULL,
		resolved_count      INTEGER NOT NULL,
		accepted_count      INTEGER NOT NULL,
		informational_count INTEGER NOT NULL,
		still_open_count    INTEGER NOT NULL,
		report_version      TEXT NOT NULL,
		entry_count         INTEGER NOT NULL,
		source_round_start  INTEGER,
		source_round_end    INTEGER,
		source_watermark    TEXT NOT NULL,
		content_hash        TEXT NOT NULL,
		last_refresh_result TEXT NOT NULL,
		first_generated_at  INTEGER NOT NULL,
		last_refreshed_at   INTEGER NOT NULL,
		finalized_at        INTEGER,
		created_at          INTEGER NOT NULL,
		updated_at          INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS review_resolution_decisions (
		id             TEXT PRIMARY KEY,
		run_id         TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
		step_result_id TEXT NOT NULL REFERENCES step_results(id) ON DELETE CASCADE,
		round_id       TEXT REFERENCES step_rounds(id) ON DELETE SET NULL,
		finding_id     TEXT NOT NULL,
		action         TEXT NOT NULL,
		actor_source   TEXT NOT NULL,
		reason         TEXT,
		created_at     INTEGER NOT NULL
	)`,
	`ALTER TABLE runs ADD COLUMN worktree_mode TEXT NOT NULL DEFAULT 'isolated'`,
	`ALTER TABLE runs ADD COLUMN work_dir TEXT`,
	`ALTER TABLE runs ADD COLUMN work_dir_label TEXT`,
	`ALTER TABLE runs ADD COLUMN current_worktree_warning TEXT`,
	`ALTER TABLE runs ADD COLUMN metadata_availability TEXT NOT NULL DEFAULT 'available'`,
	`ALTER TABLE runs ADD COLUMN evidence_state TEXT NOT NULL DEFAULT 'complete'`,
	`ALTER TABLE runs ADD COLUMN terminal_reason TEXT`,
	`ALTER TABLE runs ADD COLUMN review_base_ref TEXT`,
	`ALTER TABLE runs ADD COLUMN review_base_refresh_attempted INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE runs ADD COLUMN review_base_refresh_error TEXT`,
	`ALTER TABLE runs ADD COLUMN rejection_reason TEXT`,
	`ALTER TABLE runs ADD COLUMN skip_steps TEXT`,
}

const createReviewResolutionReportsSQL = `
CREATE TABLE review_resolution_reports (
	run_id              TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
	report_path         TEXT NOT NULL,
	status              TEXT NOT NULL,
	resolved_count      INTEGER NOT NULL,
	accepted_count      INTEGER NOT NULL,
	informational_count INTEGER NOT NULL,
	still_open_count    INTEGER NOT NULL,
	report_version      TEXT NOT NULL,
	entry_count         INTEGER NOT NULL,
	source_round_start  INTEGER,
	source_round_end    INTEGER,
	source_watermark    TEXT NOT NULL,
	content_hash        TEXT NOT NULL,
	last_refresh_result TEXT NOT NULL,
	first_generated_at  INTEGER NOT NULL,
	last_refreshed_at   INTEGER NOT NULL,
	finalized_at        INTEGER,
	created_at          INTEGER NOT NULL,
	updated_at          INTEGER NOT NULL
)`

var reviewResolutionReportCurrentColumns = []string{
	"run_id",
	"report_path",
	"status",
	"resolved_count",
	"accepted_count",
	"informational_count",
	"still_open_count",
	"report_version",
	"entry_count",
	"source_round_start",
	"source_round_end",
	"source_watermark",
	"content_hash",
	"last_refresh_result",
	"first_generated_at",
	"last_refreshed_at",
	"finalized_at",
	"created_at",
	"updated_at",
}

var reviewResolutionReportLegacyColumns = []string{
	"contract_version",
	"latest_outcome",
	"summary_counts_json",
	"generation_mode",
	"source_snapshot_at",
	"source_step_result_id",
	"source_round_ids_json",
	"latest_review_round_id",
	"latest_fix_round_id",
	"generated_at",
	"stale",
	"safe_error",
}
