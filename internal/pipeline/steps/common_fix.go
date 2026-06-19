package steps

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/agent"
	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

type fixExecutionOptions struct {
	RequirePreviousFindings bool
	MissingFindingsError    string
	LogMessage              string
	Prompt                  string
	ErrorPrefix             string
	FallbackSummary         string
	AfterAgentRun           func(*agent.Result) error
}

type commitSummary struct {
	Summary     string                `json:"summary"`
	Resolutions []fixResolutionDetail `json:"resolutions,omitempty"`
}

type fixResolutionDetail struct {
	FindingID       string   `json:"finding_id"`
	AppliedSolution string   `json:"applied_solution"`
	WhyThisSolution string   `json:"why_this_solution"`
	ChangedFiles    []string `json:"changed_files"`
}

type fixResolutionPayload struct {
	Summary     string                `json:"summary,omitempty"`
	Resolutions []fixResolutionDetail `json:"resolutions,omitempty"`
	Degraded    []string              `json:"degraded,omitempty"`
}

var commitSummarySchema = json.RawMessage(`{
	"type": "object",
	"properties": {
		"summary": {"type": "string"},
		"resolutions": {
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"finding_id": {"type": "string"},
					"applied_solution": {"type": "string"},
					"why_this_solution": {"type": "string"},
					"changed_files": {"type": "array", "items": {"type": "string"}}
				},
				"required": ["finding_id", "applied_solution", "why_this_solution", "changed_files"]
			}
		}
	},
	"required": ["summary"]
}`)

// hasBlockingFindings returns true if any finding has error or warning severity.
func hasBlockingFindings(items []Finding) bool {
	for _, f := range items {
		if f.Severity == "error" || f.Severity == "warning" {
			return true
		}
	}
	return false
}

type fixCommitEvidence struct {
	CommitSHA      string
	NoCommitReason string
}

func commitAgentFixes(sctx *pipeline.StepContext, stepName types.StepName, summary, fallbackSummary string) error {
	_, err := commitAgentFixesWithEvidence(sctx, stepName, summary, fallbackSummary)
	return err
}

func commitAgentFixesWithEvidence(sctx *pipeline.StepContext, stepName types.StepName, summary, fallbackSummary string) (fixCommitEvidence, error) {
	ctx := sctx.Ctx
	status, _ := git.Run(ctx, sctx.WorkDir, "status", "--porcelain")
	if strings.TrimSpace(status) == "" {
		sctx.Log("no agent changes to commit")
		return fixCommitEvidence{NoCommitReason: "no_changes"}, nil
	}
	if _, err := git.Run(ctx, sctx.WorkDir, "add", "-A"); err != nil {
		return fixCommitEvidence{NoCommitReason: "stage_failed"}, fmt.Errorf("stage %s changes: %w", stepName, err)
	}
	if summary == "" {
		summary = fallbackSummary
	}
	commitMessage := deterministicFixCommitMessage(stepName, summary)
	if _, err := git.Run(ctx, sctx.WorkDir, "commit", "-m", commitMessage); err != nil {
		return fixCommitEvidence{NoCommitReason: "commit_failed"}, fmt.Errorf("commit %s changes: %w", stepName, err)
	}
	headSHA, err := git.HeadSHA(ctx, sctx.WorkDir)
	if err != nil {
		return fixCommitEvidence{NoCommitReason: "head_resolution_failed"}, fmt.Errorf("resolve head after %s commit: %w", stepName, err)
	}
	ref := normalizedBranchRef(sctx.Run.Branch)
	if _, err := git.Run(ctx, sctx.WorkDir, "update-ref", ref, headSHA); err != nil {
		return fixCommitEvidence{CommitSHA: headSHA, NoCommitReason: "update_ref_failed"}, fmt.Errorf("update local branch ref: %w", err)
	}
	sctx.Run.HeadSHA = headSHA
	if err := sctx.DB.UpdateRunHeadSHA(sctx.Run.ID, headSHA); err != nil {
		return fixCommitEvidence{CommitSHA: headSHA, NoCommitReason: "run_head_update_failed"}, err
	}
	sctx.Log(fmt.Sprintf("committed agent fixes: %s", commitMessage))
	return fixCommitEvidence{CommitSHA: headSHA}, nil
}

func extractCommitSummary(result *agent.Result) (string, error) {
	var summary commitSummary
	if result.Output == nil {
		return "", fmt.Errorf("agent returned no structured summary")
	}
	if err := json.Unmarshal(result.Output, &summary); err != nil {
		return "", fmt.Errorf("parse commit summary: %w", err)
	}
	cleaned := strings.Join(strings.Fields(summary.Summary), " ")
	cleaned = strings.Trim(cleaned, " \t\r\n\"'.;:,-")
	return cleaned, nil
}

func extractFixResolutionDetails(result *agent.Result, summary string) string {
	if result == nil || result.Output == nil {
		return ""
	}
	var payload commitSummary
	if err := json.Unmarshal(result.Output, &payload); err != nil || len(payload.Resolutions) == 0 {
		return ""
	}
	seen := map[string]bool{}
	out := fixResolutionPayload{Summary: summary}
	for _, detail := range payload.Resolutions {
		id := strings.TrimSpace(detail.FindingID)
		if id == "" || strings.TrimSpace(detail.AppliedSolution) == "" || strings.TrimSpace(detail.WhyThisSolution) == "" || len(detail.ChangedFiles) == 0 {
			out.Degraded = append(out.Degraded, "invalid resolution entry")
			continue
		}
		if seen[id] {
			out.Degraded = append(out.Degraded, "duplicate resolution id: "+id)
			continue
		}
		seen[id] = true
		detail.FindingID = id
		out.Resolutions = append(out.Resolutions, detail)
	}
	if len(out.Resolutions) == 0 && len(out.Degraded) == 0 {
		return ""
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(raw)
}

func deterministicFixCommitMessage(stepName types.StepName, summary string) string {
	if summary == "" {
		summary = "apply fixes"
	}
	return fmt.Sprintf("no-mistakes(%s): %s", stepName, summary)
}

// executeFixMode runs the fix agent and commits any resulting changes. It
// returns the agent's one-line fix summary (empty when the agent returned
// nothing parseable), which the caller should place on StepOutcome.FixSummary
// so the executor can persist it on the round record.
func executeFixMode(sctx *pipeline.StepContext, stepName types.StepName, opts fixExecutionOptions) (string, error) {
	if !sctx.Fixing {
		return "", nil
	}
	if opts.RequirePreviousFindings && sctx.PreviousFindings == "" {
		return "", errors.New(opts.MissingFindingsError)
	}
	if opts.LogMessage != "" {
		sctx.Log(opts.LogMessage)
	}
	sctx.LastFixCommitSHA = ""
	sctx.LastNoCommitReason = ""
	sctx.LastFixResolutionDetailsJSON = ""
	result, err := sctx.Agent.Run(sctx.Ctx, agent.RunOpts{
		Prompt:     opts.Prompt,
		CWD:        sctx.WorkDir,
		JSONSchema: commitSummarySchema,
		OnChunk:    sctx.LogChunk,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", opts.ErrorPrefix, err)
	}
	if opts.AfterAgentRun != nil {
		if err := opts.AfterAgentRun(result); err != nil {
			return "", err
		}
	}
	summary, err := extractCommitSummary(result)
	if err != nil {
		sctx.Log(fmt.Sprintf("warning: could not parse fix summary: %v", err))
	}
	sctx.LastFixResolutionDetailsJSON = extractFixResolutionDetails(result, summary)
	evidence, err := commitAgentFixesWithEvidence(sctx, stepName, summary, opts.FallbackSummary)
	sctx.LastFixCommitSHA = evidence.CommitSHA
	sctx.LastNoCommitReason = evidence.NoCommitReason
	if err != nil {
		return "", err
	}
	return summary, nil
}
