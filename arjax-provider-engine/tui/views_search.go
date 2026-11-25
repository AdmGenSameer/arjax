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

	// Each card takes 4 content lines + 1 spacer = 5 lines. Footer takes 1 line.
	const cardLines = 5
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
	cardInnerW := innerW - 3
	if cardInnerW < 20 {
		cardInnerW = 20
	}

	// Row 1: Cursor + Index + Name + Checkmark + Version + Badge
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
		verStr = lipgloss.NewStyle().Foreground(ColorMuted).Render(" (" + truncateText(offer.Version, 16) + ")")
	}

	badgeText := strings.ToUpper(offer.Provider)
	if offer.RecommendationBadge == "Recommended" && offer.Score >= 85 {
		badgeText = strings.ToUpper(offer.Provider) + " ★ REC"
	}
	badge := ProviderBadge(offer.Provider)
	if offer.RecommendationBadge == "Recommended" && selected {
		badge = RecommendBadgeStyle.Render(badgeText)
	}

	nameMax := cardInnerW - lipgloss.Width(badge) - 18
	if nameMax < 8 {
		nameMax = 8
	}
	leftTitle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(selected).Render(prefix) +
		nameStyle.Render(truncateText(app.Name, nameMax)) +
		installedMark + verStr

	gap1 := cardInnerW - lipgloss.Width(leftTitle) - lipgloss.Width(badge)
	if gap1 < 1 {
		gap1 = 1
	}
	row1 := leftTitle + strings.Repeat(" ", gap1) + badge

	// Row 2: Summary
	summary := app.Summary
	if summary == "" {
		summary = app.Description
	}
	if summary == "" {
		summary = "Linux software package"
	}
	row2 := "   " + lipgloss.NewStyle().Foreground(ColorText).Render(truncateText(summary, cardInnerW-3))

	// Row 3: Offers Chips
	var chipParts []string
	chipParts = append(chipParts, lipgloss.NewStyle().Foreground(ColorMuted).Render("   Offers: "))
	usedW := 11
	for _, o := range app.Offers {
		lbl := strings.Title(o.Provider)
		if o.Repository != "" && o.Provider == "pacman" {
			lbl = "Pacman " + strings.Title(o.Repository)
		}
		var chip string
		if o.ID == offer.ID {
			chip = OfferChipActiveStyle.Render(lbl)
		} else {
			chip = OfferChipStyle.Render(lbl)
		}
		cw := lipgloss.Width(chip) + 1
		if usedW+cw > cardInnerW {
			break
		}
		chipParts = append(chipParts, chip, " ")
		usedW += cw
	}
	row3 := lipgloss.JoinHorizontal(lipgloss.Center, chipParts...)

	// Row 4: Install state, Size, Arch/Trust
	stateStr := lipgloss.NewStyle().Foreground(ColorMuted).Render("[Available]")
	if offer.IsInstalled || m.isOfferInstalledOnHost(offer) {
		stateStr = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render("[INSTALLED: " + strings.Title(offer.Provider) + "]")
	}
	sizeStr := offer.DownloadSize
	if sizeStr == "" {
		sizeStr = "~45 MB"
	}
	score, _ := evaluateOfferTrustScore(offer)
	metaRight := lipgloss.NewStyle().Foreground(ColorMuted).Render("Size: ") +
		lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render(sizeStr) +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("  Trust: %d/100", score))

	gap4 := cardInnerW - lipgloss.Width(stateStr) - lipgloss.Width(metaRight)
	if gap4 < 1 {
		gap4 = 1
	}
	row4 := stateStr + strings.Repeat(" ", gap4) + metaRight

	cardContent := lipgloss.JoinVertical(lipgloss.Left, row1, row2, row3, row4)
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

	// 1. App Header & Origin Identity Card
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

	titleLeft := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render(app.Name) + " " +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(offer.Version)
	titleRight := FilterInactiveStyle.Render(license) + " " +
		lipgloss.NewStyle().Foreground(ColorOnPrimary).Background(ColorPrimarySoft).Bold(true).Padding(0, 1).Render(sizeBadge)
	tGap := innerW - 2 - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight)
	if tGap < 1 {
		tGap = 1
	}
	originLine1 := titleLeft + strings.Repeat(" ", tGap) + titleRight
	originLine2 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Upstream: ") +
		lipgloss.NewStyle().Foreground(ColorText).Render(truncateText(dev, 24)) +
		lipgloss.NewStyle().Foreground(ColorMuted).Render(" • ID: ") +
		lipgloss.NewStyle().Foreground(ColorPrimarySoft).Render(truncateText(app.AppID, innerW-40))
	originLine3 := lipgloss.NewStyle().Foreground(ColorMuted).Render("Homepage: ") +
		lipgloss.NewStyle().Foreground(ColorHighlight).Underline(true).Render(truncateText(homepage, innerW-14))
	sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, originLine1, originLine2, originLine3)))

	// 2. Trust & Integrity Score Card
	score, trustTag := evaluateOfferTrustScore(offer)
	trustBadge := TrustBadgeLowStyle.Render(trustTag)
	if score >= 90 {
		trustBadge = TrustBadgeHighStyle.Render(trustTag)
	} else if score >= 65 {
		trustBadge = TrustBadgeMedStyle.Render(trustTag)
	}
	sigStatus := offer.SignatureStatus
	if sigStatus == "" {
		sigStatus = offer.Trust.SignatureStatus
	}
	if sigStatus == "" {
		sigStatus = "unverified"
	}
	isoLabel := "Host System Linkage"
	if offer.InstallScope == catalog.ScopeSandboxed {
		isoLabel = "Bubblewrap / Confined Sandbox"
	} else if offer.InstallScope == catalog.ScopeUser {
		isoLabel = "User-Scoped Build"
	}

	trustTitle := lipgloss.NewStyle().Foreground(ColorSecondary).Render("🛡 ") +
		lipgloss.NewStyle().Foreground(ColorTextBright).Bold(true).Render("Trust & Integrity: ") +
		lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(fmt.Sprintf("%d/100 ", score)) +
		trustBadge
	trustGrid1 := lipgloss.NewStyle().Foreground(ColorSecondary).Render("✔ ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Sig: ") +
		lipgloss.NewStyle().Foreground(ColorText).Bold(true).Render(truncateText(sigStatus, 22)) +
		"   " +
		lipgloss.NewStyle().Foreground(ColorSecondary).Render("✔ ") +
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Isolation: ") +
		lipgloss.NewStyle().Foreground(ColorText).Bold(true).Render(truncateText(isoLabel, 26))
	sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, trustTitle, trustGrid1)))

	// 3. Select Offer / Distribution Target
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
			offerRows = append(offerRows, lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("   ... +%d more offers (press [o] to cycle)", len(app.Offers)-maxOffers)))
			break
		}
		radio := "( )"
		rowStyle := lipgloss.NewStyle().Foreground(ColorText)
		if o.ID == offer.ID {
			radio = lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("(*)")
			rowStyle = lipgloss.NewStyle().Foreground(ColorTextBright).Bold(true)
		}
		recTag := ""
		if o.RecommendationBadge != "" {
			recTag = " (" + o.RecommendationBadge + ")"
		}
		instTag := ""
		if o.IsInstalled || m.isOfferInstalledOnHost(o) {
			instTag = lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(" [INSTALLED]")
		}
		line := fmt.Sprintf("%s %s %s%s%s  %s",
			radio,
			ProviderBadge(o.Provider),
			rowStyle.Render(truncateText(o.Version, 16)),
			lipgloss.NewStyle().Foreground(ColorMuted).Render(recTag),
			instTag,
			lipgloss.NewStyle().Foreground(ColorMuted).Render(o.DownloadSize),
		)
		offerRows = append(offerRows, clampStyledWidth(line, innerW-2))
	}
	sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, offerRows...)))

	// 4. Sandbox Permissions Matrix (render if vertical space permits)
	if boxH >= 24 {
		perms := formatOfferPermissions(offer)
		permHdr := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("SANDBOX PERMISSIONS MATRIX:")
		var permLines []string
		permLines = append(permLines, permHdr)
		for i := 0; i < len(perms); i += 2 {
			leftP := formatPermItem(perms[i], (innerW-6)/2)
			if i+1 < len(perms) {
				rightP := formatPermItem(perms[i+1], (innerW-6)/2)
				permLines = append(permLines, leftP+"  "+rightP)
			} else {
				permLines = append(permLines, leftP)
			}
		}
		sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, permLines...)))
	}

	// 5. Strategy Resolution / Dependency Tree
	if boxH >= 20 {
		treeHdr := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("STRATEGY RESOLUTION TREE:")
		var deps []string
		if len(offer.Dependencies) > 0 {
			deps = offer.Dependencies
		} else if info, ok := m.pkgInfo[offer.PackageName]; ok && len(info.Dependencies) > 0 {
			deps = info.Dependencies
		}
		var treeLines []string
		treeLines = append(treeLines, treeHdr)
		treeLines = append(treeLines, lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true).Render("📦 "+offer.PackageName+" ("+offer.Repository+")"))
		if len(deps) > 0 {
			maxDeps := 3
			for i, d := range deps {
				if i >= maxDeps {
					treeLines = append(treeLines, lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf(" └── +%d more dependencies", len(deps)-maxDeps)))
					break
				}
				branch := "├──"
				if i == len(deps)-1 || i == maxDeps-1 && len(deps) == maxDeps {
					branch = "└──"
				}
				treeLines = append(treeLines, " "+BranchTreeStyle.Render(branch)+" "+lipgloss.NewStyle().Foreground(ColorText).Render(d))
			}
		} else {
			treeLines = append(treeLines, lipgloss.NewStyle().Foreground(ColorMuted).Render(" └── Bundled / zero additional host dependencies"))
		}
		sections = append(sections, SectionCardStyle.Width(innerW).Render(lipgloss.JoinVertical(lipgloss.Left, treeLines...)))
	}

	// 6. Read-Only Executable Command Preview
	cmdPreview := offer.CommandPreview
	if cmdPreview == "" {
		cmdPreview = catalog.FormatCommandPreview(offer.Provider, offer.Repository, offer.PackageName, offer.Branch, offer.Channel)
	}
	cmdLine := lipgloss.NewStyle().Foreground(ColorMuted).Render("PREVIEW: ") +
		CommandPreviewBoxStyle.Render("$ "+truncateText(cmdPreview, innerW-14))
	sections = append(sections, cmdLine)

	// 7. Action Buttons Row
	isInstalled := offer.IsInstalled || m.isOfferInstalledOnHost(offer)
	primaryBtnLabel := "[ i: Install " + truncateText(offer.PackageName, 16) + " ]"
	if isInstalled {
		primaryBtnLabel = "[ Enter: Refresh / Launch ]"
	}
	btnRow := lipgloss.JoinHorizontal(
		lipgloss.Center,
		ActionBtnPrimaryStyle.Render(primaryBtnLabel),
		" ",
		ActionBtnSecondaryStyle.Render("[ i: Install/Switch ]"),
		" ",
		ActionBtnDangerStyle.Render("[ r: Remove ]"),
		" ",
		FilterInactiveStyle.Render("[ v: Recipe ]"),
	)
	sections = append(sections, clampStyledWidth(btnRow, innerW))

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return CardStyle.Width(outerW - 2).Height(boxH).MaxHeight(boxH + 2).Render(content)
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
			currentStatus = "Verifying reverse dependencies..."
		case 3:
			currentStatus = fmt.Sprintf("Running %s removal routine...", m.installPkg.Provider)
		case 4:
			currentStatus = "Cleaning orphaned files & desktop caches..."
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
			"Package Binary Inspection",
			"Reverse Dependency Check",
			"Removal Execution",
			"Orphan & Cache Cleanup",
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
