package delta

import (
	"os/exec"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// Render returns the diff content - we skip delta to avoid theme conflicts
// Git's native --color=always output adapts to terminal themes properly
func (s *Service) Render(diffContent string, width int) (string, error) {
	// Just return the raw git diff - it already has colors from --color=always
	// and those colors work with both light and dark terminal themes
	return diffContent, nil
}

// IsAvailable checks if delta CLI is installed
func IsAvailable() bool {
	_, err := exec.LookPath("delta")
	return err == nil
}
