package reviewreport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func openReportTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "state.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func initReportGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runReportGit(t, dir, "init")
	runReportGit(t, dir, "config", "user.email", "test@test.com")
	runReportGit(t, dir, "config", "user.name", "Test User")
	return dir
}

func runReportGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out))
}

func TestRefreshMixedResolvedAcceptedInformationalReport(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","file":"a.go","line":10,"description":"uses TOKEN=secretvalue","action":"auto-fix"},{"id":"review-2","severity":"error","file":"b.go","line":5,"description":"needs approval","action":"ask-user"},{"id":"review-3","severity":"info","description":"FYI only","action":"no-op"}],"summary":"3 findings"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "fix review finding"
	sha := "abc123"
	details := `{"summary":"fix review finding","resolutions":[{"finding_id":"review-1","applied_solution":"validated input","why_this_solution":"prevents bad state","changed_files":["a.go"]}]}`
	empty := `{"findings":[],"summary":"clean"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &empty, &summary, &sha, nil, &details, 20); err != nil {
		t.Fatalf("insert fix round: %v", err)
	}
	reason := "risk accepted by reviewer"
	if _, err := d.InsertReviewResolutionDecision(db.ReviewResolutionDecision{
		RunID:        run.ID,
		StepResultID: step.ID,
		RoundID:      &r1.ID,
		FindingID:    "review-2",
		Action:       db.ReviewResolutionDecisionApprove,
		ActorSource:  "user",
		Reason:       &reason,
	}); err != nil {
		t.Fatalf("insert decision: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.Status != db.ReviewResolutionStatusFinal {
		t.Fatalf("status = %s, want final", meta.Status)
	}
	if meta.ResolvedCount != 1 || meta.AcceptedCount != 1 || meta.InformationalCount != 1 || meta.StillOpenCount != 0 {
		t.Fatalf("unexpected counts: %+v", meta)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	for _, want := range []string{
		"# Review Resolution Report",
		"## Resolved Issues",
		"### review-1",
		"Applied Solution Source: fix agent structured output",
		"Selection source: user",
		"Fix round ID: 2",
		"Evidence quality: structured",
		"## Accepted Without Fix",
		"### review-2",
		"Decision round ID: " + r1.ID,
		"## Informational / No Action Required",
		"### review-3",
		"TOKEN=\\[REDACTED\\]",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in report:\n%s", want, md)
		}
	}
	accepted := md[strings.Index(md, "### review-2"):]
	if next := strings.Index(accepted, "### review-3"); next >= 0 {
		accepted = accepted[:next]
	}
	if strings.Contains(accepted, "Applied Solution Source: inferred from fix round summary") ||
		strings.Contains(accepted, "Applied solution or attempted solution: fix review finding") {
		t.Fatalf("accepted finding should not inherit unrelated fix evidence:\n%s", accepted)
	}
}

func TestRefreshDegradesPartialStructuredResolutionCoverage(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"first","action":"auto-fix"},{"id":"review-2","severity":"warning","description":"second","action":"auto-fix"}],"summary":"2 findings"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1","review-2"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "fix review findings"
	sha := "abc123"
	details := `{"summary":"fix review findings","resolutions":[{"finding_id":"review-1","applied_solution":"fixed first","why_this_solution":"specific fix","changed_files":["a.go"]},{"finding_id":"unknown","applied_solution":"fixed unknown","why_this_solution":"bad id","changed_files":["z.go"]}]}`
	empty := `{"findings":[],"summary":"clean"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &empty, &summary, &sha, nil, &details, 20); err != nil {
		t.Fatalf("insert fix round: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.Status != db.ReviewResolutionStatusDegraded {
		t.Fatalf("status = %s, want degraded", meta.Status)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	if !strings.Contains(md, "### review-2") ||
		!strings.Contains(md, "commit changed-file diff unavailable") ||
		!strings.Contains(md, "Evidence quality: degraded") {
		t.Fatalf("missing degraded fallback evidence for review-2:\n%s", md)
	}
}

func TestRefreshLegacyFixFallbackUsesCommitChangedFiles(t *testing.T) {
	sourceRepoDir := initReportGitRepo(t)
	if err := os.WriteFile(filepath.Join(sourceRepoDir, "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReportGit(t, sourceRepoDir, "add", ".")
	runReportGit(t, sourceRepoDir, "commit", "-m", "initial")
	if err := os.WriteFile(filepath.Join(sourceRepoDir, "a.go"), []byte("package main\nconst token = \"secret\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRepoDir, "b.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runReportGit(t, sourceRepoDir, "add", ".")
	runReportGit(t, sourceRepoDir, "commit", "-m", "fix review")
	sha := runReportGit(t, sourceRepoDir, "rev-parse", "HEAD")

	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	workingPathWithoutFixCommit := initReportGitRepo(t)
	repo, _ := d.InsertRepo(workingPathWithoutFixCommit, "git@github.com:user/project.git", "main")
	if err := os.MkdirAll(filepath.Dir(p.RepoDir(repo.ID)), 0o755); err != nil {
		t.Fatal(err)
	}
	runReportGit(t, filepath.Dir(p.RepoDir(repo.ID)), "clone", "--bare", sourceRepoDir, p.RepoDir(repo.ID))
	run, _ := d.InsertRun(repo.ID, "feature", sha, "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)
	initial := `{"findings":[{"id":"review-1","severity":"warning","file":"a.go","line":1,"description":"token leak","action":"auto-fix"}],"summary":"1 finding"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceAutoFix); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "changed if err != nil { return err }; token=supersecret\n```diff\ndiff --git a/a.go b/a.go\n+token=supersecret\n```"
	empty := `{"findings":[],"summary":"clean"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &empty, &summary, &sha, nil, nil, 20); err != nil {
		t.Fatalf("insert fix round: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.Status != db.ReviewResolutionStatusFinal {
		t.Fatalf("status = %s, want final", meta.Status)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	for _, want := range []string{
		"Applied Solution Source: inferred from fix round summary and commit changed-file diff because structured resolution details were unavailable",
		"Changed files: a.go, b.go",
		"Fix commit SHA: " + sha,
		"Selection source: auto\\_fix",
		"Evidence quality: round\\_level",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("expected %q in report:\n%s", want, md)
		}
	}
	for _, disallowed := range []string{"diff --git", "+token", "supersecret", "if err != nil", "return err"} {
		if strings.Contains(md, disallowed) {
			t.Fatalf("report leaked %q:\n%s", disallowed, md)
		}
	}
	if strings.Contains(md, "commit changed-file diff unavailable") {
		t.Fatalf("report used working-path fallback instead of gate repo evidence:\n%s", md)
	}
}

func TestRefreshChangedFollowupFindingIDKeepsOriginalStillOpen(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"same issue","action":"auto-fix"}],"summary":"1 finding"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "attempted fix"
	sha := "abc123"
	followup := `{"findings":[{"id":"review-renamed","severity":"warning","description":"same issue","action":"auto-fix"}],"summary":"still found"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &followup, &summary, &sha, nil, nil, 20); err != nil {
		t.Fatalf("insert follow-up round: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.ResolvedCount != 0 || meta.StillOpenCount != 2 {
		t.Fatalf("unexpected counts for changed follow-up ID: %+v", meta)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	original := md[strings.Index(md, "### review-1"):]
	if next := strings.Index(original, "### review-renamed"); next >= 0 {
		original = original[:next]
	}
	if !strings.Contains(original, "Outcome: Still Open") {
		t.Fatalf("original changed-ID finding should stay open:\n%s", original)
	}
}

func TestRefreshPartialFollowupResolvesFixedKnownFinding(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"fixed issue","action":"auto-fix"},{"id":"review-2","severity":"warning","description":"still pending","action":"ask-user"}],"summary":"2 findings"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceUser); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "fixed first issue"
	sha := "abc123"
	followup := `{"findings":[{"id":"review-2","severity":"warning","description":"still pending","action":"ask-user"}],"summary":"1 remaining"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &followup, &summary, &sha, nil, nil, 20); err != nil {
		t.Fatalf("insert follow-up round: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.ResolvedCount != 1 || meta.StillOpenCount != 1 {
		t.Fatalf("unexpected counts for partial follow-up: %+v", meta)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	resolved := md[strings.Index(md, "### review-1"):]
	if next := strings.Index(resolved, "### review-2"); next >= 0 {
		resolved = resolved[:next]
	}
	if !strings.Contains(resolved, "Outcome: Resolved") {
		t.Fatalf("fixed known finding should be resolved:\n%s", resolved)
	}
}

func TestRefreshNoCommitEmptyFixRoundKeepsSelectedFindingOpen(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"selected issue","action":"auto-fix"}],"summary":"1 finding"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	selected := `["review-1"]`
	if err := d.SetStepRoundSelection(r1.ID, &selected, db.RoundSelectionSourceAutoFix); err != nil {
		t.Fatalf("set selected finding ids: %v", err)
	}
	summary := "no changes to commit"
	noCommit := "no_changes"
	empty := `{"findings":[],"summary":"no reviewable changes","risk_level":"low","risk_rationale":"no reviewable changes"}`
	if _, err := d.InsertStepRoundWithEvidence(step.ID, 2, "auto_fix", &empty, &summary, nil, &noCommit, nil, 20); err != nil {
		t.Fatalf("insert no-commit fix round: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.ResolvedCount != 0 || meta.StillOpenCount != 1 || meta.Status != db.ReviewResolutionStatusIncomplete {
		t.Fatalf("metadata = %+v, want no resolved findings and one incomplete still-open finding", meta)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	section := md[strings.Index(md, "### review-1"):]
	for _, want := range []string{
		"Outcome: Still Open",
		"Verification text: verification inconclusive",
	} {
		if !strings.Contains(section, want) {
			t.Fatalf("report missing %q:\n%s", want, section)
		}
	}
	if !strings.Contains(section, "No-commit reason:") || !strings.Contains(section, "changes") {
		t.Fatalf("report missing no-commit provenance:\n%s", section)
	}
}

func TestRefreshSkippedReviewWithAcceptedFindingIsFinal(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)

	initial := `{"findings":[{"id":"review-1","severity":"warning","description":"accepted issue","action":"ask-user"}],"summary":"1 finding"}`
	r1, err := d.InsertStepRound(step.ID, 1, "initial", &initial, nil, 10)
	if err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	reason := "skipped by user"
	if _, err := d.InsertReviewResolutionDecision(db.ReviewResolutionDecision{
		RunID:        run.ID,
		StepResultID: step.ID,
		RoundID:      &r1.ID,
		FindingID:    "review-1",
		Action:       db.ReviewResolutionDecisionSkip,
		ActorSource:  "user",
		Reason:       &reason,
	}); err != nil {
		t.Fatalf("insert skip decision: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusSkipped); err != nil {
		t.Fatalf("skip step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta.Status != db.ReviewResolutionStatusFinal || meta.AcceptedCount != 1 || meta.StillOpenCount != 0 {
		t.Fatalf("metadata = %+v, want final accepted finding", meta)
	}
	raw, err := os.ReadFile(meta.ReportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	md := string(raw)
	for _, want := range []string{
		"Outcome: Accepted Without Fix",
		"Decision action: skip",
		"Report lifecycle state: final",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("report missing %q:\n%s", want, md)
		}
	}
}

func TestRenderMarkdownEnforcesTotalReportCap(t *testing.T) {
	snap := Snapshot{
		RunID:          "run-big",
		Branch:         "feature/big",
		BaseSHA:        "base",
		HeadSHA:        "head",
		ReviewStatus:   string(types.StepStatusCompleted),
		ReportStatus:   db.ReviewResolutionStatusIncomplete,
		ReportPath:     "/tmp/report.md",
		FirstGenerated: 1,
		LastRefreshed:  2,
	}
	const totalEntries = 500
	for i := 0; i < totalEntries; i++ {
		snap.Entries = append(snap.Entries, Entry{
			Finding: types.Finding{
				ID:          fmt.Sprintf("review-%04d", i),
				Severity:    "warning",
				Description: strings.Repeat("large description ", 20),
				Action:      types.ActionAskUser,
			},
			Outcome:           OutcomeStillOpen,
			OutcomeText:       "still open",
			FirstRound:        1,
			SelectionSource:   db.RoundSelectionSourceAutoFix,
			FixRound:          2,
			FixCommitSHA:      "abc123",
			SolutionSource:    "not applicable",
			ChangedFiles:      []string{"not recorded"},
			Verification:      "verification inconclusive",
			VerifierSource:    "report classifier",
			EvidenceReference: "latest Review evidence round 1",
			EvidenceQuality:   "unavailable",
		})
	}

	md := RenderMarkdown(snap)
	if !strings.Contains(md, "## Report Truncated") {
		t.Fatalf("expected detail-budget truncation marker")
	}
	if got := strings.Count(md, "### review-"); got != totalEntries {
		t.Fatalf("rendered entry headings = %d, want %d", got, totalEntries)
	}
	for _, id := range []string{"review-0000", "review-0250", "review-0499"} {
		if !strings.Contains(md, "### "+id) {
			t.Fatalf("report omitted entry %s", id)
		}
	}
	stub := md[strings.Index(md, "### review-0499"):]
	for _, want := range []string{
		"Review round ID: 1",
		"Selection source: auto\\_fix",
		"Fix round ID: 2",
		"Fix commit SHA: abc123",
		"Evidence reference: latest Review evidence round 1",
		"Evidence quality: unavailable",
		"Entry detail: truncated because report detail budget was exceeded; finding retained in counts",
	} {
		if !strings.Contains(stub, want) {
			t.Fatalf("stub omitted %q:\n%s", want, stub)
		}
	}
}

func TestMetadataStatusMarksHashMismatchStale(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-resolution.md")
	if err := os.WriteFile(path, []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := &db.ReviewResolutionReport{
		ReportPath:    path,
		Status:        db.ReviewResolutionStatusFinal,
		ContentHash:   "not-the-current-hash",
		ReportVersion: reportVersion,
	}
	if got := MetadataStatus(report); got != db.ReviewResolutionStatusStale {
		t.Fatalf("MetadataStatus() = %q, want %q", got, db.ReviewResolutionStatusStale)
	}
}

func TestMetadataStatusForRunMarksSourceWatermarkDriftStale(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)
	findings := `{"findings":[{"id":"review-1","severity":"warning","description":"needs review","action":"ask-user"}],"summary":"1 finding"}`
	if _, err := d.InsertStepRound(step.ID, 1, "initial", &findings, nil, 10); err != nil {
		t.Fatalf("insert initial round: %v", err)
	}
	if err := d.UpdateStepStatus(step.ID, types.StepStatusCompleted); err != nil {
		t.Fatalf("complete step: %v", err)
	}
	if err := d.UpdateRunStatus(run.ID, types.RunCompleted); err != nil {
		t.Fatalf("complete run: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if got := MetadataStatusForRun(d, run.ID, meta); got != db.ReviewResolutionStatusIncomplete {
		t.Fatalf("MetadataStatusForRun before drift = %q, want incomplete", got)
	}
	later := `{"findings":[{"id":"review-1","severity":"warning","description":"still open","action":"ask-user"}],"summary":"still found"}`
	if _, err := d.InsertStepRound(step.ID, 2, "initial", &later, nil, 10); err != nil {
		t.Fatalf("insert later round: %v", err)
	}

	if got := MetadataStatusForRun(d, run.ID, meta); got != db.ReviewResolutionStatusStale {
		t.Fatalf("MetadataStatusForRun after drift = %q, want stale", got)
	}
}

func TestRefreshUnparsableReviewEvidenceProducesEvidenceUnavailableReport(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)
	bad := `{"findings":`
	if _, err := d.InsertStepRound(step.ID, 1, "initial", &bad, nil, 10); err != nil {
		t.Fatalf("insert bad round: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta == nil {
		t.Fatal("expected evidence-unavailable report metadata")
	}
	if meta.Status != db.ReviewResolutionStatusEvidenceUnavailable {
		t.Fatalf("status = %s, want evidence_unavailable", meta.Status)
	}
	if meta.EntryCount != 0 {
		t.Fatalf("entry count = %d, want 0", meta.EntryCount)
	}
}

func TestRefreshCleanReviewDeletesMetadata(t *testing.T) {
	d := openReportTestDB(t)
	p := paths.WithRoot(t.TempDir())
	repo, _ := d.InsertRepo("/repo/project", "git@github.com:user/project.git", "main")
	run, _ := d.InsertRun(repo.ID, "feature", "head", "base")
	step, _ := d.InsertStepResult(run.ID, types.StepReview)
	clean := `{"findings":[],"summary":"clean"}`
	if _, err := d.InsertStepRound(step.ID, 1, "initial", &clean, nil, 10); err != nil {
		t.Fatalf("insert clean round: %v", err)
	}

	meta, err := Refresh(d, p, run.ID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if meta != nil {
		t.Fatalf("clean review produced metadata: %+v", meta)
	}
	got, err := d.GetReviewResolutionReport(run.ID)
	if err != nil {
		t.Fatalf("get report: %v", err)
	}
	if got != nil {
		t.Fatalf("expected no report row, got %+v", got)
	}
}
