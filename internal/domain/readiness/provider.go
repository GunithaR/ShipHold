package readiness

type Provider interface {
	Collect() (Evidence, error)
}
