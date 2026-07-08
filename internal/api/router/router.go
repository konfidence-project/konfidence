package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
	pkgk8s "github.com/konfidence-project/konfidence/pkg/k8s"
)

// New returns the root chi.Router with all routes and middleware registered.
//
// Middleware stack (outermost to innermost):
//
//	Recovery - last-resort panic safety net; keeps the process alive
//	Logging  - logs method, path, status, duration after the handler returns
//
// The Kubernetes client is built lazily on the first domain request via
// k8sClient(). Probe endpoints never trigger client construction, so the
// server starts cleanly without a cluster (useful for local development).
func New(logger *slog.Logger, kubeconfigPath string) http.Handler {
	k8s := lazyClient(logger, kubeconfigPath)

	r := chi.NewRouter()

	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	// Probe endpoints - no cluster dependency.
	r.Method(http.MethodGet, "/healthz", middleware.Handle(logger, handler.Healthz))
	r.Method(http.MethodGet, "/readyz", middleware.Handle(logger, handler.Readyz))

	// Domain routes are mounted here as sub-routers, e.g.:
	//   r.Mount("/api/v1", v1.NewRouter(logger, k8s))
	_ = k8s

	return r
}

// lazyClient returns a function that builds the k8s client on first call and
// caches the result. If construction fails the error is logged and nil is
// returned — domain handlers must check for nil and return handler.NewInternal.
func lazyClient(logger *slog.Logger, kubeconfigPath string) func() client.Client {
	var (
		once      sync.Once
		k8sClient client.Client
	)
	return func() client.Client {
		once.Do(func() {
			c, err := pkgk8s.NewClient(kubeconfigPath)
			if err != nil {
				logger.Error("failed to build k8s client",
					"error", fmt.Sprintf("%v", err),
					"hint", "start with --kubeconfig for local dev or ensure in-cluster config is available",
				)
				return
			}
			k8sClient = c
		})
		return k8sClient
	}
}
