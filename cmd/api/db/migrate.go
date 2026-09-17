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

// MigrateUp applies all pending migrations against the given DSN and returns the number of steps applied.
func MigrateUp(ctx context.Context, dsn string, logger *slog.Logger) (int, error) {
	return runMigrations(ctx, dsn, "up", logger)
}

// MigrateDown rolls back the most recent migration and returns the number of steps rolled back.
func MigrateDown(ctx context.Context, dsn string, logger *slog.Logger) (int, error) {
	return runMigrations(ctx, dsn, "down", logger)
}

func runMigrations(ctx context.Context, dsn string, direction string, logger *slog.Logger) (int, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return 0, fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(ctx); err != nil {
		return 0, fmt.Errorf("database unreachable: %w", err)
	}

	sqlFS, err := fs.Sub(migrations, "migration")
	if err != nil {
		return 0, fmt.Errorf("creating migration filesystem: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, sqlFS,
		goose.WithSlog(logger),
	)
	if err != nil {
		return 0, fmt.Errorf("creating migration provider: %w", err)
	}

	switch direction {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return 0, fmt.Errorf("running migrations up: %w", err)
		}
		return len(results), nil
	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return 0, fmt.Errorf("running migration down: %w", err)
		}
		if result == nil {
			return 0, nil
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("unknown direction %q: must be up or down", direction)
	}
}
