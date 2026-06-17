package reviewreport

const (
	ContractVersion = "review-resolution-report/v1"

	StatusCurrent     = "current"
	StatusStale       = "stale"
	StatusUnavailable = "unavailable"
	StatusError       = "error"

	GenerationModeLive        = "live"
	GenerationModeRegenerated = "regenerated"

	LatestOutcomeNoIssuesRemain             = "no issues remain"
	LatestOutcomeUnresolvedFindingsRemain   = "unresolved findings remain"
	LatestOutcomeNoReviewableChanges        = "no reviewable changes"
	LatestOutcomeAwaitingUserDecision       = "awaiting user decision"
	LatestOutcomeFinalFindingsUnavailable   = "final findings unavailable"
	LatestOutcomeFinalFindingsUnreadable    = "final findings unreadable"
	LatestOutcomeReviewDataInconsistent     = "review data inconsistent"
	LatestOutcomeReviewResolutionIncomplete = "review resolution incomplete"

	CountTotalFindings       = "total_findings"
	CountActionableFindings  = "actionable_findings"
	CountSelectedForFix      = "selected_for_fix"
	CountFixAttempts         = "fix_attempts"
	CountAppliedFixSummaries = "applied_fix_summaries"
	CountAccepted            = "accepted"
	CountSkipped             = "skipped"
	CountInformational       = "informational"
	CountDeferred            = "deferred"
	CountStillOpen           = "still_open"
	CountUnavailable         = "unavailable"
	CountDecisionNotRecorded = "decision_not_recorded"

	DecisionSelectedForFix          = "Selected for fix"
	DecisionAccepted                = "Accepted"
	DecisionSkipped                 = "Skipped"
	DecisionInformational           = "Informational"
	DecisionDeferred                = "Deferred"
	DecisionStillOpen               = "Still open"
	DecisionNotRecorded             = "Decision not recorded"
	DecisionUnavailable             = "Unavailable"
	IntegrityConsistent             = "consistent"
	IntegrityInconsistent           = "inconsistent"
	IntegrityPartial                = "partial"
	IntegrityUnavailable            = "unavailable"
	AppliedFixSummaryMissing        = "fix applied, no summary recorded"
	AppliedFixSummaryDisplayMissing = "fix applied (no summary recorded)"
	AppliedFixSummaryDisplayOmitted = "fix applied (summary omitted)"
	ValueUnavailable                = "unavailable"
	ValueNotRecorded                = "not recorded"
)

var (
	AllowedStatuses = []string{
		StatusCurrent,
		StatusStale,
		StatusUnavailable,
		StatusError,
	}

	AllowedLatestOutcomes = []string{
		LatestOutcomeNoIssuesRemain,
		LatestOutcomeUnresolvedFindingsRemain,
		LatestOutcomeNoReviewableChanges,
		LatestOutcomeAwaitingUserDecision,
		LatestOutcomeFinalFindingsUnavailable,
		LatestOutcomeFinalFindingsUnreadable,
		LatestOutcomeReviewDataInconsistent,
		LatestOutcomeReviewResolutionIncomplete,
	}

	SummaryCountKeys = []string{
		CountTotalFindings,
		CountActionableFindings,
		CountSelectedForFix,
		CountFixAttempts,
		CountAppliedFixSummaries,
		CountAccepted,
		CountSkipped,
		CountInformational,
		CountDeferred,
		CountStillOpen,
		CountUnavailable,
		CountDecisionNotRecorded,
	}
)

func ZeroSummaryCounts() map[string]int {
	counts := make(map[string]int, len(SummaryCountKeys))
	for _, key := range SummaryCountKeys {
		counts[key] = 0
	}
	return counts
}

// ReportSnapshot is the sanitized, renderable report model derived from stored
// run and review-round data.
type ReportSnapshot struct {
	Metadata       ReportMetadata
	Run            RunContext
	Counts         map[string]int
	Latest         LatestReviewOutcome
	Findings       []ReviewFinding
	FixAttempts    []FixAttempt
	SourceEvidence SourceEvidence
	Generation     GenerationNotes
}

type ReportMetadata struct {
	Path             string
	Status           string
	GenerationMode   string
	GeneratedAt      int64
	UpdatedAt        int64
	SourceSnapshotAt int64
}

type RunContext struct {
	ID                string
	Branch            string
	BaseSHA           string
	HeadSHA           string
	RunStatus         string
	ReviewStatus      string
	SafeIntentSummary string
	PRURL             string
}

type LatestReviewOutcome struct {
	Outcome   string
	Evidence  string
	Risk      string
	Rationale string
}

type ReviewFinding struct {
	ID               string
	Issue            string
	Severity         string
	Location         string
	Source           string
	ActionType       string
	Context          string
	Recommendation   string
	SelectedForFix   bool
	Decision         string
	DecisionActor    string
	DecisionEvidence string
	UserInstructions string
}

type FixAttempt struct {
	Round              int
	SelectedFindings   []string
	SelectionSource    string
	UserInstructions   string
	UserAuthored       []string
	AppliedFix         string
	Verification       string
	Evidence           string
	LatestFixRoundID   string
	PreviousReviewID   string
	FollowUpReviewID   string
	FollowUpReviewOkay bool
}

type SourceEvidence struct {
	ReviewStepResultID string
	IncludedRounds     []string
	LatestReviewRound  string
	LatestFixRound     string
	SourceSnapshotAt   int64
	IntegrityStatus    string
	Diagnostics        []string
}

type GenerationNotes struct {
	Warnings []string
}
