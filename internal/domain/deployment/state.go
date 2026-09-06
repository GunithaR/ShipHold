package deployment

type RuntimeState string

const (
	RuntimeUnknown RuntimeState = "UNKNOWN"
	RuntimeActive  RuntimeState = "ACTIVE"
	RuntimeStopped RuntimeState = "STOPPED"
	RuntimeFailed  RuntimeState = "FAILED"
)
