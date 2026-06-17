package boundary

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

const VerifierVersion = "yolo-boundary-v1"

// VerifyRunWorktree proves that workDir is exactly the disposable worktree
// allocated for repoID/runID and that Git metadata points at the managed gate
// repository. Anything less than fresh proof is unknown or unsafe.
func VerifyRunWorktree(ctx context.Context, p *paths.Paths, repoID, runID, workDir string) types.ExecutionBoundary {
	now := time.Now().Unix()
	result := func(status types.BoundaryStatus, reason types.BoundaryReason, detail string, expectedWorktree, actualWorktree, gitCommonDir, gateRepo string) types.ExecutionBoundary {
		boundary := types.ExecutionBoundary{
			Status:               status,
			Reason:               reason,
			Detail:               detail,
			ExpectedWorktreePath: expectedWorktree,
			ActualWorktreePath:   actualWorktree,
			GitCommonDir:         gitCommonDir,
			GateRepoPath:         gateRepo,
			VerifiedAt:           now,
			VerifierVersion:      VerifierVersion,
		}
		boundary.Fingerprint = ExecutionFingerprint(boundary)
		return boundary
	}

	if p == nil || strings.TrimSpace(repoID) == "" || strings.TrimSpace(runID) == "" || strings.TrimSpace(workDir) == "" {
		return result(types.BoundaryUnknown, types.BoundaryReasonUnknown, "missing run worktree verifier inputs", "", "", "", "")
	}

	expected := p.WorktreeDir(repoID, runID)
	expectedReal, err := canonicalExistingDir(expected)
	if err != nil {
		return result(types.BoundaryUnknown, types.BoundaryReasonMissingWorktree, fmt.Sprintf("expected run worktree is unavailable: %s", expected), "", "", "", "")
	}
	workReal, err := canonicalExistingDir(workDir)
	if err != nil {
		return result(types.BoundaryUnknown, types.BoundaryReasonMissingWorktree, fmt.Sprintf("work directory is unavailable: %s", workDir), expectedReal, "", "", "")
	}

	worktreesRoot, err := canonicalExistingDir(p.WorktreesDir())
	if err != nil {
		return result(types.BoundaryUnknown, types.BoundaryReasonUnknown, fmt.Sprintf("worktrees root is unavailable: %s", p.WorktreesDir()), expectedReal, workReal, "", "")
	}
	if !samePath(expectedReal, workReal) {
		status := types.BoundaryUnsafe
		reason := types.BoundaryReasonSourceOutside
		if !pathInside(workReal, worktreesRoot) {
			reason = types.BoundaryReasonPrimaryCheckout
		}
		return result(status, reason, fmt.Sprintf("work directory %s does not match disposable run worktree %s", workReal, expectedReal), expectedReal, workReal, "", "")
	}
	if !pathInside(workReal, worktreesRoot) {
		return result(types.BoundaryUnknown, types.BoundaryReasonSymlinkEscape, fmt.Sprintf("run worktree resolves outside managed worktrees root: %s", workReal), expectedReal, workReal, "", "")
	}

	expectedGate := p.RepoDir(repoID)
	gateReal, err := canonicalExistingDir(expectedGate)
	if err != nil {
		return result(types.BoundaryUnknown, types.BoundaryReasonGitMetadataMismatch, fmt.Sprintf("managed gate repository is unavailable: %s", expectedGate), expectedReal, workReal, "", "")
	}
	commonDir, err := git.Run(ctx, workReal, "rev-parse", "--git-common-dir")
	if err != nil {
		return result(types.BoundaryUnknown, types.BoundaryReasonGitMetadataMismatch, err.Error(), expectedReal, workReal, "", gateReal)
	}
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(workReal, commonDir)
	}
	commonReal, err := canonicalExistingDir(commonDir)
	if err != nil {
		return result(types.BoundaryUnknown, types.BoundaryReasonGitMetadataMismatch, fmt.Sprintf("git common dir is unavailable: %s", commonDir), expectedReal, workReal, "", gateReal)
	}
	if !samePath(commonReal, gateReal) {
		return result(types.BoundaryUnknown, types.BoundaryReasonGitMetadataMismatch, fmt.Sprintf("git common dir %s does not match managed gate %s", commonReal, gateReal), expectedReal, workReal, commonReal, gateReal)
	}

	return result(types.BoundarySafe, types.BoundaryReasonVerifiedRunWorktree, "source changes are confined to the managed disposable run worktree", expectedReal, workReal, commonReal, gateReal)
}

// ExecutionFingerprint hashes the verifier proof inputs. It intentionally
// excludes VerifiedAt so equal proof produces the same diagnostic fingerprint
// across refreshes; authorization still requires a fresh verifier pass.
func ExecutionFingerprint(boundary types.ExecutionBoundary) string {
	h := sha256.New()
	for _, part := range []string{
		VerifierVersion,
		string(boundary.Status),
		string(boundary.Reason),
		strings.TrimSpace(boundary.ExpectedWorktreePath),
		strings.TrimSpace(boundary.ActualWorktreePath),
		strings.TrimSpace(boundary.GitCommonDir),
		strings.TrimSpace(boundary.GateRepoPath),
	} {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

func canonicalExistingDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func pathInside(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
