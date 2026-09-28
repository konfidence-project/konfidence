package cmd

import (
	"context"
	"fmt"
	"log/slog"

	db "github.com/konfidence-project/konfidence/cmd/api/db"
	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
		Long: `migrate contains sub-commands for managing the Konfidence API database schema.

Migrations are embedded in the binary and applied via goose.`,
	}

	cmd.AddCommand(newMigrationCmd("up", "Apply all pending migrations", db.MigrateUp))
	cmd.AddCommand(newMigrationCmd("down", "Roll back the most recent migration", db.MigrateDown))

	return cmd
}

func newMigrationCmd(use, short string, migrate func(context.Context, string, *slog.Logger) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cfg.Database.Connection == "" {
				return fmt.Errorf("--db-connection must not be empty")
			}
			handler, err := log.ResolveLogHandler(cfg.Server.LogLevel, "json")
			if err != nil {
				return fmt.Errorf("failed to initialize log handler: %w", err)
			}
			return migrate(cmd.Context(), cfg.Database.Connection, slog.New(handler))
		},
	}
}
