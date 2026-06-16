package reviewhandoff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func testEntries() []FindingEntry {
	return []FindingEntry{
		{
			ID:                    "review-1",
			Severity:              "warning",
			File:                  "internal/pipeline/executor.go",
			Line:                  42,
			Issue:                 "missing validation",
			Context:               "The gate accepts stale review state.",
			Recommendations:       []string{"Recompute the current review hash before processing."},
			DefaultResponseAction: ActionFix,
		},
		{
			ID:                    "review-2",
			Severity:              "info",
			Issue:                 "document the path",
			Context:               "Users need to find the handoff file.",
			Recommendations:       []string{"Show the path in compact surfaces."},
			DefaultResponseAction: ActionAccept,
		},
	}
}

func testLiveState(entries []FindingEntry) LiveState {
	state := LiveState{
		RunID:               "01JZEXAMPLEFULL",
		Branch:              "feature/review-file-handoff",
		Step:                StepReview,
		Status:              string(types.StepStatusAwaitingApproval),
		ReviewCycleRevision: "round-1:1",
		Findings:            entries,
	}
	state.ReviewResultHash = ComputeHash(HashInput{
		RunID:               state.RunID,
		Step:                state.Step,
		Status:              state.Status,
		ReviewCycleRevision: state.ReviewCycleRevision,
		Findings:            entries,
	})
	return state
}

func TestResolvePathUsesSingleAnchorAndRejectsUnsafePaths(t *testing.T) {
	checkout := t.TempDir()
	anchorDir := filepath.Join(checkout, "specs", "001")
	if err := os.MkdirAll(anchorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(anchorDir, "plan.md"), []byte("plan"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolvePath(PathResolveInput{
		CheckoutDir:             checkout,
		RunID:                   "01JZEXAMPLEFULL",
		Branch:                  "Feature/Review File",
		UncommittedChangedPaths: []string{"specs/001/plan.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRel := "specs/001/review-issues-01JZEXAM.md"
	if got.RelPath != wantRel || got.Source != "uncommitted_anchor" || got.AnchorRelPath != "specs/001/plan.md" {
		t.Fatalf("result = %+v, want rel %q from anchor", got, wantRel)
	}

	got, err = ResolvePath(PathResolveInput{
		CheckoutDir:             checkout,
		RunID:                   "01JZEXAMPLEFULL",
		Branch:                  "Feature/Review File",
		UncommittedChangedPaths: []string{"a/plan.md", "b/tasks.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.RelPath != ".no-mistakes/issues/feature-review-file/review-issues-01JZEXAM.md" || got.Source != "fallback" {
		t.Fatalf("fallback result = %+v", got)
	}

	if _, err := SafeJoin(checkout, "../plan.md"); err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if _, err := SafeJoin(checkout, ".git/config"); err == nil {
		t.Fatal("expected .git path to be rejected")
	}
	if _, err := ResolvePath(PathResolveInput{
		CheckoutDir:             checkout,
		RunID:                   "01JZEXAMPLEFULL",
		Branch:                  "x",
		UncommittedChangedPaths: []string{"../plan.md"},
	}); err != nil {
		t.Fatalf("unsafe non-anchor changed paths should be ignored, got %v", err)
	}
}

func TestRenderParseValidateAndDeriveDecision(t *testing.T) {
	entries := testEntries()
	live := testLiveState(entries)
	file := HandoffFile{
		Metadata:  NewMetadata(live, ""),
		Summary:   SummaryFor(entries),
		Findings:  entries,
		Responses: InitialResponses(entries),
	}
	data, err := Render(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Issue", "Context", "Recommendation", "User Answer", "```no-mistakes-response"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("rendered file missing %q:\n%s", want, data)
		}
	}

	edited := strings.Replace(string(data), "solution: \"\"", "solution: |\n    # comment\n    Use a fresh hash check.", 1)
	validated, err := ValidateBytes("review.md", []byte(edited), live)
	if err != nil {
		t.Fatal(err)
	}
	if len(validated.Responses) != 2 {
		t.Fatalf("responses = %d, want 2", len(validated.Responses))
	}
	decision := DeriveDecision(validated.Responses, entries, time.Unix(100, 0).UTC())
	if decision.ExecutedAction != ProcessedFix {
		t.Fatalf("decision action = %q", decision.ExecutedAction)
	}
	if got := decision.SelectedFindingIDs; len(got) != 1 || got[0] != "review-1" {
		t.Fatalf("selected ids = %#v", got)
	}
	if got := decision.Instructions["review-1"]; got != "Use a fresh hash check." {
		t.Fatalf("instruction = %q", got)
	}

	for i := range validated.Responses {
		validated.Responses[i].Action = ActionSkip
	}
	decision = DeriveDecision(validated.Responses, entries, time.Unix(101, 0).UTC())
	if decision.ExecutedAction != ProcessedApprove {
		t.Fatalf("all-skip file decision action = %q, want approve", decision.ExecutedAction)
	}
	audit := AuditEntriesFromDecision(decision, entries)
	if audit[0].Action != ActionSkip || audit[1].Action != ActionSkip {
		t.Fatalf("all-skip audit actions = %+v", audit)
	}
}

// TestParseResponseBlocksAllowsFenceInSolution ensures a code fence pasted
// inside a solution block scalar does not prematurely terminate the response
// block. Fences nested in a solution are indented under "solution: |"; only a
// column-0 ``` closes the block.
func TestParseResponseBlocksAllowsFenceInSolution(t *testing.T) {
	fence := "```"
	lines := []string{
		fence + FenceLanguage,
		"id: review-1",
		"action: fix",
		"solution: |",
		"  Replace the parser. Example:",
		"  " + fence + "go",
		"  x := 1",
		"  " + fence,
		"  Done.",
		fence,
		fence + FenceLanguage,
		"id: review-2",
		"action: accept",
		fence,
	}
	data := []byte(strings.Join(lines, "\n") + "\n")

	blocks, err := ParseResponseBlocks(data)
	if err != nil {
		t.Fatalf("ParseResponseBlocks: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks = %d, want 2: %#v", len(blocks), blocks)
	}
	if blocks[0].ID != "review-1" || blocks[0].Action != "fix" {
		t.Fatalf("block 0 = %#v", blocks[0])
	}
	for _, want := range []string{"x := 1", fence + "go", "Done."} {
		if !strings.Contains(blocks[0].Solution, want) {
			t.Fatalf("solution truncated at nested fence, missing %q:\n%s", want, blocks[0].Solution)
		}
	}
	if blocks[1].ID != "review-2" || blocks[1].Action != "accept" {
		t.Fatalf("block 1 = %#v", blocks[1])
	}
}

func TestResolvePathRejectsSymlinkEscapes(t *testing.T) {
	checkout := t.TempDir()
	outside := t.TempDir()

	if err := os.Symlink(outside, filepath.Join(checkout, ".no-mistakes")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolvePath(PathResolveInput{
		CheckoutDir: checkout,
		RunID:       "01JZEXAMPLEFULL",
		Branch:      "feature/review-file",
	}); err == nil || !strings.Contains(err.Error(), "path escapes checkout") {
		t.Fatalf("fallback symlink ancestor error = %v, want escape rejection", err)
	}

	checkout = t.TempDir()
	outsideFile := filepath.Join(outside, "review.md")
	if err := os.WriteFile(outsideFile, []byte("review"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkRel := FileName("01JZEXAMPLEFULL")
	if err := os.Symlink(outsideFile, filepath.Join(checkout, linkRel)); err != nil {
		t.Fatal(err)
	}
	if _, err := SafeJoin(checkout, linkRel); err == nil || !strings.Contains(err.Error(), "path is a symlink") {
		t.Fatalf("final symlink error = %v, want symlink rejection", err)
	}
}

func TestValidateRejectsMalformedOrStaleFile(t *testing.T) {
	entries := testEntries()
	live := testLiveState(entries)
	data, err := Render(HandoffFile{
		Metadata:  NewMetadata(live, ""),
		Summary:   SummaryFor(entries),
		Findings:  entries,
		Responses: InitialResponses(entries),
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		edit func(string) string
		want string
	}{
		{
			name: "stale hash",
			edit: func(s string) string {
				return strings.Replace(s, live.ReviewResultHash, "sha256:stale", 1)
			},
			want: "review_result_hash mismatch",
		},
		{
			name: "processed metadata",
			edit: func(s string) string {
				s = strings.Replace(s, "processed_at: null", "processed_at: \"2026-06-16T00:00:00Z\"", 1)
				return strings.Replace(s, "processed_action: pending", "processed_action: approve", 1)
			},
			want: "processed_at must be null",
		},
		{
			name: "unknown id",
			edit: func(s string) string {
				return strings.Replace(s, "id: review-2", "id: review-404", 1)
			},
			want: "unknown finding id",
		},
		{
			name: "invalid action",
			edit: func(s string) string {
				return strings.Replace(s, "action: accept", "action: maybe", 1)
			},
			want: "invalid action",
		},
		{
			name: "missing block",
			edit: func(s string) string {
				idx := strings.LastIndex(s, "```no-mistakes-response")
				if idx < 0 {
					return s
				}
				return s[:idx]
			},
			want: "missing response block",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateBytes("review.md", []byte(tc.edit(string(data))), live)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}

	oversized := append([]byte{}, data...)
	oversized = append(oversized, make([]byte, MaxFileBytes-int64(len(oversized))+1)...)
	if _, err := ValidateBytes("review.md", oversized, live); err == nil || !strings.Contains(err.Error(), "file exceeds") {
		t.Fatalf("oversized error = %v", err)
	}
}

func TestUpdateProcessedMetadataUsesSnapshot(t *testing.T) {
	entries := testEntries()
	live := testLiveState(entries)
	data, err := Render(HandoffFile{
		Metadata:  NewMetadata(live, ""),
		Summary:   SummaryFor(entries),
		Findings:  entries,
		Responses: InitialResponses(entries),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "review.md")
	if err := WriteFileAtomic(path, data); err != nil {
		t.Fatal(err)
	}
	if err := UpdateProcessedMetadata(path, []byte("stale"), "2026-06-16T00:00:00Z", ProcessedFix); err == nil {
		t.Fatal("expected stale snapshot to fail")
	}
	if err := UpdateProcessedMetadata(path, data, "2026-06-16T00:00:00Z", ProcessedFix); err != nil {
		t.Fatal(err)
	}
	updated, err := ReadBounded(path, MaxFileBytes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "processed_action: fix") {
		t.Fatalf("unexpected updated file:\n%s", updated)
	}
	if !strings.Contains(string(updated), "#### User Answer") {
		t.Fatalf("body was not preserved:\n%s", updated)
	}
}

func TestAutomationResponsesAuditAndPhaseLabels(t *testing.T) {
	entries := testEntries()
	responses := AutomationResponses(types.ActionFix, entries, []string{"review-2"}, map[string]string{"review-2": "document it"})
	if responses[0].Action != ActionAccept || responses[1].Action != ActionFix || responses[1].Solution != "document it" {
		t.Fatalf("responses = %+v", responses)
	}
	responses = AutomationResponses(types.ActionFix, entries, nil, nil)
	if responses[0].Action != ActionAccept || responses[1].Action != ActionAccept {
		t.Fatalf("empty fix selection should accept existing findings, got %+v", responses)
	}

	decision := AutomationDecision(types.ActionFix, []string{"review-2"}, map[string]string{"review-2": "document it"}, []types.Finding{{
		ID:               "user-1",
		Severity:         "warning",
		Description:      "extra issue",
		UserInstructions: "fix extra",
	}}, time.Unix(200, 0).UTC())
	audit := AuditEntriesFromDecision(decision, entries)
	if len(audit) != 3 {
		t.Fatalf("audit entries = %d", len(audit))
	}
	if audit[1].Action != ActionFix || audit[1].Solution != "document it" || audit[2].Selection != "user-added finding sent to fixer" {
		t.Fatalf("audit = %+v", audit)
	}

	live := testLiveState(nil)
	data, err := Render(HandoffFile{
		Metadata:     NewMetadata(live, ""),
		AuditEntries: audit,
		FinalState:   FinalNoFindingsText,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), FinalNoFindingsText) || !strings.Contains(string(data), "Prior Decisions") {
		t.Fatalf("final audit render missing content:\n%s", data)
	}

	if got := PhaseLabel(types.StepReview, types.StepStatusAwaitingApproval); got != "Review preview complete" {
		t.Fatalf("phase label = %q", got)
	}
	if got := PhaseLabel(types.StepTest, types.StepStatusAwaitingApproval); got != "" {
		t.Fatalf("non-review phase label = %q", got)
	}
}

func TestEntriesFromFindingsRequiresStableIDs(t *testing.T) {
	_, err := EntriesFromFindings(types.Findings{Items: []types.Finding{{Severity: "warning", Description: "missing id"}}})
	if err == nil || !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("missing id error = %v", err)
	}
	_, err = EntriesFromFindings(types.Findings{Items: []types.Finding{{ID: "x"}, {ID: "x"}}})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate error = %v", err)
	}
	entries, err := EntriesFromFindings(types.Findings{Items: []types.Finding{{
		ID:           "review-1",
		Severity:     "warning",
		Description:  "bug",
		SuggestedFix: "fix it",
		Action:       types.ActionAutoFix,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].DefaultResponseAction != ActionFix || entries[0].Recommendations[0] != "fix it" {
		t.Fatalf("entry = %+v", entries[0])
	}
}
