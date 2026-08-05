package session

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/maypok86/otter/v2"
)

type SessionStore interface {
	Save(state Session) (string, error)
	Get(id string) (*Session, error)
	Delete(id string) error
}
type SessionCacheStore struct {
	cache otter.Cache[string, Session]
}

func (s *SessionCacheStore) Save(session Session) (string, error) {
	uuid7, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}

	id := uuid7.String()
	s.cache.Set(id, session)
	return id, nil
}

func (s *SessionCacheStore) Delete(id string) error {
	s.cache.Invalidate(id)
	return nil
}

func (s *SessionCacheStore) Get(id string) (*Session, error) {
	if data, ok := s.cache.GetIfPresent(id); ok {
		return &data, nil
	}
	return nil, nil
}

func NewSessionCacheStore(_ config.Parsed) *SessionCacheStore {
	// TODO use value from cfg
	oidcSessionExpiration := 30 * time.Minute
	cache := otter.Must(&otter.Options[string, Session]{
		ExpiryCalculator: otter.ExpiryAccessing[string, Session](oidcSessionExpiration),
	})

	return &SessionCacheStore{cache: *cache}
}
