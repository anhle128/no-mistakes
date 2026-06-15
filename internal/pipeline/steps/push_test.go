package steps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/config"
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
	if _, err := step.stageInRepoEvidence(sctx); err != nil {
		t.Fatal(err)
	}
	if status := gitStatusPorcelain(t, dir); status != "" {
		t.Fatalf("ignored evidence directory was staged: %q", status)
	}
}

func TestPushStep_IncludesProcessedReviewHandoffFile(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")
	dir, baseSHA, headSHA := setupGitRepo(t)
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")
	gitCmd(t, dir, "push", "origin", "feature")

	rel := ".no-mistakes/issues/feature/review-issues-run-1.md"
	writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), "processed review\n")
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Repo.UpstreamURL = upstream
	insertReviewHandoffState(t, sctx, rel, reviewhandoff.ProcessedApprove)

	if _, err := (&PushStep{}).Execute(sctx); err != nil {
		t.Fatal(err)
	}

	clone := t.TempDir()
	gitCmd(t, clone, "clone", "--branch", "feature", upstream, ".")
	if got := readFile(t, filepath.Join(clone, filepath.FromSlash(rel))); got != "processed review\n" {
		t.Fatalf("review handoff content = %q", got)
	}
}

func TestPushStep_FailsPendingReviewHandoff(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	rel := ".no-mistakes/issues/feature/review-issues-run-1.md"
	writeFile(t, filepath.Join(dir, filepath.FromSlash(rel)), "pending review\n")
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewHandoffState(t, sctx, rel, reviewhandoff.ProcessedPending)

	_, err := (&PushStep{}).stageReviewHandoffAudit(sctx)
	if err == nil || !strings.Contains(err.Error(), "still pending") {
		t.Fatalf("stageReviewHandoffAudit error = %v, want pending failure", err)
	}
}

func TestPushStep_FailsMissingReviewHandoff(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewHandoffState(t, sctx, ".no-mistakes/issues/feature/missing.md", reviewhandoff.ProcessedApprove)

	_, err := (&PushStep{}).stageReviewHandoffAudit(sctx)
	if err == nil || !strings.Contains(err.Error(), "is missing") {
		t.Fatalf("stageReviewHandoffAudit error = %v, want missing-file failure", err)
	}
}

func TestPushStep_FailsDirtyReviewHandoffNeighbors(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")
	dir, baseSHA, headSHA := setupGitRepo(t)
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")
	gitCmd(t, dir, "push", "origin", "feature")

	reviewRel := "docs/review-issues-run-1.md"
	writeFile(t, filepath.Join(dir, filepath.FromSlash(reviewRel)), "processed review\n")
	writeFile(t, filepath.Join(dir, "docs", "tasks.md"), "unrelated anchor\n")
	writeFile(t, filepath.Join(dir, "docs", "neighbor.txt"), "unrelated neighbor\n")
	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	sctx.Repo.UpstreamURL = upstream
	insertReviewHandoffState(t, sctx, reviewRel, reviewhandoff.ProcessedApprove)

	_, err := (&PushStep{}).Execute(sctx)
	if err == nil || !strings.Contains(err.Error(), "outside the publishable artifact allowlist") {
		t.Fatalf("Execute error = %v, want dirty neighbor allowlist failure", err)
	}
}

func TestPushStep_AllowsFormatOutput(t *testing.T) {
	t.Parallel()
	upstream := t.TempDir()
	gitCmd(t, upstream, "init", "--bare")
	dir, baseSHA, headSHA := setupGitRepo(t)
	gitCmd(t, dir, "remote", "add", "origin", upstream)
	gitCmd(t, dir, "push", "origin", "main")
	gitCmd(t, dir, "push", "origin", "feature")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{
		Format: "printf formatted > formatted-by-push.txt",
	})
	sctx.Repo.UpstreamURL = upstream

	if _, err := (&PushStep{}).Execute(sctx); err != nil {
		t.Fatal(err)
	}

	clone := t.TempDir()
	gitCmd(t, clone, "clone", "--branch", "feature", upstream, ".")
	if got := readFile(t, filepath.Join(clone, "formatted-by-push.txt")); got != "formatted" {
		t.Fatalf("formatted-by-push.txt = %q", got)
	}
}

func TestPushStep_FailsUnexpectedPreStagedPath(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	reviewRel := ".no-mistakes/issues/feature/review-issues-run-1.md"
	writeFile(t, filepath.Join(dir, filepath.FromSlash(reviewRel)), "processed review\n")
	writeFile(t, filepath.Join(dir, "unrelated.txt"), "user change\n")
	gitCmd(t, dir, "add", "unrelated.txt")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewHandoffState(t, sctx, reviewRel, reviewhandoff.ProcessedApprove)
	reviewRel, err := (&PushStep{}).stageReviewHandoffAudit(sctx)
	if err != nil {
		t.Fatalf("stage review handoff: %v", err)
	}

	err = (&PushStep{}).stagePublishableChanges(sctx, reviewRel, "", nil)
	if err == nil || !strings.Contains(err.Error(), "outside the publishable artifact allowlist") {
		t.Fatalf("stagePublishableChanges error = %v, want allowlist failure", err)
	}
}

func TestPushStep_FailsUnexpectedModifiedPath(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	trackedRel := "app.txt"
	writeFile(t, filepath.Join(dir, trackedRel), "initial\n")
	gitCmd(t, dir, "add", trackedRel)
	gitCmd(t, dir, "commit", "-m", "add app")
	headSHA = gitCmd(t, dir, "rev-parse", "HEAD")
	reviewRel := ".no-mistakes/issues/feature/review-issues-run-1.md"
	writeFile(t, filepath.Join(dir, filepath.FromSlash(reviewRel)), "processed review\n")
	writeFile(t, filepath.Join(dir, trackedRel), "user edit\n")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewHandoffState(t, sctx, reviewRel, reviewhandoff.ProcessedApprove)
	reviewRel, err := (&PushStep{}).stageReviewHandoffAudit(sctx)
	if err != nil {
		t.Fatalf("stage review handoff: %v", err)
	}

	err = (&PushStep{}).stagePublishableChanges(sctx, reviewRel, "", nil)
	if err == nil || !strings.Contains(err.Error(), "dirty path app.txt is outside the publishable artifact allowlist") {
		t.Fatalf("stagePublishableChanges error = %v, want dirty allowlist failure", err)
	}
}

func TestPushStep_FailsUnexpectedUntrackedPath(t *testing.T) {
	t.Parallel()
	dir, baseSHA, headSHA := setupGitRepo(t)
	reviewRel := ".no-mistakes/issues/feature/review-issues-run-1.md"
	writeFile(t, filepath.Join(dir, filepath.FromSlash(reviewRel)), "processed review\n")
	writeFile(t, filepath.Join(dir, "scratch.txt"), "user scratch\n")

	ag := &mockAgent{name: "test"}
	sctx := newTestContextWithDBRecords(t, ag, dir, baseSHA, headSHA, config.Commands{})
	insertReviewHandoffState(t, sctx, reviewRel, reviewhandoff.ProcessedApprove)
	reviewRel, err := (&PushStep{}).stageReviewHandoffAudit(sctx)
	if err != nil {
		t.Fatalf("stage review handoff: %v", err)
	}

	err = (&PushStep{}).stagePublishableChanges(sctx, reviewRel, "", nil)
	if err == nil || !strings.Contains(err.Error(), "untracked path scratch.txt is outside the publishable artifact allowlist") {
		t.Fatalf("stagePublishableChanges error = %v, want untracked allowlist failure", err)
	}
}

func insertReviewHandoffState(t *testing.T, sctx *pipeline.StepContext, rel, processedAction string) {
	t.Helper()
	step, err := sctx.DB.InsertStepResult(sctx.Run.ID, types.StepReview)
	if err != nil {
		t.Fatal(err)
	}
	state := reviewhandoff.NewState(rel, "cycle-1", "digest-1", "generated-1", 100)
	if processedAction != reviewhandoff.ProcessedPending {
		processedAt := int64(200)
		state.ProcessedAction = processedAction
		state.ProcessedAt = &processedAt
		state.DecisionSource = reviewhandoff.DecisionSourceFile
		state.UpdatedAt = processedAt
	}
	if err := sctx.DB.SetStepReviewHandoff(step.ID, state); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
