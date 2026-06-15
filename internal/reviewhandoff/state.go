package reviewhandoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

const (
	StateVersion = 1

	ProcessedPending    = "pending"
	ProcessedApprove    = "approve"
	ProcessedFix        = "fix"
	ProcessedSkip       = "skip"
	ProcessedAutomation = "automation"
	ProcessedAutoFix    = "auto_fix"

	DecisionSourceFile       = "file"
	DecisionSourceAutomation = "automation"
	DecisionSourceAutoFix    = "auto_fix"

	SolutionSourceUser                  = "user"
	SolutionSourceDefaultRecommendation = "default_recommendation"
	SolutionSourceNone                  = "none"
)

// State is persisted as step_results.review_handoff_json.
type State struct {
	Version                int               `json:"version"`
	RelativePath           string            `json:"relative_path"`
	CycleID                string            `json:"cycle_id"`
	FindingDigest          string            `json:"finding_digest"`
	GeneratedContentDigest string            `json:"generated_content_digest"`
	DefaultRecommendations map[string]string `json:"default_recommendations,omitempty"`
	ProcessedAction        string            `json:"processed_action"`
	ProcessedAt            *int64            `json:"processed_at"`
	DecisionSource         string            `json:"decision_source,omitempty"`
	Decisions              []Decision        `json:"decisions,omitempty"`
	UpdatedAt              int64             `json:"updated_at"`
}

// Decision is the durable per-finding audit record.
type Decision struct {
	FindingID      string `json:"finding_id"`
	Action         string `json:"action"`
	Solution       string `json:"solution,omitempty"`
	SolutionSource string `json:"solution_source"`
	DecisionSource string `json:"decision_source"`
	ProcessedAt    int64  `json:"processed_at"`
}

type Metadata struct {
	Marker          string
	RunID           string
	Step            string
	Status          string
	Branch          string
	ReviewCycleID   string
	FindingDigest   string
	ReviewFile      string
	ProcessedAction string
	ProcessedAt     string
}

type Response struct {
	FindingID string
	Action    string
	Solution  string
}

type ProcessResult struct {
	Action       types.ApprovalAction
	FindingIDs   []string
	Instructions map[string]string
	Decisions    []Decision
	State        State
}

func NewState(relativePath, cycleID, findingDigest, generatedDigest string, now int64) State {
	return State{
		Version:                StateVersion,
		RelativePath:           filepath.ToSlash(relativePath),
		CycleID:                cycleID,
		FindingDigest:          findingDigest,
		GeneratedContentDigest: generatedDigest,
		ProcessedAction:        ProcessedPending,
		UpdatedAt:              now,
	}
}

func (s State) Validate() error {
	if s.Version != StateVersion {
		return fmt.Errorf("unsupported review handoff state version %d", s.Version)
	}
	if strings.TrimSpace(s.RelativePath) == "" {
		return errors.New("review handoff relative path is required")
	}
	if filepath.IsAbs(s.RelativePath) || strings.HasPrefix(filepath.Clean(s.RelativePath), "..") {
		return fmt.Errorf("review handoff relative path escapes repository: %s", s.RelativePath)
	}
	if s.CycleID == "" {
		return errors.New("review cycle id is required")
	}
	if s.FindingDigest == "" {
		return errors.New("finding digest is required")
	}
	switch s.ProcessedAction {
	case ProcessedPending, ProcessedApprove, ProcessedFix, ProcessedSkip, ProcessedAutomation, ProcessedAutoFix:
	default:
		return fmt.Errorf("unsupported processed action %q", s.ProcessedAction)
	}
	return nil
}

func ParseState(raw string) (State, error) {
	var state State
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return State{}, err
	}
	return state, state.Validate()
}

func (s State) JSON() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func MarkAutomation(state State, findings types.Findings, source string, action types.ApprovalAction, selectedIDs []string, instructions map[string]string, now int64) State {
	state.ProcessedAction = ProcessedAutomation
	if source == DecisionSourceAutoFix {
		state.ProcessedAction = ProcessedAutoFix
	}
	if action == types.ActionApprove {
		state.ProcessedAction = ProcessedApprove
	}
	if action == types.ActionFix {
		state.ProcessedAction = ProcessedFix
	}
	if action == types.ActionSkip {
		state.ProcessedAction = ProcessedSkip
	}
	state.ProcessedAt = &now
	state.DecisionSource = source
	state.UpdatedAt = now
	selected := map[string]bool{}
	for _, id := range selectedIDs {
		selected[id] = true
	}
	state.Decisions = nil
	for _, item := range findings.Items {
		decisionAction := string(types.ActionApprove)
		solutionSource := SolutionSourceNone
		solution := ""
		if action == types.ActionFix && (len(selected) == 0 || selected[item.ID]) {
			decisionAction = "fix"
			solution = strings.TrimSpace(instructions[item.ID])
			if solution != "" {
				solutionSource = SolutionSourceUser
			}
		} else if action == types.ActionSkip {
			decisionAction = "skip"
		} else {
			decisionAction = "accept"
		}
		state.Decisions = append(state.Decisions, Decision{
			FindingID:      item.ID,
			Action:         decisionAction,
			Solution:       solution,
			SolutionSource: solutionSource,
			DecisionSource: source,
			ProcessedAt:    now,
		})
	}
	return state
}
