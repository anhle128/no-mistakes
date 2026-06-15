package reviewhandoff

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

type ProcessInput struct {
	Root     string
	RunID    string
	Branch   string
	Status   types.StepStatus
	State    State
	Findings types.Findings
	Now      int64
}

type ProcessPlan struct {
	Result           ProcessResult
	Path             string
	OriginalContent  []byte
	ProcessedContent []byte
}

type ProcessedFilePlan struct {
	State            State
	Path             string
	OriginalContent  []byte
	ProcessedContent []byte
}

func PrepareProcess(input ProcessInput) (*ProcessPlan, error) {
	if err := input.State.Validate(); err != nil {
		return nil, err
	}
	path, err := SafeJoin(input.Root, input.State.RelativePath)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read review handoff file %s: %w", input.State.RelativePath, err)
	}
	meta, responses, err := Parse(content)
	if err != nil {
		return nil, err
	}
	if err := validateMetadata(meta, input); err != nil {
		return nil, err
	}
	decisions, fixIDs, instructions, err := decisionsFromResponses(responses, input.Findings, input.State.DefaultRecommendations, input.Now)
	if err != nil {
		return nil, err
	}
	if len(input.Findings.Items) == 0 && len(decisions) == 0 && len(input.State.Decisions) > 0 {
		decisions = append(decisions, input.State.Decisions...)
	}
	action := types.ActionApprove
	processedAction := ProcessedApprove
	if len(fixIDs) > 0 {
		action = types.ActionFix
		processedAction = ProcessedFix
	}
	updatedContent := stampProcessed(content, processedAction, input.Now, decisions)
	state := input.State
	state.ProcessedAction = processedAction
	state.ProcessedAt = &input.Now
	state.DecisionSource = DecisionSourceFile
	state.Decisions = decisions
	state.UpdatedAt = input.Now
	state.GeneratedContentDigest = ContentDigest(updatedContent)
	return &ProcessPlan{
		Result: ProcessResult{
			Action:       action,
			FindingIDs:   fixIDs,
			Instructions: instructions,
			Decisions:    decisions,
			State:        state,
		},
		Path:             path,
		OriginalContent:  append([]byte(nil), content...),
		ProcessedContent: updatedContent,
	}, nil
}

func Process(input ProcessInput) (*ProcessResult, error) {
	plan, err := PrepareProcess(input)
	if err != nil {
		return nil, err
	}
	if err := CommitProcess(plan); err != nil {
		return nil, err
	}
	return &plan.Result, nil
}

func CommitProcess(plan *ProcessPlan) error {
	if plan == nil {
		return fmt.Errorf("process plan is required")
	}
	if err := writeFileAtomic(plan.Path, plan.ProcessedContent); err != nil {
		return fmt.Errorf("stamp review handoff file: %w", err)
	}
	return nil
}

func RestoreProcess(plan *ProcessPlan) error {
	if plan == nil {
		return fmt.Errorf("process plan is required")
	}
	if err := writeFileAtomic(plan.Path, plan.OriginalContent); err != nil {
		return fmt.Errorf("restore review handoff file: %w", err)
	}
	return nil
}

func validateMetadata(meta Metadata, input ProcessInput) error {
	checks := []struct {
		name string
		got  string
		want string
	}{
		{"run_id", meta.RunID, input.RunID},
		{"step", meta.Step, string(types.StepReview)},
		{"status", meta.Status, string(input.Status)},
		{"branch", meta.Branch, input.Branch},
		{"review_cycle_id", meta.ReviewCycleID, input.State.CycleID},
		{"finding_digest", meta.FindingDigest, input.State.FindingDigest},
		{"review_file", filepath.ToSlash(meta.ReviewFile), filepath.ToSlash(input.State.RelativePath)},
		{"processed_action", meta.ProcessedAction, ProcessedPending},
		{"processed_at", meta.ProcessedAt, ""},
	}
	for _, c := range checks {
		if c.got != c.want {
			return fmt.Errorf("review handoff metadata %s = %q, want %q", c.name, c.got, c.want)
		}
	}
	if got := FindingDigest(input.Findings); got != input.State.FindingDigest {
		return fmt.Errorf("active findings digest = %q, want %q", got, input.State.FindingDigest)
	}
	if input.State.ProcessedAction != ProcessedPending || input.State.ProcessedAt != nil {
		return fmt.Errorf("review handoff state is already processed")
	}
	return nil
}

func decisionsFromResponses(responses map[string]Response, findings types.Findings, defaults map[string]string, now int64) ([]Decision, []string, map[string]string, error) {
	expected := map[string]types.Finding{}
	for _, item := range findings.Items {
		if item.ID == "" {
			return nil, nil, nil, fmt.Errorf("latest finding has no id")
		}
		expected[item.ID] = item
	}
	if len(responses) != len(expected) {
		return nil, nil, nil, fmt.Errorf("response block count = %d, want %d", len(responses), len(expected))
	}
	var decisions []Decision
	var fixIDs []string
	instructions := map[string]string{}
	for _, item := range findings.Items {
		response, ok := responses[item.ID]
		if !ok {
			return nil, nil, nil, fmt.Errorf("missing response block for finding %s", item.ID)
		}
		solution := strings.TrimSpace(response.Solution)
		solutionSource := SolutionSourceNone
		if response.Action == "fix" {
			if isCommentOnly(solution) {
				var ok bool
				solution, ok = defaultRecommendation(defaults, item)
				if !ok {
					return nil, nil, nil, fmt.Errorf("fix response for finding %s has no solution or machine-detectable option 1", item.ID)
				}
				solutionSource = SolutionSourceDefaultRecommendation
			} else {
				solutionSource = SolutionSourceUser
			}
			if solution == "" {
				return nil, nil, nil, fmt.Errorf("fix response for finding %s has no solution or default recommendation", item.ID)
			}
			fixIDs = append(fixIDs, item.ID)
			instructions[item.ID] = solution
		}
		decisions = append(decisions, Decision{
			FindingID:      item.ID,
			Action:         response.Action,
			Solution:       solution,
			SolutionSource: solutionSource,
			DecisionSource: DecisionSourceFile,
			ProcessedAt:    now,
		})
	}
	for id := range responses {
		if _, ok := expected[id]; !ok {
			return nil, nil, nil, fmt.Errorf("unknown response block for finding %s", id)
		}
	}
	if len(instructions) == 0 {
		instructions = nil
	}
	return decisions, fixIDs, instructions, nil
}

func isCommentOnly(solution string) bool {
	trimmed := strings.TrimSpace(solution)
	return trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//")
}

func defaultRecommendation(defaults map[string]string, item types.Finding) (string, bool) {
	if defaults != nil {
		if recommendation := strings.TrimSpace(defaults[item.ID]); recommendation != "" {
			return recommendation, true
		}
	}
	return defaultRecommendationFromText(item.SuggestedFix)
}

func defaultRecommendationFromText(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimLeft(line, "-* "))
		lower := strings.ToLower(trimmed)
		for _, prefix := range []string{"option 1:", "option 1 -", "1.", "1)"} {
			if strings.HasPrefix(lower, prefix) {
				recommendation := strings.TrimSpace(trimmed[len(prefix):])
				return recommendation, recommendation != ""
			}
		}
	}
	return "", false
}

func stampProcessed(content []byte, action string, at int64, decisions []Decision) []byte {
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "processed_action:") {
			lines[i] = "processed_action: " + action
		}
		if strings.HasPrefix(line, "processed_at:") {
			lines[i] = fmt.Sprintf("processed_at: %d", at)
		}
	}
	body := strings.Join(lines, "\n")
	if idx := strings.Index(body, "\n## Resolved Decision Summary\n"); idx >= 0 {
		body = body[:idx]
	}
	var b bytes.Buffer
	b.WriteString(strings.TrimRight(body, "\n"))
	b.WriteString("\n\n")
	writeDecisionSummary(&b, decisions)
	return b.Bytes()
}

func WritePending(root, relativePath string, content []byte) error {
	path, err := SafeJoinForWrite(root, relativePath)
	if err != nil {
		return err
	}
	return writeFileAtomic(path, content)
}

func WriteProcessed(root string, state State) (State, error) {
	plan, err := PrepareProcessedWrite(root, state)
	if err != nil {
		return State{}, err
	}
	if err := CommitProcessedWrite(plan); err != nil {
		return State{}, err
	}
	return plan.State, nil
}

func PrepareProcessedWrite(root string, state State) (*ProcessedFilePlan, error) {
	if state.ProcessedAction == ProcessedPending || state.ProcessedAt == nil {
		return nil, fmt.Errorf("processed state is required")
	}
	path, err := SafeJoin(root, state.RelativePath)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read review handoff file %s: %w", state.RelativePath, err)
	}
	updatedContent := stampProcessed(content, state.ProcessedAction, *state.ProcessedAt, state.Decisions)
	state.GeneratedContentDigest = ContentDigest(updatedContent)
	state.UpdatedAt = *state.ProcessedAt
	return &ProcessedFilePlan{
		State:            state,
		Path:             path,
		OriginalContent:  append([]byte(nil), content...),
		ProcessedContent: updatedContent,
	}, nil
}

func CommitProcessedWrite(plan *ProcessedFilePlan) error {
	if plan == nil {
		return fmt.Errorf("processed file plan is required")
	}
	if err := writeFileAtomic(plan.Path, plan.ProcessedContent); err != nil {
		return fmt.Errorf("stamp review handoff file: %w", err)
	}
	return nil
}

func RestoreProcessedWrite(plan *ProcessedFilePlan) error {
	if plan == nil {
		return fmt.Errorf("processed file plan is required")
	}
	if err := writeFileAtomic(plan.Path, plan.OriginalContent); err != nil {
		return fmt.Errorf("restore review handoff file: %w", err)
	}
	return nil
}

func writeFileAtomic(path string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".review-handoff-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_, writeErr := tmp.Write(content)
	closeErr := tmp.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}
