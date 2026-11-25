package catalog

import (
	"testing"
)

func TestRelevanceScoring(t *testing.T) {
	firefox := UnifiedApplication{
		AppID:       "firefox",
		Name:        "Firefox",
		Summary:     "Fast, Private & Safe Web Browser",
		Description: "Fast, Private & Safe Web Browser from Mozilla",
		Aliases:     []string{"firefox", "org.mozilla.firefox"},
		IsInstalled: true,
		Offers: []PackageOffer{
			{
				Provider:   "pacman",
				Repository: "extra",
				Trust: SourceTrust{
					OfficialRepo: true,
				},
			},
		},
	}

	addWater := UnifiedApplication{
		AppID:       "addwater",
		Name:        "Add Water",
		Summary:     "Keep Firefox in fashion",
		Description: "Installer for GNOME Firefox theme",
		Aliases:     []string{"addwater"},
		IsInstalled: false,
		Offers: []PackageOffer{
			{
				Provider: "flatpak",
			},
		},
	}

	firefoxDev := UnifiedApplication{
		AppID:       "firefox-developer-edition",
		Name:        "Firefox Developer Edition",
		Summary:     "Fast, Private & Safe Web Browser (Developer Edition)",
		Description: "Fast, Private & Safe Web Browser for Developers",
		Aliases:     []string{"firefox-developer-edition"},
		IsInstalled: false,
		Offers: []PackageOffer{
			{
				Provider:   "pacman",
				Repository: "extra",
			},
		},
	}

	apps := []UnifiedApplication{addWater, firefoxDev, firefox}

	SortByRelevance(apps, "firefox browser")

	if apps[0].AppID != "firefox" {
		t.Fatalf("Expected #1 app to be 'firefox', got '%s'", apps[0].AppID)
	}

	if apps[1].AppID != "firefox-developer-edition" {
		t.Fatalf("Expected #2 app to be 'firefox-developer-edition', got '%s'", apps[1].AppID)
	}

	if apps[2].AppID != "addwater" {
		t.Fatalf("Expected #3 app to be 'addwater', got '%s'", apps[2].AppID)
	}

	scoreFirefox := CalculateRelevanceScore(firefox, "firefox browser")
	scoreAddWater := CalculateRelevanceScore(addWater, "firefox browser")

	if scoreFirefox <= scoreAddWater {
		t.Fatalf("Firefox score (%.1f) should be significantly higher than AddWater (%.1f)", scoreFirefox, scoreAddWater)
	}
}

