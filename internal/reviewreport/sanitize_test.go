package reviewreport

import "testing"

func TestSanitizeTextRejectsUnsafeSources(t *testing.T) {
	tests := []string{
		"diff --git a/a.go b/a.go\n@@ -1 +1 @@",
		"panic: boom\ngoroutine 1 [running]",
		"api_key=sk-secret",
		"assistant: here is the transcript",
		"func main() {\nreturn\n}",
		"if err != nil { return err }",
		`fmt.Sprintf("x: %s", value)`,
		`config.Path = "/tmp/report.md"`,
		`config.Paths.Report = "/tmp/report.md"`,
		`items[0] = value`,
		`state.cache[key].value = reportPath`,
		`cache[key]`,
	}
	for _, raw := range tests {
		if got := SanitizeText(raw, ValueNotRecorded); got != ValueUnavailable {
			t.Fatalf("SanitizeText(%q) = %q, want unavailable", raw, got)
		}
	}
}

func TestSanitizeAppliedFixSummaryUsesDisplayFallbacks(t *testing.T) {
	if got := SanitizeAppliedFixSummary(""); got != AppliedFixSummaryDisplayMissing {
		t.Fatalf("empty fix summary = %q, want %q", got, AppliedFixSummaryDisplayMissing)
	}
	if got := SanitizeAppliedFixSummary("diff --git a/a.go b/a.go"); got != AppliedFixSummaryDisplayOmitted {
		t.Fatalf("unsafe fix summary = %q, want %q", got, AppliedFixSummaryDisplayOmitted)
	}
	if got := SanitizeAppliedFixSummary("  handle nil pointer  "); got != "handle nil pointer" {
		t.Fatalf("safe fix summary = %q", got)
	}
}

func TestSanitizeTextKeepsPlainUserFacingText(t *testing.T) {
	got := SanitizeText("  Update report metadata after review fixes.  ", ValueUnavailable)
	if got != "Update report metadata after review fixes." {
		t.Fatalf("SanitizeText() = %q", got)
	}

	got = SanitizeText("  Update report metadata after review fixes (no code copied).  ", ValueUnavailable)
	if got != "Update report metadata after review fixes (no code copied)." {
		t.Fatalf("SanitizeText() = %q", got)
	}
}
