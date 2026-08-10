package stageconfiguration

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "StageConfiguration"

// Domain wires the stage configuration controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "StageConfiguration",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return SetupControllers(deps.Mgr, Options{Limiter: deps.Limiter})
		},
	}
}

// Options configures the stage configuration controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter
}

// SetupControllers registers all stage configuration controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}

	log := logf.Log.WithName("stageconfiguration")

	cache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*konfidence.StageConfiguration],
		controller.NewCacheFactory(log, opts.Limiter),
	)
	if err != nil {
		return fmt.Errorf("creating clientcache: %w", err)
	}

	if err := controller.NewStageConfigurationReconciler(mgr, cache).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
