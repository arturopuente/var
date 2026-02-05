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
	hunkPositions   []int  // Line positions of @@ hunk headers in rendered content
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
	rendered, hunkPos := addLineNumbers(content)
	d.hunkPositions = hunkPos
	d.viewport.SetContent(rendered)
}

func (d *DiffView) ToggleDescription() {
	d.showDescription = !d.showDescription
	d.updateContent()
}

// hunkHeaderRegex matches diff hunk headers like "@@ -10,5 +12,7 @@"
var hunkHeaderRegex = regexp.MustCompile(`^@@\s+-(\d+)(?:,\d+)?\s+\+(\d+)(?:,\d+)?\s+@@`)

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape codes to determine line type
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// diffBlock holds buffered minus/plus lines with their line numbers
type diffBlock struct {
	minusTexts []string // stripped text (no ANSI) for each minus line
	plusTexts  []string // stripped text (no ANSI) for each plus line
	minusNums  []int    // old line numbers
	plusNums   []int    // new line numbers
}

// highlightDiff applies reverse video to the changed portion between two lines.
// baseColor is the ANSI color code for the line type (31=red, 32=green).
func highlightDiff(thisText, otherText string, baseColor string) string {
	thisRunes := []rune(thisText)
	otherRunes := []rune(otherText)

	// Find longest common prefix
	prefixLen := 0
	minLen := len(thisRunes)
	if len(otherRunes) < minLen {
		minLen = len(otherRunes)
	}
	for prefixLen < minLen && thisRunes[prefixLen] == otherRunes[prefixLen] {
		prefixLen++
	}

	// Find longest common suffix (not overlapping prefix)
	suffixLen := 0
	for suffixLen < minLen-prefixLen &&
		thisRunes[len(thisRunes)-1-suffixLen] == otherRunes[len(otherRunes)-1-suffixLen] {
		suffixLen++
	}

	// If everything matches or nothing matches meaningfully, just return with base color
	changeStart := prefixLen
	changeEnd := len(thisRunes) - suffixLen
	if changeStart >= changeEnd {
		// No change region in this line
		return fmt.Sprintf("\x1b[%sm%s\x1b[0m", baseColor, string(thisRunes))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\x1b[%sm", baseColor))
	if changeStart > 0 {
		b.WriteString(string(thisRunes[:changeStart]))
	}
	// Reverse video for changed portion
	b.WriteString("\x1b[7m")
	b.WriteString(string(thisRunes[changeStart:changeEnd]))
	b.WriteString("\x1b[27m")
	if suffixLen > 0 {
		b.WriteString(string(thisRunes[changeEnd:]))
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

// flushBlock outputs buffered minus/plus lines with word-level highlighting
func flushBlock(block *diffBlock, result *[]string) {
	minCount := len(block.minusTexts)
	plusCount := len(block.plusTexts)

	// Pair lines: min(minus, plus) get highlighting
	pairCount := minCount
	if plusCount < pairCount {
		pairCount = plusCount
	}

	// Output all minus lines first
	for i := 0; i < minCount; i++ {
		text := block.minusTexts[i]
		var rendered string
		if i < pairCount {
			// Paired: apply word-level highlighting
			// Skip the leading '-' for comparison, then prepend it back
			thisContent := text[1:] // skip '-'
			otherContent := block.plusTexts[i][1:] // skip '+'
			highlighted := highlightDiff(thisContent, otherContent, "31")
			rendered = fmt.Sprintf("\x1b[31m%4d\x1b[0m %4s │ \x1b[31m-\x1b[0m%s", block.minusNums[i], "", highlighted)
		} else {
			// Unpaired: normal red
			rendered = fmt.Sprintf("\x1b[31m%4d\x1b[0m %4s │ \x1b[31m%s\x1b[0m", block.minusNums[i], "", text)
		}
		*result = append(*result, rendered)
	}

	// Output all plus lines
	for i := 0; i < plusCount; i++ {
		text := block.plusTexts[i]
		var rendered string
		if i < pairCount {
			// Paired: apply word-level highlighting
			thisContent := text[1:] // skip '+'
			otherContent := block.minusTexts[i][1:] // skip '-'
			highlighted := highlightDiff(thisContent, otherContent, "32")
			rendered = fmt.Sprintf("%4s \x1b[32m%4d\x1b[0m │ \x1b[32m+\x1b[0m%s", "", block.plusNums[i], highlighted)
		} else {
			// Unpaired: normal green
			rendered = fmt.Sprintf("%4s \x1b[32m%4d\x1b[0m │ \x1b[32m%s\x1b[0m", "", block.plusNums[i], text)
		}
		*result = append(*result, rendered)
	}

	// Reset block
	block.minusTexts = block.minusTexts[:0]
	block.plusTexts = block.plusTexts[:0]
	block.minusNums = block.minusNums[:0]
	block.plusNums = block.plusNums[:0]
}

// addLineNumbers prepends line numbers to diff content and returns hunk header positions.
// It buffers consecutive -/+ lines to apply word-level inline diff highlighting.
func addLineNumbers(content string) (string, []int) {
	if content == "" {
		return content, nil
	}

	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))
	var hunkPositions []int

	var oldLine, newLine int
	inHunk := false

	// State machine: collecting minus lines, then plus lines
	// "idle" -> saw '-' -> collecting minuses
	// collecting minuses -> saw '+' -> collecting plusses
	// collecting plusses -> saw anything else -> flush block
	var block diffBlock
	collectingMinus := false
	collectingPlus := false

	for _, line := range lines {
		stripped := stripANSI(line)

		// Check for hunk header
		if matches := hunkHeaderRegex.FindStringSubmatch(stripped); matches != nil {
			// Flush any pending block
			if collectingMinus || collectingPlus {
				flushBlock(&block, &result)
				collectingMinus = false
				collectingPlus = false
			}
			fmt.Sscanf(matches[1], "%d", &oldLine)
			fmt.Sscanf(matches[2], "%d", &newLine)
			inHunk = true
			hunkPositions = append(hunkPositions, len(result))
			result = append(result, fmt.Sprintf("%4s %4s │ %s", "", "", line))
			continue
		}

		if !inHunk {
			result = append(result, fmt.Sprintf("%4s %4s │ %s", "", "", line))
			continue
		}

		if len(stripped) == 0 {
			// Empty line in diff context — flush any block
			if collectingMinus || collectingPlus {
				flushBlock(&block, &result)
				collectingMinus = false
				collectingPlus = false
			}
			result = append(result, fmt.Sprintf("%4d %4d │ %s", oldLine, newLine, line))
			oldLine++
			newLine++
		} else if stripped[0] == '-' {
			if collectingPlus {
				// New minus after plus means end of block, flush
				flushBlock(&block, &result)
				collectingMinus = false
				collectingPlus = false
			}
			// Buffer this minus line
			block.minusTexts = append(block.minusTexts, stripped)
			block.minusNums = append(block.minusNums, oldLine)
			collectingMinus = true
			oldLine++
		} else if stripped[0] == '+' {
			if collectingMinus {
				// Transition from minus to plus
				collectingMinus = false
				collectingPlus = true
			} else if !collectingPlus {
				// Plus without preceding minus — standalone
				collectingPlus = true
			}
			block.plusTexts = append(block.plusTexts, stripped)
			block.plusNums = append(block.plusNums, newLine)
			newLine++
		} else {
			// Context line — flush any pending block
			if collectingMinus || collectingPlus {
				flushBlock(&block, &result)
				collectingMinus = false
				collectingPlus = false
			}
			result = append(result, fmt.Sprintf("%4d %4d │ %s", oldLine, newLine, line))
			oldLine++
			newLine++
		}
	}

	// Flush any remaining block
	if collectingMinus || collectingPlus {
		flushBlock(&block, &result)
	}

	return strings.Join(result, "\n"), hunkPositions
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
	tabs := []string{"diff", "ctx", "full"}
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

func (d *DiffView) jumpToNextHunk() {
	offset := d.viewport.YOffset
	for _, pos := range d.hunkPositions {
		if pos > offset {
			d.viewport.SetYOffset(pos)
			return
		}
	}
}

func (d *DiffView) jumpToPrevHunk() {
	offset := d.viewport.YOffset
	for i := len(d.hunkPositions) - 1; i >= 0; i-- {
		if d.hunkPositions[i] < offset {
			d.viewport.SetYOffset(d.hunkPositions[i])
			return
		}
	}
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
		case "n":
			d.jumpToNextHunk()
			return *d, nil
		case "N":
			d.jumpToPrevHunk()
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
		style = style.BorderForeground(lipgloss.Color("2")).Bold(true)
	}
	// inactive: no BorderForeground = terminal default

	return style.Render(content)
}
