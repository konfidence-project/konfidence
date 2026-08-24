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

func NewInMemoryStore(cfg config.Parsed) *InMemoryStore {
	cache := otter.Must(&otter.Options[string, Session]{
		ExpiryCalculator: otter.ExpiryCreating[string, Session](cfg.Session.Expiration),
	})

	return &InMemoryStore{cache: *cache}
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

func (s *InMemoryStore) Update(_ context.Context, session *Session) error {
	if session == nil || session.ID == "" {
		return fmt.Errorf("failed to update session: session is empty")
	}

	if _, ok := s.cache.GetIfPresent(session.ID); !ok {
		return fmt.Errorf("failed to update session: session %q not found", session.ID)
	}

	s.cache.Set(session.ID, *session)
	return nil
}
