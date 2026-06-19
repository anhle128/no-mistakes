package steps

import (
	"fmt"
	"strings"

	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// executionContextPromptSection returns a prompt fragment that explains the
// agent's runtime environment: either the historical isolated gate worktree or
// an explicitly requested current-worktree run.
//
// Why this exists: agents that scan their cwd to "verify" the project
// (Claude Code, opencode, etc.) frequently second-guess worktree boundaries.
// Isolated runs can be mistaken for "not the real checkout"; current-worktree
// runs can be mistaken for disposable sandboxes. The fix is not to lie about
// the cwd - it's to spell out what the cwd actually is so the agent can stop
// second-guessing it.
//
// The fragment ends with a trailing newline so callers can append it
// directly to a prompt string without worrying about spacing.
func executionContextPromptSection(sctx *pipeline.StepContext) string {
	if sctx != nil && sctx.Run != nil && types.NormalizeWorktreeMode(sctx.Run.WorktreeMode) == types.WorktreeModeCurrent {
		label := ""
		if sctx.Run.WorkDirLabel != nil {
			label = cleanExecutionContextLine(*sctx.Run.WorkDirLabel)
		}
		if label == "" {
			label = types.WorktreeModeCurrent.Label()
		}
		warning := ""
		if sctx.Run.CurrentWorktreeWarning != nil {
			warning = cleanExecutionContextLine(*sctx.Run.CurrentWorktreeWarning)
		}
		if warning == "" {
			warning = fmt.Sprintf("%s: uses this checkout; pipeline fixes may modify it and commits remain here", label)
		}
		return fmt.Sprintf(`
Execution context:
- You are running inside the current git worktree for this repository, not a disposable no-mistakes worktree.
- This run is in current-worktree mode (`+"`worktree_mode=current`"+`) and the work directory label is %q.
- %s
- The worktree is checked out to the change being processed; treat this current checkout as the project's source of truth for this run and do not search the filesystem for "the real" checkout - this is it.
- Operate only within this working directory. Do not clean up, move, or delete this checkout; automated fixes and commits remain here.
`, label, warning)
	}

	return `
Execution context:
- You are running inside an isolated git worktree at the current working directory.
- The worktree's ` + "`.git`" + ` is a pointer file (not a directory) referencing a bare gate repository elsewhere on disk; this is standard git-worktree layout and all normal git commands work as expected.
- The worktree is checked out to the change being processed; treat it as the project's source of truth for this run and do not search the filesystem for "the real" checkout - this is it.
- Operate only within this working directory. Do not modify or read from the gate's bare repository or any other clone of this project.
`
}

func cleanExecutionContextLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
