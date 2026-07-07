package vectorassembly

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	"github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/controller"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

const OperatorFlagName = "VectorAssembly"

// Options configures the vector assembly controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter
}

// SetupControllers registers all vector assembly controllers with the given manager.
func SetupControllers(mgr mcmanager.Manager, scheme *runtime.Scheme, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}

	log := logf.Log.WithName("vectorassembly")

	cache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*v1alpha1.VectorTemplate],
		controller.NewCacheFactory(log, opts.Limiter),
	)
	if err != nil {
		return fmt.Errorf("creating clientcache: %w", err)
	}

	if err := (&controller.VectorTemplateReconciler{
		Mgr:              mgr,
		Scheme:           scheme,
		Cache:            cache,
		VersionGenerator: vector.TimestampVectorVersionGenerator,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
