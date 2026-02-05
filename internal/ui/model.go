package ui

import (
	"strings"
	"var/internal/git"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusSidebar focus = iota
	focusDiffView
)

type viewMode int

const (
	viewModeDiff       viewMode = iota // Default diff (3 lines context)
	viewModeContext                    // Diff with 10 lines context
	viewModeFullFile                   // Full file view
)

// Model is the root model composing sidebar and diff view
type Model struct {
	sidebar    Sidebar
	diffView   DiffView
	gitService *git.Service

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
	viewMode         viewMode     // Current view mode in single-file mode

	err error
}

func NewModel(gitService *git.Service) Model {
	sidebar := NewSidebar([]FileItem{}, 40, 20)
	sidebar.SetFocused(true)
	sidebar.SetRevision("working copy")
	diffView := NewDiffView(80, 20)

	return Model{
		sidebar:         sidebar,
		diffView:        diffView,
		gitService:      gitService,
		focus:           focusSidebar,
		commitIndex:     0, // Start at latest commit
		fileCommitIndex: 0,
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

	// Load files from first commit
	var items []FileItem
	if len(commits) > 0 {
		files, _ := m.gitService.GetFilesInCommit(commits[0].Hash)
		stats, _ := m.gitService.GetNumstatForCommit(commits[0].Hash)
		items = make([]FileItem, len(files))
		for i, f := range files {
			item := FileItem{Path: f.Path, Status: f.Status}
			if stats != nil {
				if s, ok := stats[f.Path]; ok {
					item.Additions = s.Additions
					item.Deletions = s.Deletions
				}
			}
			items[i] = item
		}
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
				if m.singleFileMode {
					// Exit single-file mode
					m.singleFileMode = false
					m.fileCommitIndex = 0
					m.viewMode = viewModeDiff
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.diffView.SetFocused(false)
					m.diffView.SetMode(false, 0)
					m.updateRevisionDisplay()
					return m, m.loadDiffForCurrentFile
				}
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
		case " ":
			// Enter single-file mode
			if !m.sidebar.IsFiltering() && m.currentFile != "" && !m.singleFileMode {
				m.singleFileMode = true
				m.fileCommitIndex = 0 // Start at most recent commit
				m.focus = focusDiffView
				m.sidebar.SetFocused(false)
				m.diffView.SetFocused(true)
				m.diffView.SetMode(true, int(m.viewMode))
				return m, m.loadFileCommits
			}
		case "]":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					// Navigate file commits - newer
					if m.fileCommitIndex > 0 {
						m.fileCommitIndex--
						m.updateSingleFileModeDisplay()
						return m, m.loadDiffForFileCommit
					}
				} else {
					// Navigate repo commits - newer
					if m.commitIndex > 0 {
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
		case "1":
			// Switch to commit list mode
			if !m.sidebar.IsFiltering() && m.singleFileMode {
				m.singleFileMode = false
				m.fileCommitIndex = 0
				m.viewMode = viewModeDiff
				m.focus = focusSidebar
				m.sidebar.SetFocused(true)
				m.diffView.SetFocused(false)
				m.diffView.SetMode(false, 0)
				m.updateRevisionDisplay()
				return m, m.loadDiffForCurrentFile
			}
		case "2":
			// Switch to single-file mode
			if !m.sidebar.IsFiltering() && m.currentFile != "" && !m.singleFileMode {
				m.singleFileMode = true
				m.fileCommitIndex = 0
				m.focus = focusDiffView
				m.sidebar.SetFocused(false)
				m.diffView.SetFocused(true)
				m.diffView.SetMode(true, int(m.viewMode))
				return m, m.loadFileCommits
			}
		case "c":
			// Cycle diff modes in single-file mode
			if m.singleFileMode {
				m.viewMode = (m.viewMode + 1) % 3
				m.diffView.SetMode(true, int(m.viewMode))
				return m, m.loadDiffForFileCommit
			}
		case "z":
			if !m.sidebar.IsFiltering() {
				m.diffView.ToggleDescription()
				return m, nil
			}
		case "esc":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					// Exit single-file mode
					m.singleFileMode = false
					m.fileCommitIndex = 0
					m.viewMode = viewModeDiff
					m.focus = focusSidebar
					m.sidebar.SetFocused(true)
					m.diffView.SetFocused(false)
					m.diffView.SetMode(false, 0)
					m.updateRevisionDisplay()
					return m, m.loadDiffForCurrentFile
				} else if m.commitIndex > 0 {
					// Return to latest commit
					m.commitIndex = 0
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

	m.sidebar.SetSize(sidebarWidth, m.height-3)
	m.diffView.SetSize(diffWidth, m.height-3)
}

func (m *Model) updateRevisionDisplay() {
	if m.commitIndex < len(m.commits) {
		commit := m.commits[m.commitIndex]
		m.sidebar.SetRevision(commit.Hash)
		m.diffView.SetFileInfo(m.currentFile, m.commitIndex, len(m.commits), commit.Hash)
	}
}

func (m *Model) updateSingleFileModeDisplay() {
	if m.fileCommitIndex < len(m.fileCommits) {
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

	if m.commitIndex < len(m.commits) {
		commit := m.commits[m.commitIndex]
		commitFiles, _ := m.gitService.GetFilesInCommit(commit.Hash)
		stats, _ := m.gitService.GetNumstatForCommit(commit.Hash)
		for _, f := range commitFiles {
			item := FileItem{Path: f.Path, Status: f.Status}
			if stats != nil {
				if s, ok := stats[f.Path]; ok {
					item.Additions = s.Additions
					item.Deletions = s.Deletions
				}
			}
			files = append(files, item)
		}
	}

	return filesLoadedMsg{files: files}
}

func (m *Model) loadDiffForCurrentFile() tea.Msg {
	if m.currentFile == "" || m.commitIndex >= len(m.commits) {
		return diffLoadedMsg{content: ""}
	}

	commit := m.commits[m.commitIndex]
	diff, err := m.gitService.GetDiffAtCommit(m.currentFile, commit.Hash)

	if err != nil {
		return ErrorMsg{Err: err}
	}

	if diff == "" {
		return diffLoadedMsg{content: "No changes to display"}
	}

	return diffLoadedMsg{content: diff}
}

func (m *Model) loadDiffForFileCommit() tea.Msg {
	if m.currentFile == "" || m.fileCommitIndex >= len(m.fileCommits) {
		return diffLoadedMsg{content: ""}
	}

	commit := m.fileCommits[m.fileCommitIndex]
	var content string
	var err error

	switch m.viewMode {
	case viewModeFullFile:
		// Full file view
		content, err = m.gitService.GetFileContentAtCommit(m.currentFile, commit.Hash)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return diffLoadedMsg{content: content}

	case viewModeContext:
		// Diff with 10 lines of context
		content, err = m.gitService.GetDiffAtCommitWithContext(m.currentFile, commit.Hash, 10)

	default:
		// Default diff (3 lines context)
		content, err = m.gitService.GetDiffAtCommit(m.currentFile, commit.Hash)
	}

	if err != nil {
		return ErrorMsg{Err: err}
	}

	if content == "" {
		return diffLoadedMsg{content: "No changes to display"}
	}

	return diffLoadedMsg{content: content}
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
		badge := ModeBadgeFile.Render("FILE")
		helpText := HelpStyle.Render("[c: cycle view | d/u: scroll | n/N: hunks | [/]: history | z: desc | 1: back]")
		help = badge + " " + helpText
	} else {
		badge := ModeBadgeCommits.Render("COMMITS")
		helpText := HelpStyle.Render("[j/k: files | 2/space: file mode | [/]: commits | /: filter | n/N: hunks | z: desc | q: quit]")
		help = badge + " " + helpText
	}

	sidebarRendered := injectBorderLabel(m.sidebar.View(), "1", m.focus == focusSidebar)
	diffRendered := injectBorderLabel(m.diffView.View(), "2", m.focus == focusDiffView)

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarRendered,
		diffRendered,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		main,
		help,
	)
}

// injectBorderLabel replaces part of the top border with a centered label like [1]
func injectBorderLabel(rendered string, label string, focused bool) string {
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}

	clean := stripANSI(lines[0])
	runes := []rune(clean)
	labelRunes := []rune("[" + label + "]")

	start := 2 // after ╭─
	for i, r := range labelRunes {
		pos := start + i
		if pos > 0 && pos < len(runes)-1 {
			runes[pos] = r
		}
	}

	newTop := string(runes)
	if focused {
		newTop = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true).Render(newTop)
	}

	lines[0] = newTop
	return strings.Join(lines, "\n")
}
