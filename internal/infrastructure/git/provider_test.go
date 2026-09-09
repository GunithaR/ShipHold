package git

import (
	"os"
	"testing"
)

func TestProviderCollect(t *testing.T) {
	provider := Provider{}

	evidence, err := provider.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if evidence.GitCommitSHA == "" {
		t.Fatalf("expected commit SHA, got empty string")
	}

	if evidence.Branch == "" {
		t.Fatal("expected branch, got empty string")
	}
}

func TestProviderCollectOutsideGitRepository(t *testing.T) {
	tempDir := t.TempDir()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	provider := Provider{}

	_, err = provider.Collect()
	if err == nil {
		t.Fatal("expected error outside Git repository")
	}
}

func TestProviderCollectReturnsActualGitEvidence(t *testing.T) {
	provider := Provider{}

	evidence, err := provider.Collect()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedCommit, err := runGitCommand("rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("get expected commit: %v", err)
	}

	expectedBranch, err := runGitCommand("branch", "--show-current")
	if err != nil {
		t.Fatalf("get expected branch: %v", err)
	}

	if evidence.GitCommitSHA != expectedCommit {
		t.Fatalf(
			"expected commit %q, got %q",
			expectedCommit,
			evidence.GitCommitSHA,
		)
	}

	if evidence.Branch != expectedBranch {
		t.Fatalf(
			"expected branch %q, got %q",
			expectedBranch,
			evidence.Branch,
		)
	}
}
