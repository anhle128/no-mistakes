package reviewreport

import (
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestDeriveOneFixRun(t *testing.T) {
	run := &db.Run{ID: "run1", Branch: "feature", HeadSHA: "head", BaseSHA: "base", Status: types.RunCompleted}
	step := &db.StepResult{ID: "step1", RunID: run.ID, StepName: types.StepReview, Status: types.StepStatusCompleted}
	initial := `{"findings":[{"id":"review-1","severity":"warning","file":"a.go","line":10,"description":"nil check missing","context":"handler may receive nil","suggested_fix":"add guard","action":"auto-fix"},{"id":"review-2","severity":"info","description":"note","action":"no-op"}],"summary":"2 findings","risk_level":"medium","risk_rationale":"review found one issue"}`
	selected := `["review-1"]`
	fixSummary := "add nil guard"
	final := `{"findings":[],"summary":"clean","risk_level":"low","risk_rationale":"no findings remain"}`
	rounds := []*db.StepRound{
		{ID: "round-1", StepResultID: step.ID, Round: 1, Trigger: "initial", FindingsJSON: &initial, SelectedFindingIDs: &selected, SelectionSource: strPtr(db.RoundSelectionSourceUser), DurationMS: 1, CreatedAt: 1},
		{ID: "round-2", StepResultID: step.ID, Round: 2, Trigger: "auto_fix", FindingsJSON: &final, FixSummary: &fixSummary, DurationMS: 1, CreatedAt: 2},
	}

	snapshot := Derive(DeriveInput{
		Run:              run,
		ReviewStep:       step,
		Rounds:           rounds,
		ReportPath:       "/tmp/report.md",
		GenerationMode:   GenerationModeLive,
		GeneratedAt:      1700000000,
		UpdatedAt:        1700000001,
		SourceSnapshotAt: 1700000001,
	})

	if snapshot.Latest.Outcome != LatestOutcomeNoIssuesRemain {
		t.Fatalf("latest outcome = %q", snapshot.Latest.Outcome)
	}
	assertCount(t, snapshot.Counts, CountTotalFindings, 2)
	assertCount(t, snapshot.Counts, CountActionableFindings, 1)
	assertCount(t, snapshot.Counts, CountSelectedForFix, 1)
	assertCount(t, snapshot.Counts, CountInformational, 1)
	assertCount(t, snapshot.Counts, CountFixAttempts, 1)
	assertCount(t, snapshot.Counts, CountAppliedFixSummaries, 1)
	if len(snapshot.FixAttempts) != 1 || snapshot.FixAttempts[0].AppliedFix != fixSummary {
		t.Fatalf("fix attempts = %+v", snapshot.FixAttempts)
	}
	out := RenderMarkdown(snapshot)
	if !strings.Contains(out, "`Applied fix`: add nil guard") {
		t.Fatalf("rendered report missing applied fix:\n%s", out)
	}
	if strings.Contains(out, "Suggested fix") {
		t.Fatalf("rendered internal suggested_fix label:\n%s", out)
	}
}

func TestDeriveUnreadableFinalFindingsFailsClosed(t *testing.T) {
	run := &db.Run{ID: "run1", Branch: "feature", HeadSHA: "head", BaseSHA: "base", Status: types.RunCompleted}
	step := &db.StepResult{ID: "step1", RunID: run.ID, StepName: types.StepReview, Status: types.StepStatusCompleted}
	invalid := `{not-json`
	rounds := []*db.StepRound{
		{ID: "round-1", StepResultID: step.ID, Round: 1, Trigger: "initial", FindingsJSON: &invalid, DurationMS: 1, CreatedAt: 1},
	}

	snapshot := Derive(DeriveInput{Run: run, ReviewStep: step, Rounds: rounds})

	if snapshot.Latest.Outcome != LatestOutcomeFinalFindingsUnreadable {
		t.Fatalf("latest outcome = %q", snapshot.Latest.Outcome)
	}
	if snapshot.SourceEvidence.IntegrityStatus != IntegrityPartial {
		t.Fatalf("integrity = %q", snapshot.SourceEvidence.IntegrityStatus)
	}
}

func TestDeriveLegacyMissingSelectionDoesNotInferDecision(t *testing.T) {
	run := &db.Run{ID: "run1", Branch: "feature", HeadSHA: "head", BaseSHA: "base", Status: types.RunCompleted}
	step := &db.StepResult{ID: "step1", RunID: run.ID, StepName: types.StepReview, Status: types.StepStatusCompleted}
	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"legacy finding","action":"auto-fix"}],"summary":"1"}`
	final := `{"findings":[],"summary":"clean"}`
	fixSummary := "fix legacy finding"
	rounds := []*db.StepRound{
		{ID: "round-1", StepResultID: step.ID, Round: 1, Trigger: "initial", FindingsJSON: &initial, DurationMS: 1, CreatedAt: 1},
		{ID: "round-2", StepResultID: step.ID, Round: 2, Trigger: "auto_fix", FindingsJSON: &final, FixSummary: &fixSummary, DurationMS: 1, CreatedAt: 2},
	}

	snapshot := Derive(DeriveInput{Run: run, ReviewStep: step, Rounds: rounds})

	if len(snapshot.Findings) != 1 {
		t.Fatalf("findings = %+v", snapshot.Findings)
	}
	if snapshot.Findings[0].Decision != DecisionNotRecorded {
		t.Fatalf("decision = %q, want decision not recorded", snapshot.Findings[0].Decision)
	}
	assertCount(t, snapshot.Counts, CountSelectedForFix, 0)
	assertCount(t, snapshot.Counts, CountDecisionNotRecorded, 1)
	if snapshot.Counts[CountAccepted] != 0 {
		t.Fatalf("accepted count = %d, want 0", snapshot.Counts[CountAccepted])
	}
}

func TestDeriveSelectedFindingStillOpenKeepsSelectionCount(t *testing.T) {
	run := &db.Run{ID: "run1", Branch: "feature", HeadSHA: "head", BaseSHA: "base", Status: types.RunCompleted}
	step := &db.StepResult{ID: "step1", RunID: run.ID, StepName: types.StepReview, Status: types.StepStatusCompleted}
	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"still broken","action":"auto-fix"}],"summary":"1"}`
	selected := `["review-1"]`
	fixSummary := "try to fix review finding"
	final := `{"findings":[{"id":"review-1","severity":"warning","description":"still broken","action":"auto-fix"}],"summary":"1 remains"}`
	rounds := []*db.StepRound{
		{ID: "round-1", StepResultID: step.ID, Round: 1, Trigger: "initial", FindingsJSON: &initial, SelectedFindingIDs: &selected, SelectionSource: strPtr(db.RoundSelectionSourceAutoFix), DurationMS: 1, CreatedAt: 1},
		{ID: "round-2", StepResultID: step.ID, Round: 2, Trigger: "auto_fix", FindingsJSON: &final, FixSummary: &fixSummary, DurationMS: 1, CreatedAt: 2},
	}

	snapshot := Derive(DeriveInput{Run: run, ReviewStep: step, Rounds: rounds})

	if len(snapshot.Findings) != 1 {
		t.Fatalf("findings = %+v", snapshot.Findings)
	}
	finding := snapshot.Findings[0]
	if !finding.SelectedForFix {
		t.Fatalf("selected_for_fix = false, want true: %+v", finding)
	}
	if finding.Decision != DecisionStillOpen {
		t.Fatalf("decision = %q, want still open", finding.Decision)
	}
	assertCount(t, snapshot.Counts, CountSelectedForFix, 1)
	assertCount(t, snapshot.Counts, CountStillOpen, 1)
}

func TestDeriveFailedAfterCleanPostFixReviewKeepsLatestOutcome(t *testing.T) {
	run := &db.Run{ID: "run1", Branch: "feature", HeadSHA: "head", BaseSHA: "base", Status: types.RunFailed}
	step := &db.StepResult{ID: "step1", RunID: run.ID, StepName: types.StepReview, Status: types.StepStatusCompleted}
	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"issue","action":"auto-fix"}],"summary":"1"}`
	selected := `["review-1"]`
	fixSummary := "fix review finding"
	final := `{"findings":[],"summary":"clean","risk_level":"low","risk_rationale":"review clean"}`
	rounds := []*db.StepRound{
		{ID: "round-1", StepResultID: step.ID, Round: 1, Trigger: "initial", FindingsJSON: &initial, SelectedFindingIDs: &selected, SelectionSource: strPtr(db.RoundSelectionSourceAutoFix), DurationMS: 1, CreatedAt: 1},
		{ID: "round-2", StepResultID: step.ID, Round: 2, Trigger: "auto_fix", FindingsJSON: &final, FixSummary: &fixSummary, DurationMS: 1, CreatedAt: 2},
	}

	snapshot := Derive(DeriveInput{Run: run, ReviewStep: step, Rounds: rounds})

	if snapshot.Latest.Outcome != LatestOutcomeNoIssuesRemain {
		t.Fatalf("latest outcome = %q, want no issues remain", snapshot.Latest.Outcome)
	}
}

func TestDeriveFailedAfterFixWithoutFollowUpReviewIsIncomplete(t *testing.T) {
	run := &db.Run{ID: "run1", Branch: "feature", HeadSHA: "head", BaseSHA: "base", Status: types.RunFailed}
	step := &db.StepResult{ID: "step1", RunID: run.ID, StepName: types.StepReview, Status: types.StepStatusCompleted}
	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"issue","action":"auto-fix"}],"summary":"1"}`
	selected := `["review-1"]`
	fixSummary := "fix review finding"
	rounds := []*db.StepRound{
		{ID: "round-1", StepResultID: step.ID, Round: 1, Trigger: "initial", FindingsJSON: &initial, SelectedFindingIDs: &selected, SelectionSource: strPtr(db.RoundSelectionSourceAutoFix), DurationMS: 1, CreatedAt: 1},
		{ID: "round-2", StepResultID: step.ID, Round: 2, Trigger: "auto_fix", FixSummary: &fixSummary, DurationMS: 1, CreatedAt: 2},
	}

	snapshot := Derive(DeriveInput{Run: run, ReviewStep: step, Rounds: rounds})

	if snapshot.Latest.Outcome != LatestOutcomeReviewResolutionIncomplete {
		t.Fatalf("latest outcome = %q, want review resolution incomplete", snapshot.Latest.Outcome)
	}
}

func assertCount(t *testing.T, counts map[string]int, key string, want int) {
	t.Helper()
	if got := counts[key]; got != want {
		t.Fatalf("count %s = %d, want %d", key, got, want)
	}
}

func strPtr(value string) *string {
	return &value
}
