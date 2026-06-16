package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/reviewhandoff"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestExecutor_ReviewHandoffWritesFileAndEmitsPath(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()
	initGitRepo(t, workDir)
	writeTestFile(t, workDir, "tasks.md", "# tasks\n")

	findings := `{"findings":[{"id":"review-1","severity":"error","file":"app.go","line":7,"description":"bug","suggested_fix":"fix it","action":"ask-user"}],"summary":"1 issue"}`
	step := newApprovalStep(types.StepReview, findings)
	exec := NewExecutor(database, p, nil, nil, []Step{step}, nil)
	events := collectEvents(exec)

	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()

	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	event := waitForStepEvent(t, events, ipc.EventStepCompleted, types.StepReview)
	if event.ReviewFilePath == nil || *event.ReviewFilePath == "" {
		t.Fatal("expected review file path on awaiting event")
	}
	if event.ReviewPhaseLabel == nil || *event.ReviewPhaseLabel != "Review preview complete" {
		t.Fatalf("review phase label = %v, want Review preview complete", event.ReviewPhaseLabel)
	}
	if got, want := filepath.Base(*event.ReviewFilePath), reviewhandoff.FileName(run.ID); got != want {
		t.Fatalf("review file name = %q, want %q", got, want)
	}
	data, err := os.ReadFile(filepath.Join(workDir, filepath.FromSlash(*event.ReviewFilePath)))
	if err != nil {
		t.Fatalf("read review file: %v", err)
	}
	text := string(data)
	for _, want := range []string{"run_id: " + run.ID, "processed_action: pending", "```no-mistakes-response", "id: review-1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("review file missing %q:\n%s", want, text)
		}
	}

	if err := exec.Respond(types.StepReview, types.ActionApprove, nil); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}
	data, err = os.ReadFile(filepath.Join(workDir, filepath.FromSlash(*event.ReviewFilePath)))
	if err != nil {
		t.Fatalf("read processed review file: %v", err)
	}
	text = string(data)
	for _, want := range []string{"processed_action: approve", "action: accept"} {
		if !strings.Contains(text, want) {
			t.Fatalf("processed review file missing %q:\n%s", want, text)
		}
	}
}

func TestExecutor_ProcessReviewHandoffRequestsFix(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()
	initGitRepo(t, workDir)

	findings := `{"findings":[{"id":"review-1","severity":"error","description":"bug","suggested_fix":"patch bug","action":"auto-fix"},{"id":"review-2","severity":"warning","description":"warn","suggested_fix":"patch warn","action":"auto-fix"}],"summary":"2 issues"}`
	var previousFindings string
	callCount := 0
	step := &adaptiveCallStep{
		name: types.StepReview,
		fn: func(sctx *StepContext) (*StepOutcome, error) {
			callCount++
			if callCount == 1 {
				return &StepOutcome{NeedsApproval: true, Findings: findings}, nil
			}
			previousFindings = sctx.PreviousFindings
			return &StepOutcome{ExitCode: 0}, nil
		},
	}
	exec := NewExecutor(database, p, nil, nil, []Step{step}, nil)
	events := collectEvents(exec)

	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()

	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	event := waitForStepEvent(t, events, ipc.EventStepCompleted, types.StepReview)
	if event.ReviewFilePath == nil {
		t.Fatal("expected review file path")
	}
	reviewPath := filepath.Join(workDir, filepath.FromSlash(*event.ReviewFilePath))
	if err := exec.ProcessReview(types.StepReview); err != nil {
		t.Fatalf("process review: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}

	if !strings.Contains(previousFindings, "review-1") || !strings.Contains(previousFindings, "review-2") {
		t.Fatalf("expected both findings selected for fix, got %s", previousFindings)
	}
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("read processed review file: %v", err)
	}
	for _, want := range []string{"processed_action: approve", "## Prior Decisions", reviewhandoff.FinalNoFindingsText} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("processed review file missing %q:\n%s", want, data)
		}
	}
}

func TestExecutor_ProcessReviewPersistsReviewActions(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()
	initGitRepo(t, workDir)

	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"skip this","action":"ask-user"}],"summary":"1 issue"}`
	step := newApprovalStep(types.StepReview, findings)
	exec := NewExecutor(database, p, nil, nil, []Step{step}, nil)
	events := collectEvents(exec)

	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()

	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	event := waitForStepEvent(t, events, ipc.EventStepCompleted, types.StepReview)
	if event.ReviewFilePath == nil {
		t.Fatal("expected review file path")
	}
	reviewPath := filepath.Join(workDir, filepath.FromSlash(*event.ReviewFilePath))
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "action: accept", "action: skip", 1)
	if edited == string(data) {
		t.Fatalf("expected editable accept response in review file:\n%s", data)
	}
	if err := os.WriteFile(reviewPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.ProcessReview(types.StepReview); err != nil {
		t.Fatalf("process review: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}

	steps, err := database.GetStepsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one step, got %d", len(steps))
	}
	rounds, err := database.GetRoundsByStep(steps[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rounds) != 1 || rounds[0].UserFindingsJSON == nil {
		t.Fatalf("expected persisted user findings with review actions, rounds=%d", len(rounds))
	}
	for _, want := range []string{`"review_actions"`, `"review-1":"skip"`} {
		if !strings.Contains(*rounds[0].UserFindingsJSON, want) {
			t.Fatalf("persisted user findings missing %q: %s", want, *rounds[0].UserFindingsJSON)
		}
	}
}

func TestExecutor_ProcessReviewValidationErrorKeepsGateOpen(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()
	initGitRepo(t, workDir)

	findings := `{"findings":[{"id":"review-1","severity":"error","description":"bug","action":"auto-fix"}],"summary":"1 issue"}`
	step := newApprovalStep(types.StepReview, findings)
	exec := NewExecutor(database, p, nil, nil, []Step{step}, nil)
	events := collectEvents(exec)

	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()

	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	event := waitForStepEvent(t, events, ipc.EventStepCompleted, types.StepReview)
	if event.ReviewFilePath == nil {
		t.Fatal("expected review file path")
	}
	reviewPath := filepath.Join(workDir, filepath.FromSlash(*event.ReviewFilePath))
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(data), "processed_action: pending", "processed_action: approve", 1)
	if err := os.WriteFile(reviewPath, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	err = exec.ProcessReview(types.StepReview)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "processed_action must be pending") {
		t.Fatalf("unexpected error: %v", err)
	}
	validationEvent := events.findLast(ipc.EventStepCompleted, string(types.StepStatusAwaitingApproval))
	if validationEvent == nil || validationEvent.ReviewValidationError == nil {
		t.Fatal("expected validation error event")
	}

	if err := exec.Respond(types.StepReview, types.ActionApprove, nil); err == nil {
		t.Fatal("expected invalid review file to block legacy approval")
	}
	if err := os.WriteFile(reviewPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Respond(types.StepReview, types.ActionApprove, nil); err != nil {
		t.Fatalf("expected gate to proceed after file repair: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}
}

func TestExecutor_LegacyFixMirrorsFileAndWritesFinalAudit(t *testing.T) {
	database, p, run, repo := setupTest(t)
	workDir := t.TempDir()
	initGitRepo(t, workDir)

	findings := `{"findings":[{"id":"review-1","severity":"error","description":"bug","suggested_fix":"patch bug","action":"auto-fix"},{"id":"review-2","severity":"warning","description":"warn","suggested_fix":"patch warn","action":"auto-fix"}],"summary":"2 issues"}`
	var previousFindings string
	callCount := 0
	step := &adaptiveCallStep{
		name: types.StepReview,
		fn: func(sctx *StepContext) (*StepOutcome, error) {
			callCount++
			if callCount == 1 {
				return &StepOutcome{NeedsApproval: true, Findings: findings}, nil
			}
			previousFindings = sctx.PreviousFindings
			return &StepOutcome{ExitCode: 0, FixSummary: "patched bug"}, nil
		},
	}
	exec := NewExecutor(database, p, nil, nil, []Step{step}, nil)
	events := collectEvents(exec)

	done := make(chan error, 1)
	go func() {
		done <- exec.Execute(context.Background(), run, repo, workDir)
	}()

	waitForStepStatus(t, database, run.ID, types.StepReview, types.StepStatusAwaitingApproval)
	event := waitForStepEvent(t, events, ipc.EventStepCompleted, types.StepReview)
	if event.ReviewFilePath == nil {
		t.Fatal("expected review file path")
	}
	reviewPath := filepath.Join(workDir, filepath.FromSlash(*event.ReviewFilePath))
	if err := exec.RespondWithOverrides(types.StepReview, types.ActionFix, []string{"review-1"}, map[string]string{"review-1": "apply a focused patch"}, nil); err != nil {
		t.Fatalf("respond fix: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("executor timed out")
	}

	if !strings.Contains(previousFindings, "review-1") || strings.Contains(previousFindings, "review-2") {
		t.Fatalf("expected only review-1 sent to fixer, got %s", previousFindings)
	}
	data, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("read final review file: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"processed_action: approve",
		"Total findings: 0",
		"## Prior Decisions",
		"## Final State",
		reviewhandoff.FinalNoFindingsText,
		"- Action: fix",
		"- Solution: apply a focused patch",
		"- Fix summary: patched bug",
		"### review-2",
		"- Action: accept",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("final audit missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "## Findings") {
		t.Fatalf("final audit should not render current findings:\n%s", text)
	}
	steps, err := database.GetStepsByRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	rounds, err := database.GetRoundsByStep(steps[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rounds) == 0 || rounds[0].UserFindingsJSON == nil {
		t.Fatal("expected persisted user findings for the fixed review round")
	}
	for _, want := range []string{`"review_actions"`, `"review-1":"fix"`, `"review-2":"accept"`} {
		if !strings.Contains(*rounds[0].UserFindingsJSON, want) {
			t.Fatalf("persisted user findings missing %q: %s", want, *rounds[0].UserFindingsJSON)
		}
	}
}
