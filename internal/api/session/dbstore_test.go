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

var _ db.Querier = (*recordingQuerier)(nil)

type recordingQuerier struct {
	mu                        sync.Mutex
	createSessionFunc         func(context.Context, db.CreateSessionParams) (pgtype.UUID, error)
	deleteSessionFunc         func(context.Context, pgtype.UUID) error
	getSessionFunc            func(context.Context, db.GetSessionParams) (db.Session, error)
	deleteExpiredSessionsFunc func(context.Context, pgtype.Timestamptz) (int64, error)
	updateSessionFunc         func(context.Context, db.UpdateSessionParams) (int64, error)
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

func (q *recordingQuerier) DeleteSession(
	ctx context.Context,
	id pgtype.UUID,
) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleteSessionFunc == nil {
		return nil
	}

	return q.deleteSessionFunc(ctx, id)
}

func (q *recordingQuerier) UpdateSession(
	ctx context.Context,
	params db.UpdateSessionParams,
) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.updateSessionFunc == nil {
		return 0, nil
	}

	return q.updateSessionFunc(ctx, params)
}

func (q *recordingQuerier) GetSession(
	ctx context.Context,
	params db.GetSessionParams,
) (db.Session, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.getSessionFunc == nil {
		return db.Session{}, nil
	}

	return q.getSessionFunc(ctx, params)
}

func (q *recordingQuerier) DeleteExpiredSessions(
	ctx context.Context,
	expiredBefore pgtype.Timestamptz,
) (int64, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.deleteExpiredSessionsFunc == nil {
		return 0, nil
	}

	return q.deleteExpiredSessionsFunc(ctx, expiredBefore)
}

const databaseSessionID = "0194e45e-2fc0-7ef2-96be-84d2f66405de"

func databaseSessionUUID() pgtype.UUID {
	var id pgtype.UUID
	Expect(id.Scan(databaseSessionID)).To(Succeed())
	return id
}

var _ = ginkgo.Describe("DBStore", func() {

	var (
		ctx     context.Context
		queries *recordingQuerier
		store   *DBStore
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		queries = &recordingQuerier{}
		store = NewDBStore(queries, 30*time.Minute)
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
		ginkgo.It("gets a session created after the expiration cutoff", func() {
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

			var captured db.GetSessionParams
			queries.getSessionFunc = func(
				_ context.Context,
				params db.GetSessionParams,
			) (db.Session, error) {
				captured = params
				return databaseSession, nil
			}

			before := time.Now()
			stored, err := store.Get(ctx, databaseSessionID)
			after := time.Now()

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

			Expect(captured.ID).To(Equal(databaseSessionUUID()))
			Expect(captured.SessionExpiration.Valid).To(BeTrue())
			Expect(captured.SessionExpiration.Time).To(
				BeTemporally(">=", before.Add(-30*time.Minute)),
			)
			Expect(captured.SessionExpiration.Time).To(
				BeTemporally("<=", after.Add(-30*time.Minute)),
			)
		})

		ginkgo.It("returns nil when the session does not exist", func() {
			queries.getSessionFunc = func(
				context.Context,
				db.GetSessionParams,
			) (db.Session, error) {
				return db.Session{}, sql.ErrNoRows
			}

			stored, err := store.Get(ctx, databaseSessionID)

			Expect(err).NotTo(HaveOccurred())
			Expect(stored).To(BeNil())
		})

		ginkgo.It("rejects an invalid session ID", func() {
			queried := false
			queries.getSessionFunc = func(context.Context, db.GetSessionParams) (db.Session, error) {
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
			queries.getSessionFunc = func(context.Context, db.GetSessionParams) (db.Session, error) {
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

	ginkgo.Describe("Update", func() {
		ginkgo.It("updates refreshed session values without changing creation time", func() {
			refreshToken := "new-refresh-token"
			updatedSession := &Session{
				Subject:      "subject-id",
				Groups:       []string{"developers"},
				AccessToken:  "new-access-token",
				RefreshToken: &refreshToken,
				TokenExpiry:  123456789,
				Context: Context{
					ID:                databaseSessionID,
					Name:              new("Test User"),
					GivenName:         new("Test"),
					FamilyName:        new("User"),
					PreferredUsername: new("test.user"),
					Email:             new("test@example.com"),
				},
			}

			var captured db.UpdateSessionParams
			queries.updateSessionFunc = func(
				_ context.Context,
				params db.UpdateSessionParams,
			) (int64, error) {
				captured = params
				return 1, nil
			}

			Expect(store.Update(ctx, updatedSession)).To(Succeed())

			Expect(captured).To(Equal(db.UpdateSessionParams{
				Name:              updatedSession.Name,
				GivenName:         updatedSession.GivenName,
				FamilyName:        updatedSession.FamilyName,
				PreferredUserName: updatedSession.PreferredUsername,
				Email:             updatedSession.Email,
				Groups:            updatedSession.Groups,
				AccessToken:       updatedSession.AccessToken,
				RefreshToken:      updatedSession.RefreshToken,
				TokenExpiry:       updatedSession.TokenExpiry,
				ID:                databaseSessionUUID(),
				Subject:           updatedSession.Subject,
			}))
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
		ginkgo.It("deletes sessions older than the configured timeout", func() {
			now := time.Date(
				2026,
				time.August,
				19,
				12,
				0,
				0,
				0,
				time.UTC,
			)

			var captured pgtype.Timestamptz
			queries.deleteExpiredSessionsFunc = func(
				_ context.Context,
				expiredBefore pgtype.Timestamptz,
			) (int64, error) {
				captured = expiredBefore
				return 3, nil
			}

			deleted, err := store.deleteExpired(ctx, now)

			Expect(err).NotTo(HaveOccurred())
			Expect(deleted).To(Equal(int64(3)))
			Expect(captured.Valid).To(BeTrue())
			Expect(captured.Time).To(Equal(
				now.Add(-30 * time.Minute),
			))
		})

		ginkgo.It("wraps cleanup database errors", func() {
			databaseErr := errors.New("database unavailable")
			queries.deleteExpiredSessionsFunc = func(
				context.Context,
				pgtype.Timestamptz,
			) (int64, error) {
				return 0, databaseErr
			}

			deleted, err := store.deleteExpired(ctx, time.Now())

			Expect(deleted).To(BeZero())
			Expect(err).To(MatchError(
				ContainSubstring(
					"failed to delete expired sessions",
				),
			))
			Expect(errors.Is(err, databaseErr)).To(BeTrue())
		})

		ginkgo.It("runs cleanup immediately and stops after cancellation", func() {
			cleanupContext, cancel := context.WithCancel(ctx)
			calls := 0

			queries.deleteExpiredSessionsFunc = func(
				context.Context,
				pgtype.Timestamptz,
			) (int64, error) {
				calls++
				cancel()
				return 0, nil
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			store.RunCleanup(
				cleanupContext,
				logger,
				time.Hour,
			)

			Expect(calls).To(Equal(1))
		})
	})
})
