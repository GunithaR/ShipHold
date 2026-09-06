package check

import "fmt"

type EvidenceProvider interface {
	Collect() string
}

type Service struct {
	provider EvidenceProvider
}

func NewService(provider EvidenceProvider) *Service {
	return &Service{
		provider: provider,
	}
}

func (s *Service) Run() {
	evidence := s.provider.Collect()

	fmt.Println("Evidence:", evidence)
}
