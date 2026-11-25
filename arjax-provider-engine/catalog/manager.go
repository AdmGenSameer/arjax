package catalog

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/updates"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/icons"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// CatalogManager is the unified core orchestrator shared across GUI and TUI for
// identity resolution, multi-source deduplication, trust/recommendation ranking,
// progressive search, and OfferID-based installation transactions.
type CatalogManager struct {
	pm               *providers.ProviderManager
	crawler          *appstream.Crawler
	iconResolver     *icons.IconResolverService
	deduper          *Deduplicator
	recommender      *RecommendationEngine
	progressive      *ProgressiveSearchEngine
	txMgr            *TransactionManager
	updateMgr        *updates.Manager
	updateSched      *updates.Scheduler
	installedFetcher func(ctx context.Context) map[string]string
	mu               sync.RWMutex
	cfg              CatalogConfig
	cache            map[string]*UnifiedApplication
}

// NewCatalogManager initializes the shared catalog engine and all subsystems.
func NewCatalogManager(
	pm *providers.ProviderManager,
	crawler *appstream.Crawler,
	iconResolver *icons.IconResolverService,
	cfg CatalogConfig,
	installedFetcher func(ctx context.Context) map[string]string,
) *CatalogManager {
	deduper := NewDeduplicator()
	recommender := NewRecommendationEngine(cfg)
	progressive := NewProgressiveSearchEngine(pm, crawler, deduper, recommender, installedFetcher)
	txMgr := NewTransactionManager(pm)
	updateMgr := updates.NewManager(txMgr.Inner())
	updateSched := updates.NewScheduler(updateMgr, 10*time.Minute)

	// Automatically register offers discovered via progressive search so InstallOffer(offerID) is always valid
	progressive.SetOfferObserver(txMgr.RegisterOffers)

	return &CatalogManager{
		pm:               pm,
		crawler:          crawler,
		iconResolver:     iconResolver,
		deduper:          deduper,
		recommender:      recommender,
		progressive:      progressive,
		txMgr:            txMgr,
		updateMgr:        updateMgr,
		updateSched:      updateSched,
		installedFetcher: installedFetcher,
		cfg:              cfg,
		cache:            make(map[string]*UnifiedApplication),
	}
}

// Updates returns the Update Center orchestrator.
func (cm *CatalogManager) Updates() *updates.Manager {
	return cm.updateMgr
}

// UpdateScheduler returns the background update check scheduler.
func (cm *CatalogManager) UpdateScheduler() *updates.Scheduler {
	return cm.updateSched
}

// RecoverTransactions reconciles any incomplete journal transactions left by a prior crash.
func (cm *CatalogManager) RecoverTransactions(ctx context.Context, probe InstalledStateProbe) []TransactionDTO {
	return cm.txMgr.RecoverOnStartup(ctx, probe)
}

// ProviderStatuses returns runtime host availability & service health separately from code support.
func (cm *CatalogManager) ProviderStatuses(ctx context.Context) []providers.ProviderStatus {
	if cm.pm != nil {
		return cm.pm.GetProviderStatuses(ctx)
	}
	return providers.ProbeProviderStatuses(ctx, providers.DefaultCapabilityMatrix())
}

// Transactions returns the unified transaction engine.
func (cm *CatalogManager) Transactions() *TransactionManager {
	return cm.txMgr
}

// Deduplicator returns the catalog deduplicator & identity engine.
func (cm *CatalogManager) Deduplicator() *Deduplicator {
	return cm.deduper
}

// Capabilities returns the explicit provider capability matrix.
func (cm *CatalogManager) Capabilities() []providers.ProviderCapability {
	if cm.pm != nil {
		return cm.pm.GetCapabilities()
	}
	return providers.GetCapabilitiesOrdered(providers.DefaultCapabilityMatrix())
}

// SetPolicy updates the recommendation policy dynamically and invalidates cached details rankings.
func (cm *CatalogManager) SetPolicy(policy RecommendationPolicy) {
	cm.mu.Lock()
	cm.cfg.Policy = policy
	cm.recommender = NewRecommendationEngine(cm.cfg)
	cm.cache = make(map[string]*UnifiedApplication)
	cm.mu.Unlock()
}

// GetPolicy returns the active recommendation policy.
func (cm *CatalogManager) GetPolicy() RecommendationPolicy {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cfg.Policy
}

// GetUnifiedAppDetails retrieves rich metadata, icon, trust, and all distinct package offers.
func (cm *CatalogManager) GetUnifiedAppDetails(ctx context.Context, appIdOrPkg string, preferredOfferID string) (*UnifiedApplication, error) {
	clean := strings.TrimSpace(appIdOrPkg)
	if clean == "" {
		return nil, fmt.Errorf("empty application identifier")
	}

	resolved := cm.deduper.ResolveIdentity(clean, "")
	canonID := resolved.CanonicalID
	if canonID == "" {
		canonID = clean
	}

	cm.mu.RLock()
	cached, ok := cm.cache[canonID]
	rec := cm.recommender
	cm.mu.RUnlock()

	if ok && cached != nil {
		res := *cached
		res.Offers = append([]PackageOffer(nil), cached.Offers...)
		rec.RankAndAnnotate(&res, preferredOfferID)
		cm.txMgr.RegisterOffers(res.Offers)
		return &res, nil
	}

	var (
		installedMap       map[string]string
		crawlerMeta        *appstream.AppstreamMetadata
		discoveredPackages []models.Package
		topWg              sync.WaitGroup
		pkgLock            sync.Mutex
	)

	topWg.Add(1)
	go func() {
		defer topWg.Done()
		if cm.installedFetcher != nil {
			installedMap = cm.installedFetcher(ctx)
		} else {
			installedMap = make(map[string]string)
		}
	}()

	topWg.Add(1)
	go func() {
		defer topWg.Done()
		if cm.crawler != nil {
			meta, err := cm.crawler.GetAppstream(ctx, canonID)
			if err == nil && meta != nil {
				crawlerMeta = meta
			} else {
				meta, err := cm.crawler.GetAppstream(ctx, clean)
				if err == nil && meta != nil {
					crawlerMeta = meta
				}
			}
		}
	}()

	aliases := cm.deduper.IdentityResolver().GetPackageAliases(canonID)
	if len(aliases) == 0 {
		aliases = []string{clean}
	}

	for _, alias := range aliases {
		topWg.Add(1)
		go func(name string) {
			defer topWg.Done()
			subCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
			defer cancel()

			if cm.pm != nil {
				infos := cm.pm.InfoAll(subCtx, name)
				if len(infos) > 0 {
					pkgLock.Lock()
					for _, info := range infos {
						discoveredPackages = append(discoveredPackages, models.Package{
							Name:         info.Name,
							Version:      info.Version,
							Description:  info.Description,
							Size:         info.Size,
							Provider:     info.Provider,
							Repo:         info.Repo,
							Dependencies: info.Dependencies,
						})
					}
					pkgLock.Unlock()
				}
			}
		}(alias)
	}
	topWg.Wait()

	crawlerMap := make(map[string]*appstream.AppstreamMetadata)
	if crawlerMeta != nil {
		crawlerMap[crawlerMeta.ID] = crawlerMeta
	}

	unifiedList := cm.deduper.GroupAndDeduplicate(discoveredPackages, crawlerMap, installedMap)

	var targetApp *UnifiedApplication
	for i := range unifiedList {
		if strings.EqualFold(unifiedList[i].AppID, canonID) {
			targetApp = &unifiedList[i]
			break
		}
	}
	if targetApp == nil && len(unifiedList) > 0 {
		targetApp = &unifiedList[0]
	}

	if targetApp == nil {
		fallbackProv := "pacman"
		fallbackRepo := "extra"
		switch utils.DetectPackageManager() {
		case "apt":
			fallbackProv = "apt"
			fallbackRepo = "main"
		case "dnf":
			fallbackProv = "dnf"
			fallbackRepo = "fedora"
		case "zypper":
			fallbackProv = "zypper"
			fallbackRepo = "oss"
		}
		fallbackHomepage := HomepageForProvider(fallbackProv, clean)

		trust := EvaluateSourceTrust(fallbackProv, fallbackRepo, clean)
		fallbackOffer := PackageOffer{
			ID:              BuildOfferIdentity(fallbackProv, fallbackRepo, clean, "", "", ""),
			AppID:           canonID,
			PackageName:     clean,
			Provider:        fallbackProv,
			Repository:      fallbackRepo,
			Source:          fallbackRepo,
			Architecture:    hostDefaultArch(),
			Version:         "Latest",
			DownloadSize:    "~45 MB",
			InstalledSize:   "~120 MB",
			InstallScope:    ScopeSystem,
			Scope:           ScopeSystem,
			Permissions:     DeterminePermissions(fallbackProv, clean),
			IsEligible:      true,
			TrustLevel:      trust.TrustLevel,
			SignatureStatus: trust.SignatureStatus,
			Trust:           trust,
			CommandPreview:  FormatCommandPreview(fallbackProv, fallbackRepo, clean, "", ""),
		}

		targetApp = &UnifiedApplication{
			AppID:              canonID,
			Name:               strings.Title(clean),
			Summary:            fmt.Sprintf("Application package for %s", clean),
			Description:        "Software package indexed for Linux.",
			Developer:          "Linux Community Maintainers",
			Homepage:           fallbackHomepage,
			License:            "Open Source",
			Categories:         []string{"Utilities"},
			Screenshots:        []string{},
			Releases:           []string{},
			Aliases:            aliases,
			IdentityConfidence: resolved.Confidence,
			Offers:             []PackageOffer{},
		}
		if len(discoveredPackages) > 0 {
			for _, p := range discoveredPackages {
				targetApp.Offers = append(targetApp.Offers, cm.deduper.ConvertPackageToOffer(p, false, ""))
			}
		} else {
			targetApp.Offers = append(targetApp.Offers, fallbackOffer)
		}
	}

	rec.RankAndAnnotate(targetApp, preferredOfferID)
	cm.txMgr.RegisterOffers(targetApp.Offers)

	if cm.iconResolver != nil {
		iconDTO, err := cm.iconResolver.ResolveAppIcon(ctx, targetApp.AppID, clean, "", icons.ModeCatalog, 128)
		if err == nil && iconDTO != nil {
			targetApp.Icon = *iconDTO
			if iconDTO.IconURL != "" {
				targetApp.IconURL = iconDTO.IconURL
			} else if iconDTO.DataURI != "" {
				targetApp.IconURL = iconDTO.DataURI
			}
		}
	}

	cm.mu.Lock()
	cm.cache[canonID] = targetApp
	cm.mu.Unlock()

	return targetApp, nil
}

// InstallOffer executes an authoritative OfferID-based install through the TransactionManager.
func (cm *CatalogManager) InstallOffer(ctx context.Context, offerID string, onUpdate func(TransactionDTO)) (TransactionDTO, error) {
	return cm.txMgr.ExecuteOfferTransaction(ctx, offerID, OpInstall, onUpdate)
}

// RemoveOffer executes an authoritative OfferID-based removal through the TransactionManager.
func (cm *CatalogManager) RemoveOffer(ctx context.Context, offerID string, onUpdate func(TransactionDTO)) (TransactionDTO, error) {
	return cm.txMgr.ExecuteOfferTransaction(ctx, offerID, OpRemove, onUpdate)
}

// ProgressiveSearch returns a handle to the progressive search engine.
func (cm *CatalogManager) ProgressiveSearch() *ProgressiveSearchEngine {
	return cm.progressive
}
