package steps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/agent"
	"github.com/kunchenguid/no-mistakes/internal/config"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestCommitAgentFixesWithEvidenceNoChanges(t *testing.T) {
	repoDir, baseSHA, headSHA := setupGitRepo(t)
	sctx := newTestContextWithDBRecords(t, &mockAgent{name: "test"}, repoDir, baseSHA, headSHA, config.Commands{})

	evidence, err := commitAgentFixesWithEvidence(sctx, types.StepReview, "fix review", "fallback")
	if err != nil {
		t.Fatalf("commitAgentFixesWithEvidence() error = %v", err)
	}
	if evidence.CommitSHA != "" {
		t.Fatalf("CommitSHA = %q, want empty", evidence.CommitSHA)
	}
	if evidence.NoCommitReason != "no_changes" {
		t.Fatalf("NoCommitReason = %q, want no_changes", evidence.NoCommitReason)
	}
}

func TestCommitAgentFixesWithEvidenceCommitFailed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell pre-commit hook is Unix-specific")
	}
	repoDir, baseSHA, headSHA := setupGitRepo(t)
	sctx := newTestContextWithDBRecords(t, &mockAgent{name: "test"}, repoDir, baseSHA, headSHA, config.Commands{})
	hook := filepath.Join(repoDir, ".git", "hooks", "pre-commit")
	if err := os.WriteFile(hook, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write pre-commit hook: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "feature.txt"), []byte("feature code\nchanged\n"), 0o644); err != nil {
		t.Fatalf("write feature change: %v", err)
	}

	evidence, err := commitAgentFixesWithEvidence(sctx, types.StepReview, "fix review", "fallback")
	if err == nil {
		t.Fatal("expected commit failure")
	}
	if evidence.CommitSHA != "" {
		t.Fatalf("CommitSHA = %q, want empty on failed commit", evidence.CommitSHA)
	}
	if evidence.NoCommitReason != "commit_failed" {
		t.Fatalf("NoCommitReason = %q, want commit_failed", evidence.NoCommitReason)
	}
}

func TestExtractFixResolutionDetailsHandlesMissingAndInvalidEvidence(t *testing.T) {
	if got := extractFixResolutionDetails(nil, "summary"); got != "" {
		t.Fatalf("nil result details = %q, want empty", got)
	}
	if got := extractFixResolutionDetails(&agent.Result{}, "summary"); got != "" {
		t.Fatalf("nil output details = %q, want empty", got)
	}

	result := &agent.Result{Output: json.RawMessage(`{"summary":"fix","resolutions":[` +
		`{"finding_id":"review-1","applied_solution":"fixed one","why_this_solution":"safe","changed_files":["a.go"]},` +
		`{"finding_id":"review-1","applied_solution":"duplicate","why_this_solution":"bad","changed_files":["b.go"]},` +
		`{"finding_id":"review-2","applied_solution":"","why_this_solution":"missing","changed_files":["c.go"]}` +
		`]}`)}
	got := extractFixResolutionDetails(result, "normalized summary")
	if got == "" {
		t.Fatal("expected degraded structured details")
	}
	if !strings.Contains(got, `"summary":"normalized summary"`) ||
		!strings.Contains(got, `"finding_id":"review-1"`) ||
		!strings.Contains(got, "duplicate resolution id: review-1") ||
		!strings.Contains(got, "invalid resolution entry") {
		t.Fatalf("details missing expected evidence/degraded markers: %s", got)
	}
	if strings.Contains(got, `"finding_id":"review-2"`) {
		t.Fatalf("invalid resolution should be omitted: %s", got)
	}
}
