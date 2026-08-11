package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/maypok86/otter/v2"
)

type SessionStore interface {
	Save(ctx context.Context, state *Session) (string, error)
	Get(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
}
type CacheSessionStore struct {
	cache otter.Cache[string, Session]
}

func (s *CacheSessionStore) Save(_ context.Context, session *Session) (string, error) {
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

func (s *CacheSessionStore) Delete(_ context.Context, id string) error {
	s.cache.Invalidate(id)
	return nil
}

func (s *CacheSessionStore) Get(_ context.Context, id string) (*Session, error) {
	if data, ok := s.cache.GetIfPresent(id); ok {
		return &data, nil
	}
	return nil, nil
}

func NewCacheSessionStore(cfg config.Parsed) *CacheSessionStore {
	cache := otter.Must(&otter.Options[string, Session]{
		ExpiryCalculator: otter.ExpiryAccessing[string, Session](cfg.SessionExpiry),
	})

	return &CacheSessionStore{cache: *cache}
}

type DbSessionStore struct {
	queries db.Queries
}

func (s *DbSessionStore) Save(ctx context.Context, session *Session) (string, error) {
	if session == nil {
		return "", fmt.Errorf("failed to store session: session is empty")
	}
	params := db.CreateSessionParams{
		Subject:           session.Subject,
		Name:              session.Name,
		GivenName:         session.GivenName,
		FamilyName:        session.FamilyName,
		PreferredUserName: session.PreferredUsername,
		Email:             session.Email,
		Roles:             session.Roles,
		Groups:            session.Groups,
		AccessToken:       session.AccessToken,
		RefreshToken:      session.RefreshToken,
		Expiry:            session.Expiry,
	}
	dbId, err := s.queries.CreateSession(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to create session in database: %w", err)
	}

	return dbId.String(), nil
}

func (s *DbSessionStore) Delete(ctx context.Context, id string) error {
	dbId, err := getDbId(id)
	if err != nil {
		return err
	}

	err = s.queries.DeleteSession(ctx, dbId)
	if err != nil {
		return fmt.Errorf("failed to delete session from database: %w", err)
	}

	return nil
}

func (s *DbSessionStore) Get(ctx context.Context, id string) (*Session, error) {
	dbId, err := getDbId(id)
	if err != nil {
		return nil, err
	}

	dbSess, err := s.queries.GetSession(ctx, dbId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// session does not exist
			return nil, nil
		}

		return nil, fmt.Errorf("failed to get session from database: %w", err)
	}

	sess := &Session{
		Subject:           dbSess.Subject,
		Name:              dbSess.Name,
		Email:             dbSess.Email,
		GivenName:         dbSess.GivenName,
		FamilyName:        dbSess.FamilyName,
		PreferredUsername: dbSess.PreferredUserName,
		Roles:             dbSess.Roles,
		Groups:            dbSess.Groups,
		AccessToken:       dbSess.AccessToken,
		RefreshToken:      dbSess.RefreshToken,
		Expiry:            dbSess.Expiry,
	}

	return sess, nil
}

func getDbId(id string) (pgtype.UUID, error) {
	var dbId pgtype.UUID
	err := dbId.Scan(id)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to parse sessionId: %w", err)
	}
	return dbId, nil
}

func NewDbSessionStore(queries db.Queries) *DbSessionStore {
	return &DbSessionStore{queries: queries}
}
