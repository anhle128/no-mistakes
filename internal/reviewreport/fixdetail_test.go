package reviewreport

import (
	"strings"
	"testing"
)

func TestMatchingFixDetailValidatesRequiredUniqueKnownSelectedIDs(t *testing.T) {
	selected := map[string]bool{"review-1": true, "review-2": true}
	raw := `{"summary":"fixes","resolutions":[` +
		`{"finding_id":"review-1","applied_solution":"fixed one","why_this_solution":"targeted","changed_files":["b.go","a.go"]},` +
		`{"finding_id":"review-1","applied_solution":"duplicate","why_this_solution":"bad","changed_files":["dup.go"]},` +
		`{"finding_id":"unknown","applied_solution":"unknown","why_this_solution":"bad","changed_files":["z.go"]},` +
		`{"finding_id":"","applied_solution":"missing id","why_this_solution":"bad","changed_files":["x.go"]}` +
		`]}`

	detail, degraded := matchingFixDetail(raw, "review-1", selected)
	if !degraded {
		t.Fatal("expected degraded evidence for duplicate, unknown, and invalid entries")
	}
	if detail == nil {
		t.Fatal("expected matching detail for review-1")
	}
	if detail.FindingID != "review-1" || detail.AppliedSolution != "fixed one" || detail.WhyThisSolution != "targeted" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if got := sanitizeChangedFiles(detail.ChangedFiles); len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
		t.Fatalf("changed files = %#v, want sorted a.go,b.go", got)
	}

	missing, degraded := matchingFixDetail(raw, "review-2", selected)
	if !degraded {
		t.Fatal("expected degraded evidence for selected finding without matching detail")
	}
	if missing != nil {
		t.Fatalf("expected no detail for review-2, got %+v", missing)
	}
}

func TestSanitizeChangedFilesRejectsUnsafeDedupesAndBounds(t *testing.T) {
	files := sanitizeChangedFiles([]string{
		"b.go",
		"../secret",
		".",
		"b.go",
		"`unsafe`.go",
		"a.go",
		"nested/" + longRunes("x", maxChangedFileRunes+20) + ".go",
	})
	for _, disallowed := range []string{"../secret", ".", "`unsafe`.go"} {
		for _, got := range files {
			if got == disallowed {
				t.Fatalf("unsafe path %q was retained in %#v", disallowed, files)
			}
		}
	}
	counts := map[string]int{}
	for _, file := range files {
		counts[file]++
	}
	if counts["b.go"] != 1 {
		t.Fatalf("b.go count = %d, want 1 in %#v", counts["b.go"], files)
	}
	for _, want := range []string{"a.go", "b.go", "unsafe.go"} {
		if counts[want] != 1 {
			t.Fatalf("missing sanitized path %q in %#v", want, files)
		}
	}
	foundTruncated := false
	for _, file := range files {
		if strings.Contains(file, "[truncated]") {
			foundTruncated = true
			break
		}
	}
	if !foundTruncated {
		t.Fatalf("expected truncated long path marker in %#v", files)
	}
}

func longRunes(s string, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		out += s
	}
	return out
}
