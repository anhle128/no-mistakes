//go:build e2e

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestYoloNoWorktree(t *testing.T) {
	h := NewHarness(t, SetupOpts{Agent: "claude", Scenario: axiScenario(t)})

	h.CommitChange("init-yolo", "seed.txt", "seed\n", "seed for no-worktree-yolo")
	initWorktree := h.AddWorktree("init-yolo")
	if out, err := h.RunInDir(initWorktree, "init"); err != nil {
		t.Fatalf("nm init: %v\n%s", err, out)
	}

	h.CommitChange("feature/yolo-guard", "feature.txt", "change\n", "add guarded feature")
	featureWorktree := h.AddWorktree("feature/yolo-guard")

	gateOut, err := h.RunInDir(featureWorktree, "axi", "run", "--intent", axiIntent)
	if err != nil {
		t.Fatalf("axi run should stop at review gate: %v\n%s", err, gateOut)
	}
	if !strings.Contains(gateOut, "status: awaiting_approval") {
		t.Fatalf("axi run did not stop at approval gate:\n%s", gateOut)
	}

	gated := waitForStepStatus(t, h, "feature/yolo-guard", types.StepReview, types.StepStatusAwaitingApproval, 60*time.Second)
	t.Cleanup(func() { h.CancelRun(gated.ID) })

	managedWorktree := paths.WithRoot(h.NMHome).WorktreeDir(h.repoID(), gated.ID)
	if err := os.RemoveAll(managedWorktree); err != nil {
		t.Fatalf("remove managed worktree: %v", err)
	}

	withheldOut, err := h.RunInDir(featureWorktree, "axi", "run", "--yes", "--intent", axiIntent)
	if err != nil {
		t.Fatalf("axi run --yes should return the withheld gate, not fail: %v\n%s", err, withheldOut)
	}
	for _, want := range []string{
		"automation:",
		"requested_mode: yes",
		"status: withheld",
		"reason: unknown",
		"gate: review",
		"Respond manually to this gate",
		"Restart validation through no-mistakes",
	} {
		if !strings.Contains(withheldOut, want) {
			t.Fatalf("withheld output missing %q:\n%s", want, withheldOut)
		}
	}

	ref := "refs/heads/feature/yolo-guard"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if out, err := h.runGit(ctx, h.UpstreamDir, "rev-parse", "--verify", ref); err == nil {
		t.Fatalf("upstream branch %s should not have been pushed, got %s", ref, strings.TrimSpace(string(out)))
	}

	run := h.RunInfo(gated.ID)
	if run.GateAutomation == nil || run.GateAutomation.Status != types.GateAutomationWithheld {
		t.Fatalf("run gate automation = %+v, want withheld", run.GateAutomation)
	}
	if run.Boundary.Status != types.BoundaryUnknown || run.Boundary.Reason != types.BoundaryReasonMissingWorktree {
		t.Fatalf("boundary = %+v, want unknown missing_worktree", run.Boundary)
	}
	reviewStatus := stepStatus(run, types.StepReview)
	if reviewStatus != types.StepStatusAwaitingApproval {
		t.Fatalf("review status = %s, want awaiting_approval", reviewStatus)
	}
}

func stepStatus(run *ipc.RunInfo, step types.StepName) types.StepStatus {
	for _, s := range run.Steps {
		if s.StepName == step {
			return s.Status
		}
	}
	return ""
}
