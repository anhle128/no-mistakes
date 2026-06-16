package reviewhandoff

import "github.com/kunchenguid/no-mistakes/internal/types"

// PhaseLabel returns the human review sub-phase label for review statuses.
func PhaseLabel(step types.StepName, status types.StepStatus) string {
	if step != types.StepReview {
		return ""
	}
	switch status {
	case types.StepStatusRunning:
		return "Review preview"
	case types.StepStatusAwaitingApproval:
		return "Review preview complete"
	case types.StepStatusFixing:
		return "Fixing review issues"
	case types.StepStatusFixReview:
		return "Review fix result"
	default:
		return ""
	}
}
