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
	Path   string
	Status string
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

	// Truncate path to fit: width - 2 (indent) - 3 (status) - 1 (space) - 2 (margin)
	maxPathLen := width - 8
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

		line := fmt.Sprintf("  %s %s", statusStyle.Render(i.Status), pathStyle.Render(path))
		fmt.Fprint(w, lipgloss.NewStyle().Width(width).Background(bg).Render(line))
	} else {
		// Unselected: normal styling
		statusStyle := lipgloss.NewStyle().Width(3).Foreground(statusColor)
		line := fmt.Sprintf("  %s %s", statusStyle.Render(i.Status), path)
		fmt.Fprint(w, line)
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
		style = style.BorderForeground(lipgloss.Color("2")) // green
	}
	// inactive: no BorderForeground = terminal default

	return style.Render(s.list.View())
}
