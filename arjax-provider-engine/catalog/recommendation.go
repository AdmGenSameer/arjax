package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

// RecommendationEngine scores, filters, and explains package offers
type RecommendationEngine struct {
	config CatalogConfig
}

// NewRecommendationEngine initializes a recommendation engine with the provided config
func NewRecommendationEngine(cfg CatalogConfig) *RecommendationEngine {
	return &RecommendationEngine{
		config: cfg,
	}
}

// CheckEligibility tests whether an offer can be installed on the current host system
func (r *RecommendationEngine) CheckEligibility(offer *PackageOffer) {
	prov := strings.ToLower(offer.Provider)

	switch prov {
	case "flatpak":
		if !utils.CommandExists("flatpak") {
			offer.IsEligible = false
			offer.IneligibleReason = "Flatpak runtime CLI is not installed on this system"
			offer.Score = 0
			return
		}
	case "snap":
		if !utils.CommandExists("snap") {
			offer.IsEligible = false
			offer.IneligibleReason = "Snapd service is not installed or enabled on this system"
			offer.Score = 0
			return
		}
	case "aur":
		if !utils.CommandExists("paru") && !utils.CommandExists("yay") && !utils.CommandExists("makepkg") {
			offer.IsEligible = false
			offer.IneligibleReason = "No AUR helper (paru/yay) or makepkg found on system"
			offer.Score = 0
			return
		}
	case "pacman":
		if !utils.CommandExists("pacman") {
			offer.IsEligible = false
			offer.IneligibleReason = "Pacman package manager not detected on current host"
			offer.Score = 0
			return
		}
	}

	offer.IsEligible = true
	offer.IneligibleReason = ""
}

// CalculateScore computes a ranking score (0.0 - 100.0) based on provider trust, sandbox safety, and policy
func (r *RecommendationEngine) CalculateScore(offer *PackageOffer) float64 {
	if !offer.IsEligible {
		return 0.0
	}

	prov := strings.ToLower(offer.Provider)
	src := strings.ToLower(offer.Source)

	var baseScore float64

	switch r.config.Policy {
	case PolicySandboxPreferred:
		switch prov {
		case "flatpak":
			baseScore = 90.0
			if src == "flathub" {
				baseScore += 5.0
			}
		case "snap":
			baseScore = 80.0
		case "pacman":
			baseScore = 70.0
		case "aur":
			baseScore = 50.0
		default:
			baseScore = 40.0
		}

	case PolicyLatestVersion:
		switch prov {
		case "aur":
			baseScore = 85.0 // AUR often tracks -git or newest releases first
		case "flatpak":
			baseScore = 80.0
		case "pacman":
			baseScore = 75.0
		case "snap":
			baseScore = 70.0
		default:
			baseScore = 40.0
		}

	case PolicyNativePreferred:
		fallthrough
	default:
		switch prov {
		case "pacman":
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
			baseScore = 60.0
			if strings.HasSuffix(offer.PackageName, "-bin") {
				baseScore += 5.0 // Precompiled binary preferred over compile-from-source
			}
		default:
			baseScore = 45.0
		}
	}

	// Installed bonus: If currently installed, retain high score to prevent confusing the user
	if offer.IsInstalled {
		baseScore += 5.0
	}

	if baseScore > 100.0 {
		baseScore = 100.0
	}
	return baseScore
}

// GenerateReason creates a human-readable justification for the offer's score and badge
func (r *RecommendationEngine) GenerateReason(offer *PackageOffer, isTopRanked bool) (string, string) {
	prov := strings.ToLower(offer.Provider)
	src := strings.ToLower(offer.Source)

	if offer.IsInstalled {
		return "Installed", fmt.Sprintf("Currently installed on host (%s).", offer.InstalledVersion)
	}

	if !offer.IsEligible {
		return "Unavailable", offer.IneligibleReason
	}

	if isTopRanked {
		switch prov {
		case "pacman":
			return "Recommended", fmt.Sprintf("Official Arch repository package (%s) with native hardware performance and system integration.", src)
		case "flatpak":
			return "Recommended", "Verified Flathub container with filesystem sandbox isolation and bundled dependencies."
		case "snap":
			return "Recommended", "Canonical Snap package with strict sandbox confinement."
		case "aur":
			return "Recommended", "Community AUR package build configured for Arch Linux."
		default:
			return "Recommended", "Top eligible package provider for this application."
		}
	}

	// Alternative offers
	switch prov {
	case "flatpak":
		return "Sandboxed", "Flathub application container with isolated user permissions."
	case "pacman":
		return "Native", fmt.Sprintf("Native Arch distribution package from [%s].", src)
	case "aur":
		if strings.HasSuffix(offer.PackageName, "-bin") {
			return "Community (Bin)", "Community precompiled binary release from the Arch User Repository."
		}
		return "Community (Source)", "Community source build requiring local compilation via makepkg."
	case "snap":
		return "Snap Package", "Strictly confined snap package from Canonical Snapcraft."
	default:
		return "Alternative", fmt.Sprintf("Alternative package format provided by %s.", prov)
	}
}

// RankAndAnnotate evaluates all offers for an application and sets recommendations
func (r *RecommendationEngine) RankAndAnnotate(app *UnifiedApplication, userPreferredOfferID string) {
	if len(app.Offers) == 0 {
		return
	}

	// 1. Evaluate eligibility and calculate score for each offer
	for i := range app.Offers {
		r.CheckEligibility(&app.Offers[i])
		app.Offers[i].Score = r.CalculateScore(&app.Offers[i])
	}

	// 2. Sort offers: eligible higher score first, ineligible last
	sort.SliceStable(app.Offers, func(i, j int) bool {
		if app.Offers[i].IsEligible != app.Offers[j].IsEligible {
			return app.Offers[i].IsEligible
		}
		return app.Offers[i].Score > app.Offers[j].Score
	})

	// 3. Annotate badges and reasons
	for i := range app.Offers {
		isTop := (i == 0 && app.Offers[i].IsEligible)
		badge, reason := r.GenerateReason(&app.Offers[i], isTop)
		app.Offers[i].RecommendationBadge = badge
		app.Offers[i].RecommendationReason = reason
	}

	// 4. Determine default offer (respecting user preference if specified)
	selectedID := ""
	if userPreferredOfferID != "" {
		for _, o := range app.Offers {
			if o.ID == userPreferredOfferID && o.IsEligible {
				selectedID = o.ID
				break
			}
		}
	}

	// Fallback to installed offer if no user override
	if selectedID == "" && app.IsInstalled && app.InstalledOfferID != "" {
		selectedID = app.InstalledOfferID
	}

	// Fallback to highest ranked eligible offer
	if selectedID == "" && len(app.Offers) > 0 {
		selectedID = app.Offers[0].ID
	}

	app.DefaultOfferID = selectedID
}

