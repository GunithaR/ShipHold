package git

import "github.com/GunithaR/ShipHold/internal/domain/readiness"

type Provider struct{}

func (p Provider) Collect() readiness.Evidence {
	return readiness.Evidence{
		GitCommitSHA:     "abc123",
		Branch:           "main",
		CIPassed:         true,
		ImageReference:   "my-app:v1",
		ImageDigest:      "sha256:abc123",
		HealthCheckValid: true,
	}
}
