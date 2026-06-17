package reviewreport

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

type DeriveInput struct {
	Run              *db.Run
	ReviewStep       *db.StepResult
	Rounds           []*db.StepRound
	ReportPath       string
	GenerationMode   string
	GeneratedAt      int64
	UpdatedAt        int64
	SourceSnapshotAt int64
}

type parsedRound struct {
	round        *db.StepRound
	findings     *types.Findings
	parseErr     error
	selectedIDs  []string
	selectedErr  error
	userFindings *types.Findings
	userErr      error
}

func Derive(input DeriveInput) ReportSnapshot {
	now := time.Now().Unix()
	if input.SourceSnapshotAt == 0 {
		input.SourceSnapshotAt = now
	}
	if input.UpdatedAt == 0 {
		input.UpdatedAt = now
	}
	if input.GeneratedAt == 0 {
		input.GeneratedAt = input.UpdatedAt
	}
	if input.GenerationMode == "" {
		input.GenerationMode = GenerationModeLive
	}

	snapshot := ReportSnapshot{
		Metadata: ReportMetadata{
			Path:             input.ReportPath,
			Status:           StatusCurrent,
			GenerationMode:   input.GenerationMode,
			GeneratedAt:      input.GeneratedAt,
			UpdatedAt:        input.UpdatedAt,
			SourceSnapshotAt: input.SourceSnapshotAt,
		},
		Counts: ZeroSummaryCounts(),
		Generation: GenerationNotes{
			Warnings: []string{"No generation warnings"},
		},
	}
	if input.Run != nil {
		snapshot.Run = RunContext{
			ID:                input.Run.ID,
			Branch:            input.Run.Branch,
			BaseSHA:           input.Run.BaseSHA,
			HeadSHA:           input.Run.HeadSHA,
			RunStatus:         string(input.Run.Status),
			SafeIntentSummary: sanitizeOptional(input.Run.Intent, ValueUnavailable),
		}
		if input.Run.PRURL != nil {
			snapshot.Run.PRURL = SanitizeText(*input.Run.PRURL, ValueUnavailable)
		}
	}
	if input.ReviewStep != nil {
		snapshot.Run.ReviewStatus = string(input.ReviewStep.Status)
		snapshot.SourceEvidence.ReviewStepResultID = input.ReviewStep.ID
	}
	if snapshot.Run.PRURL == "" {
		snapshot.Run.PRURL = ValueUnavailable
	}
	if snapshot.Run.SafeIntentSummary == "" {
		snapshot.Run.SafeIntentSummary = ValueUnavailable
	}
	if snapshot.Run.ReviewStatus == "" {
		snapshot.Run.ReviewStatus = ValueUnavailable
	}
	snapshot.SourceEvidence.SourceSnapshotAt = input.SourceSnapshotAt
	snapshot.SourceEvidence.IntegrityStatus = IntegrityConsistent

	rounds := parseRounds(input.Rounds)
	for _, r := range rounds {
		snapshot.SourceEvidence.IncludedRounds = append(snapshot.SourceEvidence.IncludedRounds, r.round.ID)
		if r.round.FindingsJSON != nil {
			snapshot.SourceEvidence.LatestReviewRound = r.round.ID
		}
		if r.round.IsFixRound() {
			snapshot.SourceEvidence.LatestFixRound = r.round.ID
		}
	}
	if len(rounds) == 0 {
		snapshot.SourceEvidence.IntegrityStatus = IntegrityUnavailable
		snapshot.SourceEvidence.Diagnostics = append(snapshot.SourceEvidence.Diagnostics, "no review rounds recorded")
	}

	selected := selectedEvidence(rounds, &snapshot)
	findings := collectFindings(rounds)
	latestRound := latestFindingsRound(rounds)
	latestFindings := latestParsedFindings(latestRound)
	latestIDs := map[string]bool{}
	if latestFindings != nil {
		for _, item := range latestFindings.Items {
			if item.ID != "" {
				latestIDs[item.ID] = true
			}
		}
	}

	for _, item := range findings {
		reportFinding := reportFinding(item, selected, latestIDs)
		snapshot.Findings = append(snapshot.Findings, reportFinding)
		addFindingCounts(snapshot.Counts, item, reportFinding)
	}
	snapshot.Counts[CountTotalFindings] = len(snapshot.Findings)
	snapshot.FixAttempts = deriveFixAttempts(rounds)
	snapshot.Counts[CountFixAttempts] = len(snapshot.FixAttempts)
	for _, attempt := range snapshot.FixAttempts {
		if attempt.AppliedFix != AppliedFixSummaryMissing && attempt.AppliedFix != ValueUnavailable {
			snapshot.Counts[CountAppliedFixSummaries]++
		}
	}

	integrity := snapshot.SourceEvidence.IntegrityStatus
	snapshot.Latest = deriveLatestOutcome(input.Run, input.ReviewStep, latestRound, latestFindings, integrity, len(snapshot.FixAttempts), hasPostFixReviewAfterLatestFix(rounds, latestRound))
	if snapshot.Latest.Risk == "" {
		snapshot.Latest.Risk = ValueUnavailable
	}
	if snapshot.Latest.Rationale == "" {
		snapshot.Latest.Rationale = ValueUnavailable
	}
	sort.SliceStable(snapshot.Findings, func(i, j int) bool {
		return snapshot.Findings[i].ID < snapshot.Findings[j].ID
	})
	return snapshot
}

func parseRounds(rounds []*db.StepRound) []parsedRound {
	parsed := make([]parsedRound, 0, len(rounds))
	for _, round := range rounds {
		if round == nil {
			continue
		}
		p := parsedRound{round: round}
		if round.FindingsJSON != nil && strings.TrimSpace(*round.FindingsJSON) != "" {
			findings, err := types.ParseFindingsJSON(*round.FindingsJSON)
			p.findings = &findings
			p.parseErr = err
		}
		if round.UserFindingsJSON != nil && strings.TrimSpace(*round.UserFindingsJSON) != "" {
			findings, err := types.ParseFindingsJSON(*round.UserFindingsJSON)
			p.userFindings = &findings
			p.userErr = err
		}
		if round.SelectedFindingIDs != nil && strings.TrimSpace(*round.SelectedFindingIDs) != "" {
			p.selectedErr = json.Unmarshal([]byte(*round.SelectedFindingIDs), &p.selectedIDs)
		}
		parsed = append(parsed, p)
	}
	return parsed
}

type selectionInfo struct {
	selected map[string]string
	skipped  map[string]string
	actor    map[string]string
	invalid  bool
}

func selectedEvidence(rounds []parsedRound, snapshot *ReportSnapshot) selectionInfo {
	info := selectionInfo{
		selected: map[string]string{},
		skipped:  map[string]string{},
		actor:    map[string]string{},
	}
	for _, r := range rounds {
		if r.parseErr != nil {
			snapshot.SourceEvidence.IntegrityStatus = IntegrityPartial
			snapshot.SourceEvidence.Diagnostics = append(snapshot.SourceEvidence.Diagnostics, fmt.Sprintf("round %s findings unreadable", r.round.ID))
			continue
		}
		if r.userErr != nil {
			snapshot.SourceEvidence.IntegrityStatus = IntegrityPartial
			snapshot.SourceEvidence.Diagnostics = append(snapshot.SourceEvidence.Diagnostics, fmt.Sprintf("round %s user findings unreadable", r.round.ID))
		}
		if r.selectedErr != nil {
			info.invalid = true
			snapshot.SourceEvidence.IntegrityStatus = IntegrityInconsistent
			snapshot.SourceEvidence.Diagnostics = append(snapshot.SourceEvidence.Diagnostics, fmt.Sprintf("round %s selected_finding_ids unreadable", r.round.ID))
			continue
		}
		if len(r.selectedIDs) == 0 {
			continue
		}
		knownIDs := findingIDSet(r.findings, r.userFindings)
		source := selectionSource(r.round.SelectionSource)
		for _, id := range r.selectedIDs {
			if id == "" {
				continue
			}
			if len(knownIDs) > 0 && !knownIDs[id] {
				info.invalid = true
				snapshot.SourceEvidence.IntegrityStatus = IntegrityInconsistent
				snapshot.SourceEvidence.Diagnostics = append(snapshot.SourceEvidence.Diagnostics, fmt.Sprintf("round %s selected unknown finding %s", r.round.ID, id))
				continue
			}
			info.selected[id] = r.round.ID
			info.actor[id] = source
		}
		if source == db.RoundSelectionSourceUser && r.findings != nil {
			for _, item := range r.findings.Items {
				_, wasSelected := info.selected[item.ID]
				if item.ID == "" || wasSelected || findingActionIsNoOp(item) {
					continue
				}
				info.skipped[item.ID] = r.round.ID
				info.actor[item.ID] = source
			}
		}
	}
	return info
}

func findingIDSet(findings *types.Findings, userFindings *types.Findings) map[string]bool {
	set := map[string]bool{}
	if findings != nil {
		for _, item := range findings.Items {
			if item.ID != "" {
				set[item.ID] = true
			}
		}
	}
	if userFindings != nil {
		for _, item := range userFindings.Items {
			if item.ID != "" {
				set[item.ID] = true
			}
		}
	}
	return set
}

func collectFindings(rounds []parsedRound) []types.Finding {
	seen := map[string]bool{}
	var findings []types.Finding
	for _, r := range rounds {
		for _, set := range []*types.Findings{r.findings, r.userFindings} {
			if set == nil {
				continue
			}
			for _, item := range set.Items {
				id := item.ID
				if id == "" {
					id = findingFallbackID(item)
				}
				if seen[id] {
					continue
				}
				seen[id] = true
				if item.ID == "" {
					item.ID = id
				}
				findings = append(findings, item)
			}
		}
	}
	return findings
}

func findingFallbackID(item types.Finding) string {
	return fmt.Sprintf("%s:%d:%s", item.File, item.Line, item.Description)
}

func reportFinding(item types.Finding, selected selectionInfo, latestIDs map[string]bool) ReviewFinding {
	id := item.ID
	if id == "" {
		id = findingFallbackID(item)
	}
	selectedRound, wasSelected := selected.selected[id]
	decision := DecisionNotRecorded
	actor := ValueNotRecorded
	evidence := ValueNotRecorded
	if selected.invalid {
		decision = DecisionUnavailable
		evidence = "selection metadata inconsistent"
	} else if latestIDs[id] {
		decision = DecisionStillOpen
		if wasSelected {
			actor = selected.actor[id]
			evidence = "latest review findings; selected in round " + selectedRound
		} else {
			evidence = "latest review findings"
		}
	} else if wasSelected {
		decision = DecisionSelectedForFix
		actor = selected.actor[id]
		evidence = "round " + selectedRound
	} else if findingActionIsNoOp(item) {
		decision = DecisionInformational
		evidence = "finding action no-op"
	} else if roundID, ok := selected.skipped[id]; ok {
		decision = DecisionSkipped
		actor = selected.actor[id]
		evidence = "round " + roundID
	}
	return ReviewFinding{
		ID:               SanitizeText(id, ValueNotRecorded),
		Issue:            SanitizeText(item.Description, ValueUnavailable),
		Severity:         SanitizeText(item.Severity, ValueUnavailable),
		Location:         findingLocation(item),
		Source:           findingSource(item),
		ActionType:       findingActionType(item),
		Context:          SanitizeText(item.Context, ValueUnavailable),
		Recommendation:   SanitizeText(item.SuggestedFix, ValueUnavailable),
		SelectedForFix:   wasSelected && !selected.invalid,
		Decision:         decision,
		DecisionActor:    SanitizeText(actor, ValueNotRecorded),
		DecisionEvidence: SanitizeText(evidence, ValueNotRecorded),
		UserInstructions: SanitizeText(item.UserInstructions, ValueNotRecorded),
	}
}

func addFindingCounts(counts map[string]int, item types.Finding, finding ReviewFinding) {
	if findingActionIsActionable(item) {
		counts[CountActionableFindings]++
	}
	if finding.SelectedForFix {
		counts[CountSelectedForFix]++
	}
	switch finding.Decision {
	case DecisionSelectedForFix:
	case DecisionAccepted:
		counts[CountAccepted]++
	case DecisionSkipped:
		counts[CountSkipped]++
	case DecisionInformational:
		counts[CountInformational]++
	case DecisionDeferred:
		counts[CountDeferred]++
	case DecisionStillOpen:
		counts[CountStillOpen]++
	case DecisionUnavailable:
		counts[CountUnavailable]++
	case DecisionNotRecorded:
		counts[CountDecisionNotRecorded]++
	}
}

func deriveFixAttempts(rounds []parsedRound) []FixAttempt {
	var attempts []FixAttempt
	for i, r := range rounds {
		if !r.round.IsFixRound() {
			continue
		}
		prev := previousReviewRound(rounds, i)
		attempt := FixAttempt{
			Round:            r.round.Round,
			SelectionSource:  ValueNotRecorded,
			AppliedFix:       AppliedFixSummaryMissing,
			Verification:     "follow-up review unavailable",
			Evidence:         "fix round " + r.round.ID,
			LatestFixRoundID: r.round.ID,
		}
		if r.round.FixSummary != nil && strings.TrimSpace(*r.round.FixSummary) != "" {
			attempt.AppliedFix = SanitizeText(*r.round.FixSummary, AppliedFixSummaryMissing)
		}
		if prev != nil {
			attempt.PreviousReviewID = prev.round.ID
			attempt.SelectedFindings = append(attempt.SelectedFindings, prev.selectedIDs...)
			attempt.SelectionSource = selectionSource(prev.round.SelectionSource)
			attempt.UserInstructions, attempt.UserAuthored = selectedUserMetadata(prev.userFindings)
		}
		followUp := &r
		if r.round.FindingsJSON == nil {
			followUp = nextReviewRound(rounds, i)
		}
		if followUp != nil {
			attempt.FollowUpReviewID = followUp.round.ID
			if followUp.parseErr != nil {
				attempt.Verification = "follow-up review unreadable"
			} else if followUp.findings != nil && len(followUp.findings.Items) == 0 {
				attempt.Verification = "follow-up review found no findings"
				attempt.FollowUpReviewOkay = true
			} else if followUp.findings != nil {
				attempt.Verification = "follow-up review still has findings"
			}
		}
		attempt.SelectedFindings = sanitizeStringSlice(attempt.SelectedFindings)
		attempt.UserAuthored = sanitizeStringSlice(attempt.UserAuthored)
		attempts = append(attempts, attempt)
	}
	return attempts
}

func deriveLatestOutcome(run *db.Run, step *db.StepResult, latest parsedRound, findings *types.Findings, integrity string, fixAttempts int, postFixReviewAvailable bool) LatestReviewOutcome {
	out := LatestReviewOutcome{Risk: ValueUnavailable, Rationale: ValueUnavailable}
	if latest.round != nil && latest.round.FindingsJSON != nil && latest.parseErr != nil {
		out.Outcome = LatestOutcomeFinalFindingsUnreadable
		out.Evidence = "latest review findings could not be parsed"
		return out
	}
	if integrity == IntegrityInconsistent || integrity == IntegrityPartial {
		out.Outcome = LatestOutcomeReviewDataInconsistent
		out.Evidence = "stored review data is internally inconsistent"
		return out
	}
	if step != nil && (step.Status == types.StepStatusAwaitingApproval || step.Status == types.StepStatusFixReview) {
		out.Outcome = LatestOutcomeAwaitingUserDecision
		out.Evidence = "review step is awaiting a user decision"
		return out
	}
	if latest.round == nil || latest.round.FindingsJSON == nil {
		if fixAttempts > 0 {
			out.Outcome = LatestOutcomeReviewResolutionIncomplete
			out.Evidence = "no post-fix review findings are available"
		} else {
			out.Outcome = LatestOutcomeFinalFindingsUnavailable
			out.Evidence = "final review findings are unavailable"
		}
		return out
	}
	if fixAttempts > 0 && run != nil && (run.Status == types.RunFailed || run.Status == types.RunCancelled) && !postFixReviewAvailable {
		out.Outcome = LatestOutcomeReviewResolutionIncomplete
		out.Evidence = "run ended before a trustworthy post-fix review completed"
		return out
	}
	out.Risk = SanitizeText(findings.RiskLevel, ValueUnavailable)
	out.Rationale = SanitizeText(findings.RiskRationale, ValueUnavailable)
	out.Evidence = "round " + latest.round.ID
	if isNoReviewableChanges(findings) {
		out.Outcome = LatestOutcomeNoReviewableChanges
		return out
	}
	if len(findings.Items) == 0 {
		out.Outcome = LatestOutcomeNoIssuesRemain
		return out
	}
	out.Outcome = LatestOutcomeUnresolvedFindingsRemain
	return out
}

func hasPostFixReviewAfterLatestFix(rounds []parsedRound, latest parsedRound) bool {
	if latest.round == nil || latest.round.FindingsJSON == nil || latest.parseErr != nil {
		return false
	}
	latestReviewIndex := -1
	latestFixIndex := -1
	for i, r := range rounds {
		if r.round == nil {
			continue
		}
		if r.round.ID == latest.round.ID {
			latestReviewIndex = i
		}
		if r.round.IsFixRound() {
			latestFixIndex = i
		}
	}
	return latestFixIndex >= 0 && latestReviewIndex >= latestFixIndex
}

func latestFindingsRound(rounds []parsedRound) parsedRound {
	var latest parsedRound
	for _, r := range rounds {
		if r.round.FindingsJSON != nil {
			latest = r
		}
	}
	return latest
}

func latestParsedFindings(round parsedRound) *types.Findings {
	if round.parseErr != nil {
		return nil
	}
	return round.findings
}

func previousReviewRound(rounds []parsedRound, before int) *parsedRound {
	for i := before - 1; i >= 0; i-- {
		if rounds[i].round.FindingsJSON != nil {
			return &rounds[i]
		}
	}
	return nil
}

func nextReviewRound(rounds []parsedRound, after int) *parsedRound {
	for i := after + 1; i < len(rounds); i++ {
		if rounds[i].round.FindingsJSON != nil {
			return &rounds[i]
		}
	}
	return nil
}

func selectedUserMetadata(findings *types.Findings) (string, []string) {
	if findings == nil {
		return ValueNotRecorded, nil
	}
	hasInstructions := false
	var userIDs []string
	for _, item := range findings.Items {
		if strings.TrimSpace(item.UserInstructions) != "" {
			hasInstructions = true
		}
		if item.Source == types.FindingSourceUser && item.ID != "" {
			userIDs = append(userIDs, item.ID)
		}
	}
	if hasInstructions {
		return "present", userIDs
	}
	return ValueNotRecorded, userIDs
}

func sanitizeStringSlice(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		clean = append(clean, SanitizeText(value, ValueNotRecorded))
	}
	sort.Strings(clean)
	return clean
}

func selectionSource(source *string) string {
	if source == nil || strings.TrimSpace(*source) == "" {
		return ValueNotRecorded
	}
	return SanitizeText(*source, ValueNotRecorded)
}

func findingLocation(item types.Finding) string {
	file := SanitizeText(item.File, ValueUnavailable)
	if item.Line > 0 && file != ValueUnavailable {
		return fmt.Sprintf("%s:%d", file, item.Line)
	}
	return file
}

func findingSource(item types.Finding) string {
	if strings.TrimSpace(item.Source) == "" {
		return ValueNotRecorded
	}
	return SanitizeText(item.Source, ValueNotRecorded)
}

func findingActionType(item types.Finding) string {
	if strings.TrimSpace(item.Action) == "" {
		return ValueNotRecorded
	}
	return SanitizeText(item.Action, ValueUnavailable)
}

func findingActionIsNoOp(item types.Finding) bool {
	return strings.TrimSpace(item.Action) == types.ActionNoOp
}

func findingActionIsActionable(item types.Finding) bool {
	action := strings.TrimSpace(item.Action)
	return action != "" && action != types.ActionNoOp
}

func sanitizeOptional(value *string, missing string) string {
	if value == nil {
		return missing
	}
	return SanitizeText(*value, missing)
}

func isNoReviewableChanges(findings *types.Findings) bool {
	if findings == nil {
		return false
	}
	return len(findings.Items) == 0 && strings.Contains(strings.ToLower(findings.Summary), "no reviewable")
}
