package reviewhandoff

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

type RenderInput struct {
	RunID          string
	Branch         string
	Status         types.StepStatus
	RelativePath   string
	CycleID        string
	Findings       types.Findings
	FixSummaries   []string
	PriorDecisions []Decision
	Now            int64
}

func FindingDigest(findings types.Findings) string {
	raw, _ := json.Marshal(findings)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func ContentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func RenderPending(input RenderInput) ([]byte, State, error) {
	if input.RunID == "" {
		return nil, State{}, fmt.Errorf("run id is required")
	}
	if input.RelativePath == "" {
		return nil, State{}, fmt.Errorf("review file path is required")
	}
	if input.CycleID == "" {
		return nil, State{}, fmt.Errorf("cycle id is required")
	}
	findingDigest := FindingDigest(input.Findings)
	var b bytes.Buffer
	writeFrontMatter(&b, Metadata{
		Marker:          "v1",
		RunID:           input.RunID,
		Step:            string(types.StepReview),
		Status:          string(input.Status),
		Branch:          input.Branch,
		ReviewCycleID:   input.CycleID,
		FindingDigest:   findingDigest,
		ReviewFile:      input.RelativePath,
		ProcessedAction: ProcessedPending,
		ProcessedAt:     "",
	})
	b.WriteString("# Review Handoff\n\n")
	if phase := types.ReviewPhaseLabel(types.StepReview, input.Status); phase != "" {
		fmt.Fprintf(&b, "Phase: %s\n\n", phase)
	}
	if len(input.FixSummaries) > 0 {
		fmt.Fprintf(&b, "Applied fix: %s\n\n", strings.Join(strings.Fields(input.FixSummaries[len(input.FixSummaries)-1]), " "))
	}
	if len(input.Findings.Items) == 0 {
		b.WriteString("No remaining review findings.\n\n")
		writeDecisionSummary(&b, input.PriorDecisions)
	} else {
		for _, item := range input.Findings.Items {
			if _, err := writeFinding(&b, item); err != nil {
				return nil, State{}, err
			}
		}
	}
	content := b.Bytes()
	state := NewState(input.RelativePath, input.CycleID, findingDigest, ContentDigest(content), input.Now)
	state.DefaultRecommendations = defaultRecommendations(input.Findings)
	state.Decisions = append(state.Decisions, input.PriorDecisions...)
	return content, state, nil
}

func writeFrontMatter(b *bytes.Buffer, metadata Metadata) {
	b.WriteString("---\n")
	fmt.Fprintf(b, "no_mistakes_review_handoff: %s\n", metadata.Marker)
	fmt.Fprintf(b, "run_id: %s\n", metadata.RunID)
	fmt.Fprintf(b, "step: %s\n", metadata.Step)
	fmt.Fprintf(b, "status: %s\n", metadata.Status)
	fmt.Fprintf(b, "branch: %s\n", metadata.Branch)
	fmt.Fprintf(b, "review_cycle_id: %s\n", metadata.ReviewCycleID)
	fmt.Fprintf(b, "finding_digest: %s\n", metadata.FindingDigest)
	fmt.Fprintf(b, "review_file: %s\n", metadata.ReviewFile)
	fmt.Fprintf(b, "processed_action: %s\n", metadata.ProcessedAction)
	fmt.Fprintf(b, "processed_at: %s\n", metadata.ProcessedAt)
	b.WriteString("---\n\n")
}

func writeFinding(b *bytes.Buffer, item types.Finding) (string, error) {
	title := strings.TrimSpace(item.Description)
	if title == "" {
		title = item.ID
	}
	fmt.Fprintf(b, "## %s\n\n", item.ID)
	b.WriteString("### Issue\n\n")
	b.WriteString(title + "\n\n")
	if item.File != "" {
		ref := item.File
		if item.Line > 0 {
			ref = fmt.Sprintf("%s:%d", item.File, item.Line)
		}
		fmt.Fprintf(b, "Location: `%s`\n\n", ref)
	}
	if item.Context != "" {
		b.WriteString("### Context\n\n")
		b.WriteString(strings.TrimSpace(item.Context) + "\n\n")
	}
	b.WriteString("### Recommendation\n\n")
	recommendation, defaultRecommendation, err := recommendationText(item)
	if err != nil {
		return "", err
	}
	b.WriteString(recommendation + "\n\n")
	b.WriteString("### User Answer\n\n")
	fmt.Fprintf(b, "```no-mistakes-review-response %s\n", item.ID)
	fmt.Fprintf(b, "action: %s\n", defaultResponseAction(item))
	b.WriteString("solution: \n")
	b.WriteString("```\n\n")
	return defaultRecommendation, nil
}

func writeDecisionSummary(b *bytes.Buffer, decisions []Decision) {
	if len(decisions) == 0 {
		return
	}
	b.WriteString("## Resolved Decision Summary\n\n")
	for _, d := range decisions {
		fmt.Fprintf(b, "- `%s`: %s", d.FindingID, d.Action)
		if d.Solution != "" {
			fmt.Fprintf(b, " - Solution: %s", d.Solution)
		}
		if d.DecisionSource != "" {
			fmt.Fprintf(b, " (%s)", d.DecisionSource)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func defaultResponseAction(item types.Finding) string {
	switch item.Action {
	case types.ActionAskUser:
		return "accept"
	case types.ActionNoOp:
		return "skip"
	default:
		return "fix"
	}
}

func defaultRecommendations(findings types.Findings) map[string]string {
	defaults := map[string]string{}
	for _, item := range findings.Items {
		_, recommendation, err := recommendationText(item)
		if err == nil && recommendation != "" {
			defaults[item.ID] = recommendation
		}
	}
	if len(defaults) == 0 {
		return nil
	}
	return defaults
}

func recommendationText(item types.Finding) (string, string, error) {
	text := strings.TrimSpace(item.SuggestedFix)
	if defaultResponseAction(item) != "fix" {
		if text == "" {
			text = "Option 1: Accept this finding without remediation."
		}
		return text, "", nil
	}
	if recommendation, ok := defaultRecommendationFromText(text); ok {
		return text, recommendation, nil
	}
	if text == "" {
		text = "Address this finding directly."
	}
	recommendation := strings.Join(strings.Fields(text), " ")
	if recommendation == "" {
		return "", "", fmt.Errorf("finding %s has no default recommendation", item.ID)
	}
	return "Option 1: " + recommendation, recommendation, nil
}
