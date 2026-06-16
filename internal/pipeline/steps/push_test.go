package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/config"
	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/reviewhandoff"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestPushStep_ReconcilesStaleDatabaseHeadSHA(t *testing.T) {
	t.Parallel()
	// When push retries after a prior UpdateRunHeadSHA failure, there are no
	// uncommitted changes. The step must still reconcile the DB if HeadSHA is stale.
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")

	dir := t.TempDir()
	gitCmd(t, dir, "init")
	gitCmd(t, dir, "config", "user.name", "test")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "checkout", "-b", "main")
	os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init"), 0o644)
	gitCmd(t, dir, "add", "-A")
	gitCmd(t, dir, "commit", "-m", "initial")
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")

	gitCmd(t, dir, "checkout", "-b", "feature")
	os.WriteFile(filepath.Join(dir, "feature.txt"), []byte("feature"), 0o644)
	gitCmd(t, dir, "add", "-A")
	gitCmd(t, dir, "commit", "-m", "feature")
	actualHeadSHA := gitCmd(t, dir, "rev-parse", "HEAD")
	baseSHA := gitCmd(t, dir, "rev-parse", "main")
	gitCmd(t, dir, "push", "origin", "feature")

	// Create context with a stale HeadSHA (simulates prior DB write failure)
	staleHeadSHA := baseSHA // intentionally wrong
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, staleHeadSHA, config.Commands{})
	sctx.Repo.UpstreamURL = upstream

	step := &PushStep{}
	_, err := step.Execute(sctx)
	if err != nil {
		t.Fatal(err)
	}

	// In-memory HeadSHA must match actual HEAD
	if sctx.Run.HeadSHA != actualHeadSHA {
		t.Errorf("Run.HeadSHA = %s, want %s", sctx.Run.HeadSHA, actualHeadSHA)
	}

	// DB record must also be updated
	dbRun, err := sctx.DB.GetRun(sctx.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dbRun.HeadSHA != actualHeadSHA {
		t.Errorf("DB HeadSHA = %s, want %s", dbRun.HeadSHA, actualHeadSHA)
	}
}

func TestPushStep_ForceAddsInRepoEvidenceArtifacts(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")

	dir := t.TempDir()
	gitCmd(t, dir, "init")
	gitCmd(t, dir, "config", "user.name", "test")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.png\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "-A")
	gitCmd(t, dir, "commit", "-m", "initial")
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")

	gitCmd(t, dir, "checkout", "-b", "feature")
	baseSHA := gitCmd(t, dir, "rev-parse", "main")
	headSHA := gitCmd(t, dir, "rev-parse", "HEAD")
	evidenceDir := filepath.Join(dir, "evidence", "feature")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "checkout.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Repo.UpstreamURL = upstream
	sctx.Run.Branch = "feature"
	sctx.Config.Test.Evidence = config.Evidence{StoreInRepo: true, Dir: "evidence"}

	step := &PushStep{}
	if _, err := step.Execute(sctx); err != nil {
		t.Fatal(err)
	}

	clone := t.TempDir()
	gitCmd(t, clone, "clone", "--branch", "feature", upstream, ".")
	if _, err := os.Stat(filepath.Join(clone, "evidence", "feature", "checkout.png")); err != nil {
		t.Fatalf("expected ignored evidence artifact to be pushed: %v", err)
	}
}

func TestPushStep_DoesNotForceAddIgnoredEvidenceDirectory(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("evidence/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".gitignore")
	gitCmd(t, dir, "commit", "-m", "ignore evidence")
	headSHA = gitCmd(t, dir, "rev-parse", "HEAD")
	evidenceDir := filepath.Join(dir, "evidence", "feature")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceDir, "stale.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Run.Branch = "feature"
	sctx.Config.Test.Evidence = config.Evidence{StoreInRepo: true, Dir: "evidence"}

	step := &PushStep{}
	if err := step.stageInRepoEvidence(sctx); err != nil {
		t.Fatal(err)
	}
	if status := gitStatusPorcelain(t, dir); status != "" {
		t.Fatalf("ignored evidence directory was staged: %q", status)
	}
}

func TestPushStep_DoesNotForceAddReviewHandoffFileWhenNotRequired(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".no-mistakes/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".gitignore")
	gitCmd(t, dir, "commit", "-m", "ignore handoff")
	headSHA = gitCmd(t, dir, "rev-parse", "HEAD")

	rel := filepath.Join(".no-mistakes", "issues", "feature", reviewhandoff.FileName("run-1"))
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte("review audit"), 0o644); err != nil {
		t.Fatal(err)
	}

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Run.ID = "run-1"
	step := &PushStep{}
	if err := step.stageReviewHandoffFiles(sctx); err != nil {
		t.Fatal(err)
	}
	if staged := gitCmd(t, dir, "diff", "--cached", "--name-only"); staged != "" {
		t.Fatalf("non-required review handoff was staged: %q", staged)
	}
}

func TestPushStep_BlocksMissingRequiredReviewAuditFile(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","action":"ask-user"}],"summary":"1"}`, "", "")

	step := &PushStep{}
	err := step.stageReviewHandoffFiles(sctx)
	if err == nil || !strings.Contains(err.Error(), "review audit file required but absent") {
		t.Fatalf("error = %v, want missing audit file blocker", err)
	}
}

func TestPushStep_BlocksStaleExistingReviewAuditWhenExpectedMissing(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","action":"ask-user"}],"summary":"1"}`, "", "")
	strayRel := filepath.Join("stray", reviewhandoff.FileName(sctx.Run.ID))
	writeReviewAuditFile(t, dir, strayRel, "other-run", sctx.Run.Branch, true)

	step := &PushStep{}
	err := step.stageReviewHandoffFiles(sctx)
	if err == nil || !strings.Contains(err.Error(), "run_id mismatch") {
		t.Fatalf("error = %v, want stale existing audit blocker", err)
	}
	if !strings.Contains(err.Error(), filepath.ToSlash(strayRel)) {
		t.Fatalf("error should identify stale existing audit path %s: %v", filepath.ToSlash(strayRel), err)
	}
}

func TestPushStep_StagesOnlyExpectedRequiredReviewAuditFile(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".no-mistakes/\nstray/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".gitignore")
	gitCmd(t, dir, "commit", "-m", "ignore review audits")
	headSHA = gitCmd(t, dir, "rev-parse", "HEAD")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","action":"ask-user"}],"summary":"1"}`, "", "")
	expectedRel := expectedFallbackReviewAuditRel(sctx)
	strayRel := filepath.Join("stray", reviewhandoff.FileName(sctx.Run.ID))
	writeReviewAuditFile(t, dir, expectedRel, sctx.Run.ID, sctx.Run.Branch, true)
	writeReviewAuditFile(t, dir, strayRel, sctx.Run.ID, sctx.Run.Branch, true)

	step := &PushStep{}
	if err := step.stageReviewHandoffFiles(sctx); err != nil {
		t.Fatal(err)
	}
	staged := gitCmd(t, dir, "diff", "--cached", "--name-only")
	if staged != filepath.ToSlash(expectedRel) {
		t.Fatalf("staged files = %q, want only %q", staged, filepath.ToSlash(expectedRel))
	}
}

func TestPushStep_ReusesExistingRequiredReviewAuditPath(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","action":"ask-user"}],"summary":"1"}`, "", "")

	existingRel := filepath.Join("docs", reviewhandoff.FileName(sctx.Run.ID))
	writeReviewAuditFile(t, dir, existingRel, sctx.Run.ID, sctx.Run.Branch, true)
	if err := os.MkdirAll(filepath.Join(dir, "new-anchor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new-anchor", "plan.md"), []byte("# newer anchor\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	step := &PushStep{}
	if err := step.stageReviewHandoffFiles(sctx); err != nil {
		t.Fatal(err)
	}
	staged := gitCmd(t, dir, "diff", "--cached", "--name-only")
	if staged != filepath.ToSlash(existingRel) {
		t.Fatalf("staged files = %q, want existing audit path %q", staged, filepath.ToSlash(existingRel))
	}
	if _, err := os.Stat(filepath.Join(dir, "new-anchor", reviewhandoff.FileName(sctx.Run.ID))); !os.IsNotExist(err) {
		t.Fatalf("push should not regenerate audit at new anchor, stat err=%v", err)
	}
}

func TestPushStep_BlocksUnprocessedExpectedReviewAuditFile(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","action":"ask-user"}],"summary":"1"}`, "", "")
	expectedRel := expectedFallbackReviewAuditRel(sctx)
	writeReviewAuditFile(t, dir, expectedRel, sctx.Run.ID, sctx.Run.Branch, false)

	step := &PushStep{}
	err := step.stageReviewHandoffFiles(sctx)
	if err == nil || !strings.Contains(err.Error(), "processed_at must be set") {
		t.Fatalf("error = %v, want unprocessed audit blocker", err)
	}
}

func TestPushStep_RegeneratesMissingRequiredReviewAuditFile(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".no-mistakes/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".gitignore")
	gitCmd(t, dir, "commit", "-m", "ignore regenerated audit")
	headSHA = gitCmd(t, dir, "rev-parse", "HEAD")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	stepResult, err := sctx.DB.InsertStepResult(sctx.Run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","suggested_fix":"patch it","action":"ask-user"}],"summary":"1"}`
	round1, err := sctx.DB.InsertStepRound(stepResult.ID, 1, "initial", &findings, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	selected := `["review-1"]`
	if err := sctx.DB.SetStepRoundSelection(round1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatal(err)
	}
	finalFindings := `{"findings":[],"summary":"0"}`
	fixSummary := "patched review issue"
	if _, err := sctx.DB.InsertStepRound(stepResult.ID, 2, "auto_fix", &finalFindings, &fixSummary, 20); err != nil {
		t.Fatal(err)
	}
	if err := sctx.DB.UpdateStepStatus(stepResult.ID, types.StepStatusCompleted); err != nil {
		t.Fatal(err)
	}

	expectedRel := expectedFallbackReviewAuditRel(sctx)
	step := &PushStep{}
	if err := step.stageReviewHandoffFiles(sctx); err != nil {
		t.Fatal(err)
	}
	staged := gitCmd(t, dir, "diff", "--cached", "--name-only")
	if staged != filepath.ToSlash(expectedRel) {
		t.Fatalf("staged files = %q, want regenerated audit %q", staged, filepath.ToSlash(expectedRel))
	}
	data, err := os.ReadFile(filepath.Join(dir, expectedRel))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"processed_action: approve", "## Prior Decisions", "review-1", "Action: fix", "Fix summary: patched review issue", reviewhandoff.FinalNoFindingsText} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("regenerated audit missing %q:\n%s", want, data)
		}
	}
}

func TestPushStep_RegeneratesMissingRequiredReviewAuditFileWithSkipAction(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".no-mistakes/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", ".gitignore")
	gitCmd(t, dir, "commit", "-m", "ignore regenerated audit")
	headSHA = gitCmd(t, dir, "rev-parse", "HEAD")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	stepResult, err := sctx.DB.InsertStepResult(sctx.Run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"skip me","suggested_fix":"none","action":"ask-user"},{"id":"review-2","severity":"warning","description":"fix me","suggested_fix":"patch it","action":"ask-user"}],"summary":"2"}`
	round1, err := sctx.DB.InsertStepRound(stepResult.ID, 1, "initial", &findings, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	selected := `["review-2"]`
	if err := sctx.DB.SetStepRoundSelection(round1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatal(err)
	}
	userFindings := `{"findings":[{"id":"review-2","severity":"warning","description":"fix me","suggested_fix":"patch it","user_instructions":"apply the focused patch","action":"ask-user"}],"summary":"1 selected","review_actions":{"review-1":"skip","review-2":"fix"}}`
	if err := sctx.DB.SetStepRoundUserFindings(round1.ID, &userFindings); err != nil {
		t.Fatal(err)
	}
	finalFindings := `{"findings":[],"summary":"0"}`
	fixSummary := "patched review-2"
	if _, err := sctx.DB.InsertStepRound(stepResult.ID, 2, "auto_fix", &finalFindings, &fixSummary, 20); err != nil {
		t.Fatal(err)
	}
	if err := sctx.DB.UpdateStepStatus(stepResult.ID, types.StepStatusCompleted); err != nil {
		t.Fatal(err)
	}

	expectedRel := expectedFallbackReviewAuditRel(sctx)
	step := &PushStep{}
	if err := step.stageReviewHandoffFiles(sctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, expectedRel))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{"### review-1", "Action: skip", "### review-2", "Action: fix", "Solution: apply the focused patch", "Fix summary: patched review-2"} {
		if !strings.Contains(text, want) {
			t.Fatalf("regenerated audit missing %q:\n%s", want, text)
		}
	}
}

func TestPushStep_BlocksAmbiguousRegeneratedReviewAuditActions(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	stepResult, err := sctx.DB.InsertStepResult(sctx.Run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"maybe accepted or skipped","action":"ask-user"},{"id":"review-2","severity":"warning","description":"fix me","suggested_fix":"patch it","action":"ask-user"}],"summary":"2"}`
	round1, err := sctx.DB.InsertStepRound(stepResult.ID, 1, "initial", &findings, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	selected := `["review-2"]`
	if err := sctx.DB.SetStepRoundSelection(round1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatal(err)
	}
	finalFindings := `{"findings":[],"summary":"0"}`
	if _, err := sctx.DB.InsertStepRound(stepResult.ID, 2, "auto_fix", &finalFindings, nil, 20); err != nil {
		t.Fatal(err)
	}
	if err := sctx.DB.UpdateStepStatus(stepResult.ID, types.StepStatusCompleted); err != nil {
		t.Fatal(err)
	}

	step := &PushStep{}
	err = step.stageReviewHandoffFiles(sctx)
	if err == nil || !strings.Contains(err.Error(), "missing persisted review action for review-1") {
		t.Fatalf("error = %v, want ambiguous persisted review action blocker", err)
	}
}

func TestPushStep_ExecuteStagesExpectedAuditWithoutAnchorOrStray(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")

	dir := t.TempDir()
	gitCmd(t, dir, "init")
	gitCmd(t, dir, "config", "user.name", "test")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "-A")
	gitCmd(t, dir, "commit", "-m", "initial")
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")
	gitCmd(t, dir, "checkout", "-b", "feature")

	baseSHA := gitCmd(t, dir, "rev-parse", "main")
	headSHA := gitCmd(t, dir, "rev-parse", "HEAD")
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Repo.UpstreamURL = upstream
	sctx.Run.Branch = "feature"

	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "plan.md"), []byte("# plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fixed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"needs decision","action":"ask-user"}],"summary":"1"}`, "", "")
	steps, err := sctx.DB.GetStepsByRun(sctx.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one review step, got %d", len(steps))
	}
	if err := sctx.DB.UpdateStepStatus(steps[0].ID, types.StepStatusCompleted); err != nil {
		t.Fatal(err)
	}
	expectedRel := filepath.Join("docs", reviewhandoff.FileName(sctx.Run.ID))
	strayRel := filepath.Join("stray", reviewhandoff.FileName(sctx.Run.ID))
	writeReviewAuditFile(t, dir, expectedRel, sctx.Run.ID, sctx.Run.Branch, true)
	writeReviewAuditFile(t, dir, strayRel, sctx.Run.ID, sctx.Run.Branch, true)

	step := &PushStep{}
	if _, err := step.Execute(sctx); err != nil {
		t.Fatal(err)
	}

	clone := t.TempDir()
	gitCmd(t, clone, "clone", "--branch", "feature", upstream, ".")
	for _, rel := range []string{expectedRel, "fix.txt"} {
		if _, err := os.Stat(filepath.Join(clone, rel)); err != nil {
			t.Fatalf("expected %s to be pushed: %v", rel, err)
		}
	}
	for _, rel := range []string{filepath.Join("docs", "plan.md"), strayRel} {
		if _, err := os.Stat(filepath.Join(clone, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected %s not to be pushed, stat err=%v", rel, err)
		}
	}
}

func TestPushStep_ExecuteDoesNotCommitStrayAuditWhenNotRequired(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")

	dir := t.TempDir()
	gitCmd(t, dir, "init")
	gitCmd(t, dir, "config", "user.name", "test")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "init.txt"), []byte("init"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitCmd(t, dir, "add", "-A")
	gitCmd(t, dir, "commit", "-m", "initial")
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")
	gitCmd(t, dir, "checkout", "-b", "feature")

	baseSHA := gitCmd(t, dir, "rev-parse", "main")
	headSHA := gitCmd(t, dir, "rev-parse", "HEAD")
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Repo.UpstreamURL = upstream
	sctx.Run.Branch = "feature"

	if err := os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fixed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"auto fix","action":"auto-fix"}],"summary":"1"}`, `["review-1"]`, db.RoundSelectionSourceAutoFix)
	steps, err := sctx.DB.GetStepsByRun(sctx.Run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one review step, got %d", len(steps))
	}
	if err := sctx.DB.UpdateStepStatus(steps[0].ID, types.StepStatusCompleted); err != nil {
		t.Fatal(err)
	}
	strayRel := filepath.Join("stray", reviewhandoff.FileName(sctx.Run.ID))
	writeReviewAuditFile(t, dir, strayRel, sctx.Run.ID, sctx.Run.Branch, true)

	step := &PushStep{}
	if _, err := step.Execute(sctx); err != nil {
		t.Fatal(err)
	}

	clone := t.TempDir()
	gitCmd(t, clone, "clone", "--branch", "feature", upstream, ".")
	if _, err := os.Stat(filepath.Join(clone, "fix.txt")); err != nil {
		t.Fatalf("expected ordinary fix file to be pushed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clone, strayRel)); !os.IsNotExist(err) {
		t.Fatalf("expected non-required stray audit not to be pushed, stat err=%v", err)
	}
}

func TestPushStep_AllowsAutoFixOnlyReviewWithoutAuditFile(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewRound(t, sctx, `{"findings":[{"id":"review-1","severity":"warning","description":"auto fix","action":"auto-fix"}],"summary":"1"}`, `["review-1"]`, db.RoundSelectionSourceAutoFix)

	step := &PushStep{}
	if err := step.stageReviewHandoffFiles(sctx); err != nil {
		t.Fatalf("auto-fix-only review should not require handoff audit: %v", err)
	}
}

func expectedFallbackReviewAuditRel(sctx *pipeline.StepContext) string {
	return filepath.ToSlash(filepath.Join(".no-mistakes", "issues", reviewhandoff.BranchSlug(sctx.Run.Branch), reviewhandoff.FileName(sctx.Run.ID)))
}

func writeReviewAuditFile(t *testing.T, dir, rel, runID, branch string, processed bool) {
	t.Helper()
	processedAction := reviewhandoff.ProcessedPending
	var processedAt *string
	if processed {
		processedAction = reviewhandoff.ProcessedApprove
		ts := "2026-06-16T00:00:00Z"
		processedAt = &ts
	}
	data, err := reviewhandoff.Render(reviewhandoff.HandoffFile{
		Metadata: reviewhandoff.Metadata{
			RunID:               runID,
			RunShortID:          reviewhandoff.ShortRunID(runID),
			Branch:              branch,
			Step:                reviewhandoff.StepReview,
			Status:              string(types.StepStatusCompleted),
			ReviewCycleRevision: "round-1:1",
			ReviewResultHash:    "hash",
			ProcessedAt:         processedAt,
			ProcessedAction:     processedAction,
		},
		FinalState: reviewhandoff.FinalNoFindingsText,
	})
	if err != nil {
		t.Fatal(err)
	}
	abs := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func insertReviewRound(t *testing.T, sctx *pipeline.StepContext, findings string, selectedIDs string, source string) {
	t.Helper()
	step, err := sctx.DB.InsertStepResult(sctx.Run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	round, err := sctx.DB.InsertStepRound(step.ID, 1, "initial", &findings, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if selectedIDs != "" {
		if err := sctx.DB.SetStepRoundSelection(round.ID, &selectedIDs, source); err != nil {
			t.Fatal(err)
		}
	}
}
