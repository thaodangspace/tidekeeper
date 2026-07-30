package voyage

import "testing"

func TestStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status Status
		want   bool
	}{
		{status: StatusActive, want: false},
		{status: StatusCompleted, want: true},
		{status: StatusFailed, want: true},
		{status: StatusAbandoned, want: true},
		{status: Status("FUTURE"), want: false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.want {
				t.Errorf("IsTerminal() = %t, want %t", got, tt.want)
			}
		})
	}
}
