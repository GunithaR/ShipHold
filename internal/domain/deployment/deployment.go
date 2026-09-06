package deployment

import "fmt"

type Status string

const (
	StatusRequested      Status = "REQUESTED"
	StatusValidating     Status = "VALIDATING"
	StatusReady          Status = "READY"
	StatusBlocked        Status = "BLOCKED"
	StatusDeploying      Status = "DEPLOYING"
	StatusVerifying      Status = "VERIFYING"
	StatusSuccess        Status = "SUCCESS"
	StatusFailed         Status = "FAILED"
	StatusRollingBack    Status = "ROLLING_BACK"
	StatusRollbackFailed Status = "ROLLBACK_FAILED"
	StatusInterrupted    Status = "INTERRUPTED"
)

type Deployment struct {
	ID           string
	Application  string
	CommitSHA    string
	Image        string
	Status       Status
	RuntimeState RuntimeState
}

func New(id string, application string, commitSHA string, image string) Deployment {
	return Deployment{
		ID:           id,
		Application:  application,
		CommitSHA:    commitSHA,
		Image:        image,
		Status:       StatusRequested,
		RuntimeState: RuntimeUnknown,
	}
}

func (d *Deployment) TransitionTo(status Status) error {
	if !isValidTransition(d.Status, status) {
		return fmt.Errorf(
			"invalid deployment transition: %s -> %s",
			d.Status,
			status,
		)
	}

	d.Status = status
	return nil
}

func isValidTransition(from Status, to Status) bool {
	switch from {
	case StatusRequested:
		return to == StatusValidating

	case StatusValidating:
		return to == StatusReady || to == StatusBlocked

	case StatusReady:
		return to == StatusDeploying

	case StatusDeploying:
		return to == StatusVerifying

	case StatusVerifying:
		return to == StatusSuccess || to == StatusFailed

	case StatusFailed:
		return to == StatusRollingBack

	case StatusRollingBack:
		return to == StatusSuccess || to == StatusRollbackFailed
	}

	return false
}
