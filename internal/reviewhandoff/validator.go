package reviewhandoff

import (
	"fmt"
	"strings"
)

type ValidatedFile struct {
	Metadata  Metadata
	Body      []byte
	Responses []ResponseBlock
	Snapshot  []byte
}

func ValidateBytes(path string, data []byte, live LiveState) (*ValidatedFile, error) {
	if int64(len(data)) > MaxFileBytes {
		return nil, validationFailure(path, fmt.Sprintf("file exceeds %d bytes", MaxFileBytes))
	}
	meta, body, err := ParseFrontMatter(data)
	if err != nil {
		return nil, validationFailure(path, err.Error())
	}
	if err := validateMetadata(meta, live); err != nil {
		return nil, validationFailure(path, err.Error())
	}
	responses, err := ParseResponseBlocks(body)
	if err != nil {
		return nil, validationFailure(path, err.Error())
	}
	if err := validateResponseCoverage(responses, live.Findings); err != nil {
		return nil, validationFailure(path, err.Error())
	}
	return &ValidatedFile{
		Metadata:  meta,
		Body:      body,
		Responses: responses,
		Snapshot:  append([]byte(nil), data...),
	}, nil
}

func validateMetadata(meta Metadata, live LiveState) error {
	if meta.RunID == "" {
		return fmt.Errorf("front matter run_id is required")
	}
	if meta.RunID != live.RunID {
		return fmt.Errorf("run_id mismatch: got %q want %q", meta.RunID, live.RunID)
	}
	if meta.Step != StepReview {
		return fmt.Errorf("step must be review")
	}
	if meta.Status != live.Status {
		return fmt.Errorf("status mismatch: got %q want %q", meta.Status, live.Status)
	}
	if meta.ReviewCycleRevision != live.ReviewCycleRevision {
		return fmt.Errorf("review_cycle_revision mismatch")
	}
	if meta.ReviewResultHash != live.ReviewResultHash {
		return fmt.Errorf("review_result_hash mismatch")
	}
	if meta.ProcessedAt != nil {
		return fmt.Errorf("processed_at must be null before processing")
	}
	if meta.ProcessedAction != ProcessedPending {
		return fmt.Errorf("processed_action must be pending before processing")
	}
	return nil
}

func validateResponseCoverage(responses []ResponseBlock, findings []FindingEntry) error {
	expected := make(map[string]FindingEntry, len(findings))
	for _, finding := range findings {
		if strings.TrimSpace(finding.ID) == "" {
			return fmt.Errorf("latest finding missing id")
		}
		if _, exists := expected[finding.ID]; exists {
			return fmt.Errorf("duplicate latest finding id %q", finding.ID)
		}
		expected[finding.ID] = finding
	}
	seen := make(map[string]bool, len(responses))
	for _, response := range responses {
		id := strings.TrimSpace(response.ID)
		if id == "" {
			return fmt.Errorf("response block missing id")
		}
		if seen[id] {
			return fmt.Errorf("duplicate response block for %q", id)
		}
		seen[id] = true
		if _, ok := expected[id]; !ok {
			return fmt.Errorf("unknown finding id %q", id)
		}
		if !validResponseAction(response.Action) {
			return fmt.Errorf("invalid action %q for %s", response.Action, id)
		}
		if len([]byte(response.Solution)) > MaxSolutionBytes {
			return fmt.Errorf("solution for %s exceeds %d bytes", id, MaxSolutionBytes)
		}
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("missing response block for %q", id)
		}
	}
	return nil
}
