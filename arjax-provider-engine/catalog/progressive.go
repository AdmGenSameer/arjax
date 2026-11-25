package catalog

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// ProgressiveSearchEngine fans out searches concurrently across providers and emits
// ordered, sequence-guarded incremental deltas and lifecycle status events.
type ProgressiveSearchEngine struct {
	pm               *providers.ProviderManager
	crawler          *appstream.Crawler
	deduper          *Deduplicator
	recommender      *RecommendationEngine
	installedFetcher func(ctx context.Context) map[string]string
	offerObserver    func([]PackageOffer)
	mu               sync.Mutex
	activeSearches   map[string]context.CancelFunc
}

// NewProgressiveSearchEngine constructs an aggregated progressive search engine.
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

// SetOfferObserver attaches a hook (such as TransactionManager.RegisterOffers) invoked when offers are discovered.
func (pse *ProgressiveSearchEngine) SetOfferObserver(fn func([]PackageOffer)) {
	pse.mu.Lock()
	defer pse.mu.Unlock()
	pse.offerObserver = fn
}

// CancelSearch terminates an active in-flight progressive search by ID.
func (pse *ProgressiveSearchEngine) CancelSearch(searchID string) {
	pse.mu.Lock()
	defer pse.mu.Unlock()
	if cancel, exists := pse.activeSearches[searchID]; exists {
		cancel()
		delete(pse.activeSearches, searchID)
	}
}

type providerResult struct {
	provider   string
	packages   []models.Package
	crawler    map[string]*appstream.AppstreamMetadata
	err        error
	durationMs int64
}

// StreamSearch maintains compatibility with simple batch+done callbacks while using the expressive stream internally.
func (pse *ProgressiveSearchEngine) StreamSearch(
	parentCtx context.Context,
	searchID string,
	query string,
	batchFn func(batch ProgressiveSearchBatchDTO),
	doneFn func(done ProgressiveSearchDoneDTO),
) {
	pse.StreamSearchWithCallbacks(parentCtx, searchID, query, ProgressiveSearchCallbacks{
		OnBatch: batchFn,
		OnDone:  doneFn,
	})
}

// StreamSearchWithCallbacks executes a concurrent, sequence-ordered search emitting rich status and delta events.
func (pse *ProgressiveSearchEngine) StreamSearchWithCallbacks(
	parentCtx context.Context,
	searchID string,
	query string,
	cb ProgressiveSearchCallbacks,
) {
	trimmed := strings.TrimSpace(query)
	var seq atomic.Int64
	nextSeq := func() int64 { return seq.Add(1) }

	if trimmed == "" {
		if cb.OnDone != nil {
			cb.OnDone(ProgressiveSearchDoneDTO{
				SearchEventBase: SearchEventBase{
					SearchID: searchID,
					Sequence: nextSeq(),
					Status:   "completed",
				},
				TotalApps: 0,
			})
		}
		return
	}

	pse.CancelSearch(searchID)

	timeoutSec := 15
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeoutSec)*time.Second)
	pse.mu.Lock()
	pse.activeSearches[searchID] = cancel
	observer := pse.offerObserver
	pse.mu.Unlock()

	defer func() {
		pse.mu.Lock()
		delete(pse.activeSearches, searchID)
		pse.mu.Unlock()
		cancel()
	}()

	startTime := time.Now()

	var targetProviders []string
	if pse.crawler != nil {
		targetProviders = append(targetProviders, "appstream")
	}
	if pse.pm != nil {
		targetProviders = append(targetProviders, pse.pm.GetSupportedProviderNames()...)
	}

	if cb.OnStarted != nil {
		cb.OnStarted(SearchStartedDTO{
			SearchEventBase: SearchEventBase{
				SearchID: searchID,
				Sequence: nextSeq(),
				Status:   "started",
			},
			Query:     trimmed,
			Providers: targetProviders,
		})
	}

	// If parent context is already cancelled before launching providers, short-circuit cleanly
	if parentCtx.Err() != nil {
		if cb.OnCancelled != nil {
			cb.OnCancelled(ProgressiveSearchDoneDTO{
				SearchEventBase: SearchEventBase{
					SearchID:   searchID,
					Sequence:   nextSeq(),
					Status:     "cancelled",
					DurationMs: time.Since(startTime).Milliseconds(),
				},
				Cancelled: true,
			})
		}
		if cb.OnDone != nil {
			cb.OnDone(ProgressiveSearchDoneDTO{
				SearchEventBase: SearchEventBase{
					SearchID:   searchID,
					Sequence:   nextSeq(),
					Status:     "cancelled",
					DurationMs: time.Since(startTime).Milliseconds(),
				},
				Cancelled: true,
			})
		}
		return
	}

	var installedMap map[string]string
	if pse.installedFetcher != nil {
		installedMap = pse.installedFetcher(ctx)
	} else {
		installedMap = make(map[string]string)
	}

	resultsChan := make(chan providerResult, len(targetProviders)+2)
	var wg sync.WaitGroup

	// 1. AppStream / Flathub crawler goroutine
	if pse.crawler != nil {
		if cb.OnProviderStatus != nil {
			cb.OnProviderStatus(SearchProviderStatusDTO{
				SearchEventBase: SearchEventBase{
					SearchID: searchID,
					Sequence: nextSeq(),
					Provider: "appstream",
					Status:   "running",
				},
			})
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			pStart := time.Now()
			summaries, err := pse.crawler.SearchApps(ctx, trimmed, 24)
			if err != nil {
				utils.LogDebug("Crawler search note for %s: %v", trimmed, err)
				resultsChan <- providerResult{provider: "appstream", err: err, durationMs: time.Since(pStart).Milliseconds()}
				return
			}

			metaMap := make(map[string]*appstream.AppstreamMetadata)
			var pkgs []models.Package

			for _, s := range summaries {
				if ctx.Err() != nil {
					return
				}
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
					Branch:      "stable",
				})
			}

			resultsChan <- providerResult{
				provider:   "appstream",
				packages:   pkgs,
				crawler:    metaMap,
				durationMs: time.Since(pStart).Milliseconds(),
			}
		}()
	}

	// 2. Per-provider goroutines calling ONLY their own provider via SearchProvider (no N*N SearchAll)
	if pse.pm != nil {
		for _, name := range pse.pm.GetSupportedProviderNames() {
			if cb.OnProviderStatus != nil {
				cb.OnProviderStatus(SearchProviderStatusDTO{
					SearchEventBase: SearchEventBase{
						SearchID: searchID,
						Sequence: nextSeq(),
						Provider: name,
						Status:   "running",
					},
				})
			}

			wg.Add(1)
			go func(provName string) {
				defer wg.Done()
				pStart := time.Now()
				subCtx, subCancel := context.WithTimeout(ctx, 6*time.Second)
				defer subCancel()

				pkgs, err := pse.pm.SearchProvider(subCtx, provName, trimmed)
				dur := time.Since(pStart).Milliseconds()
				if err != nil {
					utils.LogError("Provider search error for %s on query %s: %v", provName, trimmed, err)
					resultsChan <- providerResult{provider: provName, err: err, durationMs: dur}
					return
				}

				resultsChan <- providerResult{
					provider:   provName,
					packages:   pkgs,
					durationMs: dur,
				}
			}(name)
		}
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	allPackages := make([]models.Package, 0, 50)
	allCrawlerMeta := make(map[string]*appstream.AppstreamMetadata)
	prevApps := make(map[string]UnifiedApplication)
	wasCancelled := false

SearchLoop:
	for {
		select {
		case <-ctx.Done():
			wasCancelled = errors.Is(ctx.Err(), context.Canceled)
			break SearchLoop
		case res, ok := <-resultsChan:
			if !ok {
				break SearchLoop
			}

			if res.err != nil {
				if cb.OnProviderError != nil {
					cb.OnProviderError(SearchProviderStatusDTO{
						SearchEventBase: SearchEventBase{
							SearchID:   searchID,
							Sequence:   nextSeq(),
							Provider:   res.provider,
							Status:     "failed",
							DurationMs: res.durationMs,
						},
						Error: res.err.Error(),
					})
				}
				continue
			}

			if cb.OnProviderStatus != nil {
				cb.OnProviderStatus(SearchProviderStatusDTO{
					SearchEventBase: SearchEventBase{
						SearchID:   searchID,
						Sequence:   nextSeq(),
						Provider:   res.provider,
						Status:     "completed",
						DurationMs: res.durationMs,
					},
					ResultCount: len(res.packages),
				})
			}

			if len(res.packages) == 0 && len(res.crawler) == 0 {
				continue
			}

			if res.crawler != nil {
				for k, v := range res.crawler {
					allCrawlerMeta[k] = v
				}
			}
			allPackages = append(allPackages, res.packages...)

			unifiedApps := pse.deduper.GroupAndDeduplicate(allPackages, allCrawlerMeta, installedMap)
			var addedApps []UnifiedApplication
			var updatedApps []UnifiedApplication
			var discoveredOffers []PackageOffer

			for i := range unifiedApps {
				pse.recommender.RankAndAnnotate(&unifiedApps[i], "")
				discoveredOffers = append(discoveredOffers, unifiedApps[i].Offers...)

				prev, existed := prevApps[unifiedApps[i].AppID]
				if !existed {
					addedApps = append(addedApps, unifiedApps[i])
				} else if !reflect.DeepEqual(prev.Offers, unifiedApps[i].Offers) || prev.DefaultOfferID != unifiedApps[i].DefaultOfferID {
					updatedApps = append(updatedApps, unifiedApps[i])
				}
				prevApps[unifiedApps[i].AppID] = unifiedApps[i]
			}

			if observer != nil && len(discoveredOffers) > 0 {
				observer(discoveredOffers)
			}

			if cb.OnBatch != nil && (len(addedApps) > 0 || len(updatedApps) > 0 || len(unifiedApps) > 0) {
				cb.OnBatch(ProgressiveSearchBatchDTO{
					SearchEventBase: SearchEventBase{
						SearchID:   searchID,
						Sequence:   nextSeq(),
						Provider:   res.provider,
						Status:     "running",
						DurationMs: time.Since(startTime).Milliseconds(),
					},
					AddedApps:   addedApps,
					UpdatedApps: updatedApps,
					Apps:        unifiedApps,
					IsFinal:     false,
				})
			}
		}
	}

	if ctx.Err() != nil && !wasCancelled {
		wasCancelled = errors.Is(ctx.Err(), context.Canceled)
	}

	finalStatus := "completed"
	if wasCancelled {
		finalStatus = "cancelled"
		if cb.OnCancelled != nil {
			cb.OnCancelled(ProgressiveSearchDoneDTO{
				SearchEventBase: SearchEventBase{
					SearchID:   searchID,
					Sequence:   nextSeq(),
					Status:     "cancelled",
					DurationMs: time.Since(startTime).Milliseconds(),
				},
				TotalApps: len(prevApps),
				Cancelled: true,
			})
		}
	}

	if cb.OnDone != nil {
		cb.OnDone(ProgressiveSearchDoneDTO{
			SearchEventBase: SearchEventBase{
				SearchID:   searchID,
				Sequence:   nextSeq(),
				Status:     finalStatus,
				DurationMs: time.Since(startTime).Milliseconds(),
			},
			TotalApps: len(prevApps),
			Cancelled: wasCancelled,
		})
	}
}

// SearchSynchronous executes an aggregated search and returns the final deduplicated result.
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
