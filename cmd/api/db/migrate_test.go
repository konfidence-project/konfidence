package db_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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
		ctx       context.Context
		container *postgres.PostgresContainer
		dsn       string
		logger    *slog.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

		var err error
		container, err = postgres.Run(ctx,
			"konfidence-postgres:local",
			postgres.WithDatabase("konfidence_test"),
			postgres.WithUsername("test"),
			postgres.WithPassword("test"),
			postgres.BasicWaitStrategies(),
		)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(container.Terminate(ctx)).To(Succeed())
		})

		host, err := container.Host(ctx)
		Expect(err).NotTo(HaveOccurred())
		port, err := container.MappedPort(ctx, "5432")
		Expect(err).NotTo(HaveOccurred())
		dsn = fmt.Sprintf("postgres://test:test@%s:%s/konfidence_test", host, port.Port())
	})

	It("applies all migrations on Up", func() {
		_, err := db.MigrateUp(ctx, dsn, logger)
		Expect(err).NotTo(HaveOccurred())
	})

	It("is idempotent when Up is called twice", func() {
		_, err := db.MigrateUp(ctx, dsn, logger)
		Expect(err).NotTo(HaveOccurred())
		_, err = db.MigrateUp(ctx, dsn, logger)
		Expect(err).NotTo(HaveOccurred())
	})

	It("rolls back the most recent migration on Down after Up", func() {
		_, err := db.MigrateUp(ctx, dsn, logger)
		Expect(err).NotTo(HaveOccurred())
		_, err = db.MigrateDown(ctx, dsn, logger)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns an error for an invalid DSN", func() {
		_, err := db.MigrateUp(ctx, "postgres://invalid:invalid@localhost:1/nodb", logger)
		Expect(err).To(MatchError(ContainSubstring("database unreachable")))
	})
})
