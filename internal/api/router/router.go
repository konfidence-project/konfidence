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

	"github.com/konfidence-project/konfidence/internal/api/middleware"
)

// MountFunc registers a domain's routes onto the root router.
// Wiring happens in cmd/api/cmd/root.go, following the same explicit pattern
// used for controller registration in cmd/konfidence.
type MountFunc func(r chi.Router, logger *slog.Logger, k8s func() (client.Client, error))

// New returns the root chi.Router with all routes and middleware registered.
func New(logger *slog.Logger, scheme *runtime.Scheme, mounts ...MountFunc) http.Handler {
	k8s := lazyClient(scheme)

	r := chi.NewRouter()

	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	r.Group(func(domain chi.Router) {
		for _, mount := range mounts {
			mount(domain, logger, k8s)
		}
	})

	return r
}

// lazyClient builds the k8s client on first call using ctrl.GetConfig() which
// reads KUBECONFIG env var or falls back to in-cluster config automatically.
// The returned closure returns (nil, error) when the client cannot be built —
// callers must check for a nil client before use.
func lazyClient(scheme *runtime.Scheme) func() (client.Client, error) {
	var (
		once      sync.Once
		k8sClient client.Client
		k8sErr    error
	)
	return func() (client.Client, error) {
		once.Do(func() {
			cfg, err := ctrl.GetConfig()
			if err != nil {
				k8sErr = fmt.Errorf("failed to get k8s config (set KUBECONFIG for local dev): %w", err)
				return
			}
			c, err := client.New(cfg, client.Options{Scheme: scheme})
			if err != nil {
				k8sErr = fmt.Errorf("failed to build k8s client: %w", err)
				return
			}
			k8sClient = c
		})
		return k8sClient, k8sErr
	}
}
