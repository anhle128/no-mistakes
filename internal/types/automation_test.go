package types

import "testing"

func TestAutomationEnumsValidate(t *testing.T) {
	if !BoundarySafe.Valid() || !BoundaryUnsafe.Valid() || !BoundaryUnknown.Valid() {
		t.Fatal("boundary statuses should validate")
	}
	if BoundaryStatus("maybe").Valid() {
		t.Fatal("unknown boundary status should not validate")
	}
	if !GateAutomationAllowed.Valid() || !GateAutomationWithheld.Valid() || !GateAutomationNotRequested.Valid() {
		t.Fatal("gate automation statuses should validate")
	}
	if GateAutomationStatus("paused").Valid() {
		t.Fatal("unknown gate automation status should not validate")
	}
}

func TestNormalizeRespondDecisionMetadataDefaultsLegacyToManual(t *testing.T) {
	got := NormalizeRespondDecisionMetadata(DecisionMetadata{})
	if got.DecisionSource != DecisionSourceManual {
		t.Fatalf("DecisionSource = %q, want %q", got.DecisionSource, DecisionSourceManual)
	}
	if got.ActorType != ActorHuman {
		t.Fatalf("ActorType = %q, want %q", got.ActorType, ActorHuman)
	}
	if got.ApprovalSurface != ApprovalSurfaceUnknown {
		t.Fatalf("ApprovalSurface = %q, want %q", got.ApprovalSurface, ApprovalSurfaceUnknown)
	}
	if got.ConsentMode != ConsentModeManual {
		t.Fatalf("ConsentMode = %q, want %q", got.ConsentMode, ConsentModeManual)
	}
}

func TestNormalizeRespondDecisionMetadataKeepsUnattendedIntent(t *testing.T) {
	got := NormalizeRespondDecisionMetadata(DecisionMetadata{
		DecisionSource:  DecisionSourceUnattended,
		ActorType:       ActorAgent,
		ApprovalSurface: ApprovalSurfaceAXI,
		ConsentMode:     ConsentModeYes,
		GateID:          "review",
	})
	if got.DecisionSource != DecisionSourceUnattended {
		t.Fatalf("DecisionSource = %q, want unattended", got.DecisionSource)
	}
	if got.ActorType != ActorAgent || got.ApprovalSurface != ApprovalSurfaceAXI || got.ConsentMode != ConsentModeYes {
		t.Fatalf("metadata changed unexpectedly: %+v", got)
	}
}
