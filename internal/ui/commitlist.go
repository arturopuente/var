package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommitItem represents a commit in the commit list
type CommitItem struct {
	Hash    string
	Message string
}

func (i CommitItem) FilterValue() string { return i.Message }

type commitItemDelegate struct{}

func (d commitItemDelegate) Height() int                             { return 1 }
func (d commitItemDelegate) Spacing() int                            { return 0 }
func (d commitItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d commitItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(CommitItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	width := m.Width()

	// Short hash (7 chars) + space + message
	hash := i.Hash
	if len(hash) > 7 {
		hash = hash[:7]
	}

	// Truncate message to fit: width - 2 (indent) - 7 (hash) - 1 (space) - 2 (margin)
	maxMsgLen := width - 12
	msg := i.Message
	if maxMsgLen > 0 && len(msg) > maxMsgLen {
		if maxMsgLen > 3 {
			msg = msg[:maxMsgLen-1] + "…"
		} else {
			msg = msg[:maxMsgLen]
		}
	}

	if isSelected {
		bg := lipgloss.Color("#0066cc")
		fg := lipgloss.Color("#ffffff")
		hashStyle := lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true)
		msgStyle := lipgloss.NewStyle().Foreground(fg).Background(bg)
		line := fmt.Sprintf("  %s %s", hashStyle.Render(hash), msgStyle.Render(msg))
		fmt.Fprint(w, lipgloss.NewStyle().Width(width).Background(bg).Render(line))
	} else {
		hashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
		line := fmt.Sprintf("  %s %s", hashStyle.Render(hash), msg)
		fmt.Fprint(w, line)
	}
}

// CommitList wraps a bubbles/list for commit selection
type CommitList struct {
	list      list.Model
	width     int
	height    int
	isFocused bool
	label     string
}

func NewCommitList(width, height int) CommitList {
	l := list.New([]list.Item{}, commitItemDelegate{}, width, height)
	l.Title = "Commits"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	return CommitList{
		list:   l,
		width:  width,
		height: height,
		label:  "Commits",
	}
}

func (c *CommitList) SetItems(items []CommitItem) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	c.list.SetItems(listItems)
}

func (c *CommitList) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.list.SetSize(width, height)
}

func (c *CommitList) SetFocused(focused bool) {
	c.isFocused = focused
}

func (c *CommitList) IsFocused() bool {
	return c.isFocused
}

func (c *CommitList) SetTitle(title string) {
	c.label = title
	c.list.Title = title
}

func (c *CommitList) SelectedItem() *CommitItem {
	item := c.list.SelectedItem()
	if item == nil {
		return nil
	}
	ci := item.(CommitItem)
	return &ci
}

func (c *CommitList) SelectedIndex() int {
	return c.list.Index()
}

func (c *CommitList) SelectIndex(index int) {
	c.list.Select(index)
}

func (c *CommitList) Update(msg tea.Msg) (CommitList, tea.Cmd) {
	var cmd tea.Cmd
	c.list, cmd = c.list.Update(msg)
	return *c, cmd
}

func (c *CommitList) View() string {
	style := lipgloss.NewStyle().
		Width(c.width).
		Height(c.height).
		BorderStyle(lipgloss.RoundedBorder())

	if c.isFocused {
		style = style.BorderForeground(lipgloss.Color("2"))
	}

	return style.Render(c.list.View())
}
