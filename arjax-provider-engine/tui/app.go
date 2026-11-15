package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type searchDoneMsg struct {
	query    string
	packages []models.Package
	duration time.Duration
	err      error
}

type actionDoneMsg struct {
	action string
	result models.Result
	err    error
}

type infoDoneMsg struct {
	name string
	info models.PackageInfo
	err  error
}

type Model struct {
	pm           *providers.ProviderManager
	distro       string
	distroName   string
	textInput    textinput.Model
	spinner      spinner.Model
	loading      bool
	installing   bool
	statusMsg    string
	statusType   string // "info", "success", "error", "warning"
	packages     []models.Package
	pkgInfo      map[string]models.PackageInfo
	cursor       int
	scrollOffset int
	width        int
	height       int
	activeTab    int // 0: Overview & Deps, 1: Activity Logs, 2: Providers
	logLines     []string
	showHelp     bool
	err          error
}

func NewModel(pm *providers.ProviderManager) Model {
	ti := textinput.New()
	ti.Placeholder = "Type to search (e.g. zapfast, brave, neovim) — Press Enter to search..."
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorTextBright)
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorHighlight)

	distroInfo := utils.DetectDistro()
	distroPM := utils.DetectPackageManager()
	if distroPM == "" {
		distroPM = "linux"
	}
	distroLabel := fmt.Sprintf("%s (%s)", strings.ToUpper(distroPM), distroInfo.Name)
	if distroInfo.Name == "" {
		distroLabel = strings.ToUpper(distroPM)
	}

	return Model{
		pm:         pm,
		distro:     distroLabel,
		distroName: distroInfo.Name,
		textInput:  ti,
		spinner:    s,
		packages:   []models.Package{},
		pkgInfo:    make(map[string]models.PackageInfo),
		activeTab:  0,
		statusMsg:  "Ready. Type a query and press Enter, or use [Tab] to browse.",
		statusType: "info",
		logLines: []string{
			fmt.Sprintf("[%s] Arjax Go Provider Engine initialized.", time.Now().Format("15:04:05")),
			fmt.Sprintf("[%s] Detected distribution environment: %s", time.Now().Format("15:04:05"), distroLabel),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 12
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		// Global quit
		if keyStr == "ctrl+c" {
			return m, tea.Quit
		}

		// Help overlay toggle
		if m.showHelp {
			if keyStr == "?" || keyStr == "esc" || keyStr == "q" || keyStr == "enter" {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		// Navigation when search input is NOT focused
		if !m.textInput.Focused() {
			switch keyStr {
			case "q":
				return m, tea.Quit

			case "?":
				m.showHelp = true
				return m, nil

			case "tab", "/":
				m.textInput.Focus()
				return m, textinput.Blink

			case "1":
				m.activeTab = 0
				return m, nil
			case "2":
				m.activeTab = 1
				return m, nil
			case "3":
				m.activeTab = 2
				return m, nil
			case "[":
				m.activeTab = (m.activeTab + 2) % 3
				return m, nil
			case "]":
				m.activeTab = (m.activeTab + 1) % 3
				return m, nil

			case "up", "k":
				if len(m.packages) > 0 && m.cursor > 0 {
					m.cursor--
					m.adjustScroll()
					cmds = append(cmds, m.fetchInfoForSelected())
				}
				return m, tea.Batch(cmds...)

			case "down", "j":
				if len(m.packages) > 0 && m.cursor < len(m.packages)-1 {
					m.cursor++
					m.adjustScroll()
					cmds = append(cmds, m.fetchInfoForSelected())
				}
				return m, tea.Batch(cmds...)

			case "g", "home":
				if len(m.packages) > 0 {
					m.cursor = 0
					m.scrollOffset = 0
					cmds = append(cmds, m.fetchInfoForSelected())
				}
				return m, tea.Batch(cmds...)

			case "G", "end":
				if len(m.packages) > 0 {
					m.cursor = len(m.packages) - 1
					m.adjustScroll()
					cmds = append(cmds, m.fetchInfoForSelected())
				}
				return m, tea.Batch(cmds...)

			case "i":
				if len(m.packages) > 0 && !m.installing && !m.loading {
					pkg := m.packages[m.cursor]
					m.installing = true
					m.statusMsg = fmt.Sprintf("Installing %s via %s...", pkg.Name, pkg.Provider)
					m.statusType = "warning"
					m.logLines = append(m.logLines, fmt.Sprintf("[%s] ▶ Install started: %s (%s)", time.Now().Format("15:04:05"), pkg.Name, pkg.Provider))
					return m, tea.Batch(m.spinner.Tick, m.runInstall(pkg))
				}

			case "r":
				if len(m.packages) > 0 && !m.installing && !m.loading {
					pkg := m.packages[m.cursor]
					m.installing = true
					m.statusMsg = fmt.Sprintf("Removing %s...", pkg.Name)
					m.statusType = "warning"
					m.logLines = append(m.logLines, fmt.Sprintf("[%s] ▶ Remove started: %s", time.Now().Format("15:04:05"), pkg.Name))
					return m, tea.Batch(m.spinner.Tick, m.runRemove(pkg))
				}
			}
		} else {
			// Keybindings when Search Input IS focused
			switch keyStr {
			case "esc":
				m.textInput.Blur()
				return m, nil

			case "tab":
				m.textInput.Blur()
				return m, nil

			case "enter":
				query := strings.TrimSpace(m.textInput.Value())
				if query != "" {
					m.loading = true
					m.statusMsg = fmt.Sprintf("Searching for '%s' across all providers...", query)
					m.statusType = "info"
					m.textInput.Blur()
					return m, tea.Batch(m.spinner.Tick, m.runSearch(query))
				}
			}
		}

	case searchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Search error: %v", msg.err)
			m.statusType = "error"
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] ✖ Search failed for '%s': %v", time.Now().Format("15:04:05"), msg.query, msg.err))
		} else {
			m.packages = msg.packages
			m.cursor = 0
			m.scrollOffset = 0
			elapsed := msg.duration.Round(time.Millisecond)
			m.statusMsg = fmt.Sprintf("✔ Found %d package(s) for '%s' in %v", len(msg.packages), msg.query, elapsed)
			m.statusType = "success"
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] ✔ Found %d package(s) for '%s' (%v)", time.Now().Format("15:04:05"), len(msg.packages), msg.query, elapsed))
			if len(msg.packages) > 0 {
				cmds = append(cmds, m.runFetchInfo(msg.packages[0].Name))
			}
		}

	case infoDoneMsg:
		if msg.err == nil {
			m.pkgInfo[msg.name] = msg.info
		}

	case actionDoneMsg:
		m.installing = false
		if msg.err != nil || !msg.result.Success {
			errStr := msg.result.Error
			if errStr == "" && msg.err != nil {
				errStr = msg.err.Error()
			}
			m.statusMsg = fmt.Sprintf("✖ %s failed: %s", strings.Title(msg.action), errStr)
			m.statusType = "error"
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] ✖ %s failed for %s: %s", time.Now().Format("15:04:05"), strings.Title(msg.action), msg.result.Package, errStr))
		} else {
			m.statusMsg = fmt.Sprintf("✔ %s completed for %s (%d ms)", strings.Title(msg.action), msg.result.Package, msg.result.DurationMs)
			m.statusType = "success"
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] ✔ %s completed: %s via %s (%d ms)", time.Now().Format("15:04:05"), strings.Title(msg.action), msg.result.Package, msg.result.Provider, msg.result.DurationMs))
		}

	case spinner.TickMsg:
		if m.loading || m.installing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.textInput.Focused() {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) adjustScroll() {
	paneHeight := m.height - 11
	if paneHeight < 8 {
		paneHeight = 8
	}
	visibleRows := paneHeight - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+visibleRows {
		m.scrollOffset = m.cursor - visibleRows + 1
	}
}

func (m Model) fetchInfoForSelected() tea.Cmd {
	if len(m.packages) > 0 && m.cursor < len(m.packages) {
		name := m.packages[m.cursor].Name
		if _, exists := m.pkgInfo[name]; !exists {
			return m.runFetchInfo(name)
		}
	}
	return nil
}

func (m Model) runSearch(query string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		pkgs, err := m.pm.SearchAll(ctx, query)
		return searchDoneMsg{
			query:    query,
			packages: pkgs,
			duration: time.Since(start),
			err:      err,
		}
	}
}

func (m Model) runInstall(pkg models.Package) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		res, err := m.pm.Install(ctx, pkg)
		return actionDoneMsg{action: "install", result: res, err: err}
	}
}

func (m Model) runRemove(pkg models.Package) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		res, err := m.pm.Remove(ctx, pkg)
		return actionDoneMsg{action: "remove", result: res, err: err}
	}
}

func (m Model) runFetchInfo(pkgName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		info, err := m.pm.Info(ctx, pkgName)
		return infoDoneMsg{name: pkgName, info: info, err: err}
	}
}

func (m Model) View() string {
	w := m.width
	if w < 70 {
		w = 70
	}
	h := m.height
	if h < 20 {
		h = 20
	}

	// 1. Header Bar
	headerLeft := lipgloss.JoinHorizontal(
		lipgloss.Center,
		AppTitleStyle.Render("⚡ ARJAX TUI"),
		BadgeStyle.Render(m.distro),
		SubTitleStyle.Render("Universal Package Helper Engine"),
	)

	// Tab Bar
	tabTitles := []string{"1: Details & Deps", "2: Logs", "3: Providers"}
	var tabs []string
	for i, t := range tabTitles {
		if i == m.activeTab {
			tabs = append(tabs, ActiveTabStyle.Render(t))
		} else {
			tabs = append(tabs, InactiveTabStyle.Render(t))
		}
	}
	headerRight := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)

	spaceGap := w - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 2
	if spaceGap < 1 {
		spaceGap = 1
	}
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		headerLeft,
		strings.Repeat(" ", spaceGap),
		headerRight,
	)

	// 2. Search Input Box
	var searchBox string
	if m.textInput.Focused() {
		searchBox = SearchFocusedStyle.Width(w - 4).Render(m.textInput.View())
	} else {
		searchBox = SearchBlurredStyle.Width(w - 4).Render(m.textInput.View())
	}

	// 3. Status Line
	var statusLine string
	statusColor := ColorHighlight
	switch m.statusType {
	case "success":
		statusColor = ColorSecondary
	case "error":
		statusColor = ColorDanger
	case "warning":
		statusColor = ColorWarning
	}

	statusContent := m.statusMsg
	if m.loading || m.installing {
		statusContent = fmt.Sprintf("%s %s", m.spinner.View(), m.statusMsg)
	}
	statusLine = lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render("  " + statusContent)

	// 4. Compute Layout Dimensions
	paneHeight := h - 11
	if paneHeight < 8 {
		paneHeight = 8
	}

	listWidth := (w * 46) / 100
	if listWidth < 32 {
		listWidth = 32
	}
	detailsWidth := w - listWidth - 5
	if detailsWidth < 34 {
		detailsWidth = 34
	}

	visibleRows := paneHeight - 2
	if visibleRows < 1 {
		visibleRows = 1
	}

	// 5. Left Pane: Package List
	var listContent strings.Builder
	if len(m.packages) == 0 {
		listContent.WriteString(
			lipgloss.NewStyle().Foreground(ColorMuted).Render(
				"\n  No packages found.\n  Press '/' or [Tab] to search.",
			),
		)
	} else {
		endIdx := m.scrollOffset + visibleRows
		if endIdx > len(m.packages) {
			endIdx = len(m.packages)
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			pkg := m.packages[i]
			badge := ProviderBadge(pkg.Provider)
			nameMaxLen := listWidth - lipgloss.Width(badge) - len(pkg.Version) - 10
			if nameMaxLen < 8 {
				nameMaxLen = 8
			}

			displayName := pkg.Name
			if len(displayName) > nameMaxLen {
				displayName = displayName[:nameMaxLen-1] + "…"
			}

			line := fmt.Sprintf("%-*s  %s  %s", nameMaxLen, displayName, badge, pkg.Version)

			if i == m.cursor && !m.textInput.Focused() {
				listContent.WriteString(ListItemSelectedStyle.Width(listWidth-4).Render("▶ "+line) + "\n")
			} else if i == m.cursor && m.textInput.Focused() {
				listContent.WriteString(
					lipgloss.NewStyle().Foreground(ColorHighlight).Bold(true).Render("▷ "+line) + "\n",
				)
			} else {
				listContent.WriteString(ListItemStyle.Width(listWidth-4).Render("  "+line) + "\n")
			}
		}
	}

	// Left Pane Header with counter
	listTitleText := "PACKAGES"
	if len(m.packages) > 0 {
		listTitleText = fmt.Sprintf("PACKAGES (%d/%d)", m.cursor+1, len(m.packages))
		if m.scrollOffset > 0 || m.scrollOffset+visibleRows < len(m.packages) {
			listTitleText += " [▲/▼]"
		}
	}
	listHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(listTitleText)

	leftBoxStyle := CardStyle
	if !m.textInput.Focused() {
		leftBoxStyle = CardFocusedStyle
	}
	leftPane := leftBoxStyle.Width(listWidth).Height(paneHeight).Render(
		lipgloss.JoinVertical(lipgloss.Left, listHeader, "\n", listContent.String()),
	)

	// 6. Right Pane: Tab Content
	var rightContent strings.Builder

	if m.showHelp {
		// Help Overlay View
		rightContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("KEYBOARD SHORTCUTS & NAVIGATION:\n\n"))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[Tab] / [/]"), DescHelpStyle.Render("Focus / Unfocus Search bar")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[Enter]"), DescHelpStyle.Render("Execute search / Inspect package")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[↑ / ↓ / j / k]"), DescHelpStyle.Render("Navigate packages list")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[g / G]"), DescHelpStyle.Render("Jump to Top / Bottom of list")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[1 / 2 / 3]"), DescHelpStyle.Render("Switch Inspector tabs (Details/Logs/Engine)")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[ [ / ] ]"), DescHelpStyle.Render("Previous / Next tab")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[i]"), DescHelpStyle.Render("Install selected package")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[r]"), DescHelpStyle.Render("Remove selected package")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n", KeyHelpStyle.Render("[?]"), DescHelpStyle.Render("Toggle this Help screen")))
		rightContent.WriteString(fmt.Sprintf("  %-16s %s\n\n", KeyHelpStyle.Render("[q / Esc]"), DescHelpStyle.Render("Close help / Quit Arjax")))
		rightContent.WriteString(lipgloss.NewStyle().Foreground(ColorSecondary).Render("Press [?] or [Esc] to return to inspector."))
	} else {
		switch m.activeTab {
		case 0:
			// Details & Dependencies Tab
			if len(m.packages) > 0 && m.cursor < len(m.packages) {
				selected := m.packages[m.cursor]
				badge := ProviderBadge(selected.Provider)

				rightContent.WriteString(fmt.Sprintf("%s  %s\n", lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright).Render(selected.Name), badge))
				rightContent.WriteString(fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(ColorMuted).Render("Version:    "), selected.Version))

				if selected.Repo != "" {
					rightContent.WriteString(fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(ColorMuted).Render("Repository: "), selected.Repo))
				}
				if selected.Description != "" {
					rightContent.WriteString(fmt.Sprintf("%s %s\n", lipgloss.NewStyle().Foreground(ColorMuted).Render("Description:"), selected.Description))
				}

				// Dependency Tree
				rightContent.WriteString("\n" + BranchTreeStyle.Bold(true).Render("DEPENDENCY GRAPH & STRATEGY:\n"))
				if info, ok := m.pkgInfo[selected.Name]; ok {
					rightContent.WriteString(fmt.Sprintf("📦 %s\n", selected.Name))
					if len(info.Dependencies) > 0 {
						for i, dep := range info.Dependencies {
							branch := "├──"
							if i == len(info.Dependencies)-1 {
								branch = "└──"
							}
							rightContent.WriteString(fmt.Sprintf(" %s 🔗 %s\n", BranchTreeStyle.Render(branch), dep))
						}
					} else {
						rightContent.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(" └── (No explicit dependencies listed)\n"))
					}
				} else {
					rightContent.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(" └── ⏳ Loading dependency metadata...\n"))
				}

				// Quick action hints
				rightContent.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorSecondary).Render("⚡ Press [i] to Install  •  [r] to Remove"))
			} else {
				rightContent.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("\n  Select a package from the left pane\n  to inspect metadata and dependencies."))
			}

		case 1:
			// Activity Logs Tab
			rightContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("ENGINE ACTIVITY & OPERATION LOGS:\n\n"))
			startLog := 0
			maxLogs := paneHeight - 4
			if len(m.logLines) > maxLogs {
				startLog = len(m.logLines) - maxLogs
			}
			for _, l := range m.logLines[startLog:] {
				if strings.Contains(l, "✔") {
					rightContent.WriteString(LogSuccessStyle.Render(l) + "\n")
				} else if strings.Contains(l, "✖") {
					rightContent.WriteString(LogErrorStyle.Render(l) + "\n")
				} else {
					rightContent.WriteString(LogInfoStyle.Render(l) + "\n")
				}
			}

		case 2:
			// Providers Status Tab
			rightContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("SYSTEM ENGINE & REGISTERED PROVIDERS:\n\n"))
			rightContent.WriteString(fmt.Sprintf("  • %-20s %s\n", "Operating System:", m.distro))
			rightContent.WriteString(fmt.Sprintf("  • %-20s %s\n", "Engine Mode:", "Go High-Performance Provider Core"))
			rightContent.WriteString(fmt.Sprintf("  • %-20s %s\n\n", "Memory Cache:", "Active (TTL Enabled)"))

			rightContent.WriteString(lipgloss.NewStyle().Bold(true).Foreground(ColorTextBright).Render("ACTIVE PROVIDERS:\n"))
			for _, pName := range m.pm.GetSupportedProviderNames() {
				rightContent.WriteString(fmt.Sprintf("  ✔ %-16s %s\n", pName, ProviderBadge(pName)))
			}
		}
	}

	tabTitle := tabTitles[m.activeTab]
	rightHeader := lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render(tabTitle)
	rightPane := CardStyle.Width(detailsWidth).Height(paneHeight).Render(
		lipgloss.JoinVertical(lipgloss.Left, rightHeader, "\n", rightContent.String()),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// 7. Footer / Keyboard Shortcuts Bar
	footer := lipgloss.JoinHorizontal(
		lipgloss.Left,
		KeyHelpStyle.Render(" [Tab]"), DescHelpStyle.Render(" Search/List "),
		KeyHelpStyle.Render(" [/]"), DescHelpStyle.Render(" Search "),
		KeyHelpStyle.Render(" [1-3]"), DescHelpStyle.Render(" Tabs "),
		KeyHelpStyle.Render(" [↑/↓/j/k]"), DescHelpStyle.Render(" Browse "),
		KeyHelpStyle.Render(" [i]"), DescHelpStyle.Render(" Install "),
		KeyHelpStyle.Render(" [r]"), DescHelpStyle.Render(" Remove "),
		KeyHelpStyle.Render(" [?]"), DescHelpStyle.Render(" Help "),
		KeyHelpStyle.Render(" [q]"), DescHelpStyle.Render(" Quit"),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"\n",
		searchBox,
		statusLine,
		body,
		"\n",
		footer,
	)
}

func RunTUI(pm *providers.ProviderManager) error {
	p := tea.NewProgram(NewModel(pm), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
