package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/updates"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/config"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
)

func setupTestModel() Model {
	cfg := config.DefaultConfig()
	pm := providers.NewProviderManager(cfg)
	m := NewModel(pm)
	m.width = 100
	m.height = 30
	return m
}

func TestTUIInitialization(t *testing.T) {
	m := setupTestModel()
	if m.textInput.Prompt != "❯ " {
		t.Errorf("expected prompt '❯ ', got %s", m.textInput.Prompt)
	}
	if m.activeTab != 0 {
		t.Errorf("expected initial activeTab 0, got %d", m.activeTab)
	}
	view := m.View()
	if !strings.Contains(view, "ARJAX TUI") {
		t.Errorf("expected view to contain 'ARJAX TUI', got: %s", view)
	}
}

func TestTUIWindowResize(t *testing.T) {
	m := setupTestModel()
	res, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := res.(Model)

	if updated.width != 120 || updated.height != 40 {
		t.Errorf("expected size 120x40, got %dx%d", updated.width, updated.height)
	}
}

func TestTUITabSwitching(t *testing.T) {
	m := setupTestModel()
	m.textInput.Blur()

	// Switch to Tab 1
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(Model)
	if m.activeTab != 1 {
		t.Errorf("expected tab 1, got %d", m.activeTab)
	}

	// Switch to Tab 2
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = res.(Model)
	if m.activeTab != 2 {
		t.Errorf("expected tab 2, got %d", m.activeTab)
	}

	// Switch back using [
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = res.(Model)
	if m.activeTab != 1 {
		t.Errorf("expected tab 1 after '[', got %d", m.activeTab)
	}
}

func TestTUISearchResultsAndScrolling(t *testing.T) {
	m := setupTestModel()
	pkgs := []models.Package{
		{Name: "pkg1", Version: "1.0", Provider: "pacman", Description: "First package"},
		{Name: "pkg2", Version: "2.0", Provider: "aur", Description: "Second package"},
		{Name: "pkg3", Version: "3.0", Provider: "flatpak", Description: "Third package"},
	}

	res, _ := m.Update(searchDoneMsg{
		query:    "test",
		packages: pkgs,
		duration: 25 * time.Millisecond,
	})
	m = res.(Model)

	if len(m.packages) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(m.packages))
	}
	if !strings.Contains(m.statusMsg, "Found 3 package(s)") {
		t.Errorf("expected statusMsg to report 3 packages, got: %s", m.statusMsg)
	}

	// Navigation with j / k
	m.textInput.Blur()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = res.(Model)
	if m.cursor != 1 {
		t.Errorf("expected cursor 1, got %d", m.cursor)
	}

	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = res.(Model)
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}

	view := m.View()
	if !strings.Contains(view, "pkg1") || !strings.Contains(view, "pkg2") {
		t.Errorf("expected view to render package names, got: %s", view)
	}
}

func TestProviderBadgeColors(t *testing.T) {
	pacmanBadge := ProviderBadge("pacman")
	aurBadge := ProviderBadge("aur")
	flatpakBadge := ProviderBadge("flatpak")

	if !strings.Contains(pacmanBadge, "PACMAN") {
		t.Errorf("expected pacman badge to render PACMAN, got %s", pacmanBadge)
	}
	if !strings.Contains(aurBadge, "AUR") {
		t.Errorf("expected aur badge to render AUR, got %s", aurBadge)
	}
	if !strings.Contains(flatpakBadge, "FLATPAK") {
		t.Errorf("expected flatpak badge to render FLATPAK, got %s", flatpakBadge)
	}
}

func TestTUIInteractiveInstallProgress(t *testing.T) {
	m := setupTestModel()
	m.textInput.Blur()
	m.packages = []models.Package{
		{Name: "brave-bin", Version: "1.70", Provider: "aur", Description: "Brave Browser"},
	}

	// Trigger install with 'i'
	res, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = res.(Model)
	if cmd == nil {
		t.Fatal("expected commands returned on install trigger")
	}

	if !m.installing {
		t.Errorf("expected installing=true, got false")
	}
	if m.installPkg.Name != "brave-bin" {
		t.Errorf("expected installPkg brave-bin, got %s", m.installPkg.Name)
	}
	if m.installProgress <= 0 {
		t.Errorf("expected initial installProgress > 0, got %f", m.installProgress)
	}

	// Check View renders interactive live progress
	view := m.View()
	if !strings.Contains(view, "INSTALLING") {
		t.Errorf("expected view to contain 'INSTALLING', got:\n%s", view)
	}
	if !strings.Contains(view, "brave-bin") {
		t.Errorf("expected view to contain 'brave-bin', got:\n%s", view)
	}
	if !strings.Contains(view, "EXECUTION PIPELINE") {
		t.Errorf("expected view to contain 'EXECUTION PIPELINE', got:\n%s", view)
	}

	// Tick install progress
	res, _ = m.Update(installTickMsg(time.Now()))
	m = res.(Model)
	if m.installProgress <= 0.08 {
		t.Errorf("expected installProgress to advance on tick, got %f", m.installProgress)
	}

	// Complete install action
	res, _ = m.Update(actionDoneMsg{
		action: "install",
		result: models.Result{Package: "brave-bin", Provider: "aur", Success: true, DurationMs: 1200},
	})
	m = res.(Model)
	if m.installing {
		t.Errorf("expected installing=false after actionDoneMsg")
	}
	if m.installProgress != 1.0 {
		t.Errorf("expected installProgress=1.0 after actionDoneMsg, got %f", m.installProgress)
	}
	if !strings.Contains(m.statusMsg, "completed") {
		t.Errorf("expected statusMsg to indicate completion, got: %s", m.statusMsg)
	}
}

func makeSampleUnifiedApps() []catalog.UnifiedApplication {
	flatTrust := catalog.EvaluateSourceTrust("flatpak", "flathub", "io.zapsoft.Zapfast")
	pacTrust := catalog.EvaluateSourceTrust("pacman", "extra", "zapfast")
	aurTrust := catalog.EvaluateSourceTrust("aur", "aur", "zapfast-git")

	return []catalog.UnifiedApplication{
		{
			AppID:          "io.zapsoft.Zapfast",
			Name:           "zapfast",
			Summary:        "Lightning-fast media stream toolkit",
			Developer:      "ZapSoft Labs",
			Homepage:       "https://zapsoft.org/zapfast",
			License:        "GPL-3.0",
			DefaultOfferID: "flatpak:flathub:io.zapsoft.zapfast",
			IsInstalled:    true,
			Offers: []catalog.PackageOffer{
				{
					ID:                  "flatpak:flathub:io.zapsoft.zapfast",
					AppID:               "io.zapsoft.Zapfast",
					PackageName:         "io.zapsoft.Zapfast",
					Provider:            "flatpak",
					Repository:          "flathub",
					Version:             "2.8.4",
					DownloadSize:        "184 MB",
					InstallScope:        catalog.ScopeSandboxed,
					Permissions:         []string{"network", "pulseaudio", "wayland/x11"},
					Dependencies:        []string{"org.freedesktop.Platform//23.08"},
					IsInstalled:         true,
					IsEligible:          true,
					TrustLevel:          flatTrust.TrustLevel,
					SignatureStatus:     flatTrust.SignatureStatus,
					Trust:               flatTrust,
					Score:               92,
					RecommendationBadge: "Recommended",
					CommandPreview:      "flatpak install --user flathub io.zapsoft.Zapfast",
				},
				{
					ID:              "pacman:extra:zapfast",
					AppID:           "io.zapsoft.Zapfast",
					PackageName:     "zapfast",
					Provider:        "pacman",
					Repository:      "extra",
					Version:         "2.8.1",
					DownloadSize:    "42 MB",
					InstallScope:    catalog.ScopeSystem,
					Permissions:     []string{"system-native", "host-filesystem"},
					IsEligible:      true,
					TrustLevel:      pacTrust.TrustLevel,
					SignatureStatus: pacTrust.SignatureStatus,
					Trust:           pacTrust,
					Score:           86,
					CommandPreview:  "sudo pacman -S zapfast",
				},
			},
		},
		{
			AppID:   "brave-browser",
			Name:    "brave-bin",
			Summary: "Privacy-first Chromium browser",
			Offers: []catalog.PackageOffer{
				{
					ID:              "aur:aur:brave-bin",
					AppID:           "brave-browser",
					PackageName:     "brave-bin",
					Provider:        "aur",
					Repository:      "aur",
					Version:         "1.65.130",
					DownloadSize:    "PKGBUILD",
					InstallScope:    catalog.ScopeUser,
					IsEligible:      true,
					TrustLevel:      aurTrust.TrustLevel,
					SignatureStatus: aurTrust.SignatureStatus,
					Trust:           aurTrust,
					Score:           68,
					CommandPreview:  "paru -S brave-bin",
				},
			},
		},
	}
}

func TestTUICanonicalInspectorAndOfferCycling(t *testing.T) {
	m := setupTestModel()
	m.width = 160
	m.height = 46
	m.textInput.Blur()

	res, _ := m.Update(searchDoneMsg{
		query:       "zapfast",
		unifiedApps: makeSampleUnifiedApps(),
		duration:    18 * time.Millisecond,
	})
	m = res.(Model)

	view := m.View()
	for _, want := range []string{
		"CANONICAL PACKAGES",
		"PACKAGE INSPECTOR: zapfast",
		"Trust & Integrity:",
		"SELECT OFFER / DISTRIBUTION TARGET:",
		"SANDBOX PERMISSIONS MATRIX:",
		"STRATEGY RESOLUTION TREE:",
		"flatpak install --user flathub io.zapsoft.Zapfast",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("expected Stitch inspector view to contain %q, got:\n%s", want, view)
		}
	}

	// Press 'o' to cycle from Flatpak offer to Pacman Extra offer
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = res.(Model)
	viewAfterCycle := m.View()
	if !strings.Contains(viewAfterCycle, "sudo pacman -S zapfast") {
		t.Errorf("expected offer cycling with 'o' to update preview to pacman command, got:\n%s", viewAfterCycle)
	}

	// Press 'v' to open provenance & recipe view
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = res.(Model)
	if !m.showRecipeModal {
		t.Errorf("expected showRecipeModal=true after pressing 'v'")
	}
	recipeView := m.View()
	if !strings.Contains(recipeView, "PROVENANCE & RECIPE") {
		t.Errorf("expected recipe overlay to render provenance details, got:\n%s", recipeView)
	}
}

func TestTUIProviderAndScopeFiltering(t *testing.T) {
	m := setupTestModel()
	m.width = 140
	m.height = 40
	m.textInput.Blur()
	m.unifiedApps = makeSampleUnifiedApps()

	if got := len(m.filteredUnifiedApps()); got != 2 {
		t.Fatalf("expected 2 unfiltered canonical apps, got %d", got)
	}

	// Cycle provider filter: all -> native
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = res.(Model)
	if m.providerFilter != "native" {
		t.Fatalf("expected providerFilter 'native', got %s", m.providerFilter)
	}
	if got := len(m.filteredUnifiedApps()); got != 1 {
		t.Errorf("expected 1 app with native offer, got %d", got)
	}

	// Cycle provider filter to aur (all -> native -> aur)
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = res.(Model)
	if m.providerFilter != "aur" {
		t.Fatalf("expected providerFilter 'aur', got %s", m.providerFilter)
	}
	filtered := m.filteredUnifiedApps()
	if len(filtered) != 1 || filtered[0].Name != "brave-bin" {
		t.Errorf("expected brave-bin under AUR filter, got %+v", filtered)
	}

	// Cycle scope filter: all -> sandboxed
	m.providerFilter = "all"
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = res.(Model)
	if m.scopeFilter != "sandboxed" {
		t.Fatalf("expected scopeFilter 'sandboxed', got %s", m.scopeFilter)
	}
	if got := len(m.filteredUnifiedApps()); got != 1 {
		t.Errorf("expected 1 sandboxed app, got %d", got)
	}
}

func TestTUIAllFiveTabsAndPolicyCycle(t *testing.T) {
	m := setupTestModel()
	m.width = 140
	m.height = 40
	m.textInput.Blur()

	// Populate updates & installed entries
	m.installedEntries = []InstalledEntry{{ID: "neovim", Name: "neovim", Version: "0.10.1", Provider: "pacman", Format: "Native Pacman"}}
	m.installedLoaded = true
	m.updateSummary = &updates.UpdateSummary{
		TotalCount:    1,
		ProviderCount: 1,
		Updates: []updates.AvailableUpdate{
			{PackageName: "firefox", Provider: "pacman", InstalledVersion: "124.0", AvailableVersion: "125.0", DownloadSize: "78 MB", IsSecurity: true},
		},
	}
	m.updatesLoaded = true

	// Tab 2: Installed & Health
	res, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = res.(Model)
	if !strings.Contains(m.View(), "INSTALLED APPLICATIONS & SYSTEM HEALTH") || !strings.Contains(m.View(), "neovim") {
		t.Errorf("expected Tab 2 to render installed applications")
	}

	// Tab 3: Updates
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = res.(Model)
	if !strings.Contains(m.View(), "UNIFIED UPDATE CENTER") || !strings.Contains(m.View(), "firefox") {
		t.Errorf("expected Tab 3 to render update center")
	}

	// Tab 4: Pipeline
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	m = res.(Model)
	if !strings.Contains(m.View(), "TRANSACTION JOURNAL") {
		t.Errorf("expected Tab 4 to render transaction journal")
	}

	// Tab 5: Settings + cycle policy with 'c'
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	m = res.(Model)
	if !strings.Contains(m.View(), "PROVIDER CAPABILITY MATRIX") {
		t.Errorf("expected Tab 5 to render provider capability matrix")
	}
	beforePolicy := m.catalogMgr.GetPolicy()
	res, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = res.(Model)
	if m.catalogMgr.GetPolicy() == beforePolicy {
		t.Errorf("expected 'c' in Settings tab to cycle recommendation policy")
	}
}

func TestTUIDebounceSearchIgnoresStaleTokens(t *testing.T) {
	m := setupTestModel()
	m.textInput.Focus()
	m.textInput.SetValue("zapfast")
	m.searchSeq = 5

	// Stale debounce seq=3 should be ignored
	res, cmd := m.Update(searchDebounceMsg{seq: 3, query: "zap"})
	m = res.(Model)
	if m.loading || cmd != nil {
		t.Errorf("expected stale debounce seq to be ignored")
	}

	// Current debounce seq=5 should trigger search
	res, cmd = m.Update(searchDebounceMsg{seq: 5, query: "zapfast"})
	m = res.(Model)
	if !m.loading || cmd == nil {
		t.Errorf("expected matching debounce seq to trigger background search")
	}
}



