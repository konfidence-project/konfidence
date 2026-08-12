package session

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/maypok86/otter/v2"
)

type InMemoryStore struct {
	cache otter.Cache[string, Session]
}

func (s *InMemoryStore) Save(_ context.Context, session *Session) (string, error) {
	if session == nil {
		return "", fmt.Errorf("failed to store session: session is empty")
	}

	uuid7, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}

	id := uuid7.String()
	session.ID = id
	s.cache.Set(id, *session)
	return id, nil
}

func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.cache.Invalidate(id)
	return nil
}

func (s *InMemoryStore) Get(_ context.Context, id string) (*Session, error) {
	if data, ok := s.cache.GetIfPresent(id); ok {
		return &data, nil
	}
	return nil, nil
}

func NewInMemoryStore(cfg config.Parsed) *InMemoryStore {
	cache := otter.Must(&otter.Options[string, Session]{
		ExpiryCalculator: otter.ExpiryAccessing[string, Session](cfg.SessionExpiry),
	})

	return &InMemoryStore{cache: *cache}
}
