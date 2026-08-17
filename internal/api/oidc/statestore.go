package oidc

import (
	"fmt"
	"sync"
	"time"

	"github.com/maypok86/otter/v2"
)

type StateStore interface {
	Save(state *StateData) error
	Consume(id string) (*StateData, error)
}
type StateCacheStore struct {
	mu    sync.Mutex
	cache otter.Cache[string, StateData]
}

func (s *StateCacheStore) Save(state *StateData) error {
	if state == nil {
		return fmt.Errorf("failed to store state: state is empty")
	}

	s.cache.Set(state.State, *state)
	return nil
}

func (s *StateCacheStore) Consume(id string) (*StateData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.cache.GetIfPresent(id)
	if !ok {
		return nil, nil
	}

	s.cache.Invalidate(id)
	return &state, nil
}

func NewStateCacheStore(expiration time.Duration) *StateCacheStore {
	cache := otter.Must(&otter.Options[string, StateData]{
		ExpiryCalculator: otter.ExpiryCreating[string, StateData](expiration),
	})

	return &StateCacheStore{cache: *cache}
}
