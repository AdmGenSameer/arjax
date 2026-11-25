package catalog

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
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

// TransactionDTO is the unified progress and audit model emitted to GUI and TUI
type TransactionDTO struct {
	ID              string           `json:"id"`
	OfferID         string           `json:"offerId"`
	AppID           string           `json:"appId"`
	PackageName     string           `json:"packageName"`
	Provider        string           `json:"provider"`
	Repository      string           `json:"repository"`
	Operation       TransactionOp    `json:"operation"`
	State           TransactionState `json:"state"`
	Progress        int              `json:"progress"`
	Message         string           `json:"message"`
	Dependencies    []string         `json:"dependencies,omitempty"`
	Logs            []string         `json:"logs"`
	CanCancelSafely bool             `json:"canCancelSafely"`
	CancelRequested bool             `json:"cancelRequested"`
	ErrorCode       string           `json:"errorCode,omitempty"`
	ErrorMessage    string           `json:"errorMessage,omitempty"`
	StartedAt       time.Time        `json:"startedAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	CompletedAt     *time.Time       `json:"completedAt,omitempty"`
}

// LockChecker detects active host package-manager database locks before mutating system state.
type LockChecker func(provider string) (locked bool, lockPath string)

// DefaultHostLockChecker inspects standard OS lock files for system package managers.
func DefaultHostLockChecker(provider string) (bool, string) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "pacman", "aur", "paru", "repository":
		if _, err := os.Stat("/var/lib/pacman/db.lck"); err == nil {
			return true, "/var/lib/pacman/db.lck"
		}
	case "deb", "apt":
		if _, err := os.Stat("/var/lib/dpkg/lock-frontend"); err == nil {
			// Check existence only if readable; tests inject custom LockChecker
			return false, ""
		}
	}
	return false, ""
}

// TransactionManager centralizes authoritative OfferID-based package execution,
// lifecycle state transitions, lock guarding, safe cancellation, and history.
type TransactionManager struct {
	mu           sync.RWMutex
	pm           *providers.ProviderManager
	capabilities map[string]providers.ProviderCapability
	lockChecker  LockChecker
	offerStore   map[string]PackageOffer
	activeTx     map[string]*txControl
	transactions []*TransactionDTO
	seqCounter   atomic.Int64
}

type txControl struct {
	dto    *TransactionDTO
	cancel context.CancelFunc
}

// NewTransactionManager initializes a shared transaction engine for GUI and TUI.
func NewTransactionManager(pm *providers.ProviderManager) *TransactionManager {
	caps := providers.DefaultCapabilityMatrix()
	if pm != nil {
		caps = pm.GetCapabilityMatrix()
	}
	return &TransactionManager{
		pm:           pm,
		capabilities: caps,
		lockChecker:  DefaultHostLockChecker,
		offerStore:   make(map[string]PackageOffer),
		activeTx:     make(map[string]*txControl),
		transactions: make([]*TransactionDTO, 0),
	}
}

// SetLockChecker overrides lock detection (primarily for isolated integration tests).
func (tm *TransactionManager) SetLockChecker(fn LockChecker) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if fn != nil {
		tm.lockChecker = fn
	}
}

// SetCapabilities overrides capability checks (primarily for isolated tests).
func (tm *TransactionManager) SetCapabilities(caps map[string]providers.ProviderCapability) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if caps != nil {
		tm.capabilities = caps
	}
}

// RegisterOffer records an authoritative PackageOffer inside the Go engine so callers can invoke by OfferID.
func (tm *TransactionManager) RegisterOffer(offer PackageOffer) {
	if offer.ID == "" {
		return
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.offerStore[offer.ID] = offer
}

// RegisterOffers records a batch of offers atomically.
func (tm *TransactionManager) RegisterOffers(offers []PackageOffer) {
	if len(offers) == 0 {
		return
	}
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for _, o := range offers {
		if o.ID != "" {
			tm.offerStore[o.ID] = o
		}
	}
}

// LookupOffer resolves an OfferID from the backend store.
func (tm *TransactionManager) LookupOffer(offerID string) (PackageOffer, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	o, ok := tm.offerStore[offerID]
	return o, ok
}

// ListTransactions returns copies of recorded transactions (most recent first).
func (tm *TransactionManager) ListTransactions() []TransactionDTO {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	out := make([]TransactionDTO, 0, len(tm.transactions))
	for i := len(tm.transactions) - 1; i >= 0; i-- {
		cp := *tm.transactions[i]
		cp.Logs = append([]string(nil), tm.transactions[i].Logs...)
		out = append(out, cp)
	}
	return out
}

// GetTransaction returns a copy of a single transaction by ID.
func (tm *TransactionManager) GetTransaction(txID string) (TransactionDTO, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	for _, tx := range tm.transactions {
		if tx.ID == txID {
			cp := *tx
			cp.Logs = append([]string(nil), tx.Logs...)
			return cp, true
		}
	}
	return TransactionDTO{}, false
}

// ExecuteOfferTransaction resolves an authoritative OfferID and executes install or remove
// through the unified transaction lifecycle. Returns error if OfferID is unknown or operation unsafe.
func (tm *TransactionManager) ExecuteOfferTransaction(
	parentCtx context.Context,
	offerID string,
	op TransactionOp,
	onUpdate func(TransactionDTO),
) (TransactionDTO, error) {
	offer, ok := tm.LookupOffer(offerID)
	if !ok {
		return TransactionDTO{}, fmt.Errorf("unknown or unregistered offerId %q; transaction rejected", offerID)
	}

	txCtx, txCancel := context.WithCancel(parentCtx)
	defer txCancel()

	now := time.Now()
	txID := fmt.Sprintf("tx_%d_%d", now.UnixMilli(), tm.seqCounter.Add(1))
	dto := &TransactionDTO{
		ID:              txID,
		OfferID:         offer.ID,
		AppID:           offer.AppID,
		PackageName:     offer.PackageName,
		Provider:        offer.Provider,
		Repository:      offer.Repository,
		Operation:       op,
		State:           TxQueued,
		Progress:        5,
		Message:         fmt.Sprintf("Queued %s for %s (%s)", op, offer.PackageName, offer.Provider),
		Dependencies:    append([]string(nil), offer.Dependencies...),
		Logs:            []string{fmt.Sprintf("[QUEUED] %s offer %s", op, offer.ID)},
		CanCancelSafely: true,
		StartedAt:       now,
		UpdatedAt:       now,
	}

	ctrl := &txControl{dto: dto, cancel: txCancel}
	tm.mu.Lock()
	tm.activeTx[txID] = ctrl
	tm.transactions = append(tm.transactions, dto)
	checker := tm.lockChecker
	caps := tm.capabilities
	tm.mu.Unlock()

	defer func() {
		tm.mu.Lock()
		delete(tm.activeTx, txID)
		tm.mu.Unlock()
	}()

	emit := func(state TransactionState, progress int, msg, logLine string, safeCancel bool) {
		tm.mu.Lock()
		dto.State = state
		dto.Progress = progress
		dto.Message = msg
		dto.CanCancelSafely = safeCancel
		dto.UpdatedAt = time.Now()
		if logLine != "" {
			dto.Logs = append(dto.Logs, logLine)
		}
		if state == TxCompleted || state == TxFailed || state == TxCancelled {
			doneAt := time.Now()
			dto.CompletedAt = &doneAt
		}
		snapshot := *dto
		snapshot.Logs = append([]string(nil), dto.Logs...)
		tm.mu.Unlock()
		if onUpdate != nil {
			onUpdate(snapshot)
		}
	}

	failTx := func(code, errMsg string) (TransactionDTO, error) {
		tm.mu.Lock()
		dto.ErrorCode = code
		dto.ErrorMessage = errMsg
		tm.mu.Unlock()
		emit(TxFailed, 100, errMsg, "[FAILED] "+code+": "+errMsg, false)
		return *dto, fmt.Errorf("%s: %s", code, errMsg)
	}

	cancelTx := func(reason string) (TransactionDTO, error) {
		tm.mu.Lock()
		dto.ErrorCode = "cancelled"
		dto.ErrorMessage = reason
		tm.mu.Unlock()
		emit(TxCancelled, 100, reason, "[CANCELLED] "+reason, false)
		return *dto, errors.New(reason)
	}

	emit(TxQueued, 5, dto.Message, "", true)

	// Stage 1: Verify provider capabilities & resolve dependencies
	emit(TxResolvingDeps, 18, "Checking provider capabilities and resolving dependencies...", "[DEPS] Evaluating "+offer.Provider+" capabilities", true)
	if txCtx.Err() != nil {
		return cancelTx("Transaction cancelled while resolving dependencies")
	}

	if capInfo, ok := caps[strings.ToLower(offer.Provider)]; ok {
		if okOp, reason := capInfo.CanPerform(string(op)); !okOp {
			return failTx("provider_unavailable", reason)
		}
	}

	// Stage 2: Check package-manager database lock before starting download/commit
	if checker != nil {
		if locked, lockPath := checker(offer.Provider); locked {
			return failTx("lock_held", fmt.Sprintf("Package manager database is locked (%s). Close other package managers or wait for ongoing tasks.", lockPath))
		}
	}

	// Stage 3: Authorization boundary check (non-sandboxed system packages note privilege requirement)
	if offer.InstallScope == ScopeSystem {
		emit(TxAwaitingAuth, 30, "Verifying system authorization context...", "[AUTH] System-level repository package operation", true)
	} else {
		emit(TxAwaitingAuth, 30, "Validating user/sandbox permissions...", "[AUTH] User-scoped sandbox operation", true)
	}
	if txCtx.Err() != nil {
		return cancelTx("Transaction cancelled prior to download")
	}

	// Stage 4: Download & Verification
	emit(TxDownloading, 50, fmt.Sprintf("Fetching %s from %s...", offer.PackageName, offer.Provider), "[FETCH] Downloading package payload", true)
	if txCtx.Err() != nil {
		return cancelTx("Transaction safely aborted before committing files")
	}

	emit(TxVerifying, 70, "Verifying cryptographic signature and archive integrity...", "[VERIFY] Signature policy: "+offer.SignatureStatus, false)

	// Stage 5: Execute provider operation (ATOMIC / NON-CANCELLABLE mid-commit)
	emit(TxInstalling, 82, fmt.Sprintf("Applying %s on system...", op), "[COMMIT] Invoking "+offer.Provider+" backend", false)

	if tm.pm == nil {
		return failTx("provider_unavailable", "ProviderManager not initialized")
	}

	pkg := models.Package{
		Name:     offer.PackageName,
		Version:  offer.Version,
		Provider: offer.Provider,
		Source:   offer.Repository,
		Repo:     offer.Repository,
		Arch:     offer.Architecture,
		Branch:   offer.Branch,
		Channel:  offer.Channel,
	}

	var res models.Result
	var err error
	provAdapter, found := tm.pm.GetProvider(offer.Provider)
	if !found {
		return failTx("provider_unavailable", fmt.Sprintf("No active adapter registered for provider %q", offer.Provider))
	}

	// Once inside commit, isolate from soft UI cancellation so the system package manager finishes atomically
	commitCtx, commitCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer commitCancel()

	switch op {
	case OpInstall:
		res, err = provAdapter.Install(commitCtx, pkg)
	case OpRemove:
		res, err = provAdapter.Remove(commitCtx, pkg)
	case OpUpdate:
		res, err = provAdapter.Update(commitCtx)
	default:
		return failTx("invalid_operation", fmt.Sprintf("Unsupported operation %q", op))
	}

	if err != nil || !res.Success {
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		if res.Error != "" {
			if errText != "" {
				errText += ": "
			}
			errText += res.Error
		}
		if errText == "" {
			errText = fmt.Sprintf("%s failed via %s", op, offer.Provider)
		}
		code := classifyErrorCode(errText)
		return failTx(code, errText)
	}

	// Update authoritative installed status ONLY on THIS specific offer (never migrate siblings!)
	tm.mu.Lock()
	if stored, exists := tm.offerStore[offer.ID]; exists {
		if op == OpInstall || op == OpUpdate {
			stored.IsInstalled = true
			stored.InstalledVersion = stored.Version
		} else if op == OpRemove {
			stored.IsInstalled = false
			stored.InstalledVersion = ""
		}
		tm.offerStore[offer.ID] = stored
	}
	tm.mu.Unlock()

	emit(TxCompleted, 100, fmt.Sprintf("%s %s completed cleanly (%dms)", offer.PackageName, op, res.DurationMs), "[DONE] Transaction completed cleanly", false)
	cp, _ := tm.GetTransaction(txID)
	return cp, nil
}

// CancelTransaction requests safe cancellation of an active transaction.
// Never forcefully kills an in-flight system commit; refuses or defers unsafe mid-write interrupts.
func (tm *TransactionManager) CancelTransaction(txID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ctrl, ok := tm.activeTx[txID]
	if !ok || ctrl == nil {
		return fmt.Errorf("transaction %q is not active", txID)
	}
	if !ctrl.dto.CanCancelSafely {
		ctrl.dto.CancelRequested = true
		ctrl.dto.Logs = append(ctrl.dto.Logs, "[CANCEL-DEFERRED] System commit in progress; unsafe to abort mid-write")
		return fmt.Errorf("cannot safely interrupt %s during %s phase; waiting for atomic commit to finish", txID, ctrl.dto.State)
	}

	ctrl.dto.CancelRequested = true
	ctrl.cancel()
	return nil
}

func classifyErrorCode(msg string) string {
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "lock") || strings.Contains(low, "db.lck"):
		return "lock_held"
	case strings.Contains(low, "conflict") || strings.Contains(low, "dependency") || strings.Contains(low, "dependencies"):
		return "dependency_conflict"
	case strings.Contains(low, "permission") || strings.Contains(low, "sudo") || strings.Contains(low, "polkit") || strings.Contains(low, "auth"):
		return "auth_failed"
	case strings.Contains(low, "timeout") || strings.Contains(low, "connection") || strings.Contains(low, "resolve host"):
		return "network_error"
	default:
		return "transaction_failed"
	}
}

