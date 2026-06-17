package reviewreport

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func Update(database *db.DB, p *paths.Paths, runID string, generationMode string) (*db.ReviewResolutionReportMetadata, error) {
	if database == nil {
		return nil, fmt.Errorf("database is required")
	}
	if p == nil {
		return nil, fmt.Errorf("paths are required")
	}
	run, err := database.GetRun(runID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("run not found: %s", runID)
	}
	steps, err := database.GetStepsByRun(runID)
	if err != nil {
		return nil, err
	}
	reviewStep := findReviewStep(steps)
	existing, _ := database.GetReviewResolutionReportMetadata(runID)
	now := time.Now().Unix()
	reportPath := p.RunReviewResolutionReportPath(runID)
	generatedAt := now
	if existing != nil && existing.GeneratedAt != nil {
		generatedAt = *existing.GeneratedAt
	}

	if reviewStep == nil {
		meta := unavailableMetadata(run, generationMode, now, "review step unavailable")
		if err := database.UpsertReviewResolutionReportMetadata(meta); err != nil {
			return nil, err
		}
		return &meta, nil
	}

	rounds, err := database.GetRoundsByStep(reviewStep.ID)
	if err != nil {
		meta := errorMetadata(run, existing, generationMode, now, "review rounds unavailable: "+err.Error())
		if dbErr := database.UpsertReviewResolutionReportMetadata(meta); dbErr != nil {
			return nil, dbErr
		}
		return &meta, err
	}
	if !hasReportableReviewEvidence(rounds) {
		meta := unavailableMetadata(run, generationMode, now, "review evidence unavailable")
		if err := database.UpsertReviewResolutionReportMetadata(meta); err != nil {
			return nil, err
		}
		return &meta, nil
	}

	snapshot := Derive(DeriveInput{
		Run:              run,
		ReviewStep:       reviewStep,
		Rounds:           rounds,
		ReportPath:       reportPath,
		GenerationMode:   generationMode,
		GeneratedAt:      generatedAt,
		UpdatedAt:        now,
		SourceSnapshotAt: now,
	})
	body := RenderMarkdown(snapshot)
	if err := os.MkdirAll(p.RunReportDir(runID), 0o755); err != nil {
		meta := metadataFromSnapshot(snapshot, existing, StatusError, true, "create report directory: "+err.Error())
		if dbErr := database.UpsertReviewResolutionReportMetadata(meta); dbErr != nil {
			return nil, dbErr
		}
		return &meta, err
	}
	if err := os.WriteFile(reportPath, []byte(body), 0o644); err != nil {
		meta := metadataFromSnapshot(snapshot, existing, StatusError, true, "write report: "+err.Error())
		if dbErr := database.UpsertReviewResolutionReportMetadata(meta); dbErr != nil {
			return nil, dbErr
		}
		return &meta, err
	}

	meta := metadataFromSnapshot(snapshot, existing, StatusCurrent, false, "")
	if err := database.UpsertReviewResolutionReportMetadata(meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func findReviewStep(steps []*db.StepResult) *db.StepResult {
	for _, step := range steps {
		if step.StepName == types.StepReview {
			return step
		}
	}
	return nil
}

func hasReportableReviewEvidence(rounds []*db.StepRound) bool {
	for _, round := range rounds {
		if round == nil {
			continue
		}
		return true
	}
	return false
}

func metadataFromSnapshot(snapshot ReportSnapshot, existing *db.ReviewResolutionReportMetadata, status string, stale bool, safeErr string) db.ReviewResolutionReportMetadata {
	counts, err := json.Marshal(snapshot.Counts)
	if err != nil {
		counts = []byte("{}")
	}
	roundIDs, err := json.Marshal(snapshot.SourceEvidence.IncludedRounds)
	if err != nil {
		roundIDs = []byte("[]")
	}
	path := snapshot.Metadata.Path
	effectiveStatus := status
	if status == StatusError && existing != nil && existing.ReportPath != nil && *existing.ReportPath != "" {
		path = *existing.ReportPath
		stale = true
		effectiveStatus = StatusStale
	} else if status == StatusError {
		path = ""
	}
	var pathPtr *string
	if strings.TrimSpace(path) != "" {
		pathPtr = &path
	}
	generatedAt := snapshot.Metadata.GeneratedAt
	if status == StatusError && effectiveStatus != StatusStale {
		generatedAt = 0
	}
	var generatedAtPtr *int64
	if generatedAt > 0 {
		generatedAtPtr = &generatedAt
	}
	meta := db.ReviewResolutionReportMetadata{
		RunID:              snapshot.Run.ID,
		ReportPath:         pathPtr,
		Status:             effectiveStatus,
		ContractVersion:    ContractVersion,
		LatestOutcome:      snapshot.Latest.Outcome,
		SummaryCountsJSON:  string(counts),
		GenerationMode:     snapshot.Metadata.GenerationMode,
		SourceSnapshotAt:   snapshot.Metadata.SourceSnapshotAt,
		SourceRoundIDsJSON: string(roundIDs),
		GeneratedAt:        generatedAtPtr,
		UpdatedAt:          snapshot.Metadata.UpdatedAt,
		Stale:              stale,
	}
	if snapshot.SourceEvidence.ReviewStepResultID != "" {
		value := snapshot.SourceEvidence.ReviewStepResultID
		meta.SourceStepResultID = &value
	}
	if snapshot.SourceEvidence.LatestReviewRound != "" {
		value := snapshot.SourceEvidence.LatestReviewRound
		meta.LatestReviewRoundID = &value
	}
	if snapshot.SourceEvidence.LatestFixRound != "" {
		value := snapshot.SourceEvidence.LatestFixRound
		meta.LatestFixRoundID = &value
	}
	if strings.TrimSpace(safeErr) != "" {
		clean := SanitizeText(safeErr, ValueUnavailable)
		meta.SafeError = &clean
	}
	if effectiveStatus == StatusError && meta.ReportPath == nil {
		meta.Status = StatusUnavailable
	}
	if effectiveStatus == StatusCurrent {
		meta.Stale = false
		meta.SafeError = nil
	}
	return meta
}

func unavailableMetadata(run *db.Run, generationMode string, now int64, safeErr string) db.ReviewResolutionReportMetadata {
	counts, _ := json.Marshal(ZeroSummaryCounts())
	cleanErr := SanitizeText(safeErr, ValueUnavailable)
	return db.ReviewResolutionReportMetadata{
		RunID:              run.ID,
		Status:             StatusUnavailable,
		ContractVersion:    ContractVersion,
		LatestOutcome:      LatestOutcomeFinalFindingsUnavailable,
		SummaryCountsJSON:  string(counts),
		GenerationMode:     generationModeOrDefault(generationMode),
		SourceSnapshotAt:   now,
		SourceRoundIDsJSON: "[]",
		UpdatedAt:          now,
		SafeError:          &cleanErr,
	}
}

func errorMetadata(run *db.Run, existing *db.ReviewResolutionReportMetadata, generationMode string, now int64, safeErr string) db.ReviewResolutionReportMetadata {
	meta := unavailableMetadata(run, generationMode, now, safeErr)
	meta.Status = StatusError
	if existing != nil && existing.ReportPath != nil && *existing.ReportPath != "" {
		meta.Status = StatusStale
		meta.ReportPath = existing.ReportPath
		meta.Stale = true
		meta.GeneratedAt = existing.GeneratedAt
	}
	return meta
}

func generationModeOrDefault(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return GenerationModeLive
	}
	return mode
}
