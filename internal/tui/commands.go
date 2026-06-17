package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kunchenguid/no-mistakes/internal/boundary"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

const spinnerTickInterval = 120 * time.Millisecond

var runBrowserCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// openBrowserCmd returns a tea.Cmd that opens the given URL in the default browser.
func openBrowserCmd(url string) tea.Cmd {
	return func() tea.Msg {
		name, args := browserCommandSpec(runtime.GOOS, url)
		if err := runBrowserCommand(name, args...); err != nil {
			return errMsg{fmt.Errorf("open PR: %w", err)}
		}
		return nil
	}
}

func browserCommandSpec(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}

func canRerun(run *ipc.RunInfo) bool {
	if run == nil {
		return false
	}
	switch run.Status {
	case types.RunFailed, types.RunCancelled:
		return true
	default:
		return false
	}
}

func (m Model) rerunCmd(requestID uint64) tea.Cmd {
	if !canRerun(m.run) || m.client == nil || m.run == nil {
		return nil
	}
	repoID := m.run.RepoID
	branch := m.run.Branch
	return func() tea.Msg {
		var rerun ipc.RerunResult
		if err := m.client.Call(ipc.MethodRerun, &ipc.RerunParams{RepoID: repoID, Branch: branch}, &rerun); err != nil {
			return rerunErrMsg{err: err, requestID: requestID}
		}
		var result ipc.GetRunResult
		if err := m.client.Call(ipc.MethodGetRun, &ipc.GetRunParams{RunID: rerun.RunID}, &result); err != nil {
			return rerunErrMsg{err: fmt.Errorf("load rerun: %w", err), requestID: requestID}
		}
		if result.Run == nil {
			return rerunErrMsg{err: fmt.Errorf("load rerun: run %s not found", rerun.RunID), requestID: requestID}
		}
		return rerunStartedMsg{run: result.Run, requestID: requestID}
	}
}

// maybeAutoApproveCmd auto-resolves the current awaiting step when yolo mode is
// on, returning nil otherwise. Yolo means "agree to fix every finding": a gate
// whose findings are actionable gets a fix request (all findings selected),
// while a gate with only non-actionable (no-op) findings - or none at all - is
// approved as-is. A step is fixed at most once; the fix re-runs the step and
// re-enters the gate as a fix_review, which yolo then approves so the pipeline
// runs to completion without looping. Each terminal action fires once so
// duplicate events while waiting for the round-trip don't resend it.
func (m Model) maybeAutoApproveCmd() tea.Cmd {
	if !m.yoloMode {
		return nil
	}
	step := awaitingStep(m.steps)
	if step == nil || m.yoloApproved[step.StepName] {
		return nil
	}
	gateID, fingerprint := m.gateIdentity(step)
	meta := types.DecisionMetadata{
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorSystem,
		ApprovalSurface: types.ApprovalSurfaceTUI,
		ConsentMode:     types.ConsentModeYolo,
		GateID:          gateID,
		GateFingerprint: fingerprint,
	}
	if step.Status != types.StepStatusFixReview && !m.yoloFixed[step.StepName] && m.stepHasActionableFindings(step.StepName) {
		m.yoloFixed[step.StepName] = true
		m.resetFindingSelection(step.StepName)
		return m.respondCmdWithMetadata(types.ActionFix, meta)
	}
	m.yoloApproved[step.StepName] = true
	return m.respondCmdWithMetadata(types.ActionApprove, meta)
}

func (m Model) respondCmd(action types.ApprovalAction) tea.Cmd {
	return m.respondCmdWithMetadata(action, types.DecisionMetadata{
		DecisionSource:  types.DecisionSourceManual,
		ActorType:       types.ActorHuman,
		ApprovalSurface: types.ApprovalSurfaceTUI,
		ConsentMode:     types.ConsentModeManual,
	})
}

func (m Model) respondCmdWithMetadata(action types.ApprovalAction, meta types.DecisionMetadata) tea.Cmd {
	step := awaitingStep(m.steps)
	if step == nil {
		return nil
	}
	if action == types.ActionFix {
		ids := m.selectedFindingIDs(step.StepName)
		userAdded := m.selectedUserAddedFindings(step.StepName)
		if len(ids) == 0 && len(userAdded) == 0 && len(m.findingItems(step.StepName)) > 0 {
			return nil
		}
	}
	return func() tea.Msg {
		meta = types.NormalizeRespondDecisionMetadata(meta)
		params := &ipc.RespondParams{
			RunID:           m.runID,
			Step:            step.StepName,
			Action:          action,
			DecisionSource:  meta.DecisionSource,
			ActorType:       meta.ActorType,
			ApprovalSurface: meta.ApprovalSurface,
			ConsentMode:     meta.ConsentMode,
			GateID:          meta.GateID,
			GateFingerprint: meta.GateFingerprint,
		}
		if action == types.ActionFix {
			ids := m.selectedFindingIDs(step.StepName)
			if len(ids) > 0 {
				params.FindingIDs = ids
				if byStep := m.findingInstructions[step.StepName]; len(byStep) > 0 {
					filtered := make(map[string]string, len(byStep))
					for _, id := range ids {
						if note, ok := byStep[id]; ok && note != "" {
							filtered[id] = note
						}
					}
					if len(filtered) > 0 {
						params.Instructions = filtered
					}
				}
			}
			if added := m.selectedUserAddedFindings(step.StepName); len(added) > 0 {
				params.AddedFindings = append([]types.Finding(nil), added...)
			}
		}
		var result ipc.RespondResult
		err := m.client.Call(ipc.MethodRespond, params, &result)
		if err != nil {
			if meta.DecisionSource == types.DecisionSourceUnattended {
				var runResult ipc.GetRunResult
				if getErr := m.client.Call(ipc.MethodGetRun, &ipc.GetRunParams{RunID: m.runID}, &runResult); getErr == nil &&
					runHasWithheldAutomation(runResult.Run, meta.GateID, meta.GateFingerprint) {
					return automationWithheldMsg{run: runResult.Run, step: step.StepName}
				}
			}
			return errMsg{err}
		}
		return nil
	}
}

func runHasWithheldAutomation(run *ipc.RunInfo, gateID, _ string) bool {
	return run != nil &&
		run.GateAutomation != nil &&
		run.GateAutomation.GateID == gateID &&
		run.GateAutomation.Status == types.GateAutomationWithheld
}

func (m Model) gateIdentity(step *ipc.StepResultInfo) (string, string) {
	if step == nil {
		return "", ""
	}
	gateID := string(step.StepName)
	findings := ""
	if raw, ok := m.stepFindings[step.StepName]; ok {
		findings = raw
	} else if step.FindingsJSON != nil {
		findings = *step.FindingsJSON
	}
	return gateID, boundary.GateFingerprint(m.runID, step.StepName, step.Status, "client", findings)
}

func (m Model) cancelRunCmd() tea.Cmd {
	if m.runID == "" {
		return nil
	}
	return func() tea.Msg {
		params := &ipc.CancelRunParams{RunID: m.runID}
		var result ipc.CancelRunResult
		err := m.client.Call(ipc.MethodCancelRun, params, &result)
		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m Model) subscribeCmd() tea.Cmd {
	return func() tea.Msg {
		events, cancel, err := ipc.Subscribe(m.socketPath, &ipc.SubscribeParams{
			RunID: m.runID,
		})
		if err != nil {
			return subscriptionErrMsg{err: fmt.Errorf("subscribe: %w", err), subscriptionID: m.subscriptionID}
		}
		return connectedMsg{events: events, cancelSub: cancel, subscriptionID: m.subscriptionID}
	}
}

func (m Model) waitForEvent() tea.Cmd {
	events := m.events
	if events == nil {
		return nil
	}
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return subscriptionErrMsg{err: fmt.Errorf("event stream closed"), subscriptionID: m.subscriptionID}
		}
		return eventMsg{event: event, subscriptionID: m.subscriptionID}
	}
}

func (m Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(spinnerTickInterval, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m Model) hasSpinningStep() bool {
	for _, step := range m.steps {
		switch step.Status {
		case types.StepStatusRunning, types.StepStatusFixing:
			return true
		}
	}
	return false
}

func (m *Model) startSpinnerIfNeeded() tea.Cmd {
	if m.done || m.quitting || m.spinnerScheduled || !m.hasSpinningStep() {
		return nil
	}
	m.spinnerScheduled = true
	return m.spinnerTickCmd()
}
