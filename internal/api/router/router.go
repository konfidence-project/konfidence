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

	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/api/handler"
	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// MountFunc registers a domain's routes onto the root router.
// Wiring happens in cmd/api/cmd/root.go, following the same explicit pattern
// used for controller registration in cmd/star and cmd/galaxy.
type MountFunc func(r chi.Router, logger *slog.Logger, k8s func() client.Client)

// New returns the root chi.Router with all routes and middleware registered.
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
		Scopes:       authCfg.ScopesSlice(),
	})
	r.Method(http.MethodGet, "/login", middleware.Handle(logger, auth.LoginStart))
	r.Method(http.MethodGet, "/auth/callback", middleware.Handle(logger, auth.Callback))
	r.Method(http.MethodPost, "/sessions/exchange", middleware.Handle(logger, auth.Exchange))

	r.Group(func(protected chi.Router) {
		protected.Use(auth.RequireSession)
		protected.Method(http.MethodPost, "/logout", middleware.Handle(logger, auth.Logout))
		protected.Method(http.MethodGet, "/identity", middleware.Handle(logger, auth.Identity))

		// Domain routes - each domain registers via MountFunc.
		for _, mount := range mounts {
			mount(protected, logger, k8s)
		}
	})

	return r
}

// lazyClient builds the k8s client on first call using ctrl.GetConfig() which
// reads KUBECONFIG env var or falls back to in-cluster config automatically.
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
					"hint", "set KUBECONFIG env var for local dev",
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
