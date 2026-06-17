package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

const (
	RunEventBoundaryRefreshed          = "boundary_refreshed"
	RunEventGateAutomationAllowed      = "gate_automation_allowed"
	RunEventGateAutomationWithheld     = "gate_automation_withheld"
	RunEventGateAutomationNotRequested = "gate_automation_not_requested"
	RunEventManualDecision             = "manual_decision"
)

// RunEvent is an append-only audit record for boundary checks and gate
// automation decisions.
type RunEvent struct {
	ID              string
	RunID           string
	EventType       string
	StepName        *types.StepName
	Action          *types.ApprovalAction
	GateID          string
	GateFingerprint string
	Status          types.GateAutomationStatus
	RequestedMode   types.ConsentMode
	Reason          string
	Message         string
	DecisionSource  types.DecisionSource
	ActorType       types.ActorType
	ApprovalSurface types.ApprovalSurface
	ConsentMode     types.ConsentMode
	CreatedAt       int64
}

// InsertRunEvent inserts a run event. Gate automation events are idempotent by
// gate identity; other audit events are append-only.
func (d *DB) InsertRunEvent(event RunEvent) (*RunEvent, error) {
	if event.ID == "" {
		event.ID = newID()
	}
	if event.CreatedAt == 0 {
		event.CreatedAt = now()
	}
	err := d.insertRunEvent(event)
	if err != nil && !isGateAutomationEvent(event.EventType) && isUniqueConstraintErr(err) {
		event.GateFingerprint = appendOnlyCompatibilityFingerprint(event)
		err = d.insertRunEvent(event)
	}
	if err != nil {
		return nil, fmt.Errorf("insert run event: %w", err)
	}
	return &event, nil
}

func (d *DB) insertRunEvent(event RunEvent) error {
	insertClause := "INSERT"
	if isGateAutomationEvent(event.EventType) {
		insertClause = "INSERT OR IGNORE"
	}
	_, err := d.sql.Exec(
		insertClause+` INTO run_events (
			id, run_id, event_type, step_name, action, gate_id, gate_fingerprint,
			status, requested_mode, reason, message, decision_source, actor_type,
			approval_surface, consent_mode, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.RunID, event.EventType, stepNameValue(event.StepName),
		actionValue(event.Action), event.GateID, event.GateFingerprint,
		string(event.Status), string(event.RequestedMode), event.Reason, event.Message,
		string(event.DecisionSource), string(event.ActorType), string(event.ApprovalSurface),
		string(event.ConsentMode), event.CreatedAt,
	)
	return err
}

func isGateAutomationEvent(eventType string) bool {
	switch eventType {
	case RunEventGateAutomationAllowed, RunEventGateAutomationWithheld, RunEventGateAutomationNotRequested:
		return true
	default:
		return false
	}
}

func isUniqueConstraintErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

func appendOnlyCompatibilityFingerprint(event RunEvent) string {
	if strings.TrimSpace(event.GateFingerprint) == "" {
		return "event:" + event.ID
	}
	return event.GateFingerprint + "#event:" + event.ID
}

func stepNameValue(v *types.StepName) any {
	if v == nil {
		return ""
	}
	return string(*v)
}

func actionValue(v *types.ApprovalAction) any {
	if v == nil {
		return ""
	}
	return string(*v)
}

// GetRunEvents returns audit events for a run in insertion order.
func (d *DB) GetRunEvents(runID string) ([]*RunEvent, error) {
	rows, err := d.sql.Query(
		`SELECT id, run_id, event_type, step_name, action, gate_id, gate_fingerprint,
			status, requested_mode, reason, message, decision_source, actor_type,
			approval_surface, consent_mode, created_at
		FROM run_events WHERE run_id = ? ORDER BY created_at ASC, id ASC`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("get run events: %w", err)
	}
	defer rows.Close()

	var events []*RunEvent
	for rows.Next() {
		event, err := scanRunEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// GetGateAutomationEvent returns the latest automation audit event for the
// current gate identity, or nil when the gate has not requested automation.
func (d *DB) GetGateAutomationEvent(runID, gateID, fingerprint string) (*RunEvent, error) {
	row := d.sql.QueryRow(
		`SELECT id, run_id, event_type, step_name, action, gate_id, gate_fingerprint,
			status, requested_mode, reason, message, decision_source, actor_type,
			approval_surface, consent_mode, created_at
		FROM run_events
		WHERE run_id = ? AND gate_id = ? AND gate_fingerprint = ?
			AND event_type IN (?, ?, ?)
		ORDER BY created_at DESC,
			CASE WHEN event_type = ? THEN 1 ELSE 0 END ASC,
			id DESC LIMIT 1`,
		runID, gateID, fingerprint,
		RunEventGateAutomationAllowed, RunEventGateAutomationWithheld, RunEventGateAutomationNotRequested,
		RunEventGateAutomationNotRequested,
	)
	event, err := scanRunEvent(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return event, nil
}

type runEventScanner interface {
	Scan(...any) error
}

func scanRunEvent(row runEventScanner) (*RunEvent, error) {
	var event RunEvent
	var stepName, action sql.NullString
	var status, requestedMode, decisionSource, actorType, approvalSurface, consentMode sql.NullString
	err := row.Scan(
		&event.ID, &event.RunID, &event.EventType, &stepName, &action,
		&event.GateID, &event.GateFingerprint, &status, &requestedMode,
		&event.Reason, &event.Message, &decisionSource, &actorType,
		&approvalSurface, &consentMode, &event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if stepName.Valid && stepName.String != "" {
		v := types.StepName(stepName.String)
		event.StepName = &v
	}
	if action.Valid && action.String != "" {
		v := types.ApprovalAction(action.String)
		event.Action = &v
	}
	event.Status = types.GateAutomationStatus(status.String)
	event.RequestedMode = types.ConsentMode(requestedMode.String)
	event.DecisionSource = types.DecisionSource(decisionSource.String)
	event.ActorType = types.ActorType(actorType.String)
	event.ApprovalSurface = types.ApprovalSurface(approvalSurface.String)
	event.ConsentMode = types.ConsentMode(consentMode.String)
	return &event, nil
}
