package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
)

func (m Model) renderInstalledAndHealthView(w, paneBoxHeight int) string {
	boxH := paneBoxHeight + 4
	innerW := w - 4

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ (2) INSTALLED APPLICATIONS & SYSTEM HEALTH ─┐") + "\n")

	// Provider counts summary
	counts := make(map[string]int)
	for _, e := range m.installedEntries {
		counts[strings.ToLower(e.Provider)]++
	}
	summaryLine := fmt.Sprintf("  Detected Installed: %d packages", len(m.installedEntries))
	if len(counts) > 0 {
		var parts []string
		for k, v := range counts {
			parts = append(parts, fmt.Sprintf("%s:%d", k, v))
		}
		summaryLine += "  (" + strings.Join(parts, " • ") + ")"
	}
	lockBadge := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("DB LOCK: FREE")
	if m.telemetry.Locked {
		lockBadge = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true).Render("DB LOCK: HELD (" + m.telemetry.LockPath + ")")
	}
	b.WriteString(summaryLine + "   " + lockBadge + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("  Shortcuts: [j/k] Navigate • [r] Remove Selected • [u] Rescan Host") + "\n\n")

	if !m.installedLoaded {
		b.WriteString("  " + m.spinner.View() + " Scanning local system packages (pacman/apt/dnf/flatpak)...\n")
	} else if len(m.installedEntries) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("  No installed packages discovered (press [u] to rescan).\n"))
	} else {
		maxRows := boxH - 8
		if maxRows < 5 {
			maxRows = 5
		}
		start := 0
		if m.installedCursor >= maxRows {
			start = m.installedCursor - maxRows + 1
		}
		end := start + maxRows
		if end > len(m.installedEntries) {
			end = len(m.installedEntries)
		}
		for i := start; i < end; i++ {
			e := m.installedEntries[i]
			cursorMark := "  "
			rowStyle := ListItemStyle
			if i == m.installedCursor {
				cursorMark = "▶ "
				rowStyle = ListItemSelectedStyle
			}
			line := fmt.Sprintf("%s%-28s  %-16s  %s  (%s)",
				cursorMark,
				truncateText(e.Name, 28),
				truncateText(e.Version, 16),
				ProviderBadge(e.Provider),
				e.Format,
			)
			b.WriteString(rowStyle.Width(innerW-2).Render(clampStyledWidth(line, innerW-2)) + "\n")
		}
	}

	return CardFocusedStyle.Width(w - 2).Height(boxH).Render(b.String())
}

func (m Model) renderUpdatesCenterView(w, paneBoxHeight int) string {
	boxH := paneBoxHeight + 4
	innerW := w - 4

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ (3) UNIFIED UPDATE CENTER ─┐") + "\n")
	b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("  Shortcuts: [j/k] Navigate • [u] Check All Updates • [i] Apply Selected / Full Upgrade") + "\n\n")

	if m.checkingUpdates {
		b.WriteString("  " + m.spinner.View() + " Checking updates across Pacman, AUR, Flatpak, Snap...\n")
		return CardFocusedStyle.Width(w - 2).Height(boxH).Render(b.String())
	}

	if !m.updatesLoaded || m.updateSummary == nil {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("  Update summary not yet loaded. Press [u] to check all providers.\n"))
		return CardFocusedStyle.Width(w - 2).Height(boxH).Render(b.String())
	}

	s := m.updateSummary
	b.WriteString(fmt.Sprintf("  Pending Updates: %s  |  Security: %s  |  Providers: %d\n",
		lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render(fmt.Sprintf("%d", s.TotalCount)),
		lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render(fmt.Sprintf("%d", s.SecurityCount)),
		s.ProviderCount,
	))
	if s.RequiresFullSysUpg {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("  ⚠ Partial upgrades unsupported on Arch: Pacman updates apply full system upgrade (pacman -Syu).") + "\n")
	}
	b.WriteString("\n")

	if len(s.Updates) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("  ✔ All tracked packages across active providers are up to date.\n"))
	} else {
		maxRows := boxH - 10
		if maxRows < 4 {
			maxRows = 4
		}
		start := 0
		if m.updatesCursor >= maxRows {
			start = m.updatesCursor - maxRows + 1
		}
		end := start + maxRows
		if end > len(s.Updates) {
			end = len(s.Updates)
		}
		for i := start; i < end; i++ {
			u := s.Updates[i]
			mark := "  "
			style := ListItemStyle
			if i == m.updatesCursor {
				mark = "▶ "
				style = ListItemSelectedStyle
			}
			sec := ""
			if u.IsSecurity {
				sec = " [SEC]"
			}
			line := fmt.Sprintf("%s%-22s %s -> %s  %s%s (%s)",
				mark,
				truncateText(u.PackageName, 22),
				truncateText(u.InstalledVersion, 12),
				truncateText(u.AvailableVersion, 12),
				ProviderBadge(u.Provider),
				sec,
				u.DownloadSize,
			)
			b.WriteString(style.Width(innerW-2).Render(clampStyledWidth(line, innerW-2)) + "\n")
		}
	}

	if len(s.ProviderErrors) > 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorMuted).Render("  Provider notes: "))
		for k, v := range s.ProviderErrors {
			b.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("[%s: %s] ", k, truncateText(v, 40))))
		}
		b.WriteString("\n")
	}

	return CardFocusedStyle.Width(w - 2).Height(boxH).Render(b.String())
}

func (m Model) renderPipelineAndLogsView(w, paneBoxHeight int) string {
	boxH := paneBoxHeight + 4
	leftOuter := (w * 48) / 100
	rightOuter := w - leftOuter - 1

	// Left: Transaction Journal
	var leftB strings.Builder
	leftB.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ (4) TRANSACTION JOURNAL ─┐\n\n"))
	var txs []catalog.TransactionDTO
	if m.catalogMgr != nil {
		txs = m.catalogMgr.Transactions().ListTransactions()
	}
	if len(txs) == 0 {
		leftB.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("  No transactions recorded in this session yet.\n  Press [i] or [r] on any package to execute via TransactionManager.\n"))
	} else {
		maxTx := boxH - 5
		if maxTx < 4 {
			maxTx = 4
		}
		start := 0
		if len(txs) > maxTx {
			start = len(txs) - maxTx
		}
		for i := len(txs) - 1; i >= start; i-- {
			tx := txs[i]
			stateTag := string(tx.State)
			leftB.WriteString(fmt.Sprintf("  %s %-10s %-18s [%s] %d%%\n",
				ProviderBadge(tx.Provider),
				strings.ToUpper(string(tx.Operation)),
				truncateText(tx.PackageName, 18),
				stateTag,
				tx.Progress,
			))
		}
	}
	leftPane := CardFocusedStyle.Width(leftOuter - 2).Height(boxH).Render(leftB.String())

	// Right: Engine Activity Logs
	var rightB strings.Builder
	rightB.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ ENGINE ACTIVITY STREAM ─┐\n\n"))
	maxLogs := boxH - 5
	if maxLogs < 4 {
		maxLogs = 4
	}
	startLog := 0
	if len(m.logLines) > maxLogs {
		startLog = len(m.logLines) - maxLogs
	}
	for _, l := range m.logLines[startLog:] {
		line := truncateText(l, rightOuter-6)
		if strings.Contains(l, "✔") {
			rightB.WriteString("  " + LogSuccessStyle.Render(line) + "\n")
		} else if strings.Contains(l, "✖") {
			rightB.WriteString("  " + LogErrorStyle.Render(line) + "\n")
		} else {
			rightB.WriteString("  " + LogInfoStyle.Render(line) + "\n")
		}
	}
	rightPane := CardStyle.Width(rightOuter - 2).Height(boxH).Render(rightB.String())

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

func (m Model) renderSettingsAndProvidersView(w, paneBoxHeight int) string {
	boxH := paneBoxHeight + 4
	innerW := w - 4

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ (5) ENGINE SETTINGS & PROVIDER CAPABILITY MATRIX ─┐\n\n"))

	policy := catalog.PolicyRecommended
	if m.catalogMgr != nil {
		policy = m.catalogMgr.GetPolicy()
	}
	b.WriteString(fmt.Sprintf("  • %-24s %s\n", "Operating System:", m.distro))
	b.WriteString(fmt.Sprintf("  • %-24s %s (%s)\n", "Host / Kernel:", m.telemetry.Hostname, m.telemetry.Kernel))
	b.WriteString(fmt.Sprintf("  • %-24s %s\n", "Architecture:", m.telemetry.Arch))
	b.WriteString(fmt.Sprintf("  • %-24s %s  %s\n\n",
		"Recommendation Policy:",
		lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render(string(policy)),
		lipgloss.NewStyle().Foreground(ColorMuted).Render("(press [c] to cycle)"),
	))

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright).Render("  PROVIDER CAPABILITY MATRIX:\n"))
	if m.catalogMgr != nil {
		for _, c := range m.catalogMgr.Capabilities() {
			mark := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("✔")
			if !c.AvailableOnHost {
				mark = lipgloss.NewStyle().Foreground(ColorMuted).Render("○")
			}
			line := fmt.Sprintf("  %s %-10s [Search:%-7s | Install:%-7s | Update:%-7s]  %s",
				mark,
				strings.ToUpper(c.Provider),
				c.Search,
				c.Install,
				c.Update,
				truncateText(c.StatusNote, innerW-62),
			)
			b.WriteString(line + "\n")
		}
	} else {
		for _, pName := range m.pm.GetSupportedProviderNames() {
			b.WriteString(fmt.Sprintf("  ✔ %-16s %s\n", pName, ProviderBadge(pName)))
		}
	}

	return CardFocusedStyle.Width(w - 2).Height(boxH).Render(b.String())
}

// Helper for synthesizing an Offer when removing directly from the Installed view
func offerFromInstalledEntry(e InstalledEntry) models.Package {
	return models.Package{
		Name:     e.Name,
		Version:  e.Version,
		Provider: e.Provider,
		Scope:    e.Scope,
	}
}
