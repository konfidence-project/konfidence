package db_test

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"

	db "github.com/konfidence-project/konfidence/cmd/api/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestDB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DB Suite")
}

var _ = Describe("Migrations", func() {
	var (
		ctx    context.Context
		dsn    string
		logger *slog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = slog.New(slog.NewTextHandler(GinkgoWriter, nil))

		// Same image as hack/kden_local_dev; the schema relies on the
		// built-in uuidv7() that Postgres 18 ships.
		container, err := postgres.Run(ctx,
			"postgres:18-alpine",
			postgres.WithDatabase("konfidence_test"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			postgres.BasicWaitStrategies(),
		)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(container.Terminate(ctx)).To(Succeed())
		})

		dsn, err = container.ConnectionString(ctx, "sslmode=disable")
		Expect(err).NotTo(HaveOccurred())
	})

	tableExists := func(name string) bool {
		conn, err := sql.Open("pgx", dsn)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = conn.Close() }()

		var exists bool
		Expect(conn.QueryRowContext(ctx,
			"SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = $1)", name,
		).Scan(&exists)).To(Succeed())
		return exists
	}

	It("creates the schema with generated session ids", func() {
		Expect(db.MigrateUp(ctx, dsn, logger)).To(Succeed())

		conn, err := sql.Open("pgx", dsn)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = conn.Close() }()

		var id string
		Expect(conn.QueryRowContext(ctx,
			"INSERT INTO session (subject, access_token, token_expiry, expires_at) VALUES ('s', 't', 0, NOW()) RETURNING id",
		).Scan(&id)).To(Succeed())
		Expect(id).NotTo(BeEmpty())
	})

	It("is idempotent when Up is called twice", func() {
		Expect(db.MigrateUp(ctx, dsn, logger)).To(Succeed())
		Expect(db.MigrateUp(ctx, dsn, logger)).To(Succeed())
	})

	It("rolls back the most recent migration on Down after Up", func() {
		Expect(db.MigrateUp(ctx, dsn, logger)).To(Succeed())
		Expect(tableExists("session")).To(BeTrue())

		Expect(db.MigrateDown(ctx, dsn, logger)).To(Succeed())
		Expect(tableExists("session")).To(BeFalse())
	})

	It("returns an error for an unreachable database", func() {
		Expect(db.MigrateUp(ctx, "postgres://invalid:invalid@localhost:1/nodb", logger)).
			To(MatchError(ContainSubstring("database unreachable")))
	})
})
