package boundary

import (
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestGateFingerprintChangesWithGateInputs(t *testing.T) {
	a := GateFingerprint("run-1", types.StepReview, types.StepStatusAwaitingApproval, "round-1", `{"findings":[]}`)
	b := GateFingerprint("run-1", types.StepReview, types.StepStatusAwaitingApproval, "round-1", `{"findings":[{"id":"f1"}]}`)
	if a == b {
		t.Fatal("fingerprint should change when decision inputs change")
	}
	if got := GateFingerprint("run-1", types.StepReview, types.StepStatusAwaitingApproval, "round-1", `{"findings":[]}`); got != a {
		t.Fatalf("fingerprint should be stable: %q != %q", got, a)
	}
	c := GateFingerprint("run-1", types.StepReview, types.StepStatusAwaitingApproval, "round-2", `{"findings":[]}`)
	if c == a {
		t.Fatal("fingerprint should change when gate version changes")
	}
}

func TestAutomationForBoundary(t *testing.T) {
	safe := types.ExecutionBoundary{Status: types.BoundarySafe, Reason: types.BoundaryReasonVerifiedRunWorktree}
	got := AutomationForBoundary(safe, "review", "fp", types.ConsentModeYes)
	if got.Status != types.GateAutomationAllowed {
		t.Fatalf("safe status = %q, want allowed", got.Status)
	}

	unknown := types.ExecutionBoundary{Status: types.BoundaryUnknown, Reason: types.BoundaryReasonMissingWorktree}
	got = AutomationForBoundary(unknown, "review", "fp", types.ConsentModeYolo)
	if got.Status != types.GateAutomationWithheld {
		t.Fatalf("unknown status = %q, want withheld", got.Status)
	}
	if len(got.RecoveryOptions) == 0 {
		t.Fatal("withheld automation should include recovery options")
	}

	got = AutomationForBoundary(unknown, "review", "fp", types.ConsentModeNone)
	if got.Status != types.GateAutomationNotRequested {
		t.Fatalf("none status = %q, want not_requested", got.Status)
	}
}

func TestRequireSafeRejectsStaleOrUnknownProof(t *testing.T) {
	err := RequireSafe(types.ExecutionBoundary{
		Status: types.BoundaryUnknown,
		Reason: types.BoundaryReasonStaleProof,
		Detail: "boundary proof is stale",
	}, "auto-fix review")
	if err == nil {
		t.Fatal("RequireSafe should reject stale unknown proof")
	}
	if got := err.Error(); got == "" || !containsAll(got, "auto-fix review", "unknown", "stale_proof") {
		t.Fatalf("error = %q, want action and stale proof reason", got)
	}
}

func containsAll(s string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(s, needle) {
			return false
		}
	}
	return true
}
