package daemon

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kunchenguid/no-mistakes/internal/agent"
	"github.com/kunchenguid/no-mistakes/internal/config"
	"github.com/kunchenguid/no-mistakes/internal/db"
	"github.com/kunchenguid/no-mistakes/internal/git"
	"github.com/kunchenguid/no-mistakes/internal/ipc"
	"github.com/kunchenguid/no-mistakes/internal/paths"
	"github.com/kunchenguid/no-mistakes/internal/pipeline"
	"github.com/kunchenguid/no-mistakes/internal/pipeline/steps"
	"github.com/kunchenguid/no-mistakes/internal/telemetry"
	"github.com/kunchenguid/no-mistakes/internal/types"
)

// StepFactory creates pipeline steps for a run. Defaults to steps.AllSteps.
type StepFactory func() []pipeline.Step

// RunManager tracks active pipeline executors and manages run lifecycle.
type RunManager struct {
	mu           sync.Mutex
	executors    map[string]*pipeline.Executor      // runID → executor
	cancels      map[string]context.CancelCauseFunc // runID → cancel function with cause
	dones        map[string]chan struct{}           // runID → closed when goroutine exits
	wg           sync.WaitGroup                     // tracks background run goroutines
	shuttingDown atomic.Bool                        // prevents new runs during shutdown
	db           *db.DB
	paths        *paths.Paths
	steps        StepFactory

	branchLocks sync.Map // repoID+"/"+branch → *sync.Mutex

	subMu          sync.RWMutex
	subscribers    map[string][]chan<- ipc.Event // runID → subscriber channels
	completedRuns  map[string]bool               // runIDs whose goroutines have finished
	completedOrder []string                      // insertion order for FIFO eviction
}

// NewRunManager creates a RunManager. Pass nil for stepFactory to use default steps.
func NewRunManager(database *db.DB, p *paths.Paths, stepFactory StepFactory) *RunManager {
	if stepFactory == nil {
		stepFactory = func() []pipeline.Step { return steps.AllSteps() }
	}
	return &RunManager{
		executors:     make(map[string]*pipeline.Executor),
		cancels:       make(map[string]context.CancelCauseFunc),
		dones:         make(map[string]chan struct{}),
		db:            database,
		paths:         p,
		steps:         stepFactory,
		subscribers:   make(map[string][]chan<- ipc.Event),
		completedRuns: make(map[string]bool),
	}
}

// Subscribe registers a channel to receive events for a run.
// Returns the channel and an unsubscribe function.
// If the run has already completed, the returned channel is immediately closed.
func (m *RunManager) Subscribe(runID string) (<-chan ipc.Event, func()) {
	ch := make(chan ipc.Event, 64)
	m.subMu.Lock()
	if m.completedRuns[runID] {
		m.subMu.Unlock()
		close(ch)
		return ch, func() {}
	}
	m.subscribers[runID] = append(m.subscribers[runID], ch)
	m.subMu.Unlock()

	unsub := func() {
		m.subMu.Lock()
		defer m.subMu.Unlock()
		subs := m.subscribers[runID]
		for i, s := range subs {
			if s == ch {
				m.subscribers[runID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
	return ch, unsub
}

// broadcast sends an event to all subscribers of the event's run.
func (m *RunManager) broadcast(event ipc.Event) {
	m.subMu.RLock()
	defer m.subMu.RUnlock()
	for _, ch := range m.subscribers[event.RunID] {
		select {
		case ch <- event:
		default:
			slog.Debug("dropped event for slow subscriber", "run_id", event.RunID, "type", event.Type)
		}
	}
}

// closeSubscribers closes all subscriber channels for a run and marks it
// as completed so future Subscribe calls return an immediately-closed channel.
func (m *RunManager) closeSubscribers(runID string) {
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for _, ch := range m.subscribers[runID] {
		close(ch)
	}
	delete(m.subscribers, runID)
	m.completedRuns[runID] = true
	m.completedOrder = append(m.completedOrder, runID)
	if len(m.completedOrder) > 1000 {
		half := len(m.completedOrder) / 2
		for _, id := range m.completedOrder[:half] {
			delete(m.completedRuns, id)
		}
		m.completedOrder = m.completedOrder[half:]
	}
}

// repoIDFromGatePath extracts the repo ID from a gate bare repo path.
// Gate paths look like: <root>/repos/<id>.git
func repoIDFromGatePath(gatePath string) (string, error) {
	base := filepath.Base(gatePath)
	if !strings.HasSuffix(base, ".git") {
		return "", fmt.Errorf("invalid gate path: %s", gatePath)
	}
	return strings.TrimSuffix(base, ".git"), nil
}

// branchFromRef extracts the branch name from a full git ref.
// "refs/heads/main" → "main", "main" → "main"
func branchFromRef(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

// HandlePushReceived processes a push notification from the post-receive hook.
// It creates a run, sets up a worktree, and launches pipeline execution in the background.
func (m *RunManager) HandlePushReceived(ctx context.Context, params *ipc.PushReceivedParams) (string, error) {
	// Ref deletion (git push remote :branch) sends new SHA as all-zeros.
	// Nothing to validate - skip pipeline.
	if git.IsZeroSHA(params.New) {
		return "", fmt.Errorf("ref deletion push, no pipeline to run")
	}

	repoID, err := repoIDFromGatePath(params.Gate)
	if err != nil {
		return "", err
	}

	repo, err := m.db.GetRepo(repoID)
	if err != nil {
		return "", fmt.Errorf("get repo: %w", err)
	}
	if repo == nil {
		return "", fmt.Errorf("unknown repo for gate %s", params.Gate)
	}

	branch := branchFromRef(params.Ref)
	return m.startRun(ctx, repo, branch, params.New, params.Old, "push", params.SkipSteps, params.Intent)
}

// HandleRerun creates a new run for the latest gate head on a branch. An
// optional intent is stamped onto the new run.
func (m *RunManager) HandleRerun(ctx context.Context, repoID, branch string, skipSteps []types.StepName, intent string) (string, error) {
	repo, err := m.db.GetRepo(repoID)
	if err != nil {
		return "", fmt.Errorf("get repo: %w", err)
	}
	if repo == nil {
		return "", fmt.Errorf("unknown repo %s", repoID)
	}

	gateDir := m.paths.RepoDir(repo.ID)
	headSHA, err := git.Run(ctx, gateDir, "rev-parse", "refs/heads/"+branch+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve gate head: %w", err)
	}

	runs, err := m.db.GetRunsByRepo(repoID)
	if err != nil {
		return "", fmt.Errorf("get runs: %w", err)
	}

	var latestForBranch *db.Run
	var matchingHead *db.Run
	for _, run := range runs {
		if run.Branch != branch {
			continue
		}
		if latestForBranch == nil {
			latestForBranch = run
		}
		if run.HeadSHA == headSHA {
			matchingHead = run
			break
		}
	}
	if latestForBranch == nil {
		return "", fmt.Errorf("no previous run for branch %s", branch)
	}

	baseSHA := latestForBranch.BaseSHA
	if matchingHead != nil {
		baseSHA = matchingHead.BaseSHA
	}

	return m.startRun(ctx, repo, branch, headSHA, baseSHA, "rerun", skipSteps, intent)
}

// ActiveRunConflictError reports that a new direct start cannot safely reuse or
// replace an existing active run.
type ActiveRunConflictError struct {
	RunID            string             `json:"run_id"`
	WorktreeMode     types.WorktreeMode `json:"worktree_mode"`
	Branch           string             `json:"branch"`
	ShortHead        string             `json:"short_head"`
	WorkDirLabel     string             `json:"work_dir_label,omitempty"`
	Status           types.RunStatus    `json:"status"`
	ResumeCommand    string             `json:"resume_command,omitempty"`
	AbortCommand     string             `json:"abort_command,omitempty"`
	RequestedMode    types.WorktreeMode `json:"requested_worktree_mode"`
	RequestedHead    string             `json:"requested_short_head"`
	RequestedWorkDir string             `json:"requested_work_dir_label,omitempty"`
}

func (e *ActiveRunConflictError) Error() string {
	return fmt.Sprintf("active run %s on %s is incompatible with this request", e.RunID, e.Branch)
}

func (e *ActiveRunConflictError) RPCErrorCode() int { return ipc.ErrInvalidParams }
func (e *ActiveRunConflictError) RPCErrorData() any { return e }

type CurrentWorktreeStartError struct {
	Message      string             `json:"message"`
	Reason       string             `json:"reason"`
	Recovery     string             `json:"recovery,omitempty"`
	WorktreeMode types.WorktreeMode `json:"worktree_mode"`
}

func (e *CurrentWorktreeStartError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Reason
}

func (e *CurrentWorktreeStartError) RPCErrorCode() int { return ipc.ErrInvalidParams }
func (e *CurrentWorktreeStartError) RPCErrorData() any { return e }

// HandleStartRun creates or resumes a compatible direct-start run.
func (m *RunManager) HandleStartRun(ctx context.Context, params *ipc.StartRunParams) (string, bool, error) {
	if params == nil {
		return "", false, currentStartError("missing start_run params", types.RejectionActiveRunConflict, "")
	}
	repo, err := m.db.GetRepo(params.RepoID)
	if err != nil {
		return "", false, fmt.Errorf("get repo: %w", err)
	}
	if repo == nil {
		return "", false, fmt.Errorf("unknown repo %s", params.RepoID)
	}
	mode := types.NormalizeWorktreeMode(params.WorktreeMode)
	if !mode.Valid() {
		return "", false, fmt.Errorf("invalid worktree mode %q", params.WorktreeMode)
	}
	startParams := *params
	startParams.WorktreeMode = mode
	if mode == types.WorktreeModeCurrent {
		runID, resumed, err := m.tryResumeCurrentRun(ctx, repo, &startParams)
		if err != nil || resumed {
			return runID, resumed, err
		}
		if startParams.RequireIntent && strings.TrimSpace(startParams.Intent) == "" {
			return "", false, currentStartError("--intent is required to start a run", types.RejectionMissingIntent, `Pass what the user set out to accomplish: no-mistakes axi run --intent "the user's goal"`)
		}
		if err := prepareCurrentStartParams(ctx, repo, &startParams); err != nil {
			return "", false, err
		}
	}
	return m.startRunWithOptions(ctx, repo, startParams.Branch, startParams.HeadSHA, startParams.BaseSHA, startRunOptions{
		trigger:                     "direct",
		skipSteps:                   startParams.SkipSteps,
		intent:                      startParams.Intent,
		worktreeMode:                mode,
		workDir:                     startParams.WorkDir,
		workDirLabel:                startParams.WorkDirLabel,
		currentWorktreeWarning:      startParams.CurrentWorktreeWarning,
		reviewBaseRef:               startParams.ReviewBaseRef,
		reviewBaseRefreshAttempted:  startParams.ReviewBaseRefreshAttempted,
		reviewBaseRefreshError:      startParams.ReviewBaseRefreshError,
		rejectIncompatibleActiveRun: mode == types.WorktreeModeCurrent,
	})
}

func (m *RunManager) tryResumeCurrentRun(ctx context.Context, repo *db.Repo, params *ipc.StartRunParams) (string, bool, error) {
	rawWorkDir := strings.TrimSpace(params.WorkDir)
	if rawWorkDir == "" || !filepath.IsAbs(rawWorkDir) {
		return "", false, nil
	}
	workDir, err := git.CurrentWorktreeRoot(ctx, rawWorkDir)
	if err != nil {
		return "", false, nil
	}
	if err := verifyCurrentWorktreeRepo(repo, workDir); err != nil {
		return "", false, err
	}
	branch, err := git.CurrentBranch(ctx, workDir)
	if err != nil || branch == "HEAD" {
		return "", false, nil
	}
	if requested := strings.TrimSpace(params.Branch); requested != "" && requested != branch {
		return "", false, nil
	}
	active, err := m.db.GetActiveRun(repo.ID, branch)
	if err != nil {
		return "", false, fmt.Errorf("get active run: %w", err)
	}
	if active == nil {
		return "", false, nil
	}
	if len(params.SkipSteps) == 0 && currentStartResumeCompatible(active, params.HeadSHA, params.BaseSHA, workDir, params.Intent) {
		return active.ID, true, nil
	}
	return "", false, activeRunConflict(active, types.WorktreeModeCurrent, params.HeadSHA, safeWorkDirLabel(workDir))
}

func prepareCurrentStartParams(ctx context.Context, repo *db.Repo, params *ipc.StartRunParams) error {
	rawWorkDir := strings.TrimSpace(params.WorkDir)
	if rawWorkDir == "" {
		return currentStartError("current worktree mode requires work_dir", types.RejectionActiveRunConflict, "Pass the absolute path to the git worktree root")
	}
	if !filepath.IsAbs(rawWorkDir) {
		return currentStartError("current worktree mode requires an absolute work_dir", types.RejectionActiveRunConflict, "Pass the absolute path to the git worktree root")
	}
	workDir, err := git.CurrentWorktreeRoot(ctx, rawWorkDir)
	if err != nil {
		return currentStartError(fmt.Sprintf("resolve current worktree: %v", err), types.RejectionActiveRunConflict, "Run from an initialized git worktree")
	}
	if err := verifyCurrentWorktreeRepo(repo, workDir); err != nil {
		return err
	}
	branch, err := git.CurrentBranch(ctx, workDir)
	if err != nil {
		return currentStartError(fmt.Sprintf("resolve current branch: %v", err), types.RejectionUnbornHead, "Check out a branch with at least one commit")
	}
	if branch == "HEAD" {
		return currentStartError("detached HEAD", types.RejectionDetachedHead, "Check out a branch before validating")
	}
	if requested := strings.TrimSpace(params.Branch); requested != "" && requested != branch {
		return currentStartError(fmt.Sprintf("requested branch %q does not match current branch %q", requested, branch), types.RejectionActiveRunConflict, "Retry from the branch you want to validate")
	}
	if repo.DefaultBranch != "" && branch == repo.DefaultBranch {
		return currentStartError(fmt.Sprintf("refusing to validate default branch %q", branch), types.RejectionDefaultBranch, "Switch to a feature branch, then retry")
	}
	headSHA, err := git.HeadSHA(ctx, workDir)
	if err != nil {
		return currentStartError(fmt.Sprintf("resolve current HEAD: %v", err), types.RejectionUnbornHead, "Create at least one commit on the branch before validating")
	}
	if requested := strings.TrimSpace(params.HeadSHA); requested != "" && requested != headSHA {
		return currentStartError("requested head does not match current HEAD", types.RejectionActiveRunConflict, "Retry after refreshing the current branch state")
	}
	dirty, err := git.HasCommittedWorktreeDirt(ctx, workDir)
	if err != nil {
		return currentStartError(fmt.Sprintf("inspect current worktree: %v", err), types.RejectionDirtyWorktree, "Run git status, then retry")
	}
	if dirty {
		return currentStartError("current worktree is not clean", types.RejectionDirtyWorktree, "Commit or remove tracked changes and untracked non-ignored files, then retry")
	}
	remoteName := currentModeRemoteName(ctx, workDir, repo.UpstreamURL)
	base, err := git.ResolveCurrentReviewBase(ctx, workDir, remoteName, repo.DefaultBranch)
	if err != nil {
		return currentStartError(fmt.Sprintf("cannot prove default-branch merge base: %v", err), types.RejectionNoTrustworthyBase, "Fetch the default branch or fix remote access, then retry")
	}
	label := safeWorkDirLabel(workDir)
	params.Branch = branch
	params.HeadSHA = headSHA
	params.BaseSHA = base.BaseSHA
	params.WorkDir = workDir
	params.WorkDirLabel = label
	params.CurrentWorktreeWarning = currentWorktreeWarning(label)
	params.ReviewBaseRef = base.Ref
	params.ReviewBaseRefreshAttempted = base.RefreshAttempted
	params.ReviewBaseRefreshError = base.RefreshError
	return nil
}

func verifyCurrentWorktreeRepo(repo *db.Repo, workDir string) error {
	if repo == nil {
		return currentStartError("unknown repo", types.RejectionRepoMismatch, "Run no-mistakes init for this checkout, then retry")
	}
	repoRoot, err := git.FindMainRepoRoot(repo.WorkingPath)
	if err != nil {
		return currentStartError(fmt.Sprintf("resolve registered repo root: %v", err), types.RejectionRepoMismatch, "Run no-mistakes init for this checkout, then retry")
	}
	workRoot, err := git.FindMainRepoRoot(workDir)
	if err != nil {
		return currentStartError(fmt.Sprintf("resolve current repo root: %v", err), types.RejectionRepoMismatch, "Run from the checkout registered with no-mistakes")
	}
	if repoRoot != workRoot {
		return currentStartError(
			fmt.Sprintf("current worktree %q does not belong to registered repo %q", workDir, repo.WorkingPath),
			types.RejectionRepoMismatch,
			"Run no-mistakes init in this checkout, or retry from the registered repo",
		)
	}
	return nil
}

func currentStartError(message, reason, recovery string) *CurrentWorktreeStartError {
	return &CurrentWorktreeStartError{
		Message:      message,
		Reason:       reason,
		Recovery:     recovery,
		WorktreeMode: types.WorktreeModeCurrent,
	}
}

type startRunOptions struct {
	trigger                     string
	skipSteps                   []types.StepName
	intent                      string
	worktreeMode                types.WorktreeMode
	workDir                     string
	workDirLabel                string
	currentWorktreeWarning      string
	reviewBaseRef               string
	reviewBaseRefreshAttempted  bool
	reviewBaseRefreshError      string
	rejectIncompatibleActiveRun bool
}

// startRun creates a run, sets up a worktree, and launches pipeline execution.
// A non-empty intent is stamped onto the run as agent-supplied, so the intent
// step uses it instead of inferring from transcripts.
func (m *RunManager) startRun(ctx context.Context, repo *db.Repo, branch, headSHA, baseSHA, trigger string, skipSteps []types.StepName, intent string) (string, error) {
	runID, _, err := m.startRunWithOptions(ctx, repo, branch, headSHA, baseSHA, startRunOptions{
		trigger:      trigger,
		skipSteps:    skipSteps,
		intent:       intent,
		worktreeMode: types.WorktreeModeIsolated,
	})
	return runID, err
}

func (m *RunManager) startRunWithOptions(ctx context.Context, repo *db.Repo, branch, headSHA, baseSHA string, opts startRunOptions) (string, bool, error) {
	trigger := opts.trigger
	if trigger == "" {
		trigger = "direct"
	}
	mode := types.NormalizeWorktreeMode(opts.worktreeMode)
	if mode == "" {
		mode = types.WorktreeModeIsolated
	}
	branchRole := telemetryBranchRole(branch, repo.DefaultBranch)
	trackStartFailure := func(stage string) {
		telemetry.Track("run", telemetry.Fields{
			"action":      "start_failed",
			"trigger":     trigger,
			"branch_role": branchRole,
			"stage":       stage,
		})
	}

	if m.shuttingDown.Load() {
		trackStartFailure("daemon_shutdown")
		return "", false, fmt.Errorf("daemon is shutting down")
	}

	// Serialize per repo+branch to prevent two concurrent pushes from both
	// passing cancelActiveRuns and creating duplicate pipelines.
	lockKey := repo.ID + "/" + branch
	lockVal, _ := m.branchLocks.LoadOrStore(lockKey, &sync.Mutex{})
	branchMu := lockVal.(*sync.Mutex)
	branchMu.Lock()
	defer branchMu.Unlock()

	if opts.rejectIncompatibleActiveRun {
		active, err := m.db.GetActiveRun(repo.ID, branch)
		if err != nil {
			trackStartFailure("active_run_lookup")
			return "", false, fmt.Errorf("get active run: %w", err)
		}
		if active != nil {
			if len(opts.skipSteps) == 0 && currentStartCompatible(active, headSHA, baseSHA, opts.workDir, opts.intent) {
				return active.ID, true, nil
			}
			trackStartFailure("active_run_conflict")
			return "", false, activeRunConflict(active, mode, headSHA, opts.workDirLabel)
		}
	} else {
		active, err := m.db.GetActiveRun(repo.ID, branch)
		if err != nil {
			trackStartFailure("active_run_lookup")
			return "", false, fmt.Errorf("get active run: %w", err)
		}
		if active != nil && types.NormalizeWorktreeMode(active.WorktreeMode) != mode {
			trackStartFailure("active_run_conflict")
			return "", false, activeRunConflict(active, mode, headSHA, opts.workDirLabel)
		}
		// Cancel same-mode active runs for this repo+branch.
		m.cancelActiveRuns(repo.ID, branch)
	}

	// Create run record.
	run, err := m.db.InsertRunWithOptions(repo.ID, branch, headSHA, baseSHA, db.RunInsertOptions{
		WorktreeMode:               mode,
		WorkDir:                    opts.workDir,
		WorkDirLabel:               opts.workDirLabel,
		CurrentWorktreeWarning:     opts.currentWorktreeWarning,
		ReviewBaseRef:              opts.reviewBaseRef,
		ReviewBaseRefreshAttempted: opts.reviewBaseRefreshAttempted,
		ReviewBaseRefreshError:     opts.reviewBaseRefreshError,
	})
	if err != nil {
		trackStartFailure("create_run")
		return "", false, fmt.Errorf("create run: %w", err)
	}

	// Stamp an agent-supplied intent onto the run before the pipeline starts,
	// so the intent step finds it already present and skips transcript-based
	// inference. A persist failure is non-fatal: the intent step would simply
	// fall back to inference.
	if trimmed := strings.TrimSpace(opts.intent); trimmed != "" {
		if err := m.db.UpdateRunIntent(run.ID, db.RunIntent{Summary: trimmed, Source: "agent", Score: 1}); err != nil {
			slog.Warn("failed to persist agent-supplied intent", "run_id", run.ID, "error", err)
		} else {
			run.Intent = &trimmed
			source := "agent"
			run.IntentSource = &source
			score := 1.0
			run.IntentScore = &score
		}
	}

	gateDir := m.paths.RepoDir(repo.ID)
	workDir := opts.workDir
	managedWorktree := mode != types.WorktreeModeCurrent
	if managedWorktree {
		// Create worktree from the gate bare repo.
		workDir = m.paths.WorktreeDir(repo.ID, run.ID)
		if err := git.WorktreeAdd(ctx, gateDir, workDir, headSHA); err != nil {
			m.db.UpdateRunError(run.ID, fmt.Sprintf("create worktree: %s", err))
			m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
			trackStartFailure("create_worktree")
			return "", false, fmt.Errorf("create worktree: %w", err)
		}
		if err := git.CopyLocalUserIdentity(ctx, repo.WorkingPath, workDir); err != nil {
			m.db.UpdateRunError(run.ID, fmt.Sprintf("configure worktree git identity: %s", err))
			m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
			trackStartFailure("configure_worktree_identity")
			return "", false, fmt.Errorf("configure worktree git identity: %w", err)
		}
		if repo.DefaultBranch != "" {
			if err := git.FetchRemoteBranch(ctx, workDir, "origin", repo.DefaultBranch); err != nil {
				slog.Warn("failed to fetch default branch into worktree", "run_id", run.ID, "branch", repo.DefaultBranch, "error", err)
			}
		}
	} else if strings.TrimSpace(workDir) == "" {
		m.db.UpdateRunError(run.ID, "current worktree mode requires work_dir")
		m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
		trackStartFailure("missing_current_workdir")
		return "", false, fmt.Errorf("current worktree mode requires work_dir")
	}

	// Track whether the background goroutine takes ownership of worktree cleanup.
	// If setup fails before the goroutine launches, we must clean up here.
	bgOwnsCleanup := false
	defer func() {
		if managedWorktree && !bgOwnsCleanup {
			if rmErr := git.WorktreeRemove(context.Background(), gateDir, workDir); rmErr != nil {
				slog.Warn("failed to remove worktree during setup cleanup", "path", workDir, "error", rmErr)
			}
		}
	}()

	globalCfg, err := config.LoadGlobal(m.paths.ConfigFile())
	if err != nil {
		m.db.UpdateRunError(run.ID, fmt.Sprintf("load config: %s", err))
		m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
		trackStartFailure("load_global_config")
		return "", false, fmt.Errorf("load global config: %w", err)
	}
	repoCfg, err := config.LoadRepo(workDir)
	if err != nil {
		m.db.UpdateRunError(run.ID, fmt.Sprintf("load config: %s", err))
		m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
		trackStartFailure("load_repo_config")
		return "", false, fmt.Errorf("load repo config: %w", err)
	}
	cfg := config.Merge(globalCfg, repoCfg)

	// Create agent. In demo mode, skip resolution and use a no-op agent.
	var ag agent.Agent
	if steps.IsDemoMode() {
		ag = agent.NewNoop()
	} else {
		if err := cfg.ResolveAgent(ctx, exec.LookPath); err != nil {
			m.db.UpdateRunError(run.ID, err.Error())
			m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
			trackStartFailure("resolve_agent")
			return "", false, err
		}
		var agErr error
		ag, agErr = agent.NewWithOptions(cfg.Agent, cfg.AgentPath(), cfg.AgentArgs(), agent.Options{
			ACPRegistryOverrides: cfg.ACPRegistryOverrides,
		})
		if agErr != nil {
			m.db.UpdateRunError(run.ID, fmt.Sprintf("create agent: %s", agErr))
			m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonSetupFailed, types.EvidenceIncomplete)
			trackStartFailure("create_agent")
			return "", false, fmt.Errorf("create agent: %w", agErr)
		}
		// Steer every pipeline agent to keep writes inside the worktree and
		// avoid mutating system state (e.g. brew/Homebrew touching
		// /Applications), which triggers macOS App Management prompts.
		ag = agent.WithSteering(ag)
	}

	execSteps := m.steps()
	telemetry.Track("run", telemetry.Fields{
		"action":      "started",
		"trigger":     trigger,
		"agent":       string(cfg.Agent),
		"branch_role": branchRole,
		"step_count":  len(execSteps),
		"demo_mode":   steps.IsDemoMode(),
	})

	// Create executor with event broadcast.
	runCtx, cancel := context.WithCancelCause(context.Background())
	executor := pipeline.NewExecutor(m.db, m.paths, cfg, ag, execSteps, m.broadcast)
	executor.SetSkippedSteps(opts.skipSteps)

	// Track executor.
	done := make(chan struct{})
	m.mu.Lock()
	m.executors[run.ID] = executor
	m.cancels[run.ID] = cancel
	m.dones[run.ID] = done
	m.mu.Unlock()

	// Background goroutine now owns worktree cleanup.
	bgOwnsCleanup = true

	// Launch pipeline in background.
	m.wg.Add(1)
	go func() {
		startedAt := time.Now()
		defer m.wg.Done()
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("internal panic: %v", r)
				slog.Error("panic in pipeline goroutine", "run_id", run.ID, "panic", r)
				run.Status = types.RunFailed
				run.Error = &errMsg
				fields := telemetry.Fields{
					"action":      "finished",
					"trigger":     trigger,
					"agent":       string(cfg.Agent),
					"branch_role": branchRole,
					"status":      string(run.Status),
					"duration_ms": time.Since(startedAt).Milliseconds(),
					"step_count":  len(execSteps),
					"pr_created":  run.PRURL != nil && *run.PRURL != "",
				}
				if failedStep := telemetryFailedStepName(m.db, run.ID); failedStep != "" {
					fields["failed_step"] = failedStep
				}
				telemetry.Track("run", fields)
				if dbErr := m.db.UpdateRunErrorStatus(run.ID, errMsg, types.RunFailed); dbErr != nil {
					slog.Error("failed to update run after panic", "run_id", run.ID, "error", dbErr)
				}
				if dbErr := m.db.UpdateRunTerminalReason(run.ID, types.RunTerminalReasonDaemonCrashed, types.EvidenceIncomplete); dbErr != nil {
					slog.Error("failed to update run terminal reason after panic", "run_id", run.ID, "error", dbErr)
				}
			}
			cancel(nil)
			ag.Close()
			// Close subscriber channels for this run.
			m.closeSubscribers(run.ID)
			// Clean up only no-mistakes-managed worktrees. Current-worktree
			// mode runs in the user's checkout and must never remove it.
			if managedWorktree {
				if rmErr := git.WorktreeRemove(context.Background(), gateDir, workDir); rmErr != nil {
					slog.Warn("failed to remove worktree", "path", workDir, "error", rmErr)
				}
			}
			// Remove tracking.
			m.mu.Lock()
			delete(m.executors, run.ID)
			delete(m.cancels, run.ID)
			delete(m.dones, run.ID)
			m.mu.Unlock()
		}()

		if err := executor.Execute(runCtx, run, repo, workDir); err != nil {
			fields := telemetry.Fields{
				"action":      "finished",
				"trigger":     trigger,
				"agent":       string(cfg.Agent),
				"branch_role": branchRole,
				"status":      string(run.Status),
				"duration_ms": time.Since(startedAt).Milliseconds(),
				"step_count":  len(execSteps),
				"pr_created":  run.PRURL != nil && *run.PRURL != "",
			}
			if failedStep := telemetryFailedStepName(m.db, run.ID); failedStep != "" {
				fields["failed_step"] = failedStep
			}
			telemetry.Track("run", fields)
			slog.Error("pipeline failed", "run_id", run.ID, "error", err)
		} else {
			telemetry.Track("run", telemetry.Fields{
				"action":      "finished",
				"trigger":     trigger,
				"agent":       string(cfg.Agent),
				"branch_role": branchRole,
				"status":      string(run.Status),
				"duration_ms": time.Since(startedAt).Milliseconds(),
				"step_count":  len(execSteps),
				"pr_created":  run.PRURL != nil && *run.PRURL != "",
			})
			slog.Info("pipeline completed", "run_id", run.ID)
		}
	}()

	return run.ID, false, nil
}

func currentStartCompatible(active *db.Run, headSHA, baseSHA, workDir, intent string) bool {
	if active == nil {
		return false
	}
	if types.NormalizeWorktreeMode(active.WorktreeMode) != types.WorktreeModeCurrent {
		return false
	}
	if active.HeadSHA != headSHA || active.BaseSHA != baseSHA {
		return false
	}
	if active.WorkDir == nil || *active.WorkDir != workDir {
		return false
	}
	requestedIntent := strings.TrimSpace(intent)
	if requestedIntent != "" {
		if active.Intent == nil || strings.TrimSpace(*active.Intent) != requestedIntent {
			return false
		}
	}
	return true
}

func currentStartResumeCompatible(active *db.Run, headSHA, baseSHA, workDir, intent string) bool {
	if active == nil {
		return false
	}
	if types.NormalizeWorktreeMode(active.WorktreeMode) != types.WorktreeModeCurrent {
		return false
	}
	if strings.TrimSpace(headSHA) == "" || active.HeadSHA != headSHA {
		return false
	}
	if strings.TrimSpace(baseSHA) != "" && active.BaseSHA != baseSHA {
		return false
	}
	if active.WorkDir == nil || *active.WorkDir != workDir {
		return false
	}
	requestedIntent := strings.TrimSpace(intent)
	if requestedIntent != "" {
		if active.Intent == nil || strings.TrimSpace(*active.Intent) != requestedIntent {
			return false
		}
	}
	return true
}

func activeRunConflict(active *db.Run, requestedMode types.WorktreeMode, requestedHead, requestedWorkDirLabel string) *ActiveRunConflictError {
	label := active.WorktreeMode.Label()
	if active.WorkDirLabel != nil && *active.WorkDirLabel != "" {
		label = *active.WorkDirLabel
	}
	shortActive := active.HeadSHA
	if len(shortActive) > 8 {
		shortActive = shortActive[:8]
	}
	shortRequested := requestedHead
	if len(shortRequested) > 8 {
		shortRequested = shortRequested[:8]
	}
	return &ActiveRunConflictError{
		RunID:            active.ID,
		WorktreeMode:     types.NormalizeWorktreeMode(active.WorktreeMode),
		Branch:           active.Branch,
		ShortHead:        shortActive,
		WorkDirLabel:     label,
		Status:           active.Status,
		ResumeCommand:    "no-mistakes axi run --intent \"...\"",
		AbortCommand:     "no-mistakes axi abort --run " + active.ID,
		RequestedMode:    requestedMode,
		RequestedHead:    shortRequested,
		RequestedWorkDir: requestedWorkDirLabel,
	}
}

func telemetryBranchRole(branch, defaultBranch string) string {
	if branch == "" {
		return "unknown"
	}
	if defaultBranch != "" && branch == defaultBranch {
		return "default"
	}
	return "feature"
}

func telemetryFailedStepName(database *db.DB, runID string) string {
	steps, err := database.GetStepsByRun(runID)
	if err != nil {
		return ""
	}
	for _, step := range steps {
		if step.Status == types.StepStatusFailed {
			return string(step.StepName)
		}
	}
	return ""
}

func currentModeRemoteName(ctx context.Context, workDir, upstreamURL string) string {
	remotes, err := git.Run(ctx, workDir, "remote")
	if err != nil {
		return "origin"
	}
	upstreamURL = strings.TrimSpace(upstreamURL)
	for _, remote := range strings.Fields(remotes) {
		url, err := git.GetRemoteURL(ctx, workDir, remote)
		if err == nil && upstreamURL != "" && strings.TrimSpace(url) == upstreamURL {
			return remote
		}
	}
	return "origin"
}

func safeWorkDirLabel(workDir string) string {
	base := filepath.Base(workDir)
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "current checkout"
	}
	return base
}

func currentWorktreeWarning(label string) string {
	return fmt.Sprintf("%s: uses this checkout; pipeline fixes may modify it and commits remain here", label)
}

// HandleRespond routes a user approval action to the executor for the given run.
func (m *RunManager) HandleRespond(runID string, step types.StepName, action types.ApprovalAction, findingIDs []string) error {
	return m.HandleRespondWithOverrides(runID, step, action, findingIDs, nil, nil)
}

// HandleRespondWithOverrides is like HandleRespond but also forwards user
// instructions and user-authored findings to the executor.
func (m *RunManager) HandleRespondWithOverrides(runID string, step types.StepName, action types.ApprovalAction, findingIDs []string, instructions map[string]string, addedFindings []types.Finding) error {
	m.mu.Lock()
	exec, ok := m.executors[runID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("no active executor for run %s", runID)
	}

	return exec.RespondWithOverrides(step, action, findingIDs, instructions, addedFindings)
}

// Shutdown cancels all active runs. Called during daemon shutdown to prevent
// orphaned goroutines from continuing agent calls and git operations.
func (m *RunManager) Shutdown() {
	m.shuttingDown.Store(true)

	m.mu.Lock()
	cancels := make(map[string]context.CancelCauseFunc, len(m.cancels))
	for id, cancel := range m.cancels {
		cancels[id] = cancel
	}
	m.mu.Unlock()

	for id, cancel := range cancels {
		cancel(fmt.Errorf("daemon shutting down"))
		slog.Info("cancelled run on shutdown", "run_id", id)
	}

	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		slog.Warn("timed out waiting for runs to finish during shutdown")
	}
}

// HandleCancel stops an active run and propagates cancellation to the executor.
func (m *RunManager) HandleCancel(runID string) error {
	m.mu.Lock()
	cancel, ok := m.cancels[runID]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("no active run %s", runID)
	}

	cancel(fmt.Errorf(types.RunCancelReasonAbortedByUser))
	return nil
}

// cancelActiveRuns cancels any in-progress runs for the given repo+branch
// and waits for their goroutines to finish before returning, preventing
// concurrent pushes to upstream.
// The cancellation cause is propagated to the executor via context.Cause,
// which uses it as the run's error message in the DB.
func (m *RunManager) cancelActiveRuns(repoID, branch string) {
	runs, err := m.db.GetRunsByRepo(repoID)
	if err != nil {
		slog.Error("failed to query active runs for cancellation", "repo", repoID, "branch", branch, "error", err)
		return
	}

	var toWait []chan struct{}
	for _, run := range runs {
		if run.Branch != branch {
			continue
		}
		if run.Status != types.RunPending && run.Status != types.RunRunning {
			continue
		}

		m.mu.Lock()
		cancel, ok := m.cancels[run.ID]
		done := m.dones[run.ID]
		m.mu.Unlock()
		if !ok {
			continue
		}

		cancel(fmt.Errorf(types.RunCancelReasonSuperseded))
		slog.Info("cancelled active run", "run_id", run.ID, "repo_id", repoID, "branch", branch)
		if done != nil {
			toWait = append(toWait, done)
		}
	}

	timeout := time.After(30 * time.Second)
	for _, done := range toWait {
		select {
		case <-done:
		case <-timeout:
			slog.Warn("timed out waiting for cancelled runs to finish")
			return
		}
	}
}
