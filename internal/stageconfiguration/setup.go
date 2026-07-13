package stageconfiguration

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

const OperatorFlagName = "StageConfiguration"

// Options configures the stage configuration controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter
}

// SetupControllers registers all stage configuration controllers with the given manager.
func SetupControllers(mgr mcmanager.Manager, scheme *runtime.Scheme, restConfig *rest.Config, opts Options) error {
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

	if err := controller.NewStageConfigurationReconciler(mgr, scheme, restConfig, cache).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
