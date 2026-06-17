package boundary

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

const gateFingerprintVersion = "gate-v1"

// GateFingerprint returns a stable identity for one pending gate decision.
func GateFingerprint(runID string, step types.StepName, status types.StepStatus, gateVersion, findingsJSON string) string {
	h := sha256.New()
	for _, part := range []string{
		gateFingerprintVersion,
		runID,
		string(step),
		string(status),
		strings.TrimSpace(gateVersion),
		strings.TrimSpace(findingsJSON),
	} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

// AutomationForBoundary converts a boundary proof and requested consent mode
// into the user-facing gate automation object.
func AutomationForBoundary(boundary types.ExecutionBoundary, gateID, fingerprint string, requested types.ConsentMode) types.GateAutomation {
	boundary = boundary.Normalize()
	if !requested.Valid() || requested == types.ConsentModeNone {
		requested = types.ConsentModeNone
	}
	automation := types.GateAutomation{
		GateID:          gateID,
		GateFingerprint: fingerprint,
		RequestedMode:   requested,
		Reason:          "not_requested",
		Status:          types.GateAutomationNotRequested,
		RecoveryOptions: nil,
		Message:         "",
	}
	if requested == types.ConsentModeNone {
		return automation
	}
	if boundary.Status == types.BoundarySafe {
		automation.Status = types.GateAutomationAllowed
		automation.Reason = string(types.BoundarySafe)
		automation.Message = "Unattended gate automation is allowed for this verified disposable worktree."
		return automation
	}
	automation.Status = types.GateAutomationWithheld
	automation.Reason = string(boundary.Status)
	automation.Message = WithheldMessage(boundary)
	automation.RecoveryOptions = RecoveryOptions()
	return automation
}

func WithheldMessage(boundary types.ExecutionBoundary) string {
	boundary = boundary.Normalize()
	detail := strings.TrimSpace(boundary.Detail)
	if detail != "" {
		return fmt.Sprintf("Unattended automation was withheld because the run boundary is %s: %s.", boundary.Status, detail)
	}
	return fmt.Sprintf("Unattended automation was withheld because the run boundary is %s (%s).", boundary.Status, boundary.Reason)
}

func RecoveryOptions() []string {
	return []string{
		"Respond manually to this gate",
		"Restart validation through no-mistakes so the run uses a disposable worktree",
	}
}

// RequireSafe returns an error suitable for failing closed before an automatic
// source-changing or remote-advancing action.
func RequireSafe(boundary types.ExecutionBoundary, action string) error {
	boundary = boundary.Normalize()
	if boundary.Status == types.BoundarySafe {
		return nil
	}
	if strings.TrimSpace(action) == "" {
		action = "automation"
	}
	return fmt.Errorf("withheld %s: boundary %s (%s): %s", action, boundary.Status, boundary.Reason, boundary.Detail)
}
