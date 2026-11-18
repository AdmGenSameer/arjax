package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
)

// TestApp_UnifiedAppDetails verifies multi-source offer aggregation on an App instance
func TestApp_UnifiedAppDetails(t *testing.T) {
	app := NewApp()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with Firefox
	details := app.GetUnifiedAppDetails("firefox", "")
	if details.AppID == "" {
		t.Fatalf("Expected non-empty AppID for firefox")
	}

	if !strings.EqualFold(details.AppID, "org.mozilla.firefox") {
		t.Errorf("Expected canonical AppID 'org.mozilla.firefox', got %s", details.AppID)
	}

	if len(details.Offers) == 0 {
		t.Errorf("Expected at least one package offer for firefox")
	}

	// Verify that user preference is honored
	if len(details.Offers) > 1 {
		secondOfferID := details.Offers[1].ID
		preferredDetails := app.GetUnifiedAppDetails("firefox", secondOfferID)
		if preferredDetails.DefaultOfferID != secondOfferID {
			t.Errorf("Expected DefaultOfferID to match user preferred offer %s, got %s", secondOfferID, preferredDetails.DefaultOfferID)
		}
	}

	_ = ctx
}

// TestApp_SearchUnifiedSync verifies that search deduplicates offers under unified applications
func TestApp_SearchUnifiedSync(t *testing.T) {
	app := NewApp()

	results := app.SearchUnifiedSync("blender")
	// If offline or live, Blender should return at least one unified app
	if len(results) > 0 {
		first := results[0]
		if first.AppID == "" {
			t.Errorf("Expected non-empty AppID in search result")
		}
		if len(first.Offers) == 0 {
			t.Errorf("Expected unified app to contain offers")
		}
	}
}

// TestApp_NoSilentMigration verifies that selecting an alternative offer does not alter installed states
func TestApp_NoSilentMigration(t *testing.T) {
	app := NewApp()

	// Synthesize a test app with one installed native offer and one uninstalled flatpak offer
	testApp := catalog.UnifiedApplication{
		AppID: "org.videolan.VLC",
		Name:  "VLC media player",
		Offers: []catalog.PackageOffer{
			{
				ID:               "pacman:extra:vlc",
				PackageName:      "vlc",
				Provider:         "pacman",
				Source:           "extra",
				IsInstalled:      true,
				InstalledVersion: "3.0.20",
				IsEligible:       true,
			},
			{
				ID:          "flatpak:flathub:org.videolan.VLC",
				PackageName: "org.videolan.VLC",
				Provider:    "flatpak",
				Source:      "flathub",
				IsInstalled: false,
				IsEligible:  true,
			},
		},
		IsInstalled:      true,
		InstalledOfferID: "pacman:extra:vlc",
	}

	recommender := catalog.NewRecommendationEngine(catalog.DefaultCatalogConfig())
	// User switches offer to Flatpak
	recommender.RankAndAnnotate(&testApp, "flatpak:flathub:org.videolan.VLC")

	if testApp.DefaultOfferID != "flatpak:flathub:org.videolan.VLC" {
		t.Errorf("Expected DefaultOfferID to be flatpak, got %s", testApp.DefaultOfferID)
	}

	// Verify Flatpak offer is NOT marked installed
	var flatpakOffer *catalog.PackageOffer
	for i := range testApp.Offers {
		if testApp.Offers[i].ID == "flatpak:flathub:org.videolan.VLC" {
			flatpakOffer = &testApp.Offers[i]
		}
	}

	if flatpakOffer == nil || flatpakOffer.IsInstalled {
		t.Fatalf("Flatpak offer must NOT be marked installed!")
	}
	_ = app
}

// TestApp_ProgressiveSearchEvents tests streaming search events
func TestApp_ProgressiveSearchEvents(t *testing.T) {
	app := NewApp()

	doneCalled := false
	batchCount := 0

	searchID := "test_prog_search"
	app.catalogMgr.ProgressiveSearch().StreamSearch(
		context.Background(),
		searchID,
		"steam",
		func(batch catalog.ProgressiveSearchBatchDTO) {
			batchCount++
			if batch.SearchID != searchID {
				t.Errorf("Expected search ID %s, got %s", searchID, batch.SearchID)
			}
		},
		func(done catalog.ProgressiveSearchDoneDTO) {
			doneCalled = true
		},
	)

	if !doneCalled {
		t.Errorf("Expected done callback to be called")
	}

	_ = models.Package{}
}

