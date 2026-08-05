package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// MountFunc registers a domain's routes onto the root router.
// Wiring happens in cmd/api/cmd/root.go, following the same explicit pattern
// used for controller registration in cmd/konfidence.
type MountFunc func(r chi.Router)

// New returns the root chi.Router with all routes and middleware registered.
func New(logger *slog.Logger, mounts ...MountFunc) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	r.Method(http.MethodGet, "/healthz", middleware.Handle(logger, handler.Healthz))
	r.Method(http.MethodGet, "/readyz", middleware.Handle(logger, handler.Readyz))

	r.Group(func(domain chi.Router) {
		for _, mount := range mounts {
			mount(domain)
		}
	})

	return r
}
