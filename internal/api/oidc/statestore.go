package oidc

import (
	"time"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/maypok86/otter/v2"
)

type StateStore interface {
	Save(state StateData) error
	Get(id string) (*StateData, error)
	Delete(state StateData) error
}
type StateCacheStore struct {
	cache otter.Cache[string, StateData]
}

func (s *StateCacheStore) Save(state StateData) error {
	s.cache.Set(state.State, state)
	return nil
}

func (s *StateCacheStore) Delete(state StateData) error {
	s.cache.Invalidate(state.State)
	return nil
}

func (s *StateCacheStore) Get(id string) (*StateData, error) {
	if data, ok := s.cache.GetIfPresent(id); ok {
		return &data, nil
	}
	return nil, nil
}

func NewStateCacheStore(_ config.Parsed) *StateCacheStore {
	// TODO use value from cfg
	oidcStateExpiration := 15 * time.Minute
	cache := otter.Must(&otter.Options[string, StateData]{
		ExpiryCalculator: otter.ExpiryCreating[string, StateData](oidcStateExpiration),
	})

	return &StateCacheStore{cache: *cache}
}
