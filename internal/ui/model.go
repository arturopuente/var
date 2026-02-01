package ui

import (
	"var/internal/delta"
	"var/internal/git"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusSidebar focus = iota
	focusDiffView
)

// Model is the root model composing sidebar and diff view
type Model struct {
	sidebar     Sidebar
	diffView    DiffView
	gitService  *git.Service
	deltaService *delta.Service

	focus       focus
	width       int
	height      int

	currentFile string
	commits     []git.Commit
	commitIndex int // -1 for working copy

	err         error
}

func NewModel(gitService *git.Service, deltaService *delta.Service) Model {
	// Initialize with empty items and default dimensions
	// They'll be resized when WindowSizeMsg arrives
	sidebar := NewSidebar([]FileItem{}, 40, 20)
	sidebar.SetFocused(true)
	diffView := NewDiffView(80, 20)

	return Model{
		sidebar:      sidebar,
		diffView:     diffView,
		gitService:   gitService,
		deltaService: deltaService,
		focus:        focusSidebar,
		commitIndex:  -1,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadFiles
}

func (m *Model) loadFiles() tea.Msg {
	files, err := m.gitService.GetModifiedFiles()
	if err != nil {
		return ErrorMsg{Err: err}
	}

	items := make([]FileItem, len(files))
	for i, f := range files {
		items[i] = FileItem{Path: f.Path, Status: f.Status}
	}

	return filesLoadedMsg{items: items}
}

type filesLoadedMsg struct {
	items []FileItem
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "q", "ctrl+c":
			if !m.sidebar.IsFiltering() {
				return m, tea.Quit
			}
		case "tab":
			if !m.sidebar.IsFiltering() {
				if m.focus == focusSidebar {
					m.focus = focusDiffView
					m.sidebar.SetFocused(false)
					m.diffView.SetFocused(true)
				} else {
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.diffView.SetFocused(false)
				}
				return m, nil
			}
		case "]":
			// Next commit (newer)
			if !m.sidebar.IsFiltering() && m.commitIndex > -1 {
				m.commitIndex--
				return m, m.loadDiffForCurrentCommit
			}
		case "[":
			// Previous commit (older)
			if !m.sidebar.IsFiltering() && m.commitIndex < len(m.commits)-1 {
				m.commitIndex++
				return m, m.loadDiffForCurrentCommit
			}
		case "esc":
			if !m.sidebar.IsFiltering() && m.commitIndex >= 0 {
				// Return to working copy
				m.commitIndex = -1
				return m, m.loadDiffForWorkingCopy
			}
		}

		// Route to focused component
		if m.sidebar.IsFiltering() || m.focus == focusSidebar {
			var cmd tea.Cmd
			prevSelected := m.sidebar.SelectedItem()
			m.sidebar, cmd = m.sidebar.Update(msg)
			cmds = append(cmds, cmd)

			// Check if selection changed
			currSelected := m.sidebar.SelectedItem()
			if currSelected != nil && (prevSelected == nil || prevSelected.Path != currSelected.Path) {
				m.currentFile = currSelected.Path
				m.commitIndex = -1
				cmds = append(cmds, m.loadDiffAndCommits)
			}
		} else if m.focus == focusDiffView {
			var cmd tea.Cmd
			m.diffView, cmd = m.diffView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case filesLoadedMsg:
		m.sidebar.SetItems(msg.items)
		if len(msg.items) > 0 {
			m.currentFile = msg.items[0].Path
			cmds = append(cmds, m.loadDiffAndCommits)
		}

	case diffLoadedMsg:
		m.diffView.SetContent(msg.content)
		m.diffView.SetFileInfo(msg.path, msg.commitIndex, msg.commitCount, msg.commitHash)

	case ErrorMsg:
		m.err = msg.Err
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * 0.25)
	diffWidth := m.width - sidebarWidth - 4 // Account for borders

	m.sidebar.SetSize(sidebarWidth, m.height-2)
	m.diffView.SetSize(diffWidth, m.height-2)
}

type diffLoadedMsg struct {
	content     string
	path        string
	commitIndex int
	commitCount int
	commitHash  string
}

func (m *Model) loadDiffAndCommits() tea.Msg {
	if m.currentFile == "" {
		return nil
	}

	// Load commits for this file
	commits, _ := m.gitService.GetFileCommits(m.currentFile)
	m.commits = commits

	// Load working copy diff
	diff, err := m.gitService.GetDiff(m.currentFile)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	// Render through delta
	diffWidth := m.width - int(float64(m.width)*0.25) - 6
	rendered, err := m.deltaService.Render(diff, diffWidth)
	if err != nil {
		rendered = diff
	}

	return diffLoadedMsg{
		content:     rendered,
		path:        m.currentFile,
		commitIndex: -1,
		commitCount: len(commits),
		commitHash:  "",
	}
}

func (m *Model) loadDiffForCurrentCommit() tea.Msg {
	if m.currentFile == "" || m.commitIndex < 0 || m.commitIndex >= len(m.commits) {
		return nil
	}

	commit := m.commits[m.commitIndex]
	diff, err := m.gitService.GetDiffAtCommit(m.currentFile, commit.Hash)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	diffWidth := m.width - int(float64(m.width)*0.25) - 6
	rendered, err := m.deltaService.Render(diff, diffWidth)
	if err != nil {
		rendered = diff
	}

	return diffLoadedMsg{
		content:     rendered,
		path:        m.currentFile,
		commitIndex: m.commitIndex,
		commitCount: len(m.commits),
		commitHash:  commit.Hash,
	}
}

func (m *Model) loadDiffForWorkingCopy() tea.Msg {
	if m.currentFile == "" {
		return nil
	}

	diff, err := m.gitService.GetDiff(m.currentFile)
	if err != nil {
		return ErrorMsg{Err: err}
	}

	diffWidth := m.width - int(float64(m.width)*0.25) - 6
	rendered, err := m.deltaService.Render(diff, diffWidth)
	if err != nil {
		rendered = diff
	}

	return diffLoadedMsg{
		content:     rendered,
		path:        m.currentFile,
		commitIndex: -1,
		commitCount: len(m.commits),
		commitHash:  "",
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	help := HelpStyle.Render("[j/k: files | d/u: scroll | [/]: revisions | t: filter | tab: switch pane | esc: working copy | q: quit]")

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.sidebar.View(),
		m.diffView.View(),
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		main,
		help,
	)
}
