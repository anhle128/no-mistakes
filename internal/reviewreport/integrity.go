package reviewreport

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// MetadataStatus returns the report status consumers should surface. It keeps
// persisted non-confident states and marks the metadata stale when the local
// Markdown artifact is missing or no longer matches the stored hash.
func MetadataStatus(report *db.ReviewResolutionReport) string {
	if report == nil {
		return ""
	}
	switch report.Status {
	case db.ReviewResolutionStatusEvidenceUnavailable, db.ReviewResolutionStatusStale, db.ReviewResolutionStatusDegraded:
		return report.Status
	}
	if report.ReportPath == "" || report.ContentHash == "" {
		return db.ReviewResolutionStatusEvidenceUnavailable
	}
	data, err := os.ReadFile(report.ReportPath)
	if err != nil {
		return db.ReviewResolutionStatusStale
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != report.ContentHash {
		return db.ReviewResolutionStatusStale
	}
	return report.Status
}

// MetadataStatusForRun validates the local artifact and the source evidence
// watermark before returning a confident status. If Review rounds or decisions
// changed after the last refresh, consumers surface stale metadata instead of
// trusting old counts.
func MetadataStatusForRun(database *db.DB, runID string, report *db.ReviewResolutionReport) string {
	status := MetadataStatus(report)
	if report == nil || database == nil || status == "" {
		return status
	}
	switch status {
	case db.ReviewResolutionStatusEvidenceUnavailable, db.ReviewResolutionStatusStale, db.ReviewResolutionStatusDegraded:
		return status
	}
	current, err := CurrentSourceWatermark(database, runID)
	if err != nil || current == "" || report.SourceWatermark == "" {
		return db.ReviewResolutionStatusEvidenceUnavailable
	}
	if current != report.SourceWatermark {
		return db.ReviewResolutionStatusStale
	}
	return status
}

// CurrentSourceWatermark returns the Review evidence watermark for a run's
// current persisted rounds and resolution decisions.
func CurrentSourceWatermark(database *db.DB, runID string) (string, error) {
	if database == nil || runID == "" {
		return "", fmt.Errorf("missing database or run ID")
	}
	steps, err := database.GetStepsByRun(runID)
	if err != nil {
		return "", err
	}
	var reviewStep *db.StepResult
	for _, step := range steps {
		if step.StepName == types.StepReview {
			reviewStep = step
			break
		}
	}
	if reviewStep == nil {
		return "", nil
	}
	rounds, err := database.GetRoundsByStep(reviewStep.ID)
	if err != nil {
		return "", err
	}
	decisions, err := database.GetReviewResolutionDecisions(runID)
	if err != nil {
		return "", err
	}
	return sourceWatermark(rounds, decisions), nil
}
