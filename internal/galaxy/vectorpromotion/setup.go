package vectorpromotion

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/controller"
	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/ocm"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"k8s.io/apimachinery/pkg/runtime"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

const OperatorFlagName = "VectorPromotion"

// Options configures the vector promotion controllers.
type Options struct {
	// VectorVerifier is used to verify vectors during promotion.
	// If nil, vector verification is disabled (NoopVerifier is used).
	VectorVerifier crypto.Verifier
}

// SetupControllers registers all vector promotion controllers with the given manager.
func SetupControllers(ctx context.Context, mgr mcmanager.Manager, scheme *runtime.Scheme, opts Options) error {
	promotionAdapterConfig := make([]ocm.PromotionAdapterOption, 0, 1)
	promotionAdapterConfig = append(promotionAdapterConfig, ocm.WithVectorVerifier(opts.VectorVerifier))

	if err := controller.NewVectorPromotionReconciler(
		mgr,
		scheme,
		ocm.NewPromotionPortProvider(promotionAdapterConfig...),
	).SetupWithManager(mgr); err != nil {
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
