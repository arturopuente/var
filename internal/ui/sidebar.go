package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileItem represents a file in the sidebar
type FileItem struct {
	Path      string
	Status    string
	Additions int
	Deletions int
}

func (i FileItem) FilterValue() string { return i.Path }

type fileItemDelegate struct{}

func (d fileItemDelegate) Height() int                             { return 1 }
func (d fileItemDelegate) Spacing() int                            { return 0 }
func (d fileItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
// truncatePath shortens a path to fit within maxLen, showing start and end
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen || maxLen <= 5 {
		return path
	}
	// Show first 3 chars + … + end
	endLen := maxLen - 4 // 3 for start + 1 for …
	return path[:3] + "…" + path[len(path)-endLen:]
}

func (d fileItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(FileItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	width := m.Width()

	// Format stats string
	var stats string
	if i.Additions > 0 || i.Deletions > 0 {
		stats = fmt.Sprintf("+%d -%d", i.Additions, i.Deletions)
	}

	// Truncate path to fit: width - 2 (indent) - 3 (status) - 1 (space) - 2 (margin) - stats - 1 (space before stats)
	statsWidth := 0
	if stats != "" {
		statsWidth = len(stats) + 1
	}
	maxPathLen := width - 8 - statsWidth
	path := truncatePath(i.Path, maxPathLen)

	// Determine status color
	var statusColor lipgloss.Color
	switch i.Status {
	case "M":
		statusColor = lipgloss.Color("3") // Yellow
	case "A", "??":
		statusColor = lipgloss.Color("2") // Green
	case "D":
		statusColor = lipgloss.Color("1") // Red
	default:
		statusColor = lipgloss.Color("7") // White/default
	}

	if isSelected {
		// Selected: blue background, white text (using hex colors)
		bg := lipgloss.Color("#0066cc")
		fg := lipgloss.Color("#ffffff")
		statusStyle := lipgloss.NewStyle().Width(3).Foreground(fg).Background(bg).Bold(true)
		pathStyle := lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true)
		statsStyle := lipgloss.NewStyle().Foreground(fg).Background(bg)

		pathRendered := pathStyle.Render(path)
		if stats != "" {
			// Pad path to push stats to the right
			padLen := maxPathLen - len(path)
			if padLen < 0 {
				padLen = 0
			}
			padding := lipgloss.NewStyle().Background(bg).Render(fmt.Sprintf("%*s", padLen, ""))
			line := fmt.Sprintf("  %s %s%s %s", statusStyle.Render(i.Status), pathRendered, padding, statsStyle.Render(stats))
			fmt.Fprint(w, lipgloss.NewStyle().Width(width).Background(bg).Render(line))
		} else {
			line := fmt.Sprintf("  %s %s", statusStyle.Render(i.Status), pathRendered)
			fmt.Fprint(w, lipgloss.NewStyle().Width(width).Background(bg).Render(line))
		}
	} else {
		// Unselected: normal styling
		statusStyle := lipgloss.NewStyle().Width(3).Foreground(statusColor)
		if stats != "" {
			padLen := maxPathLen - len(path)
			if padLen < 0 {
				padLen = 0
			}
			addStr := fmt.Sprintf("+%d", i.Additions)
			delStr := fmt.Sprintf("-%d", i.Deletions)
			greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
			redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
			line := fmt.Sprintf("  %s %s%*s %s %s", statusStyle.Render(i.Status), path, padLen, "", greenStyle.Render(addStr), redStyle.Render(delStr))
			fmt.Fprint(w, line)
		} else {
			line := fmt.Sprintf("  %s %s", statusStyle.Render(i.Status), path)
			fmt.Fprint(w, line)
		}
	}
}

// Sidebar wraps a bubbles/list for file selection
type Sidebar struct {
	list      list.Model
	width     int
	height    int
	isFocused bool
	revision  string // "working copy" or commit hash
}

func NewSidebar(items []FileItem, width, height int) Sidebar {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	l := list.New(listItems, fileItemDelegate{}, width, height)
	l.Title = "Files"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	return Sidebar{
		list:      l,
		width:     width,
		height:    height,
		isFocused: true,
	}
}

func (s *Sidebar) SetItems(items []FileItem) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	s.list.SetItems(listItems)
}

func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width, height)
}

func (s *Sidebar) SetFocused(focused bool) {
	s.isFocused = focused
}

func (s *Sidebar) IsFocused() bool {
	return s.isFocused
}

func (s *Sidebar) SetRevision(revision string) {
	s.revision = revision
	if revision == "" || revision == "working copy" {
		s.list.Title = "Files (working copy)"
	} else {
		s.list.Title = fmt.Sprintf("Files (%s)", revision)
	}
}

func (s *Sidebar) IsFiltering() bool {
	return s.list.FilterState() == list.Filtering
}

func (s *Sidebar) SelectedItem() *FileItem {
	item := s.list.SelectedItem()
	if item == nil {
		return nil
	}
	fi := item.(FileItem)
	return &fi
}

func (s *Sidebar) Update(msg tea.Msg) (Sidebar, tea.Cmd) {
	var cmd tea.Cmd
	s.list, cmd = s.list.Update(msg)
	return *s, cmd
}

func (s *Sidebar) View() string {
	style := lipgloss.NewStyle().
		Width(s.width).
		Height(s.height).
		BorderStyle(lipgloss.RoundedBorder())

	if s.isFocused {
		// lazygit: green + bold for active border
		style = style.BorderForeground(lipgloss.Color("2")) // green for active border
	}
	// inactive: no BorderForeground = terminal default

	return style.Render(s.list.View())
}
