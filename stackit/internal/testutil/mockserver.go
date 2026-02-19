package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type MockResponse struct {
	StatusCode  int
	Description string
	ToJsonBody  any
}

var _ http.Handler = (*MockServer)(nil)

type MockServer struct {
	mu           sync.Mutex
	nextResponse int
	responses    []MockResponse
	Server       *httptest.Server
	t            *testing.T
}

// NewMockServer creates a new simple mock server that returns `responses` in order for each request.
// Use the `Reset` method to reset the response order and set new responses.
func NewMockServer(t *testing.T, responses ...MockResponse) *MockServer {
	mock := &MockServer{
		nextResponse: 0,
		responses:    responses,
		t:            t,
	}
	mock.Server = httptest.NewServer(mock)
	return mock
}

func (m *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.nextResponse >= len(m.responses) {
		m.t.Fatalf("No more responses left in the mock server for request: %v", r)
	}
	next := m.responses[m.nextResponse]
	m.nextResponse++
	if next.ToJsonBody != nil {
		bs, err := json.Marshal(next.ToJsonBody)
		if err != nil {
			m.t.Fatalf("Error marshaling response body: %v", err)
		}
		w.Header().Set("content-type", "application/json")
		w.Write(bs)
	}
	status := next.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
}

func (m *MockServer) Reset(responses ...MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextResponse = 0
	m.responses = responses
}
