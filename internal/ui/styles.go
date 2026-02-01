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
)
