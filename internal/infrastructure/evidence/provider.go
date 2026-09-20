package evidence

import (
	"github.com/GunithaR/ShipHold/internal/domain/readiness"
)

type CIProvider interface {
	Check(commitSHA string) (bool, error)
}

type Provider struct {
	git readiness.Provider
	ci CIProvider
}

func NewProvider(git readiness.Provider, ci CIProvider) Provider {
	return Provider{
		git: git,
		ci: ci,
	}
}

func (p Provider) Collect() (readiness.Evidence, error) {
	evidence, err:= p.git.Collect()
	if err != nil {
		return readiness.Evidence{}, err
	}

	ciPassed, err := p.ci.Check(evidence.GitCommitSHA)
	if err != nil {
		return readiness.Evidence{}, err
	}

	evidence.ciPassed

}
