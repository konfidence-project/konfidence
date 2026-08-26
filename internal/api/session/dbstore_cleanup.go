package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type cleanupResult struct {
	sessions  int64
	states    int64
	exchanges int64
}

func (s *DBStore) deleteExpired(ctx context.Context) (cleanupResult, bool, error) {
	result := cleanupResult{}

	tx, err := s.cleanupTransactionHandler.BeginTransaction(ctx)
	if err != nil {
		return result, false, fmt.Errorf("starting cleanup transaction failed: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		_ = tx.Rollback(rollbackCtx)
	}()

	acquired, err := tx.TryAcquireCleanupLock(ctx)
	if err != nil {
		return result, false, fmt.Errorf("acquiring cleanup advisory lock failed: %w", err)
	}
	if !acquired {
		return result, false, nil
	}

	deletedSessions, err := tx.DeleteExpiredSessions(ctx)

	if err != nil {
		return result, true, fmt.Errorf("deleting expired sessions failed: %w", err)
	}
	result.sessions = deletedSessions

	deletedStates, err := tx.DeleteExpiredOIDCStates(ctx)
	if err != nil {
		return result, true, fmt.Errorf("deleting expired OIDC states failed: %w", err)
	}
	result.states = deletedStates

	deletedExchanges, err := tx.DeleteExpiredOIDCExchanges(ctx)
	if err != nil {
		return result, true, fmt.Errorf("deleting expired OIDC exchanges failed: %w", err)
	}
	result.exchanges = deletedExchanges

	if err := tx.Commit(ctx); err != nil {
		return cleanupResult{}, true, fmt.Errorf("committing cleanup transaction failed: %w", err)
	}

	committed = true
	return result, true, nil
}

func (s *DBStore) RunCleanup(
	ctx context.Context,
	logger *slog.Logger,
	interval time.Duration,
) {
	logger.Info("Initialize db state cleanup...")
	cleanup := func() {
		result, acquired, err := s.deleteExpired(ctx)
		if err != nil {
			if ctx.Err() == nil {
				logger.Error("failed to clean up expired authentication data", "error", err)
			}
			return
		}

		if !acquired {
			logger.Debug("skipped db state cleanup because another API instance holds the lock")
			return
		}

		if result.sessions != 0 || result.states != 0 || result.exchanges != 0 {
			logger.Info("deleted expired authentication data", "sessions", result.sessions,
				"oidc_states", result.states, "oidc_exchanges", result.exchanges,
			)
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
