package check

import (
	"testing"

	"github.com/GunithaR/ShipHold/internal/domain/policy"
	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

type fakeEvidenceProvider struct {
	evidence readiness.Evidence
}

func (f fakeEvidenceProvider) Collect() readiness.Evidence {
	return f.evidence
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
	result := service.Run(p)
	if result.Decision != readiness.DecisionPass {
		t.Fatalf("expected PASS, got %q", result.Decision)
	}
}
