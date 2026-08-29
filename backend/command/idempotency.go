package command

import "sync"

type IdempotencyStore struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{seen: make(map[string]struct{})}
}

func (s *IdempotencyStore) IsNew(requestID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.seen[requestID]
	return !ok
}

func (s *IdempotencyStore) MarkPublished(requestID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[requestID] = struct{}{}
}
