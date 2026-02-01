package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

type Service struct {
	repoPath string
}

type FileStatus struct {
	Path   string
	Status string // M, A, D, ??, etc.
}

type Commit struct {
	Hash    string
	Message string
}

func NewService(repoPath string) *Service {
	return &Service{repoPath: repoPath}
}

// GetModifiedFiles returns a list of modified, added, or untracked files
func (s *Service) GetModifiedFiles() ([]FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []FileStatus
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		status := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])
		// Handle renamed files (e.g., "R  old -> new")
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[1]
		}
		files = append(files, FileStatus{
			Path:   path,
			Status: status,
		})
	}
	return files, nil
}

// GetDiff returns the diff for a file in the working copy
func (s *Service) GetDiff(filePath string) (string, error) {
	cmd := exec.Command("git", "diff", "--color=always", "--", filePath)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		// If file is untracked, show the whole file as added
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
			return string(output), nil
		}
		// Check if file is untracked
		return s.getUntrackedDiff(filePath)
	}
	return string(output), nil
}

// getUntrackedDiff returns a diff-like output for untracked files
func (s *Service) getUntrackedDiff(filePath string) (string, error) {
	fullPath := filepath.Join(s.repoPath, filePath)
	cmd := exec.Command("git", "diff", "--color=always", "--no-index", "/dev/null", fullPath)
	cmd.Dir = s.repoPath
	output, _ := cmd.Output() // This will return exit code 1 for differences
	return string(output), nil
}

// GetFileCommits returns the commit history for a specific file
func (s *Service) GetFileCommits(filePath string) ([]Commit, error) {
	cmd := exec.Command("git", "log", "--oneline", "--follow", "--", filePath)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []Commit
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Message: parts[1],
		})
	}
	return commits, nil
}

// GetDiffAtCommit returns the diff for a file at a specific commit
func (s *Service) GetDiffAtCommit(filePath, commitHash string) (string, error) {
	cmd := exec.Command("git", "show", "--color=always", commitHash, "--", filePath)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// IsGitRepository checks if the path is a git repository
func IsGitRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}
