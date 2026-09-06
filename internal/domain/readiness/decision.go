package readiness

type Decision string

const (
	DecisionPass  Decision = "PASS"
	DecisionWarn  Decision = "WARN"
	DecisionBlock Decision = "BLOCK"
)
