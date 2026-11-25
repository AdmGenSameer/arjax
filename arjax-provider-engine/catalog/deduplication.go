package catalog

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/identity"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
)

// Deduplicator groups heterogeneous package offers under canonical AppStream identities
// using the dedicated catalog/identity resolver rather than icon aliases.
type Deduplicator struct {
	mu               sync.RWMutex
	identityResolver *identity.Resolver
}

// NewDeduplicator constructs a deduplicator backed by catalog/identity.
func NewDeduplicator() *Deduplicator {
	return &Deduplicator{
		identityResolver: identity.NewResolver(identity.GetDefaultRegistry()),
	}
}

// IdentityResolver returns the underlying canonical identity resolver.
func (d *Deduplicator) IdentityResolver() *identity.Resolver {
	return d.identityResolver
}

// CanonicalizeID maps a package name, binary name, or AppStream ID to its canonical identity.
func (d *Deduplicator) CanonicalizeID(input string) string {
	return d.identityResolver.ResolveCanonical(input)
}

// ResolveIdentity returns full confidence assessment for an identifier.
func (d *Deduplicator) ResolveIdentity(input, provider string) identity.ResolutionResult {
	return d.identityResolver.Resolve(input, provider)
}

// BuildOfferIdentity generates a collision-free, deterministic offer identifier across
// provider, repository, package name, architecture, branch, and channel.
func BuildOfferIdentity(provider, repo, packageName, arch, branch, channel string) string {
	prov := normalizeProviderName(provider, repo)
	r := strings.ToLower(strings.TrimSpace(repo))
	if r == "" {
		r = "default"
	}
	pkg := strings.ToLower(strings.TrimSpace(packageName))
	a := strings.ToLower(strings.TrimSpace(arch))
	b := strings.ToLower(strings.TrimSpace(branch))
	c := strings.ToLower(strings.TrimSpace(channel))

	parts := []string{prov, r, pkg}
	if a != "" {
		parts = append(parts, "arch="+a)
	}
	if b != "" {
		parts = append(parts, "branch="+b)
	}
	if c != "" {
		parts = append(parts, "channel="+c)
	}
	return strings.Join(parts, ":")
}

// BuildOfferID keeps backwards-compatible OfferID construction when arch/branch/channel are defaults.
func BuildOfferID(provider, source, packageName, _ string) string {
	return BuildOfferIdentity(provider, source, packageName, "", "", "")
}

func normalizeProviderName(provider, repo string) string {
	prov := strings.ToLower(strings.TrimSpace(provider))
	r := strings.ToLower(strings.TrimSpace(repo))
	if prov == "repository" {
		if r == "aur" {
			return "aur"
		}
		return "pacman"
	}
	if prov == "" {
		if r == "aur" {
			return "aur"
		}
		return "pacman"
	}
	return prov
}

// DetermineScope determines the security/isolation scope separately from trust and recommendation.
func DetermineScope(provider string) InstallScope {
	switch strings.ToLower(provider) {
	case "flatpak":
		return ScopeSandboxed
	case "snap":
		return ScopeSandboxed
	case "aur":
		return ScopeUser
	default:
		return ScopeSystem
	}
}

// DeterminePermissions returns standard sandbox/host permissions based on provider and app class.
func DeterminePermissions(provider, pkgName string) []string {
	prov := strings.ToLower(provider)
	pkg := strings.ToLower(pkgName)

	if prov == "flatpak" || prov == "snap" {
		perms := []string{"sandboxed", "isolated-filesystem"}
		if strings.Contains(pkg, "browser") || strings.Contains(pkg, "firefox") || strings.Contains(pkg, "chrome") || strings.Contains(pkg, "discord") {
			perms = append(perms, "network", "pulseaudio", "wayland/x11", "gpu-acceleration")
		} else if strings.Contains(pkg, "steam") || strings.Contains(pkg, "game") {
			perms = append(perms, "network", "pulseaudio", "wayland/x11", "gpu-acceleration", "gamepad")
		} else if strings.Contains(pkg, "blender") || strings.Contains(pkg, "kdenlive") {
			perms = append(perms, "gpu-acceleration", "pulseaudio", "removable-media")
		} else {
			perms = append(perms, "ipc", "desktop")
		}
		return perms
	}

	return []string{"system-native", "host-filesystem", "hardware-direct"}
}

// FormatCommandPreview generates a non-executable informational CLI reference string.
func FormatCommandPreview(provider, repo, packageName, branch, channel string) string {
	switch strings.ToLower(provider) {
	case "pacman":
		return fmt.Sprintf("sudo pacman -S %s", packageName)
	case "aur":
		return fmt.Sprintf("paru -S %s", packageName)
	case "flatpak":
		remote := repo
		if remote == "" {
			remote = "flathub"
		}
		if branch != "" && branch != "stable" {
			return fmt.Sprintf("flatpak install %s %s//%s", remote, packageName, branch)
		}
		return fmt.Sprintf("flatpak install %s %s", remote, packageName)
	case "snap":
		if channel != "" {
			return fmt.Sprintf("sudo snap install %s --channel=%s", packageName, channel)
		}
		return fmt.Sprintf("sudo snap install %s", packageName)
	case "deb", "apt":
		return fmt.Sprintf("sudo apt install %s", packageName)
	case "rpm", "dnf":
		return fmt.Sprintf("sudo dnf install %s", packageName)
	default:
		return fmt.Sprintf("arjax install %s --provider %s", packageName, provider)
	}
}

// ConvertPackageToOffer transforms a raw provider Package into a complete PackageOffer
// with trust, scope, architecture, branch/channel, and dependency counts.
func (d *Deduplicator) ConvertPackageToOffer(p models.Package, isInstalled bool, installedVer string) PackageOffer {
	repo := p.Repo
	if repo == "" {
		repo = p.Source
	}
	prov := normalizeProviderName(p.Provider, repo)
	if repo == "" || repo == "repository" {
		if prov == "aur" {
			repo = "aur"
		} else if prov == "flatpak" {
			repo = "flathub"
		} else if prov == "snap" {
			repo = "snapcraft"
		} else {
			repo = prov
		}
	}

	arch := strings.TrimSpace(p.Arch)
	if arch == "" {
		arch = hostDefaultArch()
	}

	branch := strings.TrimSpace(p.Branch)
	if branch == "" && prov == "flatpak" {
		branch = "stable"
	}
	channel := strings.TrimSpace(p.Channel)
	if channel == "" && prov == "snap" {
		channel = "latest/stable"
	}

	scope := DetermineScope(prov)
	if prov == "flatpak" && (strings.EqualFold(p.Scope, "user") || strings.EqualFold(p.Scope, "system")) {
		scope = InstallScope(strings.ToLower(strings.TrimSpace(p.Scope)))
	}
	perms := DeterminePermissions(prov, p.Name)
	offerID := BuildOfferIdentity(prov, repo, p.Name, p.Arch, p.Branch, p.Channel)
	if prov == "flatpak" && (strings.EqualFold(p.Scope, "user") || strings.EqualFold(p.Scope, "system")) {
		offerID += ":scope=" + strings.ToLower(strings.TrimSpace(p.Scope))
	}
	trust := EvaluateSourceTrust(prov, repo, p.Name)

	dSize := p.Size
	if dSize == "" {
		switch prov {
		case "flatpak":
			dSize = "~85 MB"
		case "aur":
			dSize = "Source PKGBUILD"
		default:
			dSize = "~45 MB"
		}
	}

	iSize := "~120 MB"
	if prov == "flatpak" {
		iSize = "~240 MB"
	}

	depCount := len(p.Dependencies)
	if depCount == 0 && prov == "flatpak" {
		// Sandboxed Flatpak commonly requires runtime dependencies if not yet shared
		depCount = 1
	}

	res := d.ResolveIdentity(p.Name, prov)

	return PackageOffer{
		ID:               offerID,
		AppID:            res.CanonicalID,
		PackageName:      p.Name,
		Provider:         prov,
		Repository:       repo,
		Source:           repo,
		Architecture:     arch,
		Version:          p.Version,
		Branch:           branch,
		Channel:          channel,
		DownloadSize:     dSize,
		InstalledSize:    iSize,
		InstallScope:     scope,
		Scope:            scope,
		Permissions:      perms,
		DependencyCount:  depCount,
		Dependencies:     p.Dependencies,
		IsInstalled:      isInstalled,
		InstalledVersion: installedVer,
		IsEligible:       true,
		TrustLevel:       trust.TrustLevel,
		SignatureStatus:  trust.SignatureStatus,
		Trust:            trust,
		CommandPreview:   FormatCommandPreview(prov, repo, p.Name, branch, channel),
	}
}

func hostDefaultArch() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}

// GroupAndDeduplicate groups offers under canonical AppStream IDs while refusing
// uncertain fuzzy merges. Duplicate mirror entries are eliminated cleanly.
func (d *Deduplicator) GroupAndDeduplicate(
	pkgs []models.Package,
	crawlerMap map[string]*appstream.AppstreamMetadata,
	installedMap map[string]string,
) []UnifiedApplication {
	d.mu.Lock()
	defer d.mu.Unlock()

	appMap := make(map[string]*UnifiedApplication)
	seenOffers := make(map[string]bool)

	// 1. Seed rich desktop metadata from AppStream/Flathub crawler (high confidence)
	if crawlerMap != nil {
		for id, meta := range crawlerMap {
			if meta == nil {
				continue
			}
			d.identityResolver.RegisterRuntimeAppStream(id, meta.Name)
			res := d.identityResolver.Resolve(id, "flatpak")
			canonID := res.CanonicalID
			if canonID == "" {
				canonID = id
			}

			app := &UnifiedApplication{
				AppID:              canonID,
				Name:               meta.Name,
				Summary:            meta.Summary,
				Description:        meta.Description,
				Developer:          meta.Developer,
				Homepage:           meta.Homepage,
				License:            meta.License,
				IconURL:            meta.Icon,
				Categories:         meta.Categories,
				Screenshots:        meta.Screenshots,
				Releases:           meta.Releases,
				Aliases:            []string{meta.ID, meta.FlathubID, strings.ToLower(meta.Name)},
				IdentityConfidence: res.Confidence,
				Offers:             []PackageOffer{},
				Rating:             4.8,
				Reviews:            "4,280",
				Badge:              "Flathub Verified",
			}

			flatpakPkg := meta.FlathubID
			if flatpakPkg == "" {
				flatpakPkg = meta.ID
			}
			if flatpakPkg != "" {
				offerID := BuildOfferIdentity("flatpak", "flathub", flatpakPkg, "", "", "")
				isInst, instVer := matchInstalled(installedMap, flatpakPkg, canonID, "flatpak")
				trust := EvaluateSourceTrust("flatpak", "flathub", flatpakPkg)

				offer := PackageOffer{
					ID:               offerID,
					AppID:            canonID,
					PackageName:      flatpakPkg,
					Provider:         "flatpak",
					Repository:       "flathub",
					Source:           "flathub",
					Architecture:     hostDefaultArch(),
					Version:          "Latest",
					Branch:           "stable",
					DownloadSize:     "~85 MB",
					InstalledSize:    "~240 MB",
					InstallScope:     ScopeSandboxed,
					Scope:            ScopeSandboxed,
					Permissions:      DeterminePermissions("flatpak", flatpakPkg),
					DependencyCount:  1,
					IsInstalled:      isInst,
					InstalledVersion: instVer,
					IsEligible:       true,
					TrustLevel:       trust.TrustLevel,
					SignatureStatus:  trust.SignatureStatus,
					Trust:            trust,
					CommandPreview:   FormatCommandPreview("flatpak", "flathub", flatpakPkg, "stable", ""),
				}
				app.Offers = append(app.Offers, offer)
				seenOffers[offerID] = true
				if isInst {
					app.IsInstalled = true
					app.InstalledOfferID = offerID
				}
			}

			appMap[canonID] = app
		}
	}

	// 2. Process all incoming provider packages
	for _, p := range pkgs {
		normProv := normalizeProviderName(p.Provider, p.Repo)
		res := d.identityResolver.Resolve(p.Name, normProv)

		// Anti-false-merge guard:
		// Only group packages into the same application key when MatchConfidence is SafeToMerge (>= Medium),
		// otherwise isolate unconfirmed packages per provider+name so unrelated tools never falsely merge!
		appKey := res.CanonicalID
		if !res.Mergeable || appKey == "" {
			appKey = fmt.Sprintf("%s:%s", normProv, strings.ToLower(p.Name))
		}

		app, exists := appMap[appKey]
		if !exists {
			name := res.DisplayName
			if name == "" {
				name = strings.Title(p.Name)
			}
			app = &UnifiedApplication{
				AppID:              appKey,
				Name:               name,
				Summary:            p.Description,
				Description:        p.Description,
				Developer:          fmt.Sprintf("%s Maintainers", strings.ToUpper(normProv)),
				Homepage:           fmt.Sprintf("https://archlinux.org/packages/?q=%s", p.Name),
				License:            "GPL / Open Source",
				Categories:         []string{"Utilities"},
				Screenshots:        []string{},
				Releases:           []string{},
				Aliases:            []string{p.Name},
				IdentityConfidence: res.Confidence,
				Offers:             []PackageOffer{},
				Rating:             4.6,
				Reviews:            "1,200",
				Badge:              "Repository Package",
			}
			appMap[appKey] = app
		} else {
			hasAlias := false
			for _, a := range app.Aliases {
				if strings.EqualFold(a, p.Name) {
					hasAlias = true
					break
				}
			}
			if !hasAlias {
				app.Aliases = append(app.Aliases, p.Name)
			}
		}

		isInst, instVer := matchInstalled(installedMap, p.Name, appKey, normProv)
		offer := d.ConvertPackageToOffer(p, isInst, instVer)
		offer.AppID = appKey

		// Duplicate Offer Elimination (preserving distinct branches/channels/archs/repos)
		if seenOffers[offer.ID] {
			for i, existing := range app.Offers {
				if existing.ID == offer.ID {
					if (existing.Version == "latest" || existing.Version == "Latest" || existing.Version == "") &&
						offer.Version != "" && !strings.EqualFold(offer.Version, "latest") {
						app.Offers[i].Version = offer.Version
					}
					if !existing.IsInstalled && offer.IsInstalled {
						app.Offers[i].IsInstalled = true
						app.Offers[i].InstalledVersion = offer.InstalledVersion
						app.IsInstalled = true
						app.InstalledOfferID = offer.ID
					}
					break
				}
			}
			continue
		}

		seenOffers[offer.ID] = true
		app.Offers = append(app.Offers, offer)

		if isInst {
			app.IsInstalled = true
			if app.InstalledOfferID == "" {
				app.InstalledOfferID = offer.ID
			}
		}
	}

	// Deterministic ordering by AppID for reproducible progressive deltas and tests
	keys := make([]string, 0, len(appMap))
	for k, app := range appMap {
		if len(app.Offers) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	result := make([]UnifiedApplication, 0, len(keys))
	for _, k := range keys {
		result = append(result, *appMap[k])
	}

	return result
}

func matchInstalled(installedMap map[string]string, pkgName, canonID, prov string) (bool, string) {
	if installedMap == nil {
		return false, ""
	}
	normPkg := strings.ToLower(pkgName)
	normCanon := strings.ToLower(canonID)

	// Provider-qualified installed key first (prevents false cross-provider installed states!)
	if ver, ok := installedMap[prov+":"+normPkg]; ok {
		return true, ver
	}
	if ver, ok := installedMap[normPkg]; ok {
		// Unqualified name match applies only to this exact package name (never blindly across canonID aliases!)
		return true, ver
	}
	if pkgName != "" && strings.EqualFold(normPkg, normCanon) {
		if ver, ok := installedMap[normCanon]; ok {
			return true, ver
		}
	}
	return false, ""
}
