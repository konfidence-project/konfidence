package session

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	db "github.com/konfidence-project/konfidence/cmd/api/db/sqlc"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ sessionQueries = (*recordingQuerier)(nil)
var _ cleanupTransaction = (*recordingCleanupTransaction)(nil)
var _ cleanupTransactionHandler = (*recordingCleanupTransactionBeginner)(nil)

type recordingQuerier struct {
	mu sync.Mutex

	createSessionFunc              func(context.Context, db.CreateSessionParams) (pgtype.UUID, error)
	deleteSessionFunc              func(context.Context, pgtype.UUID) error
	getSessionFunc                 func(context.Context, pgtype.UUID) (db.Session, error)
	deleteExpiredSessionsFunc      func(context.Context) (int64, error)
	deleteExpiredOIDCStatesFunc    func(context.Context) (int64, error)
	deleteExpiredOIDCExchangesFunc func(context.Context) (int64, error)
}

type recordingCleanupTransaction struct {
	*recordingQuerier
	tryAcquireCleanupLockFunc func(context.Context) (bool, error)
	commitFunc                func(context.Context) error
	rollbackFunc              func(context.Context) error
	commitCalls               int
	rollbackCalls             int
}

func (tx *recordingCleanupTransaction) TryAcquireCleanupLock(ctx context.Context) (bool, error) {
	if tx.tryAcquireCleanupLockFunc == nil {
		return true, nil
	}
	return tx.tryAcquireCleanupLockFunc(ctx)
}

func (tx *recordingCleanupTransaction) Commit(ctx context.Context) error {
	tx.commitCalls++
	if tx.commitFunc == nil {
		return nil
	}
	return tx.commitFunc(ctx)
}

func (tx *recordingCleanupTransaction) Rollback(ctx context.Context) error {
	tx.rollbackCalls++
	if tx.rollbackFunc == nil {
		return nil
	}
	return tx.rollbackFunc(ctx)
}

type recordingCleanupTransactionBeginner struct {
	transaction cleanupTransaction
	beginFunc   func(context.Context) (cleanupTransaction, error)
	beginCalls  int
}

func (b *recordingCleanupTransactionBeginner) BeginTransaction(
	ctx context.Context,
) (cleanupTransaction, error) {
	b.beginCalls++
	if b.beginFunc != nil {
		return b.beginFunc(ctx)
	}
	return b.transaction, nil
}

func (q *recordingQuerier) CreateSession(
	ctx context.Context,
	params db.CreateSessionParams,
) (pgtype.UUID, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.createSessionFunc == nil {
		return pgtype.UUID{}, nil
	}
	return q.createSessionFunc(ctx, params)
}

func (q *recordingQuerier) DeleteSession(ctx context.Context, id pgtype.UUID) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleteSessionFunc == nil {
		return nil
	}
	return q.deleteSessionFunc(ctx, id)
}

func (q *recordingQuerier) GetSession(ctx context.Context, id pgtype.UUID) (db.Session, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.getSessionFunc == nil {
		return db.Session{}, nil
	}

	return q.getSessionFunc(ctx, id)
}

func (q *recordingQuerier) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleteExpiredSessionsFunc == nil {
		return 0, nil
	}

	return q.deleteExpiredSessionsFunc(ctx)
}

func (q *recordingQuerier) DeleteExpiredOIDCStates(ctx context.Context) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleteExpiredOIDCStatesFunc == nil {
		return 0, nil
	}

	return q.deleteExpiredOIDCStatesFunc(ctx)
}

func (q *recordingQuerier) DeleteExpiredOIDCExchanges(ctx context.Context) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleteExpiredOIDCExchangesFunc == nil {
		return 0, nil
	}
	return q.deleteExpiredOIDCExchangesFunc(ctx)
}

const databaseSessionID = "0194e45e-2fc0-7ef2-96be-84d2f66405de"

func databaseSessionUUID() pgtype.UUID {
	var id pgtype.UUID
	Expect(id.Scan(databaseSessionID)).To(Succeed())
	return id
}

var _ = ginkgo.Describe("DBStore", func() {

	var (
		ctx                 context.Context
		queries             *recordingQuerier
		cleanupTx           *recordingCleanupTransaction
		cleanupTransactions *recordingCleanupTransactionBeginner
		store               *DBStore
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		queries = &recordingQuerier{}

		cleanupTx = &recordingCleanupTransaction{
			recordingQuerier: queries,
		}

		cleanupTransactions = &recordingCleanupTransactionBeginner{
			transaction: cleanupTx,
		}

		store = newDBStore(
			queries,
			cleanupTransactions,
			30*time.Minute,
		)
	})

	ginkgo.Describe("Save", func() {
		ginkgo.It("rejects a nil session", func() {
			id, err := store.Save(ctx, nil)

			Expect(id).To(BeEmpty())
			Expect(err).To(MatchError(
				"failed to store session: session is empty",
			))
		})

		ginkgo.It("maps and stores all session fields", func() {
			refreshToken := "refresh-token"
			session := &Session{
				Subject:      "subject-id",
				Groups:       []string{"developers", "admins"},
				AccessToken:  "access-token",
				RefreshToken: &refreshToken,
				TokenExpiry:  123456789,
				Context: Context{
					Name:              new("Test User"),
					GivenName:         new("Test"),
					FamilyName:        new("User"),
					PreferredUsername: new("test.user"),
					Email:             new("test@example.com"),
				},
			}

			var captured db.CreateSessionParams
			queries.createSessionFunc = func(
				_ context.Context,
				params db.CreateSessionParams,
			) (pgtype.UUID, error) {
				captured = params
				return databaseSessionUUID(), nil
			}

			id, err := store.Save(ctx, session)

			Expect(err).NotTo(HaveOccurred())
			Expect(id).To(Equal(databaseSessionID))
			Expect(captured).To(Equal(db.CreateSessionParams{
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
					Microseconds: (30 * time.Minute).Microseconds(),
					Valid:        true,
				},
			}))
		})

		ginkgo.It("wraps database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.createSessionFunc = func(
				context.Context,
				db.CreateSessionParams,
			) (pgtype.UUID, error) {
				return pgtype.UUID{}, databaseErr
			}

			id, err := store.Save(ctx, &Session{
				Subject: "subject-id",
			})

			Expect(id).To(BeEmpty())
			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to create session in database",
				),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})
	})

	ginkgo.Describe("Get", func() {
		ginkgo.It("gets an unexpired session by ID", func() {
			refreshToken := "refresh-token"
			databaseSession := db.Session{
				ID:                databaseSessionUUID(),
				Subject:           "subject-id",
				Name:              new("Test User"),
				GivenName:         new("Test"),
				FamilyName:        new("User"),
				PreferredUserName: new("test.user"),
				Email:             new("test@example.com"),
				Groups:            []string{"developers", "admins"},
				AccessToken:       "access-token",
				RefreshToken:      &refreshToken,
				TokenExpiry:       123456789,
			}

			var captured pgtype.UUID
			queries.getSessionFunc = func(_ context.Context, id pgtype.UUID) (db.Session, error) {
				captured = id
				return databaseSession, nil
			}

			stored, err := store.Get(ctx, databaseSessionID)

			Expect(err).NotTo(HaveOccurred())
			Expect(stored).To(Equal(&Session{
				Subject: databaseSession.Subject,
				Groups:  databaseSession.Groups,
				Context: Context{
					ID:                databaseSessionID,
					Name:              databaseSession.Name,
					Email:             databaseSession.Email,
					GivenName:         databaseSession.GivenName,
					FamilyName:        databaseSession.FamilyName,
					PreferredUsername: databaseSession.PreferredUserName,
				},
				AccessToken:  databaseSession.AccessToken,
				RefreshToken: databaseSession.RefreshToken,
				TokenExpiry:  databaseSession.TokenExpiry,
			}))

			Expect(captured).To(Equal(databaseSessionUUID()))
		})

		ginkgo.It("returns nil when the session does not exist", func() {
			queries.getSessionFunc = func(context.Context, pgtype.UUID) (db.Session, error) {
				return db.Session{}, sql.ErrNoRows
			}

			stored, err := store.Get(ctx, databaseSessionID)
			Expect(err).NotTo(HaveOccurred())
			Expect(stored).To(BeNil())
		})

		ginkgo.It("rejects an invalid session ID", func() {
			queried := false
			queries.getSessionFunc = func(context.Context, pgtype.UUID) (db.Session, error) {
				queried = true
				return db.Session{}, nil
			}

			stored, err := store.Get(ctx, "invalid-session-id")

			Expect(stored).To(BeNil())
			Expect(err).To(MatchError(
				ContainSubstring("failed to parse session ID"),
			))
			Expect(queried).To(BeFalse())
		})

		ginkgo.It("wraps database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.getSessionFunc = func(context.Context, pgtype.UUID) (db.Session, error) {
				return db.Session{}, databaseErr
			}

			stored, err := store.Get(ctx, databaseSessionID)

			Expect(stored).To(BeNil())
			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to get session from database",
				),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})
	})

	ginkgo.Describe("Delete", func() {
		ginkgo.It("deletes the parsed database session ID", func() {
			var captured pgtype.UUID
			queries.deleteSessionFunc = func(
				_ context.Context,
				id pgtype.UUID,
			) error {
				captured = id
				return nil
			}

			Expect(store.Delete(ctx, databaseSessionID)).To(Succeed())
			Expect(captured).To(Equal(databaseSessionUUID()))
		})

		ginkgo.It("rejects an invalid session ID", func() {
			err := store.Delete(ctx, "invalid-session-id")

			Expect(err).To(MatchError(
				ContainSubstring("failed to parse session ID"),
			))
		})

		ginkgo.It("wraps database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.deleteSessionFunc = func(
				context.Context,
				pgtype.UUID,
			) error {
				return databaseErr
			}

			err := store.Delete(ctx, databaseSessionID)

			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to delete session from database",
				),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})
	})

	ginkgo.Describe("Cleanup", func() {
		ginkgo.It("deletes expired sessions and OIDC data under the cleanup lock", func() {
			queries.deleteExpiredSessionsFunc = func(context.Context) (int64, error) {
				return 3, nil
			}

			queries.deleteExpiredOIDCStatesFunc = func(
				context.Context,
			) (int64, error) {
				return 2, nil
			}

			queries.deleteExpiredOIDCExchangesFunc = func(
				context.Context,
			) (int64, error) {
				return 1, nil
			}

			deleted, acquired, err := store.deleteExpired(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeTrue())
			Expect(deleted).To(Equal(cleanupResult{
				sessions:  3,
				states:    2,
				exchanges: 1,
			}))
			Expect(cleanupTransactions.beginCalls).To(Equal(1))
			Expect(cleanupTx.commitCalls).To(Equal(1))
			Expect(cleanupTx.rollbackCalls).To(Equal(0))
		})

		ginkgo.It("skips cleanup when another instance holds the lock", func() {
			cleanupTx.tryAcquireCleanupLockFunc = func(
				context.Context,
			) (bool, error) {
				return false, nil
			}

			sessionCalls := 0
			stateCalls := 0
			exchangeCalls := 0

			queries.deleteExpiredSessionsFunc = func(context.Context) (int64, error) {
				sessionCalls++
				return 0, nil
			}

			queries.deleteExpiredOIDCStatesFunc = func(context.Context) (int64, error) {
				stateCalls++
				return 0, nil
			}

			queries.deleteExpiredOIDCExchangesFunc = func(context.Context) (int64, error) {
				exchangeCalls++
				return 0, nil
			}

			deleted, acquired, err := store.deleteExpired(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(acquired).To(BeFalse())
			Expect(deleted).To(BeZero())
			Expect(sessionCalls).To(BeZero())
			Expect(stateCalls).To(BeZero())
			Expect(exchangeCalls).To(BeZero())
			Expect(cleanupTx.commitCalls).To(BeZero())
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
		})

		ginkgo.It("wraps transaction start errors", func() {
			databaseErr := errors.New("database unavailable")
			cleanupTransactions.beginFunc = func(
				context.Context,
			) (cleanupTransaction, error) {
				return nil, databaseErr
			}

			deleted, acquired, err := store.deleteExpired(ctx)

			Expect(deleted).To(BeZero())
			Expect(acquired).To(BeFalse())
			Expect(err).To(MatchError(
				ContainSubstring("starting cleanup transaction failed"),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})

		ginkgo.It("wraps advisory lock errors", func() {
			databaseErr := errors.New("lock query failed")
			cleanupTx.tryAcquireCleanupLockFunc = func(
				context.Context,
			) (bool, error) {
				return false, databaseErr
			}

			deleted, acquired, err := store.deleteExpired(ctx)
			Expect(deleted).To(BeZero())
			Expect(acquired).To(BeFalse())
			Expect(err).To(MatchError(
				ContainSubstring("acquiring cleanup advisory lock failed"),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
			Expect(cleanupTx.commitCalls).To(BeZero())
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
		})

		ginkgo.It("wraps session cleanup database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.deleteExpiredSessionsFunc = func(context.Context) (int64, error) {
				return 0, databaseErr
			}
			deleted, acquired, err := store.deleteExpired(ctx)
			Expect(deleted).To(BeZero())
			Expect(acquired).To(BeTrue())
			Expect(cleanupTx.commitCalls).To(BeZero())
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
			Expect(err).To(MatchError(
				ContainSubstring("deleting expired sessions failed"),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})

		ginkgo.It("wraps OIDC state cleanup database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.deleteExpiredOIDCStatesFunc = func(context.Context) (int64, error) {
				return 0, databaseErr
			}

			deleted, acquired, err := store.deleteExpired(ctx)

			Expect(deleted).To(BeZero())
			Expect(acquired).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("deleting expired OIDC states failed")))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
			Expect(cleanupTx.commitCalls).To(BeZero())
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
		})

		ginkgo.It("wraps OIDC exchange cleanup database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.deleteExpiredOIDCExchangesFunc = func(context.Context) (int64, error) {
				return 0, databaseErr
			}

			deleted, acquired, err := store.deleteExpired(ctx)

			Expect(deleted).To(BeZero())
			Expect(acquired).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("deleting expired OIDC exchanges failed")))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
			Expect(cleanupTx.commitCalls).To(BeZero())
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
		})

		ginkgo.It("wraps transaction commit errors", func() {
			databaseErr := errors.New("commit failed")
			cleanupTx.commitFunc = func(context.Context) error { return databaseErr }
			deleted, acquired, err := store.deleteExpired(ctx)

			Expect(deleted).To(BeZero())
			Expect(acquired).To(BeTrue())
			Expect(err).To(MatchError(ContainSubstring("committing cleanup transaction failed")))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
			Expect(cleanupTx.commitCalls).To(Equal(1))
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
		})

		ginkgo.It("runs all cleanup queries immediately and stops after cancellation", func() {
			cleanupContext, cancel := context.WithCancel(ctx)
			sessionCalls := 0
			stateCalls := 0
			exchangeCalls := 0

			queries.deleteExpiredSessionsFunc = func(context.Context) (int64, error) {
				sessionCalls++
				return 0, nil
			}

			queries.deleteExpiredOIDCStatesFunc = func(context.Context) (int64, error) {
				stateCalls++
				return 0, nil
			}

			queries.deleteExpiredOIDCExchangesFunc = func(context.Context) (int64, error) {
				exchangeCalls++
				return 0, nil
			}

			cleanupTx.commitFunc = func(context.Context) error {
				cancel()
				return nil
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			store.RunCleanup(cleanupContext, logger, time.Hour)
			Expect(sessionCalls).To(Equal(1))
			Expect(stateCalls).To(Equal(1))
			Expect(exchangeCalls).To(Equal(1))
			Expect(cleanupTransactions.beginCalls).To(Equal(1))
			Expect(cleanupTx.commitCalls).To(Equal(1))
			Expect(cleanupTx.rollbackCalls).To(BeZero())
		})

		ginkgo.It("skips scheduled cleanup when another instance holds the lock", func() {
			cleanupContext, cancel := context.WithCancel(ctx)
			cleanupTx.tryAcquireCleanupLockFunc = func(context.Context) (bool, error) {
				cancel()
				return false, nil
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			store.RunCleanup(
				cleanupContext,
				logger,
				time.Hour,
			)

			Expect(cleanupTransactions.beginCalls).To(Equal(1))
			Expect(cleanupTx.commitCalls).To(BeZero())
			Expect(cleanupTx.rollbackCalls).To(Equal(1))
		})
	})
})
