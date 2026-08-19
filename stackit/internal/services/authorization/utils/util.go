package utils

import (
	"encoding/json"
	"sync"
)

// TypeConverter converts objects with equal JSON tags
func TypeConverter[R any](data any) (*R, error) {
	var result R
	b, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// Global map to hold locks for specific assignment IDs
// This ensures that creating the same assignment in parallel waits for the first one to finish
var (
	assignmentLocksMu sync.Mutex
	assignmentLocks   = make(map[string]*sync.Mutex)
)

// LockAssignment acquires a lock for a specific assignment identifier.
// It returns an unlock function that must be deferred.
func LockAssignment(id string) func() {
	assignmentLocksMu.Lock()
	mu, ok := assignmentLocks[id]
	if !ok {
		mu = &sync.Mutex{}
		assignmentLocks[id] = mu
	}
	assignmentLocksMu.Unlock()

	mu.Lock()

	// Return the cleanup function
	return func() {
		mu.Unlock()
	}
}
