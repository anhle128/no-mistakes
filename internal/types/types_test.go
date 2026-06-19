package types

import (
	"encoding/json"
	"testing"
)

func TestAllStepsOrder(t *testing.T) {
	steps := AllSteps()
	if len(steps) != 9 {
		t.Fatalf("expected 9 steps, got %d", len(steps))
	}

	expected := []StepName{StepIntent, StepRebase, StepReview, StepTest, StepDocument, StepLint, StepPush, StepPR, StepCI}
	for i, s := range steps {
		if s != expected[i] {
			t.Errorf("step[%d] = %q, want %q", i, s, expected[i])
		}
	}
}

func TestStepNameOrder(t *testing.T) {
	tests := []struct {
		step StepName
		want int
	}{
		{StepIntent, 1},
		{StepRebase, 2},
		{StepReview, 3},
		{StepTest, 4},
		{StepDocument, 5},
		{StepLint, 6},
		{StepPush, 7},
		{StepPR, 8},
		{StepCI, 9},
		{StepName("unknown"), 0},
	}

	for _, tt := range tests {
		if got := tt.step.Order(); got != tt.want {
			t.Errorf("%q.Order() = %d, want %d", tt.step, got, tt.want)
		}
	}
}

func TestStepNameUnmarshalJSON_LegacyBabysit(t *testing.T) {
	var step StepName
	if err := json.Unmarshal([]byte(`"babysit"`), &step); err != nil {
		t.Fatalf("unmarshal step name: %v", err)
	}
	if step != StepCI {
		t.Fatalf("step = %q, want %q", step, StepCI)
	}
}

func TestWorktreeModeValidationAndLabels(t *testing.T) {
	tests := []struct {
		mode      WorktreeMode
		wantNorm  WorktreeMode
		wantValid bool
		wantLabel string
	}{
		{"", WorktreeModeIsolated, true, "disposable no-mistakes checkout"},
		{WorktreeModeIsolated, WorktreeModeIsolated, true, "disposable no-mistakes checkout"},
		{WorktreeModeCurrent, WorktreeModeCurrent, true, "uses this checkout"},
		{"unknown", "unknown", false, "disposable no-mistakes checkout"},
	}
	for _, tt := range tests {
		if got := NormalizeWorktreeMode(tt.mode); got != tt.wantNorm {
			t.Fatalf("NormalizeWorktreeMode(%q) = %q, want %q", tt.mode, got, tt.wantNorm)
		}
		if got := tt.mode.Valid(); got != tt.wantValid {
			t.Fatalf("%q.Valid() = %v, want %v", tt.mode, got, tt.wantValid)
		}
		if got := tt.mode.Label(); got != tt.wantLabel {
			t.Fatalf("%q.Label() = %q, want %q", tt.mode, got, tt.wantLabel)
		}
	}
}

func TestMetadataAndEvidenceStateValidation(t *testing.T) {
	if NormalizeMetadataAvailability("") != MetadataAvailable {
		t.Fatal("empty metadata availability should normalize to available")
	}
	if !MetadataNotRecorded.Valid() || !MetadataInvalid.Valid() || MetadataAvailability("bad").Valid() {
		t.Fatal("metadata availability validation mismatch")
	}
	if NormalizeEvidenceState("") != EvidenceComplete {
		t.Fatal("empty evidence state should normalize to complete")
	}
	if !EvidenceIncomplete.Valid() || !EvidenceDegraded.Valid() || EvidenceState("bad").Valid() {
		t.Fatal("evidence state validation mismatch")
	}
}
