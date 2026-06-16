package reviewhandoff

import (
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func DeriveDecision(responses []ResponseBlock, findings []FindingEntry, processedAt time.Time) ProcessedDecision {
	byID := make(map[string]FindingEntry, len(findings))
	for _, finding := range findings {
		byID[finding.ID] = finding
	}
	decision := ProcessedDecision{
		Source:         "file",
		ExecutedAction: ProcessedApprove,
		Actions:        make(map[string]string, len(responses)),
		ProcessedAt:    processedAt,
	}
	for _, response := range responses {
		decision.Actions[response.ID] = response.Action
		if response.Action != ActionFix {
			continue
		}
		decision.ExecutedAction = ProcessedFix
		decision.SelectedFindingIDs = append(decision.SelectedFindingIDs, response.ID)
		instruction := CleanSolution(response.Solution)
		if instruction == "" {
			instruction = defaultRecommendation(byID[response.ID])
		}
		if instruction != "" {
			if decision.Instructions == nil {
				decision.Instructions = make(map[string]string)
			}
			decision.Instructions[response.ID] = instruction
		}
	}
	return decision
}

func CleanSolution(solution string) string {
	lines := strings.Split(solution, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func defaultRecommendation(entry FindingEntry) string {
	if len(entry.Recommendations) == 0 {
		return ""
	}
	return strings.TrimSpace(entry.Recommendations[0])
}

func ProcessedActionForDecision(decision ProcessedDecision) string {
	switch decision.ExecutedAction {
	case ProcessedFix:
		return ProcessedFix
	case ProcessedSkip:
		return ProcessedSkip
	default:
		return ProcessedApprove
	}
}

func AutomationResponses(action types.ApprovalAction, entries []FindingEntry, selectedIDs []string, instructions map[string]string) []ResponseBlock {
	selected := make(map[string]bool, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = true
	}
	responses := make([]ResponseBlock, 0, len(entries))
	for _, entry := range entries {
		response := ResponseBlock{ID: entry.ID}
		switch action {
		case types.ActionSkip:
			response.Action = ActionSkip
		case types.ActionFix:
			if selected[entry.ID] {
				response.Action = ActionFix
				response.Solution = instructions[entry.ID]
			} else {
				response.Action = ActionAccept
			}
		default:
			response.Action = ActionAccept
		}
		responses = append(responses, response)
	}
	return responses
}

func AutomationDecision(action types.ApprovalAction, selectedIDs []string, instructions map[string]string, added []types.Finding, processedAt time.Time) ProcessedDecision {
	decision := ProcessedDecision{
		Source:             "automation",
		ProcessedAt:        processedAt,
		Actions:            map[string]string{},
		SelectedFindingIDs: append([]string(nil), selectedIDs...),
		Instructions:       instructions,
		AddedFindings:      append([]types.Finding(nil), added...),
	}
	switch action {
	case types.ActionFix:
		decision.ExecutedAction = ProcessedFix
	case types.ActionSkip:
		decision.ExecutedAction = ProcessedSkip
	default:
		decision.ExecutedAction = ProcessedApprove
	}
	return decision
}
