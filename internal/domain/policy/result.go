package policy

import "github.com/GunithaR/ShipHold/internal/domain/readiness"

type EvaluationResult struct {
	Decision readiness.Decision
	Reasons  []string
}
