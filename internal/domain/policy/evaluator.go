package policy

import "github.com/GunithaR/ShipHold/internal/domain/readiness"

func Evaluate(
	evidence readiness.Evidence,
	policy Policy,
) readiness.Decision {

	if policy.RequireCIPass && !evidence.CIPassed {
		return readiness.DecisionBlock
	}

	if policy.RequireImage && evidence.ImageReference == "" {
		return readiness.DecisionBlock
	}

	if policy.RequireImmutableDigest && evidence.ImageDigest == "" {
		return readiness.DecisionBlock
	}

	if policy.RequireHealthCheck && !evidence.HealthCheckValid {
		return readiness.DecisionBlock
	}

	return readiness.DecisionPass
}
