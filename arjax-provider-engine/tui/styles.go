package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#04B575")
	ColorAccent    = lipgloss.Color("#EE6FF8")
	ColorMuted     = lipgloss.Color("#626262")
	ColorHighlight = lipgloss.Color("#00D7D7")
	ColorBgDark    = lipgloss.Color("#1A1B26")
	ColorCardBg    = lipgloss.Color("#24283B")
	ColorDanger    = lipgloss.Color("#F7768E")
	ColorWarning   = lipgloss.Color("#E0AF68")

	// Styles
	AppTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1).
			MarginRight(1)

	BadgeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorSecondary).
			Padding(0, 1).
			MarginRight(1)

	SearchInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(1, 2)

	CardFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorHighlight).
				Padding(1, 2)

	ListItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	ListItemSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(ColorPrimary).
				Padding(0, 1)

	KeyHelpStyle = lipgloss.NewStyle().
			Foreground(ColorHighlight).
			Bold(true)

	DescHelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
