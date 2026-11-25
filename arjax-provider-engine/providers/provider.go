package providers

import (
	"context"
	"fmt"
	"strings"
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

// GetSupportedProviderNames returns list of names for all active supported providers
func (pm *ProviderManager) GetSupportedProviderNames() []string {
	names := make([]string, 0, len(pm.providers))
	for _, p := range pm.providers {
		names = append(names, p.Name())
	}
	return names
}

// GetCapabilityMatrix returns explicit capability & host health metadata across tracked ecosystems
func (pm *ProviderManager) GetCapabilityMatrix() map[string]ProviderCapability {
	matrix := DefaultCapabilityMatrix()
	// Cross-check dynamically registered mock/custom providers so test harnesses reflect availability
	for _, p := range pm.providers {
		n := strings.ToLower(p.Name())
		if rp, ok := p.(interface{ PackageManager() string }); ok {
			switch strings.ToLower(rp.PackageManager()) {
			case "apt":
				n = "deb"
			case "dnf":
				n = "rpm"
			case "zypper":
				n = "zypper"
			default:
				n = "pacman"
			}
		} else if n == "repository" {
			switch utils.DetectPackageManager() {
			case "apt":
				n = "deb"
			case "dnf":
				n = "rpm"
			case "zypper":
				n = "zypper"
			default:
				n = "pacman"
			}
		}
		if existing, ok := matrix[n]; ok {
			existing.AvailableOnHost = p.IsSupported()
			matrix[n] = existing
		} else {
			matrix[n] = ProviderCapability{
				Provider:           n,
				DisplayName:        strings.ToUpper(n),
				Search:             SupportFull,
				Install:            SupportFull,
				Remove:             SupportFull,
				Update:             SupportPartial,
				InstalledDetection: SupportPartial,
				AvailableOnHost:    p.IsSupported(),
				StatusNote:         "Registered runtime provider adapter",
			}
		}
	}
	return matrix
}

// GetCapabilities returns ordered list of tracked provider capabilities
func (pm *ProviderManager) GetCapabilities() []ProviderCapability {
	return GetCapabilitiesOrdered(pm.GetCapabilityMatrix())
}

// GetProviderStatuses probes runtime availability and service health separately from code support
func (pm *ProviderManager) GetProviderStatuses(ctx context.Context) []ProviderStatus {
	return ProbeProviderStatuses(ctx, pm.GetCapabilityMatrix())
}

// SearchProvider queries a single named provider without invoking unrelated providers
func (pm *ProviderManager) SearchProvider(ctx context.Context, providerName, query string) ([]models.Package, error) {
	for _, p := range pm.providers {
		if strings.EqualFold(p.Name(), providerName) {
			return p.Search(ctx, query)
		}
	}
	return nil, fmt.Errorf("provider %q not registered or unsupported", providerName)
}

// GetProvider returns a registered Provider by name (handling repository <-> system PM alias)
func (pm *ProviderManager) GetProvider(name string) (Provider, bool) {
	norm := strings.ToLower(strings.TrimSpace(name))
	for _, p := range pm.providers {
		pn := strings.ToLower(p.Name())
		if pn == norm {
			return p, true
		}
		if (norm == "pacman" || norm == "apt" || norm == "dnf" || norm == "zypper" || norm == "paru" || norm == "deb" || norm == "rpm") && pn == "repository" {
			return p, true
		}
	}
	return nil, false
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

// Info retrieves package information from the first matching provider
func (pm *ProviderManager) Info(ctx context.Context, pkgName string) (models.PackageInfo, error) {
	for _, p := range pm.providers {
		info, err := p.Info(ctx, pkgName)
		if err == nil && info.Name != "" {
			if info.Provider == "" {
				info.Provider = p.Name()
			}
			return info, nil
		}
	}
	return models.PackageInfo{}, fmt.Errorf("package info not found for %s", pkgName)
}

// InfoAll queries all supported providers concurrently and returns every matching PackageInfo
func (pm *ProviderManager) InfoAll(ctx context.Context, pkgName string) []models.PackageInfo {
	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		out []models.PackageInfo
	)
	for _, p := range pm.providers {
		wg.Add(1)
		go func(prov Provider) {
			defer wg.Done()
			info, err := prov.Info(ctx, pkgName)
			if err == nil && info.Name != "" {
				if info.Provider == "" {
					info.Provider = prov.Name()
				}
				mu.Lock()
				out = append(out, info)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return out
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

// Update performs an orchestrated system update across supported providers in safe order:
// native repository -> AUR -> sandboxed (Flatpak, Snap)
func (pm *ProviderManager) Update(ctx context.Context) (models.Result, error) {
	start := time.Now()
	var errs []string
	successCount := 0

	// Preferred safe execution order: host repos first, then AUR, then sandboxed runtimes
	order := []string{"repository", "pacman", "aur", "flatpak", "snap", "apt", "dnf", "zypper"}
	executed := make(map[string]bool)

	for _, name := range order {
		for _, p := range pm.providers {
			if strings.EqualFold(p.Name(), name) && !executed[p.Name()] {
				executed[p.Name()] = true
				res, err := p.Update(ctx)
				if err != nil || !res.Success {
					if err != nil {
						errs = append(errs, fmt.Sprintf("%s: %v", p.Name(), err))
					} else if res.Error != "" {
						errs = append(errs, fmt.Sprintf("%s: %s", p.Name(), res.Error))
					}
				} else {
					successCount++
				}
			}
		}
	}

	// Any remaining supported providers not in the explicit list (e.g. custom or vendor)
	for _, p := range pm.providers {
		if !executed[p.Name()] {
			executed[p.Name()] = true
			res, err := p.Update(ctx)
			if err != nil || !res.Success {
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", p.Name(), err))
				} else if res.Error != "" {
					errs = append(errs, fmt.Sprintf("%s: %s", p.Name(), res.Error))
				}
			} else {
				successCount++
			}
		}
	}

	overallSuccess := len(errs) == 0 && successCount > 0
	errMsg := strings.Join(errs, "; ")
	var returnErr error
	if len(errs) > 0 {
		returnErr = fmt.Errorf("update completed with errors: %s", errMsg)
	}

	return models.Result{
		Command:    "upgrade",
		Package:    "system",
		Provider:   "all",
		Success:    overallSuccess,
		StartTime:  start,
		EndTime:    time.Now(),
		DurationMs: time.Since(start).Milliseconds(),
		Error:      errMsg,
	}, returnErr
}

