package identity

import (
	"strings"
	"testing"
)

func TestIdentityVsIconSeparation(t *testing.T) {
	reg := NewRegistry()
	reg.Register(AppIdentityEntry{
		CanonicalID:    "org.mozilla.firefox",
		Name:           "Firefox",
		PackageAliases: []string{"firefox"},
		IconNames:      []string{"firefox", "firefox-esr", "firefox-icon"},
	})
	res := NewResolver(reg)

	// Icon names must NOT leak into package aliases for ambiguous variants like firefox-esr
	pkgAliases := res.GetPackageAliases("org.mozilla.firefox")
	for _, pa := range pkgAliases {
		if pa == "firefox-esr" || pa == "firefox-icon" {
			t.Fatalf("icon name %q leaked into package aliases %v", pa, pkgAliases)
		}
	}

	// Verify firefox-esr is NOT merged into org.mozilla.firefox unless explicitly mapped as a pkg alias
	esr := res.Resolve("firefox-esr", "pacman")
	if esr.Mergeable {
		t.Errorf("Expected unmapped firefox-esr variant to have Mergeable=false, got Mergeable=true (canon=%q)", esr.CanonicalID)
	}

	iconNames := res.GetIconNames("org.mozilla.firefox")
	foundIcon := false
	for _, ic := range iconNames {
		if ic == "firefox-esr" {
			foundIcon = true
			break
		}
	}
	if !foundIcon {
		t.Errorf("Expected 'firefox-esr' in icon names, got %v", iconNames)
	}
}

func TestConfidenceLevelsAndUncertainRejection(t *testing.T) {
	res := NewResolver(GetDefaultRegistry())

	rExact := res.Resolve("org.mozilla.firefox", "flatpak")
	if rExact.Confidence != ConfidenceExact || !rExact.Mergeable {
		t.Errorf("Expected exact mergeable match for org.mozilla.firefox, got %+v", rExact)
	}

	rHigh := res.Resolve("firefox", "pacman")
	if rHigh.Confidence != ConfidenceHigh || !rHigh.Mergeable || !strings.EqualFold(rHigh.CanonicalID, "org.mozilla.firefox") {
		t.Errorf("Expected high confidence match for firefox, got %+v", rHigh)
	}

	rGit := res.Resolve("firefox-git", "aur")
	if rGit.Mergeable {
		t.Errorf("Expected uncurated -git track to NOT auto-merge blindly (Mergeable=false), got %+v", rGit)
	}

	rUnknown := res.Resolve("totally-unrelated-pkg", "pacman")
	if rUnknown.Confidence != ConfidenceNone || rUnknown.Mergeable {
		t.Errorf("Expected unmapped package to stay unmerged, got %+v", rUnknown)
	}
}

func FuzzCanonicalizeID(f *testing.F) {
	seeds := []string{
		"firefox",
		"firefox-bin",
		"org.mozilla.firefox",
		"com.valvesoftware.Steam",
		"",
		"   ",
		"a.b.c-git",
		"pacman/extra/vlc",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	res := NewResolver(GetDefaultRegistry())
	f.Fuzz(func(t *testing.T, input string) {
		out := res.Resolve(input, "pacman")
		if strings.TrimSpace(input) != "" && out.CanonicalID == "" {
			t.Errorf("non-empty input %q produced empty CanonicalID", input)
		}
	})
}

