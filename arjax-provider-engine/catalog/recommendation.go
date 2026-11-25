package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

// RecommendationEngine evaluates eligibility, trust compliance, version recency,
// dependency footprint, and policy preferences separately from isolation/trust storage.
type RecommendationEngine struct {
	config       CatalogConfig
	capabilities map[string]providers.ProviderCapability
}

// NewRecommendationEngine initializes a recommendation engine with host capability introspection.
func NewRecommendationEngine(cfg CatalogConfig) *RecommendationEngine {
	return &RecommendationEngine{
		config:       cfg,
		capabilities: providers.DefaultCapabilityMatrix(),
	}
}

// NewRecommendationEngineWithCapabilities constructs an engine with explicit capabilities (useful in tests).
func NewRecommendationEngineWithCapabilities(cfg CatalogConfig, caps map[string]providers.ProviderCapability) *RecommendationEngine {
	if caps == nil {
		caps = providers.DefaultCapabilityMatrix()
	}
	return &RecommendationEngine{
		config:       cfg,
		capabilities: caps,
	}
}

// CheckEligibility verifies both host provider capabilities and active policy constraints.
func (r *RecommendationEngine) CheckEligibility(offer *PackageOffer) {
	prov := strings.ToLower(strings.TrimSpace(offer.Provider))
	capKey := prov
	switch prov {
	case "apt":
		capKey = "deb"
	case "dnf":
		capKey = "rpm"
	}

	// 1. Check host provider capabilities
	if capInfo, ok := r.capabilities[capKey]; ok {
		if eligible, reason := capInfo.CanPerform("install"); !eligible {
			offer.IsEligible = false
			offer.IneligibleReason = reason
			offer.Score = 0
			return
		}
	}

	// 2. Enforce OfficialOnly policy constraint
	if r.config.Policy == PolicyOfficialOnly {
		if offer.Trust.TrustLevel != TrustOfficial && offer.Trust.TrustLevel != TrustVerified {
			offer.IsEligible = false
			offer.IneligibleReason = "Excluded by Official Sources Only policy (community or unverified origin)"
			offer.Score = 0
			return
		}
	}

	offer.IsEligible = true
	offer.IneligibleReason = ""
}

// CalculateScore computes a ranking score (0.0 - 100.0) given an offer and the highest known version among peer offers.
func (r *RecommendationEngine) CalculateScore(offer *PackageOffer) float64 {
	return r.calculateScoreWithContext(offer, "")
}

func (r *RecommendationEngine) calculateScoreWithContext(offer *PackageOffer, highestVersion string) float64 {
	if !offer.IsEligible {
		return 0.0
	}

	prov := strings.ToLower(offer.Provider)
	src := strings.ToLower(offer.Repository)
	if src == "" {
		src = strings.ToLower(offer.Source)
	}

	var baseScore float64

	switch r.config.Policy {
	case PolicyOfficialOnly:
		if offer.Trust.OfficialRepo {
			baseScore = 95.0
		} else if offer.Trust.Verified {
			baseScore = 88.0
		} else {
			return 0.0
		}

	case PolicySandboxPreferred:
		switch prov {
		case "flatpak":
			baseScore = 90.0
			if src == "flathub" {
				baseScore += 5.0
			}
		case "snap":
			baseScore = 82.0
		case "pacman", "apt", "dnf":
			baseScore = 70.0
		case "aur":
			baseScore = 45.0
		default:
			baseScore = 40.0
		}

	case PolicyLatestVersion:
		baseScore = 70.0
		if highestVersion != "" && offer.Version != "" && !strings.EqualFold(offer.Version, "latest") {
			cmp := CompareVersions(prov, offer.Version, highestVersion)
			if cmp == 0 {
				baseScore = 95.0 // Highest compatible release
			} else if cmp < 0 {
				baseScore = 72.0
			}
		} else if prov == "aur" {
			baseScore = 82.0
		}

	case PolicyMinimalDeps:
		// Lower extra dependency cost yields a higher score
		baseScore = 92.0 - float64(offer.DependencyCount*4)
		if prov == "pacman" && offer.Trust.OfficialRepo {
			baseScore += 5.0
		}
		if baseScore < 30.0 {
			baseScore = 30.0
		}

	case PolicyNativePreferred:
		switch prov {
		case "pacman", "apt", "dnf":
			baseScore = 90.0
			if src == "core" || src == "extra" {
				baseScore += 5.0
			}
		case "flatpak":
			baseScore = 80.0
			if src == "flathub" {
				baseScore += 3.0
			}
		case "snap":
			baseScore = 70.0
		case "aur":
			baseScore = 58.0
			if strings.HasSuffix(offer.PackageName, "-bin") {
				baseScore += 5.0
			}
		default:
			baseScore = 45.0
		}

	case PolicyRecommended:
		fallthrough
	default:
		// PolicyRecommended weights trust + compatibility + sandbox balance without collapsing them into one property
		switch prov {
		case "pacman", "apt", "dnf":
			baseScore = 88.0
			if offer.Trust.OfficialRepo {
				baseScore += 6.0
			}
		case "flatpak":
			baseScore = 85.0
			if offer.Trust.Verified {
				baseScore += 4.0
			}
		case "snap":
			baseScore = 74.0
		case "aur":
			baseScore = 54.0
			if strings.HasSuffix(offer.PackageName, "-bin") {
				baseScore += 4.0
			}
		default:
			baseScore = 45.0
		}
	}

	if offer.IsInstalled {
		baseScore += 4.0
	}

	if baseScore > 100.0 {
		baseScore = 100.0
	}
	return baseScore
}

// GenerateReason produces a badge and clear rationale covering recommendation, trust, and isolation distinctions.
func (r *RecommendationEngine) GenerateReason(offer *PackageOffer, isTopRanked bool) (string, string) {
	prov := strings.ToLower(offer.Provider)
	src := strings.ToLower(offer.Repository)
	if src == "" {
		src = strings.ToLower(offer.Source)
	}

	if offer.IsInstalled {
		return "Installed", fmt.Sprintf("Currently installed via %s (%s). Trust: %s.", strings.ToUpper(prov), offer.InstalledVersion, offer.Trust.TrustLevel)
	}

	if !offer.IsEligible {
		return "Unavailable", offer.IneligibleReason
	}

	if isTopRanked {
		switch prov {
		case "pacman", "apt", "dnf":
			return "Recommended", fmt.Sprintf("Signed official repository (%s) with native system performance.", src)
		case "flatpak":
			return "Recommended", "Verified Flathub runtime with sandboxed filesystem isolation."
		case "snap":
			return "Recommended", "Verified Snap package with strict sandbox confinement."
		case "aur":
			return "Recommended", "Highest matched package for current policy; inspect PKGBUILD before building."
		default:
			return "Recommended", "Top eligible package offer for current recommendation policy."
		}
	}

	switch prov {
	case "flatpak":
		return "Sandboxed", "Verified Flathub container isolated from core host libraries."
	case "pacman", "apt", "dnf":
		return "Official", fmt.Sprintf("Signed official distribution package from [%s].", src)
	case "aur":
		if strings.HasSuffix(offer.PackageName, "-bin") {
			return "Community (Bin)", "Community-maintained AUR PKGBUILD repackaging upstream binaries."
		}
		return "Community (Source)", "Community AUR build recipe compiled locally via makepkg."
	case "snap":
		return "Sandboxed", "Canonical Snap package from Snapcraft store."
	default:
		return "Alternative", fmt.Sprintf("Alternative format provided by %s (%s trust).", prov, offer.Trust.TrustLevel)
	}
}

// RankAndAnnotate scores, sorts, and annotates offers while strictly honoring user preference.
func (r *RecommendationEngine) RankAndAnnotate(app *UnifiedApplication, userPreferredOfferID string) {
	if len(app.Offers) == 0 {
		return
	}

	// Ensure Trust fields are always populated even for directly synthesized test offers
	for i := range app.Offers {
		if app.Offers[i].Trust.TrustLevel == "" {
			repo := app.Offers[i].Repository
			if repo == "" {
				repo = app.Offers[i].Source
			}
			t := EvaluateSourceTrust(app.Offers[i].Provider, repo, app.Offers[i].PackageName)
			app.Offers[i].Trust = t
			app.Offers[i].TrustLevel = t.TrustLevel
			app.Offers[i].SignatureStatus = t.SignatureStatus
		}
		if app.Offers[i].Scope == "" {
			app.Offers[i].Scope = app.Offers[i].InstallScope
		}
		r.CheckEligibility(&app.Offers[i])
	}

	// Compute highest comparable numeric version across eligible offers for PolicyLatestVersion,
	// avoiding blind cross-ecosystem comparisons of channels ("stable") or conflicting distro epochs.
	highestVersion := ""
	highestProv := ""
	for _, o := range app.Offers {
		if !o.IsEligible || IsReleaseTrack(o.Version) {
			continue
		}
		if highestVersion == "" {
			highestVersion = o.Version
			highestProv = o.Provider
			continue
		}
		if cmp, err := CompareCrossEcosystem(o.Provider, o.Version, highestProv, highestVersion); err == nil && cmp > 0 {
			highestVersion = o.Version
			highestProv = o.Provider
		}
	}

	for i := range app.Offers {
		app.Offers[i].Score = r.calculateScoreWithContext(&app.Offers[i], highestVersion)
	}

	// Deterministic sort: eligible first, higher score first, then stable offer ID
	sort.SliceStable(app.Offers, func(i, j int) bool {
		if app.Offers[i].IsEligible != app.Offers[j].IsEligible {
			return app.Offers[i].IsEligible
		}
		if app.Offers[i].Score != app.Offers[j].Score {
			return app.Offers[i].Score > app.Offers[j].Score
		}
		return app.Offers[i].ID < app.Offers[j].ID
	})

	for i := range app.Offers {
		isTop := i == 0 && app.Offers[i].IsEligible
		badge, reason := r.GenerateReason(&app.Offers[i], isTop)
		app.Offers[i].RecommendationBadge = badge
		app.Offers[i].RecommendationReason = reason
	}

	// Preserve manual user override if still eligible
	selectedID := ""
	if userPreferredOfferID != "" {
		for _, o := range app.Offers {
			if o.ID == userPreferredOfferID && o.IsEligible {
				selectedID = o.ID
				break
			}
		}
	}

	if selectedID == "" && app.IsInstalled && app.InstalledOfferID != "" {
		selectedID = app.InstalledOfferID
	}

	if selectedID == "" && len(app.Offers) > 0 {
		selectedID = app.Offers[0].ID
	}

	app.DefaultOfferID = selectedID
}
