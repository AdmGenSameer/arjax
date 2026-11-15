package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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


