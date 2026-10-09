package tsukuyomi

import (
	"context"
	"sync"
)

// MemoryStore is an in-memory Store, handy for tests and prototypes.
type MemoryStore[ID, S comparable] struct {
	mu     sync.Mutex
	states map[ID]S
}

func NewMemoryStore[ID, S comparable]() *MemoryStore[ID, S] {
	return &MemoryStore[ID, S]{states: make(map[ID]S)}
}

// Set puts an entity into a state without any checks, e.g. on creation.
func (s *MemoryStore[ID, S]) Set(id ID, state S) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[id] = state
}

func (s *MemoryStore[ID, S]) State(_ context.Context, id ID) (S, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[id]
	if !ok {
		return state, ErrNotFound
	}
	return state, nil
}

func (s *MemoryStore[ID, S]) CompareAndSwap(_ context.Context, id ID, from, to S) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.states[id]
	if !ok {
		return false, ErrNotFound
	}
	if cur != from {
		return false, nil
	}
	s.states[id] = to
	return true, nil
}
