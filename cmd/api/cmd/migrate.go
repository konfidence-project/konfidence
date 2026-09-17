package cmd

import (
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

	cmd.AddCommand(newMigrateUpCmd())
	cmd.AddCommand(newMigrateDownCmd())

	return cmd
}

func newMigrateUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cfg.Database.Connection == "" {
				return fmt.Errorf("--db-connection must not be empty")
			}
			logger, err := resolveLogger(cfg.Server.LogLevel)
			if err != nil {
				return err
			}
			_, err = db.MigrateUp(cmd.Context(), cfg.Database.Connection, logger)
			return err
		},
	}
}

func newMigrateDownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: "Roll back the most recent migration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if cfg.Database.Connection == "" {
				return fmt.Errorf("--db-connection must not be empty")
			}
			logger, err := resolveLogger(cfg.Server.LogLevel)
			if err != nil {
				return err
			}
			_, err = db.MigrateDown(cmd.Context(), cfg.Database.Connection, logger)
			return err
		},
	}
}

func resolveLogger(level string) (*slog.Logger, error) {
	handler, err := log.ResolveLogHandler(level, "json")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize log handler: %w", err)
	}
	return slog.New(handler), nil
}