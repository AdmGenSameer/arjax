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
	err      error
}

type actionDoneMsg struct {
	result models.Result
	err    error
}

type infoDoneMsg struct {
	name string
	info models.PackageInfo
	err  error
}

type Model struct {
	pm          *providers.ProviderManager
	distro      string
	textInput   textinput.Model
	spinner     spinner.Model
	loading     bool
	statusMsg   string
	packages    []models.Package
	pkgInfo     map[string]models.PackageInfo
	cursor      int
	width       int
	height      int
	activeTab   int // 0: search, 1: details, 2: activity
	logLines    []string
	installing  bool
	err         error
}

func NewModel(pm *providers.ProviderManager) Model {
	ti := textinput.New()
	ti.Placeholder = "Type package name to search (e.g. firefox, neovim, git)..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorSecondary)

	distro := utils.DetectPackageManager()
	if distro == "" {
		distro = "linux"
	}

	return Model{
		pm:        pm,
		distro:    strings.ToUpper(distro),
		textInput: ti,
		spinner:   s,
		packages:  []models.Package{},
		pkgInfo:   make(map[string]models.PackageInfo),
		logLines:  []string{"Arjax TUI initialized. Ready for operations."},
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
		m.textInput.Width = msg.Width - 10
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if !m.textInput.Focused() || msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
		case "tab":
			if m.textInput.Focused() {
				m.textInput.Blur()
			} else {
				m.textInput.Focus()
			}
			return m, nil
		case "up", "k":
			if !m.textInput.Focused() && m.cursor > 0 {
				m.cursor--
				if m.cursor < len(m.packages) {
					cmds = append(cmds, m.runFetchInfo(m.packages[m.cursor].Name))
				}
			}
		case "down", "j":
			if !m.textInput.Focused() && m.cursor < len(m.packages)-1 {
				m.cursor++
				if m.cursor < len(m.packages) {
					cmds = append(cmds, m.runFetchInfo(m.packages[m.cursor].Name))
				}
			}
		case "enter":
			if m.textInput.Focused() {
				query := strings.TrimSpace(m.textInput.Value())
				if query != "" {
					m.loading = true
					m.statusMsg = fmt.Sprintf("Searching for '%s' across providers...", query)
					return m, tea.Batch(m.spinner.Tick, m.runSearch(query))
				}
			}
		case "i":
			if !m.textInput.Focused() && len(m.packages) > 0 && !m.installing {
				pkg := m.packages[m.cursor]
				m.installing = true
				m.statusMsg = fmt.Sprintf("Installing %s via %s...", pkg.Name, pkg.Provider)
				m.logLines = append(m.logLines, fmt.Sprintf("[%s] Triggered install: %s", time.Now().Format("15:04:05"), pkg.Name))
				return m, tea.Batch(m.spinner.Tick, m.runInstall(pkg))
			}
		case "r":
			if !m.textInput.Focused() && len(m.packages) > 0 && !m.installing {
				pkg := m.packages[m.cursor]
				m.statusMsg = fmt.Sprintf("Removing %s...", pkg.Name)
				m.logLines = append(m.logLines, fmt.Sprintf("[%s] Triggered remove: %s", time.Now().Format("15:04:05"), pkg.Name))
				return m, tea.Batch(m.spinner.Tick, m.runRemove(pkg))
			}
		case "/":
			m.textInput.Focus()
			return m, nil
		case "esc":
			m.textInput.Blur()
			return m, nil
		}

	case searchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Search failed: %v", msg.err)
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] Search error: %v", time.Now().Format("15:04:05"), msg.err))
		} else {
			m.packages = msg.packages
			m.cursor = 0
			m.statusMsg = fmt.Sprintf("Found %d packages for '%s'", len(msg.packages), msg.query)
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] Query '%s' found %d results", time.Now().Format("15:04:05"), msg.query, len(msg.packages)))
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
			m.statusMsg = fmt.Sprintf("Operation failed: %s", msg.result.Error)
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] Failed: %s (%s)", time.Now().Format("15:04:05"), msg.result.Package, msg.result.Error))
		} else {
			m.statusMsg = fmt.Sprintf("Successfully completed: %s", msg.result.Package)
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] Completed %s (%d ms)", time.Now().Format("15:04:05"), msg.result.Package, msg.result.DurationMs))
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

func (m Model) runSearch(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		pkgs, err := m.pm.SearchAll(ctx, query)
		return searchDoneMsg{query: query, packages: pkgs, err: err}
	}
}

func (m Model) runInstall(pkg models.Package) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		res, err := m.pm.Install(ctx, pkg)
		return actionDoneMsg{result: res, err: err}
	}
}

func (m Model) runRemove(pkg models.Package) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		res, err := m.pm.Remove(ctx, pkg)
		return actionDoneMsg{result: res, err: err}
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
	// Header bar
	header := lipgloss.JoinHorizontal(
		lipgloss.Center,
		AppTitleStyle.Render("⚡ ARJAX TUI"),
		BadgeStyle.Render(m.distro),
		lipgloss.NewStyle().Foreground(ColorMuted).Render("Cross-Distribution Package Engine"),
	)

	// Search bar
	searchBox := SearchInputStyle.Render(m.textInput.View())

	// Status Line
	statusText := m.statusMsg
	if m.loading || m.installing {
		statusText = fmt.Sprintf("%s %s", m.spinner.View(), m.statusMsg)
	}
	statusLine := lipgloss.NewStyle().Foreground(ColorHighlight).Render(statusText)

	// Dual pane: Left (Package list), Right (Details / Logs)
	listWidth := m.width/2 - 2
	if listWidth < 30 {
		listWidth = 30
	}
	detailsWidth := m.width - listWidth - 6
	if detailsWidth < 30 {
		detailsWidth = 30
	}

	// Package List Pane
	var listContent strings.Builder
	if len(m.packages) == 0 {
		listContent.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("\n  No packages found. Enter query above."))
	} else {
		for i, pkg := range m.packages {
			if i > 15 { // paginate visible items
				break
			}
			provBadge := lipgloss.NewStyle().Foreground(ColorAccent).Render(fmt.Sprintf("[%s]", pkg.Provider))
			line := fmt.Sprintf("%-24s %s %s", pkg.Name, provBadge, pkg.Version)
			if i == m.cursor {
				listContent.WriteString(ListItemSelectedStyle.Render("▶ " + line) + "\n")
			} else {
				listContent.WriteString(ListItemStyle.Render("  " + line) + "\n")
			}
		}
	}

	leftPane := CardFocusedStyle.Width(listWidth).Height(18).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("PACKAGES"),
			listContent.String(),
		),
	)

	// Details / Inspector Pane
	var detailsContent strings.Builder
	if len(m.packages) > 0 && m.cursor < len(m.packages) {
		selected := m.packages[m.cursor]
		detailsContent.WriteString(fmt.Sprintf("Name:        %s\n", selected.Name))
		detailsContent.WriteString(fmt.Sprintf("Provider:    %s\n", selected.Provider))
		detailsContent.WriteString(fmt.Sprintf("Version:     %s\n", selected.Version))
		if selected.Repo != "" {
			detailsContent.WriteString(fmt.Sprintf("Repository:  %s\n", selected.Repo))
		}
		if selected.Description != "" {
			detailsContent.WriteString(fmt.Sprintf("Description: %s\n", selected.Description))
		}
		if info, ok := m.pkgInfo[selected.Name]; ok {
			detailsContent.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(ColorAccent).Render("DEPENDENCY TREE:\n"))
			detailsContent.WriteString(fmt.Sprintf("📦 %s\n", selected.Name))
			if len(info.Dependencies) > 0 {
				for i, dep := range info.Dependencies {
					branch := "├──"
					if i == len(info.Dependencies)-1 {
						branch = "└──"
					}
					detailsContent.WriteString(fmt.Sprintf(" %s 🔗 %s\n", branch, dep))
				}
			} else {
				detailsContent.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(" └── (No explicit dependencies listed)\n"))
			}
		}
	} else {
		detailsContent.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render("Select a package from list to view details"))
	}

	// Recent activity logs in bottom of right pane
	var logSnippet strings.Builder
	logSnippet.WriteString("\n" + lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("ACTIVITY LOG:\n"))
	startLog := 0
	if len(m.logLines) > 8 {
		startLog = len(m.logLines) - 8
	}
	for _, l := range m.logLines[startLog:] {
		logSnippet.WriteString(lipgloss.NewStyle().Foreground(ColorMuted).Render(l) + "\n")
	}

	rightPane := CardStyle.Width(detailsWidth).Height(18).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("PACKAGE DETAILS"),
			detailsContent.String(),
			logSnippet.String(),
		),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Footer / Keybindings
	footer := lipgloss.JoinHorizontal(
		lipgloss.Left,
		KeyHelpStyle.Render(" [Tab]"), DescHelpStyle.Render(" Toggle Search/List "),
		KeyHelpStyle.Render(" [↑/↓/j/k]"), DescHelpStyle.Render(" Navigate "),
		KeyHelpStyle.Render(" [Enter]"), DescHelpStyle.Render(" Search "),
		KeyHelpStyle.Render(" [i]"), DescHelpStyle.Render(" Install "),
		KeyHelpStyle.Render(" [r]"), DescHelpStyle.Render(" Remove "),
		KeyHelpStyle.Render(" [q/Esc]"), DescHelpStyle.Render(" Quit"),
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
