package check

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

func (s *Service) Run() string {
	return s.provider.Collect()
}
