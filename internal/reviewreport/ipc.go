package reviewreport

import (
	"encoding/json"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
)

// IPCInfoFromMetadata converts persisted report metadata into the compact IPC
// shape used by daemon, CLI, and TUI surfaces.
func IPCInfoFromMetadata(meta *db.ReviewResolutionReportMetadata) *ipc.ReviewResolutionReportInfo {
	if meta == nil {
		return nil
	}
	info := &ipc.ReviewResolutionReportInfo{
		Status:        meta.Status,
		LatestOutcome: meta.LatestOutcome,
		UpdatedAt:     meta.UpdatedAt,
		Stale:         meta.Stale,
	}
	if meta.ReportPath != nil {
		info.Path = *meta.ReportPath
	}
	if meta.GeneratedAt != nil {
		info.GeneratedAt = *meta.GeneratedAt
	}
	if meta.SafeError != nil {
		info.Error = *meta.SafeError
	}
	if meta.SummaryCountsJSON != "" {
		counts := map[string]int{}
		if err := json.Unmarshal([]byte(meta.SummaryCountsJSON), &counts); err == nil {
			info.SummaryCounts = counts
		}
	}
	return info
}
