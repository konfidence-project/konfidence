package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migration/*.sql
var migrations embed.FS

// MigrateUp applies all pending migrations against the database at dsn.
func MigrateUp(ctx context.Context, dsn string, logger *slog.Logger) error {
	return withProvider(ctx, dsn, logger, func(p *goose.Provider) error {
		if _, err := p.Up(ctx); err != nil {
			return fmt.Errorf("running migrations up: %w", err)
		}
		return nil
	})
}

// MigrateDown rolls back the most recently applied migration.
func MigrateDown(ctx context.Context, dsn string, logger *slog.Logger) error {
	return withProvider(ctx, dsn, logger, func(p *goose.Provider) error {
		if _, err := p.Down(ctx); err != nil {
			return fmt.Errorf("running migration down: %w", err)
		}
		return nil
	})
}

func withProvider(ctx context.Context, dsn string, logger *slog.Logger, run func(*goose.Provider) error) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database unreachable: %w", err)
	}

	sqlFS, err := fs.Sub(migrations, "migration")
	if err != nil {
		return fmt.Errorf("creating migration filesystem: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, sqlFS, goose.WithSlog(logger), goose.WithVerbose(true))
	if err != nil {
		return fmt.Errorf("creating migration provider: %w", err)
	}

	return run(provider)
}
