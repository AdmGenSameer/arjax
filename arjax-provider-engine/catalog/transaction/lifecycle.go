package transaction

import (
	"time"
)

// TransactionState represents the canonical lifecycle stages of a package transaction
type TransactionState string

const (
	TxQueued        TransactionState = "queued"
	TxResolvingDeps TransactionState = "resolving_deps"
	TxAwaitingAuth  TransactionState = "awaiting_auth"
	TxDownloading   TransactionState = "downloading"
	TxVerifying     TransactionState = "verifying"
	TxInstalling    TransactionState = "installing"
	TxCompleted     TransactionState = "completed"
	TxFailed        TransactionState = "failed"
	TxCancelled     TransactionState = "cancelled"
)

// TransactionOp distinguishes installation, removal, and upgrade transactions
type TransactionOp string

const (
	OpInstall TransactionOp = "install"
	OpRemove  TransactionOp = "remove"
	OpUpdate  TransactionOp = "update"
)

// RecoveryStatus explicitly tracks normal termination and crash reconciliation verdicts
type RecoveryStatus string

const (
	RecoveryNone               RecoveryStatus = "none"
	RecoveryCompleted          RecoveryStatus = "completed"
	RecoveryFailed             RecoveryStatus = "failed"
	RecoveryReconciledComplete RecoveryStatus = "reconciled_completed"
	RecoveryReconciledFailed   RecoveryStatus = "reconciled_failed"
	RecoveryRequired           RecoveryStatus = "recovery_required"
	RecoveryOutcomeUnknown     RecoveryStatus = "outcome_unknown"
)

// TransactionDTO is the unified persistent progress and audit model shared across GUI & TUI
type TransactionDTO struct {
	ID              string           `json:"id"`
	OfferID         string           `json:"offerId"`
	AppID           string           `json:"appId"`
	PackageName     string           `json:"packageName"`
	Provider        string           `json:"provider"`
	Repository      string           `json:"repository"`
	InstallScope    string           `json:"installScope,omitempty"` // "user", "system", "sandboxed"
	PreviousVersion string           `json:"previousVersion,omitempty"`
	TargetVersion   string           `json:"targetVersion,omitempty"`
	Operation       TransactionOp    `json:"operation"`
	State           TransactionState `json:"state"`
	Progress        int              `json:"progress"`
	Message         string           `json:"message"`
	Dependencies    []string         `json:"dependencies,omitempty"`
	Logs            []string         `json:"logs"`
	CanCancelSafely bool             `json:"canCancelSafely"`
	CancelRequested bool             `json:"cancelRequested"`
	ExitCode        int              `json:"exitCode"`
	ErrorCode       string           `json:"errorCode,omitempty"`
	ErrorMessage    string           `json:"errorMessage,omitempty"`
	RecoveryStatus  RecoveryStatus   `json:"recoveryStatus"`
	StartedAt       time.Time        `json:"startedAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
}

// IsTerminal reports whether the transaction is in a finished state
func (s TransactionState) IsTerminal() bool {
	return s == TxCompleted || s == TxFailed || s == TxCancelled
}
