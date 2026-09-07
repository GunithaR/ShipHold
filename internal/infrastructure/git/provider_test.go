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
