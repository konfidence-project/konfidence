package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// New returns the root chi.Router with all routes and middleware registered.
//
// Middleware stack (outermost → innermost):
//   Recovery  — last-resort panic safety net; keeps the process alive
//   Logging   — logs method, path, status, duration after the handler returns
//
// Each route is wrapped with middleware.Handle(logger, h) which calls the
// handler.Handler directly and translates any returned error into a JSON
// response. This keeps error handling explicit at the route level rather than
// relying on context injection.
func New(logger *slog.Logger, k8s client.Client) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	// Probe endpoints — no cluster dependency.
	r.Method(http.MethodGet, "/healthz", middleware.Handle(logger, handler.Healthz))
	r.Method(http.MethodGet, "/readyz", middleware.Handle(logger, handler.Readyz))

	// Domain routes are mounted here as sub-routers, e.g.:
	//   r.Mount("/api/v1", v1.NewRouter(logger, k8s))
	_ = k8s

	return r
}
