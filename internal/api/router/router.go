package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// New returns the root chi.Router with all routes and middleware registered.
//
// Middleware stack (outermost to innermost):
//
//	Recovery - last-resort panic safety net; keeps the process alive
//	Logging  - logs method, path, status, duration after the handler returns
//
// The Kubernetes client is built lazily on the first domain request via
// k8s(). Probe endpoints never trigger client construction, so the server
// starts cleanly without a cluster (useful for local development).
// The client is built using ctrl.GetConfigOrDie() which resolves config via
// the standard KUBECONFIG env var or in-cluster config automatically.
func New(logger *slog.Logger, scheme *runtime.Scheme) http.Handler {
	k8s := lazyClient(logger, scheme)

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
// caches the result. Config is resolved via ctrl.GetConfigOrDie() which reads
// KUBECONFIG env var or falls back to in-cluster config automatically.
func lazyClient(logger *slog.Logger, scheme *runtime.Scheme) func() client.Client {
	var (
		once      sync.Once
		k8sClient client.Client
	)
	return func() client.Client {
		once.Do(func() {
			cfg, err := ctrl.GetConfig()
			if err != nil {
				logger.Error("failed to get k8s config",
					"error", fmt.Sprintf("%v", err),
					"hint", "set KUBECONFIG env var for local dev or ensure in-cluster config is available",
				)
				return
			}
			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				logger.Error("failed to build k8s client", "error", fmt.Sprintf("%v", err))
				return
			}
			k8sClient = c
		})
		return k8sClient
	}
}
