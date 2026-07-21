package utils

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

// RetryConfig defines the parameters for retrying an operation.
type RetryConfig struct {
	Attempts int

	// Delay is the default wait duration between attempts.
	// Used if Backoff is not provided. Default is 100ms
	Delay time.Duration

	// Backoff allows customizing wait duration per attempt.
	// "attempt" starts at 1 (representing the delay after the 1st failed try).
	Backoff func(attempt int) time.Duration

	// RetryStatusCodes defines a list with HTTP Status codes on which the request should be retried
	RetryStatusCodes []int
}

// RetryRequest executes fn up to config.Attempts times until it succeeds or context is canceled.
func RetryRequest[T any](ctx context.Context, fn func() (*T, error), config RetryConfig) (*T, error) {
	if fn == nil {
		return nil, errors.New("retry function fn cannot be nil")
	}

	if config.Attempts <= 0 {
		config.Attempts = 1
	}

	if config.Delay <= 0 && config.Backoff == nil {
		config.Delay = 100 * time.Millisecond
	}

	retryableSet := make(map[int]bool, len(config.RetryStatusCodes))
	for _, code := range config.RetryStatusCodes {
		retryableSet[code] = true
	}

	var lastErr error

	for attempt := 1; attempt <= config.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		result, err := fn()
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Extract status code and verify if it's in the allowed list
		if oapiErr, ok := errors.AsType[*oapierror.GenericOpenAPIError](err); ok {
			if !retryableSet[oapiErr.StatusCode] {
				return nil, err
			}
		}

		// Don't wait after the last attempt
		if attempt == config.Attempts {
			break
		}

		// Determine wait duration for the next attempt
		waitDuration := config.Delay
		if config.Backoff != nil {
			waitDuration = config.Backoff(attempt)
		}

		// Wait for delay duration or abort if context is canceled
		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, fmt.Errorf("retry limit reached (%d attempts): %w", config.Attempts, lastErr)
}

func RetryRequestWithoutResponse(ctx context.Context, fn func() error, config RetryConfig) error {
	if fn == nil {
		return errors.New("retry function fn cannot be nil")
	}
	wrapper := func() (*struct{}, error) {
		err := fn()
		return &struct{}{}, err
	}
	_, err := RetryRequest(ctx, wrapper, config)
	return err
}
