package tui

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/types"
	"github.com/muesli/termenv"
)

func TestModel_Update_YoloKeyTogglesMode(t *testing.T) {
	run := testRun()
	m := NewModel("/tmp/sock", nil, run)
	if m.yoloMode {
		t.Fatal("expected yolo mode off by default")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model := updated.(Model)
	if !model.yoloMode {
		t.Fatal("expected first y press to enable yolo mode")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model = updated.(Model)
	if model.yoloMode {
		t.Fatal("expected second y press to disable yolo mode")
	}
}

func TestModel_Yolo_AutoApprovesAwaitingStep(t *testing.T) {
	sock := testSocketPath(t)
	srv := startTestIPCServer(t, sock)

	var mu sync.Mutex
	var calls []ipc.RespondParams
	srv.Handle(ipc.MethodRespond, func(_ context.Context, raw json.RawMessage) (interface{}, error) {
		var params ipc.RespondParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		mu.Lock()
		calls = append(calls, params)
		mu.Unlock()
		return &ipc.RespondResult{}, nil
	})

	client, err := ipc.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	m := NewModel(sock, client, run)
	m.yoloMode = true

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected auto-approve command when yolo on and step awaiting")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg from auto-approve, got %#v", msg)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 respond call, got %d", len(calls))
	}
	if calls[0].Action != types.ActionApprove {
		t.Fatalf("action = %s, want %s", calls[0].Action, types.ActionApprove)
	}
	if calls[0].Step != types.StepReview {
		t.Fatalf("step = %s, want %s", calls[0].Step, types.StepReview)
	}
	if calls[0].DecisionSource != types.DecisionSourceUnattended || calls[0].ApprovalSurface != types.ApprovalSurfaceTUI || calls[0].ConsentMode != types.ConsentModeYolo {
		t.Fatalf("yolo metadata = %+v, want unattended tui yolo", calls[0])
	}
}

// captureRespond wires a model-facing IPC server that records every Respond
// call, returning the connected client plus accessors for the captured params.
func captureRespond(t *testing.T) (string, *ipc.Client, func() []ipc.RespondParams) {
	t.Helper()
	sock := testSocketPath(t)
	srv := startTestIPCServer(t, sock)

	var mu sync.Mutex
	var calls []ipc.RespondParams
	srv.Handle(ipc.MethodRespond, func(_ context.Context, raw json.RawMessage) (interface{}, error) {
		var params ipc.RespondParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		mu.Lock()
		calls = append(calls, params)
		mu.Unlock()
		return &ipc.RespondResult{}, nil
	})

	client, err := ipc.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })

	return sock, client, func() []ipc.RespondParams {
		mu.Lock()
		defer mu.Unlock()
		return append([]ipc.RespondParams(nil), calls...)
	}
}

func TestModel_Yolo_FixesActionableFindings(t *testing.T) {
	sock, client, snapshot := captureRespond(t)

	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	fj := `{"findings":[{"id":"review-1","severity":"warning","description":"design choice","action":"ask-user"}],"summary":"1 issue"}`
	run.Steps[0].FindingsJSON = &fj
	m := NewModel(sock, client, run)
	m.yoloMode = true

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected a yolo command for an awaiting step with actionable findings")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg, got %#v", msg)
	}

	calls := snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 respond call, got %d", len(calls))
	}
	t.Logf("yolo response action=%s finding_ids=%v", calls[0].Action, calls[0].FindingIDs)
	if calls[0].Action != types.ActionFix {
		t.Fatalf("action = %s, want %s", calls[0].Action, types.ActionFix)
	}
	if len(calls[0].FindingIDs) != 1 || calls[0].FindingIDs[0] != "review-1" {
		t.Fatalf("FindingIDs = %v, want [review-1]", calls[0].FindingIDs)
	}
}

func TestModel_Yolo_FixesAllActionableFindingsDespiteManualDeselection(t *testing.T) {
	sock, client, snapshot := captureRespond(t)

	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	fj := `{"findings":[{"id":"review-1","severity":"warning","description":"first","action":"ask-user"},{"id":"review-2","severity":"warning","description":"second","action":"ask-user"}],"summary":"2 issues"}`
	run.Steps[0].FindingsJSON = &fj
	m := NewModel(sock, client, run)
	m.yoloMode = true
	m.findingSelections[types.StepReview] = map[string]bool{"review-1": true}

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected a yolo command for an awaiting step with actionable findings")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg, got %#v", msg)
	}

	calls := snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 respond call, got %d", len(calls))
	}
	t.Logf("yolo response action=%s finding_ids=%v", calls[0].Action, calls[0].FindingIDs)
	if calls[0].Action != types.ActionFix {
		t.Fatalf("action = %s, want %s", calls[0].Action, types.ActionFix)
	}
	if got, want := calls[0].FindingIDs, []string{"review-1", "review-2"}; !slices.Equal(got, want) {
		t.Fatalf("FindingIDs = %v, want %v", got, want)
	}
}

func TestModel_Yolo_ApprovesNonActionableFindings(t *testing.T) {
	sock, client, snapshot := captureRespond(t)

	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	fj := `{"findings":[{"id":"review-1","severity":"info","description":"fyi","action":"no-op"}],"summary":"1 note"}`
	run.Steps[0].FindingsJSON = &fj
	m := NewModel(sock, client, run)
	m.yoloMode = true

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected a yolo command for an awaiting step")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg, got %#v", msg)
	}

	calls := snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 respond call, got %d", len(calls))
	}
	t.Logf("yolo response action=%s finding_ids=%v", calls[0].Action, calls[0].FindingIDs)
	if calls[0].Action != types.ActionApprove {
		t.Fatalf("action = %s, want %s (non-actionable findings should be approved)", calls[0].Action, types.ActionApprove)
	}
}

func TestModel_Yolo_ApprovesFixReviewAfterFixingOnce(t *testing.T) {
	sock, client, snapshot := captureRespond(t)

	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	fj := `{"findings":[{"id":"review-1","severity":"warning","description":"design choice","action":"ask-user"}],"summary":"1 issue"}`
	run.Steps[0].FindingsJSON = &fj
	m := NewModel(sock, client, run)
	m.yoloMode = true

	// First gate: actionable findings -> fix.
	if cmd := m.maybeAutoApproveCmd(); cmd != nil {
		cmd()
	} else {
		t.Fatal("expected fix command on first gate")
	}

	// The fix re-runs the step, which re-enters the gate as a fix_review. Yolo
	// must not fix again (that risks an unbounded loop); it accepts the result.
	m.steps[0].Status = types.StepStatusFixReview
	if cmd := m.maybeAutoApproveCmd(); cmd != nil {
		cmd()
	} else {
		t.Fatal("expected approve command on fix_review gate")
	}

	calls := snapshot()
	if len(calls) != 2 {
		t.Fatalf("expected 2 respond calls, got %d", len(calls))
	}
	t.Logf("yolo responses first_action=%s first_finding_ids=%v second_action=%s second_finding_ids=%v", calls[0].Action, calls[0].FindingIDs, calls[1].Action, calls[1].FindingIDs)
	if calls[0].Action != types.ActionFix {
		t.Fatalf("first action = %s, want %s", calls[0].Action, types.ActionFix)
	}
	if calls[1].Action != types.ActionApprove {
		t.Fatalf("second action = %s, want %s", calls[1].Action, types.ActionApprove)
	}
}

func TestModel_Yolo_ApprovesExistingFixReviewWithoutPriorFix(t *testing.T) {
	sock, client, snapshot := captureRespond(t)

	run := testRun()
	run.Steps[0].Status = types.StepStatusFixReview
	fj := `{"findings":[{"id":"review-1","severity":"warning","description":"still here","action":"ask-user"}],"summary":"1 issue"}`
	run.Steps[0].FindingsJSON = &fj
	m := NewModel(sock, client, run)
	m.yoloMode = true

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected yolo to approve an existing fix_review gate")
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg, got %#v", msg)
	}

	calls := snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 respond call, got %d", len(calls))
	}
	t.Logf("yolo response action=%s finding_ids=%v", calls[0].Action, calls[0].FindingIDs)
	if calls[0].Action != types.ActionApprove {
		t.Fatalf("action = %s, want %s", calls[0].Action, types.ActionApprove)
	}
}

func TestModel_Yolo_DoesNotAutoApproveTwiceForSameStep(t *testing.T) {
	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	m := NewModel("/tmp/sock", nil, run)
	m.yoloMode = true

	if cmd := m.maybeAutoApproveCmd(); cmd == nil {
		t.Fatal("expected first auto-approve command")
	}
	if cmd := m.maybeAutoApproveCmd(); cmd != nil {
		t.Fatal("expected no second auto-approve command for the same awaiting step")
	}
}

func TestModel_Yolo_NoAutoApproveWhenOff(t *testing.T) {
	run := testRun()
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	m := NewModel("/tmp/sock", nil, run)

	if cmd := m.maybeAutoApproveCmd(); cmd != nil {
		t.Fatal("expected no auto-approve command when yolo off")
	}
}

func TestModel_Yolo_WithholdsWhenBoundaryUnknown(t *testing.T) {
	sock := testSocketPath(t)
	srv := startTestIPCServer(t, sock)

	var mu sync.Mutex
	var calls []ipc.RespondParams
	var fingerprint string
	srv.Handle(ipc.MethodRespond, func(_ context.Context, raw json.RawMessage) (interface{}, error) {
		var params ipc.RespondParams
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
		mu.Lock()
		calls = append(calls, params)
		mu.Unlock()
		return nil, errors.New("withheld by daemon")
	})
	srv.Handle(ipc.MethodGetRun, func(_ context.Context, raw json.RawMessage) (interface{}, error) {
		run := testRun()
		run.Boundary = types.ExecutionBoundary{Status: types.BoundaryUnknown, Reason: types.BoundaryReasonMissingWorktree, Detail: "missing"}
		run.Steps[0].Status = types.StepStatusAwaitingApproval
		run.GateAutomation = &types.GateAutomation{
			GateID:          "review",
			GateFingerprint: fingerprint,
			Status:          types.GateAutomationWithheld,
			RequestedMode:   types.ConsentModeYolo,
			Reason:          "unknown",
			Message:         "withheld by daemon",
		}
		return &ipc.GetRunResult{Run: run}, nil
	})
	client, err := ipc.Dial(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	run := testRun()
	run.Boundary = types.ExecutionBoundary{Status: types.BoundaryUnknown, Reason: types.BoundaryReasonMissingWorktree, Detail: "missing"}
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	m := NewModel(sock, client, run)
	m.yoloMode = true
	_, fingerprint = m.gateIdentity(&m.steps[0])

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected yolo to notify daemon when boundary is unknown")
	}
	msg := cmd()
	withheld, ok := msg.(automationWithheldMsg)
	if !ok {
		t.Fatalf("expected automationWithheldMsg, got %#v", msg)
	}
	updated, _ := m.Update(withheld)
	model := updated.(Model)
	if model.err != nil {
		t.Fatalf("err = %v, want nil for persisted withheld automation", model.err)
	}
	if model.run.GateAutomation == nil {
		t.Fatal("expected local withheld automation state")
	}
	if model.run.GateAutomation.Status != types.GateAutomationWithheld {
		t.Fatalf("automation status = %q, want withheld", model.run.GateAutomation.Status)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(calls) != 1 {
		t.Fatalf("respond calls = %d, want 1", len(calls))
	}
	if calls[0].DecisionSource != types.DecisionSourceUnattended || calls[0].ConsentMode != types.ConsentModeYolo {
		t.Fatalf("metadata = %+v, want unattended yolo", calls[0])
	}
}

func TestModel_Yolo_DoesNotCreateLocalWithheldStateFromStaleBoundary(t *testing.T) {
	sock, client, snapshot := captureRespond(t)

	run := testRun()
	run.Boundary = types.ExecutionBoundary{Status: types.BoundaryUnknown, Reason: types.BoundaryReasonMissingWorktree, Detail: "stale local state"}
	run.Steps[0].Status = types.StepStatusAwaitingApproval
	fj := `{"findings":[{"id":"review-1","severity":"warning","description":"needs fix","action":"ask-user"}],"summary":"1 issue"}`
	run.Steps[0].FindingsJSON = &fj
	m := NewModel(sock, client, run)
	m.yoloMode = true

	cmd := m.maybeAutoApproveCmd()
	if cmd == nil {
		t.Fatal("expected yolo command even when local boundary is stale unknown")
	}
	if m.run.GateAutomation != nil {
		t.Fatalf("local GateAutomation = %+v, want daemon-owned state only", m.run.GateAutomation)
	}
	if msg := cmd(); msg != nil {
		t.Fatalf("expected nil msg after daemon accepted response, got %#v", msg)
	}
	calls := snapshot()
	if len(calls) != 1 {
		t.Fatalf("respond calls = %d, want 1", len(calls))
	}
	if calls[0].Action != types.ActionFix {
		t.Fatalf("action = %s, want fix", calls[0].Action)
	}
}

func TestModel_View_ShowsPersistedWithheldAutomationWhenYoloOff(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	run := testRun()
	run.GateAutomation = &types.GateAutomation{
		GateID:          string(types.StepReview),
		Status:          types.GateAutomationWithheld,
		RequestedMode:   types.ConsentModeYes,
		Reason:          string(types.BoundaryUnknown),
		Message:         "Unattended automation was withheld because the run boundary is unknown.",
		RecoveryOptions: []string{"Respond manually to this gate"},
	}
	m := NewModel("", nil, run)
	m.width = 120
	m.height = 40
	m.yoloMode = false

	plain := stripANSI(m.View())
	for _, want := range []string{
		"YOLO withheld",
		"mode yes",
		"gate review",
		"reason unknown",
		"Unattended automation was withheld because the run boundary is unknown.",
		"Respond manually to this gate",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("view missing %q:\n%s", want, plain)
		}
	}
}

func TestModel_View_FooterShowsYoloLabel(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	run := testRun()
	m := NewModel("", nil, run)
	m.width = 120
	m.height = 40

	plain := stripANSI(m.View())
	if !footerContains(plain, "y", "yolo") {
		t.Errorf("footer should show 'y yolo' when yolo off, got:\n%s", plain)
	}

	m.yoloMode = true
	plain = stripANSI(m.View())
	if !footerContains(plain, "y", "end yolo") {
		t.Errorf("footer should show 'y end yolo' when yolo on, got:\n%s", plain)
	}
}

func footerContains(plain string, needles ...string) bool {
	for _, line := range strings.Split(plain, "\n") {
		all := true
		for _, n := range needles {
			if !strings.Contains(line, n) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}
