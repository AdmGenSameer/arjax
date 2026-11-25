package updates

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/transaction"
)

const (
	PacmanFullUpgradeOfferID = "pacman:system:system-update"
	AURHelperUpgradeOfferID  = "aur:aur:aur-update"
)

// Manager orchestrates concurrent multi-provider update discovery, explicit preview/approval,
// strict Pacman-before-AUR ordering, pre-execution offer revalidation, and scoped Flatpak execution.
type Manager struct {
	mu          sync.RWMutex
	providers   []UpdateProvider
	txMgr       *transaction.Manager
	lastSummary *UpdateSummary
	previews    map[string]TransactionPreview
	previewSeq  atomic.Int64
}

func NewManager(txMgr *transaction.Manager, customProviders ...UpdateProvider) *Manager {
	provs := customProviders
	if len(provs) == 0 {
		provs = []UpdateProvider{
			NewPacmanUpdateProvider(txMgr),
			NewFlatpakUpdateProvider(txMgr),
			NewAURUpdateProvider(txMgr),
			NewSnapUpdateProvider(txMgr),
		}
	}
	return &Manager{
		providers: provs,
		txMgr:     txMgr,
		previews:  make(map[string]TransactionPreview),
	}
}

func (m *Manager) providerMap() map[string]UpdateProvider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]UpdateProvider, len(m.providers))
	for _, p := range m.providers {
		out[p.Name()] = p
	}
	return out
}

// CheckAllUpdates queries every enabled provider concurrently without aborting when one fails.
func (m *Manager) CheckAllUpdates(ctx context.Context) UpdateSummary {
	summary := UpdateSummary{
		ByProvider:     make(map[string]int),
		Updates:        make([]AvailableUpdate, 0),
		ProviderErrors: make(map[string]string),
		Notes:          make([]string, 0),
		CheckedAt:      time.Now().UTC(),
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	m.mu.RLock()
	provs := append([]UpdateProvider(nil), m.providers...)
	m.mu.RUnlock()

	for _, up := range provs {
		wg.Add(1)
		go func(p UpdateProvider) {
			defer wg.Done()
			subCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()

			list, err := p.CheckUpdates(subCtx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				summary.ProviderErrors[p.Name()] = err.Error()
				return
			}
			for _, item := range list {
				summary.Updates = append(summary.Updates, item)
				summary.ByProvider[p.Name()]++
				if item.IsSecurity {
					summary.SecurityCount++
				}
				if strings.EqualFold(item.Provider, "pacman") {
					summary.RequiresFullSysUpg = true
				}
			}
		}(up)
	}
	wg.Wait()

	sort.Slice(summary.Updates, func(i, j int) bool {
		if summary.Updates[i].IsSecurity != summary.Updates[j].IsSecurity {
			return summary.Updates[i].IsSecurity
		}
		if summary.Updates[i].Provider != summary.Updates[j].Provider {
			return summary.Updates[i].Provider < summary.Updates[j].Provider
		}
		if summary.Updates[i].PackageName != summary.Updates[j].PackageName {
			return summary.Updates[i].PackageName < summary.Updates[j].PackageName
		}
		return summary.Updates[i].InstallScope < summary.Updates[j].InstallScope
	})

	summary.TotalCount = len(summary.Updates)
	summary.ProviderCount = len(summary.ByProvider)

	if summary.RequiresFullSysUpg {
		summary.Notes = append(summary.Notes, "Arch Linux does not support partial upgrades: applying Pacman updates performs a full official system upgrade (pacman -Syu) before any AUR upgrades.")
	}
	if summary.ByProvider["snap"] > 0 {
		summary.Notes = append(summary.Notes, "Snap updates are a point-in-time snapshot; availability is revalidated before executing refreshes.")
	}

	if m.txMgr != nil {
		for _, item := range summary.Updates {
			m.txMgr.RegisterOfferDescriptor(transaction.OfferDescriptor{
				OfferID:      item.OfferID,
				AppID:        item.AppID,
				PackageName:  item.PackageName,
				Provider:     item.Provider,
				Repository:   item.Repository,
				Branch:       item.Branch,
				Scope:        item.InstallScope,
				Version:      item.AvailableVersion,
				InstalledVer: item.InstalledVersion,
				IsEligible:   true,
				IsInstalled:  true,
			})
		}
		if summary.ByProvider["pacman"] > 0 {
			m.txMgr.RegisterOfferDescriptor(transaction.OfferDescriptor{
				OfferID:     PacmanFullUpgradeOfferID,
				AppID:       "system-update",
				PackageName: "system-update",
				Provider:    "pacman",
				Repository:  "extra",
				Scope:       "system",
				IsEligible:  true,
			})
		}
		if summary.ByProvider["aur"] > 0 {
			m.txMgr.RegisterOfferDescriptor(transaction.OfferDescriptor{
				OfferID:     AURHelperUpgradeOfferID,
				AppID:       "aur-update",
				PackageName: "aur-update",
				Provider:    "aur",
				Repository:  "aur",
				Scope:       "user",
				IsEligible:  true,
			})
		}
	}

	m.mu.Lock()
	cp := summary
	m.lastSummary = &cp
	m.mu.Unlock()

	return summary
}

func (m *Manager) GetCachedSummary() (UpdateSummary, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.lastSummary == nil {
		return UpdateSummary{}, false
	}
	return *m.lastSummary, true
}

// PreviewUpdates resolves selected OfferIDs against fresh discovery data and constructs
// a TransactionPreview requiring explicit user confirmation before executing system upgrades.
func (m *Manager) PreviewUpdates(ctx context.Context, offerIDs []string) TransactionPreview {
	fresh := m.CheckAllUpdates(ctx)
	byID := make(map[string]AvailableUpdate, len(fresh.Updates))
	var pacmanUpdates []AvailableUpdate
	for _, u := range fresh.Updates {
		byID[u.OfferID] = u
		if strings.EqualFold(u.Provider, "pacman") {
			pacmanUpdates = append(pacmanUpdates, u)
		}
	}

	preview := TransactionPreview{
		PreviewID:               fmt.Sprintf("prev_%d_%d", time.Now().UnixMilli(), m.previewSeq.Add(1)),
		RequestedOfferIDs:       append([]string(nil), offerIDs...),
		EffectiveOfferIDs:       make([]string, 0, len(offerIDs)),
		PackagesToUpgrade:       make([]AvailableUpdate, 0),
		PackagesToInstall:       []string{},
		PackagesToRemove:        []string{},
		Warnings:                make([]string, 0),
		ProviderErrors:          fresh.ProviderErrors,
		RequiresExplicitApprove: true,
		CreatedAt:               time.Now().UTC(),
		ExpiresAt:               time.Now().UTC().Add(10 * time.Minute),
	}

	var hasPacmanSel, hasAURSel bool
	seenEff := make(map[string]bool)

	for _, id := range offerIDs {
		item, ok := byID[id]
		if !ok {
			preview.Warnings = append(preview.Warnings, fmt.Sprintf("Offer %s is no longer pending (already updated externally or stale).", id))
			continue
		}
		if strings.EqualFold(item.Provider, "pacman") {
			hasPacmanSel = true
		}
		if strings.EqualFold(item.Provider, "aur") {
			hasAURSel = true
			preview.IncludesAURBuild = true
		}
		if !seenEff[id] {
			seenEff[id] = true
			preview.EffectiveOfferIDs = append(preview.EffectiveOfferIDs, id)
			preview.PackagesToUpgrade = append(preview.PackagesToUpgrade, item)
		}
	}

	// Arch full-upgrade rule: selecting any Pacman update includes ALL pending Pacman repo updates
	if hasPacmanSel {
		preview.IncludesFullSysUpgrade = true
		for _, pu := range pacmanUpdates {
			if !seenEff[pu.OfferID] {
				seenEff[pu.OfferID] = true
				preview.EffectiveOfferIDs = append(preview.EffectiveOfferIDs, pu.OfferID)
				preview.PackagesToUpgrade = append(preview.PackagesToUpgrade, pu)
			}
		}
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("Full Arch upgrade (`pacman -Syu`) will upgrade all %d pending official repository packages together. Interactive confirmation protects against conflicting replacements.", len(pacmanUpdates)))
	}

	// Pacman-before-AUR rule: warn explicitly if AUR selected while Pacman repo updates are pending
	if hasAURSel && len(pacmanUpdates) > 0 && !hasPacmanSel {
		preview.RequiresOfficialFirst = true
		preview.Warnings = append(preview.Warnings, fmt.Sprintf("Blocked unless official repos update first: %d official Pacman updates are pending. Apply `pacman -Syu` before upgrading AUR packages to prevent soname breakage.", len(pacmanUpdates)))
	}

	if hasAURSel {
		preview.Warnings = append(preview.Warnings, "AUR upgrades invoke `paru -Sua` cleanly after official repos; review PKGBUILD diffs before confirming.")
	}

	preview.EstimatedDownloadSize = fmt.Sprintf("%d package(s) queued", len(preview.PackagesToUpgrade))

	m.mu.Lock()
	m.previews[preview.PreviewID] = preview
	m.mu.Unlock()

	return preview
}

// ExecuteApprovedUpdates verifies explicit approval & non-expired preview, then applies updates safely.
func (m *Manager) ExecuteApprovedUpdates(ctx context.Context, previewID string, approved bool, onProgress transaction.UpdateHandler) UpdateBatchResult {
	if !approved {
		return UpdateBatchResult{
			Note: "Transaction aborted: explicit user confirmation is required before upgrading packages.",
		}
	}

	m.mu.RLock()
	prev, ok := m.previews[previewID]
	m.mu.RUnlock()
	if !ok {
		return UpdateBatchResult{
			Failed: 1,
			Note:   fmt.Sprintf("Unknown or cleared previewId %q; re-generate transaction preview before applying updates.", previewID),
		}
	}
	if time.Now().UTC().After(prev.ExpiresAt) {
		return UpdateBatchResult{
			Failed: 1,
			Note:   "Transaction preview expired; refresh updates and generate a new preview.",
		}
	}

	return m.UpdateSelected(ctx, prev.EffectiveOfferIDs, onProgress)
}

// UpdateSelected executes selected OfferIDs with strict Arch safety:
// 1. Official Pacman updates ALWAYS run first as ONE full `pacman -Syu` transaction.
// 2. If AUR updates are selected while Pacman updates remain pending and unselected, AUR is BLOCKED (`official_updates_pending`).
// 3. When both Pacman and AUR are selected, AUR updates are RE-DISCOVERED (`paru -Qua`) after `pacman -Syu` succeeds before running `paru -Sua`.
// 4. Flatpak/Snap offers are REVALIDATED immediately before executing; already-updated offers finish as SkippedUpToDate cleanly.
func (m *Manager) UpdateSelected(ctx context.Context, offerIDs []string, onProgress transaction.UpdateHandler) UpdateBatchResult {
	res := UpdateBatchResult{
		TotalRequested: len(offerIDs),
		Results:        make([]UpdateItemOutcome, 0, len(offerIDs)),
		Note:           "Enforces Arch Pacman-before-AUR ordering, full pacman -Syu upgrades, pre-execution revalidation, and Flatpak scope isolation.",
	}

	provs := m.providerMap()
	m.mu.RLock()
	cached := m.lastSummary
	m.mu.RUnlock()

	lookupItem := func(id string) (AvailableUpdate, bool) {
		if cached != nil {
			for _, u := range cached.Updates {
				if u.OfferID == id {
					return u, true
				}
			}
		}
		return AvailableUpdate{}, false
	}

	var (
		hasPacman     bool
		pacmanOffers  []AvailableUpdate
		hasAUR        bool
		aurOffers     []AvailableUpdate
		otherOfferIDs []string
	)

	for _, id := range offerIDs {
		item, found := lookupItem(id)
		prov := strings.ToLower(item.Provider)
		if !found && strings.HasPrefix(id, "pacman:") {
			prov = "pacman"
		} else if !found && strings.HasPrefix(id, "aur:") {
			prov = "aur"
		}

		switch prov {
		case "pacman":
			hasPacman = true
			if found {
				pacmanOffers = append(pacmanOffers, item)
			}
		case "aur":
			hasAUR = true
			if found {
				aurOffers = append(aurOffers, item)
			}
		default:
			otherOfferIDs = append(otherOfferIDs, id)
		}
	}

	// Step A: Execute full Pacman upgrade FIRST if selected
	pacmanSucceeded := true
	if hasPacman {
		execID := PacmanFullUpgradeOfferID
		var okTx bool
		var txID, errMsg, errCode string

		if m.txMgr != nil {
			tx, err := m.txMgr.ExecuteOfferTransaction(ctx, execID, transaction.OpUpdate, onProgress)
			txID = tx.ID
			errCode = tx.ErrorCode
			okTx = err == nil && tx.State == transaction.TxCompleted
			if !okTx {
				errMsg = tx.ErrorMessage
				if errMsg == "" && err != nil {
					errMsg = err.Error()
				}
			}
		} else if p, ok := provs["pacman"]; ok {
			err := p.Update(ctx, execID)
			okTx = err == nil
			if err != nil {
				errMsg = err.Error()
				errCode = "transaction_failed"
			}
		} else {
			errCode = "provider_unavailable"
			errMsg = "pacman update provider not registered"
		}

		pacmanSucceeded = okTx
		res.Results = append(res.Results, UpdateItemOutcome{
			OfferID:       execID,
			PackageName:   "system-update (pacman -Syu)",
			Provider:      "pacman",
			InstallScope:  "system",
			Success:       okTx,
			ErrorCode:     errCode,
			TransactionID: txID,
			ErrorMessage:  errMsg,
		})
		countAffected := len(pacmanOffers)
		if countAffected == 0 {
			countAffected = 1
		}
		if okTx {
			res.Succeeded += countAffected
		} else {
			res.Failed += countAffected
		}
	}

	// Step B: Pacman-before-AUR Safety Gate & Re-discovery
	if hasAUR {
		countAUR := len(aurOffers)
		if countAUR == 0 {
			countAUR = 1
		}

		// Gate 1: If Pacman was NOT selected, check live pending official updates on host
		if !hasPacman {
			if pacProv, ok := provs["pacman"]; ok {
				if pendingOfficial, err := pacProv.CheckUpdates(ctx); err == nil && len(pendingOfficial) > 0 {
					msg := fmt.Sprintf("Blocked AUR upgrade: %d official Pacman updates are pending. Apply full `pacman -Syu` first before upgrading AUR packages.", len(pendingOfficial))
					res.Results = append(res.Results, UpdateItemOutcome{
						OfferID:      AURHelperUpgradeOfferID,
						PackageName:  "aur-update (paru -Sua)",
						Provider:     "aur",
						InstallScope: "user",
						Success:      false,
						ErrorCode:    "official_updates_pending",
						ErrorMessage: msg,
					})
					res.Failed += countAUR
					hasAUR = false
				}
			}
		} else if !pacmanSucceeded {
			// Gate 2: Official upgrade failed -> DO NOT proceed with AUR rebuild against broken/unupgraded system!
			msg := "Blocked AUR upgrade: official `pacman -Syu` failed or was cancelled; AUR upgrades require an up-to-date official system."
			res.Results = append(res.Results, UpdateItemOutcome{
				OfferID:      AURHelperUpgradeOfferID,
				PackageName:  "aur-update (paru -Sua)",
				Provider:     "aur",
				InstallScope: "user",
				Success:      false,
				ErrorCode:    "official_upgrade_failed",
				ErrorMessage: msg,
			})
			res.Failed += countAUR
			hasAUR = false
		}

		// If gate passed, re-discover AUR updates after official repo upgrade, then run paru -Sua
		if hasAUR {
			if aurProv, ok := provs["aur"]; ok {
				_, _ = aurProv.CheckUpdates(ctx)
			}

			execID := AURHelperUpgradeOfferID
			var okTx bool
			var txID, errMsg, errCode string

			if m.txMgr != nil {
				tx, err := m.txMgr.ExecuteOfferTransaction(ctx, execID, transaction.OpUpdate, onProgress)
				txID = tx.ID
				errCode = tx.ErrorCode
				okTx = err == nil && tx.State == transaction.TxCompleted
				if !okTx {
					errMsg = tx.ErrorMessage
					if errMsg == "" && err != nil {
						errMsg = err.Error()
					}
				}
			} else if p, ok := provs["aur"]; ok {
				err := p.Update(ctx, execID)
				okTx = err == nil
				if err != nil {
					errMsg = err.Error()
					errCode = "transaction_failed"
				}
			} else {
				errCode = "provider_unavailable"
				errMsg = "aur update provider not registered"
			}

			res.Results = append(res.Results, UpdateItemOutcome{
				OfferID:       execID,
				PackageName:   "aur-update (paru -Sua)",
				Provider:      "aur",
				InstallScope:  "user",
				Success:       okTx,
				ErrorCode:     errCode,
				TransactionID: txID,
				ErrorMessage:  errMsg,
			})
			if okTx {
				res.Succeeded += countAUR
			} else {
				res.Failed += countAUR
			}
		}
	}

	// Step C: Execute Flatpak (--user/--system scoped) and Snap updates with pre-execution revalidation
	revalidatedCache := make(map[string]map[string]bool)
	isStillPending := func(provName, offerID string) bool {
		if provName != "snap" && provName != "flatpak" {
			return true
		}
		pendingSet, checked := revalidatedCache[provName]
		if !checked {
			p, ok := provs[provName]
			if !ok {
				return true
			}
			freshList, err := p.CheckUpdates(ctx)
			if err != nil {
				// Let execution handle explicit error if check fails
				revalidatedCache[provName] = nil
				return true
			}
			pendingSet = make(map[string]bool, len(freshList))
			for _, u := range freshList {
				pendingSet[u.OfferID] = true
			}
			revalidatedCache[provName] = pendingSet
		}
		if pendingSet == nil {
			return true
		}
		return pendingSet[offerID]
	}

	for _, id := range otherOfferIDs {
		item, found := lookupItem(id)
		outcome := UpdateItemOutcome{
			OfferID:      id,
			PackageName:  item.PackageName,
			Provider:     item.Provider,
			InstallScope: item.InstallScope,
		}

		// Structured pre-execution revalidation: avoid failing if snapd/flatpak updated externally
		if found && !isStillPending(strings.ToLower(item.Provider), id) {
			outcome.Success = true
			outcome.SkippedUpToDate = true
			outcome.ErrorMessage = "Already up to date upon pre-execution revalidation (no longer pending)."
			res.Succeeded++
			res.Results = append(res.Results, outcome)
			continue
		}

		if m.txMgr != nil {
			tx, err := m.txMgr.ExecuteOfferTransaction(ctx, id, transaction.OpUpdate, onProgress)
			outcome.TransactionID = tx.ID
			outcome.ErrorCode = tx.ErrorCode
			if outcome.PackageName == "" {
				outcome.PackageName = tx.PackageName
				outcome.Provider = tx.Provider
				outcome.InstallScope = tx.InstallScope
			}
			if err == nil && tx.State == transaction.TxCompleted {
				outcome.Success = true
				res.Succeeded++
			} else {
				outcome.Success = false
				outcome.ErrorMessage = tx.ErrorMessage
				if outcome.ErrorMessage == "" && err != nil {
					outcome.ErrorMessage = err.Error()
				}
				res.Failed++
			}
		} else if found {
			if p, ok := provs[item.Provider]; ok {
				if err := p.Update(ctx, id); err == nil {
					outcome.Success = true
					res.Succeeded++
				} else {
					outcome.Success = false
					outcome.ErrorMessage = err.Error()
					res.Failed++
				}
			} else {
				outcome.ErrorCode = "provider_unavailable"
				outcome.ErrorMessage = fmt.Sprintf("no update provider registered for %q", item.Provider)
				res.Failed++
			}
		} else {
			outcome.ErrorCode = "unknown_offer"
			outcome.ErrorMessage = fmt.Sprintf("unknown update offerId %q", id)
			res.Failed++
		}

		res.Results = append(res.Results, outcome)
	}

	return res
}

func (m *Manager) UpdateAll(ctx context.Context, onProgress transaction.UpdateHandler) UpdateBatchResult {
	summary, ok := m.GetCachedSummary()
	if !ok {
		summary = m.CheckAllUpdates(ctx)
	}
	ids := make([]string, 0, len(summary.Updates))
	for _, u := range summary.Updates {
		ids = append(ids, u.OfferID)
	}
	return m.UpdateSelected(ctx, ids, onProgress)
}
