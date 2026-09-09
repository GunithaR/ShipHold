package check

import (
	"errors"
	"testing"

	"github.com/GunithaR/ShipHold/internal/domain/policy"
	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

type fakeEvidenceProvider struct {
	evidence readiness.Evidence
}

func (f fakeEvidenceProvider) Collect() (readiness.Evidence, error) {
	return f.evidence, nil
}

type failingEvidenceProvider struct{}

func (f failingEvidenceProvider) Collect() (readiness.Evidence, error) {
	return readiness.Evidence{}, errors.New("git unavailable")
}

func TestServiceRun(t *testing.T) {
	provider := fakeEvidenceProvider{
		evidence: readiness.Evidence{
			CIPassed:         true,
			ImageReference:   "my-app:v1",
			ImageDigest:      "sha256:abc",
			HealthCheckValid: true,
		},
	}
	service := NewService(provider)
	p := policy.Policy{
		RequireCIPass:          true,
		RequireImage:           true,
		RequireImmutableDigest: true,
		RequireHealthCheck:     true,
	}
	result, err := service.Run(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Decision != readiness.DecisionPass {
		t.Fatalf("expected PASS, got %q", result.Decision)
	}
}

func TestServiceRunReturnsProviderError(t *testing.T) {
	service := NewService(failingEvidenceProvider{})

	p := policy.Policy{}

	_, err := service.Run(p)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Error() != "git unavailable" {
		t.Fatalf("expected git unavailable, got %q", err.Error())
	}
}
