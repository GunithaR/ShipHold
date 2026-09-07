package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

type Provider struct{}

func (p Provider) Collect() (readiness.Evidence, error) {
	if err := ensureGitRepository(); err != nil {
		return readiness.Evidence{}, err
	}

	commitSHA, err := runGitCommand("rev-parse", "HEAD")
	if err != nil {
		return readiness.Evidence{}, fmt.Errorf("get current commit: %w", err)
	}

	branch, err := runGitCommand("branch", "--show-current")
	if err != nil {
		return readiness.Evidence{}, fmt.Errorf("get current branch: %w", err)
	}

	return readiness.Evidence{
		GitCommitSHA: commitSHA,
		Branch:       branch,
	}, nil
}

func ensureGitRepository() error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("not a Git repository")
	}

	if strings.TrimSpace(string(output)) != "true" {
		return fmt.Errorf("not a Git repository")
	}

	return nil
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
