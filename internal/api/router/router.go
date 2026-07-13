package router

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// MountFunc registers a domain's routes onto the root router.
// Each business domain exposes a MountFunc that wires its sub-router:
// Each business domain exposes a MountFunc that wires its sub-router:
//
//	func Mount(r chi.Router, logger *slog.Logger, k8s func() client.Client) {
//	    r.Mount("/api/v1/stages", NewRouter(logger, k8s))
//	}
//
// Wiring happens in cmd/api/cmd/root.go, following the same explicit pattern
// used for controller registration in cmd/star and cmd/galaxy.
type MountFunc func(r chi.Router, logger *slog.Logger, k8s func() client.Client)

// New returns the root chi.Router with all routes and middleware registered.
//
// Middleware stack (outermost to innermost):
//
//	Recovery - last-resort panic safety net; keeps the process alive
//	Logging  - logs method, path, status, duration after the handler returns
//
// The Kubernetes client is built lazily on the first domain request. Probe
// endpoints never trigger client construction, so the server starts cleanly
// without a cluster (useful for local development). Config is resolved via
// the standard KUBECONFIG env var or in-cluster config automatically.
//
// Domain routes are registered by passing MountFunc values — one per domain:
//
//	router.New(logger, scheme, stageapi.Mount, vectorpromotionapi.Mount)
func New(logger *slog.Logger, scheme *runtime.Scheme, authCfg config.AuthConfig, mounts ...MountFunc) http.Handler {
	k8s := lazyClient(logger, scheme)

	r := chi.NewRouter()

	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	// Probe endpoints - no cluster dependency.
	r.Method(http.MethodGet, "/healthz", middleware.Handle(logger, handler.Healthz))
	r.Method(http.MethodGet, "/readyz", middleware.Handle(logger, handler.Readyz))

	auth := handler.NewAuth(handler.AuthConfig{
		AuthorizeURL: authCfg.AuthorizeURL,
		TokenURL:     authCfg.TokenURL,
		UserInfoURL:  authCfg.UserInfoURL,
		ClientID:     authCfg.ClientID,
		RedirectURI:  authCfg.RedirectURI,
		Scopes:       strings.Fields(authCfg.Scopes),
	})
	r.Method(http.MethodGet, "/login", middleware.Handle(logger, auth.LoginStart))
	r.Method(http.MethodGet, "/auth/callback", middleware.Handle(logger, auth.Callback))
	r.Method(http.MethodPost, "/sessions/exchange", middleware.Handle(logger, auth.Exchange))

	r.Group(func(protected chi.Router) {
		protected.Use(auth.RequireSession)
		protected.Method(http.MethodPost, "/logout", middleware.Handle(logger, auth.Logout))
		protected.Method(http.MethodGet, "/identity", middleware.Handle(logger, auth.Identity))
		protected.Method(http.MethodGet, "/api/v1/stages", middleware.Handle(logger, handler.ListStages))

		// Domain routes - each domain registers its own sub-router via MountFunc.
		for _, mount := range mounts {
			mount(protected, logger, k8s)
		}
	})

	return r
}

// lazyClient returns a function that builds the k8s client on first call and
// caches the result. Config is resolved via ctrl.GetConfig() which reads the
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
