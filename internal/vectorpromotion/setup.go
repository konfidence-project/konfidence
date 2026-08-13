package vectorpromotion

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "VectorPromotion"

// Domain wires the vector promotion controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "VectorPromotion, VectorPromotionConfig, VectorPromotionTTL",
		Setup: func(ctx context.Context, deps operator.Deps) error {
			return SetupControllers(ctx, deps.Mgr, Options{})
		},
	}
}

// Options configures the vector promotion controllers. Currently empty; kept
// so enabling future options does not change the call signature.
type Options struct{}

// SetupControllers registers all vector promotion controllers with the given manager.
func SetupControllers(ctx context.Context, mgr ctrl.Manager, _ Options) error {
	if err := controller.RegisterFieldIndexes(ctx, mgr); err != nil {
		return err
	}

	if err := controller.NewVectorPromotionReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := controller.NewVectorPromotionConfigReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return controller.NewVectorPromotionTTLReconciler(mgr).
		SetupWithManager(mgr)
}
