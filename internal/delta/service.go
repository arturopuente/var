package delta

import (
	"fmt"
	"os/exec"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// Render pipes diff content through delta for syntax highlighting
func (s *Service) Render(diffContent string, width int) (string, error) {
	if diffContent == "" {
		return "", nil
	}

	cmd := exec.Command("delta",
		"--line-numbers",
		"--paging=never",
		"--width", fmt.Sprintf("%d", width),
	)
	cmd.Stdin = strings.NewReader(diffContent)

	output, err := cmd.Output()
	if err != nil {
		// If delta is not available, return the raw diff
		if _, ok := err.(*exec.ExitError); !ok {
			return diffContent, nil
		}
		return "", err
	}

	return string(output), nil
}

// IsAvailable checks if delta CLI is installed
func IsAvailable() bool {
	_, err := exec.LookPath("delta")
	return err == nil
}
