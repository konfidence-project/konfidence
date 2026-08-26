package oidc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
)

type exchangeQueries interface {
	SaveOIDCExchange(context.Context, db.SaveOIDCExchangeParams) error

	ConsumeOIDCExchange(context.Context, string) (db.ConsumeOIDCExchangeRow, error)
}

type DBExchangeStore struct {
	queries    exchangeQueries
	expiration time.Duration
}

func NewDBExchangeStore(queries exchangeQueries, expiration time.Duration) *DBExchangeStore {
	return &DBExchangeStore{queries: queries, expiration: expiration}
}

func (s *DBExchangeStore) Save(ctx context.Context, code string, exchange Exchange) error {
	if code == "" {
		return fmt.Errorf("exchange code must not be empty")
	}
	if exchange.SessionID == "" {
		return fmt.Errorf("exchange session ID must not be empty")
	}
	if exchange.CodeChallenge == "" {
		return fmt.Errorf("exchange code challenge must not be empty")
	}

	var sessionID pgtype.UUID
	if err := sessionID.Scan(exchange.SessionID); err != nil {
		return fmt.Errorf("failed to parse exchange session ID: %w", err)
	}

	err := s.queries.SaveOIDCExchange(
		ctx,
		db.SaveOIDCExchangeParams{
			Code:          code,
			SessionID:     sessionID,
			CodeChallenge: exchange.CodeChallenge,
			Expiration: pgtype.Interval{
				Microseconds: s.expiration.Microseconds(),
				Valid:        true,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to store OIDC exchange: %w", err)
	}

	return nil
}

func (s *DBExchangeStore) Consume(ctx context.Context, code string) (*Exchange, error) {
	if code == "" {
		return nil, nil
	}

	stored, err := s.queries.ConsumeOIDCExchange(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to consume OIDC exchange: %w", err)
	}

	return &Exchange{
		SessionID:     stored.SessionID.String(),
		CodeChallenge: stored.CodeChallenge,
	}, nil
}
