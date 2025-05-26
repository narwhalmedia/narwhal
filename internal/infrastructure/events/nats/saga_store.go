package nats

import (
	"context"
	"fmt"
	"sync"
)

// InMemorySagaStore provides an in-memory implementation of SagaStore
type InMemorySagaStore struct {
	mu    sync.RWMutex
	sagas map[string]*Saga
}

// NewInMemorySagaStore creates a new in-memory saga store
func NewInMemorySagaStore() *InMemorySagaStore {
	return &InMemorySagaStore{
		sagas: make(map[string]*Saga),
	}
}

// Save saves a saga
func (s *InMemorySagaStore) Save(ctx context.Context, saga *Saga) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sagas[saga.ID] = saga
	return nil
}

// Get retrieves a saga by ID
func (s *InMemorySagaStore) Get(ctx context.Context, id string) (*Saga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	saga, ok := s.sagas[id]
	if !ok {
		return nil, fmt.Errorf("saga not found: %s", id)
	}

	return saga, nil
}

// UpdateState updates the state of a saga
func (s *InMemorySagaStore) UpdateState(ctx context.Context, id string, state SagaState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	saga, ok := s.sagas[id]
	if !ok {
		return fmt.Errorf("saga not found: %s", id)
	}

	saga.State = state
	return nil
}