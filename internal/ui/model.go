package ui

import (
	"fmt"
	"strings"
	"var/internal/git"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusCommitList focus = iota
	focusFileList
	focusDiffView
	focusFileTree
)

type displayMode int

const (
	displayDiff    displayMode = iota // Default diff (3 lines context)
	displayContext                    // Diff with 10 lines context
	displayFull                      // Full file view
	displayBlame                     // Blame annotations
)

type sourceMode int

const (
	sourceCommits sourceMode = iota // git log --follow (default)
	sourceReflog                    // git log -g
	sourcePickaxe                   // git log -S
)

// Model is the root model composing commit list, sidebar, and diff view
type Model struct {
	commitList CommitList
	sidebar    Sidebar
	diffView   DiffView
	fileTree   FileTree
	gitService *git.Service

	focus        focus
	showFileTree bool
	width        int
	height       int

	// Commit navigation (repo-wide)
	commits     []git.Commit // All recent commits
	commitIndex int          // -1 for working copy, 0+ for commits

	// Current file selection
	currentFile string

	// Single-file mode
	singleFileMode  bool
	fileCommits     []git.Commit // Commits for current file
	fileCommitIndex int          // -1 for working copy, 0+ for file commits
	displayMode     displayMode  // Current display format
	sourceMode      sourceMode   // Current commit source

	// Source-specific state
	reflogEntries []git.Commit
	reflogIndex   int
	sourceCommits []git.Commit // Commits from pickaxe
	sourceIndex   int
	pickaxeTerm   string // Active search term for pickaxe

	// Text input for pickaxe
	textInput     textinput.Model
	textInputMode string // "pickaxe" or ""

	err error
}

func NewModel(gitService *git.Service) Model {
	commitList := NewCommitList(40, 10)
	commitList.SetFocused(true)

	sidebar := NewSidebar([]FileItem{}, 40, 10)
	sidebar.SetRevision("working copy")
	diffView := NewDiffView(80, 20)
	fileTree := NewFileTree(40, 20)

	ti := textinput.New()
	ti.CharLimit = 128

	return Model{
		commitList:      commitList,
		sidebar:         sidebar,
		diffView:        diffView,
		fileTree:        fileTree,
		gitService:      gitService,
		focus:           focusCommitList,
		commitIndex:     0, // Start at latest commit
		fileCommitIndex: 0,
		textInput:       ti,
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

type reflogLoadedMsg struct {
	entries []git.Commit
}

type sourceCommitsLoadedMsg struct {
	commits []git.Commit
	err     error
}

type treeFilesLoadedMsg struct {
	paths []string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle text input mode first
		if m.textInputMode != "" {
			switch msg.String() {
			case "enter":
				value := m.textInput.Value()
				if value != "" {
					mode := m.textInputMode
					m.textInputMode = ""
					m.textInput.Blur()
					if mode == "pickaxe" {
						m.pickaxeTerm = value
						m.sourceMode = sourcePickaxe
						m.sourceIndex = 0
						m.updateSourceIndicator()
						return m, m.loadPickaxeCommits
					}
				}
				m.textInputMode = ""
				m.textInput.Blur()
				return m, nil
			case "esc":
				m.textInputMode = ""
				m.textInput.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.sidebar.IsFiltering() {
				if m.showFileTree {
					m.showFileTree = false
					m.setFocus(focusCommitList)
					m.updateLayout()
					return m, nil
				}
				if m.singleFileMode {
					// Exit single-file mode
					m.exitSingleFileMode()
					return m, m.loadDiffForCurrentFile
				}
				return m, tea.Quit
			}
		case "tab":
			if !m.sidebar.IsFiltering() {
				if m.showFileTree {
					// Tree mode: toggle between tree and diff
					if m.focus == focusFileTree {
						m.setFocus(focusDiffView)
					} else {
						m.setFocus(focusFileTree)
					}
				} else {
					switch m.focus {
					case focusCommitList:
						m.setFocus(focusFileList)
					case focusFileList:
						m.setFocus(focusDiffView)
					case focusDiffView:
						m.setFocus(focusCommitList)
					}
				}
				return m, nil
			}
		case "t":
			// Toggle file tree (only in commits mode, not single-file, not filtering)
			if !m.sidebar.IsFiltering() && !m.singleFileMode {
				m.showFileTree = !m.showFileTree
				if m.showFileTree {
					m.setFocus(focusFileTree)
					m.updateLayout()
					return m, m.loadTreeFiles
				}
				m.setFocus(focusCommitList)
				m.updateLayout()
				return m, nil
			}
		case " ", "enter":
			// File tree: select a file to enter single-file mode
			if m.showFileTree && m.focus == focusFileTree && !m.fileTree.IsSelectedDir() {
				selectedPath := m.fileTree.SelectedPath()
				if selectedPath != "" {
					m.currentFile = selectedPath
					m.showFileTree = false
					m.enterSingleFileMode()
					m.updateLayout()
					return m, m.loadFileCommits
				}
				return m, nil
			}
			// Enter single-file mode from file list
			if !m.sidebar.IsFiltering() && m.focus == focusFileList && m.currentFile != "" && !m.singleFileMode {
				m.enterSingleFileMode()
				return m, m.loadFileCommits
			}
		case "]":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					cmd := m.navigateNewer()
					m.syncCommitListToIndex()
					return m, cmd
				}
				// Navigate repo commits - newer
				if m.commitIndex > 0 {
					m.commitIndex--
					m.commitList.SelectIndex(m.commitIndex)
					return m, m.loadFilesForCurrentCommit
				}
			}
		case "[":
			if !m.sidebar.IsFiltering() {
				if m.singleFileMode {
					cmd := m.navigateOlder()
					m.syncCommitListToIndex()
					return m, cmd
				}
				// Navigate repo commits - older
				if m.commitIndex < len(m.commits)-1 {
					m.commitIndex++
					m.commitList.SelectIndex(m.commitIndex)
					return m, m.loadFilesForCurrentCommit
				}
			}
		case "1":
			if !m.sidebar.IsFiltering() {
				if m.showFileTree {
					m.setFocus(focusFileTree)
				} else {
					m.setFocus(focusCommitList)
				}
				return m, nil
			}
		case "2":
			if !m.sidebar.IsFiltering() {
				if m.showFileTree {
					m.setFocus(focusFileTree) // no panel 2 in tree mode
				} else {
					m.setFocus(focusFileList)
				}
				return m, nil
			}
		case "3":
			if !m.sidebar.IsFiltering() {
				m.setFocus(focusDiffView)
				return m, nil
			}
		case "c":
			// Cycle display modes in single-file mode
			if m.singleFileMode {
				m.displayMode = (m.displayMode + 1) % 4
				m.diffView.SetMode(true, int(m.displayMode))
				return m, m.loadContentForCurrentSource()
			}
		case "r":
			// Toggle reflog source
			if m.singleFileMode {
				if m.sourceMode == sourceReflog {
					m.sourceMode = sourceCommits
					m.updateSourceIndicator()
					m.updateSingleFileModeDisplay()
					return m, m.loadContentForCurrentSource()
				}
				m.sourceMode = sourceReflog
				m.reflogIndex = 0
				m.updateSourceIndicator()
				return m, m.loadReflog
			}
		case "s":
			// Toggle pickaxe source
			if m.singleFileMode {
				if m.sourceMode == sourcePickaxe {
					// Deactivate pickaxe
					m.sourceMode = sourceCommits
					m.pickaxeTerm = ""
					m.updateSourceIndicator()
					m.updateSingleFileModeDisplay()
					return m, m.loadContentForCurrentSource()
				}
				// Activate text input for search term
				m.textInput.SetValue("")
				m.textInput.Placeholder = "search term"
				m.textInput.Focus()
				m.textInputMode = "pickaxe"
				return m, textinput.Blink
			}
		case "z":
			if !m.sidebar.IsFiltering() {
				m.diffView.ToggleDescription()
				return m, nil
			}
		case "esc":
			if !m.sidebar.IsFiltering() {
				if m.showFileTree {
					m.showFileTree = false
					m.setFocus(focusCommitList)
					m.updateLayout()
					return m, nil
				}
				if m.singleFileMode {
					// If a source is active, deactivate it first
					if m.sourceMode != sourceCommits {
						m.sourceMode = sourceCommits
						m.pickaxeTerm = ""
						m.updateSourceIndicator()
						m.updateSingleFileModeDisplay()
						return m, m.loadContentForCurrentSource()
					}
					// Exit single-file mode
					m.exitSingleFileMode()
					return m, m.loadDiffForCurrentFile
				} else if m.commitIndex > 0 {
					// Return to latest commit
					m.commitIndex = 0
					return m, m.loadFilesForCurrentCommit
				}
			}
		}

		// Route to focused component
		if m.focus == focusFileTree {
			var cmd tea.Cmd
			m.fileTree, cmd = m.fileTree.Update(msg)
			cmds = append(cmds, cmd)
		} else if m.focus == focusCommitList {
			var cmd tea.Cmd
			prevIdx := m.commitList.SelectedIndex()
			m.commitList, cmd = m.commitList.Update(msg)
			cmds = append(cmds, cmd)

			// Check if commit selection changed
			newIdx := m.commitList.SelectedIndex()
			if newIdx != prevIdx {
				if m.singleFileMode {
					// In single-file mode, navigate file history
					m.fileCommitIndex = newIdx
					m.updateSingleFileModeDisplay()
					cmds = append(cmds, m.loadContentForCurrentSource())
				} else {
					// In commits mode, load files for selected commit
					m.commitIndex = newIdx
					cmds = append(cmds, m.loadFilesForCurrentCommit)
				}
			}
		} else if m.sidebar.IsFiltering() || m.focus == focusFileList {
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
		m.populateCommitList(msg.commits)
		m.commitList.SelectIndex(m.commitIndex)
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
		m.populateCommitList(msg.commits)
		m.commitList.SetTitle("History")
		m.commitList.SelectIndex(m.fileCommitIndex)
		m.updateSingleFileModeDisplay()
		cmds = append(cmds, m.loadContentForCurrentSource())

	case reflogLoadedMsg:
		m.reflogEntries = msg.entries
		m.populateCommitList(msg.entries)
		m.commitList.SetTitle("Reflog")
		m.commitList.SelectIndex(m.reflogIndex)
		m.updateReflogDisplay()
		cmds = append(cmds, m.loadContentForCurrentSource())

	case sourceCommitsLoadedMsg:
		if msg.err != nil || len(msg.commits) == 0 {
			errMsg := "No commits found"
			if msg.err != nil {
				errMsg = fmt.Sprintf("Error: %v", msg.err)
			}
			m.sourceMode = sourceCommits
			m.pickaxeTerm = ""
			m.updateSourceIndicator()
			m.updateSingleFileModeDisplay()
			m.diffView.SetContent(errMsg)
		} else {
			m.sourceCommits = msg.commits
			m.populateCommitList(msg.commits)
			m.commitList.SetTitle(fmt.Sprintf("S:\"%s\"", m.pickaxeTerm))
			m.commitList.SelectIndex(m.sourceIndex)
			m.updateSourceDisplay()
			cmds = append(cmds, m.loadContentForCurrentSource())
		}

	case treeFilesLoadedMsg:
		m.fileTree.SetFiles(msg.paths)

	case diffLoadedMsg:
		m.diffView.SetContent(msg.content)

	case ErrorMsg:
		m.err = msg.Err
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) setFocus(f focus) {
	m.focus = f
	m.commitList.SetFocused(f == focusCommitList)
	m.sidebar.SetFocused(f == focusFileList)
	m.diffView.SetFocused(f == focusDiffView)
	m.fileTree.SetFocused(f == focusFileTree)
}

func (m *Model) enterSingleFileMode() {
	m.singleFileMode = true
	m.fileCommitIndex = 0
	m.setFocus(focusDiffView)
	m.diffView.SetMode(true, int(m.displayMode))
	m.updateSourceIndicator()
}

func (m *Model) exitSingleFileMode() {
	m.singleFileMode = false
	m.fileCommitIndex = 0
	m.displayMode = displayDiff
	m.sourceMode = sourceCommits
	m.pickaxeTerm = ""
	m.setFocus(focusCommitList)
	m.diffView.SetMode(false, 0)
	m.diffView.SetSourceIndicator("")
	// Restore repo commits in commit list
	m.populateCommitList(m.commits)
	m.commitList.SetTitle("Commits")
	m.commitList.SelectIndex(m.commitIndex)
	m.updateRevisionDisplay()
}

// syncCommitListToIndex updates the commit list selection to match the current index
func (m *Model) syncCommitListToIndex() {
	switch m.sourceMode {
	case sourceReflog:
		m.commitList.SelectIndex(m.reflogIndex)
	case sourcePickaxe:
		m.commitList.SelectIndex(m.sourceIndex)
	default:
		if m.singleFileMode {
			m.commitList.SelectIndex(m.fileCommitIndex)
		} else {
			m.commitList.SelectIndex(m.commitIndex)
		}
	}
}

// populateCommitList converts git.Commits to CommitItems and sets them
func (m *Model) populateCommitList(commits []git.Commit) {
	items := make([]CommitItem, len(commits))
	for i, c := range commits {
		items[i] = CommitItem{Hash: c.Hash, Message: c.Message}
	}
	m.commitList.SetItems(items)
}

func (m *Model) updateSourceIndicator() {
	switch m.sourceMode {
	case sourceReflog:
		m.diffView.SetSourceIndicator("REFLOG")
	case sourcePickaxe:
		m.diffView.SetSourceIndicator(fmt.Sprintf("S:\"%s\"", m.pickaxeTerm))
	default:
		m.diffView.SetSourceIndicator("")
	}
}

// navigateNewer moves to a newer commit in the current source
func (m *Model) navigateNewer() tea.Cmd {
	switch m.sourceMode {
	case sourceReflog:
		if m.reflogIndex > 0 {
			m.reflogIndex--
			m.updateReflogDisplay()
			return m.loadContentForCurrentSource()
		}
	case sourcePickaxe:
		if m.sourceIndex > 0 {
			m.sourceIndex--
			m.updateSourceDisplay()
			return m.loadContentForCurrentSource()
		}
	default:
		if m.fileCommitIndex > 0 {
			m.fileCommitIndex--
			m.updateSingleFileModeDisplay()
			return m.loadContentForCurrentSource()
		}
	}
	return nil
}

// navigateOlder moves to an older commit in the current source
func (m *Model) navigateOlder() tea.Cmd {
	switch m.sourceMode {
	case sourceReflog:
		if m.reflogIndex < len(m.reflogEntries)-1 {
			m.reflogIndex++
			m.updateReflogDisplay()
			return m.loadContentForCurrentSource()
		}
	case sourcePickaxe:
		if m.sourceIndex < len(m.sourceCommits)-1 {
			m.sourceIndex++
			m.updateSourceDisplay()
			return m.loadContentForCurrentSource()
		}
	default:
		if m.fileCommitIndex < len(m.fileCommits)-1 {
			m.fileCommitIndex++
			m.updateSingleFileModeDisplay()
			return m.loadContentForCurrentSource()
		}
	}
	return nil
}

// currentCommitForSource returns the commit hash and commit for the current source/index
func (m *Model) currentCommitForSource() (string, bool) {
	switch m.sourceMode {
	case sourceReflog:
		if m.reflogIndex < len(m.reflogEntries) {
			return m.reflogEntries[m.reflogIndex].Hash, true
		}
	case sourcePickaxe:
		if m.sourceIndex < len(m.sourceCommits) {
			return m.sourceCommits[m.sourceIndex].Hash, true
		}
	default:
		if m.fileCommitIndex < len(m.fileCommits) {
			return m.fileCommits[m.fileCommitIndex].Hash, true
		}
	}
	return "", false
}

// loadContentForCurrentSource returns the appropriate loader cmd for the current display+source combo
func (m *Model) loadContentForCurrentSource() tea.Cmd {
	hash, ok := m.currentCommitForSource()
	if !ok || m.currentFile == "" {
		return func() tea.Msg { return diffLoadedMsg{content: ""} }
	}

	file := m.currentFile
	dm := m.displayMode

	return func() tea.Msg {
		return m.loadContentForCommit(file, hash, dm)
	}
}

func (m *Model) loadContentForCommit(file, hash string, dm displayMode) tea.Msg {
	var content string
	var err error

	switch dm {
	case displayBlame:
		content, err = m.gitService.GetBlame(file, hash)
	case displayFull:
		content, err = m.gitService.GetFileContentAtCommit(file, hash)
	case displayContext:
		content, err = m.gitService.GetDiffAtCommitWithContext(file, hash, 10)
	default: // displayDiff
		content, err = m.gitService.GetDiffAtCommit(file, hash)
	}

	if err != nil {
		return diffLoadedMsg{content: fmt.Sprintf("Error: %v", err)}
	}
	if content == "" {
		return diffLoadedMsg{content: "No changes to display"}
	}
	return diffLoadedMsg{content: content}
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * 0.20)
	diffWidth := m.width - sidebarWidth - 4

	if m.showFileTree {
		// Tree mode: single panel on the left, same height as diff
		m.fileTree.SetSize(sidebarWidth, m.height-3)
		m.diffView.SetSize(diffWidth, m.height-3)
	} else {
		// Normal mode: two panels stacked on the left
		// Left column has two bordered panels stacked + help bar:
		// each border = 2 lines (top+bottom), help bar = 1 line,
		// JoinVertical separator = 1 line -> total overhead = 6
		leftContent := m.height - 6
		commitListHeight := leftContent / 2
		fileListHeight := leftContent - commitListHeight

		m.commitList.SetSize(sidebarWidth, commitListHeight)
		m.sidebar.SetSize(sidebarWidth, fileListHeight)
		m.diffView.SetSize(diffWidth, m.height-3)
	}
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

func (m *Model) updateReflogDisplay() {
	if m.reflogIndex < len(m.reflogEntries) {
		entry := m.reflogEntries[m.reflogIndex]
		m.sidebar.SetRevision("REFLOG: " + entry.Hash)
		m.diffView.SetFileInfo(m.currentFile, m.reflogIndex, len(m.reflogEntries), entry.Hash)
	}
}

func (m *Model) updateSourceDisplay() {
	if m.sourceIndex < len(m.sourceCommits) {
		commit := m.sourceCommits[m.sourceIndex]
		var prefix string
		if m.sourceMode == sourcePickaxe {
			prefix = fmt.Sprintf("S:\"%s\": ", m.pickaxeTerm)
		}
		m.sidebar.SetRevision(prefix + commit.Hash)
		m.diffView.SetFileInfo(m.currentFile, m.sourceIndex, len(m.sourceCommits), commit.Hash)
	}
}

func (m *Model) loadFileCommits() tea.Msg {
	commits, _ := m.gitService.GetFileCommits(m.currentFile)
	return fileCommitsLoadedMsg{commits: commits}
}

func (m *Model) loadReflog() tea.Msg {
	entries, _ := m.gitService.GetFileReflog(m.currentFile, 100)
	return reflogLoadedMsg{entries: entries}
}

func (m *Model) loadPickaxeCommits() tea.Msg {
	commits, err := m.gitService.GetPickaxeCommits(m.currentFile, m.pickaxeTerm)
	return sourceCommitsLoadedMsg{commits: commits, err: err}
}

func (m *Model) loadTreeFiles() tea.Msg {
	// Use HEAD for the tree
	paths, err := m.gitService.GetTreeFiles("HEAD")
	if err != nil {
		return treeFilesLoadedMsg{paths: nil}
	}
	return treeFilesLoadedMsg{paths: paths}
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

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	if m.err != nil {
		return "Error: " + m.err.Error()
	}

	var help string
	if m.textInputMode != "" {
		badge := ModeBadgeFile.Render("FILE")
		inputView := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render("Search: ") + m.textInput.View()
		help = badge + " " + inputView
	} else if m.singleFileMode {
		badge := ModeBadgeFile.Render("FILE")
		helpText := HelpStyle.Render("[1/2/3: focus | c: view | r: reflog | s: search | d/u: scroll | n/N: hunks | [/]: history | z: info | q: back]")
		help = badge + " " + helpText
	} else if m.showFileTree {
		badge := ModeBadgeTree.Render("TREE")
		helpText := HelpStyle.Render("[j/k: nav | enter: open | h/l: collapse/expand | t/esc: close | q: quit]")
		help = badge + " " + helpText
	} else {
		badge := ModeBadgeCommits.Render("COMMITS")
		helpText := HelpStyle.Render("[1/2/3: focus | j/k: nav | space: file mode | t: tree | [/]: commits | /: filter | n/N: hunks | z: info | q: quit]")
		help = badge + " " + helpText
	}

	diffRendered := injectBorderLabel(m.diffView.View(), "3", m.focus == focusDiffView)

	var leftColumn string
	if m.showFileTree {
		treeRendered := injectBorderLabel(m.fileTree.View(), "1", m.focus == focusFileTree)
		leftColumn = treeRendered
	} else {
		commitListRendered := injectBorderLabel(m.commitList.View(), "1", m.focus == focusCommitList)
		sidebarRendered := injectBorderLabel(m.sidebar.View(), "2", m.focus == focusFileList)
		leftColumn = lipgloss.JoinVertical(
			lipgloss.Left,
			commitListRendered,
			sidebarRendered,
		)
	}

	main := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn,
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
