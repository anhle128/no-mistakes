package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/reviewhandoff"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

type reviewGateState struct {
	RunID           string
	RepoID          string
	Step            types.StepName
	Status          types.StepStatus
	AbsPath         string
	RelPath         string
	Live            reviewhandoff.LiveState
	ValidationError string
}

type reviewGateSnapshot struct {
	RelPath         string
	ValidationError string
}

type ReviewGateInfo struct {
	ReviewFilePath        string
	ReviewValidationError string
}

func (e *Executor) prepareReviewGate(ctx context.Context, run *db.Run, repo *db.Repo, workDir string, stepName types.StepName, status types.StepStatus, rawFindings string, currentRoundID string, roundNum int) error {
	if stepName != types.StepReview || strings.TrimSpace(rawFindings) == "" {
		e.storeReviewGate(nil)
		return nil
	}

	findings, err := types.ParseFindingsJSON(rawFindings)
	if err != nil {
		e.storeReviewGate(nil)
		return nil
	}
	entries, err := reviewhandoff.EntriesFromFindings(findings)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		e.storeReviewGate(nil)
		return nil
	}

	revision := reviewCycleRevision(run.ID, currentRoundID, roundNum)
	live := reviewhandoff.LiveState{
		RunID:               run.ID,
		Branch:              run.Branch,
		Step:                reviewhandoff.StepReview,
		Status:              string(status),
		ReviewCycleRevision: revision,
		ReviewResultHash: reviewhandoff.ComputeHash(reviewhandoff.HashInput{
			RunID:               run.ID,
			Step:                reviewhandoff.StepReview,
			Status:              string(status),
			ReviewCycleRevision: revision,
			Findings:            entries,
			FixSummaries:        e.fixSummariesForStep(run.ID, stepName),
		}),
		Findings: entries,
	}

	existing, err := reviewhandoff.FindExistingReviewFiles(workDir, run.ID)
	if err != nil {
		return err
	}
	path, err := reviewhandoff.ResolvePath(reviewhandoff.PathResolveInput{
		CheckoutDir:             workDir,
		RunID:                   run.ID,
		Branch:                  run.Branch,
		ExistingReviewFilePaths: existing,
		UncommittedChangedPaths: changedPathsFromGitStatus(ctx, workDir),
		ReviewedChangedPaths:    reviewedChangedPaths(ctx, workDir, run),
	})
	if err != nil {
		return err
	}

	file := reviewhandoff.HandoffFile{
		Metadata:     reviewhandoff.NewMetadata(live, reviewhandoff.ShortRunID(run.ID)),
		Summary:      reviewhandoff.SummaryFor(entries),
		Findings:     entries,
		Responses:    reviewhandoff.InitialResponses(entries),
		AuditEntries: e.reviewAuditSnapshot(),
	}
	if err := reviewhandoff.WriteRenderedFile(path.AbsPath, file); err != nil {
		return err
	}

	repoID := ""
	if repo != nil {
		repoID = repo.ID
	}
	e.storeReviewGate(&reviewGateState{
		RunID:   run.ID,
		RepoID:  repoID,
		Step:    stepName,
		Status:  status,
		AbsPath: path.AbsPath,
		RelPath: path.RelPath,
		Live:    live,
	})
	return nil
}

func reviewCycleRevision(runID, currentRoundID string, roundNum int) string {
	if currentRoundID != "" {
		return fmt.Sprintf("%s:%d", currentRoundID, roundNum)
	}
	return fmt.Sprintf("%s:%d", runID, roundNum)
}

func (e *Executor) storeReviewGate(gate *reviewGateState) {
	e.mu.Lock()
	e.reviewGate = gate
	e.mu.Unlock()
}

func (e *Executor) currentReviewGateForEvent(runID string, step types.StepName, status string) *reviewGateSnapshot {
	if step != types.StepReview {
		return nil
	}
	switch types.StepStatus(status) {
	case types.StepStatusAwaitingApproval, types.StepStatusFixReview:
	default:
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.reviewGate == nil || e.reviewGate.RunID != runID || e.reviewGate.Step != step {
		return nil
	}
	return &reviewGateSnapshot{
		RelPath:         e.reviewGate.RelPath,
		ValidationError: e.reviewGate.ValidationError,
	}
}

func (e *Executor) ReviewGateInfo(step types.StepName) (ReviewGateInfo, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.reviewGate == nil || e.reviewGate.Step != step {
		return ReviewGateInfo{}, false
	}
	return ReviewGateInfo{
		ReviewFilePath:        e.reviewGate.RelPath,
		ReviewValidationError: e.reviewGate.ValidationError,
	}, true
}

func (e *Executor) ProcessReview(step types.StepName) error {
	gate, err := e.currentReviewGateForProcessing(step)
	if err != nil {
		return err
	}

	data, err := reviewhandoff.ReadBounded(gate.AbsPath, reviewhandoff.MaxFileBytes)
	if err != nil {
		return e.failReviewGateValidation(gate, err)
	}
	validated, err := reviewhandoff.ValidateBytes(gate.RelPath, data, gate.Live)
	if err != nil {
		return e.failReviewGateValidation(gate, err)
	}

	processedAt := time.Now().UTC()
	decision := reviewhandoff.DeriveDecision(validated.Responses, gate.Live.Findings, processedAt)
	if err := reviewhandoff.UpdateProcessedMetadata(gate.AbsPath, validated.Snapshot, processedAt.Format(time.RFC3339), reviewhandoff.ProcessedActionForDecision(decision)); err != nil {
		return e.failReviewGateValidation(gate, err)
	}

	response := approvalResponse{action: types.ActionApprove}
	response.reviewActions = decision.Actions
	switch decision.ExecutedAction {
	case reviewhandoff.ProcessedFix:
		response.action = types.ActionFix
		response.findingIDs = decision.SelectedFindingIDs
		response.instructions = decision.Instructions
		response.addedFindings = decision.AddedFindings
	case reviewhandoff.ProcessedSkip:
		response.action = types.ActionSkip
	}

	e.mu.Lock()
	if !e.waiting {
		e.mu.Unlock()
		return fmt.Errorf("no step awaiting approval")
	}
	if e.waitingStep != step {
		waiting := e.waitingStep
		e.mu.Unlock()
		return fmt.Errorf("step mismatch: processing %q but %q is awaiting approval", step, waiting)
	}
	if e.reviewGate == nil || e.reviewGate.RunID != gate.RunID || e.reviewGate.Step != step {
		e.mu.Unlock()
		return fmt.Errorf("review handoff is no longer current")
	}
	e.waiting = false
	e.reviewGate.ValidationError = ""
	e.reviewAuditEntries = append(e.reviewAuditEntries, reviewhandoff.AuditEntriesFromDecision(decision, gate.Live.Findings)...)
	e.mu.Unlock()

	e.approvalCh <- response
	return nil
}

func isLegacyReviewHandoffAction(action types.ApprovalAction) bool {
	switch action {
	case types.ActionApprove, types.ActionFix, types.ActionSkip:
		return true
	default:
		return false
	}
}

func (e *Executor) mirrorAutomationReviewGate(gate reviewGateState, action types.ApprovalAction, findingIDs []string, instructions map[string]string, addedFindings []types.Finding) ([]reviewhandoff.AuditEntry, error) {
	data, err := reviewhandoff.ReadBounded(gate.AbsPath, reviewhandoff.MaxFileBytes)
	if err != nil {
		return nil, e.failReviewGateValidation(gate, err)
	}
	validated, err := reviewhandoff.ValidateBytes(gate.RelPath, data, gate.Live)
	if err != nil {
		return nil, e.failReviewGateValidation(gate, err)
	}

	processedAt := time.Now().UTC()
	responses := reviewhandoff.AutomationResponses(action, gate.Live.Findings, findingIDs, instructions)
	decision := reviewhandoff.AutomationDecision(action, findingIDs, instructions, addedFindings, processedAt)
	decision.Actions = responseActions(responses)
	file := reviewhandoff.HandoffFile{
		Metadata:     reviewhandoff.NewMetadata(gate.Live, reviewhandoff.ShortRunID(gate.RunID)),
		Summary:      reviewhandoff.SummaryFor(gate.Live.Findings),
		Findings:     gate.Live.Findings,
		Responses:    responses,
		AuditEntries: e.reviewAuditSnapshot(),
	}
	processedAtText := processedAt.Format(time.RFC3339)
	file.Metadata.ProcessedAt = &processedAtText
	file.Metadata.ProcessedAction = reviewhandoff.ProcessedActionForDecision(decision)
	if err := reviewhandoff.WriteRenderedFileFromSnapshot(gate.AbsPath, validated.Snapshot, file); err != nil {
		return nil, e.failReviewGateValidation(gate, err)
	}
	return reviewhandoff.AuditEntriesFromDecision(decision, gate.Live.Findings), nil
}

func responseActions(responses []reviewhandoff.ResponseBlock) map[string]string {
	actions := make(map[string]string, len(responses))
	for _, response := range responses {
		actions[response.ID] = response.Action
	}
	return actions
}

func (e *Executor) reviewAuditSnapshot() []reviewhandoff.AuditEntry {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]reviewhandoff.AuditEntry(nil), e.reviewAuditEntries...)
}

func (e *Executor) maybeWriteFinalReviewAudit(ctx context.Context, run *db.Run, repo *db.Repo, workDir string, stepName types.StepName, fixing bool, rawFindings string, currentRoundID string, roundNum int) error {
	if stepName != types.StepReview || !fixing || reviewHasFindingsJSON(rawFindings) {
		return nil
	}
	audit := e.reviewAuditSnapshot()
	if len(audit) == 0 {
		return nil
	}
	fixSummaries := e.fixSummariesForStep(run.ID, stepName)
	audit = auditWithFixSummaries(audit, fixSummaries)
	path, err := e.currentReviewHandoffPath(ctx, run, workDir)
	if err != nil {
		return err
	}

	revision := reviewCycleRevision(run.ID, currentRoundID, roundNum)
	live := reviewhandoff.LiveState{
		RunID:               run.ID,
		Branch:              run.Branch,
		Step:                reviewhandoff.StepReview,
		Status:              string(types.StepStatusCompleted),
		ReviewCycleRevision: revision,
		ReviewResultHash: reviewhandoff.ComputeHash(reviewhandoff.HashInput{
			RunID:               run.ID,
			Step:                reviewhandoff.StepReview,
			Status:              string(types.StepStatusCompleted),
			ReviewCycleRevision: revision,
			FixSummaries:        fixSummaries,
		}),
	}
	meta := reviewhandoff.NewMetadata(live, reviewhandoff.ShortRunID(run.ID))
	processedAt := time.Now().UTC().Format(time.RFC3339)
	meta.ProcessedAt = &processedAt
	meta.ProcessedAction = reviewhandoff.ProcessedApprove
	file := reviewhandoff.HandoffFile{
		Metadata:     meta,
		Summary:      reviewhandoff.SummaryFor(nil),
		AuditEntries: audit,
		FinalState:   reviewhandoff.FinalNoFindingsText,
	}
	if err := reviewhandoff.WriteRenderedFile(path.AbsPath, file); err != nil {
		return err
	}

	repoID := ""
	if repo != nil {
		repoID = repo.ID
	}
	e.mu.Lock()
	e.reviewGate = &reviewGateState{
		RunID:   run.ID,
		RepoID:  repoID,
		Step:    stepName,
		Status:  types.StepStatusCompleted,
		AbsPath: path.AbsPath,
		RelPath: path.RelPath,
		Live:    live,
	}
	e.reviewAuditEntries = audit
	e.mu.Unlock()
	return nil
}

func reviewHasFindingsJSON(raw string) bool {
	if strings.TrimSpace(raw) == "" {
		return false
	}
	findings, err := types.ParseFindingsJSON(raw)
	if err != nil {
		return true
	}
	return len(findings.Items) > 0
}

func auditWithFixSummaries(entries []reviewhandoff.AuditEntry, summaries []string) []reviewhandoff.AuditEntry {
	out := append([]reviewhandoff.AuditEntry(nil), entries...)
	summary := latestNonEmpty(summaries)
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

func latestNonEmpty(values []string) string {
	for i := len(values) - 1; i >= 0; i-- {
		if value := strings.TrimSpace(values[i]); value != "" {
			return value
		}
	}
	return ""
}

func (e *Executor) currentReviewHandoffPath(ctx context.Context, run *db.Run, workDir string) (reviewhandoff.PathResult, error) {
	e.mu.Lock()
	if e.reviewGate != nil && e.reviewGate.RunID == run.ID && e.reviewGate.AbsPath != "" && e.reviewGate.RelPath != "" {
		path := reviewhandoff.PathResult{AbsPath: e.reviewGate.AbsPath, RelPath: e.reviewGate.RelPath, Source: "current"}
		e.mu.Unlock()
		return path, nil
	}
	e.mu.Unlock()

	existing, err := reviewhandoff.FindExistingReviewFiles(workDir, run.ID)
	if err != nil {
		return reviewhandoff.PathResult{}, err
	}
	return reviewhandoff.ResolvePath(reviewhandoff.PathResolveInput{
		CheckoutDir:             workDir,
		RunID:                   run.ID,
		Branch:                  run.Branch,
		ExistingReviewFilePaths: existing,
		UncommittedChangedPaths: changedPathsFromGitStatus(ctx, workDir),
		ReviewedChangedPaths:    reviewedChangedPaths(ctx, workDir, run),
	})
}

func (e *Executor) currentReviewGateForProcessing(step types.StepName) (reviewGateState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.waiting {
		return reviewGateState{}, fmt.Errorf("no step awaiting approval")
	}
	if e.waitingStep != step {
		return reviewGateState{}, fmt.Errorf("step mismatch: processing %q but %q is awaiting approval", step, e.waitingStep)
	}
	if e.reviewGate == nil || e.reviewGate.Step != step || e.reviewGate.AbsPath == "" {
		return reviewGateState{}, fmt.Errorf("no review handoff file is awaiting processing")
	}
	return *e.reviewGate, nil
}

func (e *Executor) failReviewGateValidation(gate reviewGateState, err error) error {
	message := reviewValidationMessage(gate.RelPath, err)
	e.mu.Lock()
	if e.reviewGate != nil && e.reviewGate.RunID == gate.RunID && e.reviewGate.Step == gate.Step {
		e.reviewGate.ValidationError = message
	}
	e.mu.Unlock()

	status := string(gate.Status)
	step := gate.Step
	phaseLabel := reviewhandoff.PhaseLabel(gate.Step, gate.Status)
	var phaseLabelPtr *string
	if phaseLabel != "" {
		phaseLabelPtr = &phaseLabel
	}
	e.onEvent(ipc.Event{
		Type:                  ipc.EventStepCompleted,
		RunID:                 gate.RunID,
		RepoID:                gate.RepoID,
		StepName:              &step,
		Status:                &status,
		ReviewPhaseLabel:      phaseLabelPtr,
		ReviewFilePath:        &gate.RelPath,
		ReviewValidationError: &message,
	})
	return fmt.Errorf("%s", message)
}

func reviewValidationMessage(path string, err error) string {
	var validationErr *reviewhandoff.ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Error()
	}
	if path == "" {
		return err.Error()
	}
	return path + ": " + err.Error()
}

func changedPathsFromGitStatus(ctx context.Context, workDir string) []string {
	out, err := git.Run(ctx, workDir, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		slog.Debug("failed to collect git status paths for review handoff", "error", err)
		return nil
	}
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if before, after, ok := strings.Cut(path, " -> "); ok {
			_ = before
			path = strings.TrimSpace(after)
		}
		path = strings.Trim(path, `"`)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func reviewedChangedPaths(ctx context.Context, workDir string, run *db.Run) []string {
	if run == nil || run.HeadSHA == "" || git.IsZeroSHA(run.HeadSHA) {
		return nil
	}
	base := run.BaseSHA
	if base == "" || git.IsZeroSHA(base) {
		base = git.EmptyTreeSHA
	}
	files, err := git.DiffNameOnly(ctx, workDir, base, run.HeadSHA)
	if err != nil {
		slog.Debug("failed to collect reviewed git paths for review handoff", "error", err)
		return nil
	}
	return files
}
