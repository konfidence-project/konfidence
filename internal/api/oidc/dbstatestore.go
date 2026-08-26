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

type stateQueries interface {
	SaveOIDCState(context.Context, db.SaveOIDCStateParams) error

	ConsumeOIDCState(context.Context, string) (db.ConsumeOIDCStateRow, error)
}

type DBStateStore struct {
	queries    stateQueries
	expiration time.Duration
}

func NewDBStateStore(queries stateQueries, expiration time.Duration,
) *DBStateStore {
	return &DBStateStore{queries: queries, expiration: expiration}
}

func (s *DBStateStore) Save(ctx context.Context, state *StateData) error {
	if state == nil {
		return fmt.Errorf("failed to store state: state is empty")
	}

	err := s.queries.SaveOIDCState(ctx, db.SaveOIDCStateParams{
		State:               state.State,
		Nonce:               state.Nonce,
		ReturnUrl:           state.ReturnURL,
		CodeVerifier:        state.CodeVerifier,
		CodeChallengeMethod: state.CodeChallengeMethod,
		CodeChallenge:       state.CodeChallenge,
		ClientCodeChallenge: state.ClientCodeChallenge,
		CreatedAt: pgtype.Timestamptz{
			Time:  state.CreatedAt,
			Valid: true,
		},
		Expiration: pgtype.Interval{
			Microseconds: s.expiration.Microseconds(),
			Valid:        true,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to store OIDC state: %w", err)
	}

	return nil
}

func (s *DBStateStore) Consume(ctx context.Context, id string) (*StateData, error) {
	stored, err := s.queries.ConsumeOIDCState(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to consume OIDC state: %w", err)
	}

	return &StateData{
		State:               stored.State,
		Nonce:               stored.Nonce,
		ReturnURL:           stored.ReturnUrl,
		CodeVerifier:        stored.CodeVerifier,
		CodeChallengeMethod: stored.CodeChallengeMethod,
		CodeChallenge:       stored.CodeChallenge,
		ClientCodeChallenge: stored.ClientCodeChallenge,
		CreatedAt:           stored.CreatedAt.Time,
	}, nil
}
