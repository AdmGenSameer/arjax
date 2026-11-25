package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Tokyo Midnight Precision Palette (aligned with Stitch TUI design)
	ColorSurfaceLowest  = lipgloss.Color("#0C0D18")
	ColorSurfaceDim     = lipgloss.Color("#12131D")
	ColorBgDark         = lipgloss.Color("#12131D")
	ColorSurfaceLow     = lipgloss.Color("#1A1B26")
	ColorCardBg         = lipgloss.Color("#1E1F2A")
	ColorSurfaceHigh    = lipgloss.Color("#282935")
	ColorSurfaceHighest = lipgloss.Color("#333440")
	ColorSubtle         = lipgloss.Color("#434751")
	ColorMuted          = lipgloss.Color("#8D909D")
	ColorText           = lipgloss.Color("#C3C6D3")
	ColorTextBright     = lipgloss.Color("#E2E1F1")

	ColorPrimary         = lipgloss.Color("#7AA2F7")
	ColorPrimarySoft     = lipgloss.Color("#AEC6FF")
	ColorOnPrimary       = lipgloss.Color("#002E6B")
	ColorOnPrimaryCont   = lipgloss.Color("#00367D")
	ColorAccent          = lipgloss.Color("#D4BBFF")
	ColorSecondaryCont   = lipgloss.Color("#56378D")
	ColorOnSecondaryCont = lipgloss.Color("#C8A9FF")
	ColorSecondary       = lipgloss.Color("#A5D570")
	ColorTertiaryCont    = lipgloss.Color("#83B151")
	ColorOnTertiaryCont  = lipgloss.Color("#244100")
	ColorHighlight       = lipgloss.Color("#7DCFFF")
	ColorDanger          = lipgloss.Color("#FFB4AB")
	ColorDangerCont      = lipgloss.Color("#93000A")
	ColorWarning         = lipgloss.Color("#E0AF68")
	ColorSelectionBg     = lipgloss.Color("#282935")

	// Provider Specific Colors
	ColorPacman = lipgloss.Color("#04B575")
	ColorAUR    = lipgloss.Color("#56378D")
	ColorFlat   = lipgloss.Color("#7AA2F7")
	ColorSnap   = lipgloss.Color("#FF9E64")
	ColorRecipe = lipgloss.Color("#BB9AF7")
	ColorGit    = lipgloss.Color("#9D7CD8")

	// Top Header & Navigation Styles
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOnPrimary).
			Background(ColorPrimarySoft).
			Padding(0, 1).
			MarginRight(1)

	BadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOnTertiaryCont).
			Background(ColorSecondary).
			Padding(0, 1).
			MarginRight(1)

	SubTitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	TopWindowBarStyle = lipgloss.NewStyle().
				Background(ColorSurfaceLow).
				Foreground(ColorText).
				Padding(0, 1)

	TopNavBarStyle = lipgloss.NewStyle().
			Background(ColorSurfaceLowest).
			Foreground(ColorText).
			Padding(0, 1)

	ActiveNavTabStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnPrimaryCont).
				Background(ColorPrimary).
				Padding(0, 1)

	InactiveNavTabStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorSurfaceLowest).
				Padding(0, 1)

	// Search & Filter Bar Styles
	SearchFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Background(ColorSurfaceLowest).
				Padding(0, 1)

	SearchBlurredStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSubtle).
				Background(ColorSurfaceLowest).
				Padding(0, 1)

	SearchInputStyle = SearchFocusedStyle

	FilterActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnPrimaryCont).
				Background(ColorPrimary).
				Padding(0, 1)

	FilterInactiveStyle = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(ColorCardBg).
				Padding(0, 1)

	MetaChipStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimarySoft).
			Background(ColorCardBg).
			Padding(0, 1)

	// Panes & Cards
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Background(ColorSurfaceLowest).
			Padding(0, 1)

	CardFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Background(ColorSurfaceLowest).
				Padding(0, 1)

	SectionCardStyle = lipgloss.NewStyle().
				Background(ColorSurfaceLow).
				Padding(0, 1)

	CanonicalCardStyle = lipgloss.NewStyle().
				Background(ColorSurfaceLow).
				Padding(0, 1)

	CanonicalCardSelectedStyle = lipgloss.NewStyle().
					Background(ColorSurfaceHigh).
					Border(lipgloss.NormalBorder(), false, false, false, true).
					BorderForeground(ColorPrimary).
					Padding(0, 1)

	// Inspector Tabs (used inside subviews)
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOnPrimaryCont).
			Background(ColorPrimary).
			Padding(0, 1).
			MarginRight(1)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Background(ColorCardBg).
				Padding(0, 1).
				MarginRight(1)

	// List Items
	ListItemStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	ListItemSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorTextBright).
				Background(ColorSurfaceHigh).
				Padding(0, 1)

	// Offer Chips & Trust Badges
	OfferChipStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorCardBg).
			Padding(0, 1)

	OfferChipActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnPrimaryCont).
				Background(ColorPrimary).
				Padding(0, 1)

	RecommendBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnPrimary).
				Background(ColorPrimarySoft).
				Padding(0, 1)

	TrustBadgeHighStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnTertiaryCont).
				Background(ColorTertiaryCont).
				Padding(0, 1)

	TrustBadgeMedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnSecondaryCont).
				Background(ColorSecondaryCont).
				Padding(0, 1)

	TrustBadgeLowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorDanger).
				Background(ColorDangerCont).
				Padding(0, 1)

	CommandPreviewBoxStyle = lipgloss.NewStyle().
				Background(ColorSurfaceHigh).
				Foreground(ColorPrimarySoft).
				Bold(true).
				Padding(0, 1)

	ActionBtnPrimaryStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorOnPrimary).
				Background(ColorPrimarySoft).
				Padding(0, 1)

	ActionBtnSecondaryStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimarySoft).
				Background(ColorSurfaceHigh).
				Padding(0, 1)

	ActionBtnDangerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorDanger).
				Background(ColorCardBg).
				Padding(0, 1)

	// Footer & Status Bar Styles
	ModeNormalStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOnPrimary).
			Background(ColorPrimarySoft).
			Padding(0, 1)

	ModeInsertStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOnTertiaryCont).
			Background(ColorSecondary).
			Padding(0, 1)

	ModeBusyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSurfaceLowest).
			Background(ColorWarning).
			Padding(0, 1)

	StatusPillStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Background(ColorSurfaceHigh).
			Padding(0, 1)

	SyncedPillStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorOnTertiaryCont).
			Background(ColorTertiaryCont).
			Padding(0, 1)

	// Typography & Helpers
	KeyHelpStyle = lipgloss.NewStyle().
			Foreground(ColorPrimarySoft).
			Background(ColorCardBg).
			Bold(true).
			Padding(0, 1)

	DescHelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	BranchTreeStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	LogTimestampStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	LogInfoStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	LogSuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	LogErrorStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)
)

// ProviderBadge renders a colored pill badge for each provider type.
func ProviderBadge(provider string) string {
	prov := strings.ToLower(strings.TrimSpace(provider))
	bg := ColorHighlight
	fg := ColorOnPrimary

	switch {
	case strings.Contains(prov, "pacman") || strings.Contains(prov, "repo") || strings.Contains(prov, "apt") || strings.Contains(prov, "dnf") || strings.Contains(prov, "zypper") || strings.Contains(prov, "deb") || strings.Contains(prov, "rpm"):
		bg = ColorPacman
		fg = ColorSurfaceLowest
	case strings.Contains(prov, "aur"):
		bg = ColorAUR
		fg = ColorOnSecondaryCont
	case strings.Contains(prov, "flatpak") || strings.Contains(prov, "flathub"):
		bg = ColorFlat
		fg = ColorOnPrimaryCont
	case strings.Contains(prov, "snap"):
		bg = ColorSnap
		fg = ColorSurfaceLowest
	case strings.Contains(prov, "recipe"):
		bg = ColorRecipe
		fg = ColorSurfaceLowest
	case strings.Contains(prov, "git") || strings.Contains(prov, "appimage"):
		bg = ColorGit
		fg = ColorTextBright
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Render(strings.ToUpper(provider))
}
