package deployment

import "testing"

func TestNewDeployment(t *testing.T) {
	tests := []struct {
		name    string
		from    Status
		to      Status
		wantErr bool
	}{
		{
			name:    "requested to validating",
			from:    StatusRequested,
			to:      StatusValidating,
			wantErr: false,
		},
		{
			name:    "validating to ready",
			from:    StatusValidating,
			to:      StatusReady,
			wantErr: false,
		},
		{
			name:    "requested to success is invalid",
			from:    StatusRequested,
			to:      StatusSuccess,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Deployment{Status: tt.from}

			err := d.TransitionTo(tt.to)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Unexpected error state %v", err)
			}
		})
	}
}

func TestNewDeploymentRuntimeState(t *testing.T) {
	d := New(
		"dep-001",
		"my-app",
		"abc123",
		"my-app:v1",
	)

	if d.RuntimeState != RuntimeUnknown {
		t.Fatalf(
			"expected runtime state %q, got %q",
			RuntimeUnknown,
			d.RuntimeState,
		)
	}
}
