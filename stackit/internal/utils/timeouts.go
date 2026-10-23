package utils

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TimeoutHint names the configured timeout when the deadline of ctx is what ended a wait, e.g.
// TimeoutHint(ctx, "create", createTimeout). Wait handlers report a timeout, a terminal error state and a failing poll
// through the same error, so the hint is empty unless the deadline was exceeded; otherwise it would point at the wrong
// cause. The hint starts with a newline so it can be appended to a diagnostic detail.
func TimeoutHint(ctx context.Context, operation string, timeout time.Duration) string {
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ""
	}
	return fmt.Sprintf("\nThe wait gave up after the configured `timeouts.%s` of %s; raise it if the operation regularly needs longer.", operation, timeout)
}
