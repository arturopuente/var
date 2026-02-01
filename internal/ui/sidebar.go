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
func (d fileItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(FileItem)
	if !ok {
		return
	}

	statusStyle := lipgloss.NewStyle().Width(3)
	switch i.Status {
	case "M":
		statusStyle = statusStyle.Foreground(lipgloss.Color("3")) // Yellow
	case "A", "??":
		statusStyle = statusStyle.Foreground(lipgloss.Color("2")) // Green
	case "D":
		statusStyle = statusStyle.Foreground(lipgloss.Color("1")) // Red
	}

	status := statusStyle.Render(i.Status)
	path := i.Path

	str := fmt.Sprintf("%s %s", status, path)

	fn := lipgloss.NewStyle().PaddingLeft(1).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color("6")).
				Bold(true).
				Render(s...)
		}
	}

	fmt.Fprint(w, fn(str))
}

// Sidebar wraps a bubbles/list for file selection
type Sidebar struct {
	list        list.Model
	width       int
	height      int
	isFocused   bool
	revision    string // "working copy" or commit hash
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
