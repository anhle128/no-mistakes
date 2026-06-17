package boundary

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestVerifyRunWorktreeSafe(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, worktree, _ := seedManagedWorktree(t, ctx)

	got := VerifyRunWorktree(ctx, p, repoID, runID, worktree)
	if got.Status != types.BoundarySafe {
		t.Fatalf("status = %q reason=%q detail=%q, want safe", got.Status, got.Reason, got.Detail)
	}
	if got.Reason != types.BoundaryReasonVerifiedRunWorktree {
		t.Fatalf("reason = %q, want %q", got.Reason, types.BoundaryReasonVerifiedRunWorktree)
	}
	if got.VerifierVersion != VerifierVersion || got.VerifiedAt == 0 {
		t.Fatalf("missing verifier metadata: %+v", got)
	}
	if got.ExpectedWorktreePath == "" || got.ActualWorktreePath == "" || got.GitCommonDir == "" || got.GateRepoPath == "" {
		t.Fatalf("missing proof paths: %+v", got)
	}
	if got.ExpectedWorktreePath != got.ActualWorktreePath {
		t.Fatalf("expected and actual worktree paths differ: %+v", got)
	}
	if got.Fingerprint == "" {
		t.Fatalf("missing proof fingerprint: %+v", got)
	}
	if want := ExecutionFingerprint(got); got.Fingerprint != want {
		t.Fatalf("fingerprint = %q, want %q", got.Fingerprint, want)
	}
}

func TestVerifyRunWorktreeRejectsPrimaryCheckout(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, _, source := seedManagedWorktree(t, ctx)

	got := VerifyRunWorktree(ctx, p, repoID, runID, source)
	if got.Status != types.BoundaryUnsafe {
		t.Fatalf("status = %q, want unsafe", got.Status)
	}
	if got.Reason != types.BoundaryReasonPrimaryCheckout {
		t.Fatalf("reason = %q, want primary_checkout", got.Reason)
	}
}

func TestVerifyRunWorktreeRejectsSourcePathInsideManagedWorktreesRoot(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, _, _ := seedManagedWorktree(t, ctx)
	other := p.WorktreeDir(repoID, "other-run")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}

	got := VerifyRunWorktree(ctx, p, repoID, runID, other)
	if got.Status != types.BoundaryUnsafe {
		t.Fatalf("status = %q, want unsafe", got.Status)
	}
	if got.Reason != types.BoundaryReasonSourceOutside {
		t.Fatalf("reason = %q, want source outside", got.Reason)
	}
}

func TestVerifyRunWorktreeRejectsNestedWorktreePath(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, worktree, _ := seedManagedWorktree(t, ctx)
	nested := filepath.Join(worktree, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	got := VerifyRunWorktree(ctx, p, repoID, runID, nested)
	if got.Status != types.BoundaryUnsafe {
		t.Fatalf("status = %q, want unsafe", got.Status)
	}
	if got.Reason != types.BoundaryReasonSourceOutside {
		t.Fatalf("reason = %q, want source outside", got.Reason)
	}
}

func TestVerifyRunWorktreeMissingExpectedWorktreeIsUnknown(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, worktree, _ := seedManagedWorktree(t, ctx)
	if err := os.RemoveAll(worktree); err != nil {
		t.Fatal(err)
	}

	got := VerifyRunWorktree(ctx, p, repoID, runID, worktree)
	if got.Status != types.BoundaryUnknown {
		t.Fatalf("status = %q, want unknown", got.Status)
	}
	if got.Reason != types.BoundaryReasonMissingWorktree {
		t.Fatalf("reason = %q, want missing_worktree", got.Reason)
	}
}

func TestVerifyRunWorktreeRejectsSymlinkEscape(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, worktree, _ := seedManagedWorktree(t, ctx)
	external := filepath.Join(t.TempDir(), "escaped-worktree")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(worktree); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, worktree); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got := VerifyRunWorktree(ctx, p, repoID, runID, worktree)
	if got.Status != types.BoundaryUnknown {
		t.Fatalf("status = %q, want unknown", got.Status)
	}
	if got.Reason != types.BoundaryReasonSymlinkEscape {
		t.Fatalf("reason = %q, want symlink escape", got.Reason)
	}
}

func TestVerifyRunWorktreeRejectsGitMetadataMismatch(t *testing.T) {
	ctx := context.Background()
	p, repoID, runID, worktree, _ := seedManagedWorktree(t, ctx)
	otherGate := filepath.Join(t.TempDir(), "other.git")
	if err := git.InitBare(ctx, otherGate); err != nil {
		t.Fatal(err)
	}
	gitFile := filepath.Join(worktree, ".git")
	data, err := os.ReadFile(gitFile)
	if err != nil {
		t.Fatal(err)
	}
	gitDir := ""
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "gitdir: ") {
			gitDir = strings.TrimSpace(strings.TrimPrefix(line, "gitdir: "))
			break
		}
	}
	if gitDir == "" {
		t.Fatal("worktree .git file did not contain gitdir")
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(worktree, gitDir)
	}
	commonDirFile := filepath.Join(filepath.Clean(gitDir), "commondir")
	if err := os.WriteFile(commonDirFile, []byte(otherGate+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := VerifyRunWorktree(ctx, p, repoID, runID, worktree)
	if got.Status != types.BoundaryUnknown {
		t.Fatalf("status = %q, want unknown", got.Status)
	}
	if got.Reason != types.BoundaryReasonGitMetadataMismatch {
		t.Fatalf("reason = %q, want metadata mismatch", got.Reason)
	}
}

func seedManagedWorktree(t *testing.T, ctx context.Context) (*paths.Paths, string, string, string, string) {
	t.Helper()
	root := t.TempDir()
	p := paths.WithRoot(root)
	if err := p.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	repoID := "repo1"
	runID := "run1"
	gateDir := p.RepoDir(repoID)
	if err := git.InitBare(ctx, gateDir); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(ctx, source, "init", "."); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(ctx, source, "config", "user.email", "test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(ctx, source, "config", "user.name", "Test User"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(ctx, source, "add", "README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(ctx, source, "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	if _, err := git.Run(ctx, source, "push", gateDir, "HEAD:refs/heads/main"); err != nil {
		t.Fatal(err)
	}
	sha, err := git.HeadSHA(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	worktree := p.WorktreeDir(repoID, runID)
	if err := os.MkdirAll(filepath.Dir(worktree), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := git.WorktreeAdd(ctx, gateDir, worktree, sha); err != nil {
		t.Fatal(err)
	}
	return p, repoID, runID, worktree, source
}
