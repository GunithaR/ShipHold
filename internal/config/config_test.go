package config

import (
	"testing"

	"github.com/GunithaR/ShipHold/internal/domain/policy"
)

func TestLoad(t *testing.T) {
	config, err := Load("../../examples/policy.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := policy.Policy{
		RequireCIPass:          true,
		RequireImage:           true,
		RequireImmutableDigest: true,
		RequireHealthCheck:     true,
	}

	if config.Policy != expected {
		t.Fatalf("expected %+v, got %+v", expected, config.Policy)
	}
}
