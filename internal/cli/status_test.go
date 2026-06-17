package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/boundary"
	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestStatusShowsPersistedWithheldGateAutomation(t *testing.T) {
	setupTestRepo(t)
	repoDir, err := git.FindGitRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	p := paths.WithRoot(os.Getenv("NM_HOME"))
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	database, err := db.Open(p.DB())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	repo, err := database.InsertRepo(repoDir, "git@example.com:repo.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := database.InsertRun(repo.ID, "feature/status", "abcdef1234567890", "base")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpdateRunStatus(run.ID, types.RunRunning); err != nil {
		t.Fatal(err)
	}
	verifiedAt := int64(123)
	if err := database.UpdateRunBoundary(run.ID, types.ExecutionBoundary{
		Status:          types.BoundaryUnknown,
		Reason:          types.BoundaryReasonMissingWorktree,
		Detail:          "worktree is missing",
		VerifiedAt:      verifiedAt,
		VerifierVersion: "test",
	}); err != nil {
		t.Fatal(err)
	}
	step, err := database.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.UpdateStepStatus(step.ID, types.StepStatusAwaitingApproval); err != nil {
		t.Fatal(err)
	}
	gateVersion := db.FallbackStepGateVersion(step.ID)
	fingerprint := boundary.GateFingerprint(run.ID, types.StepReview, types.StepStatusAwaitingApproval, gateVersion, "")
	stepName := types.StepReview
	if _, err := database.InsertRunEvent(db.RunEvent{
		RunID:           run.ID,
		EventType:       db.RunEventGateAutomationWithheld,
		StepName:        &stepName,
		GateID:          string(types.StepReview),
		GateFingerprint: fingerprint,
		Status:          types.GateAutomationWithheld,
		RequestedMode:   types.ConsentModeYes,
		Reason:          string(types.BoundaryUnknown),
		Message:         "Unattended automation was withheld because the run boundary is unknown.",
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorAgent,
		ApprovalSurface: types.ApprovalSurfaceAXI,
		ConsentMode:     types.ConsentModeYes,
	}); err != nil {
		t.Fatal(err)
	}

	out, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status command: %v\n%s", err, out)
	}

	for _, want := range []string{
		"automation:",
		"withheld",
		"mode:",
		"yes",
		"gate:",
		"review",
		"boundary:",
		"unknown (missing_worktree)",
		"message:",
		"Unattended automation was withheld because the run boundary is unknown.",
		"Respond manually to this gate",
		"Restart validation through no-mistakes so the run uses a disposable worktree",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output missing %q:\n%s", want, out)
		}
	}
}
