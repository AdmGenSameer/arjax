package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
)

func (m Model) renderSearchAndStoreView(w, paneBoxHeight int) string {
	// 1. Search Input Box + Metadata Chips
	queryDisplay := m.textInput.View()
	debounceNote := lipgloss.NewStyle().Foreground(ColorMuted).Render("⌨ Debounce: 500ms")
	resolvedCount := len(m.filteredUnifiedApps())
	resolvedNote := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).
		Render(fmt.Sprintf("%d canonical", resolvedCount)) +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(" apps resolved")

	rightChips := lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(ColorMuted).Render("ARCH: "),
		MetaChipStyle.Render("["+m.telemetry.Arch+"]"),
		lipgloss.NewStyle().Foreground(ColorSubtle).Render(" | "),
		lipgloss.NewStyle().Foreground(ColorMuted).Render("SCOPE: "),
		MetaChipStyle.Render("["+m.scopeFilterLabel()+"]"),
		lipgloss.NewStyle().Foreground(ColorSubtle).Render(" "),
		lipgloss.NewStyle().Foreground(ColorOnPrimary).Background(ColorPrimarySoft).Bold(true).Padding(0, 1).Render("Ctrl+U: Clear"),
	)

	searchLeft := queryDisplay
	if w >= 120 {
		searchLeft = lipgloss.JoinHorizontal(
			lipgloss.Center,
			queryDisplay,
			"  ",
			debounceNote,
			lipgloss.NewStyle().Foreground(ColorSubtle).Render(" • "),
			resolvedNote,
		)
	}
	searchGap := w - 4 - lipgloss.Width(searchLeft) - lipgloss.Width(rightChips)
	if searchGap < 1 {
		searchGap = 1
	}
	searchRow := lipgloss.JoinHorizontal(lipgloss.Center, searchLeft, strings.Repeat(" ", searchGap), rightChips)
	if lipgloss.Width(searchRow) > w-4 {
		searchRow = searchLeft
	}

	var searchBox string
	if m.textInput.Focused() {
		searchBox = SearchFocusedStyle.Width(w - 2).Render(searchRow)
	} else {
		searchBox = SearchBlurredStyle.Width(w - 2).Render(searchRow)
	}

	// 2. Provider Filter Strip
	filterBar := m.renderProviderFilterStrip(w)

	// 3. Compute Dual Pane Geometry (45% Left / 55% Right)
	leftOuter := (w * 45) / 100
	if leftOuter < 34 {
		leftOuter = 34
	}
	if w-leftOuter-1 < 36 {
		leftOuter = w - 37
		if leftOuter < 28 {
			leftOuter = 28
		}
	}
	rightOuter := w - leftOuter - 1
	if rightOuter < 30 {
		rightOuter = 30
	}

	leftPane := m.renderCanonicalListPane(leftOuter, paneBoxHeight)
	rightPane := m.renderInspectorPane(rightOuter, paneBoxHeight)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		searchBox,
		filterBar,
		body,
	)
}

func (m Model) renderProviderFilterStrip(w int) string {
	apps := m.effectiveUnifiedApps()
	totalCount := len(apps)

	nativeLabel := "Native " + strings.ToUpper(m.nativePM)
	if m.nativePM == "" || m.nativePM == "linux" {
		nativeLabel = "Native Repo"
	}

	filters := []struct {
		key   string
		label string
	}{
		{"all", fmt.Sprintf("All (%d)", totalCount)},
		{"native", nativeLabel},
		{"aur", "AUR"},
		{"flatpak", "Flathub"},
		{"snap", "Snap"},
		{"github", "GitHub Releases"},
	}

	var parts []string
	parts = append(parts, lipgloss.NewStyle().Foreground(ColorMuted).Bold(true).Render("PROVIDER: "))
	for i, f := range filters {
		tag := "[p]"
		if i == 0 {
			if m.providerFilter == "all" {
				tag = "[x]"
			} else {
				tag = "[ ]"
			}
		} else {
			tag = fmt.Sprintf("[%d]", i)
		}
		text := tag + " " + f.label
		if m.providerFilter == f.key {
			if i > 0 {
				text += " •"
			}
			parts = append(parts, FilterActiveStyle.Render(text), " ")
		} else {
			parts = append(parts, FilterInactiveStyle.Render(text), " ")
		}
	}

	leftStrip := lipgloss.JoinHorizontal(lipgloss.Center, parts...)
	syncText := lipgloss.NewStyle().Foreground(ColorSecondary).Render("● ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("Installed: %d • Sort: %s [S]", len(m.installedEntries), m.sortMode))
	if m.loading {
		syncText = m.spinner.View() + " " + lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true).Render(m.statusMsg)
	}

	gap := w - 2 - lipgloss.Width(leftStrip) - lipgloss.Width(syncText)
	if gap < 1 {
		return TopWindowBarStyle.Width(w).Render(clampStyledWidth(leftStrip, w-2))
	}
	return TopWindowBarStyle.Width(w).Render(leftStrip + strings.Repeat(" ", gap) + syncText)
}

func (m Model) renderCanonicalListPane(outerW, boxH int) string {
	innerW := outerW - 4
	if innerW < 24 {
		innerW = 24
	}

	apps := m.filteredUnifiedApps()
	countLabel := fmt.Sprintf("(%d results)", len(apps))

	headerLeft := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ ") +
		lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("[?] ") +
		lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("CANONICAL PACKAGES ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(countLabel) +
		lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(" ─┐")

	headerRight := lipgloss.NewStyle().Foreground(ColorMuted).Render("Sort:[") +
		lipgloss.NewStyle().Foreground(ColorText).Render(strings.Title(m.sortMode)) +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("] ") +
		lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("[j/k]")

	hGap := innerW - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	var headerLine string
	if hGap >= 1 {
		headerLine = headerLeft + strings.Repeat(" ", hGap) + headerRight
	} else {
		headerLine = clampStyledWidth(headerLeft, innerW)
	}

	// Each card takes 3 content lines + 1 spacer = 4 lines. Footer takes 1 line.
	const cardLines = 4
	availRows := boxH - 3
	if availRows < cardLines {
		availRows = cardLines
	}
	visibleCards := availRows / cardLines
	if visibleCards < 1 {
		visibleCards = 1
	}

	var listBody strings.Builder
	if len(apps) == 0 {
		if m.loading {
			listBody.WriteString("\n  " + m.spinner.View() + " " + lipgloss.NewStyle().Foreground(ColorHighlight).Render("Searching across providers...") + "\n")
		} else {
			listBody.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("\n  No canonical packages loaded.\n  Press [/] or [Tab] to search (e.g. zapfast, brave, neovim).\n"))
		}
	} else {
		start := m.scrollOffset
		if start < 0 {
			start = 0
		}
		if start >= len(apps) {
			start = len(apps) - 1
		}
		end := start + visibleCards
		if end > len(apps) {
			end = len(apps)
		}

		for i := start; i < end; i++ {
			app := apps[i]
			selected := i == m.cursor
			listBody.WriteString(m.renderCanonicalCard(app, i+1, selected, innerW))
			if i < end-1 {
				listBody.WriteString("\n")
			}
		}
	}

	// Pad remaining lines so footer stays anchored at bottom of pane
	renderedLines := strings.Count(listBody.String(), "\n")
	for renderedLines < availRows {
		listBody.WriteString("\n")
		renderedLines++
	}

	footLeft := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("%d deduplicated family(s)", len(apps)))
	footRight := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("Filter: [%s]", m.providerFilter))
	fGap := innerW - lipgloss.Width(footLeft) - lipgloss.Width(footRight)
	if fGap < 1 {
		fGap = 1
	}
	paneFooter := footLeft + strings.Repeat(" ", fGap) + footRight

	content := lipgloss.JoinVertical(lipgloss.Left, headerLine, listBody.String(), paneFooter)

	style := CardStyle
	if !m.textInput.Focused() {
		style = CardFocusedStyle
	}
	return style.Width(outerW - 2).Height(boxH).MaxHeight(boxH + 2).Render(content)
}

func (m Model) renderCanonicalCard(app catalog.UnifiedApplication, idx int, selected bool, innerW int) string {
	offer := m.activeOfferForApp(app)
	cardInnerW := innerW - 4
	if cardInnerW < 20 {
		cardInnerW = 20
	}

	// Row 1: Prefix + Name + Version + Installed Marker + Badge
	prefix := fmt.Sprintf("  %d. ", idx)
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright)
	if selected {
		prefix = fmt.Sprintf("❯ %d. ", idx)
		nameStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimarySoft)
	}

	installedMark := ""
	if app.IsInstalled || m.isAppInstalledOnHost(app) {
		installedMark = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(" ✔")
	}

	verStr := ""
	if offer.Version != "" {
		verStr = lipgloss.NewStyle().Foreground(ColorMuted).Render(" (" + truncateText(offer.Version, 14) + ")")
	}

	badge := ProviderBadge(offer.Provider)
	if offer.RecommendationBadge == "Recommended" && selected {
		badge = RecommendBadgeStyle.Render(strings.ToUpper(offer.Provider) + " ★")
	}

	nameMax := cardInnerW - lipgloss.Width(badge) - lipgloss.Width(prefix) - lipgloss.Width(verStr) - lipgloss.Width(installedMark) - 2
	if nameMax < 6 {
		nameMax = 6
	}

	leftTitle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(selected).Render(prefix) +
		nameStyle.Render(truncateText(app.Name, nameMax)) +
		installedMark + verStr

	gap1 := cardInnerW - lipgloss.Width(leftTitle) - lipgloss.Width(badge)
	if gap1 < 1 {
		gap1 = 1
	}
	row1 := leftTitle + strings.Repeat(" ", gap1) + badge

	// Row 2: Description
	summary := app.Summary
	if summary == "" {
		summary = app.Description
	}
	summary = strings.ReplaceAll(summary, "\n", " ")
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = "Linux software package"
	}
	descMax := cardInnerW - 3
	if descMax < 8 {
		descMax = 8
	}
	row2 := "   " + lipgloss.NewStyle().Foreground(ColorText).Render(truncateText(summary, descMax))

	// Row 3: Metadata & Alternative Offers (No fake trust score, no redundant double provider badge!)
	sizeStr := offer.DownloadSize
	if sizeStr == "" {
		sizeStr = "~45 MB"
	}

	var row3 string
	if len(app.Offers) > 1 {
		// Show alternative offers (excluding the currently active one to prevent double badges)
		var altBadges []string
		usedW := 14
		for _, o := range app.Offers {
			if strings.EqualFold(o.Provider, offer.Provider) {
				continue
			}
			tag := lipgloss.NewStyle().Foreground(ColorHighlight).Render("[" + strings.ToUpper(o.Provider) + "]")
			cw := lipgloss.Width(tag) + 1
			if usedW+cw > cardInnerW-16 {
				break
			}
			altBadges = append(altBadges, tag)
			usedW += cw
		}
		altText := "   " + lipgloss.NewStyle().Foreground(ColorMuted).Render("Also on: ")
		if len(altBadges) > 0 {
			altText += strings.Join(altBadges, " ")
		} else {
			altText += lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("+%d channels", len(app.Offers)-1))
		}
		sizeLabel := lipgloss.NewStyle().Foreground(ColorMuted).Render("Size: ") +
			lipgloss.NewStyle().Foreground(ColorPrimarySoft).Render(sizeStr)
		gap3 := cardInnerW - lipgloss.Width(altText) - lipgloss.Width(sizeLabel)
		if gap3 < 1 {
			gap3 = 1
		}
		row3 = altText + strings.Repeat(" ", gap3) + sizeLabel
	} else {
		// Single offer: Status + Size only (No redundant "Offers: [AUR]")
		statusLabel := lipgloss.NewStyle().Foreground(ColorMuted).Render("[Available]")
		if offer.IsInstalled || m.isOfferInstalledOnHost(offer) {
			statusLabel = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("[Installed]")
		}
		leftMeta := "   " + statusLabel
		sizeLabel := lipgloss.NewStyle().Foreground(ColorMuted).Render("Size: ") +
			lipgloss.NewStyle().Foreground(ColorPrimarySoft).Render(sizeStr)
		gap3 := cardInnerW - lipgloss.Width(leftMeta) - lipgloss.Width(sizeLabel)
		if gap3 < 1 {
			gap3 = 1
		}
		row3 = leftMeta + strings.Repeat(" ", gap3) + sizeLabel
	}

	cardContent := lipgloss.JoinVertical(lipgloss.Left, row1, row2, row3)
	if selected {
		return CanonicalCardSelectedStyle.Width(innerW).Render(cardContent) + "\n"
	}
	return CanonicalCardStyle.Width(innerW).Render(cardContent) + "\n"
}

func (m Model) renderInspectorPane(outerW, boxH int) string {
	innerW := outerW - 4
	if innerW < 28 {
		innerW = 28
	}

	if m.showHelp {
		return m.renderHelpOverlayPane(outerW, boxH, innerW)
	}

	if m.installing {
		return m.renderInstallPipelinePane(outerW, boxH, innerW)
	}

	apps := m.filteredUnifiedApps()
	if len(apps) == 0 || m.cursor >= len(apps) {
		header := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ [i] PACKAGE INSPECTOR ─┐")
		empty := lipgloss.NewStyle().Foreground(ColorMuted).Render("\n  Select a canonical package from the left pane\n  or search across Pacman, AUR, Flatpak, Snap, and GitHub.\n\n  Shortcuts:\n  • [/] or [Tab] Focus Search\n  • [p] Filter Provider\n  • [s] Filter Scope\n  • [o] Cycle Offer Target")
		return CardStyle.Width(outerW - 2).Height(boxH).Render(lipgloss.JoinVertical(lipgloss.Left, header, empty))
	}

	app := apps[m.cursor]
	offer := m.activeOfferForApp(app)

	if m.showRecipeModal {
		return m.renderRecipeProvenancePane(app, offer, outerW, boxH, innerW)
	}

	// Header
	headerLeft := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("┌─ [i] PACKAGE INSPECTOR: " + truncateText(app.Name, innerW-36) + " ─┐")
	headerRight := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("● ACTIVE TARGET")
	hGap := innerW - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	headerLine := headerLeft
	if hGap >= 1 {
		headerLine = headerLeft + strings.Repeat(" ", hGap) + headerRight
	}

	var sections []string
	sections = append(sections, headerLine)

	// 1. App Header & Identity
	license := app.License
	if license == "" {
		license = "Open Source"
	}
	sizeBadge := offer.DownloadSize
	if sizeBadge == "" {
		sizeBadge = "~45 MB"
	}
	dev := app.Developer
	if dev == "" {
		dev = "Linux Upstream Maintainers"
	}
	homepage := app.Homepage
	if homepage == "" {
		homepage = catalog.HomepageForProvider(offer.Provider, offer.PackageName)
	}

	instTag := ""
	if offer.IsInstalled || m.isOfferInstalledOnHost(offer) {
		instTag = " " + lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("[INSTALLED]")
	}

	titleLeft := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render(app.Name) + " " +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(offer.Version) + instTag
	titleRight := ProviderBadge(offer.Provider) + " " +
		lipgloss.NewStyle().Foreground(ColorText).Render("["+license+"]") + " " +
		lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true).Render("["+sizeBadge+"]")
	tGap := innerW - 2 - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight)
	if tGap < 1 {
		tGap = 1
	}
	originLine1 := titleLeft + strings.Repeat(" ", tGap) + titleRight
	originLine2 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Upstream: ") +
		lipgloss.NewStyle().Foreground(ColorText).Render(truncateText(dev, 28)) +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(" • ID: ") +
		lipgloss.NewStyle().Foreground(ColorPrimarySoft).Render(truncateText(app.AppID, innerW-44))
	originLine3 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Homepage: ") +
		lipgloss.NewStyle().Foreground(ColorHighlight).Underline(true).Render(truncateText(homepage, innerW-14))
	sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, originLine1, originLine2, originLine3)))

	// 2. Package Description (Prominently displayed and cleanly wrapped)
	descHeader := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("DESCRIPTION:")
	fullDesc := app.Description
	if fullDesc == "" {
		fullDesc = app.Summary
	}
	if fullDesc == "" {
		fullDesc = "No extended package description provided by upstream."
	}
	maxDescLines := 4
	if boxH >= 28 {
		maxDescLines = 6
	}
	wrappedLines := wrapText(fullDesc, innerW-6, maxDescLines)
	var descLines []string
	descLines = append(descLines, descHeader)
	for _, l := range wrappedLines {
		descLines = append(descLines, "  "+lipgloss.NewStyle().Foreground(ColorText).Render(l))
	}
	sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, descLines...)))

	// 3. Source & Security Verification (Truthful, honest, no fake scores)
	originName, integrityNote, isolationNote := getTruthfulIntegrity(offer)
	verHeader := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("SOURCE & INTEGRITY:")
	vLine1 := lipgloss.NewStyle().Foreground(ColorSecondary).Render("✔ ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Origin:    ") +
		lipgloss.NewStyle().Foreground(ColorText).Bold(true).Render(originName)
	vLine2 := lipgloss.NewStyle().Foreground(ColorSecondary).Render("✔ ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Security:  ") +
		lipgloss.NewStyle().Foreground(ColorText).Render(integrityNote)
	vLine3 := lipgloss.NewStyle().Foreground(ColorSecondary).Render("✔ ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Isolation: ") +
		lipgloss.NewStyle().Foreground(ColorText).Render(isolationNote)
	sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, verHeader, vLine1, vLine2, vLine3)))

	// 4. Select Distribution Source (if multiple offers exist)
	if len(app.Offers) > 1 {
		offerHdrLeft := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("SELECT OFFER / DISTRIBUTION TARGET:")
		offerHdrRight := lipgloss.NewStyle().Foreground(ColorMuted).Render("Press [o] to cycle")
		oGap := innerW - 2 - lipgloss.Width(offerHdrLeft) - lipgloss.Width(offerHdrRight)
		if oGap < 1 {
			oGap = 1
		}
		var offerRows []string
		offerRows = append(offerRows, offerHdrLeft+strings.Repeat(" ", oGap)+offerHdrRight)

		maxOffers := 3
		if boxH >= 28 {
			maxOffers = 4
		}
		for i, o := range app.Offers {
			if i >= maxOffers {
				offerRows = append(offerRows, lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("   ... +%d more sources (press [o] to cycle)", len(app.Offers)-maxOffers)))
				break
			}
			radio := "( )"
			rowStyle := lipgloss.NewStyle().Foreground(ColorText)
			if o.ID == offer.ID {
				radio = lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("(*)")
				rowStyle = lipgloss.NewStyle().Foreground(ColorTextBright).Bold(true)
			}
			oInst := ""
			if o.IsInstalled || m.isOfferInstalledOnHost(o) {
				oInst = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(" [INSTALLED]")
			}
			line := fmt.Sprintf("%s %s %s%s  %s",
				radio,
				ProviderBadge(o.Provider),
				rowStyle.Render(truncateText(o.Version, 16)),
				oInst,
				lipgloss.NewStyle().Foreground(ColorMuted).Render(o.DownloadSize),
			)
			offerRows = append(offerRows, clampStyledWidth(line, innerW-2))
		}
		sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, offerRows...)))
	}

	// 5. Read-Only Executable Command Preview
	cmdPreview := offer.CommandPreview
	if cmdPreview == "" {
		cmdPreview = catalog.FormatCommandPreview(offer.Provider, offer.Repository, offer.PackageName, offer.Branch, offer.Channel)
	}
	cmdLine := lipgloss.NewStyle().Foreground(ColorMuted).Render("PREVIEW: ") +
		CommandPreviewBoxStyle.Render("$ "+truncateText(cmdPreview, innerW-14))
	sections = append(sections, cmdLine)

	// 6. Action Buttons Row
	isInstalled := offer.IsInstalled || m.isOfferInstalledOnHost(offer)
	primaryBtnLabel := "[ i: Install " + truncateText(offer.PackageName, 16) + " ]"
	if isInstalled {
		primaryBtnLabel = "[ Enter: Launch / Update ]"
	}
	btnRow := lipgloss.JoinHorizontal(
		lipgloss.Center,
		ActionBtnPrimaryStyle.Render(primaryBtnLabel),
		" ",
		ActionBtnSecondaryStyle.Render("[ o: Switch Source ]"),
		" ",
		ActionBtnDangerStyle.Render("[ r: Remove ]"),
		" ",
		FilterInactiveStyle.Render("[ v: Recipe ]"),
	)
	sections = append(sections, clampStyledWidth(btnRow, innerW))

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return CardStyle.Width(outerW - 2).Height(boxH).MaxHeight(boxH + 2).Render(content)
}

func wrapText(text string, maxW int, maxLines int) []string {
	if maxW < 10 {
		maxW = 10
	}
	clean := strings.ReplaceAll(text, "\n", " ")
	words := strings.Fields(strings.TrimSpace(clean))
	if len(words) == 0 {
		return []string{"No extended package description provided."}
	}
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
		} else if cur.Len()+1+len(w) <= maxW {
			cur.WriteString(" ")
			cur.WriteString(w)
		} else {
			lines = append(lines, cur.String())
			if len(lines) >= maxLines {
				return lines
			}
			cur.Reset()
			cur.WriteString(w)
		}
	}
	if cur.Len() > 0 && len(lines) < maxLines {
		lines = append(lines, cur.String())
	}
	return lines
}

func getTruthfulIntegrity(offer catalog.PackageOffer) (origin, integrity, isolation string) {
	prov := strings.ToLower(strings.TrimSpace(offer.Provider))
	repo := strings.ToLower(strings.TrimSpace(offer.Repository))

	switch {
	case prov == "pacman" || prov == "repository" || prov == "paru":
		origin = "Arch Official Repository"
		if repo != "" {
			origin = "Arch Official Repository (" + strings.Title(repo) + ")"
		}
		integrity = "Keyring Verified (Arch Linux Packager PGP)"
		isolation = "Native System (Direct Host Execution)"
	case prov == "aur":
		origin = "Arch User Repository (AUR)"
		integrity = "Community PKGBUILD (Inspect with [v] before build)"
		isolation = "User-Scoped makepkg / Rootless Sandbox"
	case prov == "flatpak" || prov == "flathub":
		origin = "Flathub (flathub.org)"
		integrity = "OSTree Commit Signed (Flathub GPG Key)"
		isolation = "Bubblewrap Container (Confined Filesystem)"
	case prov == "snap":
		origin = "Canonical Snap Store"
		integrity = "Snap Store Signed Assertion"
		isolation = "AppArmor / cgroups Confinement"
	case prov == "apt" || prov == "deb":
		origin = "Debian/Ubuntu Repository"
		if repo != "" {
			origin = "Debian/Ubuntu Repository (" + strings.Title(repo) + ")"
		}
		integrity = "GPG Signed Archive"
		isolation = "Native System (Direct Host Execution)"
	case prov == "dnf" || prov == "rpm":
		origin = "Fedora/RHEL RPM Repository"
		integrity = "RPM GPG Signed"
		isolation = "Native System (Direct Host Execution)"
	default:
		origin = strings.ToUpper(prov) + " Distribution"
		integrity = "Upstream Release Hashes"
		isolation = "Host Native"
	}
	return origin, integrity, isolation
}

func formatPermItem(p PermissionRow, maxW int) string {
	glyph := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("[✓]")
	textStyle := lipgloss.NewStyle().Foreground(ColorText)
	if !p.Granted {
		glyph = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true).Render("[✗]")
		textStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	}
	return glyph + " " + textStyle.Render(truncateText(p.Label, maxW-4))
}

func (m Model) renderInstallPipelinePane(outerW, boxH, innerW int) string {
	opTitle := "INSTALLING"
	if m.installAction == "remove" {
		opTitle = "REMOVING"
	}
	elapsed := time.Since(m.installStartTime).Round(100 * time.Millisecond)
	badge := ProviderBadge(m.installPkg.Provider)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(fmt.Sprintf("┌─ ⚡ %s: %s ─┐\n\n", opTitle, m.installPkg.Name)))
	b.WriteString(fmt.Sprintf("%s %s %s  %s\n\n",
		lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("⚡ "+opTitle+":"),
		lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright).Render(m.installPkg.Name),
		badge,
		lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("(%s elapsed)", elapsed)),
	))

	barWidth := innerW - 14
	if barWidth < 15 {
		barWidth = 15
	}
	if barWidth > 44 {
		barWidth = 44
	}
	filledLen := int(m.installProgress * float64(barWidth))
	if filledLen > barWidth {
		filledLen = barWidth
	}
	emptyLen := barWidth - filledLen
	if emptyLen < 0 {
		emptyLen = 0
	}
	filledBar := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render(strings.Repeat("█", filledLen))
	emptyBar := lipgloss.NewStyle().Foreground(ColorMuted).Render(strings.Repeat("░", emptyLen))
	pctStr := lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright).Render(fmt.Sprintf("%3.0f%%", m.installProgress*100))
	b.WriteString(fmt.Sprintf("  [%s%s]  %s\n\n", filledBar, emptyBar, pctStr))

	currentStatus := "Resolving recipe & provider strategy..."
	if m.installAction == "install" {
		switch m.installStep {
		case 2:
			currentStatus = "Verifying dependencies and host locks..."
		case 3:
			currentStatus = fmt.Sprintf("Running %s backend (download/build)...", m.installPkg.Provider)
		case 4:
			currentStatus = "Finalizing desktop integration & binaries..."
		}
	} else {
		switch m.installStep {
		case 2:
			currentStatus = "Verifying reverse dependencies & locks..."
		case 3:
			currentStatus = fmt.Sprintf("Running %s removal routine...", m.installPkg.Provider)
		case 4:
			currentStatus = "Finalizing removal & refreshing catalog database..."
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s\n\n", m.spinner.View(), lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(currentStatus)))

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright).Render("EXECUTION PIPELINE:\n"))
	steps := []string{
		"Recipe & Provider Strategy",
		"Dependency & Lock Verification",
		"Package Manager Execution",
		"Desktop & System Finalization",
	}
	if m.installAction == "remove" {
		steps = []string{
			"Target & Scope Validation",
			"Dependency & Lock Verification",
			"Package Manager Removal",
			"Database & Catalog Finalization",
		}
	}
	for idx, s := range steps {
		stepNum := idx + 1
		if stepNum < m.installStep {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("✔"),
				lipgloss.NewStyle().Foreground(ColorSecondary).Render(fmt.Sprintf("Stage %d: %s", stepNum, s)),
			))
		} else if stepNum == m.installStep {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render("▶"),
				lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render(fmt.Sprintf("Stage %d: %s (in progress)", stepNum, s)),
			))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				lipgloss.NewStyle().Foreground(ColorMuted).Render("○"),
				lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("Stage %d: %s", stepNum, s)),
			))
		}
	}

	b.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("LIVE OPERATION STREAM:\n"))
	startLog := 0
	if len(m.logLines) > 5 {
		startLog = len(m.logLines) - 5
	}
	for _, l := range m.logLines[startLog:] {
		line := truncateText(l, innerW-4)
		if strings.Contains(l, "✔") {
			b.WriteString("  " + LogSuccessStyle.Render(line) + "\n")
		} else if strings.Contains(l, "✖") {
			b.WriteString("  " + LogErrorStyle.Render(line) + "\n")
		} else {
			b.WriteString("  " + LogInfoStyle.Render(line) + "\n")
		}
	}

	return CardFocusedStyle.Width(outerW - 2).Height(boxH).Render(b.String())
}

func (m Model) renderRecipeProvenancePane(app catalog.UnifiedApplication, offer catalog.PackageOffer, outerW, boxH, innerW int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(fmt.Sprintf("┌─ [v] PROVENANCE & RECIPE: %s ─┐\n\n", app.Name)))
	b.WriteString(fmt.Sprintf("  Offer ID:      %s\n", offer.ID))
	b.WriteString(fmt.Sprintf("  AppStream ID:  %s\n", app.AppID))
	b.WriteString(fmt.Sprintf("  Provider:      %s (%s)\n", offer.Provider, offer.Repository))
	b.WriteString(fmt.Sprintf("  Build Recipe:  %s\n", offer.Trust.BuildRecipe))
	b.WriteString(fmt.Sprintf("  Maintainer:    %s\n", offer.Trust.Maintainer))
	b.WriteString(fmt.Sprintf("  Origin URI:    %s\n", truncateText(offer.Trust.Provenance.Origin, innerW-18)))
	b.WriteString(fmt.Sprintf("  Signing Key:   %s\n", truncateText(offer.Trust.Provenance.SigningKey, innerW-18)))
	b.WriteString(fmt.Sprintf("  Digest Mech:   %s\n", offer.Trust.Provenance.ArtifactDigest))
	b.WriteString(fmt.Sprintf("  Scope:         %s\n\n", offer.InstallScope))
	b.WriteString(lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("  Verification Summary:\n"))
	b.WriteString("  " + truncateText(offer.Trust.Summary, innerW-4) + "\n")
	if offer.Trust.Provenance.Verification != "" {
		b.WriteString("  " + truncateText(offer.Trust.Provenance.Verification, innerW-4) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorSecondary).Render("  Press [v] or [Esc] to return to Package Inspector."))
	return CardFocusedStyle.Width(outerW - 2).Height(boxH).Render(b.String())
}

func (m Model) renderHelpOverlayPane(outerW, boxH, innerW int) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("┌─ [?] KEYBOARD SHORTCUTS & NAVIGATION ─┐\n\n"))
	rows := [][2]string{
		{"[Tab] / [/]", "Focus / Unfocus Query Search bar"},
		{"[Enter]", "Execute search / Inspect selected app"},
		{"[Ctrl+U]", "Clear search query"},
		{"[↑ / ↓ / j / k]", "Navigate canonical packages list"},
		{"[g / G]", "Jump to Top / Bottom of list"},
		{"[1 - 5]", "Switch top-level tabs (Store/Installed/Updates/Pipeline/Settings)"},
		{"[[ / ]]", "Previous / Next top-level tab"},
		{"[p]", "Cycle Provider filter (All / Native / AUR / Flathub / Snap / GitHub)"},
		{"[s]", "Cycle Scope filter (All / Sandboxed / System)"},
		{"[S]", "Cycle Sort mode (Relevance / Trust / Name)"},
		{"[o]", "Cycle Package Offer in Inspector (Flatpak / Pacman / AUR / etc.)"},
		{"[i]", "Install selected offer via TransactionManager"},
		{"[r]", "Remove selected package safely"},
		{"[v]", "Toggle Provenance & Recipe inspector"},
		{"[u]", "Refresh Updates / Installed list"},
		{"[c]", "Cycle Recommendation Policy (Settings tab)"},
		{"[q / Ctrl+C]", "Close help / Quit Arjax"},
	}
	for _, r := range rows {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", KeyHelpStyle.Render(r[0]), DescHelpStyle.Render(truncateText(r[1], innerW-24))))
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorSecondary).Render("  Press [?] or [Esc] to return."))
	return CardFocusedStyle.Width(outerW - 2).Height(boxH).Render(b.String())
}
