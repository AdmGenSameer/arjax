package icons

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// IconResolverService orchestrates universal icon retrieval across all package formats
type IconResolverService struct {
	pipeline  *IconPipeline
	cacheMgr  *CacheManager
	providers []IconProvider
}

// NewIconResolverService initializes the service with all priority resolvers and caches
func NewIconResolverService(customCacheDir string) (*IconResolverService, error) {
	pipe, err := NewIconPipeline(customCacheDir)
	if err != nil {
		return nil, err
	}

	httpClient := NewSecureHTTPClient(6 * time.Second)
	themeResolver := NewSystemThemeResolver()
	flatpakResolver := NewFlatpakResolver(httpClient)
	snapResolver := NewSnapResolver(httpClient)
	aurResolver := NewAURResolver(flatpakResolver, snapResolver)
	desktopResolver := NewDesktopFileResolver(themeResolver)
	appstreamResolver := NewAppStreamResolver(httpClient)
	appimageResolver := NewAppImageResolver()
	fallbackResolver := NewFallbackResolver()

	providers := []IconProvider{
		appstreamResolver,
		flatpakResolver,
		snapResolver,
		aurResolver,
		desktopResolver,
		themeResolver,
		appimageResolver,
		fallbackResolver,
	}

	return &IconResolverService{
		pipeline:  pipe,
		cacheMgr:  pipe.cacheMgr,
		providers: providers,
	}, nil
}

// ResolveAppIcon is the primary distribution-independent entry point
func (s *IconResolverService) ResolveAppIcon(ctx context.Context, appID, pkgName, preferredSource string, mode ResolutionMode, requestedSize int) (*IconResponseDTO, error) {
	aliasReg := GetAliasRegistry()
	normAppID := strings.TrimSpace(appID)
	if normAppID == "" {
		normAppID = strings.TrimSpace(pkgName)
	}
	if normAppID == "" {
		return nil, fmt.Errorf("appId or pkgName must be specified")
	}

	canonID := aliasReg.ResolveCanonical(normAppID)
	if canonID == "" {
		canonID = normAppID
	}

	if mode == "" {
		mode = ModeCatalog
	}
	if requestedSize <= 0 {
		requestedSize = DefaultIconSize
	}

	flightKey := fmt.Sprintf("%s:%s:%s:%d", canonID, preferredSource, mode, requestedSize)
	return s.cacheMgr.flight.Do(flightKey, func() (*IconResponseDTO, error) {
		return s.resolveInternal(ctx, canonID, normAppID, pkgName, preferredSource, mode, requestedSize)
	})
}

func (s *IconResolverService) resolveInternal(ctx context.Context, canonID, rawAppID, pkgName, preferredSource string, mode ResolutionMode, requestedSize int) (*IconResponseDTO, error) {
	// 1. Check Cache
	rec, exists, needsRefresh := s.cacheMgr.Get(canonID)
	if !exists {
		// Try rawAppID if different from canonID
		rec, exists, needsRefresh = s.cacheMgr.Get(rawAppID)
	}

	if exists && rec != nil {
		targetPath := rec.Variants[requestedSize]
		if targetPath == "" {
			targetPath = rec.Variants[DefaultIconSize]
		}
		if targetPath == "" {
			targetPath = rec.OriginalPath
		}

		if targetPath != "" {
			// Verify file still exists on disk
			if data, err := os.ReadFile(targetPath); err == nil && len(data) > 0 {
				dto := s.buildDTO(canonID, rec, targetPath, data, requestedSize)

				// If soft TTL expired, refresh asynchronously in background
				if needsRefresh && !rec.IsNegative {
					go func() {
						bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						_, _ = s.fetchAndCache(bgCtx, canonID, pkgName, preferredSource, mode)
					}()
				}

				return dto, nil
			}
		}
	}

	// 2. Fetch from providers in priority order for the given mode
	record, err := s.fetchAndCache(ctx, canonID, pkgName, preferredSource, mode)
	if err != nil {
		// Fallback to generic icon
		fb := NewFallbackResolver()
		matchedAsset, _ := fb.ResolveIcon(ctx, canonID, pkgName, mode)
		rec, pErr := s.pipeline.ProcessAndCache(canonID, matchedAsset)
		if pErr != nil {
			return nil, pErr
		}
		rec.Status = "fallback"
		s.cacheMgr.PutNegative(canonID, "fallback")
		targetPath := rec.Variants[requestedSize]
		if targetPath == "" {
			targetPath = rec.OriginalPath
		}
		data, _ := os.ReadFile(targetPath)
		return s.buildDTO(canonID, rec, targetPath, data, requestedSize), nil
	}

	targetPath := record.Variants[requestedSize]
	if targetPath == "" {
		targetPath = record.Variants[DefaultIconSize]
	}
	if targetPath == "" {
		targetPath = record.OriginalPath
	}
	data, _ := os.ReadFile(targetPath)
	return s.buildDTO(canonID, record, targetPath, data, requestedSize), nil
}

func (s *IconResolverService) fetchAndCache(ctx context.Context, appID, pkgName, preferredSource string, mode ResolutionMode) (*IconRecord, error) {
	// Sort providers by priority for the given mode
	sortedProviders := make([]IconProvider, len(s.providers))
	copy(sortedProviders, s.providers)
	sort.Slice(sortedProviders, func(i, j int) bool {
		return sortedProviders[i].Priority(mode) < sortedProviders[j].Priority(mode)
	})

	var matchedAsset *IconAsset
	var chosenProvider IconProvider

	for _, p := range sortedProviders {
		// Do not use fallback resolver in this loop
		if p.Priority(mode) >= 99 {
			continue
		}

		if !p.Supports(mode, preferredSource) {
			continue
		}

		asset, err := p.ResolveIcon(ctx, appID, pkgName, mode)
		if err == nil && asset != nil && len(asset.Data) > 0 {
			matchedAsset = asset
			chosenProvider = p
			break
		}
	}

	if matchedAsset == nil {
		return nil, fmt.Errorf("no provider found icon for %s", appID)
	}

	record, err := s.pipeline.ProcessAndCache(appID, matchedAsset)
	if err != nil {
		return nil, err
	}

	record.PackageSource = chosenProvider.Name()
	record.Status = "available"
	_ = s.cacheMgr.Put(record)
	return record, nil
}

func (s *IconResolverService) buildDTO(appID string, rec *IconRecord, targetPath string, data []byte, requestedSize int) *IconResponseDTO {
	iconURL := fmt.Sprintf("/api/icons/%s?size=%d", appID, requestedSize)
	dataURI := ""
	if len(data) > 0 {
		dataURI = s.pipeline.ToDataURI(data, rec.OriginalFormat)
	}

	return &IconResponseDTO{
		AppID:       appID,
		IconURL:     iconURL,
		DataURI:     dataURI,
		IconStatus:  rec.Status,
		Format:      rec.OriginalFormat,
		Width:       rec.Width,
		Height:      rec.Height,
		ContentHash: rec.ContentHash,
		Source:      rec.PackageSource,
		Repository:  rec.Repository,
		LocalPath:   targetPath,
		Provenance:  rec.Provenance,
	}
}

// RefreshIcon forces an icon re-fetch and cache invalidation
func (s *IconResolverService) RefreshIcon(ctx context.Context, appID, pkgName string) (*IconResponseDTO, error) {
	normAppID := strings.TrimSpace(appID)
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(normAppID)

	s.cacheMgr.Invalidate(canonID)
	if normAppID != canonID {
		s.cacheMgr.Invalidate(normAppID)
	}

	return s.ResolveAppIcon(ctx, canonID, pkgName, "", ModeCatalog, DefaultIconSize)
}

// ServeHTTP implements http.Handler to serve cached icons locally via Wails AssetServer
func (s *IconResolverService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/icons/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	appID := parts[0]
	size := DefaultIconSize
	if qSize := r.URL.Query().Get("size"); qSize != "" {
		if sz, err := strconv.Atoi(qSize); err == nil && sz > 0 {
			size = sz
		}
	} else if len(parts) > 1 {
		if sz, err := strconv.Atoi(parts[1]); err == nil && sz > 0 {
			size = sz
		}
	}

	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)

	rec, exists, _ := s.cacheMgr.Get(canonID)
	if !exists && appID != canonID {
		rec, exists, _ = s.cacheMgr.Get(appID)
	}

	// If not in cache, resolve on demand synchronously
	if !exists || rec == nil {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		dto, err := s.ResolveAppIcon(ctx, canonID, appID, "", ModeCatalog, size)
		if err != nil || dto == nil {
			http.NotFound(w, r)
			return
		}
		rec, exists, _ = s.cacheMgr.Get(canonID)
	}

	if rec == nil {
		http.NotFound(w, r)
		return
	}

	targetPath := rec.Variants[size]
	if targetPath == "" {
		targetPath = rec.Variants[DefaultIconSize]
	}
	if targetPath == "" {
		targetPath = rec.OriginalPath
	}

	if targetPath == "" {
		http.NotFound(w, r)
		return
	}

	// HTTP Caching & ETag
	w.Header().Set("ETag", fmt.Sprintf(`"%s"`, rec.ContentHash))
	w.Header().Set("Cache-Control", "public, max-age=2592000, immutable") // 30 days

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Trim(match, `"`) == rec.ContentHash {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	contentType := "image/png"
	if strings.HasSuffix(targetPath, ".svg") {
		contentType = "image/svg+xml"
	} else if strings.HasSuffix(targetPath, ".webp") {
		contentType = "image/webp"
	}
	w.Header().Set("Content-Type", contentType)

	http.ServeFile(w, r, targetPath)
}
