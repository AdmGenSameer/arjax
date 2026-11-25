package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/catalog/updates"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/models"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/providers"
	"github.com/AdmGenSameer/archpkg-helper/arjax-provider-engine/utils"
)

type searchDoneMsg struct {
	query       string
	packages    []models.Package
	unifiedApps []catalog.UnifiedApplication
	duration    time.Duration
	err         error
}

type searchDebounceMsg struct {
	seq   int
	query string
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

type appDetailsDoneMsg struct {
	appID string
	app   *catalog.UnifiedApplication
	err   error
}

type installedLoadedMsg struct {
	entries      []InstalledEntry
	installedMap map[string]string
}

type updatesLoadedMsg struct {
	summary updates.UpdateSummary
}

type installTickMsg time.Time
type telemetryTickMsg time.Time

func tickInstall() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg {
		return installTickMsg(t)
	})
}

func tickTelemetry() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return telemetryTickMsg(t)
	})
}

var (
	providerFilterOrder = []string{"all", "native", "aur", "flatpak", "snap", "github"}
	scopeFilterOrder    = []string{"all", "sandboxed", "system"}
	sortModeOrder       = []string{"relevance", "trust", "name"}
	policyOrder         = []catalog.RecommendationPolicy{
		catalog.PolicyRecommended,
		catalog.PolicyOfficialOnly,
		catalog.PolicyNativePreferred,
		catalog.PolicySandboxPreferred,
		catalog.PolicyLatestVersion,
		catalog.PolicyMinimalDeps,
	}
)

type Model struct {
	pm               *providers.ProviderManager
	catalogMgr       *catalog.CatalogManager
	distro           string
	distroName       string
	nativePM         string
	telemetry        HostTelemetry
	textInput        textinput.Model
	spinner          spinner.Model
	loading          bool
	installing       bool
	installStartTime time.Time
	installProgress  float64
	installStep      int // 1..4
	installAction    string
	installPkg       models.Package
	statusMsg        string
	statusType       string // "info", "success", "error", "warning"
	packages         []models.Package
	unifiedApps      []catalog.UnifiedApplication
	selectedOffer    map[string]int
	providerFilter   string
	scopeFilter      string
	sortMode         string
	pkgInfo          map[string]models.PackageInfo
	installedEntries []InstalledEntry
	installedMap     map[string]string
	installedCursor  int
	installedLoaded  bool
	updateSummary    *updates.UpdateSummary
	updatesCursor    int
	updatesLoaded    bool
	checkingUpdates  bool
	searchSeq        int
	cursor           int
	scrollOffset     int
	width            int
	height           int
	activeTab        int // 0: Search & Store, 1: Installed & Health, 2: Updates, 3: Pipeline, 4: Settings
	logLines         []string
	showHelp         bool
	showRecipeModal  bool
	err              error
}

func NewModel(pm *providers.ProviderManager) Model {
	cm := catalog.NewCatalogManager(pm, nil, nil, catalog.DefaultCatalogConfig(), nil)
	return NewModelWithCatalog(pm, cm)
}

func NewModelWithCatalog(pm *providers.ProviderManager, cm *catalog.CatalogManager) Model {
	if cm == nil {
		cm = catalog.NewCatalogManager(pm, nil, nil, catalog.DefaultCatalogConfig(), nil)
	}
	ti := textinput.New()
	ti.Placeholder = "Type to search (e.g. zapfast, brave, neovim) — Press Enter or pause 500ms..."
	ti.Prompt = "❯ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(ColorPrimarySoft).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(ColorTextBright).Bold(true)
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 42

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
		pm:              pm,
		catalogMgr:      cm,
		distro:          distroLabel,
		distroName:      distroInfo.Name,
		nativePM:        distroPM,
		telemetry:       collectHostTelemetry(distroPM),
		textInput:       ti,
		spinner:         s,
		packages:        []models.Package{},
		unifiedApps:     []catalog.UnifiedApplication{},
		selectedOffer:   make(map[string]int),
		providerFilter:  "all",
		scopeFilter:     "all",
		sortMode:        "relevance",
		pkgInfo:         make(map[string]models.PackageInfo),
		installedMap:    make(map[string]string),
		activeTab:       0,
		statusMsg:       "Ready. Type a query or use [Tab] to browse canonical packages.",
		statusType:      "info",
		installProgress: 0,
		installStep:     1,
		logLines: []string{
			fmt.Sprintf("[%s] Arjax Shared Core Engine (Identity · Catalog · Transactions) initialized.", time.Now().Format("15:04:05")),
			fmt.Sprintf("[%s] Detected distribution environment: %s", time.Now().Format("15:04:05"), distroLabel),
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		tickTelemetry(),
		m.runDiscoverInstalled(),
		m.runCheckUpdates(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		inputW := msg.Width - 78
		if inputW < 24 {
			inputW = msg.Width - 20
		}
		if inputW < 18 {
			inputW = 18
		}
		if inputW > 54 {
			inputW = 54
		}
		m.textInput.Width = inputW
		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		if keyStr == "ctrl+c" {
			return m, tea.Quit
		}

		if m.showHelp {
			if keyStr == "?" || keyStr == "esc" || keyStr == "q" || keyStr == "enter" {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}

		if m.showRecipeModal {
			if keyStr == "v" || keyStr == "esc" {
				m.showRecipeModal = false
				return m, nil
			}
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
				m.activeTab = 0
				m.textInput.Focus()
				return m, textinput.Blink

			case "ctrl+u":
				m.textInput.SetValue("")
				m.searchSeq++
				m.statusMsg = "Cleared search query."
				m.statusType = "info"
				return m, nil

			case "1":
				m.activeTab = 0
				return m, nil
			case "2":
				m.activeTab = 1
				if !m.installedLoaded {
					return m, m.runDiscoverInstalled()
				}
				return m, nil
			case "3":
				m.activeTab = 2
				if !m.updatesLoaded && !m.checkingUpdates {
					m.checkingUpdates = true
					return m, tea.Batch(m.spinner.Tick, m.runCheckUpdates())
				}
				return m, nil
			case "4":
				m.activeTab = 3
				return m, nil
			case "5":
				m.activeTab = 4
				return m, nil
			case "[":
				m.activeTab = (m.activeTab + 4) % 5
				return m, nil
			case "]":
				m.activeTab = (m.activeTab + 1) % 5
				return m, nil

			case "p":
				if m.activeTab == 0 {
					m.providerFilter = cycleSliceValue(providerFilterOrder, m.providerFilter)
					m.clampCursor()
					m.statusMsg = fmt.Sprintf("Provider filter: %s", m.providerFilter)
					m.statusType = "info"
				}
				return m, nil

			case "s":
				if m.activeTab == 0 {
					m.scopeFilter = cycleSliceValue(scopeFilterOrder, m.scopeFilter)
					m.clampCursor()
					m.statusMsg = fmt.Sprintf("Scope filter: %s", m.scopeFilterLabel())
					m.statusType = "info"
				}
				return m, nil

			case "S":
				if m.activeTab == 0 {
					m.sortMode = cycleSliceValue(sortModeOrder, m.sortMode)
					m.clampCursor()
					m.statusMsg = fmt.Sprintf("Sort mode: %s", m.sortMode)
					m.statusType = "info"
				}
				return m, nil

			case "o":
				if m.activeTab == 0 {
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 && m.cursor < len(apps) {
						app := apps[m.cursor]
						if len(app.Offers) > 1 {
							curIdx := m.activeOfferIndex(app)
							nextIdx := (curIdx + 1) % len(app.Offers)
							m.selectedOffer[app.AppID] = nextIdx
							nextOffer := app.Offers[nextIdx]
							m.statusMsg = fmt.Sprintf("Selected target: %s (%s · %s)", nextOffer.Provider, nextOffer.Version, nextOffer.InstallScope)
							m.statusType = "info"
						}
					}
				}
				return m, nil

			case "v":
				if m.activeTab == 0 {
					m.showRecipeModal = !m.showRecipeModal
				}
				return m, nil

			case "u":
				if m.activeTab == 1 {
					m.installedLoaded = false
					return m, tea.Batch(m.spinner.Tick, m.runDiscoverInstalled())
				}
				if m.activeTab == 2 || m.activeTab == 0 {
					m.checkingUpdates = true
					m.statusMsg = "Checking updates across active providers..."
					m.statusType = "info"
					return m, tea.Batch(m.spinner.Tick, m.runCheckUpdates())
				}
				return m, nil

			case "c":
				if m.activeTab == 4 && m.catalogMgr != nil {
					next := cyclePolicy(m.catalogMgr.GetPolicy())
					m.catalogMgr.SetPolicy(next)
					m.statusMsg = fmt.Sprintf("Recommendation policy set to: %s", next)
					m.statusType = "success"
					m.logLines = append(m.logLines, fmt.Sprintf("[%s] Policy changed to: %s", time.Now().Format("15:04:05"), next))
				}
				return m, nil

			case "up", "k":
				switch m.activeTab {
				case 0:
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 && m.cursor > 0 {
						m.cursor--
						m.adjustScroll()
						cmds = append(cmds, m.fetchInfoForSelected())
					}
				case 1:
					if len(m.installedEntries) > 0 && m.installedCursor > 0 {
						m.installedCursor--
					}
				case 2:
					if m.updateSummary != nil && len(m.updateSummary.Updates) > 0 && m.updatesCursor > 0 {
						m.updatesCursor--
					}
				}
				return m, tea.Batch(cmds...)

			case "down", "j":
				switch m.activeTab {
				case 0:
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 && m.cursor < len(apps)-1 {
						m.cursor++
						m.adjustScroll()
						cmds = append(cmds, m.fetchInfoForSelected())
					}
				case 1:
					if len(m.installedEntries) > 0 && m.installedCursor < len(m.installedEntries)-1 {
						m.installedCursor++
					}
				case 2:
					if m.updateSummary != nil && len(m.updateSummary.Updates) > 0 && m.updatesCursor < len(m.updateSummary.Updates)-1 {
						m.updatesCursor++
					}
				}
				return m, tea.Batch(cmds...)

			case "g", "home":
				if m.activeTab == 0 {
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 {
						m.cursor = 0
						m.scrollOffset = 0
						cmds = append(cmds, m.fetchInfoForSelected())
					}
				} else if m.activeTab == 1 {
					m.installedCursor = 0
				} else if m.activeTab == 2 {
					m.updatesCursor = 0
				}
				return m, tea.Batch(cmds...)

			case "G", "end":
				if m.activeTab == 0 {
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 {
						m.cursor = len(apps) - 1
						m.adjustScroll()
						cmds = append(cmds, m.fetchInfoForSelected())
					}
				} else if m.activeTab == 1 && len(m.installedEntries) > 0 {
					m.installedCursor = len(m.installedEntries) - 1
				} else if m.activeTab == 2 && m.updateSummary != nil && len(m.updateSummary.Updates) > 0 {
					m.updatesCursor = len(m.updateSummary.Updates) - 1
				}
				return m, tea.Batch(cmds...)

			case "enter":
				if m.activeTab == 0 {
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 && m.cursor < len(apps) {
						app := apps[m.cursor]
						offer := m.activeOfferForApp(app)
						m.statusMsg = fmt.Sprintf("Refreshing metadata for %s (%s)...", app.Name, offer.Provider)
						m.statusType = "info"
						return m, tea.Batch(m.runFetchInfo(offer.PackageName), m.runFetchAppDetails(app.AppID, offer.ID))
					}
				}
				return m, nil

			case "i":
				if m.installing || m.loading {
					return m, nil
				}
				if m.activeTab == 0 {
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 && m.cursor < len(apps) {
						app := apps[m.cursor]
						offer := m.activeOfferForApp(app)
						pkg := packageFromOffer(offer)
						m.startTransactionState("install", pkg)
						return m, tea.Batch(m.spinner.Tick, tickInstall(), m.runInstallOffer(offer, pkg))
					}
				} else if m.activeTab == 2 && m.updateSummary != nil && len(m.updateSummary.Updates) > 0 && m.updatesCursor < len(m.updateSummary.Updates) {
					upd := m.updateSummary.Updates[m.updatesCursor]
					pkg := models.Package{
						Name:     upd.PackageName,
						Version:  upd.AvailableVersion,
						Provider: upd.Provider,
						Repo:     upd.Repository,
					}
					m.startTransactionState("install", pkg)
					m.activeTab = 0
					return m, tea.Batch(m.spinner.Tick, tickInstall(), m.runApplyUpdates([]string{upd.OfferID}))
				}

			case "U", "A":
				if m.activeTab == 2 && m.updateSummary != nil && len(m.updateSummary.Updates) > 0 {
					var allIDs []string
					for _, u := range m.updateSummary.Updates {
						allIDs = append(allIDs, u.OfferID)
					}
					pkg := models.Package{
						Name:     fmt.Sprintf("Full System Upgrade (%d updates)", len(allIDs)),
						Version:  "Latest",
						Provider: "all",
					}
					m.startTransactionState("install", pkg)
					m.activeTab = 0
					return m, tea.Batch(m.spinner.Tick, tickInstall(), m.runApplyUpdates(allIDs))
				}

			case "r":
				if m.installing || m.loading {
					return m, nil
				}
				if m.activeTab == 0 {
					apps := m.filteredUnifiedApps()
					if len(apps) > 0 && m.cursor < len(apps) {
						app := apps[m.cursor]
						offer := m.activeOfferForApp(app)
						pkg := packageFromOffer(offer)
						m.startTransactionState("remove", pkg)
						return m, tea.Batch(m.spinner.Tick, tickInstall(), m.runRemoveOffer(offer, pkg))
					}
				} else if m.activeTab == 1 && len(m.installedEntries) > 0 && m.installedCursor < len(m.installedEntries) {
					entry := m.installedEntries[m.installedCursor]
					pkg := offerFromInstalledEntry(entry)
					m.startTransactionState("remove", pkg)
					m.activeTab = 0
					return m, tea.Batch(m.spinner.Tick, tickInstall(), m.runRemove(pkg))
				}
			}
		} else {
			// Keybindings when Search Input IS focused
			switch keyStr {
			case "esc", "tab":
				m.textInput.Blur()
				return m, nil

			case "ctrl+u":
				m.textInput.SetValue("")
				m.searchSeq++
				return m, nil

			case "enter":
				query := strings.TrimSpace(m.textInput.Value())
				if query != "" {
					m.searchSeq++
					m.loading = true
					m.statusMsg = fmt.Sprintf("Searching for '%s' across all providers...", query)
					m.statusType = "info"
					m.textInput.Blur()
					return m, tea.Batch(m.spinner.Tick, m.runSearch(query))
				}
			}
		}

	case searchDebounceMsg:
		if msg.seq == m.searchSeq && m.textInput.Focused() && !m.loading {
			trimmed := strings.TrimSpace(msg.query)
			if trimmed != "" {
				m.loading = true
				m.statusMsg = fmt.Sprintf("Auto-searching '%s' across providers...", trimmed)
				m.statusType = "info"
				return m, tea.Batch(m.spinner.Tick, m.runSearch(trimmed))
			}
		}
		return m, nil

	case searchDoneMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.statusMsg = fmt.Sprintf("Search error: %v", msg.err)
			m.statusType = "error"
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] ✖ Search failed for '%s': %v", time.Now().Format("15:04:05"), msg.query, msg.err))
		} else {
			m.packages = msg.packages
			if len(msg.unifiedApps) > 0 {
				m.unifiedApps = msg.unifiedApps
			} else {
				m.unifiedApps = m.synthesizeUnifiedApps(msg.packages)
			}
			m.applyInstalledFlagsToApps()
			m.cursor = 0
			m.scrollOffset = 0
			elapsed := msg.duration.Round(time.Millisecond)
			count := len(m.packages)
			if count == 0 {
				count = len(m.unifiedApps)
			}
			m.statusMsg = fmt.Sprintf("✔ Found %d package(s) (%d canonical) for '%s' in %v", count, len(m.unifiedApps), msg.query, elapsed)
			m.statusType = "success"
			m.logLines = append(m.logLines, fmt.Sprintf("[%s] ✔ Found %d package(s) / %d canonical app(s) for '%s' (%v)", time.Now().Format("15:04:05"), count, len(m.unifiedApps), msg.query, elapsed))
			if apps := m.filteredUnifiedApps(); len(apps) > 0 {
				offer := m.activeOfferForApp(apps[0])
				cmds = append(cmds, m.runFetchInfo(offer.PackageName))
			}
		}

	case infoDoneMsg:
		if msg.err == nil {
			m.pkgInfo[msg.name] = msg.info
		}

	case appDetailsDoneMsg:
		if msg.err == nil && msg.app != nil {
			for i := range m.unifiedApps {
				if strings.EqualFold(m.unifiedApps[i].AppID, msg.appID) {
					m.unifiedApps[i] = *msg.app
					break
				}
			}
			m.applyInstalledFlagsToApps()
			m.statusMsg = fmt.Sprintf("✔ Metadata refreshed for %s", msg.app.Name)
			m.statusType = "success"
		}

	case installedLoadedMsg:
		m.installedEntries = msg.entries
		m.installedMap = msg.installedMap
		m.installedLoaded = true
		m.applyInstalledFlagsToApps()

	case updatesLoadedMsg:
		m.checkingUpdates = false
		m.updatesLoaded = true
		summary := msg.summary
		m.updateSummary = &summary

	case telemetryTickMsg:
		m.telemetry = collectHostTelemetry(m.nativePM)
		return m, tickTelemetry()

	case installTickMsg:
		if m.installing {
			prevStep := m.installStep
			if m.installProgress < 0.92 {
				delta := 0.02
				if m.installProgress < 0.3 {
					delta = 0.04
				} else if m.installProgress < 0.6 {
					delta = 0.025
				} else if m.installProgress > 0.8 {
					delta = 0.008
				}
				m.installProgress += delta
				if m.installProgress > 0.92 {
					m.installProgress = 0.92
				}
			}

			if m.installProgress < 0.25 {
				m.installStep = 1
			} else if m.installProgress < 0.50 {
				m.installStep = 2
			} else if m.installProgress < 0.85 {
				m.installStep = 3
			} else {
				m.installStep = 4
			}

			if m.installStep != prevStep {
				stepDesc := ""
				if m.installAction == "install" {
					switch m.installStep {
					case 2:
						stepDesc = "Step 2/4: Verifying dependency tree & host locks..."
					case 3:
						stepDesc = fmt.Sprintf("Step 3/4: Executing %s backend (download/compile)...", m.installPkg.Provider)
					case 4:
						stepDesc = "Step 4/4: Finalizing desktop shortcuts & symlinks..."
					}
				} else {
					switch m.installStep {
					case 2:
						stepDesc = "Step 2/4: Checking reverse dependencies..."
					case 3:
						stepDesc = fmt.Sprintf("Step 3/4: Executing %s removal routine...", m.installPkg.Provider)
					case 4:
						stepDesc = "Step 4/4: Cleaning orphaned config & cache..."
					}
				}
				if stepDesc != "" {
					m.logLines = append(m.logLines, fmt.Sprintf("[%s] %s", time.Now().Format("15:04:05"), stepDesc))
				}
			}

			return m, tickInstall()
		}

	case actionDoneMsg:
		m.installing = false
		m.installProgress = 1.0
		m.installStep = 4
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
			cmds = append(cmds, m.runDiscoverInstalled())
		}

	case spinner.TickMsg:
		if m.loading || m.installing || m.checkingUpdates {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.textInput.Focused() {
		prevVal := m.textInput.Value()
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
		newVal := m.textInput.Value()
		if newVal != prevVal {
			m.searchSeq++
			trimmed := strings.TrimSpace(newVal)
			if trimmed != "" {
				m.statusMsg = fmt.Sprintf("Typing: '%s' — Enter to search, or auto-search in 500ms", trimmed)
				m.statusType = "info"
				seq := m.searchSeq
				cmds = append(cmds, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
					return searchDebounceMsg{seq: seq, query: trimmed}
				}))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) startTransactionState(action string, pkg models.Package) {
	m.installing = true
	m.installAction = action
	m.installPkg = pkg
	m.installStartTime = time.Now()
	m.installProgress = 0.08
	m.installStep = 1
	if action == "install" {
		m.statusMsg = fmt.Sprintf("Installing %s via %s...", pkg.Name, pkg.Provider)
		m.logLines = append(m.logLines, fmt.Sprintf("[%s] ▶ Install started: %s (%s)", time.Now().Format("15:04:05"), pkg.Name, pkg.Provider))
		m.logLines = append(m.logLines, fmt.Sprintf("[%s] Step 1/4: Resolving package recipe & provider strategy...", time.Now().Format("15:04:05")))
	} else {
		m.statusMsg = fmt.Sprintf("Removing %s...", pkg.Name)
		m.logLines = append(m.logLines, fmt.Sprintf("[%s] ▶ Remove started: %s", time.Now().Format("15:04:05"), pkg.Name))
		m.logLines = append(m.logLines, fmt.Sprintf("[%s] Step 1/4: Locating package binaries & dependencies...", time.Now().Format("15:04:05")))
	}
	m.statusType = "warning"
}

func (m *Model) clampCursor() {
	apps := m.filteredUnifiedApps()
	if len(apps) == 0 {
		m.cursor = 0
		m.scrollOffset = 0
		return
	}
	if m.cursor >= len(apps) {
		m.cursor = len(apps) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustScroll()
}

func (m Model) computePaneBoxHeight(h int) int {
	// Top window (1) + Nav (1) + Query (3) + Filter (1) + PaneBorders (2) + StatusBar (1) + KeyBar (1) = 10
	boxH := h - 10
	if boxH < 10 {
		boxH = 10
	}
	return boxH
}

func (m *Model) adjustScroll() {
	h := m.height
	if h < 20 {
		h = 20
	}
	boxH := m.computePaneBoxHeight(h)
	visibleCards := (boxH - 3) / 5
	if visibleCards < 1 {
		visibleCards = 1
	}

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	} else if m.cursor >= m.scrollOffset+visibleCards {
		m.scrollOffset = m.cursor - visibleCards + 1
	}
}

func (m Model) fetchInfoForSelected() tea.Cmd {
	apps := m.filteredUnifiedApps()
	if len(apps) > 0 && m.cursor < len(apps) {
		offer := m.activeOfferForApp(apps[m.cursor])
		if _, exists := m.pkgInfo[offer.PackageName]; !exists {
			return m.runFetchInfo(offer.PackageName)
		}
	}
	return nil
}

func (m Model) runSearch(query string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if m.catalogMgr != nil {
			apps, err := m.catalogMgr.ProgressiveSearch().SearchSynchronous(ctx, query)
			if err == nil && len(apps) > 0 {
				var pkgs []models.Package
				for _, app := range apps {
					for _, offer := range app.Offers {
						pkgs = append(pkgs, packageFromOffer(offer))
					}
				}
				return searchDoneMsg{
					query:       query,
					packages:    pkgs,
					unifiedApps: apps,
					duration:    time.Since(start),
				}
			}
		}

		pkgs, err := m.pm.SearchAll(ctx, query)
		return searchDoneMsg{
			query:    query,
			packages: pkgs,
			duration: time.Since(start),
			err:      err,
		}
	}
}

func (m Model) runInstallOffer(offer catalog.PackageOffer, pkg models.Package) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		if m.catalogMgr != nil {
			if offer.ID == "" {
				offer = m.catalogMgr.Deduplicator().ConvertPackageToOffer(pkg, false, "")
			}
			m.catalogMgr.Transactions().RegisterOffer(offer)
			tx, err := m.catalogMgr.InstallOffer(ctx, offer.ID, nil)
			return actionDoneMsg{
				action: "install",
				result: models.Result{
					Command:    "install",
					Package:    pkg.Name,
					Provider:   pkg.Provider,
					Success:    tx.State == catalog.TxCompleted && err == nil,
					StartTime:  tx.StartedAt,
					DurationMs: time.Since(tx.StartedAt).Milliseconds(),
					Error:      tx.ErrorMessage,
				},
				err: err,
			}
		}

		res, err := m.pm.Install(ctx, pkg)
		return actionDoneMsg{action: "install", result: res, err: err}
	}
}

func (m Model) runRemoveOffer(offer catalog.PackageOffer, pkg models.Package) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		if m.catalogMgr != nil {
			if offer.ID == "" {
				offer = m.catalogMgr.Deduplicator().ConvertPackageToOffer(pkg, true, pkg.Version)
			}
			m.catalogMgr.Transactions().RegisterOffer(offer)
			tx, err := m.catalogMgr.RemoveOffer(ctx, offer.ID, nil)
			return actionDoneMsg{
				action: "remove",
				result: models.Result{
					Command:    "remove",
					Package:    pkg.Name,
					Provider:   pkg.Provider,
					Success:    tx.State == catalog.TxCompleted && err == nil,
					StartTime:  tx.StartedAt,
					DurationMs: time.Since(tx.StartedAt).Milliseconds(),
					Error:      tx.ErrorMessage,
				},
				err: err,
			}
		}

		res, err := m.pm.Remove(ctx, pkg)
		return actionDoneMsg{action: "remove", result: res, err: err}
	}
}

func (m Model) runInstall(pkg models.Package) tea.Cmd {
	return m.runInstallOffer(catalog.PackageOffer{}, pkg)
}

func (m Model) runApplyUpdates(offerIDs []string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
		defer cancel()

		start := time.Now()
		if m.catalogMgr != nil {
			batchRes := m.catalogMgr.Updates().UpdateSelected(ctx, offerIDs, nil)
			success := batchRes.Failed == 0
			errMsg := ""
			if !success {
				errMsg = batchRes.Note
			}
			return actionDoneMsg{
				action: "install",
				result: models.Result{
					Command:    "upgrade",
					Package:    fmt.Sprintf("system-upgrade (%d packages)", batchRes.Succeeded),
					Provider:   "system",
					Success:    success,
					StartTime:  start,
					EndTime:    time.Now(),
					DurationMs: time.Since(start).Milliseconds(),
					Error:      errMsg,
				},
				err: nil,
			}
		}

		res, err := m.pm.Update(ctx)
		return actionDoneMsg{action: "install", result: res, err: err}
	}
}

func (m Model) runRemove(pkg models.Package) tea.Cmd {
	return m.runRemoveOffer(catalog.PackageOffer{}, pkg)
}

func (m Model) runFetchInfo(pkgName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		info, err := m.pm.Info(ctx, pkgName)
		return infoDoneMsg{name: pkgName, info: info, err: err}
	}
}

func (m Model) runFetchAppDetails(appID, preferredOfferID string) tea.Cmd {
	return func() tea.Msg {
		if m.catalogMgr == nil {
			return appDetailsDoneMsg{appID: appID, err: fmt.Errorf("catalog not initialized")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		app, err := m.catalogMgr.GetUnifiedAppDetails(ctx, appID, preferredOfferID)
		return appDetailsDoneMsg{appID: appID, app: app, err: err}
	}
}

func (m Model) runDiscoverInstalled() tea.Cmd {
	return func() tea.Msg {
		entries, instMap := discoverHostInstalled(context.Background())
		return installedLoadedMsg{entries: entries, installedMap: instMap}
	}
}

func (m Model) runCheckUpdates() tea.Cmd {
	return func() tea.Msg {
		if m.catalogMgr == nil || m.catalogMgr.Updates() == nil {
			return updatesLoadedMsg{}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		summary := m.catalogMgr.Updates().CheckAllUpdates(ctx)
		return updatesLoadedMsg{summary: summary}
	}
}

func (m Model) effectiveUnifiedApps() []catalog.UnifiedApplication {
	if len(m.unifiedApps) > 0 {
		return m.unifiedApps
	}
	if len(m.packages) > 0 {
		return m.synthesizeUnifiedApps(m.packages)
	}
	return nil
}

func (m Model) synthesizeUnifiedApps(pkgs []models.Package) []catalog.UnifiedApplication {
	out := make([]catalog.UnifiedApplication, 0, len(pkgs))
	for _, p := range pkgs {
		var offer catalog.PackageOffer
		if m.catalogMgr != nil {
			offer = m.catalogMgr.Deduplicator().ConvertPackageToOffer(p, false, "")
		} else {
			offer = catalog.PackageOffer{
				ID:           catalog.BuildOfferIdentity(p.Provider, p.Repo, p.Name, p.Arch, p.Branch, p.Channel),
				AppID:        p.Name,
				PackageName:  p.Name,
				Provider:     p.Provider,
				Repository:   p.Repo,
				Version:      p.Version,
				DownloadSize: p.Size,
				InstallScope: catalog.DetermineScope(p.Provider),
				Permissions:  catalog.DeterminePermissions(p.Provider, p.Name),
				Dependencies: p.Dependencies,
				Trust:        catalog.EvaluateSourceTrust(p.Provider, p.Repo, p.Name),
				IsEligible:   true,
			}
			offer.TrustLevel = offer.Trust.TrustLevel
			offer.SignatureStatus = offer.Trust.SignatureStatus
			offer.CommandPreview = catalog.FormatCommandPreview(p.Provider, p.Repo, p.Name, p.Branch, p.Channel)
		}
		out = append(out, catalog.UnifiedApplication{
			AppID:          p.Name,
			Name:           p.Name,
			Summary:        p.Description,
			Description:    p.Description,
			Homepage:       catalog.HomepageForProvider(offer.Provider, p.Name),
			Offers:         []catalog.PackageOffer{offer},
			DefaultOfferID: offer.ID,
		})
	}
	return out
}

func (m *Model) applyInstalledFlagsToApps() {
	if len(m.installedMap) == 0 {
		return
	}
	for i := range m.unifiedApps {
		app := &m.unifiedApps[i]
		for j := range app.Offers {
			o := &app.Offers[j]
			if ver, ok := m.installedVersionForOffer(*o); ok {
				o.IsInstalled = true
				o.InstalledVersion = ver
				app.IsInstalled = true
				app.InstalledOfferID = o.ID
			}
		}
	}
}

func (m Model) installedVersionForOffer(o catalog.PackageOffer) (string, bool) {
	if len(m.installedMap) == 0 {
		return "", false
	}
	prov := strings.ToLower(strings.TrimSpace(o.Provider))
	pkg := strings.ToLower(strings.TrimSpace(o.PackageName))
	appID := strings.ToLower(strings.TrimSpace(o.AppID))
	for _, k := range []string{prov + ":" + pkg, prov + ":" + appID, pkg, appID} {
		if k != "" && k != ":" {
			if ver, ok := m.installedMap[k]; ok {
				return ver, true
			}
		}
	}
	return "", false
}

func (m Model) isOfferInstalledOnHost(o catalog.PackageOffer) bool {
	_, ok := m.installedVersionForOffer(o)
	return ok
}

func (m Model) isAppInstalledOnHost(app catalog.UnifiedApplication) bool {
	for _, o := range app.Offers {
		if o.IsInstalled || m.isOfferInstalledOnHost(o) {
			return true
		}
	}
	return false
}

func (m Model) filteredUnifiedApps() []catalog.UnifiedApplication {
	all := m.effectiveUnifiedApps()
	if len(all) == 0 {
		return nil
	}

	filtered := make([]catalog.UnifiedApplication, 0, len(all))
	for _, app := range all {
		if m.appMatchesFilter(app) {
			filtered = append(filtered, app)
		}
	}

	switch m.sortMode {
	case "name":
		sort.SliceStable(filtered, func(i, j int) bool {
			return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
		})
	case "trust":
		sort.SliceStable(filtered, func(i, j int) bool {
			si, _ := evaluateOfferTrustScore(m.activeOfferForApp(filtered[i]))
			sj, _ := evaluateOfferTrustScore(m.activeOfferForApp(filtered[j]))
			return si > sj
		})
	default: // "relevance"
		catalog.SortByRelevance(filtered, m.textInput.Value())
	}
	return filtered
}

func (m Model) appMatchesFilter(app catalog.UnifiedApplication) bool {
	for _, o := range app.Offers {
		if matchesProviderFilter(o.Provider, m.providerFilter) && matchesScopeFilter(o.InstallScope, m.scopeFilter) {
			return true
		}
	}
	return false
}

func (m Model) activeOfferIndex(app catalog.UnifiedApplication) int {
	if len(app.Offers) == 0 {
		return 0
	}
	if idx, ok := m.selectedOffer[app.AppID]; ok && idx >= 0 && idx < len(app.Offers) {
		return idx
	}
	if m.providerFilter != "all" {
		for i, o := range app.Offers {
			if matchesProviderFilter(o.Provider, m.providerFilter) {
				return i
			}
		}
	}
	if app.DefaultOfferID != "" {
		for i, o := range app.Offers {
			if o.ID == app.DefaultOfferID {
				return i
			}
		}
	}
	return 0
}

func (m Model) activeOfferForApp(app catalog.UnifiedApplication) catalog.PackageOffer {
	if len(app.Offers) == 0 {
		return catalog.PackageOffer{PackageName: app.Name, AppID: app.AppID}
	}
	return app.Offers[m.activeOfferIndex(app)]
}

func (m Model) scopeFilterLabel() string {
	switch m.scopeFilter {
	case "sandboxed":
		return "Sandboxed"
	case "system":
		return "System"
	default:
		return "System + User"
	}
}

func matchesProviderFilter(prov, filter string) bool {
	if filter == "" || filter == "all" {
		return true
	}
	p := strings.ToLower(strings.TrimSpace(prov))
	switch filter {
	case "native":
		return p == "pacman" || p == "apt" || p == "deb" || p == "dnf" || p == "rpm" || p == "zypper" || p == "repository"
	case "github":
		return p == "github" || p == "appimage" || p == "vendor"
	default:
		return p == filter
	}
}

func matchesScopeFilter(scope catalog.InstallScope, filter string) bool {
	switch filter {
	case "sandboxed":
		return scope == catalog.ScopeSandboxed
	case "system":
		return scope == catalog.ScopeSystem || scope == catalog.ScopeUser
	default:
		return true
	}
}

func packageFromOffer(o catalog.PackageOffer) models.Package {
	return models.Package{
		Name:         o.PackageName,
		Version:      o.Version,
		Size:         o.DownloadSize,
		Provider:     o.Provider,
		Source:       o.Repository,
		Repo:         o.Repository,
		Arch:         o.Architecture,
		Branch:       o.Branch,
		Channel:      o.Channel,
		Scope:        string(o.InstallScope),
		Dependencies: o.Dependencies,
	}
}

func cycleSliceValue(items []string, cur string) string {
	for i, item := range items {
		if item == cur {
			return items[(i+1)%len(items)]
		}
	}
	return items[0]
}

func cyclePolicy(cur catalog.RecommendationPolicy) catalog.RecommendationPolicy {
	for i, p := range policyOrder {
		if p == cur {
			return policyOrder[(i+1)%len(policyOrder)]
		}
	}
	return policyOrder[0]
}

func (m Model) View() string {
	w := m.width
	if w < 80 {
		w = 80
	}
	h := m.height
	if h < 20 {
		h = 20
	}

	topBars := m.renderTopBars(w)
	paneBoxHeight := m.computePaneBoxHeight(h)

	var mainContent string
	switch m.activeTab {
	case 0:
		mainContent = m.renderSearchAndStoreView(w, paneBoxHeight)
	case 1:
		mainContent = m.renderInstalledAndHealthView(w, paneBoxHeight)
	case 2:
		mainContent = m.renderUpdatesCenterView(w, paneBoxHeight)
	case 3:
		mainContent = m.renderPipelineAndLogsView(w, paneBoxHeight)
	case 4:
		mainContent = m.renderSettingsAndProvidersView(w, paneBoxHeight)
	default:
		mainContent = m.renderSearchAndStoreView(w, paneBoxHeight)
	}

	bottomBars := m.renderBottomBars(w)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topBars,
		mainContent,
		bottomBars,
	)
}

func (m Model) renderTopBars(w int) string {
	// 1. Top Window Telemetry Strip
	winLeft := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		fmt.Sprintf("● ● ●  tty • %s • %s • arjax-shell (bubbletea)", m.telemetry.Arch, m.telemetry.Hostname),
	)
	winRight := lipgloss.NewStyle().Foreground(ColorPrimarySoft).Render("utf-8  ") +
		lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(m.distro)
	wGap := w - 2 - lipgloss.Width(winLeft) - lipgloss.Width(winRight)
	if wGap < 1 {
		wGap = 1
	}
	winBar := TopWindowBarStyle.Width(w).Render(winLeft + strings.Repeat(" ", wGap) + winRight)

	// 2. Primary Navigation Bar (1..5 Tabs)
	updCount := 0
	if m.updateSummary != nil {
		updCount = m.updateSummary.TotalCount
	}
	tabNames := []string{
		"(1) Search & Store",
		"(2) Installed & Health",
		fmt.Sprintf("(3) Updates [%d]", updCount),
		"(4) Pipeline",
		"(5) Settings",
	}

	var tabViews []string
	tabViews = append(tabViews, AppTitleStyle.Render("⚡ ARJAX TUI v2.4"))
	for i, name := range tabNames {
		if i == m.activeTab {
			tabViews = append(tabViews, ActiveNavTabStyle.Render(name))
		} else {
			tabViews = append(tabViews, InactiveNavTabStyle.Render(name))
		}
	}
	navLeft := lipgloss.JoinHorizontal(lipgloss.Center, tabViews...)

	navRight := lipgloss.NewStyle().Foreground(ColorMuted).Render(
		fmt.Sprintf("HOST: %s • KERNEL: %s", truncateText(m.telemetry.Hostname, 14), truncateText(m.telemetry.Kernel, 16)),
	)
	nGap := w - 2 - lipgloss.Width(navLeft) - lipgloss.Width(navRight)
	var navRow string
	if nGap >= 1 {
		navRow = navLeft + strings.Repeat(" ", nGap) + navRight
	} else {
		navRow = clampStyledWidth(navLeft, w-2)
	}
	navBar := TopNavBarStyle.Width(w).Render(navRow)

	return lipgloss.JoinVertical(lipgloss.Left, winBar, navBar)
}

func (m Model) renderBottomBars(w int) string {
	// 1. Live System & Lock Telemetry Bar
	modePill := ModeNormalStyle.Render("NORMAL")
	if m.installing || m.loading || m.checkingUpdates {
		modePill = ModeBusyStyle.Render("BUSY")
	} else if m.textInput.Focused() {
		modePill = ModeInsertStyle.Render("SEARCH")
	}

	pollerPill := StatusPillStyle.Render("poller: active")
	lockLabel := fmt.Sprintf("%s: FREE", m.telemetry.LockPath)
	lockStyle := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	if m.telemetry.Locked {
		lockLabel = fmt.Sprintf("%s: HELD", m.telemetry.LockPath)
		lockStyle = lipgloss.NewStyle().Foreground(ColorDanger).Bold(true)
	}
	lockPill := StatusPillStyle.Render(lockStyle.Render(lockLabel))

	statusText := m.statusMsg
	if m.loading || m.installing {
		statusText = m.spinner.View() + " " + statusText
	}
	statusColor := ColorPrimarySoft
	switch m.statusType {
	case "success":
		statusColor = ColorSecondary
	case "error":
		statusColor = ColorDanger
	case "warning":
		statusColor = ColorWarning
	}

	leftGroup := lipgloss.JoinHorizontal(lipgloss.Center, modePill, " ", pollerPill, " ", lockPill, "  ",
		lipgloss.NewStyle().Foreground(statusColor).Render(truncateText(statusText, w/3)))

	memText := "RAM: --"
	if m.telemetry.MemTotalGB > 0 {
		memText = fmt.Sprintf("RAM: %.1fG/%.0fG", m.telemetry.MemUsedGB, m.telemetry.MemTotalGB)
	}
	rightGroup := lipgloss.JoinHorizontal(
		lipgloss.Center,
		lipgloss.NewStyle().Foreground(ColorText).Render(memText),
		" ",
		SyncedPillStyle.Render("SYNCED"),
	)

	sGap := w - 2 - lipgloss.Width(leftGroup) - lipgloss.Width(rightGroup)
	if sGap < 1 {
		sGap = 1
	}
	statusBar := TopWindowBarStyle.Width(w).Render(leftGroup + strings.Repeat(" ", sGap) + rightGroup)

	// 2. Keyboard Shortcut Footer Bar
	keys := lipgloss.JoinHorizontal(
		lipgloss.Left,
		KeyHelpStyle.Render("Tab"), DescHelpStyle.Render(" Pane "),
		KeyHelpStyle.Render("/"), DescHelpStyle.Render(" Search "),
		KeyHelpStyle.Render("p"), DescHelpStyle.Render(" Provider "),
		KeyHelpStyle.Render("o"), DescHelpStyle.Render(" Offer "),
		KeyHelpStyle.Render("i"), DescHelpStyle.Render(" Install "),
		KeyHelpStyle.Render("r"), DescHelpStyle.Render(" Remove "),
		KeyHelpStyle.Render("v"), DescHelpStyle.Render(" Recipe "),
		KeyHelpStyle.Render("1-5"), DescHelpStyle.Render(" Tabs "),
		KeyHelpStyle.Render("?"), DescHelpStyle.Render(" Help "),
		KeyHelpStyle.Render("q"), DescHelpStyle.Render(" Quit"),
	)
	termRight := lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("term: %s • ", m.telemetry.TermName)) +
		lipgloss.NewStyle().Foreground(ColorSecondary).Render("ready")
	kGap := w - 2 - lipgloss.Width(keys) - lipgloss.Width(termRight)
	var keyRow string
	if kGap >= 1 {
		keyRow = keys + strings.Repeat(" ", kGap) + termRight
	} else {
		keyRow = clampStyledWidth(keys, w-2)
	}
	keyBar := TopNavBarStyle.Width(w).Render(keyRow)

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, keyBar)
}

func RunTUI(pm *providers.ProviderManager) error {
	return RunTUIWithCatalog(pm, nil)
}

func RunTUIWithCatalog(pm *providers.ProviderManager, cm *catalog.CatalogManager) error {
	p := tea.NewProgram(NewModelWithCatalog(pm, cm), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
