package ui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DiffView wraps a bubbles/viewport for displaying diffs
type DiffView struct {
	viewport        viewport.Model
	width           int
	height          int
	isFocused       bool
	filePath        string
	commitIndex     int    // Current commit index (-1 for working copy)
	commitCount     int    // Total commits for this file
	commitHash      string // Current commit hash (empty for working copy)
	inFileMode      bool   // Whether in single-file mode
	viewMode        int    // Current view mode (0=diff, 1=context, 2=full)
	rawContent      string // Raw diff content before line numbers
	showDescription bool   // Whether to show commit description (default false)
}

func NewDiffView(width, height int) DiffView {
	vp := viewport.New(width, height-2) // Account for borders only
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
	d.viewport.Height = height - 2 // Account for borders only
}

func (d *DiffView) SetContent(content string) {
	d.rawContent = content
	d.updateContent()
}

// stripDiffHeader removes the commit description and diff metadata from
// git show output, keeping only from the first hunk header (@@ ...) onwards.
func stripDiffHeader(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		stripped := stripANSI(line)
		if strings.HasPrefix(stripped, "@@") {
			return strings.Join(lines[i:], "\n")
		}
	}
	return content
}

func (d *DiffView) updateContent() {
	content := d.rawContent
	if !d.showDescription {
		content = stripDiffHeader(content)
	}
	d.viewport.SetContent(addLineNumbers(content))
}

func (d *DiffView) ToggleDescription() {
	d.showDescription = !d.showDescription
	d.updateContent()
}

// hunkHeaderRegex matches diff hunk headers like "@@ -10,5 +12,7 @@"
var hunkHeaderRegex = regexp.MustCompile(`^@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@`)

// stripANSI removes ANSI escape codes to determine line type
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// addLineNumbers prepends line numbers to diff content
func addLineNumbers(content string) string {
	if content == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	var oldLine, newLine int
	inHunk := false

	for _, line := range lines {
		stripped := stripANSI(line)

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(stripped); matches != nil {
			fmt.Sscanf(matches[1], "%d", &oldLine)
			fmt.Sscanf(matches[2], "%d", &newLine)
			inHunk = true
			// Hunk header - no line numbers
			result = append(result, fmt.Sprintf("%4s %4s │ %s", "", "", line))
			continue
		}

		if !inHunk {
			// Header lines (diff --git, index, ---, +++) - no line numbers
			result = append(result, fmt.Sprintf("%4s %4s │ %s", "", "", line))
			continue
		}

		// Determine line type from first character
		if len(stripped) == 0 {
			// Empty line in diff context
			result = append(result, fmt.Sprintf("%4d %4d │ %s", oldLine, newLine, line))
			oldLine++
			newLine++
		} else if stripped[0] == '-' {
			// Deletion - red line number
			result = append(result, fmt.Sprintf("\x1b[31m%4d\x1b[0m %4s │ %s", oldLine, "", line))
			oldLine++
		} else if stripped[0] == '+' {
			// Addition - green line number
			result = append(result, fmt.Sprintf("%4s \x1b[32m%4d\x1b[0m │ %s", "", newLine, line))
			newLine++
		} else {
			// Context line - both line numbers
			result = append(result, fmt.Sprintf("%4d %4d │ %s", oldLine, newLine, line))
			oldLine++
			newLine++
		}
	}

	return strings.Join(result, "\n")
}

func (d *DiffView) SetFileInfo(path string, commitIndex, commitCount int, commitHash string) {
	d.filePath = path
	d.commitIndex = commitIndex
	d.commitCount = commitCount
	d.commitHash = commitHash
}

func (d *DiffView) SetMode(inFileMode bool, viewMode int) {
	d.inFileMode = inFileMode
	d.viewMode = viewMode
}

func (d *DiffView) renderViewTabs() string {
	tabs := []string{"1:diff", "2:ctx", "3:full"}
	var parts []string
	for i, tab := range tabs {
		if i == d.viewMode {
			parts = append(parts, ViewTabActive.Render(tab))
		} else {
			parts = append(parts, ViewTabInactive.Render(tab))
		}
	}
	return strings.Join(parts, " ")
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
	// Build header - just the content, no colored styling
	header := d.filePath
	if d.commitIndex >= 0 && d.commitCount > 0 {
		header = fmt.Sprintf("%s (%d/%d: %s)", d.filePath, d.commitIndex+1, d.commitCount, d.commitHash)
	} else if d.filePath != "" {
		header = fmt.Sprintf("%s (working copy)", d.filePath)
	}

	// Add view mode tabs when in file mode
	if d.inFileMode {
		tabs := d.renderViewTabs()
		header = header + "   " + tabs
	}

	// Build footer with scroll percentage
	scrollPercent := d.viewport.ScrollPercent() * 100
	footer := fmt.Sprintf("%.0f%%", scrollPercent)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Padding(0, 1).Render(header),
		d.viewport.View(),
		lipgloss.NewStyle().Faint(true).Padding(0, 1).Render(footer),
	)

	style := lipgloss.NewStyle().
		Width(d.width).
		Height(d.height).
		BorderStyle(lipgloss.RoundedBorder())

	if d.isFocused {
		// lazygit: green for active border
		style = style.BorderForeground(lipgloss.Color("2"))
	}
	// inactive: no BorderForeground = terminal default

	return style.Render(content)
}
