package check

import (
	"github.com/GunithaR/ShipHold/internal/domain/policy"
	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

type Service struct {
	provider readiness.Provider
}

func NewService(provider readiness.Provider) *Service {
	return &Service{
		provider: provider,
	}
}

func (s *Service) Run(p policy.Policy) (policy.EvaluationResult, error) {
	evidence, err := s.provider.Collect()
	if err != nil {
		return policy.EvaluationResult{}, err
	}

	return policy.Evaluate(evidence, p), nil
}
