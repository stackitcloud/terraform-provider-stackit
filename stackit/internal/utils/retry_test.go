package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"
)

func TestRetryRequest(t *testing.T) {
	t.Parallel()

	makeOapiErr := func(statusCode int) error {
		return &oapierror.GenericOpenAPIError{
			StatusCode: statusCode,
		}
	}

	tests := []struct {
		name          string
		ctx           func() (context.Context, context.CancelFunc)
		returns       []error
		config        RetryConfig
		expectedCalls int
		wantErr       bool
		wantResult    string
	}{
		{
			name: "succeeds on first attempt",
			returns: []error{
				nil,
			},
			config: RetryConfig{
				Attempts: 3,
			},
			expectedCalls: 1,
			wantErr:       false,
			wantResult:    "success",
		},
		{
			name: "succeeds after retrying matched status code",
			returns: []error{
				makeOapiErr(429),
				makeOapiErr(429),
				nil,
			},
			config: RetryConfig{
				Attempts:         3,
				Delay:            1 * time.Millisecond,
				RetryStatusCodes: []int{429},
			},
			expectedCalls: 3,
			wantErr:       false,
			wantResult:    "success",
		},
		{
			name: "fails immediately on non-matching status code",
			returns: []error{
				makeOapiErr(400),
			},
			config: RetryConfig{
				Attempts:         5,
				RetryStatusCodes: []int{429, 500},
			},
			expectedCalls: 1,
			wantErr:       true,
		},
		{
			name: "fails after exceeding max attempts",
			returns: []error{
				makeOapiErr(500),
				makeOapiErr(500),
				makeOapiErr(500),
			},
			config: RetryConfig{
				Attempts:         3,
				Delay:            1 * time.Millisecond,
				RetryStatusCodes: []int{500},
			},
			expectedCalls: 3,
			wantErr:       true,
		},
		{
			name: "aborts when context is canceled mid-retry",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 10*time.Millisecond)
			},
			returns: []error{
				makeOapiErr(503),
				makeOapiErr(503),
			},
			config: RetryConfig{
				Attempts:         5,
				Delay:            100 * time.Millisecond,
				RetryStatusCodes: []int{503},
			},
			expectedCalls: 1,
			wantErr:       true,
		},
		{
			name:          "fails if fn is nil",
			returns:       nil,
			config:        RetryConfig{Attempts: 3},
			expectedCalls: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}

			calls := 0
			var fn func() (*string, error)

			if tt.returns != nil {
				fn = func() (*string, error) {
					if calls >= len(tt.returns) {
						t.Fatalf("fn called more times (%d) than mocked return values (%d)", calls+1, len(tt.returns))
					}
					err := tt.returns[calls]
					calls++
					if err != nil {
						return nil, err
					}
					return new("success"), nil
				}
			}

			res, err := RetryRequest(ctx, fn, tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("RetryRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if calls != tt.expectedCalls {
				t.Errorf("RetryRequest() fn called %d times, expected %d", calls, tt.expectedCalls)
			}
			if !tt.wantErr && (res == nil || *res != tt.wantResult) {
				t.Errorf("RetryRequest() result = %v, want %v", res, tt.wantResult)
			}
		})
	}
}

func TestRetryRequestWithoutResponse(t *testing.T) {
	t.Parallel()

	// Helper to build OpenAPI errors for tests
	makeOapiErr := func(statusCode int) error {
		return &oapierror.GenericOpenAPIError{
			StatusCode: statusCode,
		}
	}

	tests := []struct {
		name          string
		ctx           func() (context.Context, context.CancelFunc)
		returns       []error // Errors to return sequentially on each invocation
		config        RetryConfig
		expectedCalls int
		wantErr       bool
	}{
		{
			name: "succeeds on first attempt",
			returns: []error{
				nil,
			},
			config: RetryConfig{
				Attempts: 3,
			},
			expectedCalls: 1,
			wantErr:       false,
		},
		{
			name: "succeeds after retrying matched status code",
			returns: []error{
				makeOapiErr(429),
				makeOapiErr(429),
				nil, // Succeeds on 3rd try
			},
			config: RetryConfig{
				Attempts:         3,
				Delay:            1 * time.Millisecond,
				RetryStatusCodes: []int{429},
			},
			expectedCalls: 3,
			wantErr:       false,
		},
		{
			name: "fails immediately on non-matching status code",
			returns: []error{
				makeOapiErr(400), // Bad Request (not in retry list)
			},
			config: RetryConfig{
				Attempts:         5,
				RetryStatusCodes: []int{429, 500},
			},
			expectedCalls: 1, // Stops after 1st try
			wantErr:       true,
		},
		{
			name: "fails after exceeding max attempts",
			returns: []error{
				makeOapiErr(500),
				makeOapiErr(500),
				makeOapiErr(500),
			},
			config: RetryConfig{
				Attempts:         3,
				Delay:            1 * time.Millisecond,
				RetryStatusCodes: []int{500},
			},
			expectedCalls: 3,
			wantErr:       true,
		},
		{
			name: "aborts when context is canceled mid-retry",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
				return ctx, cancel
			},
			returns: []error{
				makeOapiErr(503),
				makeOapiErr(503),
			},
			config: RetryConfig{
				Attempts:         5,
				Delay:            100 * time.Millisecond, // Delay is longer than context timeout
				RetryStatusCodes: []int{503},
			},
			expectedCalls: 1, // Aborts during the wait after 1st attempt
			wantErr:       true,
		},
		{
			name:    "fails if fn is nil",
			returns: nil,
			config: RetryConfig{
				Attempts: 3,
			},
			expectedCalls: 0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 1. Context Setup
			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}

			// 2. Mock execution state local to this subtest
			calls := 0
			var fn func() error

			if tt.returns != nil {
				fn = func() error {
					if calls >= len(tt.returns) {
						t.Fatalf("fn called more times (%d) than mocked return values (%d)", calls+1, len(tt.returns))
					}
					err := tt.returns[calls]
					calls++
					return err
				}
			}

			// 3. Execution
			err := RetryRequestWithoutResponse(ctx, fn, tt.config)

			// 4. Assertions
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryRequestWithoutResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if calls != tt.expectedCalls {
				t.Errorf("fn() called %d times, expected %d", calls, tt.expectedCalls)
			}
		})
	}
}
