package db

import (
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestRunBoundaryRoundTrip(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc123", "def456")
	if err != nil {
		t.Fatal(err)
	}

	verifiedAt := int64(1234)
	boundary := types.ExecutionBoundary{
		Status:               types.BoundarySafe,
		Reason:               types.BoundaryReasonVerifiedRunWorktree,
		Detail:               "ok",
		ExpectedWorktreePath: "/managed/worktrees/repo/run",
		ActualWorktreePath:   "/managed/worktrees/repo/run",
		GitCommonDir:         "/managed/repos/repo.git",
		GateRepoPath:         "/managed/repos/repo.git",
		Fingerprint:          "proof-fingerprint",
		VerifiedAt:           verifiedAt,
		VerifierVersion:      "test-verifier",
	}
	if err := d.UpdateRunBoundary(run.ID, boundary); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.BoundaryStatus != boundary.Status || got.BoundaryReason != boundary.Reason || got.BoundaryDetail != boundary.Detail {
		t.Fatalf("boundary = %+v, want %+v", got.Boundary(), boundary)
	}
	if got.BoundaryVerifiedAt == nil || *got.BoundaryVerifiedAt != verifiedAt {
		t.Fatalf("BoundaryVerifiedAt = %v, want %d", got.BoundaryVerifiedAt, verifiedAt)
	}
	if got.BoundaryVerifierVersion != "test-verifier" {
		t.Fatalf("BoundaryVerifierVersion = %q", got.BoundaryVerifierVersion)
	}
	if got.BoundaryExpectedWorktreePath != boundary.ExpectedWorktreePath ||
		got.BoundaryActualWorktreePath != boundary.ActualWorktreePath ||
		got.BoundaryGitCommonDir != boundary.GitCommonDir ||
		got.BoundaryGateRepoPath != boundary.GateRepoPath ||
		got.BoundaryFingerprint != boundary.Fingerprint {
		t.Fatalf("boundary audit fields = %+v, want %+v", got.Boundary(), boundary)
	}
}

func TestRunEventsRoundTripAndAutomationLookup(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc123", "def456")
	if err != nil {
		t.Fatal(err)
	}
	step := types.StepReview
	action := types.ActionFix

	event := RunEvent{
		RunID:           run.ID,
		EventType:       RunEventGateAutomationWithheld,
		StepName:        &step,
		Action:          &action,
		GateID:          "review",
		GateFingerprint: "fp",
		Status:          types.GateAutomationWithheld,
		RequestedMode:   types.ConsentModeYes,
		Reason:          "unknown",
		Message:         "withheld",
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorAgent,
		ApprovalSurface: types.ApprovalSurfaceAXI,
		ConsentMode:     types.ConsentModeYes,
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}

	events, err := d.GetRunEvents(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want idempotent 1", len(events))
	}
	got, err := d.GetGateAutomationEvent(run.ID, "review", "fp")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected automation event")
	}
	if got.Status != types.GateAutomationWithheld || got.RequestedMode != types.ConsentModeYes {
		t.Fatalf("event = %+v", got)
	}
}

func TestRunEventsDeduplicateNilActionGateEvents(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc123", "def456")
	if err != nil {
		t.Fatal(err)
	}
	step := types.StepReview
	event := RunEvent{
		RunID:           run.ID,
		EventType:       RunEventGateAutomationWithheld,
		StepName:        &step,
		GateID:          "review",
		GateFingerprint: "fp",
		Status:          types.GateAutomationWithheld,
		RequestedMode:   types.ConsentModeAgentUnattended,
		Reason:          "unknown",
		Message:         "withheld",
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorSystem,
		ApprovalSurface: types.ApprovalSurfaceDaemon,
		ConsentMode:     types.ConsentModeAgentUnattended,
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	events, err := d.GetRunEvents(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want idempotent 1", len(events))
	}
	if events[0].Action != nil {
		t.Fatalf("Action = %v, want nil after sentinel scan", events[0].Action)
	}
}

func TestBoundaryRefreshEventsAreAppendOnly(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc123", "def456")
	if err != nil {
		t.Fatal(err)
	}
	step := types.StepReview
	event := RunEvent{
		RunID:     run.ID,
		EventType: RunEventBoundaryRefreshed,
		StepName:  &step,
		Reason:    string(types.BoundaryReasonVerifiedRunWorktree),
		Message:   "safe",
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	events, err := d.GetRunEvents(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want append-only 2", len(events))
	}
	for _, got := range events {
		if got.EventType != RunEventBoundaryRefreshed {
			t.Fatalf("event type = %q, want %q", got.EventType, RunEventBoundaryRefreshed)
		}
	}
}

func TestBoundaryRefreshEventsAppendWithLegacyBroadUniqueIndex(t *testing.T) {
	d := openTestDB(t)
	if _, err := d.sql.Exec(`CREATE UNIQUE INDEX legacy_run_events_broad_identity
		ON run_events(run_id, event_type, step_name, action, gate_id, gate_fingerprint, decision_source, approval_surface, consent_mode)`); err != nil {
		t.Fatal(err)
	}
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc123", "def456")
	if err != nil {
		t.Fatal(err)
	}
	step := types.StepReview
	event := RunEvent{
		RunID:     run.ID,
		EventType: RunEventBoundaryRefreshed,
		StepName:  &step,
		Reason:    string(types.BoundaryReasonVerifiedRunWorktree),
		Message:   "safe",
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	if _, err := d.InsertRunEvent(event); err != nil {
		t.Fatal(err)
	}
	events, err := d.GetRunEvents(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want append-only 2 with legacy index", len(events))
	}
	if events[1].GateFingerprint == "" {
		t.Fatalf("second legacy-compatible event gate fingerprint is empty, want unique insert key")
	}
}

func TestGateAutomationLookupPrefersAllowedOverNotRequestedAtSameSecond(t *testing.T) {
	d := openTestDB(t)
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc123", "def456")
	if err != nil {
		t.Fatal(err)
	}
	step := types.StepReview
	ts := int64(1234)
	if _, err := d.InsertRunEvent(RunEvent{
		RunID:           run.ID,
		EventType:       RunEventGateAutomationNotRequested,
		StepName:        &step,
		GateID:          "review",
		GateFingerprint: "fp",
		Status:          types.GateAutomationNotRequested,
		RequestedMode:   types.ConsentModeNone,
		Reason:          "not_requested",
		DecisionSource:  types.DecisionSourceManual,
		ActorType:       types.ActorSystem,
		ApprovalSurface: types.ApprovalSurfaceDaemon,
		ConsentMode:     types.ConsentModeNone,
		CreatedAt:       ts,
	}); err != nil {
		t.Fatal(err)
	}
	action := types.ActionApprove
	if _, err := d.InsertRunEvent(RunEvent{
		RunID:           run.ID,
		EventType:       RunEventGateAutomationAllowed,
		StepName:        &step,
		Action:          &action,
		GateID:          "review",
		GateFingerprint: "fp",
		Status:          types.GateAutomationAllowed,
		RequestedMode:   types.ConsentModeYolo,
		Reason:          string(types.BoundarySafe),
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorSystem,
		ApprovalSurface: types.ApprovalSurfaceTUI,
		ConsentMode:     types.ConsentModeYolo,
		CreatedAt:       ts,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetGateAutomationEvent(run.ID, "review", "fp")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Status != types.GateAutomationAllowed {
		t.Fatalf("event = %+v, want allowed event", got)
	}
}
