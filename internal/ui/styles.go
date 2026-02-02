package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("5")
	ColorSecondary = lipgloss.Color("8")
	ColorSuccess   = lipgloss.Color("2")
	ColorWarning   = lipgloss.Color("3")
	ColorError     = lipgloss.Color("1")
	ColorInfo      = lipgloss.Color("6")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4")). // blue like lazygit optionsTextColor
			Padding(0, 1)

	// Mode badges for help bar
	ModeBadgeCommits = lipgloss.NewStyle().
				Background(lipgloss.Color("6")).
				Foreground(lipgloss.Color("0")).
				Bold(true).
				Padding(0, 1)

	ModeBadgeFile = lipgloss.NewStyle().
			Background(lipgloss.Color("5")).
			Foreground(lipgloss.Color("15")).
			Bold(true).
			Padding(0, 1)

	// View mode tabs for diff header
	ViewTabActive = lipgloss.NewStyle().
			Background(lipgloss.Color("4")).
			Foreground(lipgloss.Color("15")).
			Bold(true).
			Padding(0, 1)

	ViewTabInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Padding(0, 1)

	// Dimmed border for inactive sidebar
	BorderDimmed = lipgloss.Color("8")
)
