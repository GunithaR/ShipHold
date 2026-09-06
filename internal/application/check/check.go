package check

import (
	"github.com/GunithaR/ShipHold/internal/domain/policy"
	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

type EvidenceProvider interface {
	Collect() readiness.Evidence
}

type Service struct {
	provider EvidenceProvider
}

func NewService(provider EvidenceProvider) *Service {
	return &Service{
		provider: provider,
	}
}

func (s *Service) Run(p policy.Policy) policy.EvaluationResult {
	evidence := s.provider.Collect()

	return policy.Evaluate(evidence, p)
}
