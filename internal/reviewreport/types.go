package reviewreport

import (
	"github.com/kunchenguid/no-mistakes/internal/types"
)

const reportVersion = "1"

type EntryOutcome string

const (
	OutcomeResolved      EntryOutcome = "resolved"
	OutcomeAccepted      EntryOutcome = "accepted_without_fix"
	OutcomeInformational EntryOutcome = "informational"
	OutcomeStillOpen     EntryOutcome = "still_open"
)

type Snapshot struct {
	RunID           string
	RepoIdentifier  string
	Branch          string
	BaseSHA         string
	HeadSHA         string
	ReviewStatus    string
	ReportStatus    string
	ReportPath      string
	FirstGenerated  int64
	LastRefreshed   int64
	FinalizedAt     *int64
	SourceRoundFrom *int
	SourceRoundTo   *int
	Entries         []Entry
	Degraded        bool
}

type Entry struct {
	Finding           types.Finding
	Outcome           EntryOutcome
	OutcomeText       string
	Decision          *Decision
	FirstRound        int
	LastSeenRound     int
	SelectionSource   string
	FixRound          int
	FixSummary        string
	FixCommitSHA      string
	NoCommitReason    string
	SolutionSource    string
	AppliedSolution   string
	Rationale         string
	ChangedFiles      []string
	Verification      string
	FollowupRound     int
	ScopeNote         string
	VerifierSource    string
	EvidenceReference string
	EvidenceQuality   string
	Degraded          bool
}

type Decision struct {
	Action      string
	ActorSource string
	Reason      string
	CreatedAt   int64
	RoundID     string
}

type fixResolutionPayload struct {
	Summary     string                `json:"summary,omitempty"`
	Resolutions []FixResolutionDetail `json:"resolutions,omitempty"`
	Degraded    []string              `json:"degraded,omitempty"`
}

type FixResolutionDetail struct {
	FindingID       string   `json:"finding_id"`
	AppliedSolution string   `json:"applied_solution"`
	WhyThisSolution string   `json:"why_this_solution"`
	ChangedFiles    []string `json:"changed_files"`
}
