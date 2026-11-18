package catalog

import (
	"context"
	"strings"
	"testing"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/appstream"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
)

// TestCanonicalIdentityAndAliases verifies that package aliases correctly resolve to reverse-DNS AppStream IDs
func TestCanonicalIdentityAndAliases(t *testing.T) {
	deduper := NewDeduplicator()

	tests := []struct {
		input    string
		expected string
	}{
		{"firefox", "org.mozilla.firefox"},
		{"firefox-bin", "org.mozilla.firefox"},
		{"org.mozilla.firefox", "org.mozilla.firefox"},
		{"steam", "com.valvesoftware.Steam"},
		{"steam-native", "com.valvesoftware.Steam"},
		{"com.valvesoftware.Steam", "com.valvesoftware.Steam"},
		{"blender", "org.blender.Blender"},
		{"vlc", "org.videolan.VLC"},
		{"spotify", "com.spotify.Client"},
		{"brave-bin", "com.brave.Browser"},
		{"unknown-cli-tool", "unknown-cli-tool"},
	}

	for _, tt := range tests {
		actual := deduper.CanonicalizeID(tt.input)
		if !strings.EqualFold(actual, tt.expected) {
			t.Errorf("CanonicalizeID(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}

// TestMultiSourceGroupingAndDeduplication tests grouping Firefox native, flatpak, and aur packages
func TestMultiSourceGroupingAndDeduplication(t *testing.T) {
	deduper := NewDeduplicator()

	pkgs := []models.Package{
		{
			Name:        "firefox",
			Version:     "124.0.1-1",
			Description: "Fast, private and safe web browser from Mozilla",
			Provider:    "pacman",
			Repo:        "extra",
			Size:        "78 MB",
		},
		{
			Name:        "firefox-bin",
			Version:     "124.0.1",
			Description: "Precompiled official Firefox browser release from Mozilla",
			Provider:    "aur",
			Repo:        "aur",
			Size:        "82 MB",
		},
		{
			Name:        "firefox", // Duplicate pacman entry from mirror sync
			Version:     "124.0.1-1",
			Description: "Fast, private and safe web browser from Mozilla",
			Provider:    "pacman",
			Repo:        "extra",
			Size:        "78 MB",
		},
	}

	crawlerMap := map[string]*appstream.AppstreamMetadata{
		"org.mozilla.firefox": {
			ID:          "org.mozilla.firefox",
			FlathubID:   "org.mozilla.firefox",
			Name:        "Firefox Web Browser",
			Summary:     "Fast, Private & Free Web Browser",
			Description: "Mozilla Firefox is a free and open-source web browser.",
			Developer:   "Mozilla",
			Homepage:    "https://www.mozilla.org/firefox",
			License:     "MPL-2.0",
			Icon:        "https://dl.flathub.org/media/icons/128x128/org.mozilla.firefox.png",
		},
	}

	installedMap := map[string]string{
		"firefox": "124.0.1-1", // Native package is installed
	}

	unified := deduper.GroupAndDeduplicate(pkgs, crawlerMap, installedMap)

	if len(unified) != 1 {
		t.Fatalf("Expected 1 unified application for Firefox, got %d", len(unified))
	}

	app := unified[0]
	if !strings.EqualFold(app.AppID, "org.mozilla.firefox") {
		t.Errorf("Expected AppID 'org.mozilla.firefox', got %q", app.AppID)
	}

	// Verify duplicate offer elimination:
	// We had 2 pacman entries, 1 aur entry, and 1 flatpak entry -> should have exactly 3 distinct offers!
	if len(app.Offers) != 3 {
		t.Fatalf("Expected 3 distinct offers (pacman, aur, flatpak), got %d", len(app.Offers))
	}

	// Verify installation state: Native pacman offer must be installed, Flatpak must NOT be installed
	if !app.IsInstalled {
		t.Errorf("Expected app.IsInstalled to be true")
	}

	var pacmanOffer, flatpakOffer, aurOffer *PackageOffer
	for i := range app.Offers {
		switch app.Offers[i].Provider {
		case "pacman":
			pacmanOffer = &app.Offers[i]
		case "flatpak":
			flatpakOffer = &app.Offers[i]
		case "aur":
			aurOffer = &app.Offers[i]
		}
	}

	if pacmanOffer == nil || !pacmanOffer.IsInstalled {
		t.Errorf("Expected pacmanOffer to be installed")
	}
	if flatpakOffer == nil || flatpakOffer.IsInstalled {
		t.Errorf("Expected flatpakOffer to NOT be installed")
	}
	if aurOffer == nil || aurOffer.IsInstalled {
		t.Errorf("Expected aurOffer to NOT be installed")
	}
}

// TestRecommendationEngineRanking tests policy evaluation and ranking
func TestRecommendationEngineRanking(t *testing.T) {
	nativeEngine := NewRecommendationEngine(CatalogConfig{
		Policy: PolicyNativePreferred,
	})

	sandboxEngine := NewRecommendationEngine(CatalogConfig{
		Policy: PolicySandboxPreferred,
	})

	app := UnifiedApplication{
		AppID: "org.videolan.VLC",
		Name:  "VLC media player",
		Offers: []PackageOffer{
			{
				ID:          "aur:aur:vlc-git",
				PackageName: "vlc-git",
				Provider:    "aur",
				Source:      "aur",
				IsEligible:  true,
			},
			{
				ID:          "flatpak:flathub:org.videolan.VLC",
				PackageName: "org.videolan.VLC",
				Provider:    "flatpak",
				Source:      "flathub",
				IsEligible:  true,
			},
			{
				ID:          "pacman:extra:vlc",
				PackageName: "vlc",
				Provider:    "pacman",
				Source:      "extra",
				IsEligible:  true,
			},
		},
	}

	// 1. Native preferred policy
	nativeApp := app
	nativeEngine.RankAndAnnotate(&nativeApp, "")
	if len(nativeApp.Offers) == 0 {
		t.Fatalf("Expected offers to be ranked")
	}
	if nativeApp.Offers[0].Provider != "pacman" {
		t.Errorf("Under PolicyNativePreferred, expected top offer to be pacman, got %s", nativeApp.Offers[0].Provider)
	}
	if nativeApp.Offers[0].RecommendationBadge != "Recommended" {
		t.Errorf("Expected top offer to have badge 'Recommended', got %s", nativeApp.Offers[0].RecommendationBadge)
	}

	// 2. Sandbox preferred policy
	sandboxApp := app
	sandboxEngine.RankAndAnnotate(&sandboxApp, "")
	if sandboxApp.Offers[0].Provider != "flatpak" {
		t.Errorf("Under PolicySandboxPreferred, expected top offer to be flatpak, got %s", sandboxApp.Offers[0].Provider)
	}

	// 3. User override preservation
	userApp := app
	nativeEngine.RankAndAnnotate(&userApp, "aur:aur:vlc-git")
	if userApp.DefaultOfferID != "aur:aur:vlc-git" {
		t.Errorf("Expected DefaultOfferID to preserve user choice 'aur:aur:vlc-git', got %s", userApp.DefaultOfferID)
	}
}

// TestProgressiveSearchCancellation tests cancellation context during fanned-out search
func TestProgressiveSearchCancellation(t *testing.T) {
	deduper := NewDeduplicator()
	recommender := NewRecommendationEngine(DefaultCatalogConfig())
	searchEngine := NewProgressiveSearchEngine(nil, nil, deduper, recommender, nil)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately to test graceful termination
	cancel()

	doneCalled := false
	searchEngine.StreamSearch(ctx, "test-cancel-id", "firefox", func(batch ProgressiveSearchBatchDTO) {
		t.Errorf("Unexpected batch emitted after cancellation")
	}, func(done ProgressiveSearchDoneDTO) {
		doneCalled = true
	})

	if !doneCalled {
		t.Errorf("Expected doneFn to be called even on immediate cancellation")
	}
}

// TestProviderIneligibility tests handling when a provider binary is missing
func TestProviderIneligibility(t *testing.T) {
	engine := NewRecommendationEngine(DefaultCatalogConfig())

	ineligibleOffer := PackageOffer{
		ID:          "snap:snapcraft:testapp",
		PackageName: "testapp",
		Provider:    "snap",
		Source:      "snapcraft",
	}

	engine.CheckEligibility(&ineligibleOffer)
	// If snap binary does not exist on machine, it must be marked ineligible
	score := engine.CalculateScore(&ineligibleOffer)
	if !ineligibleOffer.IsEligible && score != 0 {
		t.Errorf("Ineligible offer must have score 0, got %f", score)
	}
}

// TestDuplicateRepoMetadataElimination tests deduplication of duplicate offers from repository mirrors
func TestDuplicateRepoMetadataElimination(t *testing.T) {
	deduper := NewDeduplicator()

	pkgs := []models.Package{
		{
			Name:        "neovim",
			Version:     "0.10.0-1",
			Description: "Vim-fork focused on extensibility and usability",
			Provider:    "pacman",
			Repo:        "extra",
			Size:        "24 MB",
		},
		{
			Name:        "neovim", // Same package from another repo/mirror with identical version
			Version:     "0.10.0-1",
			Description: "Vim-fork focused on extensibility and usability",
			Provider:    "pacman",
			Repo:        "extra",
			Size:        "24 MB",
		},
		{
			Name:        "neovim", // Flathub or alternative provider
			Version:     "0.10.0",
			Description: "Neovim desktop container",
			Provider:    "flatpak",
			Repo:        "flathub",
			Size:        "45 MB",
		},
	}

	apps := deduper.GroupAndDeduplicate(pkgs, nil, nil)
	if len(apps) != 1 {
		t.Fatalf("Expected 1 unified application, got %d", len(apps))
	}

	app := apps[0]
	// Must have exactly 2 distinct offers (1 pacman, 1 flatpak); the duplicate pacman entry eliminated
	if len(app.Offers) != 2 {
		t.Errorf("Expected 2 distinct offers after deduplicating duplicate repo mirror, got %d", len(app.Offers))
	}
}

// TestSourceSelectionNoSilentMigration ensures selecting a different source does not silently mark it installed
func TestSourceSelectionNoSilentMigration(t *testing.T) {
	deduper := NewDeduplicator()
	engine := NewRecommendationEngine(DefaultCatalogConfig())

	pkgs := []models.Package{
		{
			Name:     "gimp",
			Version:  "2.10.38-1",
			Provider: "pacman",
			Repo:     "extra",
		},
		{
			Name:     "org.gimp.GIMP",
			Version:  "2.10.38",
			Provider: "flatpak",
			Repo:     "flathub",
		},
	}

	// Pacman is installed on system, Flatpak is NOT installed
	installedMap := map[string]string{
		"gimp": "2.10.38-1",
	}

	apps := deduper.GroupAndDeduplicate(pkgs, nil, installedMap)
	if len(apps) != 1 {
		t.Fatalf("Expected 1 app, got %d", len(apps))
	}

	app := apps[0]
	// User explicitly selects the Flatpak offer
	flatpakOfferID := BuildOfferID("flatpak", "flathub", "org.gimp.GIMP", "2.10.38")
	engine.RankAndAnnotate(&app, flatpakOfferID)

	if app.DefaultOfferID != flatpakOfferID {
		t.Errorf("Expected user selected offer to be default, got %s", app.DefaultOfferID)
	}

	// Find flatpak offer in app.Offers
	for _, o := range app.Offers {
		if o.ID == flatpakOfferID {
			if o.IsInstalled {
				t.Errorf("Flatpak offer must NOT be marked installed when only native package is installed!")
			}
		}
		if o.Provider == "pacman" {
			if !o.IsInstalled {
				t.Errorf("Pacman offer MUST remain marked installed!")
			}
		}
	}
}
