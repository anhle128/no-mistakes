package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/telemetry"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// --- RunManager integration tests ---

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

func TestStartRunCurrentWorktreeExecutesInCurrentDir(t *testing.T) {
	step := &mockPassStep{name: types.StepReview}
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{step}
	})

	repo, mainSHA := setupTestGitRepo(t, p, d, "current-worktree-repo")
	headSHA := commitFeatureBranch(t, repo.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.StartRunResult
	err = client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:                 repo.ID,
		Branch:                 "feature/current",
		HeadSHA:                headSHA,
		BaseSHA:                headSHA, // ignored: daemon derives the trustworthy base.
		WorktreeMode:           types.WorktreeModeCurrent,
		WorkDir:                repo.WorkingPath,
		WorkDirLabel:           "caller supplied label",
		CurrentWorktreeWarning: "caller supplied warning",
		ReviewBaseRef:          "caller supplied base ref",
	}, &result)
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID == "" {
		t.Fatal("expected run ID")
	}

	run := waitForRunTerminalState(t, d, result.RunID)
	if run.Status != types.RunCompleted {
		t.Fatalf("run status = %q, want completed", run.Status)
	}
	if run.WorktreeMode != types.WorktreeModeCurrent {
		t.Fatalf("worktree mode = %q, want current", run.WorktreeMode)
	}
	wantWorkDir, err := filepath.EvalSymlinks(repo.WorkingPath)
	if err != nil {
		t.Fatal(err)
	}
	if run.WorkDir == nil || *run.WorkDir != wantWorkDir {
		t.Fatalf("work dir = %v, want %s", run.WorkDir, wantWorkDir)
	}
	if run.BaseSHA != mainSHA {
		t.Fatalf("base SHA = %s, want daemon-derived main SHA %s", run.BaseSHA, mainSHA)
	}
	if run.WorkDirLabel == nil || *run.WorkDirLabel == "caller supplied label" {
		t.Fatalf("work dir label should be daemon-derived, got %v", run.WorkDirLabel)
	}
	if run.CurrentWorktreeWarning == nil || *run.CurrentWorktreeWarning == "caller supplied warning" {
		t.Fatalf("current warning should be daemon-derived, got %v", run.CurrentWorktreeWarning)
	}
	if run.ReviewBaseRef == nil || *run.ReviewBaseRef == "caller supplied base ref" {
		t.Fatalf("review base ref should be daemon-derived, got %v", run.ReviewBaseRef)
	}
	if _, err := os.Stat(p.WorktreeDir(repo.ID, result.RunID)); !os.IsNotExist(err) {
		t.Fatalf("managed worktree should not exist for current run, stat err=%v", err)
	}
	if _, err := os.Stat(repo.WorkingPath); err != nil {
		t.Fatalf("current worktree should remain: %v", err)
	}
}

func TestStartRunCurrentWorktreeResumesCompatibleActiveRun(t *testing.T) {
	started := make(chan string, 1)
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&notifyBlockStep{name: types.StepReview, started: started}}
	})

	repo, mainSHA := setupTestGitRepo(t, p, d, "current-resume-repo")
	headSHA := commitFeatureBranch(t, repo.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	params := &ipc.StartRunParams{
		RepoID:                 repo.ID,
		Branch:                 "feature/current",
		HeadSHA:                headSHA,
		BaseSHA:                mainSHA,
		WorktreeMode:           types.WorktreeModeCurrent,
		WorkDir:                repo.WorkingPath,
		WorkDirLabel:           "work",
		CurrentWorktreeWarning: "uses this checkout; fixes may modify it",
		ReviewBaseRef:          "origin/main",
		Intent:                 "original user goal",
	}
	var first ipc.StartRunResult
	if err := client.Call(ipc.MethodStartRun, params, &first); err != nil {
		t.Fatal(err)
	}
	waitForStartedBranch(t, started, "feature/current")

	var second ipc.StartRunResult
	if err := client.Call(ipc.MethodStartRun, params, &second); err != nil {
		t.Fatal(err)
	}
	if !second.Resumed {
		t.Fatal("expected compatible current run to resume")
	}
	if second.RunID != first.RunID {
		t.Fatalf("resumed run = %s, want %s", second.RunID, first.RunID)
	}
}

func TestStartRunCurrentWorktreeResumesActiveRunBeforeDirtyPreflight(t *testing.T) {
	started := make(chan string, 1)
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&notifyBlockStep{name: types.StepReview, started: started}}
	})

	repo, _ := setupTestGitRepo(t, p, d, "current-dirty-resume-repo")
	headSHA := commitFeatureBranch(t, repo.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var first ipc.StartRunResult
	if err := client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:       repo.ID,
		Branch:       "feature/current",
		HeadSHA:      headSHA,
		WorktreeMode: types.WorktreeModeCurrent,
		WorkDir:      repo.WorkingPath,
		Intent:       "original user goal",
	}, &first); err != nil {
		t.Fatal(err)
	}
	waitForStartedBranch(t, started, "feature/current")
	if err := os.WriteFile(filepath.Join(repo.WorkingPath, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var second ipc.StartRunResult
	if err := client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:        repo.ID,
		Branch:        "feature/current",
		HeadSHA:       headSHA,
		WorktreeMode:  types.WorktreeModeCurrent,
		WorkDir:       repo.WorkingPath,
		RequireIntent: true,
	}, &second); err != nil {
		t.Fatal(err)
	}
	if !second.Resumed {
		t.Fatal("expected compatible active run to resume before dirty preflight")
	}
	if second.RunID != first.RunID {
		t.Fatalf("resumed run = %s, want %s", second.RunID, first.RunID)
	}
}

func TestStartRunCurrentWorktreeRequiresIntentBeforeRunCreation(t *testing.T) {
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&mockPassStep{name: types.StepReview}}
	})

	repo, _ := setupTestGitRepo(t, p, d, "current-missing-intent-repo")
	headSHA := commitFeatureBranch(t, repo.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.StartRunResult
	err = client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:        repo.ID,
		Branch:        "feature/current",
		HeadSHA:       headSHA,
		WorktreeMode:  types.WorktreeModeCurrent,
		WorkDir:       repo.WorkingPath,
		RequireIntent: true,
	}, &result)
	if err == nil {
		t.Fatal("expected current worktree start without required intent to be rejected")
	}
	if !strings.Contains(err.Error(), types.RejectionMissingIntent) && !strings.Contains(err.Error(), "--intent") {
		t.Fatalf("error = %v, want missing-intent rejection", err)
	}
	runs, err := d.GetRunsByRepo(repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("missing intent rejection should not create a recoverable run, got %d", len(runs))
	}
}

func TestStartRunCurrentWorktreeRejectsDirtyBeforeRunCreation(t *testing.T) {
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&mockPassStep{name: types.StepReview}}
	})

	repo, _ := setupTestGitRepo(t, p, d, "current-dirty-repo")
	headSHA := commitFeatureBranch(t, repo.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")
	if err := os.WriteFile(filepath.Join(repo.WorkingPath, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.StartRunResult
	err = client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:       repo.ID,
		Branch:       "feature/current",
		HeadSHA:      headSHA,
		WorktreeMode: types.WorktreeModeCurrent,
		WorkDir:      repo.WorkingPath,
	}, &result)
	if err == nil {
		t.Fatal("expected dirty current worktree to be rejected")
	}
	if !strings.Contains(err.Error(), types.RejectionDirtyWorktree) && !strings.Contains(err.Error(), "not clean") {
		t.Fatalf("error = %v, want dirty-worktree rejection", err)
	}
	runs, err := d.GetRunsByRepo(repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("dirty rejection should not create a recoverable run, got %d", len(runs))
	}
}

func TestStartRunCurrentWorktreeRejectsMismatchedRepoBeforeRunCreation(t *testing.T) {
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&mockPassStep{name: types.StepReview}}
	})

	repoA, _ := setupTestGitRepo(t, p, d, "current-repo-a")
	repoB, _ := setupTestGitRepo(t, p, d, "current-repo-b")
	headSHA := commitFeatureBranch(t, repoB.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var result ipc.StartRunResult
	err = client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:       repoA.ID,
		Branch:       "feature/current",
		HeadSHA:      headSHA,
		WorktreeMode: types.WorktreeModeCurrent,
		WorkDir:      repoB.WorkingPath,
	}, &result)
	if err == nil {
		t.Fatal("expected mismatched current worktree repo to be rejected")
	}
	if !strings.Contains(err.Error(), types.RejectionRepoMismatch) && !strings.Contains(err.Error(), "does not belong") {
		t.Fatalf("error = %v, want repo-mismatch rejection", err)
	}
	runs, err := d.GetRunsByRepo(repoA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 0 {
		t.Fatalf("repo mismatch should not create a recoverable run, got %d", len(runs))
	}
}

func TestPushReceivedRejectsInsteadOfCancellingActiveCurrentRun(t *testing.T) {
	started := make(chan string, 1)
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&notifyBlockStep{name: types.StepReview, started: started}}
	})

	repo, mainSHA := setupTestGitRepo(t, p, d, "current-conflict-repo")
	headSHA := commitFeatureBranch(t, repo.WorkingPath, "feature/current", "current.txt", "current change\n", "add current change")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var current ipc.StartRunResult
	if err := client.Call(ipc.MethodStartRun, &ipc.StartRunParams{
		RepoID:       repo.ID,
		Branch:       "feature/current",
		HeadSHA:      headSHA,
		BaseSHA:      mainSHA,
		WorktreeMode: types.WorktreeModeCurrent,
		WorkDir:      repo.WorkingPath,
	}, &current); err != nil {
		t.Fatal(err)
	}
	waitForStartedBranch(t, started, "feature/current")

	gitCmd(t, repo.WorkingPath, "push", "gate", "HEAD:refs/heads/feature/current")
	var push ipc.PushReceivedResult
	err = client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir(repo.ID),
		Ref:  "refs/heads/feature/current",
		Old:  mainSHA,
		New:  headSHA,
	}, &push)
	if err == nil {
		t.Fatal("expected isolated push start to reject active current run")
	}
	if !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("error = %v, want incompatible active run", err)
	}
	run, err := d.GetRun(current.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != types.RunRunning {
		t.Fatalf("active current run status = %s, want still running", run.Status)
	}
}

func commitFeatureBranch(t *testing.T, workDir, branch, relPath, content, message string) string {
	t.Helper()
	gitCmd(t, workDir, "branch", "-f", "main", "HEAD")
	gitCmd(t, workDir, "checkout", "-b", branch, "main")
	fullPath := filepath.Join(workDir, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, workDir, "add", relPath)
	gitCmd(t, workDir, "commit", "-m", message)
	return gitOutput(t, workDir, "rev-parse", "HEAD")
}

func TestPushReceivedAllowsDifferentBranchRunsConcurrently(t *testing.T) {
	started := make(chan string, 2)
	p, d := startTestDaemonWithSteps(t, func() []pipeline.Step {
		return []pipeline.Step{&notifyBlockStep{name: types.StepReview, started: started}}
	})

	_, headSHA := setupTestGitRepo(t, p, d, "concurrent-branch-repo")

	client, err := ipc.Dial(p.Socket())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var first ipc.PushReceivedResult
	if err := client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("concurrent-branch-repo"),
		Ref:  "refs/heads/feature/one",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &first); err != nil {
		t.Fatal(err)
	}
	waitForStartedBranch(t, started, "feature/one")

	var second ipc.PushReceivedResult
	if err := client.Call(ipc.MethodPushReceived, &ipc.PushReceivedParams{
		Gate: p.RepoDir("concurrent-branch-repo"),
		Ref:  "refs/heads/feature/two",
		Old:  "0000000000000000000000000000000000000000",
		New:  headSHA,
	}, &second); err != nil {
		t.Fatal(err)
	}
	waitForStartedBranch(t, started, "feature/two")

	for _, tc := range []struct {
		branch string
		runID  string
	}{
		{branch: "feature/one", runID: first.RunID},
		{branch: "feature/two", runID: second.RunID},
	} {
		active, err := d.GetActiveRun("concurrent-branch-repo", tc.branch)
		if err != nil {
			t.Fatalf("get active run for %s: %v", tc.branch, err)
		}
		if active == nil {
			t.Fatalf("expected active run for %s", tc.branch)
		}
		if active.ID != tc.runID {
			t.Fatalf("active run for %s = %s, want %s", tc.branch, active.ID, tc.runID)
		}
		if active.Status != types.RunRunning {
			t.Fatalf("active run for %s status = %s, want running", tc.branch, active.Status)
		}
	}
}

type notifyBlockStep struct {
	name    types.StepName
	started chan<- string
}

func (s *notifyBlockStep) Name() types.StepName { return s.name }

func (s *notifyBlockStep) Execute(sctx *pipeline.StepContext) (*pipeline.StepOutcome, error) {
	select {
	case s.started <- sctx.Run.Branch:
	default:
	}
	<-sctx.Ctx.Done()
	return nil, sctx.Ctx.Err()
}

func waitForStartedBranch(t *testing.T, started <-chan string, branch string) {
	t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case got := <-started:
			if got == branch {
				return
			}
		case <-timeout:
			t.Fatalf("run for branch %s did not start", branch)
		}
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
