package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// RunStatus represents the lifecycle state of a pipeline run.
type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunCompleted RunStatus = "completed"
	RunFailed    RunStatus = "failed"
	RunCancelled RunStatus = "cancelled"
)

const (
	RunCancelReasonAbortedByUser = "cancelled: aborted by user"
	RunCancelReasonSuperseded    = "cancelled: superseded by new push"
)

// WorktreeMode describes where a run executes.
type WorktreeMode string

const (
	WorktreeModeIsolated WorktreeMode = "isolated"
	WorktreeModeCurrent  WorktreeMode = "current"
)

// NormalizeWorktreeMode maps legacy empty values to the historical isolated
// behavior. Unknown non-empty values are preserved so callers can fail closed.
func NormalizeWorktreeMode(mode WorktreeMode) WorktreeMode {
	if mode == "" {
		return WorktreeModeIsolated
	}
	return mode
}

func (m WorktreeMode) Valid() bool {
	switch NormalizeWorktreeMode(m) {
	case WorktreeModeIsolated, WorktreeModeCurrent:
		return true
	default:
		return false
	}
}

func (m WorktreeMode) Label() string {
	switch NormalizeWorktreeMode(m) {
	case WorktreeModeCurrent:
		return "uses this checkout"
	default:
		return "disposable no-mistakes checkout"
	}
}

// MetadataAvailability describes whether run metadata is trustworthy enough to
// render and clean up normally.
type MetadataAvailability string

const (
	MetadataAvailable   MetadataAvailability = "available"
	MetadataNotRecorded MetadataAvailability = "not_recorded"
	MetadataInvalid     MetadataAvailability = "invalid"
)

func NormalizeMetadataAvailability(v MetadataAvailability) MetadataAvailability {
	if v == "" {
		return MetadataAvailable
	}
	return v
}

func (v MetadataAvailability) Valid() bool {
	switch NormalizeMetadataAvailability(v) {
	case MetadataAvailable, MetadataNotRecorded, MetadataInvalid:
		return true
	default:
		return false
	}
}

// EvidenceState is the run-level evidence reconstruction state rendered after
// failures, cancellations, stale recovery, or malformed metadata.
type EvidenceState string

const (
	EvidenceComplete   EvidenceState = "complete"
	EvidenceIncomplete EvidenceState = "incomplete"
	EvidenceDegraded   EvidenceState = "degraded"
)

func NormalizeEvidenceState(v EvidenceState) EvidenceState {
	if v == "" {
		return EvidenceComplete
	}
	return v
}

func (v EvidenceState) Valid() bool {
	switch NormalizeEvidenceState(v) {
	case EvidenceComplete, EvidenceIncomplete, EvidenceDegraded:
		return true
	default:
		return false
	}
}

const (
	RunTerminalReasonSetupFailed       = "setup_failed"
	RunTerminalReasonDaemonCrashed     = "daemon_crashed"
	RunTerminalReasonCancelledByUser   = "cancelled_by_user"
	RunTerminalReasonSuperseded        = "superseded"
	RunTerminalReasonNoTrustworthyBase = "rejected_no_trustworthy_base"
)

const (
	RejectionNoTrustworthyBase = "rejected_no_trustworthy_base"
	RejectionDirtyWorktree     = "rejected_dirty_worktree"
	RejectionDetachedHead      = "rejected_detached_head"
	RejectionDefaultBranch     = "rejected_default_branch"
	RejectionUnbornHead        = "rejected_unborn_head"
	RejectionActiveRunConflict = "rejected_active_run_conflict"
	RejectionRepoMismatch      = "rejected_repo_mismatch"
	RejectionMissingIntent     = "missing_intent"
)

// StepName identifies a pipeline step.
type StepName string

const (
	StepIntent   StepName = "intent"
	StepRebase   StepName = "rebase"
	StepReview   StepName = "review"
	StepTest     StepName = "test"
	StepDocument StepName = "document"
	StepLint     StepName = "lint"
	StepPush     StepName = "push"
	StepPR       StepName = "pr"
	StepCI       StepName = "ci"
)

func normalizeStepName(s StepName) StepName {
	if s == "babysit" {
		return StepCI
	}
	return s
}

func (s *StepName) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = normalizeStepName(StepName(raw))
	return nil
}

func (s *StepName) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*s = normalizeStepName(StepName(v))
		return nil
	case []byte:
		*s = normalizeStepName(StepName(v))
		return nil
	case nil:
		*s = ""
		return nil
	default:
		return fmt.Errorf("scan StepName from %T", src)
	}
}

func (s StepName) Value() (driver.Value, error) {
	return string(s), nil
}

// StepOrder returns the fixed execution order for a step (1-indexed).
func (s StepName) Order() int {
	switch s {
	case StepIntent:
		return 1
	case StepRebase:
		return 2
	case StepReview:
		return 3
	case StepTest:
		return 4
	case StepDocument:
		return 5
	case StepLint:
		return 6
	case StepPush:
		return 7
	case StepPR:
		return 8
	case StepCI:
		return 9
	default:
		return 0
	}
}

// AllSteps returns all pipeline steps in execution order.
func AllSteps() []StepName {
	return []StepName{StepIntent, StepRebase, StepReview, StepTest, StepDocument, StepLint, StepPush, StepPR, StepCI}
}

// StepStatus represents the lifecycle state of a pipeline step.
type StepStatus string

const (
	StepStatusPending          StepStatus = "pending"
	StepStatusRunning          StepStatus = "running"
	StepStatusAwaitingApproval StepStatus = "awaiting_approval"
	StepStatusFixing           StepStatus = "fixing"
	StepStatusFixReview        StepStatus = "fix_review"
	StepStatusCompleted        StepStatus = "completed"
	StepStatusSkipped          StepStatus = "skipped"
	StepStatusFailed           StepStatus = "failed"
)

// ApprovalAction represents user responses at approval points.
type ApprovalAction string

const (
	ActionApprove ApprovalAction = "approve"
	ActionFix     ApprovalAction = "fix"
	ActionSkip    ApprovalAction = "skip"
	ActionAbort   ApprovalAction = "abort"
)

// AgentName identifies a supported agent backend.
// ACP agent names use dynamic acp:<target> values instead of constants.
type AgentName string

const (
	AgentAuto     AgentName = "auto"
	AgentClaude   AgentName = "claude"
	AgentCodex    AgentName = "codex"
	AgentRovoDev  AgentName = "rovodev"
	AgentOpenCode AgentName = "opencode"
	AgentPi       AgentName = "pi"
)
