package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
)

type DBStore struct {
	queries db.Querier
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
		Expiry:            session.Expiry,
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

	return &Session{
		Subject: dbSession.Subject,
		Context: Context{
			ID:                id,
			Name:              dbSession.Name,
			Email:             dbSession.Email,
			GivenName:         dbSession.GivenName,
			FamilyName:        dbSession.FamilyName,
			PreferredUsername: dbSession.PreferredUserName,
			Groups:            dbSession.Groups,
		},
		AccessToken:  dbSession.AccessToken,
		RefreshToken: dbSession.RefreshToken,
		Expiry:       dbSession.Expiry,
	}, nil
}

func getDBID(id string) (pgtype.UUID, error) {
	var dbID pgtype.UUID
	if err := dbID.Scan(id); err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to parse session ID: %w", err)
	}
	return dbID, nil
}

func NewDBStore(queries db.Querier) *DBStore {
	return &DBStore{queries: queries}
}
