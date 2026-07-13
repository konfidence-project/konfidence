package vectorpromotion

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
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
func SetupControllers(ctx context.Context, mgr ctrl.Manager, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}

	log := logf.Log.WithName("vectorpromotion")

	cache, err := clientcache.New(
		clientcache.DefaultClientCacheSize,
		clientcache.DefaultExtract[*konfidence.VectorPromotionConfig],
		controller.NewCacheFactory(log, opts.Limiter),
	)
	if err != nil {
		return fmt.Errorf("creating clientcache: %w", err)
	}

	if err := controller.NewVectorPromotionReconciler(mgr, cache).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := controller.NewVectorPromotionTTLReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := controller.NewVectorPromotionStatusPropagationReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
