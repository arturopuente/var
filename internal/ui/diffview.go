package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffView wraps a bubbles/viewport for displaying diffs
type DiffView struct {
	viewport    viewport.Model
	width       int
	height      int
	isFocused   bool
	filePath    string
	commitIndex int    // Current commit index (-1 for working copy)
	commitCount int    // Total commits for this file
	commitHash  string // Current commit hash (empty for working copy)
}

func NewDiffView(width, height int) DiffView {
	vp := viewport.New(width, height-4) // Account for header and footer
	vp.Style = lipgloss.NewStyle()

	return DiffView{
		viewport:    vp,
		width:       width,
		height:      height,
		isFocused:   false,
		commitIndex: -1,
	}
}

func (d *DiffView) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.viewport.Width = width - 2  // Account for borders
	d.viewport.Height = height - 4 // Account for header, footer, and borders
}

func (d *DiffView) SetContent(content string) {
	d.viewport.SetContent(content)
}

func (d *DiffView) SetFileInfo(path string, commitIndex, commitCount int, commitHash string) {
	d.filePath = path
	d.commitIndex = commitIndex
	d.commitCount = commitCount
	d.commitHash = commitHash
}

func (d *DiffView) SetFocused(focused bool) {
	d.isFocused = focused
}

func (d *DiffView) IsFocused() bool {
	return d.isFocused
}

// CommitIndex returns the current commit index (-1 for working copy)
func (d *DiffView) CommitIndex() int {
	return d.commitIndex
}

// CommitCount returns the total number of commits
func (d *DiffView) CommitCount() int {
	return d.commitCount
}

func (d *DiffView) Update(msg tea.Msg) (DiffView, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "d":
			// Half page down
			d.viewport.HalfViewDown()
			return *d, nil
		case "u":
			// Half page up
			d.viewport.HalfViewUp()
			return *d, nil
		}
	}

	d.viewport, cmd = d.viewport.Update(msg)
	return *d, cmd
}

func (d *DiffView) View() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("5")).
		Padding(0, 1)

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1)

	// Build header
	header := d.filePath
	if d.commitIndex >= 0 && d.commitCount > 0 {
		header = fmt.Sprintf("%s (%d/%d: %s)", d.filePath, d.commitIndex+1, d.commitCount, d.commitHash)
	} else if d.filePath != "" {
		header = fmt.Sprintf("%s (working copy)", d.filePath)
	}

	// Build footer with scroll percentage
	scrollPercent := d.viewport.ScrollPercent() * 100
	footer := fmt.Sprintf("%.0f%%", scrollPercent)

	borderStyle := lipgloss.RoundedBorder()
	borderColor := lipgloss.Color("8")
	if d.isFocused {
		borderColor = lipgloss.Color("5")
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(header),
		d.viewport.View(),
		footerStyle.Render(footer),
	)

	return lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		BorderStyle(borderStyle).
		BorderForeground(borderColor).
		Render(content)
}
