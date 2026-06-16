package steps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
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

	// Commit any uncommitted changes from agent fixes
	if err := s.stageInRepoEvidence(sctx); err != nil {
		return nil, err
	}
	if err := s.stageReviewHandoffFiles(sctx); err != nil {
		return nil, err
	}
	status, _ := git.Run(ctx, sctx.WorkDir, "status", "--porcelain")
	if strings.TrimSpace(status) != "" {
		sctx.Log("committing agent changes...")
		if err := s.stageAgentChanges(sctx); err != nil {
			return nil, fmt.Errorf("stage agent changes: %w", err)
		}
		staged, _ := git.Run(ctx, sctx.WorkDir, "diff", "--cached", "--name-only")
		if strings.TrimSpace(staged) == "" {
			sctx.Log("no staged agent changes after review audit exclusions")
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

func (s *PushStep) stageReviewHandoffFiles(sctx *pipeline.StepContext) error {
	if sctx == nil || sctx.Run == nil || sctx.WorkDir == "" {
		return nil
	}
	requirement, err := s.reviewAuditRequirement(sctx)
	if err != nil {
		return err
	}
	if !requirement.Required {
		return nil
	}
	if err := ensureProcessedReviewAuditFile(sctx.WorkDir, requirement.ExpectedRel, sctx.Run.ID); err != nil {
		if !isRegenerableReviewAuditError(err) {
			return err
		}
		if regenErr := regenerateReviewAuditFile(sctx, requirement); regenErr != nil {
			return fmt.Errorf("%v; regeneration failed: %w", err, regenErr)
		}
		if err := ensureProcessedReviewAuditFile(sctx.WorkDir, requirement.ExpectedRel, sctx.Run.ID); err != nil {
			return err
		}
	}
	staged, err := stageReviewHandoffFile(sctx, requirement.ExpectedRel)
	if err != nil {
		return err
	}
	if !staged {
		return missingReviewAuditFileError(requirement.ExpectedRel)
	}
	return nil
}

type reviewAuditRequirementInfo struct {
	Required    bool
	ExpectedRel string
	AnchorRel   string
	Step        *db.StepResult
	Rounds      []*db.StepRound
}

func (s *PushStep) reviewAuditRequirement(sctx *pipeline.StepContext) (reviewAuditRequirementInfo, error) {
	expected := reviewhandoff.FileName(sctx.Run.ID)
	if sctx.DB == nil || sctx.Run == nil {
		return reviewAuditRequirementInfo{ExpectedRel: expected}, nil
	}
	steps, err := sctx.DB.GetStepsByRun(sctx.Run.ID)
	if err != nil {
		return reviewAuditRequirementInfo{ExpectedRel: expected}, fmt.Errorf("load review step for audit file check: %w", err)
	}
	for _, step := range steps {
		if step.StepName != types.StepReview {
			continue
		}
		rounds, err := sctx.DB.GetRoundsByStep(step.ID)
		if err != nil {
			return reviewAuditRequirementInfo{ExpectedRel: expected, Step: step}, fmt.Errorf("load review rounds for audit file check: %w", err)
		}
		required := reviewRoundsRequireAudit(rounds)
		if !required {
			return reviewAuditRequirementInfo{ExpectedRel: expected, Step: step, Rounds: rounds}, nil
		}
		changedPaths, err := pushChangedPathsFromGitStatus(sctx)
		if err != nil {
			return reviewAuditRequirementInfo{Required: true, ExpectedRel: expected, Step: step, Rounds: rounds}, fmt.Errorf("load changed paths for review audit check: %w", err)
		}
		reviewedPaths, err := pushReviewedChangedPaths(sctx)
		if err != nil {
			return reviewAuditRequirementInfo{Required: true, ExpectedRel: expected, Step: step, Rounds: rounds}, fmt.Errorf("load reviewed paths for review audit check: %w", err)
		}
		existing, err := reviewhandoff.FindExistingReviewFiles(sctx.WorkDir, sctx.Run.ID)
		if err != nil {
			return reviewAuditRequirementInfo{Required: true, ExpectedRel: expected, Step: step, Rounds: rounds}, fmt.Errorf("find existing review audit files: %w", err)
		}
		path, err := reviewhandoff.ResolvePath(reviewhandoff.PathResolveInput{
			CheckoutDir:             sctx.WorkDir,
			RunID:                   sctx.Run.ID,
			Branch:                  sctx.Run.Branch,
			ExistingReviewFilePaths: existing,
			UncommittedChangedPaths: changedPaths,
			ReviewedChangedPaths:    reviewedPaths,
		})
		if err != nil {
			return reviewAuditRequirementInfo{Required: true, ExpectedRel: expected, Step: step, Rounds: rounds}, fmt.Errorf("resolve expected review audit file: %w", err)
		}
		return reviewAuditRequirementInfo{Required: true, ExpectedRel: path.RelPath, AnchorRel: path.AnchorRelPath, Step: step, Rounds: rounds}, nil
	}
	return reviewAuditRequirementInfo{ExpectedRel: expected}, nil
}

func (s *PushStep) stageAgentChanges(sctx *pipeline.StepContext) error {
	out, err := git.Run(sctx.Ctx, sctx.WorkDir, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return err
	}
	requirement, err := s.reviewAuditRequirement(sctx)
	if err != nil {
		return err
	}
	paths := parsePorcelainPaths(out)
	stage := make([]string, 0, len(paths))
	for _, path := range paths {
		if shouldExcludeFromBroadPushStage(path, sctx.Run.ID, requirement) {
			continue
		}
		stage = append(stage, filepath.ToSlash(path))
	}
	if len(stage) == 0 {
		return nil
	}
	args := append([]string{"add", "-A", "--"}, stage...)
	_, err = git.Run(sctx.Ctx, sctx.WorkDir, args...)
	return err
}

func shouldExcludeFromBroadPushStage(path, runID string, requirement reviewAuditRequirementInfo) bool {
	rel := filepath.ToSlash(path)
	if filepath.Base(rel) == reviewhandoff.FileName(runID) {
		return true
	}
	if requirement.AnchorRel != "" && rel == filepath.ToSlash(requirement.AnchorRel) {
		return true
	}
	return false
}

func stageReviewHandoffFile(sctx *pipeline.StepContext, rel string) (bool, error) {
	abs, err := reviewhandoff.SafeJoin(sctx.WorkDir, rel)
	if err != nil {
		return false, fmt.Errorf("resolve review handoff file: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat review handoff file: %w", err)
	}
	if info.IsDir() {
		return false, nil
	}
	if _, err := git.Run(sctx.Ctx, sctx.WorkDir, "add", "-f", "--", filepath.ToSlash(rel)); err != nil {
		return false, fmt.Errorf("stage review handoff file: %w", err)
	}
	return true, nil
}

func ensureProcessedReviewAuditFile(workDir, rel, runID string) error {
	abs, err := reviewhandoff.SafeJoin(workDir, rel)
	if err != nil {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; %v", rel, err)}
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return missingReviewAuditFileError(rel)
		}
		return reviewAuditFileProblem{Kind: reviewAuditProblemUnreadable, Message: fmt.Sprintf("review audit file required but unreadable: expected %s; %v", rel, err)}
	}
	if info.IsDir() {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; path is a directory", rel)}
	}
	data, err := reviewhandoff.ReadBounded(abs, reviewhandoff.MaxFileBytes)
	if err != nil {
		return reviewAuditFileProblem{Kind: reviewAuditProblemUnreadable, Message: fmt.Sprintf("review audit file required but unreadable: expected %s; %v", rel, err)}
	}
	meta, _, err := reviewhandoff.ParseFrontMatter(data)
	if err != nil {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; %v", rel, err)}
	}
	if meta.RunID != runID {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; run_id mismatch", rel)}
	}
	if meta.Step != reviewhandoff.StepReview {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; step must be review", rel)}
	}
	if meta.ProcessedAt == nil || strings.TrimSpace(*meta.ProcessedAt) == "" {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; processed_at must be set", rel)}
	}
	if !isProcessedReviewAuditAction(meta.ProcessedAction) {
		return reviewAuditFileProblem{Kind: reviewAuditProblemInvalid, Message: fmt.Sprintf("review audit file required but invalid: expected %s; processed_action must be approve, fix, or skip", rel)}
	}
	return nil
}

func isProcessedReviewAuditAction(action string) bool {
	switch action {
	case reviewhandoff.ProcessedApprove, reviewhandoff.ProcessedFix, reviewhandoff.ProcessedSkip:
		return true
	default:
		return false
	}
}

func missingReviewAuditFileError(expectedRel string) error {
	return reviewAuditFileProblem{
		Kind:    reviewAuditProblemMissing,
		Message: fmt.Sprintf("review audit file required but absent: expected %s; missing persisted review handoff path or recoverable audit file", expectedRel),
	}
}

type reviewAuditProblemKind string

const (
	reviewAuditProblemMissing    reviewAuditProblemKind = "missing"
	reviewAuditProblemUnreadable reviewAuditProblemKind = "unreadable"
	reviewAuditProblemInvalid    reviewAuditProblemKind = "invalid"
)

type reviewAuditFileProblem struct {
	Kind    reviewAuditProblemKind
	Message string
}

func (e reviewAuditFileProblem) Error() string {
	return e.Message
}

func isRegenerableReviewAuditError(err error) bool {
	var problem reviewAuditFileProblem
	if !errors.As(err, &problem) {
		return false
	}
	return problem.Kind == reviewAuditProblemMissing || problem.Kind == reviewAuditProblemUnreadable
}

func regenerateReviewAuditFile(sctx *pipeline.StepContext, requirement reviewAuditRequirementInfo) error {
	if sctx == nil || sctx.Run == nil || sctx.DB == nil {
		return fmt.Errorf("missing persisted review state")
	}
	if requirement.Step == nil {
		return fmt.Errorf("missing persisted review step")
	}
	if requirement.Step.Status != types.StepStatusCompleted {
		return fmt.Errorf("review step is not completed")
	}
	audit, fixSummaries, latestRound, err := persistedReviewAudit(requirement.Rounds)
	if err != nil {
		return err
	}
	if latestRound == nil {
		return fmt.Errorf("missing persisted review rounds")
	}
	if len(audit) == 0 {
		return fmt.Errorf("missing persisted review decisions")
	}
	audit = pushAuditWithFixSummaries(audit, fixSummaries)

	abs, err := reviewhandoff.SafeJoin(sctx.WorkDir, requirement.ExpectedRel)
	if err != nil {
		return err
	}
	revision := pushReviewCycleRevision(sctx.Run.ID, latestRound.ID, latestRound.Round)
	live := reviewhandoff.LiveState{
		RunID:               sctx.Run.ID,
		Branch:              sctx.Run.Branch,
		Step:                reviewhandoff.StepReview,
		Status:              string(types.StepStatusCompleted),
		ReviewCycleRevision: revision,
		ReviewResultHash: reviewhandoff.ComputeHash(reviewhandoff.HashInput{
			RunID:               sctx.Run.ID,
			Step:                reviewhandoff.StepReview,
			Status:              string(types.StepStatusCompleted),
			ReviewCycleRevision: revision,
			FixSummaries:        fixSummaries,
		}),
	}
	meta := reviewhandoff.NewMetadata(live, reviewhandoff.ShortRunID(sctx.Run.ID))
	processedAt := time.Now().UTC().Format(time.RFC3339)
	meta.ProcessedAt = &processedAt
	meta.ProcessedAction = reviewhandoff.ProcessedApprove
	return reviewhandoff.WriteRenderedFile(abs, reviewhandoff.HandoffFile{
		Metadata:     meta,
		Summary:      reviewhandoff.SummaryFor(nil),
		AuditEntries: audit,
		FinalState:   reviewhandoff.FinalNoFindingsText,
	})
}

func persistedReviewAudit(rounds []*db.StepRound) ([]reviewhandoff.AuditEntry, []string, *db.StepRound, error) {
	processedAt := time.Now().UTC()
	var audit []reviewhandoff.AuditEntry
	var fixSummaries []string
	var latestRound *db.StepRound
	for _, round := range rounds {
		if round == nil {
			continue
		}
		latestRound = round
		if round.IsFixRound() {
			summary := ""
			if round.FixSummary != nil {
				summary = *round.FixSummary
			}
			fixSummaries = append(fixSummaries, summary)
		}
		if round.FindingsJSON == nil || strings.TrimSpace(*round.FindingsJSON) == "" {
			continue
		}
		findings, err := types.ParseFindingsJSON(*round.FindingsJSON)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("parse persisted review findings: %w", err)
		}
		if len(findings.Items) == 0 {
			continue
		}
		if !hasBlockingFindings(findings.Items) && !types.HasAskUserFindings(findings) {
			continue
		}
		entries, err := reviewhandoff.EntriesFromFindings(findings)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("build persisted audit entries: %w", err)
		}
		selectedIDs, err := persistedSelectedFindingIDs(round)
		if err != nil {
			return nil, nil, nil, err
		}
		reviewActions, err := persistedReviewActions(round)
		if err != nil {
			return nil, nil, nil, err
		}
		instructions, added, err := persistedFixPayload(round, entries, selectedIDs)
		if err != nil {
			return nil, nil, nil, err
		}
		actions, err := persistedAuditActions(entries, selectedIDs, reviewActions)
		if err != nil {
			return nil, nil, nil, err
		}
		decision := reviewhandoff.ProcessedDecision{
			Source:         "regenerated",
			ExecutedAction: reviewhandoff.ProcessedApprove,
			Actions:        actions,
			ProcessedAt:    processedAt,
		}
		if len(selectedIDs) > 0 {
			decision.ExecutedAction = reviewhandoff.ProcessedFix
			decision.SelectedFindingIDs = selectedIDs
			decision.Instructions = instructions
			decision.AddedFindings = added
		}
		audit = append(audit, reviewhandoff.AuditEntriesFromDecision(decision, entries)...)
	}
	return audit, fixSummaries, latestRound, nil
}

func persistedSelectedFindingIDs(round *db.StepRound) ([]string, error) {
	if round == nil || round.SelectedFindingIDs == nil || strings.TrimSpace(*round.SelectedFindingIDs) == "" {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(*round.SelectedFindingIDs), &ids); err != nil {
		return nil, fmt.Errorf("parse persisted selected finding ids: %w", err)
	}
	out := ids[:0]
	for _, id := range ids {
		if strings.TrimSpace(id) != "" {
			out = append(out, id)
		}
	}
	return out, nil
}

func persistedFixPayload(round *db.StepRound, entries []reviewhandoff.FindingEntry, selectedIDs []string) (map[string]string, []types.Finding, error) {
	if round == nil || round.UserFindingsJSON == nil || strings.TrimSpace(*round.UserFindingsJSON) == "" {
		if missing := selectedIDsWithoutEntries(entries, selectedIDs, nil); len(missing) > 0 {
			return nil, nil, fmt.Errorf("missing persisted user findings for selected ids: %s", strings.Join(missing, ", "))
		}
		return nil, nil, nil
	}
	findings, err := types.ParseFindingsJSON(*round.UserFindingsJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("parse persisted user findings: %w", err)
	}
	original := make(map[string]bool, len(entries))
	for _, entry := range entries {
		original[entry.ID] = true
	}
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}
	instructions := make(map[string]string)
	var added []types.Finding
	addedIDs := make(map[string]bool)
	for _, item := range findings.Items {
		if original[item.ID] {
			if note := strings.TrimSpace(item.UserInstructions); note != "" {
				instructions[item.ID] = note
			}
			continue
		}
		if selected[item.ID] || item.Source == types.FindingSourceUser {
			added = append(added, item)
			addedIDs[item.ID] = true
		}
	}
	if missing := selectedIDsWithoutEntries(entries, selectedIDs, addedIDs); len(missing) > 0 {
		return nil, nil, fmt.Errorf("missing persisted user findings for selected ids: %s", strings.Join(missing, ", "))
	}
	if len(instructions) == 0 {
		instructions = nil
	}
	return instructions, added, nil
}

func selectedIDsWithoutEntries(entries []reviewhandoff.FindingEntry, selectedIDs []string, added map[string]bool) []string {
	known := make(map[string]bool, len(entries))
	for _, entry := range entries {
		known[entry.ID] = true
	}
	var missing []string
	for _, id := range selectedIDs {
		if known[id] || (added != nil && added[id]) {
			continue
		}
		missing = append(missing, id)
	}
	return missing
}

type persistedReviewActionEnvelope struct {
	ReviewActions map[string]string `json:"review_actions"`
}

func persistedReviewActions(round *db.StepRound) (map[string]string, error) {
	if round == nil || round.UserFindingsJSON == nil || strings.TrimSpace(*round.UserFindingsJSON) == "" {
		return nil, nil
	}
	var payload persistedReviewActionEnvelope
	if err := json.Unmarshal([]byte(*round.UserFindingsJSON), &payload); err != nil {
		return nil, fmt.Errorf("parse persisted review actions: %w", err)
	}
	clean := make(map[string]string, len(payload.ReviewActions))
	for id, action := range payload.ReviewActions {
		id = strings.TrimSpace(id)
		action = strings.TrimSpace(action)
		if id == "" || action == "" {
			continue
		}
		switch action {
		case reviewhandoff.ActionAccept, reviewhandoff.ActionSkip, reviewhandoff.ActionFix:
			clean[id] = action
		default:
			return nil, fmt.Errorf("invalid persisted review action for %s: %s", id, action)
		}
	}
	if len(clean) == 0 {
		return nil, nil
	}
	return clean, nil
}

func persistedAuditActions(entries []reviewhandoff.FindingEntry, selectedIDs []string, reviewActions map[string]string) (map[string]string, error) {
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}
	actions := make(map[string]string, len(entries))
	for _, entry := range entries {
		if selected[entry.ID] {
			actions[entry.ID] = reviewhandoff.ActionFix
			continue
		}
		action, ok := reviewActions[entry.ID]
		if !ok {
			return nil, fmt.Errorf("missing persisted review action for %s; cannot regenerate exact review audit", entry.ID)
		}
		if action == reviewhandoff.ActionFix {
			return nil, fmt.Errorf("persisted review action for %s is fix without persisted selection", entry.ID)
		}
		actions[entry.ID] = action
	}
	return actions, nil
}

func pushAuditWithFixSummaries(entries []reviewhandoff.AuditEntry, summaries []string) []reviewhandoff.AuditEntry {
	out := append([]reviewhandoff.AuditEntry(nil), entries...)
	summary := latestPushFixSummary(summaries)
	if summary == "" {
		return out
	}
	for i := range out {
		if out[i].Action == reviewhandoff.ActionFix && strings.TrimSpace(out[i].FixSummary) == "" {
			out[i].FixSummary = summary
		}
	}
	return out
}

func latestPushFixSummary(values []string) string {
	for i := len(values) - 1; i >= 0; i-- {
		if value := strings.TrimSpace(values[i]); value != "" {
			return value
		}
	}
	return ""
}

func pushReviewCycleRevision(runID, currentRoundID string, roundNum int) string {
	if currentRoundID != "" {
		return fmt.Sprintf("%s:%d", currentRoundID, roundNum)
	}
	return fmt.Sprintf("%s:%d", runID, roundNum)
}

func reviewRoundsRequireAudit(rounds []*db.StepRound) bool {
	for _, round := range rounds {
		if round == nil || round.FindingsJSON == nil || strings.TrimSpace(*round.FindingsJSON) == "" {
			continue
		}
		findings, err := types.ParseFindingsJSON(*round.FindingsJSON)
		if err != nil || len(findings.Items) == 0 {
			continue
		}
		if !hasBlockingFindings(findings.Items) && !types.HasAskUserFindings(findings) {
			continue
		}
		if round.SelectionSource != nil && *round.SelectionSource == db.RoundSelectionSourceAutoFix && round.SelectedFindingIDs != nil && strings.TrimSpace(*round.SelectedFindingIDs) != "" {
			continue
		}
		return true
	}
	return false
}

func pushChangedPathsFromGitStatus(sctx *pipeline.StepContext) ([]string, error) {
	out, err := git.Run(sctx.Ctx, sctx.WorkDir, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	return parsePorcelainPaths(out), nil
}

func pushReviewedChangedPaths(sctx *pipeline.StepContext) ([]string, error) {
	if sctx.Run == nil || sctx.Run.HeadSHA == "" || git.IsZeroSHA(sctx.Run.HeadSHA) {
		return nil, nil
	}
	base := sctx.Run.BaseSHA
	if base == "" || git.IsZeroSHA(base) {
		base = git.EmptyTreeSHA
	}
	files, err := git.DiffNameOnly(sctx.Ctx, sctx.WorkDir, base, sctx.Run.HeadSHA)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func parsePorcelainPaths(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if _, after, ok := strings.Cut(path, " -> "); ok {
			path = strings.TrimSpace(after)
		}
		path = strings.Trim(path, `"`)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func (s *PushStep) stageInRepoEvidence(sctx *pipeline.StepContext) error {
	ctx := sctx.Ctx
	location := resolveTestEvidenceLocation(sctx.WorkDir, sctx.Run.Branch, sctx.Run.ID, sctx.Config.Test.Evidence)
	if !location.StoreInRepo {
		return nil
	}
	if gitIgnoresPath(ctx, sctx.WorkDir, location.Dir) {
		return nil
	}
	if !dirHasFiles(location.Dir) {
		return nil
	}
	rel, err := filepath.Rel(sctx.WorkDir, location.Dir)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return nil
	}
	if _, err := git.Run(ctx, sctx.WorkDir, "add", "-f", "--", filepath.ToSlash(rel)); err != nil {
		return fmt.Errorf("stage test evidence: %w", err)
	}
	return nil
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
