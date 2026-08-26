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

const databaseExchangeSessionID = "0194e45e-2fc0-7ef2-96be-84d2f66405de"

type recordingExchangeQueries struct {
	saveFunc func(
		context.Context,
		db.SaveOIDCExchangeParams,
	) error
	consumeFunc func(
		context.Context,
		string,
	) (db.ConsumeOIDCExchangeRow, error)
}

func (q *recordingExchangeQueries) SaveOIDCExchange(
	ctx context.Context,
	params db.SaveOIDCExchangeParams,
) error {
	if q.saveFunc == nil {
		return nil
	}

	return q.saveFunc(ctx, params)
}

func (q *recordingExchangeQueries) ConsumeOIDCExchange(
	ctx context.Context,
	code string,
) (db.ConsumeOIDCExchangeRow, error) {
	if q.consumeFunc == nil {
		return db.ConsumeOIDCExchangeRow{}, nil
	}

	return q.consumeFunc(ctx, code)
}

func databaseExchangeSessionUUID() pgtype.UUID {
	var id pgtype.UUID
	Expect(id.Scan(databaseExchangeSessionID)).To(Succeed())
	return id
}

var _ = Describe("DBExchangeStore", func() {
	const expiration = 5 * time.Minute

	var (
		ctx     context.Context
		queries *recordingExchangeQueries
		store   *DBExchangeStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		queries = &recordingExchangeQueries{}
		store = NewDBExchangeStore(queries, expiration)
	})

	Describe("Save", func() {
		It("rejects an empty exchange code", func() {
			err := store.Save(ctx, "", Exchange{
				SessionID:     databaseExchangeSessionID,
				CodeChallenge: "challenge",
			})

			Expect(err).To(MatchError(
				"exchange code must not be empty",
			))
		})

		It("rejects an empty session ID", func() {
			err := store.Save(ctx, "exchange-code", Exchange{
				CodeChallenge: "challenge",
			})

			Expect(err).To(MatchError(
				"exchange session ID must not be empty",
			))
		})

		It("rejects an empty code challenge", func() {
			err := store.Save(ctx, "exchange-code", Exchange{
				SessionID: databaseExchangeSessionID,
			})

			Expect(err).To(MatchError(
				"exchange code challenge must not be empty",
			))
		})

		It("rejects an invalid session ID", func() {
			err := store.Save(ctx, "exchange-code", Exchange{
				SessionID:     "invalid-session-id",
				CodeChallenge: "challenge",
			})

			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to parse exchange session ID",
				),
			))
		})

		It("maps and stores the exchange", func() {
			exchange := Exchange{
				SessionID:     databaseExchangeSessionID,
				CodeChallenge: "challenge",
			}

			var (
				capturedContext context.Context
				capturedParams  db.SaveOIDCExchangeParams
			)
			queries.saveFunc = func(
				queryContext context.Context,
				params db.SaveOIDCExchangeParams,
			) error {
				capturedContext = queryContext
				capturedParams = params
				return nil
			}

			err := store.Save(
				ctx,
				"exchange-code",
				exchange,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(capturedContext).To(Equal(ctx))
			Expect(capturedParams).To(Equal(
				db.SaveOIDCExchangeParams{
					Code:          "exchange-code",
					SessionID:     databaseExchangeSessionUUID(),
					CodeChallenge: exchange.CodeChallenge,
					Expiration: pgtype.Interval{
						Microseconds: expiration.Microseconds(),
						Valid:        true,
					},
				},
			))
		})

		It("wraps database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.saveFunc = func(
				context.Context,
				db.SaveOIDCExchangeParams,
			) error {
				return databaseErr
			}

			err := store.Save(ctx, "exchange-code", Exchange{
				SessionID:     databaseExchangeSessionID,
				CodeChallenge: "challenge",
			})

			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to store OIDC exchange",
				),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})
	})

	Describe("Consume", func() {
		It("returns nil without querying for an empty code", func() {
			called := false
			queries.consumeFunc = func(
				context.Context,
				string,
			) (db.ConsumeOIDCExchangeRow, error) {
				called = true
				return db.ConsumeOIDCExchangeRow{}, nil
			}

			exchange, err := store.Consume(ctx, "")

			Expect(err).NotTo(HaveOccurred())
			Expect(exchange).To(BeNil())
			Expect(called).To(BeFalse())
		})

		It("maps a consumed exchange", func() {
			expiresAt := time.Date(
				2026,
				time.August,
				26,
				12,
				0,
				0,
				0,
				time.UTC,
			)

			var (
				capturedContext context.Context
				capturedCode    string
			)
			queries.consumeFunc = func(
				queryContext context.Context,
				code string,
			) (db.ConsumeOIDCExchangeRow, error) {
				capturedContext = queryContext
				capturedCode = code

				return db.ConsumeOIDCExchangeRow{
					Code:          code,
					SessionID:     databaseExchangeSessionUUID(),
					CodeChallenge: "challenge",
					ExpiresAt: pgtype.Timestamptz{
						Time:  expiresAt,
						Valid: true,
					},
				}, nil
			}

			exchange, err := store.Consume(
				ctx,
				"exchange-code",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(capturedContext).To(Equal(ctx))
			Expect(capturedCode).To(Equal("exchange-code"))
			Expect(exchange).To(Equal(&Exchange{
				SessionID:     databaseExchangeSessionID,
				CodeChallenge: "challenge",
			}))
		})

		It("returns nil for an unknown or expired exchange", func() {
			queries.consumeFunc = func(
				context.Context,
				string,
			) (db.ConsumeOIDCExchangeRow, error) {
				return db.ConsumeOIDCExchangeRow{}, sql.ErrNoRows
			}

			exchange, err := store.Consume(
				ctx,
				"unknown-exchange",
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(exchange).To(BeNil())
		})

		It("wraps database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.consumeFunc = func(
				context.Context,
				string,
			) (db.ConsumeOIDCExchangeRow, error) {
				return db.ConsumeOIDCExchangeRow{}, databaseErr
			}

			exchange, err := store.Consume(
				ctx,
				"exchange-code",
			)

			Expect(exchange).To(BeNil())
			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to consume OIDC exchange",
				),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})
	})
})
