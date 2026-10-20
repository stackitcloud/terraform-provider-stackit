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
		MaxRetries:  3,
		BaseBackoff: 10 * time.Second,
		MaxJitter:   500 * time.Millisecond,
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
		{name: "success", statuses: []int{http.StatusOK}},
		{name: "rate limit then success", statuses: []int{http.StatusTooManyRequests, http.StatusOK}},
		{name: "multiple rate limits then success", statuses: []int{http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusOK}},
		{name: "retry limit", statuses: []int{http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests}},
		{name: "not found is not retried", statuses: []int{http.StatusNotFound}},
		{name: "server error is not retried", statuses: []int{http.StatusInternalServerError}},
		{name: "transport error is not retried", transportErr: transportErr},
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
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := transport.RoundTrip(req)
		if resp != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				t.Error(closeErr)
			}
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want context deadline exceeded", err)
		}
		if attempts != 1 {
			t.Fatalf("attempts = %d, want 1", attempts)
		}
	})
}
