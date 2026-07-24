package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/router"
)

// Server wraps an http.Server with graceful shutdown support.
type Server struct {
	cfg    config.Parsed
	logger *slog.Logger
	mounts []router.MountFunc
}

// New creates a Server from a validated Parsed config and optional domain
// mount functions. Signal handling belongs in the caller (cmd layer).
func New(cfg config.Parsed, mounts ...router.MountFunc) *Server {
	level := resolveLogLevel(cfg.LogLevel)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	return &Server{cfg: cfg, logger: logger, mounts: mounts}
}

// Run starts the HTTP server and blocks until ctx is cancelled, then performs
// a graceful shutdown. The optional onAddr callback is called with the actual
// bound address once the server is listening — useful in tests with ":0".
func (s *Server) Run(ctx context.Context, onAddr ...func(string)) error {
	ln, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.Addr, err)
	}

	if len(onAddr) > 0 && onAddr[0] != nil {
		onAddr[0](ln.Addr().String())
	}

	srv := &http.Server{
		Handler:      router.New(s.logger, s.cfg.Scheme, s.mounts...),
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("api server starting", "addr", ln.Addr().String())
		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
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
