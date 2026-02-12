package ui

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeNode represents a file or directory in the tree
type TreeNode struct {
	Path     string
	Name     string
	Depth    int
	IsDir    bool
	Expanded bool // only meaningful for directories
}

// TreeItem wraps TreeNode for use with bubbles/list
type TreeItem struct {
	Node TreeNode
}

func (i TreeItem) FilterValue() string { return i.Node.Path }

type treeItemDelegate struct{}

func (d treeItemDelegate) Height() int                             { return 1 }
func (d treeItemDelegate) Spacing() int                            { return 0 }
func (d treeItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d treeItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(TreeItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	node := i.Node

	indent := strings.Repeat("  ", node.Depth)
	var icon string
	if node.IsDir {
		if node.Expanded {
			icon = "v "
		} else {
			icon = "> "
		}
	} else {
		icon = "  "
	}

	label := indent + icon + node.Name

	width := m.Width()
	if len(label) > width-2 {
		label = label[:width-2]
	}

	if isSelected {
		bg := lipgloss.Color("#0066cc")
		fg := lipgloss.Color("#ffffff")
		style := lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true)
		fmt.Fprint(w, lipgloss.NewStyle().Width(width).Background(bg).Render(style.Render(label)))
	} else if node.IsDir {
		dirStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true)
		fmt.Fprint(w, dirStyle.Render(label))
	} else {
		fmt.Fprint(w, label)
	}
}

// FileTree displays a full repository file tree with expand/collapse
type FileTree struct {
	list      list.Model
	width     int
	height    int
	isFocused bool
	allNodes  []TreeNode // full sorted tree (dirs + files)
	expanded  map[string]bool
}

func NewFileTree(width, height int) FileTree {
	l := list.New([]list.Item{}, treeItemDelegate{}, width, height)
	l.Title = "Tree"
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	return FileTree{
		list:     l,
		width:    width,
		height:   height,
		expanded: make(map[string]bool),
	}
}

func (ft *FileTree) SetSize(width, height int) {
	ft.width = width
	ft.height = height
	ft.list.SetSize(width, height)
}

func (ft *FileTree) SetFocused(focused bool) {
	ft.isFocused = focused
}

// SetFiles builds the tree from a flat list of file paths
func (ft *FileTree) SetFiles(paths []string) {
	ft.allNodes = buildTreeNodes(paths)
	ft.expanded = make(map[string]bool)
	// Expand root-level directories by default
	for _, node := range ft.allNodes {
		if node.IsDir && node.Depth == 0 {
			ft.expanded[node.Path] = true
		}
	}
	ft.rebuildVisibleItems()
}

// SelectedPath returns the path of the currently selected item
func (ft *FileTree) SelectedPath() string {
	item := ft.list.SelectedItem()
	if item == nil {
		return ""
	}
	return item.(TreeItem).Node.Path
}

// IsSelectedDir returns true if the selected item is a directory
func (ft *FileTree) IsSelectedDir() bool {
	item := ft.list.SelectedItem()
	if item == nil {
		return false
	}
	return item.(TreeItem).Node.IsDir
}

func (ft *FileTree) toggleExpand(dirPath string) {
	ft.expanded[dirPath] = !ft.expanded[dirPath]
	ft.rebuildVisibleItems()
}

func (ft *FileTree) collapseSelected() {
	item := ft.list.SelectedItem()
	if item == nil {
		return
	}
	node := item.(TreeItem).Node
	if node.IsDir && ft.expanded[node.Path] {
		ft.expanded[node.Path] = false
		ft.rebuildVisibleItems()
	} else {
		// Collapse parent directory and move cursor there
		parent := path.Dir(node.Path)
		if parent != "." && parent != "" {
			ft.expanded[parent] = false
			ft.rebuildVisibleItems()
			// Move selection to the parent dir
			for idx, li := range ft.list.Items() {
				if t, ok := li.(TreeItem); ok && t.Node.Path == parent {
					ft.list.Select(idx)
					break
				}
			}
		}
	}
}

func (ft *FileTree) rebuildVisibleItems() {
	selectedPath := ft.SelectedPath()
	var items []list.Item
	newSelectedIdx := 0
	for _, node := range ft.allNodes {
		if !ft.isVisible(node) {
			continue
		}
		n := node
		if n.IsDir {
			n.Expanded = ft.expanded[n.Path]
		}
		if n.Path == selectedPath {
			newSelectedIdx = len(items)
		}
		items = append(items, TreeItem{Node: n})
	}
	ft.list.SetItems(items)
	ft.list.Select(newSelectedIdx)
}

func (ft *FileTree) isVisible(node TreeNode) bool {
	if node.Depth == 0 {
		return true
	}
	// Check that all ancestor directories are expanded
	parts := strings.Split(node.Path, "/")
	for i := 1; i < len(parts); i++ {
		ancestor := strings.Join(parts[:i], "/")
		if !ft.expanded[ancestor] {
			return false
		}
	}
	return true
}

func (ft *FileTree) Update(msg tea.Msg) (FileTree, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", " ", "l":
			if ft.IsSelectedDir() {
				ft.toggleExpand(ft.SelectedPath())
				return *ft, nil
			}
			// File selection is handled by model.go
			return *ft, nil
		case "h":
			ft.collapseSelected()
			return *ft, nil
		}
	}

	var cmd tea.Cmd
	ft.list, cmd = ft.list.Update(msg)
	return *ft, cmd
}

func (ft *FileTree) View() string {
	style := lipgloss.NewStyle().
		Width(ft.width).
		Height(ft.height).
		BorderStyle(lipgloss.RoundedBorder())

	if ft.isFocused {
		style = style.BorderForeground(lipgloss.Color("2"))
	}

	return style.Render(ft.list.View())
}

// buildTreeNodes creates a sorted tree structure from flat file paths
func buildTreeNodes(paths []string) []TreeNode {
	sort.Strings(paths)

	dirSet := make(map[string]bool)
	for _, p := range paths {
		parts := strings.Split(p, "/")
		for i := 1; i < len(parts); i++ {
			dirSet[strings.Join(parts[:i], "/")] = true
		}
	}

	// Collect all entries
	type entry struct {
		path  string
		isDir bool
	}
	var entries []entry
	for d := range dirSet {
		entries = append(entries, entry{path: d, isDir: true})
	}
	for _, p := range paths {
		entries = append(entries, entry{path: p, isDir: false})
	}

	// Sort in tree-walk order: compare component by component,
	// dirs before files at each level, then alphabetical
	sort.Slice(entries, func(i, j int) bool {
		aParts := strings.Split(entries[i].path, "/")
		bParts := strings.Split(entries[j].path, "/")

		for k := 0; k < len(aParts) && k < len(bParts); k++ {
			if aParts[k] != bParts[k] {
				// At this level, check if each side is a dir
				// (either an intermediate component or a dir entry)
				aIsDir := k < len(aParts)-1 || entries[i].isDir
				bIsDir := k < len(bParts)-1 || entries[j].isDir
				if aIsDir != bIsDir {
					return aIsDir
				}
				return aParts[k] < bParts[k]
			}
		}
		// One is a prefix of the other -- parent comes first
		return len(aParts) < len(bParts)
	})

	var nodes []TreeNode
	for _, e := range entries {
		depth := strings.Count(e.path, "/")
		nodes = append(nodes, TreeNode{
			Path:  e.path,
			Name:  path.Base(e.path),
			Depth: depth,
			IsDir: e.isDir,
		})
	}

	return nodes
}
