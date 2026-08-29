package state

import (
	"fmt"
	"sync"
)

type Status string

const (
	StatusPending      Status = "PENDING"
	StatusPublished    Status = "PUBLISHED"
	StatusAcknowledged Status = "ACKNOWLEDGED"
	StatusSuccess      Status = "SUCCESS"
	StatusFailed       Status = "FAILED"
	StatusExpired      Status = "EXPIRED"
	StatusDuplicate    Status = "DUPLICATE"
)

type Tracker struct {
	mu     sync.Mutex
	states map[string]Status
}

func NewTracker() *Tracker {
	return &Tracker{states: make(map[string]Status)}
}

func (t *Tracker) Set(requestID string, status Status) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.states[requestID] = status
}

func (t *Tracker) Get(requestID string) (Status, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.states[requestID]
	return s, ok
}

func (t *Tracker) Transition(requestID string, to Status) (Status, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	from, ok := t.states[requestID]
	if !ok {
		t.states[requestID] = to
		return StatusPending, nil
	}

	if !validTransition(from, to) {
		return from, fmt.Errorf("invalid transition %s -> %s for request_id=%s", from, to, requestID)
	}

	t.states[requestID] = to
	return from, nil
}

func validTransition(from, to Status) bool {
	switch to {
	case StatusPublished:
		return from == StatusPending
	case StatusAcknowledged:
		return from == StatusPublished
	case StatusSuccess:
		return from == StatusAcknowledged || from == StatusPublished
	case StatusFailed, StatusExpired, StatusDuplicate:
		return true
	default:
		return false
	}
}
