package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
	"github.com/konfidence-project/konfidence/internal/api/ui"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
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

// ListenAndServe starts the HTTP server and blocks until the context is
// canceled or a managed component fails. Before returning, it stops the
// watcher and shuts down the HTTP server. The optional onAddr callback is called with the actual
// bound address once the server is listening — useful in tests with ":0".
func (s *Server) ListenAndServe(ctx context.Context, onAddr ...func(string)) error {
	rootHandler, err := s.router()
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", s.cfg.Server.Addr)
	if err != nil {
		return fmt.Errorf(
			"failed to listen on %s: %w",
			s.cfg.Server.Addr,
			err,
		)
	}

	// covers initialization failures before Serve takes ownership and is
	// harmless after Shutdown has already closed the listener.
	defer func() {
		_ = ln.Close()
	}()

	var watcher *certwatcher.CertWatcher
	if s.cfg.Server.TLSCertFile != "" {
		watcher, err = certwatcher.New(s.cfg.Server.TLSCertFile, s.cfg.Server.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to initialize cert watcher: %w", err)
		}
	}

	srv := &http.Server{
		Handler:      rootHandler,
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		WriteTimeout: s.cfg.Server.WriteTimeout,
	}

	if watcher != nil {
		srv.TLSConfig = &tls.Config{GetCertificate: watcher.GetCertificate}
	}

	// report the address only after all startup initialization succeeded.
	if len(onAddr) > 0 && onAddr[0] != nil {
		onAddr[0](ln.Addr().String())
	}

	group, groupCtx := errgroup.WithContext(ctx)
	if watcher != nil {
		group.Go(func() error {
			err := watcher.Start(groupCtx)
			if err == nil {
				return nil
			}

			// cancellation is expected when the server stops or another
			// component fails. CertWatcher may return an error while
			// unwinding its initial watch setup.
			if groupCtx.Err() != nil {
				return nil
			}

			return fmt.Errorf("cert watcher failed: %w", err)
		})
	}

	group.Go(func() error {
		s.logger.Info("api server starting", "addr", ln.Addr().String(), "tls", watcher != nil)
		var serveErr error
		if watcher != nil {
			serveErr = srv.ServeTLS(ln, "", "")
		} else {
			serveErr = srv.Serve(ln)
		}

		if errors.Is(serveErr, http.ErrServerClosed) {
			return nil
		}

		if serveErr == nil {
			return errors.New("api server stopped unexpectedly")
		}

		return fmt.Errorf("server error: %w", serveErr)
	})

	group.Go(func() error {
		<-groupCtx.Done()
		s.logger.Info("shutting down api server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			// shutdown does not close active connections after its
			// deadline, so force closure before returning.
			closeErr := srv.Close()
			return fmt.Errorf("graceful shutdown failed: %w", errors.Join(err, closeErr))
		}

		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	s.logger.Info("api server stopped")
	return nil
}

func (s *Server) router() (http.Handler, error) {
	r := chi.NewRouter()
	r.Use(middleware.Recovery(s.logger))
	r.Use(middleware.Logging(s.logger))
	r.Method(http.MethodGet, "/healthz", middleware.Handle(s.logger, handler.Healthz))
	r.Method(http.MethodGet, "/readyz", middleware.Handle(s.logger, handler.Readyz))
	r.Handle("/api", s.handler)
	r.Handle("/api/*", s.handler)

	if s.cfg.Server.UIAssetPath != "" {
		uiHandler, err := ui.New(os.DirFS(s.cfg.Server.UIAssetPath))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize dashboard: %w", err)
		}
		r.NotFound(uiHandler.ServeHTTP)
	}

	return r, nil
}
