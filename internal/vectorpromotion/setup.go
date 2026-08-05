package vectorpromotion

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "VectorPromotion"

// Domain wires the vector promotion controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "VectorPromotion, VectorPromotionTTL, VectorPromotionStatusPropagation",
		Setup: func(ctx context.Context, deps operator.Deps) error {
			return SetupControllers(ctx, deps.Mgr, Options{Limiter: deps.Limiter})
		},
	}
}

// Options configures the vector promotion controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Currently unused:
	// promotion execution is stubbed pending the ADR-0032 execution rework.
	Limiter crypto.Limiter
}

// SetupControllers registers all vector promotion controllers with the given manager.
func SetupControllers(_ context.Context, mgr ctrl.Manager, _ Options) error {
	if err := controller.NewVectorPromotionReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	if err := controller.NewVectorPromotionTTLReconciler(mgr).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return controller.NewVectorPromotionStatusPropagationReconciler(mgr).
		SetupWithManager(mgr)
}
