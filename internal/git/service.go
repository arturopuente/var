package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
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
	return s.GetDiffWithContext(filePath, 3) // default context
}

// GetDiffWithContext returns the diff with specified lines of context
func (s *Service) GetDiffWithContext(filePath string, context int) (string, error) {
	cmd := exec.Command("git", "diff", "--color=always", fmt.Sprintf("-U%d", context), "--", filePath)
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

// GetFileContent returns the full content of a file in the working copy with line numbers
func (s *Service) GetFileContent(filePath string) (string, error) {
	fullPath := filepath.Join(s.repoPath, filePath)
	cmd := exec.Command("cat", "-n", fullPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
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
	return s.GetDiffAtCommitWithContext(filePath, commitHash, 3)
}

// GetDiffAtCommitWithContext returns the diff with specified lines of context
func (s *Service) GetDiffAtCommitWithContext(filePath, commitHash string, context int) (string, error) {
	cmd := exec.Command("git", "show", "--color=always", fmt.Sprintf("-U%d", context), commitHash, "--", filePath)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetFileContentAtCommit returns the full content of a file at a specific commit
func (s *Service) GetFileContentAtCommit(filePath, commitHash string) (string, error) {
	cmd := exec.Command("git", "show", fmt.Sprintf("%s:%s", commitHash, filePath))
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		// File might be deleted in this commit, try parent commit
		cmd = exec.Command("git", "show", fmt.Sprintf("%s^:%s", commitHash, filePath))
		cmd.Dir = s.repoPath
		output, err = cmd.Output()
		if err != nil {
			return "", err
		}
	}
	// Add line numbers manually
	lines := strings.Split(string(output), "\n")
	var result strings.Builder
	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			continue
		}
		result.WriteString(fmt.Sprintf("%6d\t%s\n", i+1, line))
	}
	return result.String(), nil
}

// GetRecentCommits returns recent commits for the repository
func (s *Service) GetRecentCommits(limit int) ([]Commit, error) {
	cmd := exec.Command("git", "log", "--oneline", "-n", fmt.Sprintf("%d", limit))
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

// GetFilesInCommit returns files changed in a specific commit
func (s *Service) GetFilesInCommit(commitHash string) ([]FileStatus, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-status", "-r", commitHash)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []FileStatus
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		files = append(files, FileStatus{
			Status: parts[0],
			Path:   parts[1],
		})
	}
	return files, nil
}

// FileStats holds additions and deletions for a file in a commit
type FileStats struct {
	Additions int
	Deletions int
}

// GetNumstatForCommit returns per-file addition/deletion counts for a commit
func (s *Service) GetNumstatForCommit(commitHash string) (map[string]FileStats, error) {
	cmd := exec.Command("git", "diff-tree", "--numstat", "--no-commit-id", "-r", commitHash)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]FileStats)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		// Binary files show "-" for additions/deletions
		adds, _ := strconv.Atoi(parts[0])
		dels, _ := strconv.Atoi(parts[1])
		path := parts[2]
		stats[path] = FileStats{Additions: adds, Deletions: dels}
	}
	return stats, nil
}

// GetFileReflog returns reflog entries where the given file was changed
func (s *Service) GetFileReflog(filePath string, limit int) ([]Commit, error) {
	cmd := exec.Command("git", "log", "-g", "--oneline", "-n", fmt.Sprintf("%d", limit), "--", filePath)
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

// GetBlame returns blame output for a file at a specific commit
func (s *Service) GetBlame(filePath, commitHash string) (string, error) {
	cmd := exec.Command("git", "--no-pager", "blame", commitHash, "--", filePath)
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// GetPickaxeCommits returns commits where the given search term was added or removed
func (s *Service) GetPickaxeCommits(filePath, searchTerm string) ([]Commit, error) {
	cmd := exec.Command("git", "log", "--oneline", "-S", searchTerm, "--", filePath)
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

// GetFunctionLogCommits returns commits that modified a specific function
func (s *Service) GetFunctionLogCommits(filePath, funcName string) ([]Commit, error) {
	cmd := exec.Command("git", "--no-pager", "log", "--oneline", fmt.Sprintf("-L:%s:%s", funcName, filePath))
	cmd.Dir = s.repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// git log -L output interleaves commit lines with diff content
	// Commit lines from --oneline look like: "abc1234 message"
	// Diff lines start with diff/---/+++/@@/+/-/space or are empty
	var commits []Commit
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip diff content lines
		if strings.HasPrefix(line, "diff ") || strings.HasPrefix(line, "---") ||
			strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "@@") ||
			strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") ||
			strings.HasPrefix(line, " ") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		// Deduplicate — git log -L can repeat commit hashes
		if len(commits) > 0 && commits[len(commits)-1].Hash == parts[0] {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Message: parts[1],
		})
	}
	return commits, nil
}

// GetFunctionDiff returns the diff of a specific function at a specific commit
func (s *Service) GetFunctionDiff(filePath, funcName, commitHash string) (string, error) {
	cmd := exec.Command("git", "--no-pager", "log", "--color=always", "-1", fmt.Sprintf("-L:%s:%s", funcName, filePath), commitHash)
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
