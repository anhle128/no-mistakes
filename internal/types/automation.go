package types

// BoundaryStatus is the controller-owned execution boundary classification.
type BoundaryStatus string

const (
	BoundarySafe    BoundaryStatus = "safe"
	BoundaryUnsafe  BoundaryStatus = "unsafe"
	BoundaryUnknown BoundaryStatus = "unknown"
)

func (s BoundaryStatus) Valid() bool {
	switch s {
	case BoundarySafe, BoundaryUnsafe, BoundaryUnknown:
		return true
	default:
		return false
	}
}

// BoundaryReason explains why a run received its current boundary status.
type BoundaryReason string

const (
	BoundaryReasonVerifiedRunWorktree BoundaryReason = "verified_run_worktree"
	BoundaryReasonPrimaryCheckout     BoundaryReason = "primary_checkout"
	BoundaryReasonMissingWorktree     BoundaryReason = "missing_worktree"
	BoundaryReasonGitMetadataMismatch BoundaryReason = "git_metadata_mismatch"
	BoundaryReasonSymlinkEscape       BoundaryReason = "symlink_escape"
	BoundaryReasonStaleProof          BoundaryReason = "stale_proof"
	BoundaryReasonSourceOutside       BoundaryReason = "source_outside_worktree"
	BoundaryReasonUnknown             BoundaryReason = "unknown"
)

func (r BoundaryReason) Valid() bool {
	switch r {
	case BoundaryReasonVerifiedRunWorktree, BoundaryReasonPrimaryCheckout, BoundaryReasonMissingWorktree,
		BoundaryReasonGitMetadataMismatch, BoundaryReasonSymlinkEscape, BoundaryReasonStaleProof,
		BoundaryReasonSourceOutside, BoundaryReasonUnknown:
		return true
	default:
		return false
	}
}

// ExecutionBoundary is the persisted and IPC-visible boundary proof.
type ExecutionBoundary struct {
	Status               BoundaryStatus `json:"status"`
	Reason               BoundaryReason `json:"reason"`
	Detail               string         `json:"detail,omitempty"`
	ExpectedWorktreePath string         `json:"expected_worktree_path,omitempty"`
	ActualWorktreePath   string         `json:"actual_worktree_path,omitempty"`
	GitCommonDir         string         `json:"git_common_dir,omitempty"`
	GateRepoPath         string         `json:"gate_repo_path,omitempty"`
	Fingerprint          string         `json:"fingerprint,omitempty"`
	VerifiedAt           int64          `json:"verified_at,omitempty"`
	VerifierVersion      string         `json:"verifier_version,omitempty"`
}

// Normalize returns a conservative, display-safe boundary value.
func (b ExecutionBoundary) Normalize() ExecutionBoundary {
	if !b.Status.Valid() {
		b.Status = BoundaryUnknown
	}
	if !b.Reason.Valid() {
		b.Reason = BoundaryReasonUnknown
	}
	return b
}

// GateAutomationStatus is the current unattended automation state for a gate.
type GateAutomationStatus string

const (
	GateAutomationAllowed      GateAutomationStatus = "allowed"
	GateAutomationWithheld     GateAutomationStatus = "withheld"
	GateAutomationNotRequested GateAutomationStatus = "not_requested"
)

func (s GateAutomationStatus) Valid() bool {
	switch s {
	case GateAutomationAllowed, GateAutomationWithheld, GateAutomationNotRequested:
		return true
	default:
		return false
	}
}

type DecisionSource string

const (
	DecisionSourceManual     DecisionSource = "manual"
	DecisionSourceUnattended DecisionSource = "unattended"
)

func (s DecisionSource) Valid() bool {
	switch s {
	case DecisionSourceManual, DecisionSourceUnattended:
		return true
	default:
		return false
	}
}

type ActorType string

const (
	ActorHuman  ActorType = "human"
	ActorAgent  ActorType = "agent"
	ActorSystem ActorType = "system"
)

func (a ActorType) Valid() bool {
	switch a {
	case ActorHuman, ActorAgent, ActorSystem:
		return true
	default:
		return false
	}
}

type ApprovalSurface string

const (
	ApprovalSurfaceTUI        ApprovalSurface = "tui"
	ApprovalSurfaceAXI        ApprovalSurface = "axi"
	ApprovalSurfaceHeadless   ApprovalSurface = "headless"
	ApprovalSurfaceAgentSkill ApprovalSurface = "agent-skill"
	ApprovalSurfaceDaemon     ApprovalSurface = "daemon"
	ApprovalSurfaceTerminal   ApprovalSurface = "terminal"
	ApprovalSurfaceUnknown    ApprovalSurface = "unknown"
)

func (s ApprovalSurface) Valid() bool {
	switch s {
	case ApprovalSurfaceTUI, ApprovalSurfaceAXI, ApprovalSurfaceHeadless,
		ApprovalSurfaceAgentSkill, ApprovalSurfaceDaemon, ApprovalSurfaceTerminal,
		ApprovalSurfaceUnknown:
		return true
	default:
		return false
	}
}

type ConsentMode string

const (
	ConsentModeNone            ConsentMode = "none"
	ConsentModeManual          ConsentMode = "manual"
	ConsentModeYolo            ConsentMode = "yolo"
	ConsentModeYes             ConsentMode = "yes"
	ConsentModeAgentUnattended ConsentMode = "agent-unattended"
)

func (m ConsentMode) Valid() bool {
	switch m {
	case ConsentModeNone, ConsentModeManual, ConsentModeYolo, ConsentModeYes, ConsentModeAgentUnattended:
		return true
	default:
		return false
	}
}

// DecisionMetadata identifies how a gate response was authorized.
type DecisionMetadata struct {
	DecisionSource  DecisionSource  `json:"decision_source,omitempty"`
	ActorType       ActorType       `json:"actor_type,omitempty"`
	ApprovalSurface ApprovalSurface `json:"approval_surface,omitempty"`
	ConsentMode     ConsentMode     `json:"consent_mode,omitempty"`
	GateID          string          `json:"gate_id,omitempty"`
	GateFingerprint string          `json:"gate_fingerprint,omitempty"`
}

// NormalizeRespondDecisionMetadata applies IPC backwards-compatible defaults.
func NormalizeRespondDecisionMetadata(meta DecisionMetadata) DecisionMetadata {
	if !meta.DecisionSource.Valid() {
		meta.DecisionSource = DecisionSourceManual
	}
	if !meta.ActorType.Valid() {
		meta.ActorType = ActorHuman
	}
	if !meta.ApprovalSurface.Valid() {
		meta.ApprovalSurface = ApprovalSurfaceUnknown
	}
	if !meta.ConsentMode.Valid() {
		if meta.DecisionSource == DecisionSourceUnattended {
			meta.ConsentMode = ConsentModeYolo
		} else {
			meta.ConsentMode = ConsentModeManual
		}
	}
	return meta
}

// GateAutomation reports the current automation state for an awaiting gate.
type GateAutomation struct {
	GateID          string               `json:"gate_id,omitempty"`
	GateFingerprint string               `json:"gate_fingerprint,omitempty"`
	Status          GateAutomationStatus `json:"status"`
	RequestedMode   ConsentMode          `json:"requested_mode"`
	Reason          string               `json:"reason"`
	Message         string               `json:"message,omitempty"`
	RecoveryOptions []string             `json:"recovery_options,omitempty"`
}
