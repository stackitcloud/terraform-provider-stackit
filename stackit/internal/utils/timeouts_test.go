package utils

import (
	"context"
	"testing"
	"time"
)

func TestTimeoutHint(t *testing.T) {
	deadlineExceeded, cancelDeadlineExceeded := context.WithDeadline(context.Background(), time.Now().Add(-time.Minute))
	defer cancelDeadlineExceeded()
	deadlineAhead, cancelDeadlineAhead := context.WithTimeout(context.Background(), time.Hour)
	defer cancelDeadlineAhead()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		description string
		ctx         context.Context
		expected    string
	}{
		{
			"deadline exceeded",
			deadlineExceeded,
			"\nThe wait gave up after the configured `timeouts.create` of 20m0s; raise it if the operation regularly needs longer.",
		},
		{
			"wait failed before the deadline",
			deadlineAhead,
			"",
		},
		{
			"canceled",
			canceled,
			"",
		},
		{
			"no deadline",
			context.Background(),
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			output := TimeoutHint(tt.ctx, "create", 20*time.Minute)
			if output != tt.expected {
				t.Fatalf("TimeoutHint() = %q, want %q", output, tt.expected)
			}
		})
	}
}
