package check

import "testing"

type fakeEvidenceProvider struct {
	evidence string
}

func (f fakeEvidenceProvider) Collect() string {
	return f.evidence
}

func TestServiceRun(t *testing.T) {
	tests := []struct {
		name     string
		evidence string
		expected string
	}{
		{
			name:     "returns collected evidence",
			evidence: "fake evidence",
			expected: "fake evidence",
		},
		{
			name:     "returns different evidence",
			evidence: "another result",
			expected: "another result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := fakeEvidenceProvider{
				evidence: tt.evidence,
			}

			service := NewService(provider)

			result := service.Run()

			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}

}
