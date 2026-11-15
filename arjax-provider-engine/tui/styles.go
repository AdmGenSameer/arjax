package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// Tokyo Night & Catppuccin Modern Terminal Palette
	ColorPrimary    = lipgloss.Color("#7AA2F7") // Tokyo Night Cyan/Blue
	ColorSecondary  = lipgloss.Color("#9ECE6A") // Emerald Green
	ColorAccent     = lipgloss.Color("#BB9AF7") // Vibrant Purple / Violet
	ColorHighlight  = lipgloss.Color("#7DCFFF") // Electric Cyan
	ColorMuted      = lipgloss.Color("#565F89") // Cool Muted Slate
	ColorSubtle     = lipgloss.Color("#3B4261") // Subtle Border
	ColorBgDark     = lipgloss.Color("#1A1B26") // Deep Background
	ColorCardBg     = lipgloss.Color("#24283B") // Card Background
	ColorDanger     = lipgloss.Color("#F7768E") // Rose Red
	ColorWarning    = lipgloss.Color("#E0AF68") // Warm Amber
	ColorText       = lipgloss.Color("#C0CAF5") // Crisp Foreground
	ColorTextBright = lipgloss.Color("#FFFFFF") // Crisp White

	// Provider Specific Colors
	ColorPacman = lipgloss.Color("#04B575") // Pacman Emerald
	ColorAUR    = lipgloss.Color("#E0AF68") // AUR Gold
	ColorFlat   = lipgloss.Color("#7DCFFF") // Flatpak Cyan
	ColorSnap   = lipgloss.Color("#FF9E64") // Snap Orange
	ColorRecipe = lipgloss.Color("#BB9AF7") // Recipe Purple
	ColorGit    = lipgloss.Color("#9D7CD8") // GitHub Violet

	// Header Styles
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextBright).
			Background(ColorPrimary).
			Padding(0, 1).
			MarginRight(1)

	BadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextBright).
			Background(ColorSecondary).
			Padding(0, 1).
			MarginRight(1)

	SubTitleStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	// Search Input
	SearchFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorHighlight).
				Padding(0, 1)

	SearchBlurredStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSubtle).
				Padding(0, 1)

	// Backward Compatibility
	SearchInputStyle = SearchFocusedStyle

	// Panes & Cards
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Padding(0, 1)

	CardFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Tabs
	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorTextBright).
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
				Background(lipgloss.Color("#3D59A1")).
				Padding(0, 1)

	// Typography & Helpers
	KeyHelpStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

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

// ProviderBadge renders a colored pill badge for each provider type
func ProviderBadge(provider string) string {
	prov := strings.ToLower(strings.TrimSpace(provider))
	var color lipgloss.Color

	switch {
	case strings.Contains(prov, "pacman") || strings.Contains(prov, "repo") || strings.Contains(prov, "apt") || strings.Contains(prov, "dnf") || strings.Contains(prov, "zypper"):
		color = ColorPacman
	case strings.Contains(prov, "aur"):
		color = ColorAUR
	case strings.Contains(prov, "flatpak"):
		color = ColorFlat
	case strings.Contains(prov, "snap"):
		color = ColorSnap
	case strings.Contains(prov, "recipe"):
		color = ColorRecipe
	case strings.Contains(prov, "git"):
		color = ColorGit
	default:
		color = ColorHighlight
	}

	return lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorTextBright).
		Background(color).
		Padding(0, 1).
		Render(strings.ToUpper(provider))
}
