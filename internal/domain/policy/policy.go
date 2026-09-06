package policy

type Policy struct {
	RequireCIPass          bool `yaml:"require_ci_pass"`
	RequireImage           bool `yaml:"require_image"`
	RequireImmutableDigest bool `yaml:"require_immutable_digest"`
	RequireHealthCheck     bool `yaml:"require_healthcheck"`
}
