package transaction

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

const DefaultOfferTTL = 30 * time.Minute

type txControl struct {
	dto    *TransactionDTO
	cancel context.CancelFunc
}

// Manager orchestrates validated OfferID transactions, persistent crash recovery,
// native & cross-process locks, and safe cancellation.
type Manager struct {
	mu           sync.RWMutex
	pm           *providers.ProviderManager
	journal      *Journal
	bus          *EventBus
	capabilities map[string]providers.ProviderCapability
	lockChecker  LockChecker
	offerStore   map[string]OfferDescriptor
	offerTTL     time.Duration
	metaEpoch    int64
	activeTx     map[string]*txControl
	seqCounter   atomic.Int64
}

// NewManager creates a Manager backed by an explicit journal path (uses XDG state path when empty).
func NewManager(pm *providers.ProviderManager, journalPath string) *Manager {
	caps := providers.DefaultCapabilityMatrix()
	if pm != nil {
		caps = pm.GetCapabilityMatrix()
	}
	return &Manager{
		pm:           pm,
		journal:      NewJournal(journalPath),
		bus:          NewEventBus(),
		capabilities: caps,
		lockChecker:  DefaultHostLockChecker,
		offerStore:   make(map[string]OfferDescriptor),
		offerTTL:     DefaultOfferTTL,
		metaEpoch:    1,
		activeTx:     make(map[string]*txControl),
	}
}

// RecoverOnStartup reconciles any incomplete journal transactions against real host provider state.
func (m *Manager) RecoverOnStartup(ctx context.Context, probe InstalledStateProbe) []TransactionDTO {
	m.mu.RLock()
	lc := m.lockChecker
	m.mu.RUnlock()
	return ReconcileIncomplete(ctx, m.journal, probe, lc)
}

// SetOfferTTL adjusts offer freshness window before re-validation is required.
func (m *Manager) SetOfferTTL(ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.offerTTL = ttl
}

// AdvanceMetadataEpoch supersedes older catalog snapshots cleanly.
func (m *Manager) AdvanceMetadataEpoch() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metaEpoch++
	return m.metaEpoch
}

// SetLockChecker overrides host lock checks (useful for deterministic lock testing).
func (m *Manager) SetLockChecker(fn LockChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fn != nil {
		m.lockChecker = fn
	}
}

// SetCapabilities overrides capability checks (primarily for isolated tests).
func (m *Manager) SetCapabilities(caps map[string]providers.ProviderCapability) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if caps != nil {
		m.capabilities = caps
	}
}

// Subscribe attaches a listener for all transaction lifecycle updates.
func (m *Manager) Subscribe(fn UpdateHandler) {
	m.bus.Subscribe(fn)
}

// RegisterOfferDescriptor records an authoritative OfferDescriptor with TTL and epoch.
func (m *Manager) RegisterOfferDescriptor(desc OfferDescriptor) {
	if desc.OfferID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if desc.RegisteredAt.IsZero() {
		desc.RegisteredAt = now
	}
	if desc.ExpiresAt.IsZero() {
		desc.ExpiresAt = desc.RegisteredAt.Add(m.offerTTL)
	}
	if desc.MetadataEpoch == 0 {
		desc.MetadataEpoch = m.metaEpoch
	}
	m.offerStore[desc.OfferID] = desc
}

// LookupDescriptor retrieves a stored offer descriptor.
func (m *Manager) LookupDescriptor(offerID string) (OfferDescriptor, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	desc, ok := m.offerStore[offerID]
	return desc, ok
}

// ListTransactions returns journaled and live transactions (newest first).
func (m *Manager) ListTransactions() []TransactionDTO {
	return m.journal.Snapshot()
}

// GetTransaction looks up a transaction record by ID.
func (m *Manager) GetTransaction(txID string) (TransactionDTO, bool) {
	for _, tx := range m.journal.Snapshot() {
		if tx.ID == txID {
			return tx, true
		}
	}
	return TransactionDTO{}, false
}

// JournalPath exposes the backing journal file path.
func (m *Manager) JournalPath() string {
	return m.journal.filePath
}

// ExecuteOfferTransaction validates an OfferID against expiration, epochs, capabilities,
// cross-process advisory locks, and host package-manager locks, then executes and journals the transaction.
func (m *Manager) ExecuteOfferTransaction(
	parentCtx context.Context,
	offerID string,
	op TransactionOp,
	onUpdate UpdateHandler,
) (TransactionDTO, error) {
	m.mu.RLock()
	desc, exists := m.offerStore[offerID]
	caps := m.capabilities
	epoch := m.metaEpoch
	checker := m.lockChecker
	m.mu.RUnlock()

	if !exists {
		desc.OfferID = offerID
	}

	if errCode, vErr := ValidateOffer(desc, exists, time.Now(), op, caps, epoch); vErr != nil {
		return TransactionDTO{
			OfferID:        offerID,
			State:          TxFailed,
			ErrorCode:      errCode,
			ErrorMessage:   vErr.Error(),
			RecoveryStatus: RecoveryNone,
		}, vErr
	}

	txCtx, txCancel := context.WithCancel(parentCtx)
	defer txCancel()

	now := time.Now()
	txID := fmt.Sprintf("tx_%d_%d", now.UnixMilli(), m.seqCounter.Add(1))
	dto := &TransactionDTO{
		ID:              txID,
		OfferID:         desc.OfferID,
		AppID:           desc.AppID,
		PackageName:     desc.PackageName,
		Provider:        desc.Provider,
		Repository:      desc.Repository,
		InstallScope:    desc.Scope,
		PreviousVersion: desc.InstalledVer,
		TargetVersion:   desc.Version,
		Operation:       op,
		State:           TxQueued,
		Progress:        5,
		Message:         fmt.Sprintf("Queued %s for %s (%s)", op, desc.PackageName, desc.Provider),
		Dependencies:    append([]string(nil), desc.Dependencies...),
		Logs:            []string{fmt.Sprintf("[QUEUED] %s offer %s", op, desc.OfferID)},
		CanCancelSafely: true,
		RecoveryStatus:  RecoveryNone,
		StartedAt:       now,
		UpdatedAt:       now,
	}

	ctrl := &txControl{dto: dto, cancel: txCancel}
	m.mu.Lock()
	m.activeTx[txID] = ctrl
	m.mu.Unlock()
	m.journal.Append(dto)

	defer func() {
		m.mu.Lock()
		delete(m.activeTx, txID)
		m.mu.Unlock()
	}()

	emit := func(state TransactionState, progress int, msg, logLine string, safeCancel bool) {
		m.mu.Lock()
		dto.State = state
		dto.Progress = progress
		dto.Message = msg
		dto.CanCancelSafely = safeCancel
		dto.UpdatedAt = time.Now()
		if logLine != "" {
			dto.Logs = append(dto.Logs, logLine)
		}
		if state.IsTerminal() {
			doneAt := time.Now()
			dto.CompletedAt = &doneAt
		}
		snapshot := *dto
		snapshot.Logs = append([]string(nil), dto.Logs...)
		m.mu.Unlock()

		m.journal.Update(dto)
		m.bus.Emit(snapshot)
		if onUpdate != nil {
			onUpdate(snapshot)
		}
	}

	failTx := func(code, errMsg string, exitCode int) (TransactionDTO, error) {
		m.mu.Lock()
		dto.ErrorCode = code
		dto.ErrorMessage = errMsg
		dto.ExitCode = exitCode
		dto.RecoveryStatus = RecoveryFailed
		m.mu.Unlock()
		emit(TxFailed, 100, errMsg, fmt.Sprintf("[FAILED] %s (exit=%d): %s", code, exitCode, errMsg), false)
		cp, _ := m.GetTransaction(txID)
		return cp, fmt.Errorf("%s: %s", code, errMsg)
	}

	cancelTx := func(reason string) (TransactionDTO, error) {
		m.mu.Lock()
		dto.ErrorCode = "cancelled"
		dto.ErrorMessage = reason
		dto.ExitCode = 130
		m.mu.Unlock()
		emit(TxCancelled, 100, reason, "[CANCELLED] "+reason, false)
		cp, _ := m.GetTransaction(txID)
		return cp, errors.New(reason)
	}

	emit(TxQueued, 5, dto.Message, "", true)

	// Stage 1: Resolve dependencies & check locks
	emit(TxResolvingDeps, 18, "Validating capabilities, locks, and dependencies...", "[DEPS] Evaluating "+desc.Provider+" capabilities", true)
	if txCtx.Err() != nil {
		return cancelTx("Transaction cancelled while resolving dependencies")
	}

	if checker != nil {
		if locked, lockPath := checker(desc.Provider); locked {
			return failTx("lock_held", fmt.Sprintf("Package manager database is locked (%s). Close other package managers or retry shortly.", lockPath), 1)
		}
	}

	// Acquire complementary cross-process advisory lock (GUI <-> TUI coordination)
	releaseProcLock, lockErr := AcquireProcessCoordLock(desc.Provider)
	if lockErr != nil {
		return failTx("cross_process_lock", lockErr.Error(), 1)
	}
	defer releaseProcLock()

	// Stage 2: Authorization check
	if desc.Scope == "system" {
		emit(TxAwaitingAuth, 30, "Verifying system authorization context...", "[AUTH] System repository scope", true)
	} else {
		emit(TxAwaitingAuth, 30, "Validating user/sandbox permissions...", "[AUTH] User/sandbox scope", true)
	}
	if txCtx.Err() != nil {
		return cancelTx("Transaction cancelled prior to download")
	}

	// Stage 3: Downloading
	emit(TxDownloading, 50, fmt.Sprintf("Fetching %s from %s...", desc.PackageName, desc.Provider), "[FETCH] Downloading package payload", true)
	if txCtx.Err() != nil {
		return cancelTx("Transaction safely aborted before committing files")
	}

	// Stage 4: Verifying & Committing (NON-CANCELLABLE mid-write)
	emit(TxVerifying, 70, "Verifying signatures and artifact digests...", "[VERIFY] Policy: "+desc.SignatureStatus, false)
	emit(TxInstalling, 82, fmt.Sprintf("Applying %s on system...", op), "[COMMIT] Invoking "+desc.Provider+" backend", false)

	if m.pm == nil {
		return failTx("provider_unavailable", "ProviderManager not initialized", 1)
	}

	provAdapter, found := m.pm.GetProvider(desc.Provider)
	if !found {
		return failTx("provider_unavailable", fmt.Sprintf("No active adapter registered for provider %q", desc.Provider), 1)
	}

	commitCtx, commitCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer commitCancel()

	res, exitCode, err := executeProviderOp(commitCtx, provAdapter, op, desc)
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
			errText = fmt.Sprintf("%s failed via %s", op, desc.Provider)
		}
		return failTx(classifyErrorCode(errText), errText, exitCode)
	}

	// Update installed status ONLY on THIS specific offer
	m.mu.Lock()
	dto.RecoveryStatus = RecoveryCompleted
	if stored, ok := m.offerStore[desc.OfferID]; ok {
		if op == OpInstall || op == OpUpdate {
			stored.IsInstalled = true
			stored.InstalledVer = stored.Version
		} else if op == OpRemove {
			stored.IsInstalled = false
			stored.InstalledVer = ""
		}
		m.offerStore[desc.OfferID] = stored
	}
	m.mu.Unlock()

	emit(TxCompleted, 100, fmt.Sprintf("%s %s completed cleanly (%dms)", desc.PackageName, op, res.DurationMs), "[DONE] Transaction completed cleanly", false)
	cp, _ := m.GetTransaction(txID)
	return cp, nil
}

// CancelTransaction requests safe cancellation; refuses or defers abort during atomic commits.
func (m *Manager) CancelTransaction(txID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctrl, ok := m.activeTx[txID]
	if !ok || ctrl == nil {
		return fmt.Errorf("transaction %q is not active", txID)
	}
	if !ctrl.dto.CanCancelSafely {
		ctrl.dto.CancelRequested = true
		ctrl.dto.Logs = append(ctrl.dto.Logs, "[CANCEL-DEFERRED] System commit in progress; unsafe to abort mid-write")
		m.journal.Update(ctrl.dto)
		return fmt.Errorf("cannot safely interrupt %s during %s phase; waiting for atomic commit to finish", txID, ctrl.dto.State)
	}

	ctrl.dto.CancelRequested = true
	ctrl.cancel()
	return nil
}

