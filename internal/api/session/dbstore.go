package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
)

var _ cleanupTransaction = (*postgresCleanupTransaction)(nil)
var _ cleanupTransactionHandler = postgresCleanupTransactionHandler{}

type sessionQueries interface {
	CreateSession(context.Context, db.CreateSessionParams) (pgtype.UUID, error)
	GetSession(context.Context, pgtype.UUID) (db.Session, error)
	DeleteSession(context.Context, pgtype.UUID) error
}

type cleanupQueries interface {
	TryAcquireCleanupLock(context.Context) (bool, error)
	DeleteExpiredSessions(context.Context) (int64, error)
	DeleteExpiredOIDCStates(context.Context) (int64, error)
	DeleteExpiredOIDCExchanges(context.Context) (int64, error)
}

type cleanupTransaction interface {
	cleanupQueries
	Commit(context.Context) error
	Rollback(context.Context) error
}

type cleanupTransactionHandler interface {
	BeginTransaction(context.Context) (cleanupTransaction, error)
}

type postgresCleanupTransaction struct {
	pgx.Tx
	*db.Queries
}

type postgresCleanupTransactionHandler struct {
	pool *pgxpool.Pool
}

func (b postgresCleanupTransactionHandler) BeginTransaction(ctx context.Context) (cleanupTransaction, error) {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &postgresCleanupTransaction{Tx: tx, Queries: db.New(tx)}, nil
}

type DBStore struct {
	queries                   sessionQueries
	cleanupTransactionHandler cleanupTransactionHandler
	sessionExpiration         time.Duration
}

func NewDBStore(queries sessionQueries, pool *pgxpool.Pool, sessionExpiration time.Duration) *DBStore {
	return &DBStore{
		queries:                   queries,
		cleanupTransactionHandler: postgresCleanupTransactionHandler{pool: pool},
		sessionExpiration:         sessionExpiration,
	}
}

func newDBStore(queries sessionQueries, cleanupTransactionHandler cleanupTransactionHandler,
	sessionExpiration time.Duration) *DBStore {
	return &DBStore{
		queries:                   queries,
		cleanupTransactionHandler: cleanupTransactionHandler,
		sessionExpiration:         sessionExpiration,
	}
}

func (s *DBStore) Save(ctx context.Context, session *Session) (string, error) {
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
		Groups:            session.Groups,
		AccessToken:       session.AccessToken,
		RefreshToken:      session.RefreshToken,
		TokenExpiry:       session.TokenExpiry,
		Expiration: pgtype.Interval{
			Microseconds: s.sessionExpiration.Microseconds(),
			Valid:        true,
		},
	}
	dbID, err := s.queries.CreateSession(ctx, params)
	if err != nil {
		return "", fmt.Errorf("failed to create session in database: %w", err)
	}

	return dbID.String(), nil
}

func (s *DBStore) Delete(ctx context.Context, id string) error {
	dbID, err := getDBID(id)
	if err != nil {
		return err
	}

	if err := s.queries.DeleteSession(ctx, dbID); err != nil {
		return fmt.Errorf("failed to delete session from database: %w", err)
	}
	return nil
}

func (s *DBStore) Get(ctx context.Context, id string) (*Session, error) {
	dbID, err := getDBID(id)
	if err != nil {
		return nil, err
	}

	dbSession, err := s.queries.GetSession(ctx, dbID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session from database: %w", err)
	}

	return mapDBSession(dbSession), nil
}

func getDBID(id string) (pgtype.UUID, error) {
	var dbID pgtype.UUID
	if err := dbID.Scan(id); err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to parse session ID: %w", err)
	}
	return dbID, nil
}

func mapDBSession(stored db.Session) *Session {
	return &Session{
		Subject: stored.Subject,
		Groups:  append([]string(nil), stored.Groups...),
		Context: Context{
			ID:                stored.ID.String(),
			Name:              stored.Name,
			Email:             stored.Email,
			GivenName:         stored.GivenName,
			FamilyName:        stored.FamilyName,
			PreferredUsername: stored.PreferredUserName,
		},
		AccessToken:  stored.AccessToken,
		RefreshToken: stored.RefreshToken,
		TokenExpiry:  stored.TokenExpiry,
	}
}
