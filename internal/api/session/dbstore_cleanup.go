package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func (s *DBStore) deleteExpired(ctx context.Context, now time.Time) (int64, error) {
	deleted, err := s.queries.DeleteExpiredSessions(ctx, pgtype.Timestamptz{
		Time:  now.Add(-s.sessionExpiration),
		Valid: true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return deleted, nil
}

func (s *DBStore) RunCleanup(
	ctx context.Context,
	logger *slog.Logger,
	interval time.Duration,
) {
	logger.Info("Initialize db session cleanup...")
	cleanup := func() {
		deleted, err := s.deleteExpired(ctx, time.Now())
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("failed to clean up expired sessions", "error", err)
			}
			return
		}

		if deleted > 0 {
			logger.Info("deleted expired sessions", "count", deleted)
		}
	}

	// Remove sessions accumulated while the API was not running.
	cleanup()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}
