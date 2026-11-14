package providers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/cache"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// Provider interface defines methods for package operations
type Provider interface {
	Name() string
	IsSupported() bool

	Search(ctx context.Context, query string) ([]models.Package, error)
	Install(ctx context.Context, pkg models.Package) (models.Result, error)
	Remove(ctx context.Context, pkg models.Package) (models.Result, error)
	Update(ctx context.Context) (models.Result, error)
	Info(ctx context.Context, pkgName string) (models.PackageInfo, error)
	BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error)
}

// ProviderManager orchestrates registered providers
type ProviderManager struct {
	providers []Provider
	cache     *cache.ProviderCache
	config    config.Config
}

func NewProviderManager(cfg config.Config, providers ...Provider) *ProviderManager {
	supported := make([]Provider, 0)
	for _, p := range providers {
		if p.IsSupported() {
			supported = append(supported, p)
		}
	}
	return &ProviderManager{
		providers: supported,
		cache:     cache.NewProviderCache(cfg.CacheTTL()),
		config:    cfg,
	}
}

// SearchAll searches concurrently across all supported providers
func (pm *ProviderManager) SearchAll(ctx context.Context, query string) ([]models.Package, error) {
	if cached, ok := pm.cache.GetSearch(query); ok {
		utils.LogDebug("Cache hit for query: %s", query)
		return cached, nil
	}

	resultsChan := make(chan []models.Package, len(pm.providers))
	var wg sync.WaitGroup

	for _, provider := range pm.providers {
		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()
			pkgs, err := p.Search(ctx, query)
			if err != nil {
				utils.LogError("Search error from provider %s: %v", p.Name(), err)
				return
			}
			resultsChan <- pkgs
		}(provider)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var allPackages []models.Package
	for pkgs := range resultsChan {
		allPackages = append(allPackages, pkgs...)
	}

	pm.cache.SetSearch(query, allPackages)
	return allPackages, nil
}

// Install attempts installation, trying providers in priority order
func (pm *ProviderManager) Install(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()

	// If provider explicitly defined on package, try it first
	if pkg.Provider != "" {
		for _, p := range pm.providers {
			if p.Name() == pkg.Provider {
				res, err := p.Install(ctx, pkg)
				res.Command = "install"
				if err == nil && res.Success {
					return res, nil
				}
				if err != nil {
					return res, fmt.Errorf("%s provider failed: %w", pkg.Provider, err)
				}
				if !res.Success {
					return res, fmt.Errorf("%s provider failed: %s", pkg.Provider, res.Error)
				}
			}
		}
	}

	// Otherwise iterate through available providers as fallbacks
	var lastErr error
	for _, p := range pm.providers {
		utils.LogInfo("Attempting install with provider: %s", p.Name())
		res, err := p.Install(ctx, pkg)
		res.Command = "install"
		if err == nil && res.Success {
			return res, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	duration := time.Since(start).Milliseconds()
	errMsg := "all providers failed to install package"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	return models.Result{
		Command:    "install",
		Package:    pkg.Name,
		Provider:   pkg.Provider,
		Success:    false,
		StartTime:  start,
		EndTime:    time.Now(),
		DurationMs: duration,
		Error:      errMsg,
	}, fmt.Errorf("installation failed for %s: %w", pkg.Name, lastErr)
}

// Remove uninstalls a package
func (pm *ProviderManager) Remove(ctx context.Context, pkg models.Package) (models.Result, error) {
	start := time.Now()
	for _, p := range pm.providers {
		if pkg.Provider == "" || p.Name() == pkg.Provider {
			res, err := p.Remove(ctx, pkg)
			res.Command = "remove"
			if err == nil {
				return res, nil
			}
		}
	}
	return models.Result{
		Command:    "remove",
		Package:    pkg.Name,
		Provider:   pkg.Provider,
		Success:    false,
		StartTime:  start,
		EndTime:    time.Now(),
		DurationMs: time.Since(start).Milliseconds(),
		Error:      "no provider succeeded in removing package",
	}, fmt.Errorf("remove failed for %s", pkg.Name)
}

// Info retrieves package information
func (pm *ProviderManager) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	for _, p := range pm.providers {
		info, err := p.Info(ctx, pkgName)
		if err == nil && info.Name != "" {
			info.Provider = p.Name()
			return info, nil
		}
	}
	return models.PackageInfo{}, fmt.Errorf("package info not found for %s", pkgName)
}

// BatchInstall handles concurrent or bulk package installations
func (pm *ProviderManager) BatchInstall(ctx context.Context, pkgs []models.Package) ([]models.Result, error) {
	results := make([]models.Result, len(pkgs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, pm.config.Provider.MaxConcurrentDownloads)

	for i, pkg := range pkgs {
		wg.Add(1)
		go func(idx int, p models.Package) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, _ := pm.Install(ctx, p)
			results[idx] = res
		}(i, pkg)
	}

	wg.Wait()
	return results, nil
}
