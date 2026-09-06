package policy

import "github.com/GunithaR/ShipHold/internal/domain/readiness"

func Evaluate(
	evidence readiness.Evidence,
	policy Policy,
) EvaluationResult {

	var reasons []string

	if policy.RequireCIPass && !evidence.CIPassed {
		reasons = append(reasons, "CI has not passed")
	}

	if policy.RequireImage && evidence.ImageReference == "" {
		reasons = append(reasons, "image is missing")
	}

	if policy.RequireImmutableDigest && evidence.ImageDigest == "" {
		reasons = append(reasons, "immutable image digest is missing")
	}

	if policy.RequireHealthCheck && !evidence.HealthCheckValid {
		reasons = append(reasons, "health check is invalid")
	}

	if len(reasons) > 0 {
		return EvaluationResult{
			Decision: readiness.DecisionBlock,
			Reasons: reasons,
		}
	}

	return EvaluationResult{
		Decision: readiness.DecisionPass,
	}
}
