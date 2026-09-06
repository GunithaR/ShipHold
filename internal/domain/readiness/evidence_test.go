package readiness

import "testing"

func TestEvidence(t *testing.T) {
	evidence := Evidence{
		GitCommitSHA:     "abc123",
		Branch:           "main",
		CIPassed:         true,
		ImageReference:   "my-app:v1",
		ImageDigest:      "sha256:abc",
		HealthCheckValid: true,
	}

	if !evidence.CIPassed {
		t.Fatal("expected CI to have passed.")
	}

	if evidence.GitCommitSHA != "abc123" {
		t.Fatalf("unexpected commit SHA: %s", evidence.GitCommitSHA)
	}
}
