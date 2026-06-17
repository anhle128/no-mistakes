package pipeline

import (
	"log/slog"

	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/reviewreport"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func (e *Executor) refreshReviewResolutionReport(runID string, stepName types.StepName) {
	if stepName != types.StepReview || e == nil || e.db == nil || e.paths == nil {
		return
	}
	if _, err := reviewreport.Update(e.db, e.paths, runID, reviewreport.GenerationModeLive); err != nil {
		slog.Warn("failed to update review resolution report", "run", runID, "error", err)
	}
}

func (e *Executor) refreshRunReviewResolutionReport(runID string) {
	if e == nil || e.db == nil || e.paths == nil {
		return
	}
	if _, err := reviewreport.Update(e.db, e.paths, runID, reviewreport.GenerationModeLive); err != nil {
		slog.Warn("failed to update review resolution report", "run", runID, "error", err)
	}
}

func (e *Executor) reviewResolutionReportInfo(runID string) *ipc.ReviewResolutionReportInfo {
	if e == nil || e.db == nil {
		return nil
	}
	meta, err := e.db.GetReviewResolutionReportMetadata(runID)
	if err != nil || meta == nil {
		return nil
	}
	return reviewreport.IPCInfoFromMetadata(meta)
}
