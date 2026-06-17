package reviewreport

import (
	"strings"
	"testing"
)

func TestRenderMarkdownContractHeadingsAndCounts(t *testing.T) {
	snapshot := ReportSnapshot{
		Metadata: ReportMetadata{
			Path:             "/tmp/report.md",
			Status:           StatusCurrent,
			GenerationMode:   GenerationModeLive,
			GeneratedAt:      1700000000,
			UpdatedAt:        1700000001,
			SourceSnapshotAt: 1700000001,
		},
		Run: RunContext{
			ID:                "run1",
			Branch:            "feature",
			BaseSHA:           "base",
			HeadSHA:           "head",
			RunStatus:         "completed",
			ReviewStatus:      "completed",
			SafeIntentSummary: "add report",
			PRURL:             ValueUnavailable,
		},
		Counts: ZeroSummaryCounts(),
		Latest: LatestReviewOutcome{
			Outcome:   LatestOutcomeNoIssuesRemain,
			Evidence:  "round r3",
			Risk:      "low",
			Rationale: "clean",
		},
		SourceEvidence: SourceEvidence{
			ReviewStepResultID: "step1",
			IncludedRounds:     []string{"r1"},
			LatestReviewRound:  "r1",
			SourceSnapshotAt:   1700000001,
			IntegrityStatus:    IntegrityConsistent,
		},
	}
	out := RenderMarkdown(snapshot)
	headings := []string{
		"# Review Resolution Report",
		"## Report Metadata",
		"## Purpose",
		"## Run Context",
		"## Summary Counts",
		"## Latest Review Outcome",
		"## Review Findings",
		"## Fix Attempts",
		"## Remaining Risks",
		"## Source Evidence",
		"## Generation Notes",
	}
	last := -1
	for _, heading := range headings {
		idx := strings.Index(out, heading)
		if idx < 0 {
			t.Fatalf("missing heading %q in:\n%s", heading, out)
		}
		if idx <= last {
			t.Fatalf("heading %q is out of order", heading)
		}
		last = idx
	}
	for _, key := range SummaryCountKeys {
		if !strings.Contains(out, "`"+key+"`") {
			t.Fatalf("missing summary count %q", key)
		}
	}
	if strings.Contains(out, "`resolved`") {
		t.Fatal("rendered forbidden aggregate resolved count")
	}
}
