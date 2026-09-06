package deployment

import "fmt"

type Deployment struct {
	ID          string
	Application string
	CommitSHA   string
	Image       string
}

func New(id string, application string, commitSHA string, image string) Deployment {
	return Deployment{
		ID:          id,
		Application: application,
		CommitSHA:   commitSHA,
		Image:       image,
	}
}
