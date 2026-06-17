package reviewreport

type CompactSummaryCounts struct {
	SelectedForFix      int
	FixAttempts         int
	AppliedFixSummaries int
	StillOpen           int
	DecisionNotRecorded int
}

func CompactSummaryCountKeys() []string {
	return []string{
		CountSelectedForFix,
		CountFixAttempts,
		CountAppliedFixSummaries,
		CountStillOpen,
		CountDecisionNotRecorded,
	}
}

func CompactSummaryCountsFromMap(counts map[string]int) CompactSummaryCounts {
	return CompactSummaryCounts{
		SelectedForFix:      counts[CountSelectedForFix],
		FixAttempts:         counts[CountFixAttempts],
		AppliedFixSummaries: counts[CountAppliedFixSummaries],
		StillOpen:           counts[CountStillOpen],
		DecisionNotRecorded: counts[CountDecisionNotRecorded],
	}
}
