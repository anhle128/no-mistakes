package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/boundary"
	"github.com/kunchenguid/no-mistakes/internal/config"
	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/telemetry"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// --- RunManager integration tests ---

func TestGateIdentityIgnoresClientMetadataAndUsesLatestRound(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc", "def")
	if err != nil {
		t.Fatal(err)
	}
	step, err := d.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"same"}],"summary":"1"}`
	if err := d.SetStepFindings(step.ID, findings); err != nil {
		t.Fatal(err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusAwaitingApproval); err != nil {
		t.Fatal(err)
	}
	if _, err := d.InsertStepRound(step.ID, 1, "initial", &findings, nil, 100); err != nil {
		t.Fatal(err)
	}

	mgr := &RunManager{db: d}
	meta := types.DecisionMetadata{GateID: "client-review", GateFingerprint: "client-fingerprint"}
	gateID, first := mgr.gateIdentity(run.ID, types.StepReview, meta)
	if gateID != string(types.StepReview) {
		t.Fatalf("gateID = %q, want server step name", gateID)
	}
	version, err := d.StepGateVersion(step.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := boundary.GateFingerprint(run.ID, types.StepReview, types.StepStatusAwaitingApproval, version, findings)
	if first != want {
		t.Fatalf("fingerprint = %q, want server-computed %q", first, want)
	}
	if first == meta.GateFingerprint {
		t.Fatal("fingerprint should not trust client metadata")
	}
	repeatGateID, repeat := mgr.gateIdentity(run.ID, types.StepReview, types.DecisionMetadata{GateFingerprint: "different-client-fingerprint"})
	if repeatGateID != gateID || repeat != first {
		t.Fatalf("same server gate should be stable, got gate=%q fp=%q want gate=%q fp=%q", repeatGateID, repeat, gateID, first)
	}

	if _, err := d.InsertStepRound(step.ID, 2, "auto_fix", &findings, nil, 100); err != nil {
		t.Fatal(err)
	}
	_, second := mgr.gateIdentity(run.ID, types.StepReview, meta)
	if second == first {
		t.Fatal("same findings on a later round should produce a distinct gate fingerprint")
	}
}

func TestPushReceivedTracksRunTelemetry(t *testing.T) {
	recorder := &telemetryRecorder{}
	restore := telemetry.SetDefaultForTesting(recorder)
	defer restore()

	step := &mockPassStep{name: types.StepReview}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{step}
	})

	_, headSHA := setupTestGitRepo(t, p, d, "telemetry-run-repo")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("telemetry-run-repo"),
		Ref:  "refs/heads/main",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &result)
	if err != nil {
		t.Fatal(err)
	}

	run := waitForRunTerminalState(t, d, result.RunID)
	if run.Status != types.RunCompleted {
		t.Fatalf("run status = %q, want %q", run.Status, types.RunCompleted)
	}

	started := recorder.find("run", "action", "started")
	if started == nil {
		t.Fatal("expected run started telemetry event")
	}
	if got := started.fields["trigger"]; got != "push" {
		t.Fatalf("started trigger = %v, want push", got)
	}
	if got := started.fields["agent"]; got != string(types.AgentClaude) {
		t.Fatalf("started agent = %v, want %q", got, types.AgentClaude)
	}
	if got := started.fields["branch_role"]; got != "default" {
		t.Fatalf("started branch_role = %v, want default", got)
	}

	finished := recorder.find("run", "action", "finished")
	if finished == nil {
		t.Fatal("expected run finished telemetry event")
	}
	if got := finished.fields["status"]; got != string(types.RunCompleted) {
		t.Fatalf("finished status = %v, want %q", got, types.RunCompleted)
	}
	if _, ok := finished.fields["duration_ms"]; !ok {
		t.Fatal("expected duration_ms in run finished telemetry")
	}
}

func TestPushReceivedSkipStepsConfiguresExecutor(t *testing.T) {
	review := &mockPassStep{name: types.StepReview}
	testStep := &mockPassStep{name: types.StepTest}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{review, testStep}
	})

	_, headSHA := setupTestGitRepo(t, p, d, "skip-run-repo")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate:      p.RepoDir("skip-run-repo"),
		Ref:       "refs/heads/main",
		Old:       "0000000000000000000000000000000000000000",
		New:       headSHA,
		SkipSteps: []types.StepName{types.StepReview},
	}, &result)
	if err != nil {
		t.Fatal(err)
	}

	run := waitForRunTerminalState(t, d, result.RunID)
	if run.Status != types.RunCompleted {
		t.Fatalf("run status = %q, want %q", run.Status, types.RunCompleted)
	}
	if got := review.execCnt.Load(); got != 0 {
		t.Fatalf("review executed %d times, want 0", got)
	}
	if got := testStep.execCnt.Load(); got != 1 {
		t.Fatalf("test executed %d times, want 1", got)
	}
	steps, err := d.GetStepsByRun(result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		if step.StepName == types.StepReview && step.Status != types.StepStatusSkipped {
			t.Fatalf("review status = %s, want %s", step.Status, types.StepStatusSkipped)
		}
	}
}

func TestRespondWithUnattendedMetadataWithholdsUnknownBoundary(t *testing.T) {
	review := &mockApprovalStep{name: types.StepReview}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{review}
	})

	repoID := "withheld-yolo-repo"
	_, headSHA := setupTestGitRepo(t, p, d, repoID)

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var pushResult ipc.PushReceivedResult
	if err := client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir(repoID),
		Ref:  "refs/heads/main",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &pushResult); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		steps, _ := d.GetStepsByRun(pushResult.RunID)
		for _, s := range steps {
			if s.StepName == types.StepReview && s.Status == types.StepStatusAwaitingApproval {
				goto awaiting
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("step never reached awaiting_approval")

awaiting:
	if err := os.RemoveAll(p.WorktreeDir(repoID, pushResult.RunID)); err != nil {
		t.Fatal(err)
	}

	var respondResult ipc.RespondResult
	err = client.Call(ipc.MethodRespond, &ipc.RespondParams{
		RunID:           pushResult.RunID,
		Step:            types.StepReview,
		Action:          types.ActionApprove,
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorAgent,
		ApprovalSurface: types.ApprovalSurfaceAXI,
		ConsentMode:     types.ConsentModeYes,
		GateID:          "review",
		GateFingerprint: "fp",
	}, &respondResult)
	if err == nil {
		t.Fatal("expected unattended response to be withheld")
	}
	if !strings.Contains(err.Error(), "withheld") {
		t.Fatalf("error = %v, want withheld reason", err)
	}

	steps, err := d.GetStepsByRun(pushResult.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) == 0 || steps[0].Status != types.StepStatusAwaitingApproval {
		t.Fatalf("step status = %+v, want still awaiting approval", steps)
	}
	events, err := d.GetRunEvents(pushResult.RunID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range events {
		if event.EventType == db.RunEventGateAutomationWithheld && event.Status == types.GateAutomationWithheld {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected withheld gate automation event, got %+v", events)
	}
}

func TestRespondWithUnattendedMetadataFailsWhenAutomationEventCannotPersist(t *testing.T) {
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	repo, err := d.InsertRepo("/home/user/project", "git@github.com:user/project.git", "main")
	if err != nil {
		t.Fatal(err)
	}
	run, err := d.InsertRun(repo.ID, "feature", "abc", "def")
	if err != nil {
		t.Fatal(err)
	}
	step, err := d.InsertStepResult(run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusAwaitingApproval); err != nil {
		t.Fatal(err)
	}
	p := paths.WithRoot(t.TempDir())
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	mgr := &RunManager{
		db:        d,
		paths:     p,
		executors: map[string]*pipeline.Executor{run.ID: pipeline.NewExecutor(d, p, &config.Config{}, nil, nil, nil)},
	}
	if err := d.DeleteRepo(repo.ID); err != nil {
		t.Fatal(err)
	}

	err = mgr.HandleRespondWithMetadata(run.ID, types.StepReview, types.ActionApprove, nil, nil, nil, types.DecisionMetadata{
		DecisionSource:  types.DecisionSourceUnattended,
		ActorType:       types.ActorAgent,
		ApprovalSurface: types.ApprovalSurfaceAXI,
		ConsentMode:     types.ConsentModeYes,
	})
	if err == nil {
		t.Fatal("expected automation audit persistence failure")
	}
	if !strings.Contains(err.Error(), "record gate automation event") {
		t.Fatalf("error = %v, want audit persistence failure before executor response", err)
	}
	if strings.Contains(err.Error(), "no step awaiting approval") {
		t.Fatalf("error = %v, response reached executor instead of failing closed on audit persistence", err)
	}
}

func TestRerunSkipStepsConfiguresExecutor(t *testing.T) {
	review := &mockPassStep{name: types.StepReview}
	testStep := &mockPassStep{name: types.StepTest}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{review, testStep}
	})

	_, headSHA := setupTestGitRepo(t, p, d, "skip-rerun-repo")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var first ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("skip-rerun-repo"),
		Ref:  "refs/heads/main",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &first)
	if err != nil {
		t.Fatal(err)
	}
	waitForRunTerminalState(t, d, first.RunID)

	var second ipc.RerunResult
	err = client.Call(ipc.MethodRerun, &ipc.RerunParams{
		RepoID:    "skip-rerun-repo",
		Branch:    "main",
		SkipSteps: []types.StepName{types.StepReview},
	}, &second)
	if err != nil {
		t.Fatal(err)
	}
	waitForRunTerminalState(t, d, second.RunID)

	if got := review.execCnt.Load(); got != 1 {
		t.Fatalf("review executed %d times, want 1", got)
	}
	if got := testStep.execCnt.Load(); got != 2 {
		t.Fatalf("test executed %d times, want 2", got)
	}
	steps, err := d.GetStepsByRun(second.RunID)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		if step.StepName == types.StepReview && step.Status != types.StepStatusSkipped {
			t.Fatalf("review status = %s, want %s", step.Status, types.StepStatusSkipped)
		}
	}
}

func TestPushReceivedReturnsBeforeIntentSummarization(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)
	t.Setenv("USERPROFILE", fakeHome)

	step := &mockPassStep{name: types.StepReview}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{step}
	})

	slowClaude := writeSlowMockClaude(t, t.TempDir())
	if err := os.WriteFile(p.ConfigFile(), []byte("agent: claude\nagent_path_override:\n  claude: "+slowClaude+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repo, headSHA := setupTestGitRepo(t, p, d, "intent-start-run-repo")
	writeManagerClaudeFixture(t, fakeHome, repo.WorkingPath, []string{
		`{"type":"user","cwd":` + testJSONString(t, repo.WorkingPath) + `,"timestamp":"2026-04-18T02:15:37.407Z","uuid":"u1","sessionId":"s1","message":{"role":"user","content":"please update test.txt"}}`,
	})

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	started := time.Now()
	var result ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("intent-start-run-repo"),
		Ref:  "refs/heads/main",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 2500*time.Millisecond {
		t.Fatalf("PushReceived took %s, want under 2.5s", elapsed)
	}
	if result.RunID == "" {
		t.Fatal("expected non-empty run ID")
	}

	waitForRunTerminalState(t, d, result.RunID)
}

func writeManagerClaudeFixture(t *testing.T, home, repoCWD string, lines []string) {
	t.Helper()
	encoded := testClaudeProjectDirName(repoCWD)
	dir := filepath.Join(home, ".claude", "projects", encoded)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session-uuid-1.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPushReceivedTracksRunTelemetryAfterPanic(t *testing.T) {
	recorder := &telemetryRecorder{}
	restore := telemetry.SetDefaultForTesting(recorder)
	defer restore()

	step := &mockPanicStep{name: types.StepReview}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{step}
	})

	_, headSHA := setupTestGitRepo(t, p, d, "telemetry-panic-repo")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("telemetry-panic-repo"),
		Ref:  "refs/heads/main",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &result)
	if err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		run, err := d.GetRun(result.RunID)
		if err != nil {
			t.Fatal(err)
		}
		if run != nil && run.Error != nil && strings.Contains(*run.Error, "internal panic") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	finished := recorder.find("run", "action", "finished")
	if finished == nil {
		t.Fatal("expected run finished telemetry event after panic")
	}
	if got := finished.fields["status"]; got != string(types.RunFailed) {
		t.Fatalf("finished status = %v, want %q", got, types.RunFailed)
	}
	if _, ok := finished.fields["duration_ms"]; !ok {
		t.Fatal("expected duration_ms in run finished telemetry after panic")
	}
}

func TestPushReceivedDemoModeBypassesAgentResolution(t *testing.T) {
	t.Setenv("NM_DEMO", "1")

	step := &mockPassStep{name: types.StepReview}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{step}
	})

	if err := os.WriteFile(p.ConfigFile(), []byte("agent: claude\nagent_path_override:\n  claude: /path/that/does/not/exist\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, headSHA := setupTestGitRepo(t, p, d, "testrepo-demo")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("testrepo-demo"),
		Ref:  "refs/heads/main",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID == "" {
		t.Fatal("expected non-empty run ID")
	}

	waitForRunTerminalState(t, d, result.RunID)
	run, err := d.GetRun(result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != types.RunCompleted {
		var runErr string
		if run.Error != nil {
			runErr = *run.Error
		}
		t.Fatalf("run status = %q, want %q (error: %s)", run.Status, types.RunCompleted, runErr)
	}
	if step.execCnt.Load() == 0 {
		t.Error("mock step was never executed")
	}
}
