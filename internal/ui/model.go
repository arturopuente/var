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
	sidebar      Sidebar
	diffView     DiffView
	gitService   *git.Service
	deltaService *delta.Service

	focus  focus
	width  int
	height int

	// Commit navigation (repo-wide)
	commits     []git.Commit // All recent commits
	commitIndex int          // -1 for working copy, 0+ for commits

	// Current file selection
	currentFile string

	// Single-file mode
	singleFileMode   bool
	fileCommits      []git.Commit // Commits for current file
	fileCommitIndex  int          // -1 for working copy, 0+ for file commits

	err error
}

func NewModel(gitService *git.Service, deltaService *delta.Service) Model {
	sidebar := NewSidebar([]FileItem{}, 40, 20)
	sidebar.SetFocused(true)
	sidebar.SetRevision("working copy")
	diffView := NewDiffView(80, 20)

	return Model{
		sidebar:         sidebar,
		diffView:        diffView,
		gitService:      gitService,
		deltaService:    deltaService,
		focus:           focusSidebar,
		commitIndex:     -1, // Start at working copy
		fileCommitIndex: -1,
	}
}

func (m Model) Init() tea.Cmd {
	return m.loadInitialData
}

type initialDataMsg struct {
	commits []git.Commit
	files   []FileItem
}

func (m *Model) loadInitialData() tea.Msg {
	// Load recent commits
	commits, _ := m.gitService.GetRecentCommits(100)

	// Load working copy files
	files, _ := m.gitService.GetModifiedFiles()
	items := make([]FileItem, len(files))
	for i, f := range files {
		items[i] = FileItem{Path: f.Path, Status: f.Status}
	}

	return initialDataMsg{
		commits: commits,
		files:   items,
	}
}

type filesLoadedMsg struct {
	files []FileItem
}

type diffLoadedMsg struct {
	content string
}

type fileCommitsLoadedMsg struct {
	commits []git.Commit
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		case "enter":
			// Enter single-file mode
			if !m.sidebar.IsFiltering() && m.currentFile != "" && !m.singleFileMode {
				m.singleFileMode = true
				m.fileCommitIndex = -1 // Start at working copy
				m.focus = focusDiffView
				m.sidebar.SetFocused(false)
				m.diffView.SetFocused(true)
				return m, m.loadFileCommits
			}
		case "]":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					// Navigate file commits - newer
					if m.fileCommitIndex > -1 {
						m.fileCommitIndex--
						m.updateSingleFileModeDisplay()
						return m, m.loadDiffForFileCommit
					}
				} else {
					// Navigate repo commits - newer
					if m.commitIndex > -1 {
						m.commitIndex--
						return m, m.loadFilesForCurrentCommit
					}
				}
			}
		case "[":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					// Navigate file commits - older
					if m.fileCommitIndex < len(m.fileCommits)-1 {
						m.fileCommitIndex++
						m.updateSingleFileModeDisplay()
						return m, m.loadDiffForFileCommit
					}
				} else {
					// Navigate repo commits - older
					if m.commitIndex < len(m.commits)-1 {
						m.commitIndex++
						return m, m.loadFilesForCurrentCommit
					}
				}
			}
		case "esc":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					// Exit single-file mode
					m.singleFileMode = false
					m.fileCommitIndex = -1
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.diffView.SetFocused(false)
					m.updateRevisionDisplay()
					return m, m.loadDiffForCurrentFile
				} else if m.commitIndex >= 0 {
					// Return to working copy
					m.commitIndex = -1
					return m, m.loadFilesForCurrentCommit
				}
			}
		}

		// Route to focused component
		if !m.singleFileMode && (m.sidebar.IsFiltering() || m.focus == focusSidebar) {
			var cmd tea.Cmd
			prevSelected := m.sidebar.SelectedItem()
			m.sidebar, cmd = m.sidebar.Update(msg)
			cmds = append(cmds, cmd)

			// Check if selection changed
			currSelected := m.sidebar.SelectedItem()
			if currSelected != nil && (prevSelected == nil || prevSelected.Path != currSelected.Path) {
				m.currentFile = currSelected.Path
				cmds = append(cmds, m.loadDiffForCurrentFile)
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

	case initialDataMsg:
		m.commits = msg.commits
		m.sidebar.SetItems(msg.files)
		if len(msg.files) > 0 {
			m.currentFile = msg.files[0].Path
			cmds = append(cmds, m.loadDiffForCurrentFile)
		}
		m.updateRevisionDisplay()

	case filesLoadedMsg:
		m.sidebar.SetItems(msg.files)
		if len(msg.files) > 0 {
			m.currentFile = msg.files[0].Path
			cmds = append(cmds, m.loadDiffForCurrentFile)
		} else {
			m.currentFile = ""
			m.diffView.SetContent("No files changed in this commit")
		}
		m.updateRevisionDisplay()

	case fileCommitsLoadedMsg:
		m.fileCommits = msg.commits
		m.updateSingleFileModeDisplay()

	case diffLoadedMsg:
		m.diffView.SetContent(msg.content)

	case ErrorMsg:
		m.err = msg.Err
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * 0.20)
	diffWidth := m.width - sidebarWidth - 4

	m.sidebar.SetSize(sidebarWidth, m.height-2)
	m.diffView.SetSize(diffWidth, m.height-2)
}

func (m *Model) updateRevisionDisplay() {
	if m.commitIndex < 0 {
		m.sidebar.SetRevision("working copy")
		m.diffView.SetFileInfo(m.currentFile, -1, len(m.commits), "")
	} else if m.commitIndex < len(m.commits) {
		commit := m.commits[m.commitIndex]
		m.sidebar.SetRevision(commit.Hash)
		m.diffView.SetFileInfo(m.currentFile, m.commitIndex, len(m.commits), commit.Hash)
	}
}

func (m *Model) updateSingleFileModeDisplay() {
	if m.fileCommitIndex < 0 {
		m.sidebar.SetRevision("FILE: working copy")
		m.diffView.SetFileInfo(m.currentFile, -1, len(m.fileCommits), "")
	} else if m.fileCommitIndex < len(m.fileCommits) {
		commit := m.fileCommits[m.fileCommitIndex]
		m.sidebar.SetRevision("FILE: " + commit.Hash)
		m.diffView.SetFileInfo(m.currentFile, m.fileCommitIndex, len(m.fileCommits), commit.Hash)
	}
}

func (m *Model) loadFileCommits() tea.Msg {
	commits, _ := m.gitService.GetFileCommits(m.currentFile)
	return fileCommitsLoadedMsg{commits: commits}
}

func (m *Model) loadFilesForCurrentCommit() tea.Msg {
	var files []FileItem

	if m.commitIndex < 0 {
		// Working copy
		statusFiles, _ := m.gitService.GetModifiedFiles()
		for _, f := range statusFiles {
			files = append(files, FileItem{Path: f.Path, Status: f.Status})
		}
	} else if m.commitIndex < len(m.commits) {
		// Specific commit
		commit := m.commits[m.commitIndex]
		commitFiles, _ := m.gitService.GetFilesInCommit(commit.Hash)
		for _, f := range commitFiles {
			files = append(files, FileItem{Path: f.Path, Status: f.Status})
		}
	}

	return filesLoadedMsg{files: files}
}

func (m *Model) loadDiffForCurrentFile() tea.Msg {
	if m.currentFile == "" {
		return diffLoadedMsg{content: ""}
	}

	var diff string
	var err error

	if m.commitIndex < 0 {
		// Working copy diff
		diff, err = m.gitService.GetDiff(m.currentFile)
	} else if m.commitIndex < len(m.commits) {
		// Commit diff
		commit := m.commits[m.commitIndex]
		diff, err = m.gitService.GetDiffAtCommit(m.currentFile, commit.Hash)
	}

	if err != nil {
		return ErrorMsg{Err: err}
	}

	// Render through delta
	diffWidth := m.width - int(float64(m.width)*0.20) - 6
	rendered, err := m.deltaService.Render(diff, diffWidth)
	if err != nil {
		rendered = diff
	}

	return diffLoadedMsg{content: rendered}
}

func (m *Model) loadDiffForFileCommit() tea.Msg {
	if m.currentFile == "" {
		return diffLoadedMsg{content: ""}
	}

	var diff string
	var err error

	if m.fileCommitIndex < 0 {
		// Working copy diff
		diff, err = m.gitService.GetDiff(m.currentFile)
	} else if m.fileCommitIndex < len(m.fileCommits) {
		// File commit diff
		commit := m.fileCommits[m.fileCommitIndex]
		diff, err = m.gitService.GetDiffAtCommit(m.currentFile, commit.Hash)
	}

	if err != nil {
		return ErrorMsg{Err: err}
	}

	// Render through delta
	diffWidth := m.width - int(float64(m.width)*0.20) - 6
	rendered, err := m.deltaService.Render(diff, diffWidth)
	if err != nil {
		rendered = diff
	}

	return diffLoadedMsg{content: rendered}
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	var help string
	if m.singleFileMode {
		help = HelpStyle.Render("[d/u: scroll | [/]: file history | esc: exit file mode | q: quit]")
	} else {
		help = HelpStyle.Render("[j/k: files | enter: file mode | [/]: commits | t: filter | esc: working copy | q: quit]")
	}

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
