package core

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackitcloud/stackit-sdk-go/core/config"
)

func expectedRetryTransport(base http.RoundTripper) *RetryTransport {
	return &RetryTransport{
		Base:        base,
		MaxRetries:  defaultMaxRetries,
		BaseBackoff: defaultBaseBackoff,
		MaxJitter:   defaultMaxJitter,
	}
}

type retryTestRoundTripper func(*http.Request) (*http.Response, error)

func (f retryTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDefaultClientFactory_RetryRequests(t *testing.T) {
	transportErr := errors.New("transport error")
	tests := []struct {
		name         string
		statuses     []int
		transportErr error
	}{
		{
			name:     "success",
			statuses: []int{http.StatusOK},
		},
		{
			name: "rate limit then success",
			statuses: []int{
				http.StatusTooManyRequests,
				http.StatusOK,
			},
		},
		{
			name: "multiple rate limits then success",
			statuses: []int{
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusOK,
			},
		},
		{
			name: "retry limit",
			statuses: []int{
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
			},
		},
		{
			name:     "not found is not retried",
			statuses: []int{http.StatusNotFound},
		},
		{
			name:     "server error is not retried",
			statuses: []int{http.StatusInternalServerError},
		},
		{
			name:         "transport error is not retried",
			transportErr: transportErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				attempts := 0
				const requestBody = `{"name":"test"}`
				f := &DefaultClientFactory{
					RoundTripper: retryTestRoundTripper(func(req *http.Request) (*http.Response, error) {
						attempts++
						if tt.transportErr != nil {
							return nil, tt.transportErr
						}
						if attempts > len(tt.statuses) {
							t.Fatal("unexpected extra retry")
						}
						body, err := io.ReadAll(req.Body)
						if err != nil {
							t.Fatal(err)
						}
						if string(body) != requestBody {
							t.Fatalf("request body was not preserved: %q", body)
						}
						if err := req.Body.Close(); err != nil {
							t.Fatal(err)
						}
						return &http.Response{
							StatusCode: tt.statuses[attempts-1],
							Header:     make(http.Header),
							Body:       io.NopCloser(strings.NewReader("response")),
						}, nil
					}),
				}
				cfg := config.Configuration{}
				for _, opt := range f.defaultConfigOptions("") {
					if err := opt(&cfg); err != nil {
						t.Fatal(err)
					}
				}
				req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://example.com", strings.NewReader(requestBody))
				if err != nil {
					t.Fatal(err)
				}
				resp, err := cfg.CustomAuth.RoundTrip(req)
				if !errors.Is(err, tt.transportErr) {
					t.Fatalf("RoundTrip() error = %v, want %v", err, tt.transportErr)
				}
				wantAttempts := len(tt.statuses)
				if tt.transportErr != nil {
					wantAttempts = 1
				} else {
					defer func() {
						if err := resp.Body.Close(); err != nil {
							t.Error(err)
						}
					}()
					if got, want := resp.StatusCode, tt.statuses[len(tt.statuses)-1]; got != want {
						t.Fatalf("status = %d, want %d", got, want)
					}
					body, err := io.ReadAll(resp.Body)
					if err != nil || string(body) != "response" {
						t.Fatalf("final response body = %q, error = %v", body, err)
					}
				}
				if attempts != wantAttempts {
					t.Fatalf("attempts = %d, want %d", attempts, wantAttempts)
				}
			})
		})
	}
}

func TestRetryTransport_ExponentialBackoffWithJitter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		attempts := 0
		factory := &DefaultClientFactory{
			RoundTripper: retryTestRoundTripper(func(*http.Request) (*http.Response, error) {
				attempts++
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Header:     make(http.Header), // No Retry-After: use exponential backoff.
					Body:       http.NoBody,
				}, nil
			}),
		}
		cfg := config.Configuration{}
		for _, opt := range factory.defaultConfigOptions("") {
			if err := opt(&cfg); err != nil {
				t.Fatal(err)
			}
		}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}

		start := time.Now()
		resp, err := cfg.CustomAuth.RoundTrip(req)
		elapsed := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", resp.StatusCode)
		}
		expected := expectedRetryTransport(nil)
		if wantAttempts := expected.MaxRetries + 1; attempts != wantAttempts {
			t.Fatalf("attempts = %d, want %d", attempts, wantAttempts)
		}

		var backoff time.Duration
		for attempt := 0; attempt < expected.MaxRetries; attempt++ {
			backoff += expected.BaseBackoff * (1 << attempt)
		}
		maxJitter := time.Duration(expected.MaxRetries) * expected.MaxJitter
		if elapsed <= backoff {
			t.Errorf("elapsed = %s, want greater than %s due to jitter", elapsed, backoff)
		}
		if elapsed >= backoff+maxJitter {
			t.Errorf("elapsed = %s, want less than %s", elapsed, backoff+maxJitter)
		}
	})
}

func TestRetryTransport_ContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		attempts := 0
		transport := expectedRetryTransport(retryTestRoundTripper(func(*http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     http.Header{"Retry-After": {"60"}},
				Body:       http.NoBody,
			}, nil
		}))
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		resp, err := transport.RoundTrip(req)
		if elapsed := time.Since(start); elapsed != time.Second {
			t.Fatalf("elapsed = %s, want %s", elapsed, time.Second)
		}
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
		if attempts != 1 {
			t.Fatalf("attempts = %d, want 1", attempts)
		}
	})
}

func TestRetryTransport_ContextDeadline(t *testing.T) {
	tests := []struct {
		name         string
		retryAfter   string
		timeout      time.Duration
		wantElapsed  time.Duration
		wantAttempts int
		wantErr      error
	}{
		{
			name:         "retry after exceeds remaining time",
			retryAfter:   "60",
			timeout:      time.Second,
			wantAttempts: 1,
			wantErr:      context.DeadlineExceeded,
		},
		{
			name:         "backoff exceeds remaining time",
			timeout:      time.Second,
			wantAttempts: 1,
			wantErr:      context.DeadlineExceeded,
		},
		{
			name:         "backoff equals remaining time",
			timeout:      defaultBaseBackoff,
			wantAttempts: 1,
			wantErr:      context.DeadlineExceeded,
		},
		{
			name:         "retry fits before deadline",
			timeout:      2 * defaultBaseBackoff,
			wantElapsed:  defaultBaseBackoff,
			wantAttempts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				attempts := 0
				transport := expectedRetryTransport(retryTestRoundTripper(func(*http.Request) (*http.Response, error) {
					attempts++
					status := http.StatusTooManyRequests
					if attempts > 1 {
						status = http.StatusOK
					}
					return &http.Response{
						StatusCode: status,
						Header:     http.Header{"Retry-After": {tt.retryAfter}},
						Body:       http.NoBody,
					}, nil
				}))
				// A one-nanosecond upper bound makes the jitter zero for exact timing checks.
				transport.MaxJitter = time.Nanosecond
				ctx, cancel := context.WithTimeout(t.Context(), tt.timeout)
				defer cancel()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", http.NoBody)
				if err != nil {
					t.Fatal(err)
				}

				start := time.Now()
				resp, err := transport.RoundTrip(req)
				if resp != nil {
					if closeErr := resp.Body.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if elapsed := time.Since(start); elapsed != tt.wantElapsed {
					t.Errorf("elapsed = %s, want %s", elapsed, tt.wantElapsed)
				}
				if attempts != tt.wantAttempts {
					t.Errorf("attempts = %d, want %d", attempts, tt.wantAttempts)
				}
				if err := ctx.Err(); err != nil {
					t.Errorf("context expired instead of returning early: %v", err)
				}
			})
		})
	}
}
