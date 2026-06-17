package reviewreport

import "testing"

func TestCanonicalReportConstants(t *testing.T) {
	if ContractVersion != "review-resolution-report/v1" {
		t.Fatalf("contract version = %q", ContractVersion)
	}
	assertContains(t, AllowedStatuses, StatusCurrent)
	assertContains(t, AllowedStatuses, StatusStale)
	assertContains(t, AllowedStatuses, StatusUnavailable)
	assertContains(t, AllowedStatuses, StatusError)
	assertContains(t, AllowedLatestOutcomes, LatestOutcomeNoIssuesRemain)
	assertContains(t, AllowedLatestOutcomes, LatestOutcomeReviewResolutionIncomplete)
}

func TestZeroSummaryCountsIncludesEveryCanonicalKey(t *testing.T) {
	counts := ZeroSummaryCounts()
	if len(counts) != len(SummaryCountKeys) {
		t.Fatalf("summary counts has %d keys, want %d", len(counts), len(SummaryCountKeys))
	}
	for _, key := range SummaryCountKeys {
		got, ok := counts[key]
		if !ok {
			t.Fatalf("missing summary count key %q", key)
		}
		if got != 0 {
			t.Fatalf("summary count %q = %d, want 0", key, got)
		}
	}
}

func assertContains(t *testing.T, values []string, want string) {
	t.Helper()
	for _, got := range values {
		if got == want {
			return
		}
	}
	t.Fatalf("%q not found in %v", want, values)
}
