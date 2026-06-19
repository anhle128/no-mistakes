package steps

import (
	"strings"
	"testing"

	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

func TestExecutionContextPromptSection_IsolatedMentions(t *testing.T) {
	got := executionContextPromptSection(&pipeline.StepContext{Run: &db.Run{WorktreeMode: types.WorktreeModeIsolated}})
	for _, want := range []string{
		"isolated git worktree",
		"pointer file",
		"do not search the filesystem",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("execution context section missing %q; got:\n%s", want, got)
		}
	}
}

func TestExecutionContextPromptSection_CurrentWorktreeMentions(t *testing.T) {
	label := "uses this checkout"
	warning := "uses this checkout: pipeline fixes may modify it"
	got := executionContextPromptSection(&pipeline.StepContext{Run: &db.Run{
		WorktreeMode:           types.WorktreeModeCurrent,
		WorkDirLabel:           &label,
		CurrentWorktreeWarning: &warning,
	}})
	for _, want := range []string{
		"current git worktree",
		"worktree_mode=current",
		"uses this checkout",
		"do not clean up, move, or delete this checkout",
		"do not search the filesystem",
	} {
		if !strings.Contains(strings.ToLower(got), want) {
			t.Errorf("current execution context section missing %q; got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "isolated git worktree") || strings.Contains(got, "bare gate repository") {
		t.Errorf("current execution context should not use isolated wording; got:\n%s", got)
	}
}

// The section is injected into review/test/lint/document/pr step prompts.
// It must be task-neutral - words like "review" or "lint" leak the wrong
// framing into other steps.
func TestExecutionContextPromptSection_TaskNeutral(t *testing.T) {
	sections := []string{
		executionContextPromptSection(&pipeline.StepContext{Run: &db.Run{WorktreeMode: types.WorktreeModeIsolated}}),
		executionContextPromptSection(&pipeline.StepContext{Run: &db.Run{WorktreeMode: types.WorktreeModeCurrent}}),
	}
	for _, banned := range []string{
		"reviewed",
		"review",
		"linted",
		"lint",
		"tested",
		"test",
		"document",
	} {
		for _, got := range sections {
			if strings.Contains(strings.ToLower(got), banned) {
				t.Errorf("execution context section contains task-specific word %q; should be neutral. Section:\n%s", banned, got)
			}
		}
	}
}

func TestExecutionContextPromptSection_NewlineSafe(t *testing.T) {
	sections := []string{
		executionContextPromptSection(&pipeline.StepContext{Run: &db.Run{WorktreeMode: types.WorktreeModeIsolated}}),
		executionContextPromptSection(&pipeline.StepContext{Run: &db.Run{WorktreeMode: types.WorktreeModeCurrent}}),
	}
	for _, got := range sections {
		if !strings.HasPrefix(got, "\n") {
			t.Error("expected leading newline so callers can append cleanly")
		}
		if !strings.HasSuffix(got, "\n") {
			t.Error("expected trailing newline")
		}
	}
}
