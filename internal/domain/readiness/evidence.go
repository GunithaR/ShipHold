package readiness

type Evidence struct {
	GitCommitSHA     string
	Branch           string
	CIPassed         bool
	ImageReference   string
	ImageDigest      string
	HealthCheckValid bool
}
