package policy

import (
	"testing"

	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

func TestEvaluate(t *testing.T) {
	policy := Policy{
		RequireCIPass:          true,
		RequireImage:           true,
		RequireImmutableDigest: true,
		RequireHealthCheck:     true,
	}

	tests := []struct {
		name     string
		evidence readiness.Evidence
		expected readiness.Decision
	}{
		{
			name: "all requirements pass",
			evidence: readiness.Evidence{
				CIPassed:         true,
				ImageReference:   "my-app:v1",
				ImageDigest:      "sha256:abc",
				HealthCheckValid: true,
			},
			expected: readiness.DecisionPass,
		},
		{
			name: "CI failed",
			evidence: readiness.Evidence{
				CIPassed:         false,
				ImageReference:   "my-app:v1",
				ImageDigest:      "sha256:abc",
				HealthCheckValid: true,
			},
			expected: readiness.DecisionBlock,
		},
		{
			name: "image missing",
			evidence: readiness.Evidence{
				CIPassed:         true,
				ImageDigest:      "sha256:abc",
				HealthCheckValid: true,
			},
			expected: readiness.DecisionBlock,
		},
		{
			name: "digest missing",
			evidence: readiness.Evidence{
				CIPassed:         true,
				ImageReference:   "my-app:v1",
				HealthCheckValid: true,
			},
			expected: readiness.DecisionBlock,
		},
		{
			name: "health check invalid",
			evidence: readiness.Evidence{
				CIPassed:       true,
				ImageReference: "my-app:v1",
				ImageDigest:    "sha256:abc",
			},
			expected: readiness.DecisionBlock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Evaluate(tt.evidence, policy)

			if result.Decision != tt.expected {
				t.Fatalf(
					"expected %q, got %q",
					tt.expected,
					result.Decision,
				)
			}
		})
	}
}

func TestEvaluateReturnReasons(t *testing.T) {
	policy := Policy{
		RequireCIPass:          true,
		RequireImage:           true,
		RequireImmutableDigest: true,
		RequireHealthCheck:     true,
	}

	evidence := readiness.Evidence{}

	result := Evaluate(evidence, policy)

	if result.Decision != readiness.DecisionBlock {
		t.Fatalf("expected BLOCK, got %q", result.Decision)
	}

	if len(result.Reasons) != 4 {
		t.Fatalf("expected 4 reasons, got %d", len(result.Reasons))
	}
}
