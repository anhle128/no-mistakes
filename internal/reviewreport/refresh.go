package reviewreport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
	gitutil "github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// Refresh rebuilds the Review resolution report for a run from persisted Review
// rounds and decisions. Clean Review runs intentionally produce no report row.
func Refresh(database *db.DB, p *paths.Paths, runID string) (*db.ReviewResolutionReport, error) {
	run, err := database.GetRun(runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run %s not found", runID)
	}
	repo, err := database.GetRepo(run.RepoID)
	if err != nil {
		return nil, err
	}
	steps, err := database.GetStepsByRun(runID)
	if err != nil {
		return nil, err
	}
	var reviewStep *db.StepResult
	for _, step := range steps {
		if step.StepName == types.StepReview {
			reviewStep = step
			break
		}
	}
	if reviewStep == nil {
		if err := database.DeleteReviewResolutionReport(runID); err != nil {
			return nil, err
		}
		if err := removeReportIfExists(p.ReviewResolutionReportPath(runID)); err != nil {
			return nil, err
		}
		return nil, nil
	}
	rounds, err := database.GetRoundsByStep(reviewStep.ID)
	if err != nil {
		return nil, err
	}
	decisions, err := database.GetReviewResolutionDecisions(runID)
	if err != nil {
		return nil, err
	}

	reportPath := p.ReviewResolutionReportPath(runID)
	existing, err := database.GetReviewResolutionReport(runID)
	if err != nil {
		return nil, err
	}
	snap, hasFindings, err := buildSnapshot(run, repo, reviewStep, rounds, decisions, reportPath, p.RepoDir(run.RepoID), existing)
	if err != nil {
		return nil, err
	}
	if !hasFindings {
		if err := database.DeleteReviewResolutionReport(runID); err != nil {
			return nil, err
		}
		if err := removeReportIfExists(reportPath); err != nil {
			return nil, err
		}
		return nil, nil
	}

	markdown := RenderMarkdown(snap)
	hash := sha256.Sum256([]byte(markdown))
	contentHash := hex.EncodeToString(hash[:])
	watermark := sourceWatermark(rounds, decisions)

	tmpPath, err := writeReportTemp(reportPath, []byte(markdown))
	if err != nil {
		return nil, fmt.Errorf("write review resolution report: %w", err)
	}
	defer os.Remove(tmpPath)

	meta := reportMetadata(snap, watermark, contentHash)
	unavailable := meta
	unavailable.Status = db.ReviewResolutionStatusEvidenceUnavailable
	unavailable.ResolvedCount = 0
	unavailable.AcceptedCount = 0
	unavailable.InformationalCount = 0
	unavailable.StillOpenCount = 0
	unavailable.LastRefreshResult = "evidence unavailable: refresh in progress"
	if err := database.UpsertReviewResolutionReport(unavailable); err != nil {
		return nil, fmt.Errorf("stage review resolution metadata: %w", err)
	}
	if err := promoteReportTemp(tmpPath, reportPath); err != nil {
		unavailable.LastRefreshResult = "evidence unavailable: report promotion failed"
		if stageErr := database.UpsertReviewResolutionReport(unavailable); stageErr != nil {
			return nil, fmt.Errorf("promote review resolution report: %w; additionally failed to persist evidence-unavailable metadata: %v", err, stageErr)
		}
		return nil, fmt.Errorf("promote review resolution report: %w", err)
	}
	tmpPath = ""
	if err := database.UpsertReviewResolutionReport(meta); err != nil {
		unavailable.LastRefreshResult = "evidence unavailable: final metadata persistence failed"
		if stageErr := database.UpsertReviewResolutionReport(unavailable); stageErr != nil {
			return nil, fmt.Errorf("persist review resolution metadata: %w; additionally failed to persist evidence-unavailable metadata: %v", err, stageErr)
		}
		return nil, fmt.Errorf("persist review resolution metadata: %w", err)
	}
	return database.GetReviewResolutionReport(runID)
}

func buildSnapshot(run *db.Run, repo *db.Repo, reviewStep *db.StepResult, rounds []*db.StepRound, decisions []*db.ReviewResolutionDecision, reportPath, gateRepoPath string, existing *db.ReviewResolutionReport) (Snapshot, bool, error) {
	now := time.Now().Unix()
	snap := Snapshot{
		RunID:          run.ID,
		Branch:         run.Branch,
		BaseSHA:        run.BaseSHA,
		HeadSHA:        run.HeadSHA,
		ReviewStatus:   string(reviewStep.Status),
		ReportPath:     reportPath,
		FirstGenerated: now,
		LastRefreshed:  now,
	}
	if repo != nil {
		snap.RepoIdentifier = repo.WorkingPath
	}
	if existing != nil {
		snap.FirstGenerated = existing.FirstGeneratedAt
	}
	if len(rounds) > 0 {
		from, to := rounds[0].Round, rounds[len(rounds)-1].Round
		snap.SourceRoundFrom = &from
		snap.SourceRoundTo = &to
	}

	entriesByID := map[string]*Entry{}
	order := []string{}
	var latestResolutionEvidence *types.Findings
	var latestResolutionRound int
	parseFailed := false
	for _, round := range rounds {
		for i, raw := range []*string{round.FindingsJSON, round.UserFindingsJSON} {
			isReviewOutput := i == 0
			if raw == nil || strings.TrimSpace(*raw) == "" {
				continue
			}
			findings, err := types.ParseFindingsJSON(*raw)
			if err != nil {
				parseFailed = true
				continue
			}
			if isReviewOutput && isResolutionEvidenceRound(round, findings) {
				latest := findings
				latestResolutionEvidence = &latest
				latestResolutionRound = round.Round
			}
			for i, finding := range findings.Items {
				if finding.ID == "" {
					finding.ID = fmt.Sprintf("review-%d-%d", round.Round, i+1)
				}
				entry, ok := entriesByID[finding.ID]
				if !ok {
					entry = &Entry{Finding: finding, FirstRound: round.Round}
					entriesByID[finding.ID] = entry
					order = append(order, finding.ID)
				}
				entry.Finding = mergeFinding(entry.Finding, finding)
				entry.LastSeenRound = round.Round
			}
		}
	}
	if len(order) == 0 && parseFailed {
		snap.ReportStatus = db.ReviewResolutionStatusEvidenceUnavailable
		snap.Degraded = true
		return snap, true, nil
	}
	if len(order) == 0 {
		return snap, false, nil
	}

	decisionByFinding := latestDecisions(decisions)
	for _, id := range order {
		entry := entriesByID[id]
		if decision := decisionByFinding[id]; decision != nil {
			entry.Decision = &Decision{
				Action:      decision.Action,
				ActorSource: decision.ActorSource,
				CreatedAt:   decision.CreatedAt,
			}
			if decision.Reason != nil {
				entry.Decision.Reason = *decision.Reason
			}
			if decision.RoundID != nil {
				entry.Decision.RoundID = *decision.RoundID
			}
		}
		applyFixEvidence(entry, rounds, gateRepoPath)
		classifyEntry(entry, latestResolutionEvidence, latestResolutionRound, reviewStep.Status, entriesByID)
		if parseFailed {
			entry.Degraded = true
			entry.EvidenceQuality = "degraded"
			if entry.Outcome == OutcomeResolved || entry.Outcome == OutcomeAccepted {
				entry.Outcome = OutcomeStillOpen
				entry.OutcomeText = "Review evidence was partially unreadable; unresolved status is preserved."
			}
		}
		entry.EvidenceReference = evidenceReference(entry)
		snap.Degraded = snap.Degraded || entry.Degraded
		snap.Entries = append(snap.Entries, *entry)
	}
	snap.ReportStatus = classifyReportStatus(run.Status, reviewStep.Status, snap)
	if snap.ReportStatus == db.ReviewResolutionStatusFinal {
		finalized := now
		snap.FinalizedAt = &finalized
	}
	return snap, true, nil
}

func isResolutionEvidenceRound(round *db.StepRound, findings types.Findings) bool {
	if round == nil || len(findings.Items) > 0 || !round.IsFixRound() {
		return true
	}
	if round.FixCommitSHA != nil && strings.TrimSpace(*round.FixCommitSHA) != "" {
		return true
	}
	return round.NoCommitReason == nil || strings.TrimSpace(*round.NoCommitReason) == ""
}

func mergeFinding(prev, next types.Finding) types.Finding {
	if next.Severity == "" {
		next.Severity = prev.Severity
	}
	if next.File == "" {
		next.File = prev.File
	}
	if next.Line == 0 {
		next.Line = prev.Line
	}
	if next.Description == "" {
		next.Description = prev.Description
	}
	if next.Action == "" {
		next.Action = prev.Action
	}
	if next.Source == "" {
		next.Source = prev.Source
	}
	if next.UserInstructions == "" {
		next.UserInstructions = prev.UserInstructions
	}
	return next
}

func latestDecisions(decisions []*db.ReviewResolutionDecision) map[string]*db.ReviewResolutionDecision {
	out := map[string]*db.ReviewResolutionDecision{}
	for _, decision := range decisions {
		if decision == nil {
			continue
		}
		current := out[decision.FindingID]
		if current == nil || decision.CreatedAt >= current.CreatedAt {
			out[decision.FindingID] = decision
		}
	}
	return out
}

func applyFixEvidence(entry *Entry, rounds []*db.StepRound, repoPath string) {
	for _, round := range rounds {
		if !round.IsFixRound() || round.Round < entry.FirstRound {
			continue
		}
		selection := selectionForFixRound(round, rounds)
		if len(selection.ids) > 0 && !selection.ids[entry.Finding.ID] {
			continue
		}
		entry.FixRound = round.Round
		if selection.source != "" {
			entry.SelectionSource = selection.source
		}
		if round.FixSummary != nil && strings.TrimSpace(*round.FixSummary) != "" {
			entry.FixSummary = sanitizeField(*round.FixSummary)
		}
		if round.FixCommitSHA != nil {
			entry.FixCommitSHA = sanitizeShort(*round.FixCommitSHA)
		}
		if round.NoCommitReason != nil {
			entry.NoCommitReason = sanitizeShort(*round.NoCommitReason)
		}
		if round.FixResolutionDetailsJSON != nil {
			detail, degraded := matchingFixDetail(*round.FixResolutionDetailsJSON, entry.Finding.ID, selection.ids)
			if degraded {
				entry.Degraded = true
			}
			if detail != nil {
				entry.SolutionSource = "fix agent structured output"
				entry.AppliedSolution = sanitizeField(detail.AppliedSolution)
				entry.Rationale = sanitizeField(detail.WhyThisSolution)
				entry.ChangedFiles = sanitizeChangedFiles(detail.ChangedFiles)
				entry.EvidenceQuality = "structured"
			}
		}
	}
	if entry.SolutionSource == "" {
		switch {
		case entry.FixSummary != "":
			applySummaryFallbackEvidence(entry, repoPath)
		case entry.FixRound > 0:
			entry.SolutionSource = "evidence unavailable"
			entry.AppliedSolution = "fix attempt recorded without parseable structured resolution details"
			entry.Rationale = "not recorded"
			entry.EvidenceQuality = "unavailable"
			entry.Degraded = true
		default:
			entry.SolutionSource = "not applicable"
			entry.AppliedSolution = "not recorded"
			entry.Rationale = "not recorded"
			entry.EvidenceQuality = "unavailable"
		}
	}
	if len(entry.ChangedFiles) == 0 {
		entry.ChangedFiles = []string{"not recorded"}
	}
}

func applySummaryFallbackEvidence(entry *Entry, repoPath string) {
	entry.EvidenceQuality = "round_level"
	switch {
	case entry.FixCommitSHA != "":
		entry.SolutionSource = "inferred from fix round summary and commit changed-file diff because structured resolution details were unavailable"
		files, err := changedFilesForFixCommit(repoPath, entry.FixCommitSHA)
		if err == nil && len(files) > 0 {
			entry.ChangedFiles = files
		} else {
			entry.SolutionSource = "inferred from fix round summary because structured resolution details were unavailable; commit changed-file diff unavailable"
			entry.EvidenceQuality = "degraded"
			entry.Degraded = true
		}
	default:
		entry.SolutionSource = "inferred from fix round summary because structured resolution details were unavailable; fix commit SHA unavailable"
	}
	switch {
	case entry.FixCommitSHA != "" && len(entry.ChangedFiles) > 0:
		entry.AppliedSolution = sanitizeField(fmt.Sprintf("Fix round %d recorded commit %s touching %s. Legacy summary text was not embedded because structured resolution details were unavailable.", entry.FixRound, entry.FixCommitSHA, strings.Join(entry.ChangedFiles, ", ")))
	case entry.FixCommitSHA != "":
		entry.AppliedSolution = sanitizeField(fmt.Sprintf("Fix round %d recorded commit %s. Legacy summary text was not embedded because structured resolution details were unavailable.", entry.FixRound, entry.FixCommitSHA))
	default:
		entry.AppliedSolution = sanitizeField(fmt.Sprintf("Fix round %d recorded a legacy summary-only fix attempt; raw summary text is omitted because structured resolution details and fix commit evidence were unavailable.", entry.FixRound))
	}
	entry.Rationale = "Structured rationale was unavailable; this is round-level evidence derived from persisted fix-round evidence and commit changed-file paths when available."
}

func changedFilesForFixCommit(repoPath, sha string) ([]string, error) {
	if strings.TrimSpace(repoPath) == "" || strings.TrimSpace(sha) == "" {
		return nil, fmt.Errorf("missing repository path or commit SHA")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := gitutil.Run(ctx, repoPath, "diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "-m", sha)
	if err != nil {
		return nil, err
	}
	return sanitizeChangedFiles(strings.Split(out, "\n")), nil
}

type fixRoundSelection struct {
	ids    map[string]bool
	source string
}

func selectionForFixRound(fixRound *db.StepRound, rounds []*db.StepRound) fixRoundSelection {
	if fixRound == nil {
		return fixRoundSelection{}
	}
	for _, round := range rounds {
		if round == nil || round.Round != fixRound.Round-1 || round.SelectedFindingIDs == nil {
			continue
		}
		var ids []string
		if err := json.Unmarshal([]byte(*round.SelectedFindingIDs), &ids); err != nil {
			return fixRoundSelection{}
		}
		selected := make(map[string]bool, len(ids))
		for _, id := range ids {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				selected[trimmed] = true
			}
		}
		out := fixRoundSelection{ids: selected}
		if round.SelectionSource != nil {
			out.source = *round.SelectionSource
		}
		return out
	}
	return fixRoundSelection{}
}

func matchingFixDetail(raw string, findingID string, selected map[string]bool) (*FixResolutionDetail, bool) {
	var payload fixResolutionPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, true
	}
	degraded := len(payload.Degraded) > 0
	seen := map[string]bool{}
	var matched *FixResolutionDetail
	for _, detail := range payload.Resolutions {
		id := strings.TrimSpace(detail.FindingID)
		if id == "" || strings.TrimSpace(detail.AppliedSolution) == "" || strings.TrimSpace(detail.WhyThisSolution) == "" || len(detail.ChangedFiles) == 0 {
			degraded = true
			continue
		}
		if selected != nil && !selected[id] {
			degraded = true
			continue
		}
		if seen[id] {
			degraded = true
			continue
		}
		seen[id] = true
		if id == findingID {
			detail.FindingID = id
			detail := detail
			matched = &detail
		}
	}
	if selected != nil && selected[findingID] && !seen[findingID] {
		degraded = true
	}
	return matched, degraded
}

func sanitizeChangedFiles(files []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, file := range files {
		clean := sanitizeChangedFile(file)
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}

func classifyEntry(entry *Entry, latest *types.Findings, latestRound int, reviewStatus types.StepStatus, entriesByID map[string]*Entry) {
	action := entry.Finding.Action
	if action == "" {
		action = types.ActionAutoFix
	}
	if action == types.ActionNoOp {
		entry.Outcome = OutcomeInformational
		entry.OutcomeText = "Review marked this finding as no action required."
		entry.Verification = "no action required"
		entry.VerifierSource = "review finding action"
		entry.EvidenceQuality = "structured"
		return
	}
	if entry.Decision != nil {
		switch entry.Decision.Action {
		case db.ReviewResolutionDecisionApprove, db.ReviewResolutionDecisionSkip, db.ReviewResolutionDecisionPolicyAccept:
			entry.Outcome = OutcomeAccepted
			entry.OutcomeText = "Persisted Review terminal decision accepted the finding without a fix."
			entry.Verification = "accepted without fix by " + sanitizeShort(entry.Decision.ActorSource)
			entry.VerifierSource = "review terminal decision"
			entry.EvidenceQuality = "structured"
			return
		case db.ReviewResolutionDecisionNoOp:
			entry.Outcome = OutcomeInformational
			entry.OutcomeText = "Persisted Review decision marked the finding as informational."
			entry.Verification = "no action required"
			entry.VerifierSource = "review terminal decision"
			entry.EvidenceQuality = "structured"
			return
		}
	}
	if latest != nil && latestRound > entry.LastSeenRound && !containsFindingID(latest, entry.Finding.ID) && !hasNewActionableFindingAfterFix(latest, entriesByID, entry.FixRound) && entry.FixRound > 0 {
		entry.Outcome = OutcomeResolved
		entry.OutcomeText = "Comparable follow-up Review round no longer reported this normalized finding ID."
		entry.Verification = "finding absent from follow-up Review output"
		entry.FollowupRound = latestRound
		entry.ScopeNote = "same Review step run; normalized ID absent from later parsed Review output"
		entry.VerifierSource = "follow-up review"
		if entry.EvidenceQuality == "" {
			entry.EvidenceQuality = "inferred"
		}
		return
	}
	entry.Outcome = OutcomeStillOpen
	if reviewStatus == types.StepStatusFailed {
		entry.OutcomeText = "Review stopped before terminal acceptance or comparable resolved evidence was recorded."
	} else {
		entry.OutcomeText = "No persisted acceptance or comparable resolved evidence was recorded."
	}
	entry.Verification = "verification inconclusive"
	entry.ScopeNote = "no comparable parsed follow-up evidence"
	entry.VerifierSource = "report classifier"
	if entry.EvidenceQuality == "" {
		entry.EvidenceQuality = "unavailable"
	}
}

func evidenceReference(entry *Entry) string {
	if entry.Decision != nil {
		if entry.Decision.RoundID != "" {
			return "persisted review resolution decision " + sanitizeShort(entry.Decision.RoundID)
		}
		return "persisted review resolution decision"
	}
	switch entry.Outcome {
	case OutcomeResolved:
		if entry.FixRound > 0 && entry.FollowupRound > 0 {
			return fmt.Sprintf("fix round %d and follow-up Review round %d", entry.FixRound, entry.FollowupRound)
		}
	case OutcomeInformational:
		if entry.FirstRound > 0 {
			return fmt.Sprintf("Review round %d finding action", entry.FirstRound)
		}
	case OutcomeStillOpen:
		if entry.LastSeenRound > 0 {
			return fmt.Sprintf("latest Review evidence round %d", entry.LastSeenRound)
		}
	}
	if entry.FixRound > 0 {
		return fmt.Sprintf("fix round %d", entry.FixRound)
	}
	if entry.FirstRound > 0 {
		return fmt.Sprintf("Review round %d", entry.FirstRound)
	}
	return "not recorded"
}

func containsFindingID(findings *types.Findings, id string) bool {
	for _, item := range findings.Items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func hasNewActionableFindingAfterFix(findings *types.Findings, entriesByID map[string]*Entry, fixRound int) bool {
	if findings == nil || fixRound == 0 {
		return false
	}
	for _, item := range findings.Items {
		if item.Action == types.ActionNoOp {
			continue
		}
		entry := entriesByID[item.ID]
		if entry == nil || entry.FirstRound >= fixRound {
			return true
		}
	}
	return false
}

func classifyReportStatus(runStatus types.RunStatus, reviewStatus types.StepStatus, snap Snapshot) string {
	if snap.Degraded {
		return db.ReviewResolutionStatusDegraded
	}
	counts := CountEntries(snap.Entries)
	if counts.StillOpen > 0 {
		if runStatus == types.RunRunning || reviewStatus == types.StepStatusRunning || reviewStatus == types.StepStatusAwaitingApproval || reviewStatus == types.StepStatusFixing || reviewStatus == types.StepStatusFixReview {
			return db.ReviewResolutionStatusInProgress
		}
		return db.ReviewResolutionStatusIncomplete
	}
	if runStatus == types.RunCompleted && isTerminalReviewStatus(reviewStatus) {
		return db.ReviewResolutionStatusFinal
	}
	return db.ReviewResolutionStatusInProgress
}

func isTerminalReviewStatus(status types.StepStatus) bool {
	return status == types.StepStatusCompleted || status == types.StepStatusSkipped
}

type Counts struct {
	Resolved      int
	Accepted      int
	Informational int
	StillOpen     int
}

func CountEntries(entries []Entry) Counts {
	var c Counts
	for _, entry := range entries {
		switch entry.Outcome {
		case OutcomeResolved:
			c.Resolved++
		case OutcomeAccepted:
			c.Accepted++
		case OutcomeInformational:
			c.Informational++
		default:
			c.StillOpen++
		}
	}
	return c
}

func reportMetadata(snap Snapshot, watermark, contentHash string) db.ReviewResolutionReport {
	counts := CountEntries(snap.Entries)
	return db.ReviewResolutionReport{
		RunID:              snap.RunID,
		ReportPath:         snap.ReportPath,
		Status:             snap.ReportStatus,
		ResolvedCount:      counts.Resolved,
		AcceptedCount:      counts.Accepted,
		InformationalCount: counts.Informational,
		StillOpenCount:     counts.StillOpen,
		ReportVersion:      reportVersion,
		EntryCount:         len(snap.Entries),
		SourceRoundStart:   snap.SourceRoundFrom,
		SourceRoundEnd:     snap.SourceRoundTo,
		SourceWatermark:    watermark,
		ContentHash:        contentHash,
		LastRefreshResult:  lastRefreshResult(snap),
		FirstGeneratedAt:   snap.FirstGenerated,
		LastRefreshedAt:    snap.LastRefreshed,
		FinalizedAt:        snap.FinalizedAt,
	}
}

func lastRefreshResult(snap Snapshot) string {
	if snap.Degraded {
		return "degraded"
	}
	return "ok"
}

func sourceWatermark(rounds []*db.StepRound, decisions []*db.ReviewResolutionDecision) string {
	type roundMark struct {
		ID           string
		Round        int
		Trigger      string
		Findings     *string
		UserFindings *string
		Selected     *string
		FixSummary   *string
		FixCommit    *string
		NoCommit     *string
		Resolution   *string
	}
	var payload struct {
		Rounds    []roundMark
		Decisions []*db.ReviewResolutionDecision
	}
	for _, round := range rounds {
		payload.Rounds = append(payload.Rounds, roundMark{
			ID: round.ID, Round: round.Round, Trigger: round.Trigger,
			Findings: round.FindingsJSON, UserFindings: round.UserFindingsJSON,
			Selected: round.SelectedFindingIDs, FixSummary: round.FixSummary,
			FixCommit: round.FixCommitSHA, NoCommit: round.NoCommitReason,
			Resolution: round.FixResolutionDetailsJSON,
		})
	}
	payload.Decisions = decisions
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func writeReportTemp(path string, data []byte) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(dir, ".review-resolution-*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer func() {
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	out := tmpPath
	tmpPath = ""
	return out, nil
}

func promoteReportTemp(tmpPath, path string) error {
	return os.Rename(tmpPath, path)
}

func removeReportIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale review resolution report: %w", err)
	}
	return nil
}
