package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// Server wraps an http.Server with graceful shutdown support.
type Server struct {
	cfg     config.Parsed
	logger  *slog.Logger
	handler http.Handler
}

// New creates a Server from validated config and an http.Handler. Signal
// handling belongs in the caller (cmd layer).
func New(cfg config.Parsed, logger *slog.Logger, apiHandler http.Handler) *Server {
	return &Server{cfg: cfg, logger: logger, handler: apiHandler}
}

// ListenAndServe starts the HTTP server and blocks until ctx is canceled, then performs
// a graceful shutdown. The optional onAddr callback is called with the actual
// bound address once the server is listening — useful in tests with ":0".
func (s *Server) ListenAndServe(ctx context.Context, onAddr ...func(string)) error {
	ln, err := net.Listen("tcp", s.cfg.Server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.Server.Addr, err)
	}

	if len(onAddr) > 0 && onAddr[0] != nil {
		onAddr[0](ln.Addr().String())
	}

	srv := &http.Server{
		Handler:      s.router(),
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	s.logger.Info("api server stopped")
	return nil
}

func (s *Server) router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recovery(s.logger))
	r.Use(middleware.Logging(s.logger))
	r.Method(http.MethodGet, "/healthz", middleware.Handle(s.logger, handler.Healthz))
	r.Method(http.MethodGet, "/readyz", middleware.Handle(s.logger, handler.Readyz))
	r.Mount("/", s.handler)
	return r
}
