package oidc

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type recordingStateQueries struct {
	saveFunc    func(context.Context, db.SaveOIDCStateParams) error
	consumeFunc func(
		context.Context,
		string,
	) (db.ConsumeOIDCStateRow, error)
}

func (q *recordingStateQueries) SaveOIDCState(
	ctx context.Context,
	params db.SaveOIDCStateParams,
) error {
	return q.saveFunc(ctx, params)
}

func (q *recordingStateQueries) ConsumeOIDCState(
	ctx context.Context,
	state string,
) (db.ConsumeOIDCStateRow, error) {
	return q.consumeFunc(ctx, state)
}

var _ = Describe("DBStateStore", func() {
	var (
		ctx     context.Context
		queries *recordingStateQueries
		store   *DBStateStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		queries = &recordingStateQueries{}
		store = NewDBStateStore(queries, 5*time.Minute)
	})

	It("maps and saves state", func() {
		createdAt := time.Date(
			2026, time.August, 26, 12, 0, 0, 0, time.UTC,
		)
		state := &StateData{
			State:               "state-id",
			Nonce:               "nonce",
			ReturnURL:           "https://example.com/callback",
			CodeVerifier:        "verifier",
			CodeChallengeMethod: "S256",
			CodeChallenge:       "challenge",
			ClientCodeChallenge: "client-challenge",
			CreatedAt:           createdAt,
		}

		var captured db.SaveOIDCStateParams
		queries.saveFunc = func(
			_ context.Context,
			params db.SaveOIDCStateParams,
		) error {
			captured = params
			return nil
		}

		Expect(store.Save(ctx, state)).To(Succeed())
		Expect(captured.State).To(Equal(state.State))
		Expect(captured.Nonce).To(Equal(state.Nonce))
		Expect(captured.ReturnUrl).To(Equal(state.ReturnURL))
		Expect(captured.CodeVerifier).To(Equal(state.CodeVerifier))
		Expect(captured.CodeChallengeMethod).To(
			Equal(state.CodeChallengeMethod),
		)
		Expect(captured.CodeChallenge).To(Equal(state.CodeChallenge))
		Expect(captured.ClientCodeChallenge).To(
			Equal(state.ClientCodeChallenge),
		)
		Expect(captured.CreatedAt).To(Equal(pgtype.Timestamptz{
			Time:  createdAt,
			Valid: true,
		}))
		Expect(captured.Expiration).To(Equal(pgtype.Interval{
			Microseconds: (5 * time.Minute).Microseconds(),
			Valid:        true,
		}))
	})

	It("returns nil for unknown state", func() {
		queries.consumeFunc = func(
			context.Context,
			string,
		) (db.ConsumeOIDCStateRow, error) {
			return db.ConsumeOIDCStateRow{}, sql.ErrNoRows
		}

		state, err := store.Consume(ctx, "unknown")

		Expect(err).NotTo(HaveOccurred())
		Expect(state).To(BeNil())
	})

	It("wraps database errors", func() {
		databaseErr := errors.New("database unavailable")
		queries.consumeFunc = func(
			context.Context,
			string,
		) (db.ConsumeOIDCStateRow, error) {
			return db.ConsumeOIDCStateRow{}, databaseErr
		}

		state, err := store.Consume(ctx, "state-id")

		Expect(state).To(BeNil())
		Expect(errors.Is(err, databaseErr)).To(BeTrue())
	})
})
