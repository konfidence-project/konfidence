package vectorpromotion

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/controller"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

const OperatorFlagName = "VectorPromotion"

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
