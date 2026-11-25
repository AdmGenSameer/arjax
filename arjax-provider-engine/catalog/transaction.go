package catalog

import (
	"context"
	"sync"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/transaction"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

// Re-export unified transaction types from catalog/transaction so existing callers & tests remain cohesive.
type (
	TransactionState    = transaction.TransactionState
	TransactionOp       = transaction.TransactionOp
	RecoveryStatus      = transaction.RecoveryStatus
	TransactionDTO      = transaction.TransactionDTO
	LockChecker         = transaction.LockChecker
	InstalledStateProbe = transaction.InstalledStateProbe
)

const (
	TxQueued        = transaction.TxQueued
	TxResolvingDeps = transaction.TxResolvingDeps
	TxAwaitingAuth  = transaction.TxAwaitingAuth
	TxDownloading   = transaction.TxDownloading
	TxVerifying     = transaction.TxVerifying
	TxInstalling    = transaction.TxInstalling
	TxCompleted     = transaction.TxCompleted
	TxFailed        = transaction.TxFailed
	TxCancelled     = transaction.TxCancelled

	OpInstall = transaction.OpInstall
	OpRemove  = transaction.OpRemove
	OpUpdate  = transaction.OpUpdate

	RecoveryNone               = transaction.RecoveryNone
	RecoveryCompleted          = transaction.RecoveryCompleted
	RecoveryFailed             = transaction.RecoveryFailed
	RecoveryReconciledComplete = transaction.RecoveryReconciledComplete
	RecoveryReconciledFailed   = transaction.RecoveryReconciledFailed
	RecoveryRequired           = transaction.RecoveryRequired
	RecoveryOutcomeUnknown     = transaction.RecoveryOutcomeUnknown
)

// TransactionManager wraps catalog/transaction.Manager and maps catalog.PackageOffer records into OfferDescriptors.
type TransactionManager struct {
	mu     sync.RWMutex
	inner  *transaction.Manager
	offers map[string]PackageOffer
}

// NewTransactionManager initializes a persistent transaction engine at the default XDG journal path.
func NewTransactionManager(pm *providers.ProviderManager) *TransactionManager {
	return NewTransactionManagerWithJournal(pm, "")
}

// NewTransactionManagerWithJournal initializes a transaction engine with a custom journal path (useful in tests).
func NewTransactionManagerWithJournal(pm *providers.ProviderManager, journalPath string) *TransactionManager {
	return &TransactionManager{
		inner:  transaction.NewManager(pm, journalPath),
		offers: make(map[string]PackageOffer),
	}
}

// Inner returns the underlying persistent transaction.Manager.
func (tm *TransactionManager) Inner() *transaction.Manager {
	return tm.inner
}

// RecoverOnStartup reconciles interrupted journal entries against real host provider state.
func (tm *TransactionManager) RecoverOnStartup(ctx context.Context, probe InstalledStateProbe) []TransactionDTO {
	return tm.inner.RecoverOnStartup(ctx, probe)
}

func (tm *TransactionManager) SetOfferTTL(ttl time.Duration) {
	tm.inner.SetOfferTTL(ttl)
}

func (tm *TransactionManager) AdvanceMetadataEpoch() int64 {
	return tm.inner.AdvanceMetadataEpoch()
}

func (tm *TransactionManager) SetLockChecker(fn LockChecker) {
	tm.inner.SetLockChecker(fn)
}

func (tm *TransactionManager) SetCapabilities(caps map[string]providers.ProviderCapability) {
	tm.inner.SetCapabilities(caps)
}

func (tm *TransactionManager) RegisterOffer(offer PackageOffer) {
	if offer.ID == "" {
		return
	}
	tm.mu.Lock()
	tm.offers[offer.ID] = offer
	tm.mu.Unlock()
	scope := string(offer.InstallScope)
	if scope == "" {
		scope = string(offer.Scope)
	}
	tm.inner.RegisterOfferDescriptor(transaction.OfferDescriptor{
		OfferID:         offer.ID,
		AppID:           offer.AppID,
		PackageName:     offer.PackageName,
		Provider:        offer.Provider,
		Repository:      offer.Repository,
		Architecture:    offer.Architecture,
		Branch:          offer.Branch,
		Channel:         offer.Channel,
		Version:         offer.Version,
		Scope:           scope,
		SignatureStatus: offer.SignatureStatus,
		Dependencies:    offer.Dependencies,
		IsEligible:      offer.IsEligible,
		IsInstalled:     offer.IsInstalled,
		InstalledVer:    offer.InstalledVersion,
	})
}

func (tm *TransactionManager) RegisterOffers(offers []PackageOffer) {
	for _, o := range offers {
		tm.RegisterOffer(o)
	}
}

func (tm *TransactionManager) LookupOffer(offerID string) (PackageOffer, bool) {
	desc, ok := tm.inner.LookupDescriptor(offerID)
	if !ok {
		return PackageOffer{}, false
	}
	tm.mu.RLock()
	o, okMap := tm.offers[offerID]
	tm.mu.RUnlock()
	if okMap {
		o.IsInstalled = desc.IsInstalled
		o.InstalledVersion = desc.InstalledVer
		return o, true
	}
	return PackageOffer{
		ID:               desc.OfferID,
		AppID:            desc.AppID,
		PackageName:      desc.PackageName,
		Provider:         desc.Provider,
		Repository:       desc.Repository,
		Architecture:     desc.Architecture,
		Branch:           desc.Branch,
		Channel:          desc.Channel,
		Version:          desc.Version,
		IsEligible:       desc.IsEligible,
		IsInstalled:      desc.IsInstalled,
		InstalledVersion: desc.InstalledVer,
	}, true
}

func (tm *TransactionManager) ListTransactions() []TransactionDTO {
	return tm.inner.ListTransactions()
}

func (tm *TransactionManager) GetTransaction(txID string) (TransactionDTO, bool) {
	return tm.inner.GetTransaction(txID)
}

func (tm *TransactionManager) ExecuteOfferTransaction(
	parentCtx context.Context,
	offerID string,
	op TransactionOp,
	onUpdate func(TransactionDTO),
) (TransactionDTO, error) {
	return tm.inner.ExecuteOfferTransaction(parentCtx, offerID, op, onUpdate)
}

func (tm *TransactionManager) CancelTransaction(txID string) error {
	return tm.inner.CancelTransaction(txID)
}
