package icons

import (
	"bufio"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// =========================================================================
// 1. AppStream Resolver (Priority 1 in Catalog Mode, Priority 6 in Installed)
// =========================================================================

type AppStreamResolver struct {
	client     *http.Client
	xmlBaseDirs []string
}

func NewAppStreamResolver(client *http.Client) *AppStreamResolver {
	if client == nil {
		client = NewSecureHTTPClient(5 * time.Second)
	}
	return &AppStreamResolver{
		client: client,
		xmlBaseDirs: []string{
			"/var/lib/flatpak/appstream",
			"/var/cache/app-info/xmls",
			"/usr/share/app-info/xmls",
			"/var/lib/swcatalog/xml",
			"/var/cache/swcatalog/xml",
		},
	}
}

func (r *AppStreamResolver) Name() string { return "appstream" }

func (r *AppStreamResolver) Priority(mode ResolutionMode) int {
	if mode == ModeInstalled {
		return 6
	}
	return 1
}

func (r *AppStreamResolver) Supports(mode ResolutionMode, s string) bool {
	return true // Universal metadata catalog
}

func (r *AppStreamResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)
	aliases := aliasReg.GetAliases(appID)

	xmlFiles := r.collectXMLFiles()
	for _, xf := range xmlFiles {
		asset, err := r.parseXMLForIcon(ctx, xf, canonID, aliases)
		if err == nil && asset != nil && len(asset.Data) > 0 {
			return asset, nil
		}
	}

	return nil, fmt.Errorf("appstream icon not found for %s", appID)
}

func (r *AppStreamResolver) collectXMLFiles() []string {
	var files []string
	for _, base := range r.xmlBaseDirs {
		info, err := os.Stat(base)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if strings.HasSuffix(base, ".xml") || strings.HasSuffix(base, ".xml.gz") {
				files = append(files, base)
			}
			continue
		}

		_ = filepath.Walk(base, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi == nil || fi.IsDir() {
				return nil
			}
			if strings.HasSuffix(fi.Name(), ".xml") || strings.HasSuffix(fi.Name(), ".xml.gz") {
				files = append(files, path)
			}
			return nil
		})
	}
	return files
}

func (r *AppStreamResolver) parseXMLForIcon(ctx context.Context, xmlPath, canonID string, aliases []string) (*IconAsset, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(xmlPath, ".gz") {
		uncompressed, err := SafeDecompressGzip(data, 20*1024*1024)
		if err != nil {
			return nil, err
		}
		data = uncompressed
	}

	type XMLIcon struct {
		Type   string `xml:"type,attr"`
		Width  string `xml:"width,attr"`
		Height string `xml:"height,attr"`
		Value  string `xml:",chardata"`
	}

	type XMLComponent struct {
		ID    string    `xml:"id"`
		Pkg   string    `xml:"pkgname"`
		Icons []XMLIcon `xml:"icon"`
	}

	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	origin := "appstream"
	normCanon := strings.ToLower(canonID)

	for {
		t, err := decoder.Token()
		if err != nil {
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "components" {
				for _, attr := range se.Attr {
					if attr.Name.Local == "origin" {
						origin = attr.Value
					}
				}
			}

			if se.Name.Local == "component" {
				var comp XMLComponent
				if err := decoder.DecodeElement(&comp, &se); err == nil {
					cid := strings.ToLower(strings.TrimSuffix(comp.ID, ".desktop"))
					matched := (cid == normCanon)

					if !matched {
						for _, a := range aliases {
							if cid == a || strings.EqualFold(comp.Pkg, a) || strings.HasSuffix(cid, "."+a) {
								matched = true
								break
							}
						}
					}

					if matched {
						// 1. Check cached icons in adjacent directory
						xmlDir := filepath.Dir(xmlPath)
						for _, ic := range comp.Icons {
							val := strings.TrimSpace(ic.Value)
							if ic.Type == "cached" && val != "" {
								// Check adjacent icon directory
								checkPaths := []string{
									filepath.Join(xmlDir, "icons", "128x128", val),
									filepath.Join(xmlDir, "icons", "64x64", val),
									filepath.Join(xmlDir, "..", "icons", "128x128", val),
									filepath.Join("/var/cache/app-info/icons", origin, "128x128", val),
									filepath.Join("/usr/share/app-info/icons", origin, "128x128", val),
								}
								for _, cp := range checkPaths {
									if icBytes, err := os.ReadFile(cp); err == nil && len(icBytes) > 0 {
										return &IconAsset{
											AppID:      canonID,
											Source:     "appstream_cached",
											Repository: origin,
											Data:       icBytes,
											LocalPath:  cp,
											IsLocal:    true,
										}, nil
									}
								}
							}

							// 2. Remote icon URL
							if ic.Type == "remote" && strings.HasPrefix(val, "https://") {
								if err := ValidateRemoteURL(val); err == nil {
									if remoteBytes, err := r.download(ctx, val); err == nil && len(remoteBytes) > 0 {
										return &IconAsset{
											AppID:       canonID,
											Source:      "appstream_remote",
											Repository:  origin,
											OriginalURL: val,
											Data:        remoteBytes,
											IsLocal:     false,
										}, nil
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("not found in %s", xmlPath)
}

func (r *AppStreamResolver) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Arjax-Universal-Icon-Resolver/2.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(io.LimitReader(resp.Body, MaxIconSizeBytes))
}

// =========================================================================
// 2. Flatpak Resolver (Priority 2 in Catalog & Installed Modes)
// =========================================================================

type FlatpakResolver struct {
	client *http.Client
}

func NewFlatpakResolver(client *http.Client) *FlatpakResolver {
	if client == nil {
		client = NewSecureHTTPClient(5 * time.Second)
	}
	return &FlatpakResolver{client: client}
}

func (r *FlatpakResolver) Name() string { return "flatpak" }

func (r *FlatpakResolver) Priority(mode ResolutionMode) int {
	return 2
}

func (r *FlatpakResolver) Supports(mode ResolutionMode, s string) bool {
	if mode == ModeCatalog {
		return true // Can resolve upstream Flathub icons for any uninstalled package
	}
	return s == "flatpak" || s == ""
}

func (r *FlatpakResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)
	candidates := aliasReg.GetAliases(appID)
	if canonID != "" {
		candidates = append([]string{canonID}, candidates...)
	}

	home, _ := os.UserHomeDir()

	for _, cand := range candidates {
		// 1. Local exports (checked in both installed and catalog modes)
		exportPaths := []string{
			fmt.Sprintf("/var/lib/flatpak/exports/share/icons/hicolor/128x128/apps/%s.png", cand),
			fmt.Sprintf("/var/lib/flatpak/exports/share/icons/hicolor/scalable/apps/%s.svg", cand),
			fmt.Sprintf("%s/.local/share/flatpak/exports/share/icons/hicolor/128x128/apps/%s.png", home, cand),
			fmt.Sprintf("%s/.local/share/flatpak/exports/share/icons/hicolor/scalable/apps/%s.svg", home, cand),
		}

		for _, p := range exportPaths {
			if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
				return &IconAsset{
					AppID:      canonID,
					Source:     "flatpak_local",
					Repository: "flathub_local",
					Data:       data,
					LocalPath:  p,
					IsLocal:    true,
				}, nil
			}
		}

		// 2. Remote Flathub CDN (Catalog mode or when local export missing)
		if strings.Contains(cand, ".") {
			remoteURLs := []string{
				fmt.Sprintf("https://dl.flathub.org/media/icons/128x128/%s.png", cand),
				fmt.Sprintf("https://dl.flathub.org/media/icons/64x64/%s.png", cand),
				fmt.Sprintf("https://dl.flathub.org/repo/appstream/x86_64/icons/128x128/%s.png", cand),
			}

			for _, u := range remoteURLs {
				if err := ValidateRemoteURL(u); err != nil {
					continue
				}
				req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
				if err != nil {
					continue
				}
				req.Header.Set("User-Agent", "Arjax-Universal-Icon-Resolver/2.0")
				resp, err := r.client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					data, err := io.ReadAll(io.LimitReader(resp.Body, MaxIconSizeBytes))
					if err == nil && len(data) > 0 {
						return &IconAsset{
							AppID:       canonID,
							Source:      "flatpak_flathub",
							Repository:  "flathub",
							OriginalURL: u,
							Data:        data,
							IsLocal:     false,
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("flatpak icon not found for %s", appID)
}

// =========================================================================
// 3. Snap Resolver (Priority 3 in Catalog & Installed Modes)
// =========================================================================

type SnapResolver struct {
	client *http.Client
}

func NewSnapResolver(client *http.Client) *SnapResolver {
	if client == nil {
		client = NewSecureHTTPClient(5 * time.Second)
	}
	return &SnapResolver{client: client}
}

func (r *SnapResolver) Name() string { return "snap" }

func (r *SnapResolver) Priority(mode ResolutionMode) int {
	return 3
}

func (r *SnapResolver) Supports(mode ResolutionMode, s string) bool {
	if mode == ModeCatalog {
		return true // Can query Snapcraft catalog for uninstalled packages
	}
	return s == "snap" || s == ""
}

func (r *SnapResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)
	aliases := aliasReg.GetAliases(appID)

	candidates := append([]string{pkgName, appID}, aliases...)

	for _, name := range candidates {
		clean := strings.ToLower(strings.TrimSpace(name))
		if clean == "" || strings.Contains(clean, ".") {
			continue
		}

		// 1. Check local installed Snap icons
		localPaths := []string{
			fmt.Sprintf("/snap/%s/current/meta/gui/icon.png", clean),
			fmt.Sprintf("/snap/%s/current/meta/gui/icon.svg", clean),
			fmt.Sprintf("/var/lib/snapd/desktop/applications/%s_%s.png", clean, clean),
		}

		for _, p := range localPaths {
			if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
				return &IconAsset{
					AppID:      canonID,
					Source:     "snap_local",
					Repository: "snapcraft_local",
					Data:       data,
					LocalPath:  p,
					IsLocal:    true,
				}, nil
			}
		}

		// 2. Query Snapcraft API in Catalog Mode
		apiUrl := fmt.Sprintf("https://api.snapcraft.io/v2/snaps/info/%s", clean)
		if err := ValidateRemoteURL(apiUrl); err == nil {
			req, err := http.NewRequestWithContext(ctx, "GET", apiUrl, nil)
			if err == nil {
				req.Header.Set("Snap-Device-Series", "16")
				req.Header.Set("User-Agent", "Arjax-Universal-Icon-Resolver/2.0")
				resp, err := r.client.Do(req)
				if err == nil && resp.StatusCode == http.StatusOK {
					defer resp.Body.Close()
					var snapInfo struct {
						Snap struct {
							Media []struct {
								Type string `json:"type"`
								URL  string `json:"url"`
							} `json:"media"`
						} `json:"snap"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&snapInfo); err == nil {
						for _, m := range snapInfo.Snap.Media {
							if m.Type == "icon" && strings.HasPrefix(m.URL, "https://") {
								if err := ValidateRemoteURL(m.URL); err == nil {
									iconReq, _ := http.NewRequestWithContext(ctx, "GET", m.URL, nil)
									iconResp, err := r.client.Do(iconReq)
									if err == nil && iconResp.StatusCode == http.StatusOK {
										defer iconResp.Body.Close()
										data, _ := io.ReadAll(io.LimitReader(iconResp.Body, MaxIconSizeBytes))
										if len(data) > 0 {
											return &IconAsset{
												AppID:       canonID,
												Source:      "snap_store",
												Repository:  "snapcraft",
												OriginalURL: m.URL,
												Data:        data,
												IsLocal:     false,
											}, nil
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("snap icon not found for %s", appID)
}

// =========================================================================
// 4. AUR Resolver (Priority 4 in Catalog Mode - Strictly NO PKGBUILD execution)
// =========================================================================

type AURResolver struct {
	flatpakResolver *FlatpakResolver
	snapResolver    *SnapResolver
}

func NewAURResolver(flatpak *FlatpakResolver, snap *SnapResolver) *AURResolver {
	return &AURResolver{
		flatpakResolver: flatpak,
		snapResolver:    snap,
	}
}

func (r *AURResolver) Name() string { return "aur" }

func (r *AURResolver) Priority(mode ResolutionMode) int {
	if mode == ModeInstalled {
		return 5
	}
	return 4
}

func (r *AURResolver) Supports(mode ResolutionMode, s string) bool {
	return s == "aur" || s == "pacman" || s == ""
}

// ResolveIcon maps AUR package names to canonical AppIDs and resolves upstream icons
// CRITICAL SECURITY GUARANTEE: Strictly never executes PKGBUILD or external bash scripts
func (r *AURResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)
	if canonID == "" || !strings.Contains(canonID, ".") {
		canonID = aliasReg.ResolveCanonical(pkgName)
	}

	// 1. If we found a canonical AppID, query Flathub CDN directly
	if r.flatpakResolver != nil && strings.Contains(canonID, ".") {
		if asset, err := r.flatpakResolver.ResolveIcon(ctx, canonID, pkgName, mode); err == nil && asset != nil {
			asset.Repository = "aur_upstream"
			return asset, nil
		}
	}

	// 2. Query Snapcraft catalog as secondary upstream
	if r.snapResolver != nil {
		if asset, err := r.snapResolver.ResolveIcon(ctx, canonID, pkgName, mode); err == nil && asset != nil {
			asset.Repository = "aur_upstream_snap"
			return asset, nil
		}
	}

	return nil, fmt.Errorf("aur upstream icon not found for %s (no PKGBUILD execution)", appID)
}

// =========================================================================
// 5. Installed Desktop File Resolver (Priority 1 in Installed Mode, Priority 5 in Catalog)
// =========================================================================

type DesktopFileResolver struct {
	themeResolver *SystemThemeResolver
}

func NewDesktopFileResolver(themeResolver *SystemThemeResolver) *DesktopFileResolver {
	return &DesktopFileResolver{themeResolver: themeResolver}
}

func (r *DesktopFileResolver) Name() string { return "desktop_file" }

func (r *DesktopFileResolver) Priority(mode ResolutionMode) int {
	if mode == ModeInstalled {
		return 1
	}
	return 5
}

func (r *DesktopFileResolver) Supports(mode ResolutionMode, s string) bool {
	return true
}

func (r *DesktopFileResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)
	aliases := aliasReg.GetAliases(appID)

	home, _ := os.UserHomeDir()
	desktopDirs := []string{
		"/usr/share/applications",
		fmt.Sprintf("%s/.local/share/applications", home),
		"/var/lib/flatpak/exports/share/applications",
		fmt.Sprintf("%s/.local/share/flatpak/exports/share/applications", home),
		"/var/lib/snapd/desktop/applications",
	}

	candidates := append([]string{appID, pkgName, canonID}, aliases...)

	for _, dir := range desktopDirs {
		for _, name := range candidates {
			cleanName := strings.TrimSpace(name)
			if cleanName == "" {
				continue
			}

			desktopFile := filepath.Join(dir, fmt.Sprintf("%s.desktop", cleanName))
			iconVal, err := r.extractIconField(desktopFile)
			if err == nil && iconVal != "" {
				// If absolute path
				if filepath.IsAbs(iconVal) {
					if data, err := os.ReadFile(iconVal); err == nil && len(data) > 0 {
						return &IconAsset{
							AppID:      canonID,
							Source:     "desktop_file",
							Repository: "system_installed",
							Data:       data,
							LocalPath:  iconVal,
							IsLocal:    true,
						}, nil
					}
				}

				// Otherwise, look up via theme resolver
				if r.themeResolver != nil {
					if asset, err := r.themeResolver.lookupIconName(ctx, canonID, iconVal); err == nil && asset != nil {
						asset.Source = "desktop_file_theme"
						asset.Repository = "system_theme"
						return asset, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("desktop file not found for %s", appID)
}

func (r *DesktopFileResolver) extractIconField(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inDesktopEntry := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inDesktopEntry = (line == "[Desktop Entry]")
			continue
		}
		if inDesktopEntry && strings.HasPrefix(line, "Icon=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Icon=")), nil
		}
	}
	return "", fmt.Errorf("no Icon field found in %s", path)
}

// =========================================================================
// 6. System Icon Theme Resolver (Priority 4 in Installed Mode, Priority 6 in Catalog)
// =========================================================================

type SystemThemeResolver struct{}

func NewSystemThemeResolver() *SystemThemeResolver {
	return &SystemThemeResolver{}
}

func (r *SystemThemeResolver) Name() string { return "system_theme" }

func (r *SystemThemeResolver) Priority(mode ResolutionMode) int {
	if mode == ModeInstalled {
		return 4
	}
	return 6
}

func (r *SystemThemeResolver) Supports(mode ResolutionMode, s string) bool {
	return true
}

func (r *SystemThemeResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	aliasReg := GetAliasRegistry()
	canonID := aliasReg.ResolveCanonical(appID)
	aliases := aliasReg.GetAliases(appID)

	candidates := append([]string{appID, pkgName, canonID}, aliases...)

	for _, n := range candidates {
		if asset, err := r.lookupIconName(ctx, canonID, n); err == nil && asset != nil {
			return asset, nil
		}
	}
	return nil, fmt.Errorf("system theme icon not found for %s", appID)
}

func (r *SystemThemeResolver) lookupIconName(ctx context.Context, appID, iconName string) (*IconAsset, error) {
	cleanName := strings.TrimSpace(iconName)
	if cleanName == "" {
		return nil, fmt.Errorf("empty icon name")
	}

	home, _ := os.UserHomeDir()
	themeBases := []string{
		"/usr/share/icons/hicolor",
		"/usr/share/pixmaps",
		fmt.Sprintf("%s/.local/share/icons/hicolor", home),
		"/usr/share/icons/Adwaita",
		"/usr/share/icons/breeze",
		"/usr/share/icons/Papirus",
	}

	sizes := []string{"128x128", "scalable", "256x256", "64x64", "48x48"}
	extensions := []string{".png", ".svg", ".svgz", ".xpm"}

	for _, base := range themeBases {
		// If direct directory like /usr/share/pixmaps
		if strings.HasSuffix(base, "pixmaps") {
			for _, ext := range extensions {
				p := filepath.Join(base, cleanName+ext)
				if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
					if ext == ".svgz" {
						if uncompressed, err := SafeDecompressGzip(data, MaxIconSizeBytes); err == nil {
							data = uncompressed
						}
					}
					return &IconAsset{
						AppID:      appID,
						Source:     "system_pixmap",
						Repository: "system_theme",
						Data:       data,
						LocalPath:  p,
						IsLocal:    true,
					}, nil
				}
			}
			continue
		}

		// Icon theme directories
		for _, sz := range sizes {
			subDir := "apps"
			for _, ext := range extensions {
				p := filepath.Join(base, sz, subDir, cleanName+ext)
				if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
					if ext == ".svgz" {
						if uncompressed, err := SafeDecompressGzip(data, MaxIconSizeBytes); err == nil {
							data = uncompressed
						}
					}
					return &IconAsset{
						AppID:      appID,
						Source:     "system_theme",
						Repository: "system_theme",
						Data:       data,
						LocalPath:  p,
						IsLocal:    true,
					}, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("icon name %s not found in theme paths", cleanName)
}

// =========================================================================
// 7. AppImage Resolver (Priority 5 in Installed Mode, Priority 7 in Catalog)
// =========================================================================

type AppImageResolver struct{}

func NewAppImageResolver() *AppImageResolver {
	return &AppImageResolver{}
}

func (r *AppImageResolver) Name() string { return "appimage" }

func (r *AppImageResolver) Priority(mode ResolutionMode) int {
	if mode == ModeInstalled {
		return 5
	}
	return 7
}

func (r *AppImageResolver) Supports(mode ResolutionMode, s string) bool {
	return s == "appimage" || s == ""
}

// ResolveIcon inspects pre-extracted icons without executing untrusted external binaries
func (r *AppImageResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	home, _ := os.UserHomeDir()
	candidates := []string{
		fmt.Sprintf("%s/.local/share/icons/hicolor/128x128/apps/appimagekit_%s.png", home, appID),
		fmt.Sprintf("%s/.local/share/icons/hicolor/scalable/apps/appimagekit_%s.svg", home, appID),
		fmt.Sprintf("%s/.cache/appimage-run/%s.DirIcon", home, appID),
	}

	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil && len(data) > 0 {
			return &IconAsset{
				AppID:      appID,
				Source:     "appimage_integration",
				Repository: "appimage",
				Data:       data,
				LocalPath:  p,
				IsLocal:    true,
			}, nil
		}
	}

	return nil, fmt.Errorf("appimage icon not found for %s", appID)
}

// =========================================================================
// 8. Generic Fallback Resolver (Priority 99 - Ultimate Fallback)
// =========================================================================

type FallbackResolver struct{}

func NewFallbackResolver() *FallbackResolver {
	return &FallbackResolver{}
}

func (r *FallbackResolver) Name() string { return "fallback" }

func (r *FallbackResolver) Priority(mode ResolutionMode) int {
	return 99
}

func (r *FallbackResolver) Supports(mode ResolutionMode, s string) bool {
	return true
}

func (r *FallbackResolver) ResolveIcon(ctx context.Context, appID, pkgName string, mode ResolutionMode) (*IconAsset, error) {
	// Generates a modern Windows 11 Fluent dark acrylic grid app tile (NO tickmark!)
	svg := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="128" height="128">
  <defs>
    <linearGradient id="bgGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" stop-color="#0078d4" stop-opacity="0.85" />
      <stop offset="100%%" stop-color="#2563eb" stop-opacity="0.95" />
    </linearGradient>
    <linearGradient id="tileGrad" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
      <stop offset="0%%" stop-color="#ffffff" stop-opacity="0.35" />
      <stop offset="100%%" stop-color="#ffffff" stop-opacity="0.1" />
    </linearGradient>
  </defs>
  <rect width="128" height="128" rx="28" fill="url(#bgGrad)" />
  <rect x="2" y="2" width="124" height="124" rx="26" fill="none" stroke="rgba(255,255,255,0.25)" stroke-width="2" />
  <!-- Modern 4-quadrant software application tile (NO tickmark) -->
  <rect x="36" y="36" width="24" height="24" rx="6" fill="url(#tileGrad)" />
  <rect x="68" y="36" width="24" height="24" rx="6" fill="url(#tileGrad)" />
  <rect x="36" y="68" width="24" height="24" rx="6" fill="url(#tileGrad)" />
  <rect x="68" y="68" width="24" height="24" rx="6" fill="url(#tileGrad)" />
</svg>`)

	return &IconAsset{
		AppID:      appID,
		Source:     "fallback",
		Repository: "system_fallback",
		Format:     "svg",
		Width:      128,
		Height:     128,
		Data:       []byte(svg),
		IsLocal:    true,
	}, nil
}
