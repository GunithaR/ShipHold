package policy

import "testing"

func TestPolicy(t *testing.T) {
	p := Policy{
		RequireCIPass:          true,
		RequireImage:           true,
		RequireImmutableDigest: true,
		RequireHealthCheck:     true,
	}

	if !p.RequireCIPass {
		t.Fatal("expected CI pass to be required")
	}

	if !p.RequireImage {
		t.Fatal("expected image to be required")
	}
}
