package tui

import "github.com/charmbracelet/lipgloss"

// Стили оформления Lip Gloss (WOW-эффект)
var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF")).
			Background(lipgloss.Color("#1A1A2E")).
			Padding(1, 3).
			MarginBottom(1)

	styleTab = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#3A3F58")).
			Padding(0, 2)

	styleActiveTab = styleTab.Copy().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("#00F0FF")).
			Bold(true).
			Foreground(lipgloss.Color("#00F0FF"))

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3A3F58")).
			Padding(1, 2)

	styleCardActive = styleCard.Copy().
			BorderForeground(lipgloss.Color("#00F0FF"))

	styleGreen = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87"))
	styleRed   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF416C"))
	styleCyan  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00F0FF"))
	styleGray  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7E849E"))
)
