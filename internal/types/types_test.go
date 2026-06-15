package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAllStepsInExecutionOrder(t *testing.T) {
	want := []StepName{StepIntent, StepRebase, StepReview, StepTest, StepDocument, StepLint, StepPush, StepPR, StepCI}
	if got := AllSteps(); !reflect.DeepEqual(got, want) {
		t.Fatalf("AllSteps = %v, want %v", got, want)
	}
}

func TestStepNameOrder(t *testing.T) {
	for i, step := range AllSteps() {
		if got := step.Order(); got != i+1 {
			t.Fatalf("%s.Order() = %d, want %d", step, got, i+1)
		}
	}
	if got := StepName("unknown").Order(); got != 0 {
		t.Fatalf("unknown step order = %d, want 0", got)
	}
}

func TestStepNameLegacyBabysitCompatibility(t *testing.T) {
	var step StepName
	if err := json.Unmarshal([]byte(`"babysit"`), &step); err != nil {
		t.Fatalf("UnmarshalJSON: %v", err)
	}
	if step != StepCI {
		t.Fatalf("legacy babysit step = %q, want %q", step, StepCI)
	}
	var scanned StepName
	if err := scanned.Scan("babysit"); err != nil {
		t.Fatalf("Scan string: %v", err)
	}
	if scanned != StepCI {
		t.Fatalf("legacy babysit scan = %q, want %q", scanned, StepCI)
	}
}

func TestReviewPhaseLabel(t *testing.T) {
	tests := []struct {
		name   string
		step   StepName
		status StepStatus
		want   string
	}{
		{"review running", StepReview, StepStatusRunning, ReviewPhasePreview},
		{"review awaiting", StepReview, StepStatusAwaitingApproval, ReviewPhasePreviewComplete},
		{"review fixing", StepReview, StepStatusFixing, ReviewPhaseFixing},
		{"review fix review", StepReview, StepStatusFixReview, ReviewPhaseFixResult},
		{"review completed omitted", StepReview, StepStatusCompleted, ""},
		{"non review omitted", StepTest, StepStatusAwaitingApproval, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReviewPhaseLabel(tt.step, tt.status); got != tt.want {
				t.Fatalf("ReviewPhaseLabel(%q, %q) = %q, want %q", tt.step, tt.status, got, tt.want)
			}
		})
	}
}
