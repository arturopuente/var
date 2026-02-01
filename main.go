package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"var/internal/delta"
	"var/internal/git"
	"var/internal/ui"
)

func main() {
	// Parse optional path argument
	repoPath := "."
	if len(os.Args) > 1 {
		repoPath = os.Args[1]
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid path: %v\n", err)
		os.Exit(1)
	}

	// Validate it's a directory
	info, err := os.Stat(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", absPath)
		os.Exit(1)
	}

	// Validate it's a git repository
	if !git.IsGitRepository(absPath) {
		fmt.Fprintf(os.Stderr, "Error: %s is not a git repository\n", absPath)
		os.Exit(1)
	}

	// Check if delta is available
	if !delta.IsAvailable() {
		fmt.Fprintf(os.Stderr, "Warning: delta is not installed. Diffs will be shown without syntax highlighting.\n")
	}

	// Initialize services
	gitService := git.NewService(absPath)
	deltaService := delta.NewService()

	// Create and run the program
	model := ui.NewModel(gitService, deltaService)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
