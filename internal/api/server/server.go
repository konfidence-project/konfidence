package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/router"
)

// Server wraps an http.Server with graceful shutdown support
type Server struct {
	cfg    config.Config
	k8s    client.Client
	logger *slog.Logger
}

// New creates a Server from the given config and a ready-to-use Kubernetes client.
// The client is passed through to the router so that handlers can read and write
// Konfidence CRDs (Stage, VectorDeployment, VectorPromotion, etc.) on the star cluster
func New(cfg config.Config, k8s client.Client) *Server {
	level := resolveLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	return &Server{cfg: cfg, k8s: k8s, logger: logger}
}

// Run starts the HTTP server and blocks until the context is cancelled or a
// termination signal is received, then attempts a graceful shutdown
func (s *Server) Run(ctx context.Context) error {
	parsed := s.cfg.Parse()

	srv := &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      router.New(s.logger, s.k8s),
		ReadTimeout:  parsed.ReadTimeout,
		WriteTimeout: parsed.WriteTimeout,
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("api server starting", "addr", s.cfg.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("server error: %w", err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down api server")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), parsed.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	s.logger.Info("api server stopped")
	return nil
}

func resolveLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
