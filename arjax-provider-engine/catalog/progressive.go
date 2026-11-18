package catalog

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// ProgressiveSearchEngine fans out searches across package providers and streams deduplicated results
type ProgressiveSearchEngine struct {
	pm               *providers.ProviderManager
	crawler          *appstream.Crawler
	deduper          *Deduplicator
	recommender      *RecommendationEngine
	installedFetcher func(ctx context.Context) map[string]string
	mu               sync.Mutex
	activeSearches   map[string]context.CancelFunc
}

// NewProgressiveSearchEngine constructs an aggregated progressive search engine
func NewProgressiveSearchEngine(
	pm *providers.ProviderManager,
	crawler *appstream.Crawler,
	deduper *Deduplicator,
	recommender *RecommendationEngine,
	installedFetcher func(ctx context.Context) map[string]string,
) *ProgressiveSearchEngine {
	return &ProgressiveSearchEngine{
		pm:               pm,
		crawler:          crawler,
		deduper:          deduper,
		recommender:      recommender,
		installedFetcher: installedFetcher,
		activeSearches:   make(map[string]context.CancelFunc),
	}
}

// CancelSearch terminates an active in-flight progressive search by ID
func (pse *ProgressiveSearchEngine) CancelSearch(searchID string) {
	pse.mu.Lock()
	defer pse.mu.Unlock()
	if cancel, exists := pse.activeSearches[searchID]; exists {
		cancel()
		delete(pse.activeSearches, searchID)
	}
}

type providerResult struct {
	provider string
	packages []models.Package
	crawler  map[string]*appstream.AppstreamMetadata
	err      error
}

// StreamSearch executes an asynchronous fanned-out search streaming batches to batchFn
func (pse *ProgressiveSearchEngine) StreamSearch(
	parentCtx context.Context,
	searchID string,
	query string,
	batchFn func(batch ProgressiveSearchBatchDTO),
	doneFn func(done ProgressiveSearchDoneDTO),
) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		if doneFn != nil {
			doneFn(ProgressiveSearchDoneDTO{SearchID: searchID, TotalApps: 0, DurationMs: 0})
		}
		return
	}

	// Cancel any prior search with the same searchID
	pse.CancelSearch(searchID)

	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	pse.mu.Lock()
	pse.activeSearches[searchID] = cancel
	pse.mu.Unlock()

	defer func() {
		pse.mu.Lock()
		delete(pse.activeSearches, searchID)
		pse.mu.Unlock()
		cancel()
	}()

	startTime := time.Now()

	// Get installed packages for state determination
	var installedMap map[string]string
	if pse.installedFetcher != nil {
		installedMap = pse.installedFetcher(ctx)
	} else {
		installedMap = make(map[string]string)
	}

	resultsChan := make(chan providerResult, 10)
	var wg sync.WaitGroup

	// 1. AppStream / Flathub crawler task
	if pse.crawler != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			summaries, err := pse.crawler.SearchApps(ctx, trimmed, 24)
			if err != nil {
				utils.LogDebug("Crawler search note for %s: %v", trimmed, err)
				resultsChan <- providerResult{provider: "appstream", err: err}
				return
			}

			metaMap := make(map[string]*appstream.AppstreamMetadata)
			var pkgs []models.Package

			for _, s := range summaries {
				meta, err := pse.crawler.GetAppstream(ctx, s.ID)
				if err == nil && meta != nil {
					metaMap[meta.ID] = meta
				} else {
					metaMap[s.ID] = &appstream.AppstreamMetadata{
						ID:          s.ID,
						FlathubID:   s.FlathubID,
						Name:        s.Name,
						Summary:     s.Summary,
						Description: s.Summary,
						Icon:        s.Icon,
						License:     "Open Source",
					}
				}
				// Also create a package entry
				flatpakID := s.FlathubID
				if flatpakID == "" {
					flatpakID = s.ID
				}
				pkgs = append(pkgs, models.Package{
					Name:        flatpakID,
					Version:     "Latest",
					Description: s.Summary,
					Provider:    "flatpak",
					Source:      "flathub",
				})
			}

			resultsChan <- providerResult{
				provider: "appstream",
				packages: pkgs,
				crawler:  metaMap,
			}
		}()
	}

	// 2. Package Providers (Pacman, AUR, Snap, GitHub, etc.)
	if pse.pm != nil {
		providerNames := pse.pm.GetSupportedProviderNames()
		for _, name := range providerNames {
			wg.Add(1)
			go func(provName string) {
				defer wg.Done()
				// Query individual provider through SearchAll or specific provider search
				subCtx, subCancel := context.WithTimeout(ctx, 6*time.Second)
				defer subCancel()

				pkgs, err := pse.pm.SearchAll(subCtx, trimmed)
				if err != nil {
					utils.LogError("Provider search error for %s on query %s: %v", provName, trimmed, err)
					resultsChan <- providerResult{provider: provName, err: err}
					return
				}

				// Filter packages specifically for this provider
				var filtered []models.Package
				for _, p := range pkgs {
					if strings.EqualFold(p.Provider, provName) {
						filtered = append(filtered, p)
					}
				}

				resultsChan <- providerResult{
					provider: provName,
					packages: filtered,
				}
			}(name)
		}
	}

	// Closer goroutine
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Accumulator state
	allPackages := make([]models.Package, 0, 50)
	allCrawlerMeta := make(map[string]*appstream.AppstreamMetadata)
	knownApps := make(map[string]UnifiedApplication)

SearchLoop:
	for res := range resultsChan {
		select {
		case <-ctx.Done():
			break SearchLoop
		default:
		}

		if res.err != nil || (len(res.packages) == 0 && len(res.crawler) == 0) {
			continue
		}

		if res.crawler != nil {
			for k, v := range res.crawler {
				allCrawlerMeta[k] = v
			}
		}
		allPackages = append(allPackages, res.packages...)

		// Run deduplication & recommendation ranking on current cumulative state
		unifiedApps := pse.deduper.GroupAndDeduplicate(allPackages, allCrawlerMeta, installedMap)
		for i := range unifiedApps {
			pse.recommender.RankAndAnnotate(&unifiedApps[i], "")
			knownApps[unifiedApps[i].AppID] = unifiedApps[i]
		}

		// Emit progressive batch
		if batchFn != nil && len(unifiedApps) > 0 {
			batchList := make([]UnifiedApplication, 0, len(knownApps))
			for _, app := range knownApps {
				batchList = append(batchList, app)
			}

			batchFn(ProgressiveSearchBatchDTO{
				SearchID:   searchID,
				Apps:       batchList,
				Provider:   res.provider,
				IsFinal:    false,
				DurationMs: time.Since(startTime).Milliseconds(),
			})
		}
	}

	// Final completion notification
	if doneFn != nil {
		doneFn(ProgressiveSearchDoneDTO{
			SearchID:   searchID,
			TotalApps:  len(knownApps),
			DurationMs: time.Since(startTime).Milliseconds(),
		})
	}
}

// SearchSynchronous executes an aggregated search and returns the final deduplicated result
func (pse *ProgressiveSearchEngine) SearchSynchronous(ctx context.Context, query string) ([]UnifiedApplication, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []UnifiedApplication{}, nil
	}

	var finalApps []UnifiedApplication
	var finalLock sync.Mutex

	searchID := "sync-" + query
	doneChan := make(chan struct{})

	go func() {
		defer close(doneChan)
		pse.StreamSearch(ctx, searchID, query, func(batch ProgressiveSearchBatchDTO) {
			finalLock.Lock()
			finalApps = batch.Apps
			finalLock.Unlock()
		}, func(done ProgressiveSearchDoneDTO) {
			// Search completed
		})
	}()

	select {
	case <-doneChan:
		finalLock.Lock()
		defer finalLock.Unlock()
		return finalApps, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
