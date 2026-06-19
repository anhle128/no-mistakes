package reviewreport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestRenderMarkdownMixedResolvedAcceptedGolden(t *testing.T) {
	finalized := int64(1700000120)
	snap := Snapshot{
		RunID:          "run-123",
		RepoIdentifier: "/repo/project",
		Branch:         "feature/review-report",
		BaseSHA:        "base123",
		HeadSHA:        "head456",
		ReviewStatus:   "completed",
		ReportStatus:   "final",
		ReportPath:     "/tmp/nm/reports/run-123/review-resolution.md",
		FirstGenerated: 1700000000,
		LastRefreshed:  1700000060,
		FinalizedAt:    &finalized,
		Entries: []Entry{
			{
				Finding:           types.Finding{ID: "review-1", Severity: "warning", File: "a.go", Line: 10, Description: "fixed issue", Action: types.ActionAutoFix, Source: "agent"},
				Outcome:           OutcomeResolved,
				OutcomeText:       "Comparable follow-up Review round no longer reported this normalized finding ID.",
				FirstRound:        1,
				LastSeenRound:     1,
				SelectionSource:   "user",
				FixRound:          2,
				FixCommitSHA:      "abc123",
				SolutionSource:    "fix agent structured output",
				AppliedSolution:   "Changed a.go to handle nil input.",
				Rationale:         "This preserves behavior and removes the warning.",
				ChangedFiles:      []string{"a.go"},
				Verification:      "finding absent from follow-up Review output",
				FollowupRound:     3,
				ScopeNote:         "same Review step run",
				VerifierSource:    "follow-up review",
				EvidenceReference: "fix round 2 and follow-up Review round 3",
				EvidenceQuality:   "structured",
			},
			{
				Finding:           types.Finding{ID: "review-2", Severity: "error", File: "b.go", Line: 5, Description: "accepted issue", Action: types.ActionAskUser, Source: "agent"},
				Outcome:           OutcomeAccepted,
				OutcomeText:       "Persisted Review terminal decision accepted the finding without a fix.",
				Decision:          &Decision{Action: "approve", ActorSource: "user", CreatedAt: 1700000030, RoundID: "round-1", Reason: "approved tradeoff"},
				FirstRound:        1,
				LastSeenRound:     1,
				SelectionSource:   "not recorded",
				SolutionSource:    "not applicable",
				AppliedSolution:   "not recorded",
				Rationale:         "not recorded",
				ChangedFiles:      []string{"not recorded"},
				Verification:      "accepted without fix by user",
				VerifierSource:    "review terminal decision",
				EvidenceReference: "persisted review resolution decision round-1",
				EvidenceQuality:   "structured",
			},
			{
				Finding:           types.Finding{ID: "review-3", Severity: "info", Description: "FYI only", Action: types.ActionNoOp, Source: "agent"},
				Outcome:           OutcomeInformational,
				OutcomeText:       "Review marked this finding as no action required.",
				FirstRound:        1,
				LastSeenRound:     1,
				SolutionSource:    "not applicable",
				AppliedSolution:   "not recorded",
				Rationale:         "not recorded",
				ChangedFiles:      []string{"not recorded"},
				Verification:      "no action required",
				VerifierSource:    "review finding action",
				EvidenceReference: "Review round 1 finding action",
				EvidenceQuality:   "structured",
			},
			{
				Finding:           types.Finding{ID: "review-4", Severity: "warning", File: "c.go", Line: 8, Description: "still open issue", Action: types.ActionAutoFix, Source: "agent"},
				Outcome:           OutcomeStillOpen,
				OutcomeText:       "No persisted acceptance or comparable resolved evidence was recorded.",
				FirstRound:        2,
				LastSeenRound:     2,
				SolutionSource:    "not applicable",
				AppliedSolution:   "not recorded",
				Rationale:         "not recorded",
				ChangedFiles:      []string{"not recorded"},
				Verification:      "verification inconclusive",
				ScopeNote:         "no comparable parsed follow-up evidence",
				VerifierSource:    "report classifier",
				EvidenceReference: "latest Review evidence round 2",
				EvidenceQuality:   "unavailable",
			},
		},
	}
	assertGolden(t, "mixed_resolved_accepted.golden.md", RenderMarkdown(snap))
}

func TestRenderMarkdownStructuredAndLegacyGolden(t *testing.T) {
	snap := Snapshot{
		RunID:          "run-legacy",
		RepoIdentifier: "/repo/project",
		Branch:         "feature/legacy",
		BaseSHA:        "base999",
		HeadSHA:        "head999",
		ReviewStatus:   "completed",
		ReportStatus:   "final",
		ReportPath:     "/tmp/nm/reports/run-legacy/review-resolution.md",
		FirstGenerated: 1700000000,
		LastRefreshed:  1700000060,
		Entries: []Entry{
			{
				Finding:           types.Finding{ID: "review-1", Severity: "warning", File: "structured.go", Line: 1, Description: "structured issue", Action: types.ActionAutoFix, Source: "agent"},
				Outcome:           OutcomeResolved,
				OutcomeText:       "Comparable follow-up Review round no longer reported this normalized finding ID.",
				FirstRound:        1,
				LastSeenRound:     1,
				SelectionSource:   "auto_fix",
				FixRound:          2,
				FixCommitSHA:      "abc123",
				SolutionSource:    "fix agent structured output",
				AppliedSolution:   "Removed the unsafe branch.",
				Rationale:         "The branch was dead code.",
				ChangedFiles:      []string{"structured.go"},
				Verification:      "finding absent from follow-up Review output",
				FollowupRound:     3,
				ScopeNote:         "same Review step run",
				VerifierSource:    "follow-up review",
				EvidenceReference: "fix round 2 and follow-up Review round 3",
				EvidenceQuality:   "structured",
			},
			{
				Finding:           types.Finding{ID: "review-2", Severity: "warning", File: "legacy.go", Line: 2, Description: "legacy issue", Action: types.ActionAutoFix, Source: "agent"},
				Outcome:           OutcomeResolved,
				OutcomeText:       "Comparable follow-up Review round no longer reported this normalized finding ID.",
				FirstRound:        1,
				LastSeenRound:     1,
				SelectionSource:   "auto_fix",
				FixRound:          2,
				FixCommitSHA:      "def456",
				SolutionSource:    "inferred from fix round summary and commit changed-file diff because structured resolution details were unavailable",
				AppliedSolution:   "Fix round 2 recorded commit def456 touching legacy.go. Legacy summary text was not embedded because structured resolution details were unavailable.",
				Rationale:         "Structured rationale was unavailable; this is round-level evidence derived from persisted fix-round evidence and commit changed-file paths when available.",
				ChangedFiles:      []string{"legacy.go"},
				Verification:      "finding absent from follow-up Review output",
				FollowupRound:     3,
				ScopeNote:         "same Review step run",
				VerifierSource:    "follow-up review",
				EvidenceReference: "fix round 2 and follow-up Review round 3",
				EvidenceQuality:   "round_level",
			},
		},
	}
	assertGolden(t, "structured_resolution.golden.md", RenderMarkdown(snap))
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	if got != string(want) {
		t.Fatalf("rendered markdown mismatch for %s\nwant:\n%s\ngot:\n%s", name, string(want), got)
	}
}
