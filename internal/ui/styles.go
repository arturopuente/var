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

	// Mode badges for help bar (using hex colors for consistent contrast)
	ModeBadgeCommits = lipgloss.NewStyle().
				Background(lipgloss.Color("#2d7d9a")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1)

	ModeBadgeFile = lipgloss.NewStyle().
			Background(lipgloss.Color("#7c4dff")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)

	// View mode tabs for diff header
	ViewTabActive = lipgloss.NewStyle().
			Background(lipgloss.Color("#7c4dff")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)

	ViewTabInactive = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Padding(0, 1)

	// Source mode badge for header (e.g., REFLOG, S:"term", L:func)
	SourceBadge = lipgloss.NewStyle().
			Background(lipgloss.Color("#e65100")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)
)
