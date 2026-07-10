package vectorpromotion

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/ocm/clientcache"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

const OperatorFlagName = "VectorPromotion"

// Options configures the vector promotion controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter
}

// SetupControllers registers all vector promotion controllers with the given manager.
func SetupControllers(ctx context.Context, mgr mcmanager.Manager, scheme *runtime.Scheme, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}

	log := logf.Log.WithName("vectorpromotion")

	cache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*galaxy.VectorPromotionConfig],
		controller.NewCacheFactory(log, opts.Limiter),
	)
	if err != nil {
		return fmt.Errorf("creating clientcache: %w", err)
	}

	if err := (&controller.VectorPromotionReconciler{
		Mgr:    mgr,
		Scheme: scheme,
		Cache:  cache,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controller.VectorPromotionTTLReconciler{
		Mgr:    mgr,
		Scheme: scheme,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	if err := (&controller.VectorPromotionStatusPropagationReconciler{
		Mgr:    mgr,
		Scheme: scheme,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
