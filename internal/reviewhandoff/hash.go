package reviewhandoff

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type HashInput struct {
	RunID               string
	Step                string
	Status              string
	ReviewCycleRevision string
	Findings            []FindingEntry
	FixSummaries        []string
}

func ComputeHash(input HashInput) string {
	h := sha256.New()
	writeHashField(h, "run_id", input.RunID)
	writeHashField(h, "step", input.Step)
	writeHashField(h, "status", input.Status)
	writeHashField(h, "review_cycle_revision", input.ReviewCycleRevision)
	for i, finding := range input.Findings {
		prefix := fmt.Sprintf("finding.%06d.", i)
		writeHashField(h, prefix+"id", finding.ID)
		writeHashField(h, prefix+"severity", finding.Severity)
		writeHashField(h, prefix+"issue", finding.Issue)
		writeHashField(h, prefix+"context", finding.Context)
		writeHashField(h, prefix+"default_response_action", finding.DefaultResponseAction)
		for j, rec := range finding.Recommendations {
			writeHashField(h, fmt.Sprintf("%srecommendation.%06d", prefix, j), rec)
		}
	}
	for i, summary := range input.FixSummaries {
		writeHashField(h, fmt.Sprintf("fix_summary.%06d", i), summary)
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

type hashWriter interface {
	Write([]byte) (int, error)
}

func writeHashField(h hashWriter, key, value string) {
	fmt.Fprintf(h, "%s=%d:%s\n", key, len([]byte(value)), value)
}
