package steps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/reviewhandoff"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// PushStep force-pushes the worktree state to the upstream remote.
type PushStep struct{}

func (s *PushStep) Name() types.StepName { return types.StepPush }

func (s *PushStep) Execute(sctx *pipeline.StepContext) (*pipeline.StepOutcome, error) {
	ctx := sctx.Ctx
	newHeadSHA := ""

	preFormatDirty, err := dirtyPathSet(ctx, sctx.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("snapshot pre-format changes: %w", err)
	}

	// Run format command if configured (before committing, so changes are formatted)
	if fmtCmd := sctx.Config.Commands.Format; fmtCmd != "" {
		sctx.Log(fmt.Sprintf("running formatter: %s", fmtCmd))
		output, exitCode, err := runStepShellCommand(sctx, fmtCmd)
		if err != nil {
			sctx.Log(fmt.Sprintf("warning: format command failed: %v", err))
		} else if exitCode != 0 {
			sctx.Log(fmt.Sprintf("warning: format command exited with code %d: %s", exitCode, output))
		}
	}
	formatRel, err := s.formatOutputPaths(sctx, preFormatDirty)
	if err != nil {
		return nil, fmt.Errorf("resolve format outputs: %w", err)
	}

	// Commit publishable changes from agent fixes and audit artifacts.
	evidenceRel, err := s.stageInRepoEvidence(sctx)
	if err != nil {
		return nil, err
	}
	reviewRel, err := s.stageReviewHandoffAudit(sctx)
	if err != nil {
		return nil, err
	}
	status, err := git.Run(ctx, sctx.WorkDir, "status", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("check worktree status: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		sctx.Log("committing agent changes...")
		if err := s.stagePublishableChanges(sctx, reviewRel, evidenceRel, formatRel); err != nil {
			return nil, fmt.Errorf("stage agent changes: %w", err)
		}
		staged, err := git.Run(ctx, sctx.WorkDir, "diff", "--cached", "--name-only")
		if err != nil {
			return nil, fmt.Errorf("check staged agent changes: %w", err)
		}
		if strings.TrimSpace(staged) == "" {
			sctx.Log("no publishable agent changes to commit")
		} else {
			_, err := git.Run(ctx, sctx.WorkDir, "commit", "-m", "no-mistakes: apply agent fixes")
			if err != nil {
				return nil, fmt.Errorf("commit agent changes: %w", err)
			}
			headSHA, err := git.HeadSHA(ctx, sctx.WorkDir)
			if err != nil {
				return nil, fmt.Errorf("resolve head after commit: %w", err)
			}
			newHeadSHA = headSHA
		}
	}

	ref := normalizedBranchRef(sctx.Run.Branch)

	upstream := sctx.Repo.UpstreamURL
	sctx.Log(fmt.Sprintf("pushing to %s (%s)...", upstream, ref))

	// Query upstream for current ref SHA to enable safe --force-with-lease.
	// Without an explicit SHA, --force-with-lease offers no protection when
	// pushing to a URL (no remote tracking refs), silently degrading to --force.
	upstreamSHA, lsErr := git.LsRemote(ctx, sctx.WorkDir, upstream, ref)
	if lsErr != nil {
		return nil, fmt.Errorf("ls-remote upstream: %w", lsErr)
	}
	if upstreamSHA != "" {
		// Existing branch: force-with-lease with explicit expected SHA
		if err := git.Push(ctx, sctx.WorkDir, upstream, ref, upstreamSHA, true); err != nil {
			return nil, fmt.Errorf("push to upstream: %w", err)
		}
	} else {
		// New branch: regular push (no force needed)
		if err := git.Push(ctx, sctx.WorkDir, upstream, ref, "", false); err != nil {
			return nil, fmt.Errorf("push to upstream: %w", err)
		}
	}

	if newHeadSHA != "" {
		if _, err := git.Run(ctx, sctx.WorkDir, "update-ref", ref, newHeadSHA); err != nil {
			return nil, fmt.Errorf("update local branch ref: %w", err)
		}
	}

	headSHA, err := git.HeadSHA(ctx, sctx.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("resolve HEAD after push: %w", err)
	}
	if headSHA != sctx.Run.HeadSHA {
		sctx.Run.HeadSHA = headSHA
		if err := sctx.DB.UpdateRunHeadSHA(sctx.Run.ID, headSHA); err != nil {
			return nil, err
		}
	}

	sctx.Log("pushed successfully")
	return &pipeline.StepOutcome{}, nil
}

func (s *PushStep) stageReviewHandoffAudit(sctx *pipeline.StepContext) (string, error) {
	steps, err := sctx.DB.GetStepsByRun(sctx.Run.ID)
	if err != nil {
		return "", fmt.Errorf("load review handoff state: %w", err)
	}
	for _, step := range steps {
		if step.StepName != types.StepReview || step.ReviewHandoffJSON == nil {
			continue
		}
		state, err := reviewhandoff.ParseState(*step.ReviewHandoffJSON)
		if err != nil {
			return "", fmt.Errorf("parse review handoff state: %w", err)
		}
		if state.ProcessedAction == reviewhandoff.ProcessedPending {
			return "", fmt.Errorf("review handoff %s is still pending", state.RelativePath)
		}
		path, err := reviewhandoff.SafeJoin(sctx.WorkDir, state.RelativePath)
		if err != nil {
			return "", fmt.Errorf("validate review handoff path: %w", err)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return "", fmt.Errorf("review handoff %s is missing: %w", state.RelativePath, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("review handoff %s must not be a symlink", state.RelativePath)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("review handoff %s is not a regular file", state.RelativePath)
		}
		if _, err := git.Run(sctx.Ctx, sctx.WorkDir, "add", "-f", "--", filepath.ToSlash(state.RelativePath)); err != nil {
			return "", fmt.Errorf("stage review handoff: %w", err)
		}
		return filepath.ToSlash(state.RelativePath), nil
	}
	return "", nil
}

func (s *PushStep) stageInRepoEvidence(sctx *pipeline.StepContext) (string, error) {
	ctx := sctx.Ctx
	location := resolveTestEvidenceLocation(sctx.WorkDir, sctx.Run.Branch, sctx.Run.ID, sctx.Config.Test.Evidence)
	if !location.StoreInRepo {
		return "", nil
	}
	if gitIgnoresPath(ctx, sctx.WorkDir, location.Dir) {
		return "", nil
	}
	if !dirHasFiles(location.Dir) {
		return "", nil
	}
	rel, err := filepath.Rel(sctx.WorkDir, location.Dir)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", nil
	}
	rel = filepath.ToSlash(rel)
	if _, err := git.Run(ctx, sctx.WorkDir, "add", "-f", "--", rel); err != nil {
		return "", fmt.Errorf("stage test evidence: %w", err)
	}
	return rel, nil
}

func (s *PushStep) formatOutputPaths(sctx *pipeline.StepContext, before map[string]bool) ([]string, error) {
	after, err := dirtyPathSet(sctx.Ctx, sctx.WorkDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for path := range after {
		if !before[path] {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func dirtyPathSet(ctx context.Context, workDir string) (map[string]bool, error) {
	paths := map[string]bool{}
	addPaths := func(args ...string) error {
		out, err := git.Run(ctx, workDir, args...)
		if err != nil {
			return err
		}
		for _, path := range splitNUL(out) {
			if path != "" {
				paths[path] = true
			}
		}
		return nil
	}
	if err := addPaths("diff", "--name-only", "-z"); err != nil {
		return nil, err
	}
	if err := addPaths("diff", "--cached", "--name-only", "-z"); err != nil {
		return nil, err
	}
	if err := addPaths("ls-files", "--others", "--exclude-standard", "-z"); err != nil {
		return nil, err
	}
	return paths, nil
}

func (s *PushStep) stagePublishableChanges(sctx *pipeline.StepContext, reviewRel, evidenceRel string, formatRel []string) error {
	allowed := map[string]bool{}
	if reviewRel != "" {
		allowed[filepath.ToSlash(reviewRel)] = true
	}
	if evidenceRel != "" {
		allowed[filepath.ToSlash(evidenceRel)] = true
	}
	for _, rel := range formatRel {
		if rel != "" {
			allowed[filepath.ToSlash(rel)] = true
		}
	}
	if err := s.stageTrackedPublishableChanges(sctx, reviewRel, evidenceRel, allowed); err != nil {
		return err
	}
	if err := s.stageUntrackedPublishableChanges(sctx, reviewRel, evidenceRel, allowed); err != nil {
		return err
	}
	return s.validateStagedPublishableChanges(sctx, allowed, evidenceRel)
}

func (s *PushStep) stageTrackedPublishableChanges(sctx *pipeline.StepContext, reviewRel, evidenceRel string, allowed map[string]bool) error {
	out, err := git.Run(sctx.Ctx, sctx.WorkDir, "diff", "--name-only", "-z")
	if err != nil {
		return err
	}
	for _, path := range splitNUL(out) {
		if path == "" {
			continue
		}
		if !allowed[path] && !isWithinRel(path, evidenceRel) {
			return fmt.Errorf("dirty path %s is outside the publishable artifact allowlist", path)
		}
		if _, err := git.Run(sctx.Ctx, sctx.WorkDir, "add", "--", path); err != nil {
			return err
		}
	}
	return nil
}

func (s *PushStep) stageUntrackedPublishableChanges(sctx *pipeline.StepContext, reviewRel, evidenceRel string, allowed map[string]bool) error {
	out, err := git.Run(sctx.Ctx, sctx.WorkDir, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return err
	}
	for _, path := range splitNUL(out) {
		if path == "" {
			continue
		}
		if !allowed[path] && !isWithinRel(path, evidenceRel) {
			return fmt.Errorf("untracked path %s is outside the publishable artifact allowlist", path)
		}
		if _, err := git.Run(sctx.Ctx, sctx.WorkDir, "add", "--", path); err != nil {
			return err
		}
	}
	return nil
}

func (s *PushStep) validateStagedPublishableChanges(sctx *pipeline.StepContext, allowed map[string]bool, evidenceRel string) error {
	out, err := git.Run(sctx.Ctx, sctx.WorkDir, "diff", "--cached", "--name-only", "-z")
	if err != nil {
		return err
	}
	for _, path := range splitNUL(out) {
		if path == "" || allowed[path] || isWithinRel(path, evidenceRel) {
			continue
		}
		return fmt.Errorf("staged path %s is outside the publishable artifact allowlist", path)
	}
	return nil
}

func splitNUL(out string) []string {
	parts := strings.Split(out, "\x00")
	paths := parts[:0]
	for _, part := range parts {
		part = filepath.ToSlash(strings.TrimSpace(part))
		if part != "" {
			paths = append(paths, part)
		}
	}
	return paths
}

func isReviewNeighbor(path, reviewRel, evidenceRel string) bool {
	path = filepath.ToSlash(path)
	reviewRel = filepath.ToSlash(reviewRel)
	if reviewRel == "" || path == reviewRel || isWithinRel(path, evidenceRel) {
		return false
	}
	return filepath.ToSlash(filepath.Dir(path)) == filepath.ToSlash(filepath.Dir(reviewRel))
}

func isWithinRel(path, dir string) bool {
	if dir == "" {
		return false
	}
	dir = strings.TrimSuffix(filepath.ToSlash(dir), "/")
	path = filepath.ToSlash(path)
	return path == dir || strings.HasPrefix(path, dir+"/")
}

func dirHasFiles(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if !d.IsDir() {
			found = true
		}
		return nil
	})
	return found
}
