package reviewhandoff

import (
	"fmt"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

const (
	StepReview = "review"

	FileNamePrefix = "review-issues-"
	FileNameSuffix = ".md"
	FenceLanguage  = "no-mistakes-response"

	MaxFileBytes     int64 = 1 << 20
	MaxSolutionBytes       = 16 << 10

	ActionFix    = "fix"
	ActionAccept = "accept"
	ActionSkip   = "skip"

	ProcessedPending = "pending"
	ProcessedFix     = "fix"
	ProcessedApprove = "approve"
	ProcessedSkip    = "skip"

	FinalNoFindingsText = "No remaining review findings."
)

// Metadata is the YAML front matter persisted at the top of the handoff file.
type Metadata struct {
	RunID               string  `yaml:"run_id"`
	RunShortID          string  `yaml:"run_short_id"`
	Branch              string  `yaml:"branch"`
	Step                string  `yaml:"step"`
	Status              string  `yaml:"status"`
	ReviewCycleRevision string  `yaml:"review_cycle_revision"`
	ReviewResultHash    string  `yaml:"review_result_hash"`
	ProcessedAt         *string `yaml:"processed_at"`
	ProcessedAction     string  `yaml:"processed_action"`
}

// Summary contains the aggregate severity counts rendered for humans.
type Summary struct {
	Total      int
	Severities map[string]int
}

// FindingEntry is the review finding projection rendered into Markdown.
type FindingEntry struct {
	ID                    string
	Severity              string
	File                  string
	Line                  int
	Issue                 string
	Context               string
	Recommendations       []string
	DefaultResponseAction string
}

// ResponseBlock is the machine-readable user answer parsed from a fenced block.
type ResponseBlock struct {
	ID       string `yaml:"id"`
	Action   string `yaml:"action"`
	Solution string `yaml:"solution"`
}

// HandoffFile is the complete render model for a current or final review file.
type HandoffFile struct {
	Metadata     Metadata
	Summary      Summary
	Findings     []FindingEntry
	Responses    []ResponseBlock
	AuditEntries []AuditEntry
	FinalState   string
}

// LiveState is the authoritative state used to validate a saved handoff file.
type LiveState struct {
	RunID               string
	Branch              string
	Step                string
	Status              string
	ReviewCycleRevision string
	ReviewResultHash    string
	Findings            []FindingEntry
}

// ValidationError is intentionally short enough to show in compact terminals.
type ValidationError struct {
	Path       string
	Summary    string
	FirstError string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Path == "" {
		return e.FirstError
	}
	if e.FirstError == "" {
		return e.Path + ": " + e.Summary
	}
	return e.Path + ": " + e.FirstError
}

func validationFailure(path, first string) *ValidationError {
	return &ValidationError{
		Path:       path,
		Summary:    "review file validation failed",
		FirstError: first,
	}
}

// ProcessedDecision is the existing gate decision derived from a valid file or
// mirrored automation response.
type ProcessedDecision struct {
	Source             string
	ExecutedAction     string
	Actions            map[string]string
	SelectedFindingIDs []string
	Instructions       map[string]string
	AddedFindings      []types.Finding
	ProcessedAt        time.Time
}

// AuditEntry is the compact per-cycle decision record rendered in the final
// no-findings audit file.
type AuditEntry struct {
	FindingID     string
	Severity      string
	Issue         string
	Action        string
	Solution      string
	Selection     string
	FixSummary    string
	Source        string
	ProcessedTime string
}

// ShortRunID returns the stable filename prefix token for a run.
func ShortRunID(runID string) string {
	clean := strings.TrimSpace(runID)
	if len(clean) <= 8 {
		return clean
	}
	return clean[:8]
}

// FileName returns review-issues-<run-short-id>.md.
func FileName(runID string) string {
	return FileNamePrefix + ShortRunID(runID) + FileNameSuffix
}

// NewMetadata builds initial pending front matter for a review file.
func NewMetadata(state LiveState, runShortID string) Metadata {
	if runShortID == "" {
		runShortID = ShortRunID(state.RunID)
	}
	return Metadata{
		RunID:               state.RunID,
		RunShortID:          runShortID,
		Branch:              state.Branch,
		Step:                state.Step,
		Status:              state.Status,
		ReviewCycleRevision: state.ReviewCycleRevision,
		ReviewResultHash:    state.ReviewResultHash,
		ProcessedAt:         nil,
		ProcessedAction:     ProcessedPending,
	}
}

func SummaryFor(entries []FindingEntry) Summary {
	s := Summary{Total: len(entries), Severities: make(map[string]int)}
	for _, entry := range entries {
		key := strings.TrimSpace(entry.Severity)
		if key == "" {
			key = "unknown"
		}
		s.Severities[key]++
	}
	return s
}

// EntriesFromFindings adapts existing structured findings to the handoff model.
func EntriesFromFindings(findings types.Findings) ([]FindingEntry, error) {
	entries := make([]FindingEntry, 0, len(findings.Items))
	seen := make(map[string]bool, len(findings.Items))
	for _, item := range findings.Items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return nil, fmt.Errorf("finding missing id")
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate finding id %q", id)
		}
		seen[id] = true
		recommendations := recommendationOptions(item)
		entries = append(entries, FindingEntry{
			ID:                    id,
			Severity:              item.Severity,
			File:                  item.File,
			Line:                  item.Line,
			Issue:                 item.Description,
			Context:               item.Context,
			Recommendations:       recommendations,
			DefaultResponseAction: defaultResponseAction(item.Action),
		})
	}
	return entries, nil
}

func recommendationOptions(item types.Finding) []string {
	first := strings.TrimSpace(item.SuggestedFix)
	if first == "" {
		first = "Review this finding and address the underlying issue."
	}
	return []string{first}
}

func defaultResponseAction(action string) string {
	switch action {
	case types.ActionAutoFix:
		return ActionFix
	case types.ActionNoOp:
		return ActionSkip
	default:
		return ActionAccept
	}
}

func validResponseAction(action string) bool {
	switch action {
	case ActionFix, ActionAccept, ActionSkip:
		return true
	default:
		return false
	}
}
